package middleware

import (
	"bytes"
	"encoding/json"
	"regexp"
	"strings"
)

// 敏感字段关键词列表（不区分大小写）
var sensitiveKeywords = []string{
	"password",
	"passwd",
	"pwd",
	"secret",
	"token",
	"key",
	"auth",
	"authorization",
	"api_key",
	"apikey",
	"access_token",
	"access_token_secret",
	"refresh_token",
	"private_key",
	"privatekey",
	"credential",
	"credentials",
	"session",
	"cookie",
	"bearer",
	"x-api-key",
	"x-auth-token",
}

// SanitizeString 脱敏字符串中的敏感信息
// 如果字符串包含敏感关键词，则将其替换为 "***"
func SanitizeString(s string) string {
	if s == "" {
		return s
	}

	// 检查是否是键值对格式（使用 = 或 : 分隔）
	if idx := strings.IndexAny(s, "=:"); idx > 0 {
		key := s[:idx]
		lowerKey := strings.ToLower(strings.TrimSpace(key))

		// 检查键是否包含敏感关键词
		for _, keyword := range sensitiveKeywords {
			// 对于 "key" 这个通用词，只有当它是更长的敏感关键词的一部分时才脱敏
			// 例如 "api_key" 应该脱敏，但单独的 "key" 不应该
			if keyword == "key" {
				// 检查是否是组合词（如 "api_key", "private_key"）
				if lowerKey != "key" && strings.Contains(lowerKey, keyword) {
					return key + string(s[idx]) + "***"
				}
			} else if strings.Contains(lowerKey, keyword) {
				// 键包含敏感关键词，脱敏值
				return key + string(s[idx]) + "***"
			}
		}
		// 键不包含敏感关键词，返回原字符串
		return s
	}

	// 不是键值对格式，检查整个字符串是否包含敏感关键词
	lowerS := strings.ToLower(s)
	for _, keyword := range sensitiveKeywords {
		if strings.Contains(lowerS, keyword) {
			return "***"
		}
	}

	return s
}

// SanitizeHeader 脱敏HTTP头中的敏感信息
func SanitizeHeader(key, value string) string {
	lowerKey := strings.ToLower(key)
	for _, keyword := range sensitiveKeywords {
		if strings.Contains(lowerKey, keyword) {
			return "***"
		}
	}
	return value
}

// SanitizeJSON 脱敏JSON字符串中的敏感字段值
func SanitizeJSON(jsonStr string) string {
	if jsonStr == "" {
		return jsonStr
	}

	// 尝试解析JSON
	var jsonObj interface{}
	if err := json.Unmarshal([]byte(jsonStr), &jsonObj); err != nil {
		// 如果不是有效的JSON，使用简单的字符串替换
		return SanitizeString(jsonStr)
	}

	// 递归脱敏JSON对象
	sanitized := sanitizeJSONValue(jsonObj)

	// 重新序列化为JSON
	result, err := json.Marshal(sanitized)
	if err != nil {
		// 如果序列化失败，返回脱敏后的字符串
		return "***"
	}

	return string(result)
}

// sanitizeJSONValue 递归脱敏JSON值
func sanitizeJSONValue(v interface{}) interface{} {
	switch val := v.(type) {
	case map[string]interface{}:
		result := make(map[string]interface{})
		for k, v := range val {
			lowerKey := strings.ToLower(k)
			// 检查键是否包含敏感关键词
			isSensitive := false
			for _, keyword := range sensitiveKeywords {
				if strings.Contains(lowerKey, keyword) {
					isSensitive = true
					break
				}
			}
			if isSensitive {
				result[k] = "***"
			} else {
				result[k] = sanitizeJSONValue(v)
			}
		}
		return result
	case []interface{}:
		result := make([]interface{}, len(val))
		for i, item := range val {
			result[i] = sanitizeJSONValue(item)
		}
		return result
	case string:
		// 如果是字符串，检查是否包含敏感信息
		return SanitizeString(val)
	default:
		return val
	}
}

// SanitizeRequestLine 脱敏HTTP请求行
func SanitizeRequestLine(line string) string {
	// 请求行通常不包含敏感信息，但为了安全起见，检查URL参数
	if strings.Contains(line, "?") {
		parts := strings.SplitN(line, "?", 2)
		if len(parts) == 2 {
			// 脱敏查询参数
			query := SanitizeQueryString(parts[1])
			return parts[0] + "?" + query
		}
	}
	return line
}

// SanitizeQueryString 脱敏查询字符串
func SanitizeQueryString(query string) string {
	if query == "" {
		return query
	}

	// 解析查询参数
	parts := strings.Split(query, "&")
	sanitized := make([]string, 0, len(parts))

	for _, part := range parts {
		if idx := strings.Index(part, "="); idx > 0 {
			key := part[:idx]
			lowerKey := strings.ToLower(key)
			// 检查键是否包含敏感关键词
			isSensitive := false
			for _, keyword := range sensitiveKeywords {
				if strings.Contains(lowerKey, keyword) {
					isSensitive = true
					break
				}
			}
			if isSensitive {
				sanitized = append(sanitized, key+"=***")
			} else {
				sanitized = append(sanitized, part)
			}
		} else {
			sanitized = append(sanitized, part)
		}
	}

	return strings.Join(sanitized, "&")
}

