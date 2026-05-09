package claude

import (
	"bufio"
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"sub2api/server/internal/provider"
)

// AnthropicRequest Anthropic Claude 原生 API 请求格式
type AnthropicRequest struct {
	Model        string                 `json:"model"`
	MaxTokens    int                    `json:"max_tokens"`
	System       json.RawMessage        `json:"system,omitempty"`         // 系统提示词
	Messages     []AnthropicMessage     `json:"messages"`                 // 消息列表
	Tools        []AnthropicTool        `json:"tools,omitempty"`          // 工具定义
	Stream       bool                   `json:"stream,omitempty"`         // 是否流式
	Temperature  *float64               `json:"temperature,omitempty"`    // 温度
	TopP         *float64               `json:"top_p,omitempty"`          // Top-P 采样
	StopSeqs     []string               `json:"stop_sequences,omitempty"` // 停止序列
	Thinking     *AnthropicThinking     `json:"thinking,omitempty"`       // 思考模式配置
	ToolChoice   json.RawMessage        `json:"tool_choice,omitempty"`    // 工具选择策略
	Metadata     json.RawMessage        `json:"metadata,omitempty"`       // 元数据
	OutputConfig *AnthropicOutputConfig `json:"output_config,omitempty"`  // 输出配置
}

// AnthropicOutputConfig 输出配置（如推理力度）
type AnthropicOutputConfig struct {
	Effort string `json:"effort,omitempty"` // low/medium/high/max
}

// AnthropicThinking 思考模式配置
type AnthropicThinking struct {
	Type         string `json:"type"`                    // enabled/disabled
	BudgetTokens int    `json:"budget_tokens,omitempty"` // 思考 Token 预算
}

// AnthropicMessage Anthropic 消息格式
type AnthropicMessage struct {
	Role    string          `json:"role"`    // user/assistant
	Content json.RawMessage `json:"content"` // 内容（文本或内容块数组）
}

// AnthropicContentBlock Anthropic 内容块
// 支持文本、思考、图片、工具调用、工具结果等多种类型
type AnthropicContentBlock struct {
	Type string `json:"type"`

	Text      string                `json:"text,omitempty"`      // 文本内容
	Thinking  string                `json:"thinking,omitempty"`  // 思考内容
	Signature string                `json:"signature,omitempty"` // 思考签名
	Source    *AnthropicImageSource `json:"source,omitempty"`    // 图片源

	ID    string          `json:"id,omitempty"`    // 工具调用 ID
	Name  string          `json:"name,omitempty"`  // 工具名称
	Input json.RawMessage `json:"input,omitempty"` // 工具输入参数

	ToolUseID string          `json:"tool_use_id,omitempty"` // 工具使用 ID
	Content   json.RawMessage `json:"content,omitempty"`     // 工具结果内容
	IsError   bool            `json:"is_error,omitempty"`    // 是否错误
}

// AnthropicImageSource Anthropic 图片源（Base64 格式）
type AnthropicImageSource struct {
	Type      string `json:"type"`       // base64
	MediaType string `json:"media_type"` // MIME 类型
	Data      string `json:"data"`       // Base64 数据
}

// AnthropicTool Anthropic 工具定义
type AnthropicTool struct {
	Type        string          `json:"type,omitempty"`        // 工具类型（如 web_search_20250305）
	Name        string          `json:"name"`                  // 工具名称
	Description string          `json:"description,omitempty"` // 工具描述
	InputSchema json.RawMessage `json:"input_schema"`          // 输入 JSON Schema
}

// AnthropicResponse Anthropic Claude 原生 API 响应格式
type AnthropicResponse struct {
	ID           string                  `json:"id"`
	Type         string                  `json:"type"`
	Role         string                  `json:"role"`
	Content      []AnthropicContentBlock `json:"content"` // 响应内容块列表
	Model        string                  `json:"model"`
	StopReason   string                  `json:"stop_reason"`             // 停止原因
	StopSequence *string                 `json:"stop_sequence,omitempty"` // 停止序列
	Usage        AnthropicUsage          `json:"usage"`                   // Token 用量
}

// AnthropicUsage Anthropic Token 用量
type AnthropicUsage struct {
	InputTokens              int `json:"input_tokens"`
	OutputTokens             int `json:"output_tokens"`
	CacheCreationInputTokens int `json:"cache_creation_input_tokens,omitempty"` // 缓存创建 Token
	CacheReadInputTokens     int `json:"cache_read_input_tokens,omitempty"`     // 缓存读取 Token
}

// AnthropicStreamEvent Anthropic 流式事件
type AnthropicStreamEvent struct {
	Type string `json:"type"`

	Message *AnthropicResponse `json:"message,omitempty"` // message_start 时携带完整消息

	Index        *int                   `json:"index,omitempty"`         // 内容块索引
	ContentBlock *AnthropicContentBlock `json:"content_block,omitempty"` // content_block_start 时携带

	Delta *AnthropicDelta `json:"delta,omitempty"` // 增量内容
	Usage *AnthropicUsage `json:"usage,omitempty"` // Token 用量（message_delta 时携带）
}

// AnthropicDelta Anthropic 流式增量内容
type AnthropicDelta struct {
	Type string `json:"type,omitempty"`

	Text         string  `json:"text,omitempty"`          // 文本增量
	PartialJSON  string  `json:"partial_json,omitempty"`  // 工具输入 JSON 增量
	Thinking     string  `json:"thinking,omitempty"`      // 思考增量
	Signature    string  `json:"signature,omitempty"`     // 思考签名
	StopReason   string  `json:"stop_reason,omitempty"`   // 停止原因
	StopSequence *string `json:"stop_sequence,omitempty"` // 停止序列
}

// openAIResponsesRequest OpenAI Responses API 请求格式
// openAIResponsesRequest OpenAI Responses API 请求格式
type openAIResponsesRequest struct {
	Model           string                    `json:"model"`
	Instructions    string                    `json:"instructions,omitempty"`      // 系统指令
	Input           json.RawMessage           `json:"input"`                       // 输入（字符串或消息数组）
	MaxOutputTokens *int                      `json:"max_output_tokens,omitempty"` // 最大输出 Token
	Temperature     *float64                  `json:"temperature,omitempty"`       // 温度
	TopP            *float64                  `json:"top_p,omitempty"`             // Top-P 采样
	Stream          bool                      `json:"stream,omitempty"`            // 是否流式
	Tools           []openAIResponsesTool     `json:"tools,omitempty"`             // 工具列表
	Include         []string                  `json:"include,omitempty"`           // 包含的额外信息
	Store           *bool                     `json:"store,omitempty"`             // 是否存储
	Reasoning       *openAIResponsesReasoning `json:"reasoning,omitempty"`         // 推理配置
	ToolChoice      json.RawMessage           `json:"tool_choice,omitempty"`       // 工具选择策略
	ServiceTier     string                    `json:"service_tier,omitempty"`      // 服务层级
}

// openAIResponsesReasoning 推理配置
type openAIResponsesReasoning struct {
	Effort  string `json:"effort"`            // 推理力度：low/medium/high
	Summary string `json:"summary,omitempty"` // 摘要模式：auto
}

// openAIResponsesInputItem Responses API 输入项
type openAIResponsesInputItem struct {
	Type      string          `json:"type,omitempty"`      // 消息类型
	Role      string          `json:"role,omitempty"`      // 角色
	Content   json.RawMessage `json:"content,omitempty"`   // 内容
	CallID    string          `json:"call_id,omitempty"`   // 函数调用 ID
	Name      string          `json:"name,omitempty"`      // 函数名
	Arguments string          `json:"arguments,omitempty"` // 函数参数
	ID        string          `json:"id,omitempty"`        // 消息 ID
	Output    string          `json:"output,omitempty"`    // 函数输出
}

// openAIResponsesContentPart Responses API 内容部分
type openAIResponsesContentPart struct {
	Type     string `json:"type"`                // input_text/input_image/output_text
	Text     string `json:"text,omitempty"`      // 文本内容
	ImageURL string `json:"image_url,omitempty"` // 图片 URL（data URI）
}

// openAIResponsesTool Responses API 工具定义
type openAIResponsesTool struct {
	Type        string          `json:"type"`                  // 工具类型
	Name        string          `json:"name,omitempty"`        // 工具名称
	Description string          `json:"description,omitempty"` // 工具描述
	Parameters  json.RawMessage `json:"parameters,omitempty"`  // 参数 Schema
	Strict      *bool           `json:"strict,omitempty"`      // 是否严格模式
}

// openAIResponsesResponse OpenAI Responses API 响应格式
type openAIResponsesResponse struct {
	ID                string                            `json:"id"`
	Object            string                            `json:"object"`
	Model             string                            `json:"model"`
	Status            string                            `json:"status"`                       // completed/incomplete
	Output            []openAIResponsesOutput           `json:"output"`                       // 输出项列表
	Usage             *openAIResponsesUsage             `json:"usage,omitempty"`              // Token 用量
	IncompleteDetails *openAIResponsesIncompleteDetails `json:"incomplete_details,omitempty"` // 未完成详情
	Error             *openAIResponsesError             `json:"error,omitempty"`              // 错误信息
}

// openAIResponsesError 错误信息
type openAIResponsesError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// openAIResponsesIncompleteDetails 未完成详情
type openAIResponsesIncompleteDetails struct {
	Reason string `json:"reason"` // max_output_tokens 等
}

// openAIResponsesOutput Responses API 输出项
type openAIResponsesOutput struct {
	Type      string                       `json:"type"`                // message/function_call/reasoning
	ID        string                       `json:"id,omitempty"`        // 输出项 ID
	Role      string                       `json:"role,omitempty"`      // 角色
	Content   []openAIResponsesContentPart `json:"content,omitempty"`   // 内容部分
	Status    string                       `json:"status,omitempty"`    // 状态
	Summary   []openAIResponsesSummary     `json:"summary,omitempty"`   // 推理摘要
	CallID    string                       `json:"call_id,omitempty"`   // 函数调用 ID
	Name      string                       `json:"name,omitempty"`      // 函数名
	Arguments string                       `json:"arguments,omitempty"` // 函数参数
	Action    *openAIWebSearchAction       `json:"action,omitempty"`    // 网络搜索动作
}

// openAIWebSearchAction 网络搜索动作
type openAIWebSearchAction struct {
	Type  string `json:"type,omitempty"` // 搜索类型
	Query string `json:"query,omitempty"`
}

// openAIResponsesSummary 推理摘要
type openAIResponsesSummary struct {
	Type string `json:"type"` // summary_text
	Text string `json:"text"`
}

// openAIResponsesUsage Responses API Token 用量
type openAIResponsesUsage struct {
	InputTokens         int                                `json:"input_tokens"`
	OutputTokens        int                                `json:"output_tokens"`
	TotalTokens         int                                `json:"total_tokens"`
	InputTokensDetails  *openAIResponsesInputTokenDetails  `json:"input_tokens_details,omitempty"`  // 输入 Token 详情
	OutputTokensDetails *openAIResponsesOutputTokenDetails `json:"output_tokens_details,omitempty"` // 输出 Token 详情
}

