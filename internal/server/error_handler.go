package server

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/soulteary/webhook/internal/hook"
	"github.com/soulteary/webhook/internal/security"
)

// ErrorType 定义错误类型
type ErrorType string

const (
	// ErrorTypeClient 客户端错误（4xx）
	ErrorTypeClient ErrorType = "client"
	// ErrorTypeServer 服务器错误（5xx）
	ErrorTypeServer ErrorType = "server"
	// ErrorTypeTimeout 超时错误（408/504）
	ErrorTypeTimeout ErrorType = "timeout"
)

// ErrorResponse 标准错误响应格式
type ErrorResponse struct {
	Error     string `json:"error"`
	Message   string `json:"message,omitempty"`
	RequestID string `json:"request_id,omitempty"`
	HookID    string `json:"hook_id,omitempty"`
}

// HTTPError 封装HTTP错误信息
type HTTPError struct {
	Type      ErrorType
	Status    int
	Message   string
	Err       error
	RequestID string
	HookID    string
}

func (e *HTTPError) Error() string {
	if e.Err != nil {
		return e.Err.Error()
	}
	return e.Message
}

// NewHTTPError 创建新的HTTP错误
func NewHTTPError(errType ErrorType, status int, message string, err error) *HTTPError {
	return &HTTPError{
		Type:    errType,
		Status:  status,
		Message: message,
		Err:     err,
	}
}

// WithRequestID 设置请求ID
func (e *HTTPError) WithRequestID(requestID string) *HTTPError {
	e.RequestID = requestID
	return e
}

// WithHookID 设置Hook ID
func (e *HTTPError) WithHookID(hookID string) *HTTPError {
	e.HookID = hookID
	return e
}

// ClassifyError 根据错误类型分类错误
func ClassifyError(err error, requestID, hookID string) *HTTPError {
	if err == nil {
		return nil
	}

	// 检查是否是已分类的HTTPError
	if httpErr, ok := err.(*HTTPError); ok {
		httpErr.RequestID = requestID
		httpErr.HookID = hookID
		return httpErr
	}

	// 检查是否是context超时/取消错误
	if errors.Is(err, context.DeadlineExceeded) {
		return NewHTTPError(ErrorTypeTimeout, http.StatusRequestTimeout,
			"Request timeout. Please check your logs for more details.", err).
			WithRequestID(requestID).WithHookID(hookID)
	}
	if errors.Is(err, context.Canceled) {
		return NewHTTPError(ErrorTypeTimeout, http.StatusRequestTimeout,
			"Request cancelled. Please check your logs for more details.", err).
			WithRequestID(requestID).WithHookID(hookID)
	}

	// 检查是否是hook相关的错误
	if hook.IsParameterNodeError(err) {
		// 参数节点错误通常是客户端问题（缺少必需参数）
		return NewHTTPError(ErrorTypeClient, http.StatusBadRequest,
			"Invalid request parameters.", err).
			WithRequestID(requestID).WithHookID(hookID)
	}

	if hook.IsSignatureError(err) {
		// 签名错误通常是客户端问题（无效签名）
		return NewHTTPError(ErrorTypeClient, http.StatusUnauthorized,
			"Invalid payload signature.", err).
			WithRequestID(requestID).WithHookID(hookID)
	}

	// 检查是否是命令验证错误
	if security.IsCommandValidationError(err) {
		// 命令验证错误通常是服务器配置问题
		return NewHTTPError(ErrorTypeServer, http.StatusInternalServerError,
			"Command validation failed. Please check your logs for more details.", err).
			WithRequestID(requestID).WithHookID(hookID)
	}

	// 检查错误消息中是否包含客户端错误的特征
	errMsg := err.Error()
	if contains(errMsg, []string{"permission denied", "not found", "invalid", "bad request", "unauthorized", "forbidden"}) {
		return NewHTTPError(ErrorTypeClient, http.StatusBadRequest,
			"Invalid request.", err).
			WithRequestID(requestID).WithHookID(hookID)
	}

	// 默认作为服务器错误
	return NewHTTPError(ErrorTypeServer, http.StatusInternalServerError,
		"An internal server error occurred. Please check your logs for more details.", err).
		WithRequestID(requestID).WithHookID(hookID)
}

