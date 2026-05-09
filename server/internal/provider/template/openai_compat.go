package template

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"sub2api/server/internal/provider"
)

// ==================== OpenAI 公开协议类型 ====================

// openAIResponsesRequest OpenAI Responses API 请求
type openAIResponsesRequest struct {
	Model  string `json:"model"`            // 模型名称
	Stream bool   `json:"stream,omitempty"` // 是否流式
}

// openAIResponsesResponse OpenAI Responses API 响应
type openAIResponsesResponse struct {
	ID     string                  `json:"id"`              // 响应 ID
	Object string                  `json:"object"`          // 对象类型
	Model  string                  `json:"model"`           // 模型名称
	Status string                  `json:"status"`          // 响应状态
	Output []openAIResponsesOutput `json:"output"`          // 输出项列表
	Usage  *openAIResponsesUsage   `json:"usage,omitempty"` // 用量信息
}

// openAIResponsesOutput Responses API 输出项
type openAIResponsesOutput struct {
	Type    string                       `json:"type"`              // 输出类型（message/function_call）
	ID      string                       `json:"id,omitempty"`      // 输出项 ID
	Role    string                       `json:"role,omitempty"`    // 角色（assistant）
	Content []openAIResponsesContentPart `json:"content,omitempty"` // 内容列表
	Status  string                       `json:"status,omitempty"`  // 状态
}

// openAIResponsesContentPart Responses API 内容部分
type openAIResponsesContentPart struct {
	Type string `json:"type"`           // 内容类型（output_text）
	Text string `json:"text,omitempty"` // 文本内容
}

// openAIResponsesUsage Responses API 用量信息
type openAIResponsesUsage struct {
	InputTokens  int `json:"input_tokens"`  // 输入 Token 数
	OutputTokens int `json:"output_tokens"` // 输出 Token 数
	TotalTokens  int `json:"total_tokens"`  // 总 Token 数
}

// openAIResponsesStreamEvent Responses API 流式事件
type openAIResponsesStreamEvent struct {
	Type     string                   `json:"type"`               // 事件类型
	Response *openAIResponsesResponse `json:"response,omitempty"` // 响应数据
}

// openAIChatCompletionsRequest OpenAI Chat Completions API 请求
type openAIChatCompletionsRequest struct {
	Model         string                  `json:"model"`                    // 模型名称
	Stream        bool                    `json:"stream,omitempty"`         // 是否流式
	Messages      []openAIChatMessage     `json:"messages"`                 // 消息列表
	StreamOptions *openAIChatStreamOption `json:"stream_options,omitempty"` // 流式选项
}

// openAIChatStreamOption Chat Completions 流式选项
type openAIChatStreamOption struct {
	IncludeUsage bool `json:"include_usage,omitempty"` // 是否包含用量信息
}

// openAIChatCompletionsResponse OpenAI Chat Completions API 响应
type openAIChatCompletionsResponse struct {
	ID      string             `json:"id"`              // 响应 ID
	Object  string             `json:"object"`          // 对象类型
	Created int64              `json:"created"`         // 创建时间戳
	Model   string             `json:"model"`           // 模型名称
	Choices []openAIChatChoice `json:"choices"`         // 选择项列表
	Usage   *openAIChatUsage   `json:"usage,omitempty"` // 用量信息
}

// openAIChatChoice Chat Completions 选择项
type openAIChatChoice struct {
	Index        int               `json:"index"`         // 选择项索引
	Message      openAIChatMessage `json:"message"`       // 消息内容
	FinishReason string            `json:"finish_reason"` // 完成原因
}

// openAIChatMessage Chat Completions 消息
type openAIChatMessage struct {
	Role    string          `json:"role"`              // 角色
	Content json.RawMessage `json:"content,omitempty"` // 内容（JSON 原始格式）
}

// openAIChatUsage Chat Completions 用量信息
type openAIChatUsage struct {
	PromptTokens     int `json:"prompt_tokens"`     // 提示 Token 数
	CompletionTokens int `json:"completion_tokens"` // 完成 Token 数
	TotalTokens      int `json:"total_tokens"`      // 总 Token 数
}