// openAIResponsesInputTokenDetails 输入 Token 详情
type openAIResponsesInputTokenDetails struct {
	CachedTokens int `json:"cached_tokens,omitempty"` // 缓存命中 Token
}

// openAIResponsesOutputTokenDetails 输出 Token 详情
type openAIResponsesOutputTokenDetails struct {
	ReasoningTokens int `json:"reasoning_tokens,omitempty"` // 推理 Token
}

// openAIResponsesStreamEvent Responses API 流式事件
type openAIResponsesStreamEvent struct {
	Type           string                   `json:"type"`                      // 事件类型
	Response       *openAIResponsesResponse `json:"response,omitempty"`        // 响应对象
	Item           *openAIResponsesOutput   `json:"item,omitempty"`            // 输出项
	OutputIndex    int                      `json:"output_index,omitempty"`    // 输出项索引
	ContentIndex   int                      `json:"content_index,omitempty"`   // 内容索引
	Delta          string                   `json:"delta,omitempty"`           // 增量文本
	Text           string                   `json:"text,omitempty"`            // 完整文本
	ItemID         string                   `json:"item_id,omitempty"`         // 输出项 ID
	CallID         string                   `json:"call_id,omitempty"`         // 函数调用 ID
	Name           string                   `json:"name,omitempty"`            // 函数名
	Arguments      string                   `json:"arguments,omitempty"`       // 函数参数
	SummaryIndex   int                      `json:"summary_index,omitempty"`   // 摘要索引
	Code           string                   `json:"code,omitempty"`            // 错误码
	Param          string                   `json:"param,omitempty"`           // 错误参数
	SequenceNumber int                      `json:"sequence_number,omitempty"` // 序列号
}

// openAIChatCompletionsRequest OpenAI Chat Completions API 请求格式
type openAIChatCompletionsRequest struct {
	Model               string                   `json:"model"`
	Messages            []openAIChatMessage      `json:"messages"`                        // 消息列表
	Instructions        string                   `json:"instructions,omitempty"`          // 系统指令
	MaxTokens           *int                     `json:"max_tokens,omitempty"`            // 最大 Token（旧版）
	MaxCompletionTokens *int                     `json:"max_completion_tokens,omitempty"` // 最大完成 Token（新版）
	Temperature         *float64                 `json:"temperature,omitempty"`           // 温度
	TopP                *float64                 `json:"top_p,omitempty"`                 // Top-P 采样
	Stream              bool                     `json:"stream,omitempty"`                // 是否流式
	StreamOptions       *openAIChatStreamOptions `json:"stream_options,omitempty"`        // 流式选项
	Tools               []openAIChatTool         `json:"tools,omitempty"`                 // 工具列表
	ToolChoice          json.RawMessage          `json:"tool_choice,omitempty"`           // 工具选择策略
	ReasoningEffort     string                   `json:"reasoning_effort,omitempty"`      // 推理力度
	ServiceTier         string                   `json:"service_tier,omitempty"`          // 服务层级
	Stop                json.RawMessage          `json:"stop,omitempty"`                  // 停止序列
	Functions           []openAIChatFunction     `json:"functions,omitempty"`             // 函数列表（旧版）
	FunctionCall        json.RawMessage          `json:"function_call,omitempty"`         // 函数调用策略（旧版）
}

// openAIChatStreamOptions 流式选项
type openAIChatStreamOptions struct {
	IncludeUsage bool `json:"include_usage,omitempty"` // 是否包含用量
}

// openAIChatMessage Chat Completions 消息
type openAIChatMessage struct {
	Role             string                  `json:"role"`                        // 角色
	Content          json.RawMessage         `json:"content,omitempty"`           // 内容
	ReasoningContent string                  `json:"reasoning_content,omitempty"` // 推理内容
	Name             string                  `json:"name,omitempty"`              // 名称
	ToolCalls        []openAIChatToolCall    `json:"tool_calls,omitempty"`        // 工具调用列表
	ToolCallID       string                  `json:"tool_call_id,omitempty"`      // 工具调用 ID
	FunctionCall     *openAIChatFunctionCall `json:"function_call,omitempty"`     // 函数调用（旧版）
}

// openAIChatContentPart Chat 内容部分
type openAIChatContentPart struct {
	Type     string              `json:"type"`                // text/image_url
	Text     string              `json:"text,omitempty"`      // 文本
	ImageURL *openAIChatImageURL `json:"image_url,omitempty"` // 图片 URL
}

// openAIChatImageURL 图片 URL
type openAIChatImageURL struct {
	URL    string `json:"url"`              // 图片 URL（data URI 或 HTTP URL）
	Detail string `json:"detail,omitempty"` // 图片精度：auto/low/high
}

// openAIChatTool Chat 工具定义
type openAIChatTool struct {
	Type     string              `json:"type"`               // function
	Function *openAIChatFunction `json:"function,omitempty"` // 函数定义
}

// openAIChatFunction Chat 函数定义
type openAIChatFunction struct {
	Name        string          `json:"name"`                  // 函数名
	Description string          `json:"description,omitempty"` // 函数描述
	Parameters  json.RawMessage `json:"parameters,omitempty"`  // 参数 Schema
	Strict      *bool           `json:"strict,omitempty"`      // 是否严格模式
}

// openAIChatToolCall Chat 工具调用
type openAIChatToolCall struct {
	Index    *int                   `json:"index,omitempty"` // 工具调用索引
	ID       string                 `json:"id,omitempty"`    // 调用 ID
	Type     string                 `json:"type,omitempty"`  // 类型
	Function openAIChatFunctionCall `json:"function"`        // 函数调用
}

// openAIChatFunctionCall Chat 函数调用
type openAIChatFunctionCall struct {
	Name      string `json:"name"`      // 函数名
	Arguments string `json:"arguments"` // 函数参数（JSON 字符串）
}

// openAIChatCompletionsResponse Chat Completions 响应格式
type openAIChatCompletionsResponse struct {
	ID                string             `json:"id"`
	Object            string             `json:"object"`
	Created           int64              `json:"created"`
	Model             string             `json:"model"`
	Choices           []openAIChatChoice `json:"choices"`                      // 选择列表
	Usage             *openAIChatUsage   `json:"usage,omitempty"`              // Token 用量
	SystemFingerprint string             `json:"system_fingerprint,omitempty"` // 系统指纹
	ServiceTier       string             `json:"service_tier,omitempty"`       // 服务层级
}

// openAIChatChoice Chat 选择项
type openAIChatChoice struct {
	Index        int               `json:"index"`         // 选择索引
	Message      openAIChatMessage `json:"message"`       // 消息
	FinishReason string            `json:"finish_reason"` // 完成原因
}

// openAIChatUsage Chat Token 用量
type openAIChatUsage struct {
	PromptTokens        int                    `json:"prompt_tokens"`
	CompletionTokens    int                    `json:"completion_tokens"`
	TotalTokens         int                    `json:"total_tokens"`
	PromptTokensDetails *openAIChatTokenDetail `json:"prompt_tokens_details,omitempty"` // 输入 Token 详情
}

// openAIChatTokenDetail Chat Token 详情
type openAIChatTokenDetail struct {
	CachedTokens int `json:"cached_tokens,omitempty"` // 缓存命中 Token
}

// openAIChatCompletionsChunk Chat Completions 流式块
// openAIChatCompletionsChunk Chat Completions 流式块
type openAIChatCompletionsChunk struct {
	ID                string                  `json:"id"`
	Object            string                  `json:"object"`
	Created           int64                   `json:"created"`
	Model             string                  `json:"model"`
	Choices           []openAIChatChunkChoice `json:"choices"`                      // 选择列表
	Usage             *openAIChatUsage        `json:"usage,omitempty"`              // Token 用量
	SystemFingerprint string                  `json:"system_fingerprint,omitempty"` // 系统指纹
	ServiceTier       string                  `json:"service_tier,omitempty"`       // 服务层级
}

// openAIChatChunkChoice Chat 流式选择项
type openAIChatChunkChoice struct {
	Index        int             `json:"index"`         // 选择索引
	Delta        openAIChatDelta `json:"delta"`         // 增量内容
	FinishReason *string         `json:"finish_reason"` // 完成原因（最后一个块才有）
}

// openAIChatDelta Chat 流式增量
type openAIChatDelta struct {
	Role             string               `json:"role,omitempty"`              // 角色
	Content          *string              `json:"content,omitempty"`           // 文本增量
	ReasoningContent *string              `json:"reasoning_content,omitempty"` // 推理增量
	ToolCalls        []openAIChatToolCall `json:"tool_calls,omitempty"`        // 工具调用增量
}

// openAIMinMaxOutputTokens 最小最大输出 Token 数
const openAIMinMaxOutputTokens = 128

// chatMessageContent 解析后的消息内容（纯文本或多部分内容）
type chatMessageContent struct {
	Text  *string                 // 纯文本内容
	Parts []openAIChatContentPart // 多部分内容
}

// parseAssistantContent 解析助手消息内容
// 支持纯文本和多部分内容（含 thinking 块，用 <thinking> 标签包裹）
func parseAssistantContent(raw json.RawMessage) (string, error) {
	if len(raw) == 0 {
		return "", nil
	}
	// 尝试解析为纯文本
	var s string
	if err := json.Unmarshal(raw, &s); err == nil {
		return s, nil
	}
	// 解析为多部分内容
	var parts []map[string]any
	if err := json.Unmarshal(raw, &parts); err != nil {
		return "", nil
	}
	var b strings.Builder
	for _, p := range parts {
		typ, _ := p["type"].(string)
		text, _ := p["text"].(string)
		thinking, _ := p["thinking"].(string)
		switch typ {
		case "thinking", "reasoning":
			// 思考内容用 <thinking> 标签包裹
			value := thinking
			if value == "" {
				value = text
			}
			if value != "" {
				b.WriteString("<thinking>")
				b.WriteString(value)
				b.WriteString("</thinking>")
			}
		default:
			if text != "" {
				b.WriteString(text)
			}
		}
	}
	return b.String(), nil
}

// parseChatContent 解析 Chat 消息内容为纯文本
// 优先使用纯文本，否则拼接多部分文本
func parseChatContent(raw json.RawMessage) (string, error) {
	parsed, err := parseChatMessageContent(raw)
	if err != nil {
		return "", err
	}
	if parsed.Text != nil {
		return *parsed.Text, nil // 纯文本直接返回
	}
	return flattenChatContentParts(parsed.Parts), nil // 拼接多部分文本
}