// SanitizeRequestBody 脱敏请求体
// contentType: Content-Type头，用于判断请求体格式
// body: 原始请求体
// includeBody: 是否包含请求体（如果为false，则返回空字符串）
func SanitizeRequestBody(contentType string, body []byte, includeBody bool) string {
	if !includeBody {
		return ""
	}

	if len(body) == 0 {
		return ""
	}

	bodyStr := string(body)

	// 根据Content-Type选择脱敏策略
	lowerContentType := strings.ToLower(contentType)
	if strings.Contains(lowerContentType, "json") {
		return SanitizeJSON(bodyStr)
	} else if strings.Contains(lowerContentType, "x-www-form-urlencoded") {
		return SanitizeQueryString(bodyStr)
	} else if strings.Contains(lowerContentType, "multipart/form-data") {
		// Multipart表单数据比较复杂，简单处理：如果包含敏感关键词则脱敏
		return SanitizeString(bodyStr)
	} else {
		// 其他类型，使用通用脱敏
		return SanitizeString(bodyStr)
	}
}

// SanitizeDumpRequest 脱敏httputil.DumpRequest的输出
func SanitizeDumpRequest(dump []byte, includeBody bool) []byte {
	if len(dump) == 0 {
		return dump
	}

	lines := bytes.Split(dump, []byte("\n"))
	result := make([][]byte, 0, len(lines))
	var contentType string
	var bodyStartIdx int = -1

	// 第一遍：找到Content-Type和请求体位置
	for i, line := range lines {
		lineStr := string(line)
		// 查找Content-Type头
		if strings.HasPrefix(strings.ToLower(lineStr), "content-type:") {
			parts := strings.SplitN(lineStr, ":", 2)
			if len(parts) == 2 {
				contentType = strings.TrimSpace(parts[1])
			}
		}
		// 查找空行（请求头和请求体的分隔符）
		if lineStr == "" && bodyStartIdx == -1 {
			bodyStartIdx = i + 1
			break // 找到第一个空行即可
		}
	}

	// 第二遍：处理每一行
	for i, line := range lines {
		lineStr := string(line)

		// 如果已经到达请求体，根据配置决定是否处理
		if bodyStartIdx != -1 && i >= bodyStartIdx {
			if includeBody {
				// 收集所有请求体行
				bodyLines := make([]string, 0)
				for j := bodyStartIdx; j < len(lines); j++ {
					bodyLines = append(bodyLines, string(lines[j]))
				}
				bodyStr := strings.Join(bodyLines, "\n")
				sanitizedBody := SanitizeRequestBody(contentType, []byte(bodyStr), true)
				if sanitizedBody != "" {
					result = append(result, []byte(sanitizedBody))
				}
			}
			// 无论是否包含请求体，都停止处理
			break
		}

		// 处理请求行
		if i == 0 && (strings.HasPrefix(lineStr, "GET") || strings.HasPrefix(lineStr, "POST") ||
			strings.HasPrefix(lineStr, "PUT") || strings.HasPrefix(lineStr, "DELETE") ||
			strings.HasPrefix(lineStr, "PATCH") || strings.HasPrefix(lineStr, "HEAD") ||
			strings.HasPrefix(lineStr, "OPTIONS")) {
			result = append(result, []byte(SanitizeRequestLine(lineStr)))
			continue
		}

		// 处理HTTP头
		if strings.Contains(lineStr, ":") {
			parts := strings.SplitN(lineStr, ":", 2)
			if len(parts) == 2 {
				key := strings.TrimSpace(parts[0])
				value := strings.TrimSpace(parts[1])
				sanitizedValue := SanitizeHeader(key, value)
				result = append(result, []byte(key+": "+sanitizedValue))
				continue
			}
		}

		// 其他行（如空行）直接添加
		result = append(result, line)
	}

	return bytes.Join(result, []byte("\n"))
}

// SanitizeError 脱敏错误消息中的敏感信息
func SanitizeError(errMsg string) string {
	if errMsg == "" {
		return errMsg
	}

	// 使用正则表达式查找可能的敏感信息模式
	patterns := []*regexp.Regexp{
		// 匹配 password=xxx 或 password:xxx
		regexp.MustCompile(`(?i)(password|passwd|pwd)\s*[=:]\s*[^\s,;]+`),
		// 匹配 token=xxx 或 token:xxx
		regexp.MustCompile(`(?i)(token|secret|key|auth)\s*[=:]\s*[^\s,;]+`),
		// 匹配 API key 模式
		regexp.MustCompile(`(?i)(api[_-]?key|apikey)\s*[=:]\s*[^\s,;]+`),
		// 匹配 Bearer token
		regexp.MustCompile(`(?i)bearer\s+[^\s,;]+`),
	}

	result := errMsg
	for _, pattern := range patterns {
		result = pattern.ReplaceAllStringFunc(result, func(match string) string {
			// 提取键和值
			parts := regexp.MustCompile(`[=:]`).Split(match, 2)
			if len(parts) == 2 {
				return parts[0] + "=***"
			}
			return "***"
		})
	}

	return result
}