// openAIChatCompletionsChunk Chat Completions 流式增量块
type openAIChatCompletionsChunk struct {
	ID      string `json:"id"`      // 响应 ID
	Object  string `json:"object"`  // 对象类型
	Created int64  `json:"created"` // 创建时间戳
	Model   string `json:"model"`   // 模型名称
}

// ==================== 上游原生协议类型 ====================

// upstreamRequest 上游原生请求格式
type upstreamRequest struct {
	Model  string `json:"model"`            // 模型名称
	Stream bool   `json:"stream,omitempty"` // 是否流式
}

// upstreamUsage 上游用量信息
type upstreamUsage struct {
	InputTokens  int `json:"input_tokens"`  // 输入 Token 数
	OutputTokens int `json:"output_tokens"` // 输出 Token 数
}

// upstreamResponse 上游原生响应格式
type upstreamResponse struct {
	ID      string          `json:"id"`              // 响应 ID
	Model   string          `json:"model"`           // 模型名称
	Content string          `json:"content"`         // 文本内容
	Usage   *upstreamUsage  `json:"usage,omitempty"` // 用量信息
	Raw     json.RawMessage `json:"-"`               // 原始 JSON 数据（不序列化）
}

// 编译期断言：如果这个模板声明需要协议回写，后续实现时必须补齐。
var _ provider.GatewayResponseAdapter = (*Provider)(nil)

// ==================== 路径标准化与请求转换 ====================

// normalizeOpenAIPath 归一化 OpenAI 路径
// /chat/completions → /v1/chat/completions
// /responses → /v1/responses
// /responses/* → /v1/responses/*
func normalizeOpenAIPath(path string) string {
	switch {
	case path == "/chat/completions":
		return "/v1/chat/completions"
	case path == "/responses":
		return "/v1/responses"
	case strings.HasPrefix(path, "/responses/"):
		return "/v1" + path
	default:
		return path
	}
}

// convertOpenAIRequest 将 OpenAI 格式请求转换为上游原生格式
// 返回：转换后的请求体、模型名、是否流式、是否包含用量、错误
func convertOpenAIRequest(publicPath string, body []byte) ([]byte, string, bool, bool, error) {
	switch normalizeOpenAIPath(publicPath) {
	case "/v1/chat/completions":
		// Chat Completions 请求转换
		var req openAIChatCompletionsRequest
		if err := json.Unmarshal(body, &req); err != nil {
			return nil, "", false, false, err
		}
		upstream := upstreamRequest{
			Model:  strings.TrimSpace(req.Model),
			Stream: req.Stream,
		}
		converted, err := json.Marshal(upstream)
		if err != nil {
			return nil, "", false, false, err
		}
		includeUsage := req.StreamOptions != nil && req.StreamOptions.IncludeUsage
		return converted, upstream.Model, upstream.Stream, includeUsage, nil
	case "/v1/responses", "/backend-api/codex/responses":
		// Responses 请求转换
		var req openAIResponsesRequest
		if err := json.Unmarshal(body, &req); err != nil {
			return nil, "", false, false, err
		}
		upstream := upstreamRequest{
			Model:  strings.TrimSpace(req.Model),
			Stream: req.Stream,
		}
		converted, err := json.Marshal(upstream)
		if err != nil {
			return nil, "", false, false, err
		}
		return converted, upstream.Model, upstream.Stream, false, nil
	default:
		// 非转换路径原样返回
		return body, "", false, false, nil
	}
}