// parseChatMessageContent 解析 Chat 消息内容为 chatMessageContent
// 支持纯文本字符串和内容部分数组两种格式
func parseChatMessageContent(raw json.RawMessage) (chatMessageContent, error) {
	if len(raw) == 0 {
		return chatMessageContent{Text: stringPtr("")}, nil
	}
	// 尝试解析为纯文本
	var s string
	if err := json.Unmarshal(raw, &s); err == nil {
		return chatMessageContent{Text: &s}, nil
	}
	// 尝试解析为内容部分数组
	var parts []openAIChatContentPart
	if err := json.Unmarshal(raw, &parts); err == nil {
		return chatMessageContent{Parts: parts}, nil
	}
	return chatMessageContent{}, fmt.Errorf("parse content as string or parts array")
}

// marshalChatInputContent 将 chatMessageContent 序列化为 Responses API 格式
func marshalChatInputContent(content chatMessageContent) (json.RawMessage, error) {
	if content.Text != nil {
		return json.Marshal(*content.Text)
	}
	return json.Marshal(convertChatContentPartsToResponses(content.Parts))
}

// convertChatContentPartsToResponses 将 Chat 内容部分转换为 Responses API 格式
// text → input_text, image_url → input_image
func convertChatContentPartsToResponses(parts []openAIChatContentPart) []openAIResponsesContentPart {
	var responseParts []openAIResponsesContentPart
	for _, p := range parts {
		switch p.Type {
		case "text":
			if p.Text != "" {
				responseParts = append(responseParts, openAIResponsesContentPart{Type: "input_text", Text: p.Text})
			}
		case "image_url":
			// 跳过空的 Base64 data URI
			if p.ImageURL != nil && p.ImageURL.URL != "" && !isEmptyBase64DataURI(p.ImageURL.URL) {
				responseParts = append(responseParts, openAIResponsesContentPart{Type: "input_image", ImageURL: p.ImageURL.URL})
			}
		}
	}
	return responseParts
}

// isEmptyBase64DataURI 检查 data URI 是否为空的 Base64 数据
func isEmptyBase64DataURI(raw string) bool {
	if !strings.HasPrefix(raw, "data:") {
		return false
	}
	rest := strings.TrimPrefix(raw, "data:")
	semicolonIdx := strings.Index(rest, ";")
	if semicolonIdx < 0 {
		return false
	}
	mediaType := rest[:semicolonIdx] // 提取 MIME 类型
	_ = mediaType
	rest = rest[semicolonIdx+1:]
	if !strings.HasPrefix(rest, "base64,") {
		return false
	}
	// 检查 Base64 数据是否为空
	return strings.TrimSpace(strings.TrimPrefix(rest, "base64,")) == ""
}

// flattenChatContentParts 将多部分内容拼接为纯文本
func flattenChatContentParts(parts []openAIChatContentPart) string {
	var textParts []string
	for _, p := range parts {
		if p.Type == "text" && p.Text != "" {
			textParts = append(textParts, p.Text)
		}
	}
	return strings.Join(textParts, "")
}

// stringPtr 返回字符串指针
func stringPtr(s string) *string {
	return &s
}

// firstNonEmpty 返回第一个非空字符串
func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

// convertChatToolsToResponses 将 Chat 工具和函数转换为 Responses API 工具格式
// convertChatToolsToResponses 将 Chat 工具和函数转换为 Responses API 工具格式
func convertChatToolsToResponses(tools []openAIChatTool, functions []openAIChatFunction) []openAIResponsesTool {
	var out []openAIResponsesTool
	// 转换 tools 中的 function 类型
	for _, t := range tools {
		if t.Type != "function" || t.Function == nil {
			continue
		}
		out = append(out, openAIResponsesTool{
			Type:        "function",
			Name:        t.Function.Name,
			Description: t.Function.Description,
			Parameters:  t.Function.Parameters,
			Strict:      t.Function.Strict,
		})
	}
	// 转换旧版 functions
	for _, f := range functions {
		out = append(out, openAIResponsesTool{
			Type:        "function",
			Name:        f.Name,
			Description: f.Description,
			Parameters:  f.Parameters,
			Strict:      f.Strict,
		})
	}
	return out
}

// convertChatFunctionCallToToolChoice 将旧版 function_call 转换为 tool_choice 格式
func convertChatFunctionCallToToolChoice(raw json.RawMessage) (json.RawMessage, error) {
	// 尝试解析为字符串（如 "auto"、"none"）
	var s string
	if err := json.Unmarshal(raw, &s); err == nil {
		return json.Marshal(s)
	}
	// 解析为对象（如 {"name": "func_name"}）
	var obj struct {
		Name string `json:"name"`
	}
	if err := json.Unmarshal(raw, &obj); err != nil {
		return nil, err
	}
	return json.Marshal(map[string]any{
		"type": "function",
		"name": obj.Name,
	})
}

// convertOpenAIRequest 将 OpenAI 格式请求转换为 Anthropic 格式
// 支持 Chat Completions 和 Responses API 两种输入格式
func convertOpenAIRequest(publicPath string, body []byte) ([]byte, string, bool, bool, error) {
	switch normalizeOpenAIPath(publicPath) {
	case "/v1/chat/completions":
		// Chat Completions → Responses → Anthropic
		var req openAIChatCompletionsRequest
		if err := json.Unmarshal(body, &req); err != nil {
			return nil, "", false, false, err
		}
		// 第一步：Chat Completions → Responses
		responsesReq, err := chatCompletionsToResponses(&req)
		if err != nil {
			return nil, "", false, false, err
		}
		// 第二步：Responses → Anthropic
		anthReq, err := responsesToAnthropicRequest(responsesReq)
		if err != nil {
			return nil, "", false, false, err
		}
		anthReq.Model = firstNonEmpty(strings.TrimSpace(req.Model), strings.TrimSpace(anthReq.Model))
		anthReq.Stream = req.Stream
		converted, err := json.Marshal(anthReq)
		if err != nil {
			return nil, "", false, false, err
		}
		return converted, anthReq.Model, req.Stream, detectChatIncludeUsage(req), nil
	case "/v1/responses", "/backend-api/codex/responses":
		// Responses → Anthropic
		var req openAIResponsesRequest
		if err := json.Unmarshal(body, &req); err != nil {
			return nil, "", false, false, err
		}
		anthReq, err := responsesToAnthropicRequest(&req)
		if err != nil {
			return nil, "", false, false, err
		}
		anthReq.Stream = req.Stream
		converted, err := json.Marshal(anthReq)
		if err != nil {
			return nil, "", false, false, err
		}
		return converted, anthReq.Model, req.Stream, false, nil
	default:
		// 非转换路径，原样返回
		return body, "", false, false, nil
	}
}

// normalizeOpenAIPath 归一化 OpenAI 路径
// /chat/completions → /v1/chat/completions
// /responses → /v1/responses
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

// detectChatIncludeUsage 检测 Chat Completions 请求是否要求包含用量
func detectChatIncludeUsage(req openAIChatCompletionsRequest) bool {
	return req.StreamOptions != nil && req.StreamOptions.IncludeUsage
}

// chatCompletionsToResponses 将 Chat Completions 请求转换为 Responses API 请求
// 这是 OpenAI 格式转换的第一步
func chatCompletionsToResponses(req *openAIChatCompletionsRequest) (*openAIResponsesRequest, error) {
	// 转换消息列表
	input, err := convertChatMessagesToResponsesInput(req.Messages)
	if err != nil {
		return nil, err
	}
	inputJSON, err := json.Marshal(input)
	if err != nil {
		return nil, err
	}
	out := &openAIResponsesRequest{
		Model:        req.Model,
		Instructions: req.Instructions,
		Input:        inputJSON,
		Temperature:  req.Temperature,
		TopP:         req.TopP,
		Stream:       true,
		Include:      []string{"reasoning.encrypted_content"}, // 请求加密推理内容
		ServiceTier:  req.ServiceTier,
	}
	storeFalse := false
	out.Store = &storeFalse // 不存储响应
	// 处理最大 Token 数
	maxTokens := 0
	if req.MaxTokens != nil {
		maxTokens = *req.MaxTokens
	}
	if req.MaxCompletionTokens != nil {
		maxTokens = *req.MaxCompletionTokens // 新版字段优先
	}
	if maxTokens > 0 {
		if maxTokens < openAIMinMaxOutputTokens {
			maxTokens = openAIMinMaxOutputTokens // 最小值限制
		}
		out.MaxOutputTokens = &maxTokens
	}
	// 处理推理配置
	if req.ReasoningEffort != "" {
		out.Reasoning = &openAIResponsesReasoning{Effort: req.ReasoningEffort, Summary: "auto"}
	}
	// 转换工具
	if len(req.Tools) > 0 || len(req.Functions) > 0 {
		out.Tools = convertChatToolsToResponses(req.Tools, req.Functions)
	}
	// 转换工具选择策略
	if len(req.ToolChoice) > 0 {
		out.ToolChoice = req.ToolChoice
	} else if len(req.FunctionCall) > 0 {
		tc, err := convertChatFunctionCallToToolChoice(req.FunctionCall)
		if err != nil {
			return nil, fmt.Errorf("convert function_call: %w", err)
		}
		out.ToolChoice = tc
	}
	return out, nil
}

// convertChatMessagesToResponsesInput 将 Chat 消息列表转换为 Responses API 输入项
func convertChatMessagesToResponsesInput(msgs []openAIChatMessage) ([]openAIResponsesInputItem, error) {
	var out []openAIResponsesInputItem
	for _, m := range msgs {
		items, err := chatMessageToResponsesItems(m)
		if err != nil {
			return nil, err
		}
		out = append(out, items...)
	}
	return out, nil
}

// chatMessageToResponsesItems 根据角色将 Chat 消息转换为 Responses API 输入项
func chatMessageToResponsesItems(m openAIChatMessage) ([]openAIResponsesInputItem, error) {
	switch m.Role {
	case "system":
		return chatSystemToResponses(m)
	case "user":
		return chatUserToResponses(m)
	case "assistant":
		return chatAssistantToResponses(m)
	case "tool":
		return chatToolToResponses(m)
	case "function":
		return chatFunctionToResponses(m)
	default:
		return chatUserToResponses(m) // 未知角色按 user 处理
	}
}

// chatSystemToResponses 转换系统消息
func chatSystemToResponses(m openAIChatMessage) ([]openAIResponsesInputItem, error) {
	parsed, err := parseChatMessageContent(m.Content)
	if err != nil {
		return nil, err
	}
	content, err := marshalChatInputContent(parsed)
	if err != nil {
		return nil, err
	}
	return []openAIResponsesInputItem{{Role: "system", Content: content}}, nil
}

// chatUserToResponses 转换用户消息
func chatUserToResponses(m openAIChatMessage) ([]openAIResponsesInputItem, error) {
	parsed, err := parseChatMessageContent(m.Content)
	if err != nil {
		return nil, fmt.Errorf("parse user content: %w", err)
	}
	content, err := marshalChatInputContent(parsed)
	if err != nil {
		return nil, err
	}
	return []openAIResponsesInputItem{{Role: "user", Content: content}}, nil
}

