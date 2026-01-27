package middleware

// Derived from from the Goa project, MIT Licensed
// https://github.com/goadesign/goa/blob/v3/http/middleware/debug.go

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httputil"
	"sort"
	"strings"

	loggerkit "github.com/soulteary/logger-kit"
)

// responseDupper tees the response to a buffer and a response writer.
type responseDupper struct {
	http.ResponseWriter
	Buffer *bytes.Buffer
	Status int
}

// DumperConfig 配置Dumper中间件的行为
type DumperConfig struct {
	IncludeRequestBody bool // 是否包含请求体（默认false，避免敏感信息泄露）
}

// Dumper returns a debug middleware which prints detailed information about
// incoming requests and outgoing responses including all headers, parameters
// and bodies. 敏感信息会被自动脱敏。
func Dumper(w io.Writer) func(http.Handler) http.Handler {
	return DumperWithConfig(w, DumperConfig{
		IncludeRequestBody: false, // 默认不包含请求体，避免敏感信息泄露
	})
}

// DumperWithConfig returns a debug middleware with custom configuration.
func DumperWithConfig(w io.Writer, config DumperConfig) func(http.Handler) http.Handler {
	return func(h http.Handler) http.Handler {
		return http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
			buf := &bytes.Buffer{}
			// Request ID：优先使用本包注入的 ID，否则从 logger-kit 读取
			rid := GetReqID(r.Context())
			if rid == "" {
				rid = loggerkit.RequestIDFromRequest(r)
			}

			// Dump request (包含请求体)
			bd, err := httputil.DumpRequest(r, true)
			if err != nil {
				buf.WriteString(fmt.Sprintf("[%s] Error dumping request for debugging: %s\n", rid, err))
			} else {
				// 脱敏请求转储
				sanitized := SanitizeDumpRequest(bd, config.IncludeRequestBody)

				sc := bufio.NewScanner(bytes.NewBuffer(sanitized))
				sc.Split(bufio.ScanLines)
				for sc.Scan() {
					line := sc.Text()
					// 如果配置不包含请求体，跳过空行后的内容
					if !config.IncludeRequestBody && line == "" {
						// 找到空行，停止处理（空行后是请求体）
						break
					}
					buf.WriteString(fmt.Sprintf("> [%s] ", rid))
					buf.WriteString(line + "\n")
				}

				// 如果不包含请求体，添加提示
				if !config.IncludeRequestBody && len(bd) > 0 {
					buf.WriteString(fmt.Sprintf("> [%s] [Request body omitted for security - use --log-request-body to include]\n", rid))
				}
			}

			_, err = w.Write(buf.Bytes())
			if err != nil {
				fmt.Println("Error writing to debug writer before buf reset: ", err)
			}
			buf.Reset()

			// Dump Response

			dupper := &responseDupper{ResponseWriter: rw, Buffer: &bytes.Buffer{}}
			h.ServeHTTP(dupper, r)

			// Response Status
			buf.WriteString(fmt.Sprintf("< [%s] %d %s\n", rid, dupper.Status, http.StatusText(dupper.Status)))

			// Response Headers
			keys := make([]string, len(dupper.Header()))
			i := 0
			for k := range dupper.Header() {
				keys[i] = k
				i++
			}
			sort.Strings(keys)
			for _, k := range keys {
				buf.WriteString(fmt.Sprintf("< [%s] %s: %s\n", rid, k, strings.Join(dupper.Header()[k], ", ")))
			}

			// Response Body (脱敏处理)
			if dupper.Buffer.Len() > 0 {
				responseBody := dupper.Buffer.Bytes()
				// 获取响应Content-Type
				responseContentType := dupper.Header().Get("Content-Type")
				// 脱敏响应体
				sanitizedBody := SanitizeRequestBody(responseContentType, responseBody, config.IncludeRequestBody)

				if sanitizedBody != "" {
					buf.WriteString(fmt.Sprintf("< [%s]\n", rid))
					sc := bufio.NewScanner(bytes.NewBufferString(sanitizedBody))
					sc.Split(bufio.ScanLines)
					for sc.Scan() {
						buf.WriteString(fmt.Sprintf("< [%s] ", rid))
						buf.WriteString(sc.Text() + "\n")
					}
				} else if !config.IncludeRequestBody {
					buf.WriteString(fmt.Sprintf("< [%s] [Response body omitted for security]\n", rid))
				}
			}
			_, err = w.Write(buf.Bytes())
			if err != nil {
				fmt.Println("Error writing to debug writer: ", err)
			}
		})
	}
}

// Write writes the data to the buffer and connection as part of an HTTP reply.
func (r *responseDupper) Write(b []byte) (int, error) {
	r.Buffer.Write(b)
	return r.ResponseWriter.Write(b)
}

// WriteHeader records the status and sends an HTTP response header with status code.
func (r *responseDupper) WriteHeader(s int) {
	r.Status = s
	r.ResponseWriter.WriteHeader(s)
}

// Hijack supports the http.Hijacker interface.
func (r *responseDupper) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	if hijacker, ok := r.ResponseWriter.(http.Hijacker); ok {
		return hijacker.Hijack()
	}
	return nil, nil, fmt.Errorf("dumper middleware: inner ResponseWriter cannot be hijacked: %T", r.ResponseWriter)
}
