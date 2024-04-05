package server

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

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

func createHookHandler(appFlags flags.AppFlags) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		requestID := middleware.GetReqID(r.Context())
		req := &hook.Request{
			ID:         requestID,
			RawRequest: r,
		}

		log.Printf("[%s] incoming HTTP %s request from %s\n", requestID, r.Method, r.RemoteAddr)

		hookID := strings.TrimSpace(mux.Vars(r)["id"])
		hookID = fn.GetEscapedLogItem(hookID)

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
			log.Printf("[%s] error parsing body payload due to unsupported content type header: %s\n", requestID, req.ContentType)
		}

		// handle hook
		errors := matchedHook.ParseJSONParameters(req)
		for _, err := range errors {
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

			if matchedHook.StreamCommandOutput {
				_, err := handleHook(matchedHook, req, w)
				if err != nil {
					fmt.Fprint(w, "Error occurred while executing the hook's stream command. Please check your logs for more details.")
				}
			} else if matchedHook.CaptureCommandOutput {
				response, err := handleHook(matchedHook, req, nil)

				if err != nil {
					w.WriteHeader(http.StatusInternalServerError)
					if matchedHook.CaptureCommandOutputOnError {
						fmt.Fprint(w, response)
					} else {
						w.Header().Set("Content-Type", "text/plain; charset=utf-8")
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
				go handleHook(matchedHook, req, nil)

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

func handleHook(h *hook.Hook, r *hook.Request, w http.ResponseWriter) (string, error) {
	var errors []error

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

		// check if parameters specified in execute-command by mistake
		if strings.IndexByte(h.ExecuteCommand, ' ') != -1 {
			s := strings.Fields(h.ExecuteCommand)[0]
			log.Printf("[%s] use 'pass-arguments-to-command' to specify args for '%s'", r.ID, s)
		}

		return "", err
	}

	cmd := exec.Command(cmdPath)
	cmd.Dir = h.CommandWorkingDirectory

	cmd.Args, errors = h.ExtractCommandArguments(r)
	for _, err := range errors {
		log.Printf("[%s] error extracting command arguments: %s\n", r.ID, err)
	}

	var envs []string
	envs, errors = h.ExtractCommandArgumentsForEnv(r)

	for _, err := range errors {
		log.Printf("[%s] error extracting command arguments for environment: %s\n", r.ID, err)
	}

	files, errors := h.ExtractCommandArgumentsForFile(r)

	for _, err := range errors {
		log.Printf("[%s] error extracting command arguments for file: %s\n", r.ID, err)
	}

	for i := range files {
		tmpfile, err := os.CreateTemp(h.CommandWorkingDirectory, files[i].EnvName)
		if err != nil {
			log.Printf("[%s] error creating temp file [%s]", r.ID, err)
			continue
		}
		log.Printf("[%s] writing env %s file %s", r.ID, files[i].EnvName, tmpfile.Name())
		if _, err := tmpfile.Write(files[i].Data); err != nil {
			log.Printf("[%s] error writing file %s [%s]", r.ID, tmpfile.Name(), err)
			continue
		}
		if err := tmpfile.Close(); err != nil {
			log.Printf("[%s] error closing file %s [%s]", r.ID, tmpfile.Name(), err)
			continue
		}

		files[i].File = tmpfile
		envs = append(envs, files[i].EnvName+"="+tmpfile.Name())
	}

	cmd.Env = append(os.Environ(), envs...)

	log.Printf("[%s] executing %s (%s) with arguments %q and environment %s using %s as cwd\n", r.ID, h.ExecuteCommand, cmd.Path, cmd.Args, envs, cmd.Dir)

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
			log.Printf("[%s] error occurred: %+v\n", r.ID, err)
		}
	} else {
		out, err = cmd.CombinedOutput()

		log.Printf("[%s] command output: %s\n", r.ID, out)

		if err != nil {
			log.Printf("[%s] error occurred: %+v\n", r.ID, err)
		}
	}

	for i := range files {
		if files[i].File != nil {
			log.Printf("[%s] removing file %s\n", r.ID, files[i].File.Name())
			err := os.Remove(files[i].File.Name())
			if err != nil {
				log.Printf("[%s] error removing file %s [%s]", r.ID, files[i].File.Name(), err)
			}
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