// chatAssistantToResponses 转换助手消息
// 将文本内容和工具调用分别转换为对应的输入项
func chatAssistantToResponses(m openAIChatMessage) ([]openAIResponsesInputItem, error) {
	var items []openAIResponsesInputItem
	// 处理文本内容
	if len(m.Content) > 0 {
		s, err := parseAssistantContent(m.Content)
		if err != nil {
			return nil, err
		}
		if s != "" {
			parts := []openAIResponsesContentPart{{Type: "output_text", Text: s}}
			partsJSON, err := json.Marshal(parts)
			if err != nil {
				return nil, err
			}
			items = append(items, openAIResponsesInputItem{Role: "assistant", Content: partsJSON})
		}
	}
	// 处理工具调用
	for _, tc := range m.ToolCalls {
		args := tc.Function.Arguments
		if args == "" {
			args = "{}" // 默认空对象
		}
		items = append(items, openAIResponsesInputItem{
			Type:      "function_call",
			CallID:    tc.ID,
			Name:      tc.Function.Name,
			Arguments: args,
		})
	}
	return items, nil
}

// chatToolToResponses 转换工具结果消息
func chatToolToResponses(m openAIChatMessage) ([]openAIResponsesInputItem, error) {
	output, err := parseChatContent(m.Content)
	if err != nil {
		return nil, err
	}
	if output == "" {
		output = "(empty)" // 空结果用占位符
	}
	return []openAIResponsesInputItem{{
		Type:   "function_call_output",
		CallID: m.ToolCallID,
		Output: output,
	}}, nil
}

// chatFunctionToResponses 转换旧版函数结果消息
func chatFunctionToResponses(m openAIChatMessage) ([]openAIResponsesInputItem, error) {
	output, err := parseChatContent(m.Content)
	if err != nil {
		return nil, err
	}
	if output == "" {
		output = "(empty)"
	}
	// 旧版函数调用使用函数名作为 CallID
	return []openAIResponsesInputItem{{
		Type:   "function_call_output",
		CallID: m.Name,
		Output: output,
	}}, nil
}

// responsesToAnthropicRequest 将 Responses API 请求转换为 Anthropic 原生请求
// 这是 OpenAI 格式转换的第二步
func responsesToAnthropicRequest(req *openAIResponsesRequest) (*AnthropicRequest, error) {
	// 转换输入为 Anthropic 消息格式
	system, messages, err := convertResponsesInputToAnthropic(req.Input)
	if err != nil {
		return nil, err
	}
	out := &AnthropicRequest{
		Model:       req.Model,
		Messages:    messages,
		Temperature: req.Temperature,
		TopP:        req.TopP,
		Stream:      req.Stream,
	}
	// 设置系统提示词
	if len(system) > 0 {
		out.System = system
	}
	// 设置最大输出 Token
	if req.MaxOutputTokens != nil && *req.MaxOutputTokens > 0 {
		out.MaxTokens = *req.MaxOutputTokens
	}
	if out.MaxTokens == 0 {
		out.MaxTokens = 8192 // 默认值
	}
	// 转换工具
	if len(req.Tools) > 0 {
		out.Tools = convertResponsesToAnthropicTools(req.Tools)
	}
	// 转换工具选择策略
	if len(req.ToolChoice) > 0 {
		tc, err := convertResponsesToAnthropicToolChoice(req.ToolChoice)
		if err != nil {
			return nil, err
		}
		out.ToolChoice = tc
	}
	// 处理推理配置（thinking 模式）
	if req.Reasoning != nil && req.Reasoning.Effort != "" {
		effort := mapResponsesEffortToAnthropic(req.Reasoning.Effort)
		out.OutputConfig = &AnthropicOutputConfig{Effort: effort}
		if effort != "low" {
			// 非 low 力度启用 thinking 模式
			out.Thinking = &AnthropicThinking{
				Type:         "enabled",
				BudgetTokens: defaultThinkingBudget(effort),
			}
		}
	}
	return out, nil
}

// defaultThinkingBudget 根据推理力度返回默认思考 Token 预算
func defaultThinkingBudget(effort string) int {
	switch effort {
	case "low":
		return 1024
	case "medium":
		return 4096
	case "high":
		return 10240
	case "max":
		return 32768
	default:
		return 10240
	}
}

// mapResponsesEffortToAnthropic 将 Responses API 推理力度映射到 Anthropic 格式
// xhigh → max
func mapResponsesEffortToAnthropic(effort string) string {
	if effort == "xhigh" {
		return "max"
	}
	return effort
}

// convertResponsesInputToAnthropic 将 Responses API 输入转换为 Anthropic 消息格式
// 返回系统提示词和消息列表
func convertResponsesInputToAnthropic(inputRaw json.RawMessage) (json.RawMessage, []AnthropicMessage, error) {
	// 尝试解析为纯文本输入
	var inputStr string
	if err := json.Unmarshal(inputRaw, &inputStr); err == nil {
		content, _ := json.Marshal(inputStr)
		return nil, []AnthropicMessage{{Role: "user", Content: content}}, nil
	}
	// 解析为输入项数组
	var items []openAIResponsesInputItem
	if err := json.Unmarshal(inputRaw, &items); err != nil {
		return nil, nil, err
	}
	var system json.RawMessage
	var messages []AnthropicMessage
	for _, item := range items {
		switch {
		case item.Role == "system":
			// 系统消息提取为 system 字段
			text := extractTextFromContent(item.Content)
			if text != "" {
				system, _ = json.Marshal(text)
			}
		case item.Type == "function_call":
			// 函数调用 → assistant 消息中的 tool_use 块
			input := json.RawMessage("{}")
			if item.Arguments != "" {
				input = json.RawMessage(item.Arguments)
			}
			block := AnthropicContentBlock{
				Type:  "tool_use",
				ID:    fromResponsesCallIDToAnthropic(item.CallID),
				Name:  item.Name,
				Input: input,
			}
			blockJSON, _ := json.Marshal([]AnthropicContentBlock{block})
			messages = append(messages, AnthropicMessage{Role: "assistant", Content: blockJSON})
		case item.Type == "function_call_output":
			// 函数结果 → user 消息中的 tool_result 块
			outputContent := item.Output
			if outputContent == "" {
				outputContent = "(empty)"
			}
			contentJSON, _ := json.Marshal(outputContent)
			block := AnthropicContentBlock{
				Type:      "tool_result",
				ToolUseID: fromResponsesCallIDToAnthropic(item.CallID),
				Content:   contentJSON,
			}
			blockJSON, _ := json.Marshal([]AnthropicContentBlock{block})
			messages = append(messages, AnthropicMessage{Role: "user", Content: blockJSON})
		case item.Role == "user":
			// 用户消息
			content, err := convertResponsesUserToAnthropicContent(item.Content)
			if err != nil {
				return nil, nil, err
			}
			messages = append(messages, AnthropicMessage{Role: "user", Content: content})
		case item.Role == "assistant":
			// 助手消息
			content, err := convertResponsesAssistantToAnthropicContent(item.Content)
			if err != nil {
				return nil, nil, err
			}
			messages = append(messages, AnthropicMessage{Role: "assistant", Content: content})
		default:
			// 其他类型作为用户消息
			if item.Content != nil {
				messages = append(messages, AnthropicMessage{Role: "user", Content: item.Content})
			}
		}
	}
	// 合并连续的同角色消息
	return system, mergeConsecutiveMessages(messages), nil
}

// extractTextFromContent 从内容中提取纯文本
// 支持纯文本字符串和内容部分数组
func extractTextFromContent(raw json.RawMessage) string {
	if len(raw) == 0 {
		return ""
	}
	// 尝试解析为纯文本
	var s string
	if err := json.Unmarshal(raw, &s); err == nil {
		return s
	}
	// 尝试解析为内容部分数组
	var parts []openAIResponsesContentPart
	if err := json.Unmarshal(raw, &parts); err == nil {
		var texts []string
		for _, p := range parts {
			if (p.Type == "input_text" || p.Type == "output_text" || p.Type == "text") && p.Text != "" {
				texts = append(texts, p.Text)
			}
		}
		return strings.Join(texts, "\n\n")
	}
	return ""
}

// convertResponsesUserToAnthropicContent 将 Responses API 用户内容转换为 Anthropic 格式
// 支持 input_text 和 input_image 类型
func convertResponsesUserToAnthropicContent(raw json.RawMessage) (json.RawMessage, error) {
	if len(raw) == 0 {
		return json.Marshal("")
	}
	// 尝试解析为纯文本
	var s string
	if err := json.Unmarshal(raw, &s); err == nil {
		return json.Marshal(s)
	}
	// 解析为内容部分数组
	var parts []openAIResponsesContentPart
	if err := json.Unmarshal(raw, &parts); err != nil {
		return raw, nil // 无法解析时原样返回
	}
	var blocks []AnthropicContentBlock
	for _, p := range parts {
		switch p.Type {
		case "input_text", "text":
			if p.Text != "" {
				blocks = append(blocks, AnthropicContentBlock{Type: "text", Text: p.Text})
			}
		case "input_image":
			// 将 data URI 转换为 Anthropic 图片源
			src := dataURIToAnthropicImageSource(p.ImageURL)
			if src != nil {
				blocks = append(blocks, AnthropicContentBlock{Type: "image", Source: src})
			}
		}
	}
	if len(blocks) == 0 {
		return json.Marshal("")
	}
	return json.Marshal(blocks)
}

// convertResponsesAssistantToAnthropicContent 将 Responses API 助手内容转换为 Anthropic 格式
func convertResponsesAssistantToAnthropicContent(raw json.RawMessage) (json.RawMessage, error) {
	if len(raw) == 0 {
		return json.Marshal([]AnthropicContentBlock{{Type: "text", Text: ""}})
	}
	// 尝试解析为纯文本
	var s string
	if err := json.Unmarshal(raw, &s); err == nil {
		return json.Marshal([]AnthropicContentBlock{{Type: "text", Text: s}})
	}
	// 解析为内容部分数组
	var parts []openAIResponsesContentPart
	if err := json.Unmarshal(raw, &parts); err != nil {
		return raw, nil
	}
	var blocks []AnthropicContentBlock
	for _, p := range parts {
		if (p.Type == "output_text" || p.Type == "text") && p.Text != "" {
			blocks = append(blocks, AnthropicContentBlock{Type: "text", Text: p.Text})
		}
	}
	if len(blocks) == 0 {
		blocks = append(blocks, AnthropicContentBlock{Type: "text", Text: ""})
	}
	return json.Marshal(blocks)
}

// mergeConsecutiveMessages 合并连续的同角色消息
// Anthropic API 不允许连续的同角色消息，需要合并
// mergeConsecutiveMessages 合并连续的同角色消息
// Anthropic API 不允许连续的同角色消息，需要合并
func mergeConsecutiveMessages(messages []AnthropicMessage) []AnthropicMessage {
	if len(messages) <= 1 {
		return messages
	}
	var merged []AnthropicMessage
	for _, msg := range messages {
		if len(merged) == 0 || merged[len(merged)-1].Role != msg.Role {
			merged = append(merged, msg)
			continue
		}
		// 合并同角色消息的内容块
		last := &merged[len(merged)-1]
		lastBlocks := parseContentBlocks(last.Content)
		newBlocks := parseContentBlocks(msg.Content)
		combined := append(lastBlocks, newBlocks...)
		last.Content, _ = json.Marshal(combined)
	}
	return merged
}

