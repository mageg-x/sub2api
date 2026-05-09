package provider

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

// PKCEChallenge 根据 PKCE code_verifier 生成 code_challenge
// 使用 SHA-256 哈希和 Base64URL 编码（无填充）
func PKCEChallenge(verifier string) string {
	sum := sha256.Sum256([]byte(verifier))
	return base64.RawURLEncoding.EncodeToString(sum[:])
}

// FormRequest 发送 application/x-www-form-urlencoded 格式的 POST 请求
// 用于 OAuth Token 交换等场景
// 返回解析后的键值对（所有值转为字符串）
func FormRequest(ctx context.Context, client *http.Client, endpoint string, form url.Values) (map[string]string, error) {
	// 构建表单请求
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, strings.NewReader(form.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	// 检查响应状态码
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("oauth request failed: %s", strings.TrimSpace(string(body)))
	}
	// 解析 JSON 响应
	var data map[string]any
	if err := json.Unmarshal(body, &data); err != nil {
		return nil, err
	}
	// 将所有值转为字符串
	result := map[string]string{}
	for k, v := range data {
		switch val := v.(type) {
		case string:
			result[k] = val
		case float64:
			result[k] = strconv.FormatFloat(val, 'f', -1, 64) // 数字转字符串
		default:
			result[k] = fmt.Sprintf("%v", v) // 其他类型用默认格式
		}
	}
	return result, nil
}

// JSONRequest 发送 application/json 格式的 POST 请求
// 用于 OAuth 刷新 Token 等场景
// 返回原始响应体
func JSONRequest(ctx context.Context, client *http.Client, endpoint string, payload any) ([]byte, error) {
	// 序列化请求体
	raw, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	// 构建 JSON 请求
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, strings.NewReader(string(raw)))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	// 检查响应状态码
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("oauth request failed: %s", strings.TrimSpace(string(body)))
	}
	return body, nil
}

// ParseExpires 解析过期时间字符串为整数
// 用于解析 OAuth 响应中的 expires_in 字段
func ParseExpires(raw string) int64 {
	if strings.TrimSpace(raw) == "" {
		return 0 // 空字符串返回 0
	}
	var n int64
	fmt.Sscanf(raw, "%d", &n) // 提取数字部分
	return n
}
