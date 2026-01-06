package server

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/gorilla/mux"
	"github.com/soulteary/webhook/internal/flags"
	"github.com/soulteary/webhook/internal/fn"
	"github.com/soulteary/webhook/internal/hook"
	"github.com/soulteary/webhook/internal/middleware"
	"github.com/soulteary/webhook/internal/rules"
)

type flushWriter struct {
	f http.Flusher
	w io.Writer
}

func (fw *flushWriter) Write(p []byte) (n int, err error) {
	n, err = fw.w.Write(p)
	if fw.f != nil {
		fw.f.Flush()
	}
	return
}

// trackingResponseWriter 用于跟踪是否已经写入响应，以便在错误时设置状态码
type trackingResponseWriter struct {
	http.ResponseWriter
	written bool
	status  int
}

func (trw *trackingResponseWriter) Write(p []byte) (n int, err error) {
	if !trw.written {
		// 第一次写入时，如果没有设置状态码，使用默认的 200
		if trw.status == 0 {
			trw.status = http.StatusOK
		}
		trw.ResponseWriter.WriteHeader(trw.status)
		trw.written = true
	}
	return trw.ResponseWriter.Write(p)
}

func (trw *trackingResponseWriter) WriteHeader(code int) {
	if !trw.written {
		trw.status = code
		trw.ResponseWriter.WriteHeader(code)
		trw.written = true
	}
}

func (trw *trackingResponseWriter) HasWritten() bool {
	return trw.written
}

// Flush 实现 http.Flusher 接口，以便支持流式输出
func (trw *trackingResponseWriter) Flush() {
	if f, ok := trw.ResponseWriter.(http.Flusher); ok {
		f.Flush()
	}
}