// parseContentBlocks 解析内容为 AnthropicContentBlock 数组
// 支持内容块数组和纯文本两种格式
func parseContentBlocks(raw json.RawMessage) []AnthropicContentBlock {
	var blocks []AnthropicContentBlock
	if err := json.Unmarshal(raw, &blocks); err == nil {
		return blocks
	}
	var s string
	if err := json.Unmarshal(raw, &s); err == nil {
		return []AnthropicContentBlock{{Type: "text", Text: s}}
	}
	return nil
}

// convertResponsesToAnthropicTools 将 Responses API 工具转换为 Anthropic 工具格式
func convertResponsesToAnthropicTools(tools []openAIResponsesTool) []AnthropicTool {
	var out []AnthropicTool
	for _, t := range tools {
		switch t.Type {
		case "web_search", "google_search", "web_search_20250305":
			// 网络搜索工具使用 Anthropic 内置类型
			out = append(out, AnthropicTool{Type: "web_search_20250305", Name: "web_search"})
		case "function":
			out = append(out, AnthropicTool{
				Name:        t.Name,
				Description: t.Description,
				InputSchema: normalizeAnthropicInputSchema(t.Parameters),
			})
		default:
			out = append(out, AnthropicTool{
				Type:        t.Type,
				Name:        t.Name,
				Description: t.Description,
				InputSchema: t.Parameters,
			})
		}
	}
	return out
}

// normalizeAnthropicInputSchema 归一化 Anthropic 工具输入 Schema
// 空值时返回默认的空对象 Schema
func normalizeAnthropicInputSchema(schema json.RawMessage) json.RawMessage {
	if len(schema) == 0 || string(schema) == "null" {
		return json.RawMessage(`{"type":"object","properties":{}}`)
	}
	return schema
}

// convertResponsesToAnthropicToolChoice 将 Responses API 工具选择策略转换为 Anthropic 格式
// auto → auto, required → any, none → none, function → tool
func convertResponsesToAnthropicToolChoice(raw json.RawMessage) (json.RawMessage, error) {
	// 尝试解析为字符串
	var s string
	if err := json.Unmarshal(raw, &s); err == nil {
		switch s {
		case "auto":
			return json.Marshal(map[string]string{"type": "auto"})
		case "required":
			return json.Marshal(map[string]string{"type": "any"})
		case "none":
			return json.Marshal(map[string]string{"type": "none"})
		default:
			return raw, nil
		}
	}
	// 解析为对象（指定具体函数）
	var tc struct {
		Type     string `json:"type"`
		Name     string `json:"name"`
		Function struct {
			Name string `json:"name"`
		} `json:"function"`
	}
	if err := json.Unmarshal(raw, &tc); err == nil && tc.Type == "function" {
		name := strings.TrimSpace(tc.Name)
		if name == "" {
			name = strings.TrimSpace(tc.Function.Name)
		}
		if name == "" {
			return raw, nil
		}
		return json.Marshal(map[string]string{"type": "tool", "name": name})
	}
	return raw, nil
}

// fromResponsesCallIDToAnthropic 将 Responses API 调用 ID 转换为 Anthropic 格式
// 去掉 fc_ 前缀，确保以 toolu_ 或 call_ 开头
func fromResponsesCallIDToAnthropic(id string) string {
	if after, ok := strings.CutPrefix(id, "fc_"); ok {
		if strings.HasPrefix(after, "toolu_") || strings.HasPrefix(after, "call_") {
			return after
		}
	}
	// 如果没有 Anthropic 前缀，添加 toolu_ 前缀
	if !strings.HasPrefix(id, "toolu_") && !strings.HasPrefix(id, "call_") {
		return "toolu_" + id
	}
	return id
}

// dataURIToAnthropicImageSource 将 data URI 转换为 Anthropic 图片源
// dataURIToAnthropicImageSource 将 data URI 转换为 Anthropic 图片源
// 解析 data:<mediaType>;base64,<data> 格式
func dataURIToAnthropicImageSource(dataURI string) *AnthropicImageSource {
	if !strings.HasPrefix(dataURI, "data:") {
		return nil // 非 data URI 不支持
	}
	rest := strings.TrimPrefix(dataURI, "data:")
	semicolonIdx := strings.Index(rest, ";")
	if semicolonIdx < 0 {
		return nil
	}
	mediaType := rest[:semicolonIdx] // 提取 MIME 类型
	rest = rest[semicolonIdx+1:]
	if !strings.HasPrefix(rest, "base64,") {
		return nil
	}
	data := strings.TrimPrefix(rest, "base64,")
	return &AnthropicImageSource{Type: "base64", MediaType: mediaType, Data: data}
}

// AdaptGatewayResponse 适配 Claude 响应为 OpenAI 格式
// Chat Completions 路径：Claude → Responses → Chat Completions
// Responses 路径：Claude → Responses
func (p *Provider) AdaptGatewayResponse(req provider.GatewayRequest, resp *http.Response, body []byte) (*http.Response, []byte, error) {
	switch normalizeOpenAIPath(req.PublicPath) {
	case "/v1/chat/completions":
		return p.adaptClaudeToOpenAIChatResponse(req, resp, body)
	case "/v1/responses", "/backend-api/codex/responses":
		return p.adaptClaudeToOpenAIResponsesResponse(resp, body)
	default:
		return resp, body, nil // 非转换路径原样返回
	}
}

// AdaptGatewayStream 适配 Claude 流式响应为 OpenAI 格式
func (p *Provider) AdaptGatewayStream(req provider.GatewayRequest, resp *http.Response) (*http.Response, error) {
	switch normalizeOpenAIPath(req.PublicPath) {
	case "/v1/chat/completions":
		return p.wrapClaudeToOpenAIChatStream(req, resp), nil
	case "/v1/responses", "/backend-api/codex/responses":
		return p.wrapClaudeToOpenAIResponsesStream(resp), nil
	default:
		return resp, nil
	}
}

// adaptClaudeToOpenAIResponsesResponse 将 Claude 响应转换为 OpenAI Responses API 格式
func (p *Provider) adaptClaudeToOpenAIResponsesResponse(resp *http.Response, body []byte) (*http.Response, []byte, error) {
	if resp.StatusCode >= 400 {
		return resp, body, nil // 错误响应原样返回
	}
	var anth AnthropicResponse
	if err := json.Unmarshal(body, &anth); err != nil {
		return resp, body, err
	}
	out := anthropicToResponsesResponse(&anth)
	converted, err := json.Marshal(out)
	if err != nil {
		return resp, body, err
	}
	resp.Header = cloneHTTPHeader(resp.Header)
	resp.Header.Set("Content-Type", "application/json")
	resp.ContentLength = int64(len(converted))
	return resp, converted, nil
}