// AdaptGatewayResponse 负责把上游非流式响应转换为公开协议响应。
// Chat Completions 路径：上游 → Responses → Chat Completions
// Responses 路径：上游 → Responses
func (p *Provider) AdaptGatewayResponse(req provider.GatewayRequest, resp *http.Response, body []byte) (*http.Response, []byte, error) {
	switch normalizeOpenAIPath(req.PublicPath) {
	case "/v1/chat/completions":
		return p.adaptUpstreamToOpenAIChatResponse(req, resp, body)
	case "/v1/responses", "/backend-api/codex/responses":
		return p.adaptUpstreamToOpenAIResponsesResponse(resp, body)
	default:
		return resp, body, nil // 非转换路径原样返回
	}
}

// AdaptGatewayStream 负责把上游 SSE 流转换为公开协议 SSE。
func (p *Provider) AdaptGatewayStream(req provider.GatewayRequest, resp *http.Response) (*http.Response, error) {
	switch normalizeOpenAIPath(req.PublicPath) {
	case "/v1/chat/completions":
		return p.wrapUpstreamToOpenAIChatStream(req, resp), nil
	case "/v1/responses", "/backend-api/codex/responses":
		return p.wrapUpstreamToOpenAIResponsesStream(resp), nil
	default:
		return resp, nil
	}
}

// ==================== 响应转换 ====================

// adaptUpstreamToOpenAIResponsesResponse 将上游响应转换为 OpenAI Responses API 格式
func (p *Provider) adaptUpstreamToOpenAIResponsesResponse(resp *http.Response, body []byte) (*http.Response, []byte, error) {
	if resp.StatusCode >= 400 {
		return resp, body, nil // 错误响应原样返回
	}
	upstream, err := parseUpstreamResponse(body)
	if err != nil {
		return resp, body, err
	}
	out := upstreamToResponsesResponse(upstream)
	converted, err := json.Marshal(out)
	if err != nil {
		return resp, body, err
	}
	resp.Header = cloneHTTPHeader(resp.Header)
	resp.Header.Set("Content-Type", "application/json")
	resp.ContentLength = int64(len(converted))
	return resp, converted, nil
}

// adaptUpstreamToOpenAIChatResponse 将上游响应转换为 OpenAI Chat Completions 格式
// 先转换为 Responses 格式，再转换为 Chat Completions 格式
func (p *Provider) adaptUpstreamToOpenAIChatResponse(req provider.GatewayRequest, resp *http.Response, body []byte) (*http.Response, []byte, error) {
	// 第一步：上游 → Responses
	resp, converted, err := p.adaptUpstreamToOpenAIResponsesResponse(resp, body)
	if err != nil || resp.StatusCode >= 400 {
		return resp, converted, err
	}
	// 第二步：Responses → Chat Completions
	var responsesResp openAIResponsesResponse
	if err := json.Unmarshal(converted, &responsesResp); err != nil {
		return resp, converted, err
	}
	chatResp := responsesToChatCompletions(&responsesResp, req.Model)
	chatBody, err := json.Marshal(chatResp)
	if err != nil {
		return resp, converted, err
	}
	resp.Header = cloneHTTPHeader(resp.Header)
	resp.Header.Set("Content-Type", "application/json")
	resp.ContentLength = int64(len(chatBody))
	return resp, chatBody, nil
}

// parseUpstreamResponse 解析上游响应体
func parseUpstreamResponse(body []byte) (*upstreamResponse, error) {
	var out upstreamResponse
	if err := json.Unmarshal(body, &out); err != nil {
		return nil, err
	}
	out.Raw = body // 保存原始 JSON
	return &out, nil
}

// upstreamToResponsesResponse 将上游响应转换为 Responses API 响应
func upstreamToResponsesResponse(resp *upstreamResponse) *openAIResponsesResponse {
	id := firstNonEmpty(strings.TrimSpace(resp.ID), "resp_template")
	out := &openAIResponsesResponse{
		ID:     id,
		Object: "response",
		Model:  strings.TrimSpace(resp.Model),
		Status: "completed",
		// 构建输出项：将上游内容包装为 message 输出项
		Output: []openAIResponsesOutput{{
			Type:   "message",
			ID:     "item_message",
			Role:   "assistant",
			Status: "completed",
			Content: []openAIResponsesContentPart{{
				Type: "output_text",
				Text: resp.Content,
			}},
		}},
	}
	// 设置用量信息
	if resp.Usage != nil {
		out.Usage = &openAIResponsesUsage{
			InputTokens:  resp.Usage.InputTokens,
			OutputTokens: resp.Usage.OutputTokens,
			TotalTokens:  resp.Usage.InputTokens + resp.Usage.OutputTokens,
		}
	}
	return out
}