func createHookHandler(appFlags flags.AppFlags) func(w http.ResponseWriter, r *http.Request) {
	// 从配置中获取超时和并发设置，如果未配置则使用默认值
	maxConcurrent := appFlags.MaxConcurrentHooks
	if maxConcurrent <= 0 {
		maxConcurrent = DefaultMaxConcurrentHooks
	}

	hookTimeout := time.Duration(appFlags.HookTimeoutSeconds) * time.Second
	if hookTimeout <= 0 {
		hookTimeout = DefaultHookTimeout
	}

	executionTimeout := time.Duration(appFlags.HookExecutionTimeout) * time.Second
	if executionTimeout <= 0 {
		executionTimeout = HookExecutionTimeout
	}

	// 创建 HookExecutor 实例，管理并发控制
	// 创建一个包装函数，将 appFlags 传递给 handleHook
	executorFunc := func(ctx context.Context, h *hook.Hook, r *hook.Request, w http.ResponseWriter) (string, error) {
		return handleHook(ctx, h, r, w, appFlags)
	}
	executor := NewHookExecutorWithFunc(maxConcurrent, hookTimeout, executorFunc)

	return func(w http.ResponseWriter, r *http.Request) {
		requestID := middleware.GetReqID(r.Context())
		req := &hook.Request{
			ID:         requestID,
			RawRequest: r,
		}

		log.Printf("[%s] incoming HTTP %s request from %s\n", requestID, r.Method, r.RemoteAddr)

		hookID := strings.TrimSpace(mux.Vars(r)["id"])
		hookID = fn.RemoveNewlinesAndTabs(hookID)

		matchedHook := rules.MatchLoadedHook(hookID)
		if matchedHook == nil {
			w.WriteHeader(http.StatusNotFound)
			fmt.Fprint(w, "Hook not found.")
			return
		}

		// Check for allowed methods
		var allowedMethod bool

		switch {
		case len(matchedHook.HTTPMethods) != 0:
			for i := range matchedHook.HTTPMethods {
				// TODO(moorereason): refactor config loading and reloading to
				// sanitize these methods once at load time.
				if r.Method == strings.ToUpper(strings.TrimSpace(matchedHook.HTTPMethods[i])) {
					allowedMethod = true
					break
				}
			}
		case appFlags.HttpMethods != "":
			for _, v := range strings.Split(appFlags.HttpMethods, ",") {
				if r.Method == v {
					allowedMethod = true
					break
				}
			}
		default:
			allowedMethod = true
		}

		if !allowedMethod {
			w.WriteHeader(http.StatusMethodNotAllowed)
			log.Printf("[%s] HTTP %s method not allowed for hook %q", requestID, r.Method, hookID)

			return
		}

		log.Printf("[%s] %s got matched\n", requestID, hookID)

		for _, responseHeader := range appFlags.ResponseHeaders {
			w.Header().Set(responseHeader.Name, responseHeader.Value)
		}

		var err error

		// set contentType to IncomingPayloadContentType or header value
		req.ContentType = r.Header.Get("Content-Type")
		if len(matchedHook.IncomingPayloadContentType) != 0 {
			req.ContentType = matchedHook.IncomingPayloadContentType
		}

		isMultipart := strings.HasPrefix(req.ContentType, "multipart/form-data;")

		if !isMultipart {
			req.Body, err = io.ReadAll(r.Body)
			if err != nil {
				log.Printf("[%s] error reading the request body: %+v\n", requestID, err)
			}
		}

		req.ParseHeaders(r.Header)
		req.ParseQuery(r.URL.Query())

		switch {
		case strings.Contains(req.ContentType, "json"):
			err = req.ParseJSONPayload()
			if err != nil {
				log.Printf("[%s] %s", requestID, err)
			}

		case strings.Contains(req.ContentType, "x-www-form-urlencoded"):
			err = req.ParseFormPayload()
			if err != nil {
				log.Printf("[%s] %s", requestID, err)
			}

		case strings.Contains(req.ContentType, "xml"):
			err = req.ParseXMLPayload()
			if err != nil {
				log.Printf("[%s] %s", requestID, err)
			}

		case isMultipart:
			err = r.ParseMultipartForm(appFlags.MaxMultipartMem)
			if err != nil {
				msg := fmt.Sprintf("[%s] error parsing multipart form: %+v\n", requestID, err)
				log.Println(msg)
				w.WriteHeader(http.StatusInternalServerError)
				fmt.Fprint(w, "Error occurred while parsing multipart form.")
				return
			}

			for k, v := range r.MultipartForm.Value {
				log.Printf("[%s] found multipart form value %q", requestID, k)

				if req.Payload == nil {
					req.Payload = make(map[string]interface{})
				}

				// TODO(moorereason): support duplicate, named values
				req.Payload[k] = v[0]
			}

			for k, v := range r.MultipartForm.File {
				// Force parsing as JSON regardless of Content-Type.
				var parseAsJSON bool
				for _, j := range matchedHook.JSONStringParameters {
					if j.Source == "payload" && j.Name == k {
						parseAsJSON = true
						break
					}
				}

				// TODO(moorereason): we need to support multiple parts
				// with the same name instead of just processing the first
				// one. Will need #215 resolved first.

				// MIME encoding can contain duplicate headers, so check them
				// all.
				if !parseAsJSON && len(v[0].Header["Content-Type"]) > 0 {
					for _, j := range v[0].Header["Content-Type"] {
						if j == "application/json" {
							parseAsJSON = true
							break
						}
					}
				}

				if parseAsJSON {
					log.Printf("[%s] parsing multipart form file %q as JSON\n", requestID, k)

					f, err := v[0].Open()
					if err != nil {
						msg := fmt.Sprintf("[%s] error parsing multipart form file: %+v\n", requestID, err)
						log.Println(msg)
						w.WriteHeader(http.StatusInternalServerError)
						fmt.Fprint(w, "Error occurred while parsing multipart form file.")
						return
					}

					decoder := json.NewDecoder(f)
					decoder.UseNumber()

					var part map[string]interface{}
					err = decoder.Decode(&part)
					if err != nil {
						log.Printf("[%s] error parsing JSON payload file: %+v\n", requestID, err)
					}

					if req.Payload == nil {
						req.Payload = make(map[string]interface{})
					}
					req.Payload[k] = part
				}
			}

		default:
			logContent := fmt.Sprintf("[%s] error parsing body payload due to unsupported content type header: %s\n", requestID, req.ContentType)
			log.Println(fn.RemoveNewlinesAndTabs(logContent))
		}

		// handle hook
		errs := matchedHook.ParseJSONParameters(req)
		for _, err := range errs {
			log.Printf("[%s] error parsing JSON parameters: %s\n", requestID, err)
		}

		var ok bool

		if matchedHook.TriggerRule == nil {
			ok = true
		} else {
			// Save signature soft failures option in request for evaluators
			req.AllowSignatureErrors = matchedHook.TriggerSignatureSoftFailures

			ok, err = matchedHook.TriggerRule.Evaluate(req)
			if err != nil {
				if !hook.IsParameterNodeError(err) {
					msg := fmt.Sprintf("[%s] error evaluating hook: %s", requestID, err)
					log.Println(msg)
					w.WriteHeader(http.StatusInternalServerError)
					fmt.Fprint(w, "Error occurred while evaluating hook rules.")
					return
				}

				log.Printf("[%s] %v", requestID, err)
			}
		}

		if ok {
			log.Printf("[%s] %s hook triggered successfully\n", requestID, matchedHook.ID)

			for _, responseHeader := range matchedHook.ResponseHeaders {
				w.Header().Set(responseHeader.Name, responseHeader.Value)
			}

			// 使用请求的 context，支持取消和超时
			ctx := r.Context()

			// 获取执行超时时间
			executionTimeout := time.Duration(appFlags.HookExecutionTimeout) * time.Second
			if executionTimeout <= 0 {
				executionTimeout = HookExecutionTimeout
			}

			if matchedHook.StreamCommandOutput {
				// 使用 trackingResponseWriter 来跟踪是否已经写入响应
				trw := &trackingResponseWriter{ResponseWriter: w}
				_, err := executor.Execute(ctx, matchedHook, req, trw, executionTimeout)
				if err != nil {
					// 如果还没有写入响应，可以设置错误状态码
					if !trw.HasWritten() {
						if errors.Is(err, context.DeadlineExceeded) {
							log.Printf("[%s] hook execution timeout: %v", requestID, err)
							w.Header().Set("Content-Type", "text/plain; charset=utf-8")
							w.WriteHeader(http.StatusRequestTimeout)
							fmt.Fprint(w, "Hook execution timeout. Please check your logs for more details.")
						} else if errors.Is(err, context.Canceled) {
							log.Printf("[%s] hook execution cancelled: %v", requestID, err)
							w.Header().Set("Content-Type", "text/plain; charset=utf-8")
							w.WriteHeader(http.StatusRequestTimeout)
							fmt.Fprint(w, "Hook execution cancelled. Please check your logs for more details.")
						} else {
							log.Printf("[%s] error executing hook: %v", requestID, err)
							w.Header().Set("Content-Type", "text/plain; charset=utf-8")
							w.WriteHeader(http.StatusInternalServerError)
							fmt.Fprint(w, "Error occurred while executing the hook's stream command. Please check your logs for more details.")
						}
					} else {
						// 如果已经开始输出，只能记录错误，无法设置状态码
						if errors.Is(err, context.DeadlineExceeded) {
							log.Printf("[%s] hook execution timeout (output already started): %v", requestID, err)
						} else if errors.Is(err, context.Canceled) {
							log.Printf("[%s] hook execution cancelled (output already started): %v", requestID, err)
						} else {
							log.Printf("[%s] error executing hook (output already started): %v", requestID, err)
						}
						// 尝试刷新输出
						if f, ok := w.(http.Flusher); ok {
							f.Flush()
						}
					}
				}
			} else if matchedHook.CaptureCommandOutput {
				response, err := executor.Execute(ctx, matchedHook, req, nil, executionTimeout)

				if err != nil {
					// 在 WriteHeader 之前设置所有 headers
					w.Header().Set("Content-Type", "text/plain; charset=utf-8")
					w.WriteHeader(http.StatusInternalServerError)
					if errors.Is(err, context.DeadlineExceeded) {
						log.Printf("[%s] hook execution timeout: %v", requestID, err)
						fmt.Fprint(w, "Hook execution timeout. Please check your logs for more details.")
					} else if matchedHook.CaptureCommandOutputOnError {
						fmt.Fprint(w, response)
					} else {
						fmt.Fprint(w, "Error occurred while executing the hook's command. Please check your logs for more details.")
					}
				} else {
					// Check if a success return code is configured for the hook
					if matchedHook.SuccessHttpResponseCode != 0 {
						writeHttpResponseCode(w, requestID, matchedHook.ID, matchedHook.SuccessHttpResponseCode)
					}
					fmt.Fprint(w, response)
				}
			} else {
				// 异步执行，但仍需要并发控制和超时
				go func() {
					_, err := executor.Execute(ctx, matchedHook, req, nil, executionTimeout)
					if err != nil {
						if errors.Is(err, context.DeadlineExceeded) {
							log.Printf("[%s] hook execution timeout: %v", requestID, err)
						} else if errors.Is(err, context.Canceled) {
							log.Printf("[%s] async hook execution cancelled due to request context: %v", requestID, err)
						} else {
							log.Printf("[%s] error executing hook: %v", requestID, err)
						}
					}
				}()

				// Check if a success return code is configured for the hook
				if matchedHook.SuccessHttpResponseCode != 0 {
					writeHttpResponseCode(w, requestID, matchedHook.ID, matchedHook.SuccessHttpResponseCode)
				}

				fmt.Fprint(w, matchedHook.ResponseMessage)
			}
			return
		}

		// Check if a return code is configured for the hook
		if matchedHook.TriggerRuleMismatchHttpResponseCode != 0 {
			writeHttpResponseCode(w, requestID, matchedHook.ID, matchedHook.TriggerRuleMismatchHttpResponseCode)
		}

		// if none of the hooks got triggered
		log.Printf("[%s] %s got matched, but didn't get triggered because the trigger rules were not satisfied\n", requestID, matchedHook.ID)

		fmt.Fprint(w, "Hook rules were not satisfied.")
	}
}