// adaptClaudeToOpenAIChatResponse 将 Claude 响应转换为 OpenAI Chat Completions 格式
// 先转换为 Responses 格式，再转换为 Chat Completions 格式
func (p *Provider) adaptClaudeToOpenAIChatResponse(req provider.GatewayRequest, resp *http.Response, body []byte) (*http.Response, []byte, error) {
	// 第一步：Claude → Responses
	resp, converted, err := p.adaptClaudeToOpenAIResponsesResponse(resp, body)
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

// wrapClaudeToOpenAIResponsesStream 包装 Claude 流式响应为 OpenAI Responses API 流式格式
func (p *Provider) wrapClaudeToOpenAIResponsesStream(resp *http.Response) *http.Response {
	if resp.StatusCode >= 400 || !isSSEHeader(resp.Header) {
		return resp // 错误或非 SSE 响应原样返回
	}
	resp.Header = cloneHTTPHeader(resp.Header)
	resp.Header.Set("Content-Type", "text/event-stream; charset=utf-8")
	resp.Body = newClaudeResponsesStreamAdapter(resp.Body)
	resp.ContentLength = -1 // 流式响应长度未知
	return resp
}

// wrapClaudeToOpenAIChatStream 包装 Claude 流式响应为 OpenAI Chat Completions 流式格式
func (p *Provider) wrapClaudeToOpenAIChatStream(req provider.GatewayRequest, resp *http.Response) *http.Response {
	if resp.StatusCode >= 400 || !isSSEHeader(resp.Header) {
		return resp
	}
	resp.Header = cloneHTTPHeader(resp.Header)
	resp.Header.Set("Content-Type", "text/event-stream; charset=utf-8")
	resp.Body = newClaudeChatStreamAdapter(resp.Body, req.Model, req.IncludeUsage)
	resp.ContentLength = -1
	return resp
}

// anthropicToResponsesResponse 将 Anthropic 响应转换为 OpenAI Responses API 响应
// anthropicToResponsesResponse 将 Anthropic 响应转换为 OpenAI Responses API 响应
func anthropicToResponsesResponse(resp *AnthropicResponse) *openAIResponsesResponse {
	id := strings.TrimSpace(resp.ID)
	if id == "" {
		id = generateResponsesID() // 生成随机 ID
	}
	out := &openAIResponsesResponse{
		ID:     id,
		Object: "response",
		Model:  resp.Model,
		Status: anthropicStopReasonToResponsesStatus(resp.StopReason),
	}
	var outputs []openAIResponsesOutput
	var msgParts []openAIResponsesContentPart
	// 遍历内容块，分类转换为输出项
	for _, block := range resp.Content {
		switch block.Type {
		case "thinking":
			// 思考内容 → reasoning 输出项
			if block.Thinking != "" {
				outputs = append(outputs, openAIResponsesOutput{
					Type: "reasoning",
					ID:   generateItemID(),
					Summary: []openAIResponsesSummary{{
						Type: "summary_text",
						Text: block.Thinking,
					}},
				})
			}
		case "text":
			// 文本内容收集到消息部分
			if block.Text != "" {
				msgParts = append(msgParts, openAIResponsesContentPart{Type: "output_text", Text: block.Text})
			}
		case "tool_use":
			// 工具调用 → function_call 输出项
			args := "{}"
			if len(block.Input) > 0 {
				args = string(block.Input)
			}
			outputs = append(outputs, openAIResponsesOutput{
				Type:      "function_call",
				ID:        generateItemID(),
				CallID:    toResponsesCallID(block.ID),
				Name:      block.Name,
				Arguments: args,
				Status:    "completed",
			})
		}
	}
	// 将文本内容包装为 message 输出项
	if len(msgParts) > 0 {
		outputs = append(outputs, openAIResponsesOutput{
			Type:    "message",
			ID:      generateItemID(),
			Role:    "assistant",
			Content: msgParts,
			Status:  "completed",
		})
	}
	// 确保至少有一个输出项
	if len(outputs) == 0 {
		outputs = append(outputs, openAIResponsesOutput{
			Type:    "message",
			ID:      generateItemID(),
			Role:    "assistant",
			Content: []openAIResponsesContentPart{{Type: "output_text", Text: ""}},
			Status:  "completed",
		})
	}
	out.Output = outputs
	// 设置用量信息
	out.Usage = &openAIResponsesUsage{
		InputTokens:  resp.Usage.InputTokens,
		OutputTokens: resp.Usage.OutputTokens,
		TotalTokens:  resp.Usage.InputTokens + resp.Usage.OutputTokens,
	}
	if resp.Usage.CacheReadInputTokens > 0 {
		out.Usage.InputTokensDetails = &openAIResponsesInputTokenDetails{CachedTokens: resp.Usage.CacheReadInputTokens}
	}
	// 设置未完成详情
	if out.Status == "incomplete" {
		out.IncompleteDetails = &openAIResponsesIncompleteDetails{Reason: "max_output_tokens"}
	}
	return out
}

// anthropicStopReasonToResponsesStatus 将 Anthropic 停止原因映射为 Responses API 状态
// max_tokens → incomplete, 其他 → completed
func anthropicStopReasonToResponsesStatus(stopReason string) string {
	if stopReason == "max_tokens" {
		return "incomplete"
	}
	return "completed"
}

// responsesToChatCompletions 将 Responses API 响应转换为 Chat Completions 格式
func responsesToChatCompletions(resp *openAIResponsesResponse, model string) *openAIChatCompletionsResponse {
	id := strings.TrimSpace(resp.ID)
	if id == "" {
		id = generateChatCmplID()
	}
	out := &openAIChatCompletionsResponse{
		ID:      id,
		Object:  "chat.completion",
		Created: time.Now().Unix(),
		Model:   firstNonEmpty(strings.TrimSpace(model), strings.TrimSpace(resp.Model)),
	}
	var contentText string
	var reasoningText string
	var toolCalls []openAIChatToolCall
	// 遍历输出项，提取文本、推理和工具调用
	for _, item := range resp.Output {
		switch item.Type {
		case "message":
			for _, part := range item.Content {
				if part.Type == "output_text" && part.Text != "" {
					contentText += part.Text
				}
			}
		case "function_call":
			toolCalls = append(toolCalls, openAIChatToolCall{
				ID:   item.CallID,
				Type: "function",
				Function: openAIChatFunctionCall{
					Name:      item.Name,
					Arguments: item.Arguments,
				},
			})
		case "reasoning":
			// 提取推理摘要文本
			for _, s := range item.Summary {
				if s.Type == "summary_text" && s.Text != "" {
					reasoningText += s.Text
				}
			}
		}
	}
	// 构建助手消息
	msg := openAIChatMessage{Role: "assistant"}
	if len(toolCalls) > 0 {
		msg.ToolCalls = toolCalls
	}
	if contentText != "" {
		raw, _ := json.Marshal(contentText)
		msg.Content = raw
	}
	if reasoningText != "" {
		msg.ReasoningContent = reasoningText
	}
	// 构建选择项
	out.Choices = []openAIChatChoice{{
		Index:        0,
		Message:      msg,
		FinishReason: responsesStatusToChatFinishReason(resp.Status, resp.IncompleteDetails, len(toolCalls) > 0),
	}}
	// 设置用量
	if resp.Usage != nil {
		usage := &openAIChatUsage{
			PromptTokens:     resp.Usage.InputTokens,
			CompletionTokens: resp.Usage.OutputTokens,
			TotalTokens:      resp.Usage.InputTokens + resp.Usage.OutputTokens,
		}
		if resp.Usage.InputTokensDetails != nil && resp.Usage.InputTokensDetails.CachedTokens > 0 {
			usage.PromptTokensDetails = &openAIChatTokenDetail{CachedTokens: resp.Usage.InputTokensDetails.CachedTokens}
		}
		out.Usage = usage
	}
	return out
}

// responsesStatusToChatFinishReason 将 Responses API 状态映射为 Chat Completions 完成原因
// incomplete + max_output_tokens → length, 有工具调用 → tool_calls, 其他 → stop
func responsesStatusToChatFinishReason(status string, details *openAIResponsesIncompleteDetails, hasToolCalls bool) string {
	switch status {
	case "incomplete":
		if details != nil && details.Reason == "max_output_tokens" {
			return "length"
		}
		return "stop"
	default:
		if hasToolCalls {
			return "tool_calls"
		}
		return "stop"
	}
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

// anthropicResponsesStreamState Anthropic → Responses 流式转换状态
type anthropicResponsesStreamState struct {
	ResponseID           string // 响应 ID
	Model                string // 模型名
	SequenceNumber       int    // 序列号计数器
	CreatedSent          bool   // 是否已发送 created 事件
	CompletedSent        bool   // 是否已发送 completed 事件
	OutputIndex          int    // 当前输出项索引
	CurrentItemID        string // 当前输出项 ID
	CurrentItemType      string // 当前输出项类型
	CurrentCallID        string // 当前函数调用 ID
	CurrentName          string // 当前函数名
	ContentIndex         int    // 当前内容索引
	InputTokens          int    // 输入 Token 数
	OutputTokens         int    // 输出 Token 数
	CacheReadInputTokens int    // 缓存读取 Token 数
}

// newClaudeResponsesStreamAdapter 创建 Claude → Responses 流式适配器
// newClaudeResponsesStreamAdapter 创建 Claude → Responses 流式适配器
// 将 Anthropic SSE 事件转换为 OpenAI Responses API SSE 事件
func newClaudeResponsesStreamAdapter(src io.ReadCloser) io.ReadCloser {
	return newSSETransformReadCloser(src, func(data string, st any) ([]string, error) {
		state := st.(*anthropicResponsesStreamState)
		if strings.TrimSpace(data) == "" {
			return nil, nil
		}
		var evt AnthropicStreamEvent
		if err := json.Unmarshal([]byte(data), &evt); err != nil {
			return nil, err
		}
		// 将 Anthropic 事件转换为 Responses 事件
		events := anthropicEventToResponsesEvents(&evt, state)
		if len(events) == 0 {
			return nil, nil
		}
		out := make([]string, 0, len(events))
		for _, item := range events {
			payload, err := json.Marshal(item)
			if err != nil {
				return nil, err
			}
			out = append(out, fmt.Sprintf("event: %s\ndata: %s\n\n", item.Type, payload))
		}
		return out, nil
	}, &anthropicResponsesStreamState{})
}

// responsesChatStreamState Responses → Chat Completions 流式转换状态
type responsesChatStreamState struct {
	ID                     string           // 响应 ID
	Model                  string           // 模型名
	Created                int64            // 创建时间戳
	SentRole               bool             // 是否已发送角色
	SawToolCall            bool             // 是否看到工具调用
	Finalized              bool             // 是否已结束
	NextToolCallIndex      int              // 下一个工具调用索引
	OutputIndexToToolIndex map[int]int      // 输出项索引到工具调用索引的映射
	IncludeUsage           bool             // 是否包含用量
	Usage                  *openAIChatUsage // 累积的用量信息
}

// newClaudeChatStreamAdapter 创建 Claude → Chat Completions 流式适配器
// 两步转换：Anthropic → Responses → Chat Completions
func newClaudeChatStreamAdapter(src io.ReadCloser, model string, includeUsage bool) io.ReadCloser {
	respState := &anthropicResponsesStreamState{}
	chatState := &responsesChatStreamState{
		ID:                     generateChatCmplID(),
		Model:                  model,
		Created:                time.Now().Unix(),
		OutputIndexToToolIndex: make(map[int]int),
		IncludeUsage:           includeUsage,
	}
	return newSSETransformReadCloser(src, func(data string, st any) ([]string, error) {
		if strings.TrimSpace(data) == "" {
			return nil, nil
		}
		var evt AnthropicStreamEvent
		if err := json.Unmarshal([]byte(data), &evt); err != nil {
			return nil, err
		}
		resEvents := anthropicEventToResponsesEvents(&evt, respState)
		var out []string
		for _, resEvt := range resEvents {
			chunks := responsesEventToChatChunks(&resEvt, chatState)
			for _, chunk := range chunks {
				payload, err := json.Marshal(chunk)
				if err != nil {
					return nil, err
				}
				out = append(out, fmt.Sprintf("data: %s\n\n", payload))
			}
			if isTerminalResponsesEvent(resEvt.Type) {
				out = append(out, "data: [DONE]\n\n")
			}
		}
		return out, nil
	}, chatState)
}

// sseTransformReadCloser SSE 流式转换器
// 读取源 SSE 流，通过 transform 函数转换每个 data 行，输出转换后的 SSE 流
type sseTransformReadCloser struct {
	src        io.ReadCloser                       // 源响应体
	scanner    *bufio.Scanner                      // 行扫描器
	transform  func(string, any) ([]string, error) // 转换函数
	state      any                                 // 状态对象
	pending    bytes.Buffer                        // 待输出的缓冲区
	eofHandled bool                                // 是否已处理 EOF
	closeErr   error                               // 关闭错误
}

// newSSETransformReadCloser 创建 SSE 流式转换器
func newSSETransformReadCloser(src io.ReadCloser, transform func(string, any) ([]string, error), state any) io.ReadCloser {
	scanner := bufio.NewScanner(src)
	scanner.Buffer(make([]byte, 0, 4096), 1024*1024) // 最大 1MB 行
	return &sseTransformReadCloser{src: src, scanner: scanner, transform: transform, state: state}
}

// Read 实现 io.Reader 接口
// 从源 SSE 流读取数据行，转换后写入输出缓冲区
func (r *sseTransformReadCloser) Read(p []byte) (int, error) {
	for r.pending.Len() == 0 {
		if !r.scanner.Scan() {
			if err := r.scanner.Err(); err != nil {
				return 0, err
			}
			// 源流结束，发送终止事件
			if !r.eofHandled {
				r.eofHandled = true
				r.appendEOF()
			}
			if r.pending.Len() == 0 {
				return 0, io.EOF
			}
			break
		}
		line := r.scanner.Text()
		if !strings.HasPrefix(line, "data:") {
			continue
		}
		data := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
		events, err := r.transform(data, r.state)
		if err != nil {
			return 0, err
		}
		for _, item := range events {
			_, _ = r.pending.WriteString(item)
		}
	}
	return r.pending.Read(p)
}

// appendEOF 在源流结束时发送终止事件
func (r *sseTransformReadCloser) appendEOF() {
	switch state := r.state.(type) {
	case *anthropicResponsesStreamState:
		// Responses 流：发送 completed 事件
		if state.CreatedSent && !state.CompletedSent {
			events := finalizeAnthropicResponsesStream(state)
			for _, item := range events {
				payload, err := json.Marshal(item)
				if err != nil {
					continue
				}
				_, _ = r.pending.WriteString(fmt.Sprintf("event: %s\ndata: %s\n\n", item.Type, payload))
			}
		}
	case *responsesChatStreamState:
		// Chat 流：发送 finish chunk 和 [DONE]
		if !state.Finalized {
			for _, chunk := range finalizeResponsesChatStream(state) {
				payload, err := json.Marshal(chunk)
				if err != nil {
					continue
				}
				_, _ = r.pending.WriteString(fmt.Sprintf("data: %s\n\n", payload))
			}
			_, _ = r.pending.WriteString("data: [DONE]\n\n")
		}
	}
}

// Close 关闭源流
func (r *sseTransformReadCloser) Close() error {
	if r.src != nil {
		r.closeErr = r.src.Close()
	}
	return r.closeErr
}

// anthropicEventToResponsesEvents 将 Anthropic 流式事件转换为 Responses API 事件
// anthropicEventToResponsesEvents 将 Anthropic 流式事件转换为 Responses API 事件
// 根据事件类型分发到对应的处理函数
func anthropicEventToResponsesEvents(evt *AnthropicStreamEvent, state *anthropicResponsesStreamState) []openAIResponsesStreamEvent {
	switch evt.Type {
	case "message_start":
		return anthToResHandleMessageStart(evt, state)
	case "content_block_start":
		return anthToResHandleContentBlockStart(evt, state)
	case "content_block_delta":
		return anthToResHandleContentBlockDelta(evt, state)
	case "content_block_stop":
		return anthToResHandleContentBlockStop(state)
	case "message_delta":
		return anthToResHandleMessageDelta(evt, state)
	case "message_stop":
		return anthToResHandleMessageStop(state)
	default:
		return nil
	}
}

// anthToResHandleMessageStart 处理 message_start 事件
// 发送 response.created 事件
func anthToResHandleMessageStart(evt *AnthropicStreamEvent, state *anthropicResponsesStreamState) []openAIResponsesStreamEvent {
	if evt.Message != nil {
		state.ResponseID = evt.Message.ID
		if state.Model == "" {
			state.Model = evt.Message.Model
		}
		// 记录初始用量
		state.InputTokens = evt.Message.Usage.InputTokens
		state.OutputTokens = evt.Message.Usage.OutputTokens
		state.CacheReadInputTokens = evt.Message.Usage.CacheReadInputTokens
	}
	if state.CreatedSent {
		return nil // 避免重复发送
	}
	state.CreatedSent = true
	return []openAIResponsesStreamEvent{makeResponsesCreatedEvent(state)}
}

// anthToResHandleContentBlockStart 处理 content_block_start 事件
// 根据内容块类型发送对应的 output_item.added 事件
// anthToResHandleContentBlockStart 处理 content_block_start 事件
// 根据内容块类型发送对应的 output_item.added 事件
func anthToResHandleContentBlockStart(evt *AnthropicStreamEvent, state *anthropicResponsesStreamState) []openAIResponsesStreamEvent {
	if evt.ContentBlock == nil {
		return nil
	}
	var events []openAIResponsesStreamEvent
	switch evt.ContentBlock.Type {
	case "thinking":
		// 思考块 → reasoning 输出项
		state.CurrentItemID = generateItemID()
		state.CurrentItemType = "reasoning"
		events = append(events, makeResponsesEvent(state, "response.output_item.added", &openAIResponsesStreamEvent{
			OutputIndex: state.OutputIndex,
			Item: &openAIResponsesOutput{
				Type: "reasoning",
				ID:   state.CurrentItemID,
			},
		}))
	case "text":
		// 文本块 → message 输出项（如果还没有创建）
		if state.CurrentItemType != "message" {
			state.CurrentItemID = generateItemID()
			state.CurrentItemType = "message"
			state.ContentIndex = 0
			events = append(events, makeResponsesEvent(state, "response.output_item.added", &openAIResponsesStreamEvent{
				OutputIndex: state.OutputIndex,
				Item: &openAIResponsesOutput{
					Type:   "message",
					ID:     state.CurrentItemID,
					Role:   "assistant",
					Status: "in_progress",
				},
			}))
		}
	case "tool_use":
		// 工具调用块 → function_call 输出项
		events = append(events, closeCurrentResponsesItem(state)...)
		state.CurrentItemID = generateItemID()
		state.CurrentItemType = "function_call"
		state.CurrentCallID = toResponsesCallID(evt.ContentBlock.ID)
		state.CurrentName = evt.ContentBlock.Name
		events = append(events, makeResponsesEvent(state, "response.output_item.added", &openAIResponsesStreamEvent{
			OutputIndex: state.OutputIndex,
			Item: &openAIResponsesOutput{
				Type:   "function_call",
				ID:     state.CurrentItemID,
				CallID: state.CurrentCallID,
				Name:   state.CurrentName,
				Status: "in_progress",
			},
		}))
	}
	return events
}

// anthToResHandleContentBlockDelta 处理 content_block_delta 事件
// 根据增量类型发送对应的 delta 事件
// anthToResHandleContentBlockDelta 处理 content_block_delta 事件
// 根据增量类型发送对应的 delta 事件
func anthToResHandleContentBlockDelta(evt *AnthropicStreamEvent, state *anthropicResponsesStreamState) []openAIResponsesStreamEvent {
	if evt.Delta == nil {
		return nil
	}
	switch evt.Delta.Type {
	case "text_delta":
		// 文本增量 → output_text.delta
		if evt.Delta.Text == "" {
			return nil
		}
		return []openAIResponsesStreamEvent{makeResponsesEvent(state, "response.output_text.delta", &openAIResponsesStreamEvent{
			OutputIndex:  state.OutputIndex,
			ContentIndex: state.ContentIndex,
			Delta:        evt.Delta.Text,
			ItemID:       state.CurrentItemID,
		})}
	case "thinking_delta":
		// 思考增量 → reasoning_summary_text.delta
		if evt.Delta.Thinking == "" {
			return nil
		}
		return []openAIResponsesStreamEvent{makeResponsesEvent(state, "response.reasoning_summary_text.delta", &openAIResponsesStreamEvent{
			OutputIndex:  state.OutputIndex,
			SummaryIndex: 0,
			Delta:        evt.Delta.Thinking,
			ItemID:       state.CurrentItemID,
		})}
	case "input_json_delta":
		// 工具输入 JSON 增量 → function_call_arguments.delta
		if evt.Delta.PartialJSON == "" {
			return nil
		}
		return []openAIResponsesStreamEvent{makeResponsesEvent(state, "response.function_call_arguments.delta", &openAIResponsesStreamEvent{
			OutputIndex: state.OutputIndex,
			Delta:       evt.Delta.PartialJSON,
			ItemID:      state.CurrentItemID,
			CallID:      state.CurrentCallID,
			Name:        state.CurrentName,
		})}
	default:
		return nil
	}
}

// anthToResHandleContentBlockStop 处理 content_block_stop 事件
// 根据当前项类型发送对应的 done 事件
// anthToResHandleContentBlockStop 处理 content_block_stop 事件
// 根据当前项类型发送对应的 done 事件
func anthToResHandleContentBlockStop(state *anthropicResponsesStreamState) []openAIResponsesStreamEvent {
	switch state.CurrentItemType {
	case "reasoning":
		// 推理完成：发送 reasoning_summary_text.done 和 output_item.done
		events := []openAIResponsesStreamEvent{
			makeResponsesEvent(state, "response.reasoning_summary_text.done", &openAIResponsesStreamEvent{
				OutputIndex:  state.OutputIndex,
				SummaryIndex: 0,
				ItemID:       state.CurrentItemID,
			}),
		}
		return append(events, closeCurrentResponsesItem(state)...)
	case "function_call":
		// 函数调用完成：发送 function_call_arguments.done 和 output_item.done
		events := []openAIResponsesStreamEvent{
			makeResponsesEvent(state, "response.function_call_arguments.done", &openAIResponsesStreamEvent{
				OutputIndex: state.OutputIndex,
				ItemID:      state.CurrentItemID,
				CallID:      state.CurrentCallID,
				Name:        state.CurrentName,
			}),
		}
		return append(events, closeCurrentResponsesItem(state)...)
	case "message":
		return []openAIResponsesStreamEvent{makeResponsesEvent(state, "response.output_text.done", &openAIResponsesStreamEvent{
			OutputIndex:  state.OutputIndex,
			ContentIndex: state.ContentIndex,
			ItemID:       state.CurrentItemID,
		})}
	default:
		return nil
	}
}

// anthToResHandleMessageDelta 处理 message_delta 事件
// 更新输出 Token 数和缓存读取 Token 数
func anthToResHandleMessageDelta(evt *AnthropicStreamEvent, state *anthropicResponsesStreamState) []openAIResponsesStreamEvent {
	if evt.Usage != nil {
		state.OutputTokens = evt.Usage.OutputTokens
		if evt.Usage.CacheReadInputTokens > 0 {
			state.CacheReadInputTokens = evt.Usage.CacheReadInputTokens
		}
	}
	return nil
}

// anthToResHandleMessageStop 处理 message_stop 事件
// 关闭当前项并发送 completed 事件
func anthToResHandleMessageStop(state *anthropicResponsesStreamState) []openAIResponsesStreamEvent {
	if state.CompletedSent {
		return nil
	}
	events := closeCurrentResponsesItem(state)
	events = append(events, makeResponsesCompletedEvent(state, "completed", nil))
	state.CompletedSent = true
	return events
}

// closeCurrentResponsesItem 关闭当前输出项
// 发送 output_item.done 事件并递增输出索引
func closeCurrentResponsesItem(state *anthropicResponsesStreamState) []openAIResponsesStreamEvent {
	if state.CurrentItemType == "" {
		return nil
	}
	itemType := state.CurrentItemType
	itemID := state.CurrentItemID
	// 重置当前项状态
	state.CurrentItemType = ""
	state.CurrentItemID = ""
	state.CurrentCallID = ""
	state.CurrentName = ""
	state.OutputIndex++
	state.ContentIndex = 0
	return []openAIResponsesStreamEvent{makeResponsesEvent(state, "response.output_item.done", &openAIResponsesStreamEvent{
		OutputIndex: state.OutputIndex - 1,
		Item: &openAIResponsesOutput{
			Type:   itemType,
			ID:     itemID,
			Status: "completed",
		},
	})}
}

// makeResponsesCreatedEvent 创建 response.created 事件
func makeResponsesCreatedEvent(state *anthropicResponsesStreamState) openAIResponsesStreamEvent {
	seq := state.SequenceNumber
	state.SequenceNumber++
	return openAIResponsesStreamEvent{
		Type:           "response.created",
		SequenceNumber: seq,
		Response: &openAIResponsesResponse{
			ID:     firstNonEmpty(state.ResponseID, generateResponsesID()),
			Object: "response",
			Model:  state.Model,
			Status: "in_progress",
			Output: []openAIResponsesOutput{},
		},
	}
}

// makeResponsesCompletedEvent 创建 response.completed 事件
// 包含用量信息和未完成详情
func makeResponsesCompletedEvent(state *anthropicResponsesStreamState, status string, incomplete *openAIResponsesIncompleteDetails) openAIResponsesStreamEvent {
	seq := state.SequenceNumber
	state.SequenceNumber++
	usage := &openAIResponsesUsage{
		InputTokens:  state.InputTokens,
		OutputTokens: state.OutputTokens,
		TotalTokens:  state.InputTokens + state.OutputTokens,
	}
	// 设置缓存 Token 详情
	if state.CacheReadInputTokens > 0 {
		usage.InputTokensDetails = &openAIResponsesInputTokenDetails{CachedTokens: state.CacheReadInputTokens}
	}
	return openAIResponsesStreamEvent{
		Type:           "response.completed",
		SequenceNumber: seq,
		Response: &openAIResponsesResponse{
			ID:                firstNonEmpty(state.ResponseID, generateResponsesID()),
			Object:            "response",
			Model:             state.Model,
			Status:            status,
			Output:            []openAIResponsesOutput{},
			Usage:             usage,
			IncompleteDetails: incomplete,
		},
	}
}

// makeResponsesEvent 创建 Responses 流式事件
// 设置事件类型和序列号
func makeResponsesEvent(state *anthropicResponsesStreamState, eventType string, template *openAIResponsesStreamEvent) openAIResponsesStreamEvent {
	seq := state.SequenceNumber
	state.SequenceNumber++
	evt := *template
	evt.Type = eventType
	evt.SequenceNumber = seq
	return evt
}

// finalizeAnthropicResponsesStream 完成 Anthropic → Responses 流
// 关闭当前项并发送 completed 事件
func finalizeAnthropicResponsesStream(state *anthropicResponsesStreamState) []openAIResponsesStreamEvent {
	if !state.CreatedSent || state.CompletedSent {
		return nil
	}
	events := closeCurrentResponsesItem(state)
	events = append(events, makeResponsesCompletedEvent(state, "completed", nil))
	state.CompletedSent = true
	return events
}

// responsesEventToChatChunks 将 Responses 事件转换为 Chat Completions 增量块
// 根据事件类型分发到对应的处理函数
func responsesEventToChatChunks(evt *openAIResponsesStreamEvent, state *responsesChatStreamState) []openAIChatCompletionsChunk {
	switch evt.Type {
	case "response.created":
		return resToChatHandleCreated(evt, state)
	case "response.output_text.delta":
		return resToChatHandleTextDelta(evt, state)
	case "response.output_item.added":
		return resToChatHandleOutputItemAdded(evt, state)
	case "response.function_call_arguments.delta":
		return resToChatHandleFuncArgsDelta(evt, state)
	case "response.reasoning_summary_text.delta":
		return resToChatHandleReasoningDelta(evt, state)
	case "response.completed", "response.done", "response.incomplete", "response.failed":
		return resToChatHandleCompleted(evt, state)
	default:
		return nil
	}
}

// resToChatHandleCreated 处理 response.created 事件
// 记录响应 ID 和模型名，发送角色增量块
func resToChatHandleCreated(evt *openAIResponsesStreamEvent, state *responsesChatStreamState) []openAIChatCompletionsChunk {
	if evt.Response != nil {
		if evt.Response.ID != "" {
			state.ID = evt.Response.ID
		}
		if state.Model == "" && evt.Response.Model != "" {
			state.Model = evt.Response.Model
		}
	}
	if state.SentRole {
		return nil // 避免重复发送角色
	}
	state.SentRole = true
	role := "assistant"
	return []openAIChatCompletionsChunk{makeChatDeltaChunk(state, openAIChatDelta{Role: role})}
}

// resToChatHandleTextDelta 处理 output_text.delta 事件
// 发送文本内容增量块
func resToChatHandleTextDelta(evt *openAIResponsesStreamEvent, state *responsesChatStreamState) []openAIChatCompletionsChunk {
	if evt.Delta == "" {
		return nil
	}
	content := evt.Delta
	return []openAIChatCompletionsChunk{makeChatDeltaChunk(state, openAIChatDelta{Content: &content})}
}

// resToChatHandleOutputItemAdded 处理 output_item.added 事件
// 如果是函数调用项，发送工具调用增量块
func resToChatHandleOutputItemAdded(evt *openAIResponsesStreamEvent, state *responsesChatStreamState) []openAIChatCompletionsChunk {
	if evt.Item == nil || evt.Item.Type != "function_call" {
		return nil
	}
	state.SawToolCall = true
	idx := state.NextToolCallIndex
	state.OutputIndexToToolIndex[evt.OutputIndex] = idx // 记录输出索引到工具调用索引的映射
	state.NextToolCallIndex++
	return []openAIChatCompletionsChunk{makeChatDeltaChunk(state, openAIChatDelta{
		ToolCalls: []openAIChatToolCall{{
			Index: &idx,
			ID:    evt.Item.CallID,
			Type:  "function",
			Function: openAIChatFunctionCall{
				Name: evt.Item.Name,
			},
		}},
	})}
}

// resToChatHandleFuncArgsDelta 处理 function_call_arguments.delta 事件
// 发送函数参数增量块
func resToChatHandleFuncArgsDelta(evt *openAIResponsesStreamEvent, state *responsesChatStreamState) []openAIChatCompletionsChunk {
	if evt.Delta == "" {
		return nil
	}
	idx, ok := state.OutputIndexToToolIndex[evt.OutputIndex]
	if !ok {
		return nil
	}
	return []openAIChatCompletionsChunk{makeChatDeltaChunk(state, openAIChatDelta{
		ToolCalls: []openAIChatToolCall{{
			Index: &idx,
			Function: openAIChatFunctionCall{
				Arguments: evt.Delta,
			},
		}},
	})}
}

// resToChatHandleReasoningDelta 处理 reasoning_summary_text.delta 事件
// 发送推理内容增量块
func resToChatHandleReasoningDelta(evt *openAIResponsesStreamEvent, state *responsesChatStreamState) []openAIChatCompletionsChunk {
	if evt.Delta == "" {
		return nil
	}
	reasoning := evt.Delta
	return []openAIChatCompletionsChunk{makeChatDeltaChunk(state, openAIChatDelta{ReasoningContent: &reasoning})}
}

// resToChatHandleCompleted 处理 response.completed/done/incomplete/failed 事件
// 发送完成原因增量块和可选的用量信息块
func resToChatHandleCompleted(evt *openAIResponsesStreamEvent, state *responsesChatStreamState) []openAIChatCompletionsChunk {
	state.Finalized = true
	finishReason := "stop"
	if evt.Response != nil {
		// 提取用量信息
		if evt.Response.Usage != nil {
			u := evt.Response.Usage
			state.Usage = &openAIChatUsage{
				PromptTokens:     u.InputTokens,
				CompletionTokens: u.OutputTokens,
				TotalTokens:      u.InputTokens + u.OutputTokens,
			}
			if u.InputTokensDetails != nil && u.InputTokensDetails.CachedTokens > 0 {
				state.Usage.PromptTokensDetails = &openAIChatTokenDetail{CachedTokens: u.InputTokensDetails.CachedTokens}
			}
		}
		// 确定完成原因
		if evt.Response.Status == "incomplete" && evt.Response.IncompleteDetails != nil && evt.Response.IncompleteDetails.Reason == "max_output_tokens" {
			finishReason = "length" // 输出截断
		} else if state.SawToolCall {
			finishReason = "tool_calls" // 工具调用
		}
	} else if state.SawToolCall {
		finishReason = "tool_calls"
	}
	chunks := []openAIChatCompletionsChunk{makeChatFinishChunk(state, finishReason)}
	// 如果需要包含用量信息，追加用量块
	if state.IncludeUsage && state.Usage != nil {
		chunks = append(chunks, openAIChatCompletionsChunk{
			ID:      state.ID,
			Object:  "chat.completion.chunk",
			Created: state.Created,
			Model:   state.Model,
			Choices: []openAIChatChunkChoice{},
			Usage:   state.Usage,
		})
	}
	return chunks
}

// makeChatDeltaChunk 创建 Chat Completions 增量块
func makeChatDeltaChunk(state *responsesChatStreamState, delta openAIChatDelta) openAIChatCompletionsChunk {
	return openAIChatCompletionsChunk{
		ID:      state.ID,
		Object:  "chat.completion.chunk",
		Created: state.Created,
		Model:   state.Model,
		Choices: []openAIChatChunkChoice{{
			Index: 0,
			Delta: delta,
		}},
	}
}

// makeChatFinishChunk 创建 Chat Completions 完成块
func makeChatFinishChunk(state *responsesChatStreamState, finishReason string) openAIChatCompletionsChunk {
	empty := ""
	return openAIChatCompletionsChunk{
		ID:      state.ID,
		Object:  "chat.completion.chunk",
		Created: state.Created,
		Model:   state.Model,
		Choices: []openAIChatChunkChoice{{
			Index:        0,
			Delta:        openAIChatDelta{Content: &empty},
			FinishReason: &finishReason,
		}},
	}
}

// finalizeResponsesChatStream 完成 Responses → Chat Completions 流
// 发送完成原因增量块和可选的用量信息块
func finalizeResponsesChatStream(state *responsesChatStreamState) []openAIChatCompletionsChunk {
	if state.Finalized {
		return nil
	}
	state.Finalized = true
	finishReason := "stop"
	if state.SawToolCall {
		finishReason = "tool_calls"
	}
	chunks := []openAIChatCompletionsChunk{makeChatFinishChunk(state, finishReason)}
	// 如果需要包含用量信息，追加用量块
	if state.IncludeUsage && state.Usage != nil {
		chunks = append(chunks, openAIChatCompletionsChunk{
			ID:      state.ID,
			Object:  "chat.completion.chunk",
			Created: state.Created,
			Model:   state.Model,
			Choices: []openAIChatChunkChoice{},
			Usage:   state.Usage,
		})
	}
	return chunks
}

// isTerminalResponsesEvent 判断是否为终止类型的 Responses 事件
func isTerminalResponsesEvent(eventType string) bool {
	switch eventType {
	case "response.completed", "response.done", "response.incomplete", "response.failed":
		return true
	default:
		return false
	}
}

// generateResponsesID 生成 Responses API 响应 ID（resp_ 前缀）
func generateResponsesID() string {
	b := make([]byte, 12)
	_, _ = rand.Read(b)
	return "resp_" + hex.EncodeToString(b)
}

// generateItemID 生成输出项 ID（item_ 前缀）
func generateItemID() string {
	b := make([]byte, 12)
	_, _ = rand.Read(b)
	return "item_" + hex.EncodeToString(b)
}

// generateChatCmplID 生成 Chat Completions 响应 ID（chatcmpl- 前缀）
func generateChatCmplID() string {
	b := make([]byte, 12)
	_, _ = rand.Read(b)
	return "chatcmpl-" + hex.EncodeToString(b)
}

// toResponsesCallID 将 ID 转换为 Responses API 调用 ID 格式
// 确保以 fc_ 前缀开头
func toResponsesCallID(id string) string {
	if strings.HasPrefix(id, "fc_") {
		return id
	}
	return "fc_" + id
}