// responsesToChatCompletions 将 Responses API 响应转换为 Chat Completions 格式
func responsesToChatCompletions(resp *openAIResponsesResponse, model string) *openAIChatCompletionsResponse {
	out := &openAIChatCompletionsResponse{
		ID:      firstNonEmpty(strings.TrimSpace(resp.ID), "chatcmpl_template"),
		Object:  "chat.completion",
		Created: time.Now().Unix(),
		Model:   firstNonEmpty(strings.TrimSpace(model), strings.TrimSpace(resp.Model)),
	}
	// 提取所有文本内容
	var text string
	for _, item := range resp.Output {
		if item.Type != "message" {
			continue
		}
		for _, part := range item.Content {
			if part.Type == "output_text" {
				text += part.Text
			}
		}
	}
	raw, _ := json.Marshal(text)
	// 构建选择项
	out.Choices = []openAIChatChoice{{
		Index: 0,
		Message: openAIChatMessage{
			Role:    "assistant",
			Content: raw,
		},
		FinishReason: "stop",
	}}
	// 设置用量信息
	if resp.Usage != nil {
		out.Usage = &openAIChatUsage{
			PromptTokens:     resp.Usage.InputTokens,
			CompletionTokens: resp.Usage.OutputTokens,
			TotalTokens:      resp.Usage.TotalTokens,
		}
	}
	return out
}

// ==================== SSE 转换 ====================

// wrapUpstreamToOpenAIResponsesStream 包装上游 SSE 流为 Responses API SSE 格式
func (p *Provider) wrapUpstreamToOpenAIResponsesStream(resp *http.Response) *http.Response {
	if resp.StatusCode >= 400 || !isSSEHeader(resp.Header) {
		return resp // 错误或非 SSE 响应原样返回
	}
	resp.Header = cloneHTTPHeader(resp.Header)
	resp.Header.Set("Content-Type", "text/event-stream; charset=utf-8")
	resp.ContentLength = -1 // 流式响应长度未知
	return resp
}

// wrapUpstreamToOpenAIChatStream 包装上游 SSE 流为 Chat Completions SSE 格式
func (p *Provider) wrapUpstreamToOpenAIChatStream(_ provider.GatewayRequest, resp *http.Response) *http.Response {
	if resp.StatusCode >= 400 || !isSSEHeader(resp.Header) {
		return resp
	}
	resp.Header = cloneHTTPHeader(resp.Header)
	resp.Header.Set("Content-Type", "text/event-stream; charset=utf-8")
	resp.ContentLength = -1
	return resp
}

// ==================== 辅助函数 ====================

// firstNonEmpty 返回第一个非空字符串
func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

// cloneHTTPHeader 深拷贝 HTTP 头
func cloneHTTPHeader(src http.Header) http.Header {
	if src == nil {
		return make(http.Header)
	}
	dst := make(http.Header, len(src))
	for k, values := range src {
		copied := make([]string, len(values))
		copy(copied, values)
		dst[k] = copied
	}
	return dst
}

// isSSEHeader 检查是否为 SSE 响应头
func isSSEHeader(header http.Header) bool {
	return strings.Contains(strings.ToLower(header.Get("Content-Type")), "event-stream")
}

// nopReadCloser 无操作关闭的 ReadCloser 包装
type nopReadCloser struct {
	io.Reader
}

func (n nopReadCloser) Close() error { return nil }

// formatSSEEvent 格式化 SSE 事件
func formatSSEEvent(event, data string) string {
	return fmt.Sprintf("event: %s\ndata: %s\n\n", event, data)
}