func makeSureCallable(h *hook.Hook, r *hook.Request, appFlags flags.AppFlags) (string, error) {
	// check the command exists
	var lookpath string
	if filepath.IsAbs(h.ExecuteCommand) || h.CommandWorkingDirectory == "" {
		lookpath = h.ExecuteCommand
	} else {
		lookpath = filepath.Join(h.CommandWorkingDirectory, h.ExecuteCommand)
	}

	cmdPath, err := exec.LookPath(lookpath)
	if err != nil {
		log.Printf("[%s] error in %s", r.ID, err)

		if strings.Contains(err.Error(), "permission denied") {
			if !appFlags.AllowAutoChmod {
				log.Printf("[%s] SECURITY WARNING: Command file '%s' is not executable. Auto-chmod is disabled for security reasons. Please manually set correct file permissions (chmod +x) or enable auto-chmod with --allow-auto-chmod flag (NOT RECOMMENDED)", r.ID, lookpath)
				return "", fmt.Errorf("permission denied: file '%s' is not executable and auto-chmod is disabled", lookpath)
			}

			// SECURITY WARNING: Only modify permissions if explicitly enabled
			log.Printf("[%s] SECURITY WARNING: Automatically modifying file permissions for '%s' (auto-chmod is enabled). This is a security risk and should be avoided in production.", r.ID, lookpath)
			// try to make the command executable
			// #nosec G302 - file permissions are intentionally modified when AllowAutoChmod is enabled
			err2 := os.Chmod(lookpath, 0o755)
			if err2 != nil {
				log.Printf("[%s] make command script executable error in %s", r.ID, err2)
				return "", err
			}

			log.Printf("[%s] make command script executable success", r.ID)
			// retry
			return makeSureCallable(h, r, appFlags)
		}

		// check if parameters specified in execute-command by mistake
		if strings.IndexByte(h.ExecuteCommand, ' ') != -1 {
			s := strings.Fields(h.ExecuteCommand)[0]
			log.Printf("[%s] use 'pass-arguments-to-command' to specify args for '%s'", r.ID, s)
		}

		return "", err
	}

	return cmdPath, nil
}