// HandleError 统一处理错误并写入HTTP响应
func HandleError(w http.ResponseWriter, r *http.Request, err error, requestID, hookID string) {
	if err == nil {
		return
	}

	httpErr := ClassifyError(err, requestID, hookID)

	// 记录错误日志
	logError(httpErr)

	// 设置响应头
	w.Header().Set("Content-Type", "application/json; charset=utf-8")

	// 写入状态码和响应体
	w.WriteHeader(httpErr.Status)

	// 创建错误响应
	errorResp := ErrorResponse{
		Error:     http.StatusText(httpErr.Status),
		Message:   httpErr.Message,
		RequestID: requestID,
		HookID:    hookID,
	}

	// 序列化为JSON
	if jsonErr := json.NewEncoder(w).Encode(errorResp); jsonErr != nil {
		// 如果JSON编码失败，回退到纯文本
		log.Printf("[%s] error encoding error response to JSON: %v", requestID, jsonErr)
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		fmt.Fprintf(w, "%s: %s", errorResp.Error, errorResp.Message)
	}
}

// HandleErrorPlain 处理错误并返回纯文本响应（用于向后兼容）
func HandleErrorPlain(w http.ResponseWriter, err error, requestID, hookID string) {
	if err == nil {
		return
	}

	httpErr := ClassifyError(err, requestID, hookID)

	// 记录错误日志
	logError(httpErr)

	// 设置响应头
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")

	// 写入状态码和响应体
	w.WriteHeader(httpErr.Status)
	fmt.Fprint(w, httpErr.Message)
}

// HandleErrorWithCustomMessage 处理错误并使用自定义消息
func HandleErrorWithCustomMessage(w http.ResponseWriter, err error, requestID, hookID, customMessage string, statusCode int) {
	if err == nil {
		return
	}

	httpErr := ClassifyError(err, requestID, hookID)
	httpErr.Message = customMessage
	if statusCode > 0 {
		httpErr.Status = statusCode
	}

	// 记录错误日志
	logError(httpErr)

	// 设置响应头
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")

	// 写入状态码和响应体
	w.WriteHeader(httpErr.Status)
	fmt.Fprint(w, httpErr.Message)
}

// logError 记录错误日志
func logError(httpErr *HTTPError) {
	if httpErr == nil {
		return
	}

	var logMsg string
	if httpErr.RequestID != "" {
		logMsg = fmt.Sprintf("[%s] ", httpErr.RequestID)
	}

	if httpErr.HookID != "" {
		logMsg += fmt.Sprintf("hook %s: ", httpErr.HookID)
	}

	logMsg += fmt.Sprintf("%s error (status: %d): %s", httpErr.Type, httpErr.Status, httpErr.Message)

	if httpErr.Err != nil {
		logMsg += fmt.Sprintf(" - %v", httpErr.Err)
	}

	switch httpErr.Type {
	case ErrorTypeClient:
		log.Printf("%s", logMsg)
	case ErrorTypeServer:
		log.Printf("%s", logMsg)
	case ErrorTypeTimeout:
		log.Printf("%s", logMsg)
	default:
		log.Printf("%s", logMsg)
	}
}

// contains 检查字符串是否包含任一子字符串
func contains(s string, substrs []string) bool {
	for _, substr := range substrs {
		if strings.Contains(s, substr) {
			return true
		}
	}
	return false
}

// IsClientError 检查错误是否为客户端错误
func IsClientError(err error) bool {
	httpErr, ok := err.(*HTTPError)
	return ok && httpErr.Type == ErrorTypeClient
}

// IsServerError 检查错误是否为服务器错误
func IsServerError(err error) bool {
	httpErr, ok := err.(*HTTPError)
	return ok && httpErr.Type == ErrorTypeServer
}

// IsTimeoutError 检查错误是否为超时错误
func IsTimeoutError(err error) bool {
	httpErr, ok := err.(*HTTPError)
	return ok && httpErr.Type == ErrorTypeTimeout
}