func handleHook(ctx context.Context, h *hook.Hook, r *hook.Request, w http.ResponseWriter, appFlags flags.AppFlags) (string, error) {
	var errs []error

	cmdPath, err := makeSureCallable(h, r, appFlags)
	if err != nil {
		return "", err
	}

	// 使用 exec.CommandContext 替代 exec.Command，支持超时和取消
	cmd := exec.CommandContext(ctx, cmdPath)
	cmd.Dir = h.CommandWorkingDirectory

	cmd.Args, errs = h.ExtractCommandArguments(r)
	for _, err := range errs {
		log.Printf("[%s] error extracting command arguments: %s\n", r.ID, err)
	}

	var envs []string
	envs, errs = h.ExtractCommandArgumentsForEnv(r)

	for _, err := range errs {
		log.Printf("[%s] error extracting command arguments for environment: %s\n", r.ID, err)
	}

	files, errs := h.ExtractCommandArgumentsForFile(r)

	for _, err := range errs {
		log.Printf("[%s] error extracting command arguments for file: %s\n", r.ID, err)
	}

	// 跟踪所有创建的临时文件，确保在任何情况下都能被清理
	var tempFileNames []string

	// 使用 defer 确保临时文件在任何情况下都会被清理
	defer func() {
		for _, fileName := range tempFileNames {
			log.Printf("[%s] removing temp file %s\n", r.ID, fileName)
			err := os.Remove(fileName)
			if err != nil {
				// 如果文件不存在（可能已经被删除），这是正常的，只记录警告
				if !os.IsNotExist(err) {
					log.Printf("[%s] error removing temp file %s [%s]", r.ID, fileName, err)
				}
			}
		}
		// 同时清理 files 中记录的文件（双重保险）
		for i := range files {
			if files[i].File != nil {
				fileName := files[i].File.Name()
				// 确保文件句柄已关闭
				_ = files[i].File.Close()
				// 如果文件名不在 tempFileNames 中，也尝试清理（避免重复）
				found := false
				for _, name := range tempFileNames {
					if name == fileName {
						found = true
						break
					}
				}
				if !found {
					log.Printf("[%s] removing file %s (from files array)\n", r.ID, fileName)
					if err := os.Remove(fileName); err != nil && !os.IsNotExist(err) {
						log.Printf("[%s] error removing file %s [%s]", r.ID, fileName, err)
					}
				}
			}
		}
	}()

	for i := range files {
		tmpfile, err := os.CreateTemp(h.CommandWorkingDirectory, files[i].EnvName)
		if err != nil {
			log.Printf("[%s] error creating temp file [%s]", r.ID, err)
			continue
		}

		// 立即记录文件名，确保即使后续步骤失败也能清理
		fileName := tmpfile.Name()
		tempFileNames = append(tempFileNames, fileName)

		log.Printf("[%s] writing env %s file %s", r.ID, files[i].EnvName, fileName)
		if _, err := tmpfile.Write(files[i].Data); err != nil {
			log.Printf("[%s] error writing file %s [%s]", r.ID, fileName, err)
			// 确保文件关闭后再删除
			_ = tmpfile.Close()
			if removeErr := os.Remove(fileName); removeErr != nil {
				log.Printf("[%s] error removing failed temp file %s [%s]", r.ID, fileName, removeErr)
			}
			// 从列表中移除，避免 defer 中重复删除
			tempFileNames = tempFileNames[:len(tempFileNames)-1]
			continue
		}

		if err := tmpfile.Close(); err != nil {
			log.Printf("[%s] error closing file %s [%s]", r.ID, fileName, err)
			// 尝试删除文件（即使关闭失败）
			if removeErr := os.Remove(fileName); removeErr != nil {
				log.Printf("[%s] error removing failed temp file %s [%s]", r.ID, fileName, removeErr)
			}
			// 从列表中移除，避免 defer 中重复删除
			tempFileNames = tempFileNames[:len(tempFileNames)-1]
			continue
		}

		// 文件创建成功，保存到 files 数组
		files[i].File = tmpfile
		envs = append(envs, files[i].EnvName+"="+fileName)
	}

	cmd.Env = append(os.Environ(), envs...)

	logsContent := fmt.Sprintf("[%s] executing %s (%s) with arguments %q and environment %s using %s as cwd\n", r.ID, h.ExecuteCommand, cmd.Path, cmd.Args, envs, cmd.Dir)
	log.Println(fn.RemoveNewlinesAndTabs(logsContent))

	var out []byte
	if w != nil {
		log.Printf("[%s] command output will be streamed to response", r.ID)

		// Implementation from https://play.golang.org/p/PpbPyXbtEs
		// as described in https://stackoverflow.com/questions/19292113/not-buffered-http-responsewritter-in-golang
		fw := flushWriter{w: w}
		if f, ok := w.(http.Flusher); ok {
			fw.f = f
		}
		cmd.Stderr = &fw
		cmd.Stdout = &fw

		if err := cmd.Run(); err != nil {
			// 检查是否是超时错误
			if ctx.Err() == context.DeadlineExceeded {
				log.Printf("[%s] command execution timeout: %+v\n", r.ID, err)
				return "", context.DeadlineExceeded
			} else if ctx.Err() == context.Canceled {
				log.Printf("[%s] command execution canceled: %+v\n", r.ID, err)
				return "", context.Canceled
			}
			log.Printf("[%s] error occurred: %+v\n", r.ID, err)
		}
	} else {
		out, err = cmd.CombinedOutput()

		log.Printf("[%s] command output: %s\n", r.ID, out)

		if err != nil {
			// 检查是否是超时错误
			if ctx.Err() == context.DeadlineExceeded {
				log.Printf("[%s] command execution timeout: %+v\n", r.ID, err)
				return string(out), context.DeadlineExceeded
			} else if ctx.Err() == context.Canceled {
				log.Printf("[%s] command execution canceled: %+v\n", r.ID, err)
				return string(out), context.Canceled
			}
			log.Printf("[%s] error occurred: %+v\n", r.ID, err)
		}
	}

	log.Printf("[%s] finished handling %s\n", r.ID, h.ID)

	return string(out), err
}

func writeHttpResponseCode(w http.ResponseWriter, rid, hookId string, responseCode int) {
	// Check if the given return code is supported by the http package
	// by testing if there is a StatusText for this code.
	if len(http.StatusText(responseCode)) > 0 {
		w.WriteHeader(responseCode)
	} else {
		log.Printf("[%s] %s got matched, but the configured return code %d is unknown - defaulting to 200\n", rid, hookId, responseCode)
	}
}
