// openai_compat.go 实现 OpenAI 兼容协议与 Gemini 原生协议之间的双向转换
// 主要功能：
// 1. 请求转换：将 OpenAI 格式的请求（Chat Completions / Responses）转换为 Gemini 原生格式
// 2. 响应转换：将 Gemini 原生响应转换为 OpenAI 格式（非流式和流式）
// 3. JSON Schema 清理：将 OpenAI 的 JSON Schema 转换为 Gemini 兼容的格式
package gemini

import (
	"bufio"           // 用于流式响应的逐行扫描
	"bytes"           // 用于字节切片操作
	"encoding/base64" // 用于图片 Data URI 的 Base64 解码
	"encoding/json"   // JSON 序列化/反序列化
	"fmt"             // 格式化输出
	"io"              // io.ReadCloser 接口
	"net/http"        // HTTP 响应处理
	"strings"         // 字符串工具函数
	"time"            // 时间戳生成

	"sub2api/server/internal/provider" // Provider 接口定义
)

// ==================== OpenAI Responses API 类型定义 ====================
// 以下结构体对应 OpenAI Responses API 的请求和响应格式

// openAIResponsesRequest OpenAI Responses API 请求体
// 对应 /v1/responses 端点的请求格式
type openAIResponsesRequest struct {
	Model           string                    `json:"model"`                       // 模型名称
	Instructions    string                    `json:"instructions,omitempty"`      // 系统指令（类似 system prompt）
	Input           json.RawMessage           `json:"input"`                       // 输入内容，可以是字符串或消息数组
	MaxOutputTokens *int                      `json:"max_output_tokens,omitempty"` // 最大输出 Token 数
	Temperature     *float64                  `json:"temperature,omitempty"`       // 温度参数，控制随机性
	TopP            *float64                  `json:"top_p,omitempty"`             // Top-P 采样参数
	Stream          bool                      `json:"stream,omitempty"`            // 是否启用流式响应
	Tools           []openAIResponsesTool     `json:"tools,omitempty"`             // 工具列表
	Include         []string                  `json:"include,omitempty"`           // 包含的额外信息（如推理内容）
	Store           *bool                     `json:"store,omitempty"`             // 是否存储响应
	Reasoning       *openAIResponsesReasoning `json:"reasoning,omitempty"`         // 推理配置
	ToolChoice      json.RawMessage           `json:"tool_choice,omitempty"`       // 工具选择策略
	ServiceTier     string                    `json:"service_tier,omitempty"`      // 服务层级
}

// openAIResponsesReasoning 推理/思考配置
type openAIResponsesReasoning struct {
	Effort  string `json:"effort"`            // 推理力度：low/medium/high/xhigh
	Summary string `json:"summary,omitempty"` // 推理摘要模式：auto
}

// openAIResponsesInputItem Responses API 的输入项
// 可以是系统消息、用户消息、助手消息、函数调用或函数调用结果
type openAIResponsesInputItem struct {
	Type      string          `json:"type,omitempty"`      // 类型：function_call / function_call_output
	Role      string          `json:"role,omitempty"`      // 角色：system / user / assistant
	Content   json.RawMessage `json:"content,omitempty"`   // 内容，可以是字符串或内容数组
	CallID    string          `json:"call_id,omitempty"`   // 函数调用 ID
	Name      string          `json:"name,omitempty"`      // 函数名称
	Arguments string          `json:"arguments,omitempty"` // 函数调用参数（JSON 字符串）
	Output    string          `json:"output,omitempty"`    // 函数调用输出结果
}

// openAIResponsesContentPart Responses API 的内容部分
type openAIResponsesContentPart struct {
	Type     string `json:"type"`                // 类型：input_text / output_text / input_image
	Text     string `json:"text,omitempty"`      // 文本内容
	ImageURL string `json:"image_url,omitempty"` // 图片 URL（Data URI 格式）
}

// openAIResponsesTool Responses API 的工具定义
type openAIResponsesTool struct {
	Type        string          `json:"type"`                  // 类型：function / web_search
	Name        string          `json:"name,omitempty"`        // 函数名称
	Description string          `json:"description,omitempty"` // 函数描述
	Parameters  json.RawMessage `json:"parameters,omitempty"`  // 函数参数 JSON Schema
	Strict      *bool           `json:"strict,omitempty"`      // 是否严格模式
}

// openAIResponsesResponse OpenAI Responses API 响应体
type openAIResponsesResponse struct {
	ID                string                           `json:"id"`                           // 响应 ID
	Object            string                           `json:"object"`                       // 对象类型：response
	Model             string                           `json:"model"`                        // 使用的模型
	Status            string                           `json:"status"`                       // 状态：completed / incomplete / failed
	Output            []openAIResponsesOutput          `json:"output"`                       // 输出项列表
	Usage             *openAIResponsesUsage            `json:"usage,omitempty"`              // Token 用量
	IncompleteDetails *openAIResponsesIncompleteDetail `json:"incomplete_details,omitempty"` // 不完整详情
	Error             *openAIResponsesError            `json:"error,omitempty"`              // 错误信息
}

// openAIResponsesError 错误信息
type openAIResponsesError struct {
	Code    string `json:"code"`    // 错误码
	Message string `json:"message"` // 错误消息
}

// openAIResponsesIncompleteDetail 不完整响应的详情
type openAIResponsesIncompleteDetail struct {
	Reason string `json:"reason"` // 原因：max_output_tokens 等
}

// openAIResponsesOutput Responses API 的输出项
// 可以是消息、函数调用、推理内容或网页搜索调用
type openAIResponsesOutput struct {
	Type      string                       `json:"type"`                // 类型：message / function_call / reasoning / web_search_call
	ID        string                       `json:"id,omitempty"`        // 输出项 ID
	Role      string                       `json:"role,omitempty"`      // 角色：assistant
	Content   []openAIResponsesContentPart `json:"content,omitempty"`   // 内容部分列表
	Status    string                       `json:"status,omitempty"`    // 状态：completed / in_progress
	Summary   []openAIResponsesSummary     `json:"summary,omitempty"`   // 推理摘要
	CallID    string                       `json:"call_id,omitempty"`   // 函数调用 ID
	Name      string                       `json:"name,omitempty"`      // 函数名称
	Arguments string                       `json:"arguments,omitempty"` // 函数参数
	Action    *openAIWebSearchAction       `json:"action,omitempty"`    // 网页搜索动作
}

// openAIResponsesSummary 推理摘要
type openAIResponsesSummary struct {
	Type string `json:"type"` // 类型：summary_text
	Text string `json:"text"` // 摘要文本
}

// openAIWebSearchAction 网页搜索动作
type openAIWebSearchAction struct {
	Query string `json:"query,omitempty"` // 搜索查询
}

// openAIResponsesUsage Responses API 的 Token 用量
type openAIResponsesUsage struct {
	InputTokens        int                               `json:"input_tokens"`                   // 输入 Token 数
	OutputTokens       int                               `json:"output_tokens"`                  // 输出 Token 数
	TotalTokens        int                               `json:"total_tokens"`                   // 总 Token 数
	InputTokensDetails *openAIResponsesInputTokenDetails `json:"input_tokens_details,omitempty"` // 输入 Token 详情（缓存等）
	OutputTokenDetails *openAIResponsesOutputTokenDetail `json:"output_token_details,omitempty"` // 输出 Token 详情（推理等）
}

// openAIResponsesInputTokenDetails 输入 Token 详情
type openAIResponsesInputTokenDetails struct {
	CachedTokens int `json:"cached_tokens,omitempty"` // 缓存命中 Token 数
}

// openAIResponsesOutputTokenDetail 输出 Token 详情
type openAIResponsesOutputTokenDetail struct {
	ReasoningTokens int `json:"reasoning_tokens,omitempty"` // 推理 Token 数
}

// openAIResponsesStreamEvent Responses API 的流式事件
type openAIResponsesStreamEvent struct {
	Type           string                   `json:"type"`                      // 事件类型
	Response       *openAIResponsesResponse `json:"response,omitempty"`        // 响应对象（部分事件携带）
	Item           *openAIResponsesOutput   `json:"item,omitempty"`            // 输出项（部分事件携带）
	OutputIndex    int                      `json:"output_index,omitempty"`    // 输出项索引
	ContentIndex   int                      `json:"content_index,omitempty"`   // 内容索引
	Delta          string                   `json:"delta,omitempty"`           // 增量内容
	ItemID         string                   `json:"item_id,omitempty"`         // 输出项 ID
	CallID         string                   `json:"call_id,omitempty"`         // 函数调用 ID
	Name           string                   `json:"name,omitempty"`            // 函数名称
	SummaryIndex   int                      `json:"summary_index,omitempty"`   // 摘要索引
	SequenceNumber int                      `json:"sequence_number,omitempty"` // 序列号
}

// ==================== OpenAI Chat Completions API 类型定义 ====================
// 以下结构体对应 OpenAI Chat Completions API 的请求和响应格式

// openAIChatCompletionsRequest OpenAI Chat Completions API 请求体
// 对应 /v1/chat/completions 端点的请求格式
type openAIChatCompletionsRequest struct {
	Model               string                   `json:"model"`                           // 模型名称
	Messages            []openAIChatMessage      `json:"messages"`                        // 消息列表
	Instructions        string                   `json:"instructions,omitempty"`          // 系统指令
	MaxTokens           *int                     `json:"max_tokens,omitempty"`            // 最大 Token 数（旧版）
	MaxCompletionTokens *int                     `json:"max_completion_tokens,omitempty"` // 最大输出 Token 数（新版）
	Temperature         *float64                 `json:"temperature,omitempty"`           // 温度参数
	TopP                *float64                 `json:"top_p,omitempty"`                 // Top-P 采样参数
	Stream              bool                     `json:"stream,omitempty"`                // 是否流式
	StreamOptions       *openAIChatStreamOptions `json:"stream_options,omitempty"`        // 流式选项
	Tools               []openAIChatTool         `json:"tools,omitempty"`                 // 工具列表
	ToolChoice          json.RawMessage          `json:"tool_choice,omitempty"`           // 工具选择策略
	ReasoningEffort     string                   `json:"reasoning_effort,omitempty"`      // 推理力度
	Functions           []openAIChatFunction     `json:"functions,omitempty"`             // 函数列表（旧版，兼容）
	FunctionCall        json.RawMessage          `json:"function_call,omitempty"`         // 函数调用策略（旧版，兼容）
	ServiceTier         string                   `json:"service_tier,omitempty"`          // 服务层级
}

// openAIChatStreamOptions 流式响应选项
type openAIChatStreamOptions struct {
	IncludeUsage bool `json:"include_usage,omitempty"` // 是否在流式响应中包含用量信息
}

// openAIChatMessage Chat Completions API 的消息
type openAIChatMessage struct {
	Role             string                  `json:"role"`                        // 角色：system / user / assistant / tool / function
	Content          json.RawMessage         `json:"content,omitempty"`           // 内容，可以是字符串或内容数组
	ReasoningContent string                  `json:"reasoning_content,omitempty"` // 推理内容（思维链）
	Name             string                  `json:"name,omitempty"`              // 消息发送者名称
	ToolCalls        []openAIChatToolCall    `json:"tool_calls,omitempty"`        // 工具调用列表
	ToolCallID       string                  `json:"tool_call_id,omitempty"`      // 工具调用 ID
	FunctionCall     *openAIChatFunctionCall `json:"function_call,omitempty"`     // 函数调用（旧版，兼容）
}

// openAIChatContentPart Chat Completions API 的内容部分
type openAIChatContentPart struct {
	Type     string              `json:"type"`                // 类型：text / image_url
	Text     string              `json:"text,omitempty"`      // 文本内容
	ImageURL *openAIChatImageURL `json:"image_url,omitempty"` // 图片 URL
}

// openAIChatImageURL 图片 URL 结构
type openAIChatImageURL struct {
	URL string `json:"url"` // 图片地址（可以是 Data URI）
}

// openAIChatTool Chat Completions API 的工具定义
type openAIChatTool struct {
	Type     string              `json:"type"`               // 类型：function
	Function *openAIChatFunction `json:"function,omitempty"` // 函数定义
}

// openAIChatFunction Chat Completions API 的函数定义
type openAIChatFunction struct {
	Name        string          `json:"name"`                  // 函数名称
	Description string          `json:"description,omitempty"` // 函数描述
	Parameters  json.RawMessage `json:"parameters,omitempty"`  // 函数参数 JSON Schema
}

// openAIChatToolCall Chat Completions API 的工具调用
type openAIChatToolCall struct {
	Index    *int                   `json:"index,omitempty"` // 调用索引（流式时使用）
	ID       string                 `json:"id,omitempty"`    // 调用 ID
	Type     string                 `json:"type,omitempty"`  // 类型：function
	Function openAIChatFunctionCall `json:"function"`        // 函数调用详情
}

// openAIChatFunctionCall 函数调用详情
type openAIChatFunctionCall struct {
	Name      string `json:"name"`      // 函数名称
	Arguments string `json:"arguments"` // 函数参数（JSON 字符串）
}

// openAIChatCompletionsResponse Chat Completions API 的非流式响应
type openAIChatCompletionsResponse struct {
	ID      string             `json:"id"`              // 响应 ID
	Object  string             `json:"object"`          // 对象类型：chat.completion
	Created int64              `json:"created"`         // 创建时间戳
	Model   string             `json:"model"`           // 使用的模型
	Choices []openAIChatChoice `json:"choices"`         // 选择列表
	Usage   *openAIChatUsage   `json:"usage,omitempty"` // Token 用量
}

// openAIChatChoice Chat Completions API 的选择项
type openAIChatChoice struct {
	Index        int               `json:"index"`         // 选择索引
	Message      openAIChatMessage `json:"message"`       // 消息内容
	FinishReason string            `json:"finish_reason"` // 结束原因：stop / length / tool_calls
}

// openAIChatUsage Chat Completions API 的 Token 用量
type openAIChatUsage struct {
	PromptTokens        int                    `json:"prompt_tokens"`                       // 输入 Token 数
	CompletionTokens    int                    `json:"completion_tokens"`                   // 输出 Token 数
	TotalTokens         int                    `json:"total_tokens"`                        // 总 Token 数
	PromptTokensDetails *openAIChatTokenDetail `json:"prompt_tokens_details,omitempty"`     // 输入 Token 详情
	CompletionTokenInfo *openAIChatTokenDetail `json:"completion_tokens_details,omitempty"` // 输出 Token 详情
}

// openAIChatTokenDetail Token 详情（缓存/推理）
type openAIChatTokenDetail struct {
	CachedTokens    int `json:"cached_tokens,omitempty"`    // 缓存命中 Token 数
	ReasoningTokens int `json:"reasoning_tokens,omitempty"` // 推理 Token 数
}

// openAIChatCompletionsChunk Chat Completions API 的流式响应块
type openAIChatCompletionsChunk struct {
	ID      string                  `json:"id"`              // 响应 ID
	Object  string                  `json:"object"`          // 对象类型：chat.completion.chunk
	Created int64                   `json:"created"`         // 创建时间戳
	Model   string                  `json:"model"`           // 使用的模型
	Choices []openAIChatChunkChoice `json:"choices"`         // 选择列表
	Usage   *openAIChatUsage        `json:"usage,omitempty"` // Token 用量（仅在最后一个块中）
}

// openAIChatChunkChoice 流式响应的选择项
type openAIChatChunkChoice struct {
	Index        int             `json:"index"`         // 选择索引
	Delta        openAIChatDelta `json:"delta"`         // 增量内容
	FinishReason *string         `json:"finish_reason"` // 结束原因（仅在最后一个块中非空）
}

// openAIChatDelta 流式响应的增量内容
type openAIChatDelta struct {
	Role             string               `json:"role,omitempty"`              // 角色（仅在第一个块中）
	Content          *string              `json:"content,omitempty"`           // 文本内容增量
	ReasoningContent *string              `json:"reasoning_content,omitempty"` // 推理内容增量
	ToolCalls        []openAIChatToolCall `json:"tool_calls,omitempty"`        // 工具调用增量
}

// ==================== Gemini 原生 API 类型定义 ====================
// 以下结构体对应 Google Gemini API 的请求和响应格式

// GeminiRequest Gemini API 请求体
type GeminiRequest struct {
	Contents          []GeminiContent         `json:"contents"`                    // 对话内容列表
	SystemInstruction *GeminiContent          `json:"systemInstruction,omitempty"` // 系统指令
	GenerationConfig  *GeminiGenerationConfig `json:"generationConfig,omitempty"`  // 生成配置
	Tools             []GeminiToolDeclaration `json:"tools,omitempty"`             // 工具声明列表
	ToolConfig        *GeminiToolConfig       `json:"toolConfig,omitempty"`        // 工具配置
	SafetySettings    []GeminiSafetySetting   `json:"safetySettings,omitempty"`    // 安全设置
	SessionID         string                  `json:"sessionId,omitempty"`         // 会话 ID
}

// GeminiContent Gemini 的内容单元，包含角色和部件列表
type GeminiContent struct {
	Role  string       `json:"role"`  // 角色：user / model
	Parts []GeminiPart `json:"parts"` // 内容部件列表
}

// GeminiPart Gemini 的内容部件，可以是文本、思维、图片、函数调用或函数响应
type GeminiPart struct {
	Text             string                  `json:"text,omitempty"`             // 文本内容
	Thought          bool                    `json:"thought,omitempty"`          // 是否为思维/推理内容
	ThoughtSignature string                  `json:"thoughtSignature,omitempty"` // 思维签名（用于验证）
	InlineData       *GeminiInlineData       `json:"inlineData,omitempty"`       // 内联二进制数据（图片等）
	FunctionCall     *GeminiFunctionCall     `json:"functionCall,omitempty"`     // 函数调用
	FunctionResponse *GeminiFunctionResponse `json:"functionResponse,omitempty"` // 函数响应
}

// GeminiInlineData 内联二进制数据
type GeminiInlineData struct {
	MimeType string `json:"mimeType"` // MIME 类型
	Data     string `json:"data"`     // Base64 编码的数据
}

// GeminiFunctionCall Gemini 的函数调用
type GeminiFunctionCall struct {
	Name string `json:"name"`           // 函数名称
	Args any    `json:"args,omitempty"` // 函数参数（任意 JSON 对象）
	ID   string `json:"id,omitempty"`   // 调用 ID
}

// GeminiFunctionResponse Gemini 的函数响应
type GeminiFunctionResponse struct {
	Name     string         `json:"name"`         // 函数名称
	Response map[string]any `json:"response"`     // 函数返回结果
	ID       string         `json:"id,omitempty"` // 调用 ID
}

// GeminiGenerationConfig Gemini 的生成配置
type GeminiGenerationConfig struct {
	MaxOutputTokens int                   `json:"maxOutputTokens,omitempty"` // 最大输出 Token 数
	Temperature     *float64              `json:"temperature,omitempty"`     // 温度参数
	TopP            *float64              `json:"topP,omitempty"`            // Top-P 采样参数
	TopK            *int                  `json:"topK,omitempty"`            // Top-K 采样参数
	ThinkingConfig  *GeminiThinkingConfig `json:"thinkingConfig,omitempty"`  // 思维/推理配置
	StopSequences   []string              `json:"stopSequences,omitempty"`   // 停止序列
	ImageConfig     *GeminiImageConfig    `json:"imageConfig,omitempty"`     // 图片生成配置
}

// GeminiImageConfig 图片生成配置
type GeminiImageConfig struct {
	AspectRatio string `json:"aspectRatio,omitempty"` // 宽高比
	ImageSize   string `json:"imageSize,omitempty"`   // 图片尺寸
}

// GeminiThinkingConfig 思维/推理配置
type GeminiThinkingConfig struct {
	IncludeThoughts bool `json:"includeThoughts"`          // 是否包含思维内容
	ThinkingBudget  int  `json:"thinkingBudget,omitempty"` // 思维 Token 预算
}

// GeminiToolDeclaration Gemini 的工具声明
type GeminiToolDeclaration struct {
	FunctionDeclarations []GeminiFunctionDecl `json:"functionDeclarations,omitempty"` // 函数声明列表
	GoogleSearch         *GeminiGoogleSearch  `json:"googleSearch,omitempty"`         // Google 搜索工具
}

// GeminiFunctionDecl Gemini 的函数声明
type GeminiFunctionDecl struct {
	Name        string         `json:"name"`                  // 函数名称
	Description string         `json:"description,omitempty"` // 函数描述
	Parameters  map[string]any `json:"parameters,omitempty"`  // 函数参数 Schema
}

// GeminiGoogleSearch Google 搜索工具配置
type GeminiGoogleSearch struct {
	EnhancedContent *GeminiEnhancedContent `json:"enhancedContent,omitempty"` // 增强内容配置
}

// GeminiEnhancedContent 增强内容配置
type GeminiEnhancedContent struct {
	ImageSearch *GeminiImageSearch `json:"imageSearch,omitempty"` // 图片搜索配置
}

// GeminiImageSearch 图片搜索配置
type GeminiImageSearch struct {
	MaxResultCount int `json:"maxResultCount,omitempty"` // 最大结果数
}

// GeminiToolConfig 工具调用配置
type GeminiToolConfig struct {
	FunctionCallingConfig *GeminiFunctionCallingConfig `json:"functionCallingConfig,omitempty"` // 函数调用配置
}

// GeminiFunctionCallingConfig 函数调用配置
type GeminiFunctionCallingConfig struct {
	Mode string `json:"mode,omitempty"` // 模式：AUTO / NONE / VALIDATED
}

// GeminiSafetySetting 安全设置
type GeminiSafetySetting struct {
	Category  string `json:"category"`  // 伤害类别
	Threshold string `json:"threshold"` // 阈值级别
}

// GeminiResponse Gemini API 响应体
type GeminiResponse struct {
	Candidates    []GeminiCandidate    `json:"candidates,omitempty"`    // 候选响应列表
	UsageMetadata *GeminiUsageMetadata `json:"usageMetadata,omitempty"` // Token 用量元数据
	ResponseID    string               `json:"responseId,omitempty"`    // 响应 ID
	ModelVersion  string               `json:"modelVersion,omitempty"`  // 模型版本
}

// GeminiCandidate Gemini 的候选响应
type GeminiCandidate struct {
	Content           *GeminiContent           `json:"content,omitempty"`           // 响应内容
	FinishReason      string                   `json:"finishReason,omitempty"`      // 结束原因：STOP / MAX_TOKENS
	Index             int                      `json:"index,omitempty"`             // 候选索引
	GroundingMetadata *GeminiGroundingMetadata `json:"groundingMetadata,omitempty"` // 搜索 grounding 元数据
}

// GeminiUsageMetadata Gemini 的 Token 用量元数据
type GeminiUsageMetadata struct {
	PromptTokenCount        int                 `json:"promptTokenCount,omitempty"`        // 输入 Token 数
	CandidatesTokenCount    int                 `json:"candidatesTokenCount,omitempty"`    // 输出 Token 数
	CachedContentTokenCount int                 `json:"cachedContentTokenCount,omitempty"` // 缓存内容 Token 数
	TotalTokenCount         int                 `json:"totalTokenCount,omitempty"`         // 总 Token 数
	ThoughtsTokenCount      int                 `json:"thoughtsTokenCount,omitempty"`      // 思维 Token 数
	CandidatesTokensDetails []GeminiTokenDetail `json:"candidatesTokensDetails,omitempty"` // 输出 Token 详情
	PromptTokensDetails     []GeminiTokenDetail `json:"promptTokensDetails,omitempty"`     // 输入 Token 详情
}

// GeminiTokenDetail Token 详情（按模态分类）
type GeminiTokenDetail struct {
	Modality   string `json:"modality"`   // 模态：TEXT / IMAGE
	TokenCount int    `json:"tokenCount"` // Token 数量
}

// GeminiGroundingMetadata 搜索 grounding 元数据
type GeminiGroundingMetadata struct {
	WebSearchQueries []string               `json:"webSearchQueries,omitempty"` // 搜索查询列表
	GroundingChunks  []GeminiGroundingChunk `json:"groundingChunks,omitempty"`  // Grounding 片段列表
}

// GeminiGroundingChunk Grounding 片段
type GeminiGroundingChunk struct {
	Web *GeminiGroundingWeb `json:"web,omitempty"` // 网页信息
}

// GeminiGroundingWeb Grounding 网页信息
type GeminiGroundingWeb struct {
	Title string `json:"title,omitempty"` // 网页标题
	URI   string `json:"uri,omitempty"`   // 网页 URI
}

// ==================== 路径标准化与请求转换 ====================

// normalizeGeminiOpenAIPath 将非标准化的 OpenAI 路径标准化为带版本前缀的路径
// 例如 /chat/completions → /v1/chat/completions
func normalizeGeminiOpenAIPath(path string) string {
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

// convertGeminiOpenAIRequest 将 OpenAI 格式的请求体转换为 Gemini 原生格式
// 根据请求路径区分两种转换：
// - /v1/chat/completions: Chat Completions → Responses → Gemini（两步转换）
// - /v1/responses: Responses → Gemini（一步转换）
// 返回值：(转换后的请求体, 模型名称, 是否流式, 是否包含用量, 错误)
func convertGeminiOpenAIRequest(publicPath string, body []byte) ([]byte, string, bool, bool, error) {
	switch normalizeGeminiOpenAIPath(publicPath) {
	case "/v1/chat/completions":
		// Chat Completions 路径：先转为 Responses 格式，再转为 Gemini 格式
		var req openAIChatCompletionsRequest
		if err := json.Unmarshal(body, &req); err != nil {
			return nil, "", false, false, err
		}
		// 第一步：Chat Completions → Responses
		responsesReq, err := geminiChatCompletionsToResponses(&req)
		if err != nil {
			return nil, "", false, false, err
		}
		// 第二步：Responses → Gemini
		gemReq, err := responsesToGeminiRequest(responsesReq)
		if err != nil {
			return nil, "", false, false, err
		}
		converted, err := json.Marshal(gemReq)
		if err != nil {
			return nil, "", false, false, err
		}
		// 检查是否需要在流式响应中包含用量信息
		includeUsage := req.StreamOptions != nil && req.StreamOptions.IncludeUsage
		return converted, strings.TrimSpace(req.Model), req.Stream, includeUsage, nil
	case "/v1/responses", "/backend-api/codex/responses":
		// Responses 路径：直接转为 Gemini 格式
		var req openAIResponsesRequest
		if err := json.Unmarshal(body, &req); err != nil {
			return nil, "", false, false, err
		}
		gemReq, err := responsesToGeminiRequest(&req)
		if err != nil {
			return nil, "", false, false, err
		}
		converted, err := json.Marshal(gemReq)
		if err != nil {
			return nil, "", false, false, err
		}
		return converted, strings.TrimSpace(req.Model), req.Stream, false, nil
	default:
		// 非 OpenAI 兼容路径，直接透传原始请求体
		return body, "", false, false, nil
	}
}

// geminiChatCompletionsToResponses 将 Chat Completions 请求转换为 Responses 请求
// 转换逻辑：
// 1. 将 messages 转换为 Responses 的 input 格式
// 2. 将 instructions 作为 system 消息插入
// 3. 转换 max_tokens / max_completion_tokens（最小值 128）
// 4. 转换 reasoning_effort 为 reasoning 配置
// 5. 转换 tools / functions 为 Responses 的工具格式
func geminiChatCompletionsToResponses(req *openAIChatCompletionsRequest) (*openAIResponsesRequest, error) {
	// 将 Chat 消息列表转换为 Responses 输入项
	input, err := convertGeminiChatMessagesToResponsesInput(req.Messages)
	if err != nil {
		return nil, err
	}
	// 如果有 instructions，作为 system 消息插入到输入列表最前面
	if strings.TrimSpace(req.Instructions) != "" {
		instructionJSON, err := json.Marshal(req.Instructions)
		if err != nil {
			return nil, err
		}
		input = append([]openAIResponsesInputItem{{
			Role:    "system",
			Content: instructionJSON,
		}}, input...)
	}
	inputJSON, err := json.Marshal(input)
	if err != nil {
		return nil, err
	}
	storeFalse := false
	// 构建 Responses 请求，设置通用字段
	out := &openAIResponsesRequest{
		Model:       req.Model,
		Input:       inputJSON,
		Temperature: req.Temperature,
		TopP:        req.TopP,
		Stream:      req.Stream,
		Include:     []string{"reasoning.encrypted_content"}, // 请求包含加密的推理内容
		Store:       &storeFalse,                             // 不存储响应
		ServiceTier: req.ServiceTier,
	}
	// 处理 max_tokens / max_completion_tokens，取较大值，最小为 128
	maxTokens := 0
	if req.MaxTokens != nil {
		maxTokens = *req.MaxTokens
	}
	if req.MaxCompletionTokens != nil {
		maxTokens = *req.MaxCompletionTokens
	}
	if maxTokens > 0 {
		if maxTokens < 128 {
			maxTokens = 128 // Gemini 要求最小输出 Token 数为 128
		}
		out.MaxOutputTokens = &maxTokens
	}
	// 转换推理力度配置
	if req.ReasoningEffort != "" {
		out.Reasoning = &openAIResponsesReasoning{
			Effort:  req.ReasoningEffort,
			Summary: "auto", // 自动生成推理摘要
		}
	}
	// 转换工具列表（兼容新版 tools 和旧版 functions）
	if len(req.Tools) > 0 || len(req.Functions) > 0 {
		out.Tools = geminiConvertChatToolsToResponses(req.Tools, req.Functions)
	}
	// 转换工具选择策略（兼容新版 tool_choice 和旧版 function_call）
	if len(req.ToolChoice) > 0 {
		out.ToolChoice = req.ToolChoice
	} else if len(req.FunctionCall) > 0 {
		choice, err := geminiConvertChatFunctionCallToToolChoice(req.FunctionCall)
		if err != nil {
			return nil, err
		}
		out.ToolChoice = choice
	}
	return out, nil
}

// convertGeminiChatMessagesToResponsesInput 将 Chat Completions 的消息列表批量转换为 Responses 输入项
func convertGeminiChatMessagesToResponsesInput(msgs []openAIChatMessage) ([]openAIResponsesInputItem, error) {
	var out []openAIResponsesInputItem
	for _, m := range msgs {
		items, err := geminiChatMessageToResponsesItems(m)
		if err != nil {
			return nil, err
		}
		out = append(out, items...)
	}
	return out, nil
}

// geminiChatMessageToResponsesItems 将单条 Chat 消息转换为 Responses 输入项
// 根据消息角色（system/user/assistant/tool/function）进行不同的转换：
// - system → system 角色输入项
// - user → user 角色输入项
// - assistant → 可能生成文本输出项 + 函数调用项
// - tool/function → 函数调用输出项
func geminiChatMessageToResponsesItems(m openAIChatMessage) ([]openAIResponsesInputItem, error) {
	switch m.Role {
	case "system":
		// 系统消息：直接转换为 system 角色的输入项
		content, err := geminiMarshalChatInputContent(m.Content)
		if err != nil {
			return nil, err
		}
		return []openAIResponsesInputItem{{Role: "system", Content: content}}, nil
	case "user":
		// 用户消息：直接转换为 user 角色的输入项
		content, err := geminiMarshalChatInputContent(m.Content)
		if err != nil {
			return nil, err
		}
		return []openAIResponsesInputItem{{Role: "user", Content: content}}, nil
	case "assistant":
		// 助手消息：可能包含文本和工具调用，需要拆分为多个输入项
		var items []openAIResponsesInputItem
		// 提取文本内容
		text, err := geminiParseAssistantContent(m.Content)
		if err != nil {
			return nil, err
		}
		// 如果有文本，创建一个 assistant 角色的输出文本项
		if text != "" {
			partsJSON, _ := json.Marshal([]openAIResponsesContentPart{{Type: "output_text", Text: text}})
			items = append(items, openAIResponsesInputItem{Role: "assistant", Content: partsJSON})
		}
		// 将工具调用转换为 function_call 类型的输入项
		for _, tc := range m.ToolCalls {
			args := tc.Function.Arguments
			if args == "" {
				args = "{}" // 空参数默认为空对象
			}
			items = append(items, openAIResponsesInputItem{
				Type:      "function_call",
				CallID:    tc.ID,
				Name:      tc.Function.Name,
				Arguments: args,
			})
		}
		return items, nil
	case "tool", "function":
		// 工具/函数响应：转换为 function_call_output 类型的输入项
		output, err := geminiParseChatContent(m.Content)
		if err != nil {
			return nil, err
		}
		// 获取调用 ID，旧版 function 角色使用函数名作为 ID
		callID := m.ToolCallID
		if m.Role == "function" && callID == "" {
			callID = m.Name
		}
		// 空输出用 "(empty)" 占位
		if output == "" {
			output = "(empty)"
		}
		return []openAIResponsesInputItem{{
			Type:   "function_call_output",
			CallID: callID,
			Output: output,
		}}, nil
	default:
		// 未知角色默认作为 user 处理
		content, err := geminiMarshalChatInputContent(m.Content)
		if err != nil {
			return nil, err
		}
		return []openAIResponsesInputItem{{Role: "user", Content: content}}, nil
	}
}

// responsesToGeminiRequest 将 OpenAI Responses 请求转换为 Gemini 原生请求
// 转换逻辑：
// 1. 将 Responses 的 input 转换为 Gemini 的 contents 和 systemInstruction
// 2. 将 instructions 合并到 systemInstruction
// 3. 设置生成配置（温度、TopP、最大输出 Token、思维配置等）
// 4. 设置默认安全配置（全部关闭限制）
// 5. 转换工具声明和工具配置
func responsesToGeminiRequest(req *openAIResponsesRequest) (*GeminiRequest, error) {
	// 将 Responses 的输入转换为 Gemini 的系统指令和对话内容
	systemInstruction, contents, err := convertResponsesInputToGemini(req.Input)
	if err != nil {
		return nil, err
	}
	// 如果有 instructions，合并到系统指令中
	if strings.TrimSpace(req.Instructions) != "" {
		textPart := GeminiPart{Text: req.Instructions}
		if systemInstruction == nil {
			// 没有已有的系统指令，创建新的
			systemInstruction = &GeminiContent{Role: "user", Parts: []GeminiPart{textPart}}
		} else {
			// 已有系统指令，将 instructions 插入到最前面
			systemInstruction.Parts = append([]GeminiPart{textPart}, systemInstruction.Parts...)
		}
	}
	// 构建 Gemini 请求，设置基础生成配置和默认安全设置
	out := &GeminiRequest{
		Contents: contents,
		GenerationConfig: &GeminiGenerationConfig{
			Temperature: req.Temperature,
			TopP:        req.TopP,
		},
		SafetySettings: defaultGeminiSafetySettings(), // 默认关闭所有安全限制
	}
	if systemInstruction != nil {
		out.SystemInstruction = systemInstruction
	}
	// 设置最大输出 Token 数
	if req.MaxOutputTokens != nil && *req.MaxOutputTokens > 0 {
		out.GenerationConfig.MaxOutputTokens = *req.MaxOutputTokens
	}
	// 转换推理/思维配置
	if req.Reasoning != nil && req.Reasoning.Effort != "" {
		out.GenerationConfig.ThinkingConfig = &GeminiThinkingConfig{
			IncludeThoughts: true, // 始终包含思维内容
			ThinkingBudget:  geminiReasoningBudget(req.Reasoning.Effort),
		}
	}
	// 转换工具声明
	if len(req.Tools) > 0 {
		out.Tools = convertResponsesToolsToGemini(req.Tools)
	}
	// 如果有工具，设置工具调用配置（AUTO/NONE/VALIDATED）
	if len(out.Tools) > 0 {
		out.ToolConfig = &GeminiToolConfig{
			FunctionCallingConfig: &GeminiFunctionCallingConfig{
				Mode: geminiToolChoiceMode(req.ToolChoice),
			},
		}
	}
	// 如果生成配置中没有有效的 MaxOutputTokens，移除整个生成配置
	if out.GenerationConfig != nil && out.GenerationConfig.MaxOutputTokens == 0 {
		out.GenerationConfig = nil
	}
	return out, nil
}

// convertResponsesInputToGemini 将 Responses 的 input 字段转换为 Gemini 的系统指令和对话内容
// input 可以是纯字符串或输入项数组
// 返回值：(系统指令, 对话内容列表, 错误)
func convertResponsesInputToGemini(inputRaw json.RawMessage) (*GeminiContent, []GeminiContent, error) {
	// 尝试将 input 解析为纯字符串
	var inputStr string
	if err := json.Unmarshal(inputRaw, &inputStr); err == nil {
		return nil, []GeminiContent{{Role: "user", Parts: []GeminiPart{{Text: inputStr}}}}, nil
	}
	// 解析为输入项数组
	var items []openAIResponsesInputItem
	if err := json.Unmarshal(inputRaw, &items); err != nil {
		return nil, nil, err
	}
	var systemParts []GeminiPart     // 收集系统指令部件
	var contents []GeminiContent     // 收集对话内容
	toolNames := map[string]string{} // 记录函数调用 ID 到名称的映射，用于后续函数响应匹配
	// appendContent 辅助函数：将部件追加到对话内容中，相同角色的连续内容会合并
	appendContent := func(role string, parts []GeminiPart) {
		if len(parts) == 0 {
			return
		}
		// 如果最后一个内容的角色相同，合并部件
		if len(contents) > 0 && contents[len(contents)-1].Role == role {
			contents[len(contents)-1].Parts = append(contents[len(contents)-1].Parts, parts...)
			return
		}
		contents = append(contents, GeminiContent{Role: role, Parts: parts})
	}
	// 遍历每个输入项，根据类型进行转换
	for _, item := range items {
		switch {
		case item.Role == "system":
			// 系统消息：收集到系统指令部件中
			systemParts = append(systemParts, geminiResponsesContentToParts(item.Content)...)
		case item.Type == "function_call":
			// 函数调用：记录调用 ID→名称映射，转换为 Gemini 的 FunctionCall
			args := geminiParseArguments(item.Arguments)
			toolNames[item.CallID] = item.Name
			appendContent("model", []GeminiPart{{FunctionCall: &GeminiFunctionCall{Name: item.Name, Args: args, ID: item.CallID}}})
		case item.Type == "function_call_output":
			// 函数调用结果：查找对应的函数名称，转换为 Gemini 的 FunctionResponse
			name := toolNames[item.CallID]
			if name == "" {
				name = item.CallID // 如果找不到名称，使用调用 ID 作为名称
			}
			output := item.Output
			if output == "" {
				output = "(empty)" // 空输出用占位符
			}
			appendContent("user", []GeminiPart{{FunctionResponse: &GeminiFunctionResponse{
				Name: name,
				ID:   item.CallID,
				Response: map[string]any{
					"result": output, // Gemini 的函数响应需要 result 字段
				},
			}}})
		case item.Role == "user":
			// 用户消息：转换为 Gemini 的 user 角色
			appendContent("user", geminiResponsesContentToParts(item.Content))
		case item.Role == "assistant":
			// 助手消息：转换为 Gemini 的 model 角色
			appendContent("model", geminiResponsesContentToAssistantParts(item.Content))
		}
	}
	// 如果有系统指令部件，构建系统指令内容
	var systemInstruction *GeminiContent
	if len(systemParts) > 0 {
		systemInstruction = &GeminiContent{Role: "user", Parts: systemParts}
	}
	return systemInstruction, contents, nil
}

// geminiResponsesContentToParts 将 Responses 的内容（字符串或内容数组）转换为 Gemini 部件列表
// 支持文本和图片两种内容类型
func geminiResponsesContentToParts(raw json.RawMessage) []GeminiPart {
	if len(raw) == 0 {
		return nil
	}
	// 尝试解析为纯字符串
	var s string
	if err := json.Unmarshal(raw, &s); err == nil {
		if strings.TrimSpace(s) == "" {
			return nil
		}
		return []GeminiPart{{Text: s}}
	}
	// 解析为内容数组
	var parts []openAIResponsesContentPart
	if err := json.Unmarshal(raw, &parts); err != nil {
		return nil
	}
	out := make([]GeminiPart, 0, len(parts))
	for _, p := range parts {
		switch p.Type {
		case "input_text", "output_text", "text":
			// 文本内容：直接转换为 Gemini 文本部件
			if p.Text != "" {
				out = append(out, GeminiPart{Text: p.Text})
			}
		case "input_image":
			// 图片内容：将 Data URI 转换为 Gemini 的内联数据
			if inline := geminiDataURIToInlineData(p.ImageURL); inline != nil {
				out = append(out, GeminiPart{InlineData: inline})
			}
		}
	}
	return out
}

// geminiResponsesContentToAssistantParts 将 Responses 内容转换为助手消息的 Gemini 部件
// 过滤掉空的文本部件，只保留有实际内容的部件
func geminiResponsesContentToAssistantParts(raw json.RawMessage) []GeminiPart {
	parts := geminiResponsesContentToParts(raw)
	out := make([]GeminiPart, 0, len(parts))
	for _, part := range parts {
		if part.Text != "" || part.InlineData != nil {
			out = append(out, part)
		}
	}
	return out
}

// geminiReasoningBudget 将推理力度字符串转换为 Gemini 的思维 Token 预算
// low=1024, medium=4096, high=8192, xhigh=16384
func geminiReasoningBudget(effort string) int {
	switch effort {
	case "low":
		return 1024
	case "medium":
		return 4096
	case "high":
		return 8192
	case "xhigh":
		return 16384
	default:
		return 4096 // 默认中等力度
	}
}

// convertResponsesToolsToGemini 将 Responses 的工具列表转换为 Gemini 的工具声明
// 支持两种工具类型：
// - web_search/google_search: 转换为 Gemini 的 GoogleSearch 工具
// - function: 转换为 Gemini 的函数声明，并清理 JSON Schema
func convertResponsesToolsToGemini(tools []openAIResponsesTool) []GeminiToolDeclaration {
	if len(tools) == 0 {
		return nil
	}
	var funcDecls []GeminiFunctionDecl
	var hasWebSearch bool
	for _, t := range tools {
		switch t.Type {
		case "web_search", "google_search", "web_search_20250305":
			// 网络搜索工具：标记需要添加 GoogleSearch 声明
			hasWebSearch = true
		case "function":
			// 函数工具：解析参数 Schema 并清理
			var schema map[string]any
			if len(t.Parameters) > 0 && string(t.Parameters) != "null" {
				_ = json.Unmarshal(t.Parameters, &schema)
			}
			// 如果没有参数 Schema，创建默认的空对象 Schema
			if schema == nil {
				schema = map[string]any{"type": "object", "properties": map[string]any{}}
			}
			// 清理 Schema 中的未定义字段和无效结构
			DeepCleanUndefined(schema)
			schema = CleanJSONSchema(schema)
			// 清理后如果 Schema 为空，恢复为默认空对象
			if schema == nil {
				schema = map[string]any{"type": "object", "properties": map[string]any{}}
			}
			funcDecls = append(funcDecls, GeminiFunctionDecl{
				Name:        t.Name,
				Description: t.Description,
				Parameters:  schema,
			})
		}
	}
	// 构建工具声明列表
	var out []GeminiToolDeclaration
	if len(funcDecls) > 0 {
		out = append(out, GeminiToolDeclaration{FunctionDeclarations: funcDecls})
	}
	// 如果有网络搜索工具，添加 GoogleSearch 声明
	if hasWebSearch {
		out = append(out, GeminiToolDeclaration{GoogleSearch: &GeminiGoogleSearch{
			EnhancedContent: &GeminiEnhancedContent{
				ImageSearch: &GeminiImageSearch{MaxResultCount: 10}, // 最多返回 10 个图片搜索结果
			},
		}})
	}
	return out
}

// geminiToolChoiceMode 将 OpenAI 的工具选择模式转换为 Gemini 的函数调用模式
// OpenAI 模式 → Gemini 模式：
// - "none" → "NONE"（不调用任何函数）
// - "required" → "VALIDATED"（必须调用指定函数）
// - "auto" → "AUTO"（自动决定是否调用函数）
// - {"type":"function"} → "VALIDATED"
// - {"type":"none"} → "NONE"
func geminiToolChoiceMode(raw json.RawMessage) string {
	if len(raw) == 0 {
		return "AUTO" // 默认自动模式
	}
	// 尝试解析为字符串
	var s string
	if err := json.Unmarshal(raw, &s); err == nil {
		switch s {
		case "none":
			return "NONE"
		case "required":
			return "VALIDATED"
		case "auto":
			return "AUTO"
		default:
			return "AUTO"
		}
	}
	// 尝试解析为对象（指定特定函数的情况）
	var choice struct {
		Type string `json:"type"`
	}
	if err := json.Unmarshal(raw, &choice); err == nil {
		switch choice.Type {
		case "function":
			return "VALIDATED"
		case "none":
			return "NONE"
		}
	}
	return "AUTO" // 无法识别的模式默认自动
}

// AdaptGatewayResponse 适配 Gemini 网关响应为 OpenAI 格式
// 根据请求路径选择不同的转换方式：
// - /v1/chat/completions: Gemini → Responses → Chat Completions
// - /v1/responses: Gemini → Responses
// - 其他路径: 直接透传
func (p *Provider) AdaptGatewayResponse(req provider.GatewayRequest, resp *http.Response, body []byte) (*http.Response, []byte, error) {
	switch normalizeGeminiOpenAIPath(req.PublicPath) {
	case "/v1/chat/completions":
		return adaptGeminiToOpenAIChatResponse(req, resp, body)
	case "/v1/responses", "/backend-api/codex/responses":
		return adaptGeminiToOpenAIResponsesResponse(resp, body)
	default:
		return resp, body, nil
	}
}

// AdaptGatewayStream 适配 Gemini 网关流式响应为 OpenAI 格式
// 根据请求路径选择不同的流式适配器：
// - /v1/chat/completions: 使用 Chat Completions 流式适配器
// - /v1/responses: 使用 Responses 流式适配器
// - 其他路径: 直接透传
func (p *Provider) AdaptGatewayStream(req provider.GatewayRequest, resp *http.Response) (*http.Response, error) {
	switch normalizeGeminiOpenAIPath(req.PublicPath) {
	case "/v1/chat/completions":
		return wrapGeminiToOpenAIChatStream(req, resp), nil
	case "/v1/responses", "/backend-api/codex/responses":
		return wrapGeminiToOpenAIResponsesStream(resp), nil
	default:
		return resp, nil
	}
}

// adaptGeminiToOpenAIResponsesResponse 将 Gemini 响应转换为 OpenAI Responses 格式
// 步骤：解析 Gemini 响应 → 转换为 Responses 格式 → 序列化
func adaptGeminiToOpenAIResponsesResponse(resp *http.Response, body []byte) (*http.Response, []byte, error) {
	// 错误响应直接透传，不做转换
	if resp.StatusCode >= 400 {
		return resp, body, nil
	}
	// 解析 Gemini 响应
	gem, err := parseGeminiResponse(body)
	if err != nil {
		return resp, body, err
	}
	// 转换为 Responses 格式
	out := geminiToResponsesResponse(gem)
	converted, err := json.Marshal(out)
	if err != nil {
		return resp, body, err
	}
	// 更新响应头和内容
	resp.Header = cloneGeminiHeader(resp.Header)
	resp.Header.Set("Content-Type", "application/json")
	resp.ContentLength = int64(len(converted))
	return resp, converted, nil
}

// adaptGeminiToOpenAIChatResponse 将 Gemini 响应转换为 OpenAI Chat Completions 格式
// 步骤：先转为 Responses 格式 → 再转为 Chat Completions 格式
func adaptGeminiToOpenAIChatResponse(req provider.GatewayRequest, resp *http.Response, body []byte) (*http.Response, []byte, error) {
	// 第一步：转换为 Responses 格式
	resp, converted, err := adaptGeminiToOpenAIResponsesResponse(resp, body)
	if err != nil || resp.StatusCode >= 400 {
		return resp, converted, err
	}
	// 第二步：从 Responses 格式转换为 Chat Completions 格式
	var responsesResp openAIResponsesResponse
	if err := json.Unmarshal(converted, &responsesResp); err != nil {
		return resp, converted, err
	}
	chat := geminiResponsesToChatCompletions(&responsesResp, req.Model)
	chatBody, err := json.Marshal(chat)
	if err != nil {
		return resp, converted, err
	}
	// 更新响应头和内容
	resp.Header = cloneGeminiHeader(resp.Header)
	resp.Header.Set("Content-Type", "application/json")
	resp.ContentLength = int64(len(chatBody))
	return resp, chatBody, nil
}

// wrapGeminiToOpenAIResponsesStream 包装 Gemini 流式响应为 OpenAI Responses 流式格式
// 如果不是成功的流式响应，直接透传
func wrapGeminiToOpenAIResponsesStream(resp *http.Response) *http.Response {
	// 非成功状态码或非 SSE 流直接透传
	if resp.StatusCode >= 400 || !strings.Contains(strings.ToLower(resp.Header.Get("Content-Type")), "event-stream") {
		return resp
	}
	// 替换响应头和 Body 为适配器
	resp.Header = cloneGeminiHeader(resp.Header)
	resp.Header.Set("Content-Type", "text/event-stream; charset=utf-8")
	resp.Body = newGeminiResponsesStreamAdapter(resp.Body)
	resp.ContentLength = -1 // 流式响应长度未知
	return resp
}

// wrapGeminiToOpenAIChatStream 包装 Gemini 流式响应为 OpenAI Chat Completions 流式格式
// 如果不是成功的流式响应，直接透传
func wrapGeminiToOpenAIChatStream(req provider.GatewayRequest, resp *http.Response) *http.Response {
	// 非成功状态码或非 SSE 流直接透传
	if resp.StatusCode >= 400 || !strings.Contains(strings.ToLower(resp.Header.Get("Content-Type")), "event-stream") {
		return resp
	}
	resp.Header = cloneGeminiHeader(resp.Header)
	resp.Header.Set("Content-Type", "text/event-stream; charset=utf-8")
	resp.Body = newGeminiChatStreamAdapter(resp.Body, req.Model, req.IncludeUsage)
	resp.ContentLength = -1 // 流式响应长度未知
	return resp
}

// parseGeminiResponse 解析 Gemini 响应体
// 支持两种格式：
// - 直接的 GeminiResponse 对象
// - 包装在 {"response": ...} 中的对象（某些 API 版本使用）
func parseGeminiResponse(body []byte) (*GeminiResponse, error) {
	// 首先尝试直接解析
	var resp GeminiResponse
	if err := json.Unmarshal(body, &resp); err == nil && (len(resp.Candidates) > 0 || resp.UsageMetadata != nil || resp.ResponseID != "") {
		return &resp, nil
	}
	// 尝试解析包装格式
	var wrapped struct {
		Response GeminiResponse `json:"response"`
	}
	if err := json.Unmarshal(body, &wrapped); err != nil {
		return nil, err
	}
	return &wrapped.Response, nil
}

// geminiToResponsesResponse 将 Gemini 响应转换为 OpenAI Responses 格式
// 转换逻辑：
// 1. 提取响应 ID、模型版本等基础信息
// 2. 遍历候选结果的部件，分别处理思维内容、函数调用、文本和图片
// 3. 组装输出项列表
// 4. 处理用量信息
func geminiToResponsesResponse(resp *GeminiResponse) *openAIResponsesResponse {
	id := strings.TrimSpace(resp.ResponseID)
	if id == "" {
		id = "resp_gemini" // 默认响应 ID
	}
	out := &openAIResponsesResponse{
		ID:     id,
		Object: "response",
		Model:  strings.TrimSpace(resp.ModelVersion),
		Status: "completed",
	}
	var outputs []openAIResponsesOutput       // 输出项列表
	var msgParts []openAIResponsesContentPart // 消息文本/图片部件
	var hasToolCall bool                      // 标记是否有工具调用
	// 遍历第一个候选结果的部件
	if len(resp.Candidates) > 0 && resp.Candidates[0].Content != nil {
		candidate := resp.Candidates[0]
		for _, part := range candidate.Content.Parts {
			switch {
			case part.Thought && part.Text != "":
				// 思维内容：转换为 reasoning 类型的输出项
				outputs = append(outputs, openAIResponsesOutput{
					Type: "reasoning",
					ID:   "item_reasoning",
					Summary: []openAIResponsesSummary{{
						Type: "summary_text",
						Text: part.Text,
					}},
				})
			case part.FunctionCall != nil:
				// 函数调用：转换为 function_call 类型的输出项
				hasToolCall = true
				args := "{}"
				if part.FunctionCall.Args != nil {
					if b, err := json.Marshal(part.FunctionCall.Args); err == nil {
						args = string(b)
					}
				}
				callID := strings.TrimSpace(part.FunctionCall.ID)
				if callID == "" {
					callID = "fc_gemini" // 默认调用 ID
				}
				outputs = append(outputs, openAIResponsesOutput{
					Type:      "function_call",
					ID:        "item_" + callID,
					CallID:    callID,
					Name:      part.FunctionCall.Name,
					Arguments: args,
					Status:    "completed",
				})
			case part.Text != "":
				// 文本内容：收集到消息部件中
				msgParts = append(msgParts, openAIResponsesContentPart{Type: "output_text", Text: part.Text})
			case part.InlineData != nil && part.InlineData.Data != "":
				// 图片内容：转换为 base64 图片 URL 格式
				msgParts = append(msgParts, openAIResponsesContentPart{
					Type: "output_text",
					Text: fmt.Sprintf("![image](data:%s;base64,%s)", part.InlineData.MimeType, part.InlineData.Data),
				})
			}
		}
		// 处理搜索 grounding 元数据
		if candidate.GroundingMetadata != nil {
			// 如果有搜索查询，添加 web_search_call 输出项
			if query := firstGeminiGroundingQuery(candidate.GroundingMetadata); query != "" {
				outputs = append(outputs, openAIResponsesOutput{
					Type:   "web_search_call",
					ID:     "item_web_search",
					Status: "completed",
					Action: &openAIWebSearchAction{Query: query},
				})
			}
		}
		// 将 grounding 文本（搜索结果摘要）添加到消息部件中
		if grounding := buildGeminiGroundingText(candidate.GroundingMetadata); grounding != "" {
			msgParts = append(msgParts, openAIResponsesContentPart{Type: "output_text", Text: grounding})
		}
		// 处理完成原因
		switch candidate.FinishReason {
		case "MAX_TOKENS":
			// 达到最大 Token 数，标记为不完整
			out.Status = "incomplete"
			out.IncompleteDetails = &openAIResponsesIncompleteDetail{Reason: "max_output_tokens"}
		default:
			if hasToolCall {
				out.Status = "completed" // 工具调用完成
			}
		}
	}
	// 如果有消息部件，创建 message 类型的输出项
	if len(msgParts) > 0 {
		outputs = append(outputs, openAIResponsesOutput{
			Type:    "message",
			ID:      "item_message",
			Role:    "assistant",
			Content: msgParts,
			Status:  "completed",
		})
	}
	// 如果没有任何输出，创建一个空消息
	if len(outputs) == 0 {
		outputs = append(outputs, openAIResponsesOutput{
			Type:    "message",
			ID:      "item_message",
			Role:    "assistant",
			Content: []openAIResponsesContentPart{{Type: "output_text", Text: ""}},
			Status:  "completed",
		})
	}
	out.Output = outputs
	// 处理用量信息
	if resp.UsageMetadata != nil {
		// 计算实际输入 Token 数（总输入 - 缓存 Token）
		cached := resp.UsageMetadata.CachedContentTokenCount
		input := resp.UsageMetadata.PromptTokenCount - cached
		if input < 0 {
			input = 0
		}
		// 输出 Token = 候选 Token + 思维 Token
		output := resp.UsageMetadata.CandidatesTokenCount + resp.UsageMetadata.ThoughtsTokenCount
		out.Usage = &openAIResponsesUsage{
			InputTokens:  input,
			OutputTokens: output,
			TotalTokens:  input + output,
		}
		// 如果有缓存 Token，添加输入 Token 详情
		if cached > 0 {
			out.Usage.InputTokensDetails = &openAIResponsesInputTokenDetails{CachedTokens: cached}
		}
		// 如果有思维 Token，添加输出 Token 详情
		if resp.UsageMetadata.ThoughtsTokenCount > 0 {
			out.Usage.OutputTokenDetails = &openAIResponsesOutputTokenDetail{
				ReasoningTokens: resp.UsageMetadata.ThoughtsTokenCount,
			}
		}
	}
	return out
}

// geminiResponsesToChatCompletions 将 OpenAI Responses 格式转换为 Chat Completions 格式
// 遍历 Responses 的输出项，提取文本、推理内容和工具调用
func geminiResponsesToChatCompletions(resp *openAIResponsesResponse, model string) *openAIChatCompletionsResponse {
	out := &openAIChatCompletionsResponse{
		ID:      firstNonEmptyGemini(resp.ID, "chatcmpl_gemini"),
		Object:  "chat.completion",
		Created: time.Now().Unix(),
		Model:   firstNonEmptyGemini(strings.TrimSpace(model), strings.TrimSpace(resp.Model)),
	}
	var contentText, reasoningText string
	var toolCalls []openAIChatToolCall
	// 遍历输出项，提取各类内容
	for _, item := range resp.Output {
		switch item.Type {
		case "message":
			// 消息类型：提取文本内容
			for _, part := range item.Content {
				if part.Type == "output_text" {
					contentText += part.Text
				}
			}
		case "function_call":
			// 函数调用类型：转换为 Chat Completions 的工具调用格式
			toolCalls = append(toolCalls, openAIChatToolCall{
				ID:   item.CallID,
				Type: "function",
				Function: openAIChatFunctionCall{
					Name:      item.Name,
					Arguments: item.Arguments,
				},
			})
		case "reasoning":
			// 推理类型：提取推理摘要文本
			for _, summary := range item.Summary {
				if summary.Type == "summary_text" {
					reasoningText += summary.Text
				}
			}
		}
	}
	// 构建助手消息
	msg := openAIChatMessage{Role: "assistant"}
	if contentText != "" {
		raw, _ := json.Marshal(contentText)
		msg.Content = raw
	}
	if reasoningText != "" {
		msg.ReasoningContent = reasoningText // 推理内容放在单独字段
	}
	if len(toolCalls) > 0 {
		msg.ToolCalls = toolCalls
	}
	// 确定完成原因
	finishReason := "stop"
	if len(toolCalls) > 0 {
		finishReason = "tool_calls" // 有工具调用时标记为 tool_calls
	}
	if resp.Status == "incomplete" {
		finishReason = "length" // 不完整时标记为 length
	}
	out.Choices = []openAIChatChoice{{Index: 0, Message: msg, FinishReason: finishReason}}
	// 转换用量信息
	if resp.Usage != nil {
		out.Usage = &openAIChatUsage{
			PromptTokens:     resp.Usage.InputTokens,
			CompletionTokens: resp.Usage.OutputTokens,
			TotalTokens:      resp.Usage.TotalTokens,
		}
		// 如果有缓存 Token，添加输入 Token 详情
		if resp.Usage.InputTokensDetails != nil && resp.Usage.InputTokensDetails.CachedTokens > 0 {
			out.Usage.PromptTokensDetails = &openAIChatTokenDetail{CachedTokens: resp.Usage.InputTokensDetails.CachedTokens}
		}
		// 如果有推理 Token，添加输出 Token 详情
		if resp.Usage.OutputTokenDetails != nil && resp.Usage.OutputTokenDetails.ReasoningTokens > 0 {
			if out.Usage.CompletionTokenInfo == nil {
				out.Usage.CompletionTokenInfo = &openAIChatTokenDetail{}
			}
			out.Usage.CompletionTokenInfo.ReasoningTokens = resp.Usage.OutputTokenDetails.ReasoningTokens
		}
	}
	return out
}

// geminiResponsesStreamState Responses 流式适配器的状态
// 跟踪流式响应的进度，包括响应 ID、序列号、输出索引和 Token 用量等
type geminiResponsesStreamState struct {
	ResponseID           string // 响应 ID
	Model                string // 模型名称
	SequenceNumber       int    // 序列号，用于 SSE 事件的递增编号
	CreatedSent          bool   // 是否已发送 response.created 事件
	CompletedSent        bool   // 是否已发送 response.completed 事件
	OutputIndex          int    // 当前输出项的索引
	CurrentMessageItemID string // 当前消息输出项的 ID
	CurrentReasonItemID  string // 当前推理输出项的 ID
	InputTokens          int    // 输入 Token 数
	OutputTokens         int    // 输出 Token 数
	CacheReadTokens      int    // 缓存读取 Token 数
}

// newGeminiResponsesStreamAdapter 创建 Responses 流式适配器
// 将 Gemini 的 SSE 流转换为 OpenAI Responses 的 SSE 流
func newGeminiResponsesStreamAdapter(src io.ReadCloser) io.ReadCloser {
	return newGeminiSSETransformReadCloser(src, func(data string, st any) ([]string, error) {
		state := st.(*geminiResponsesStreamState)
		// 解析 Gemini 响应块
		gemResp, err := parseGeminiResponse([]byte(data))
		if err != nil {
			return nil, err
		}
		// 将 Gemini 响应转换为 Responses 流式事件
		events := geminiResponseToResponsesEvents(gemResp, state)
		out := make([]string, 0, len(events))
		for _, evt := range events {
			payload, err := json.Marshal(evt)
			if err != nil {
				return nil, err
			}
			// 格式化为 SSE 事件
			out = append(out, fmt.Sprintf("event: %s\ndata: %s\n\n", evt.Type, payload))
		}
		return out, nil
	}, &geminiResponsesStreamState{})
}

// geminiChatStreamState Chat Completions 流式适配器的状态
type geminiChatStreamState struct {
	ID                     string           // 响应 ID
	Model                  string           // 模型名称
	Created                int64            // 创建时间戳
	SentRole               bool             // 是否已发送角色信息
	SawToolCall            bool             // 是否看到了工具调用
	Finalized              bool             // 是否已发送终止事件
	NextToolCallIndex      int              // 下一个工具调用的索引
	OutputIndexToToolIndex map[int]int      // 输出索引到工具索引的映射
	IncludeUsage           bool             // 是否在流中包含用量信息
	Usage                  *openAIChatUsage // 累计用量信息
}

// newGeminiChatStreamAdapter 创建 Chat Completions 流式适配器
// 将 Gemini 的 SSE 流转换为 OpenAI Chat Completions 的 SSE 流
// 内部先转换为 Responses 流式事件，再转换为 Chat Completions 流式块
func newGeminiChatStreamAdapter(src io.ReadCloser, model string, includeUsage bool) io.ReadCloser {
	respState := &geminiResponsesStreamState{}
	chatState := &geminiChatStreamState{
		ID:                     "chatcmpl_gemini_stream",
		Model:                  model,
		Created:                time.Now().Unix(),
		OutputIndexToToolIndex: make(map[int]int),
		IncludeUsage:           includeUsage,
	}
	return newGeminiSSETransformReadCloser(src, func(data string, st any) ([]string, error) {
		// 解析 Gemini 响应块
		gemResp, err := parseGeminiResponse([]byte(data))
		if err != nil {
			return nil, err
		}
		// 第一步：转换为 Responses 流式事件
		resEvents := geminiResponseToResponsesEvents(gemResp, respState)
		var out []string
		for _, evt := range resEvents {
			// 第二步：将 Responses 事件转换为 Chat Completions 流式块
			chunks := geminiResponsesEventToChatChunks(&evt, chatState)
			for _, chunk := range chunks {
				payload, err := json.Marshal(chunk)
				if err != nil {
					return nil, err
				}
				out = append(out, fmt.Sprintf("data: %s\n\n", payload))
			}
			// 终止事件后发送 [DONE]
			if isGeminiTerminalResponsesEvent(evt.Type) {
				out = append(out, "data: [DONE]\n\n")
			}
		}
		return out, nil
	}, chatState)
}

// geminiSSETransformReadCloser SSE 流转换读取器
// 从源 SSE 流中读取数据，通过转换函数处理后输出新的 SSE 流
type geminiSSETransformReadCloser struct {
	src       io.ReadCloser                       // 原始响应体
	scanner   *bufio.Scanner                      // 行扫描器
	transform func(string, any) ([]string, error) // 转换函数
	state     any                                 // 状态对象
	pending   bytes.Buffer                        // 待输出的缓冲区
	doneEOF   bool                                // 是否已到达 EOF
}

// newGeminiSSETransformReadCloser 创建 SSE 流转换读取器
// 参数：src - 原始响应体，transform - 数据转换函数，state - 状态对象
func newGeminiSSETransformReadCloser(src io.ReadCloser, transform func(string, any) ([]string, error), state any) io.ReadCloser {
	scanner := bufio.NewScanner(src)
	scanner.Buffer(make([]byte, 0, 4096), 1024*1024) // 最大 1MB 行缓冲
	return &geminiSSETransformReadCloser{src: src, scanner: scanner, transform: transform, state: state}
}

// Read 实现 io.Reader 接口
// 从源 SSE 流中逐行读取 data: 行，通过转换函数处理后输出
func (r *geminiSSETransformReadCloser) Read(p []byte) (int, error) {
	// 当缓冲区为空时，继续从源读取并转换
	for r.pending.Len() == 0 {
		if !r.scanner.Scan() {
			if err := r.scanner.Err(); err != nil {
				return 0, err
			}
			// 源已结束，发送终止事件
			if !r.doneEOF {
				r.doneEOF = true
				r.appendEOF()
			}
			if r.pending.Len() == 0 {
				return 0, io.EOF
			}
			break
		}
		line := r.scanner.Text()
		// 只处理 data: 开头的行
		if !strings.HasPrefix(line, "data:") {
			continue
		}
		data := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
		// 跳过空数据和 [DONE] 标记
		if data == "" || data == "[DONE]" {
			continue
		}
		// 通过转换函数处理数据
		events, err := r.transform(data, r.state)
		if err != nil {
			return 0, err
		}
		for _, evt := range events {
			_, _ = r.pending.WriteString(evt)
		}
	}
	return r.pending.Read(p)
}

// appendEOF 在流结束时发送终止事件
// 根据状态类型分别处理 Responses 流和 Chat Completions 流
func (r *geminiSSETransformReadCloser) appendEOF() {
	switch state := r.state.(type) {
	case *geminiResponsesStreamState:
		// Responses 流：如果已创建但未完成，发送完成事件
		if !state.CompletedSent && state.CreatedSent {
			for _, evt := range geminiFinalizeResponsesEvents(state) {
				payload, err := json.Marshal(evt)
				if err != nil {
					continue
				}
				_, _ = r.pending.WriteString(fmt.Sprintf("event: %s\ndata: %s\n\n", evt.Type, payload))
			}
		}
	case *geminiChatStreamState:
		// Chat Completions 流：如果未终止，发送终止块和 [DONE]
		if !state.Finalized {
			for _, chunk := range geminiFinalizeChatChunks(state) {
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

// Close 关闭读取器，释放底层资源
func (r *geminiSSETransformReadCloser) Close() error { return r.src.Close() }

// geminiResponseToResponsesEvents 将 Gemini 响应转换为 Responses 流式事件列表
// 这是流式转换的核心函数，处理以下几种部件：
// - 思维内容（Thought）→ reasoning 事件
// - 函数调用（FunctionCall）→ function_call 事件
// - 文本内容 → output_text 事件
// - 图片内容 → output_text 事件（以 Markdown 图片格式）
func geminiResponseToResponsesEvents(resp *GeminiResponse, state *geminiResponsesStreamState) []openAIResponsesStreamEvent {
	var events []openAIResponsesStreamEvent
	// 首次收到数据时发送 response.created 事件
	if !state.CreatedSent {
		state.CreatedSent = true
		state.ResponseID = firstNonEmptyGemini(strings.TrimSpace(resp.ResponseID), "resp_gemini_stream")
		state.Model = strings.TrimSpace(resp.ModelVersion)
		events = append(events, geminiMakeResponsesCreatedEvent(state))
	}
	// 累计 Token 用量
	if resp.UsageMetadata != nil {
		state.CacheReadTokens = resp.UsageMetadata.CachedContentTokenCount
		state.InputTokens = resp.UsageMetadata.PromptTokenCount - state.CacheReadTokens
		if state.InputTokens < 0 {
			state.InputTokens = 0
		}
		state.OutputTokens = resp.UsageMetadata.CandidatesTokenCount + resp.UsageMetadata.ThoughtsTokenCount
	}
	// 如果没有候选结果，直接返回
	if len(resp.Candidates) == 0 || resp.Candidates[0].Content == nil {
		return events
	}
	candidate := resp.Candidates[0]
	// 遍历候选结果的每个部件
	for _, part := range candidate.Content.Parts {
		switch {
		case part.Thought:
			// 思维内容：先关闭当前消息项（如果有），然后处理推理
			if state.CurrentMessageItemID != "" {
				events = append(events, geminiCloseMessageItem(state)...)
			}
			// 如果没有推理项，创建新的
			if state.CurrentReasonItemID == "" {
				state.CurrentReasonItemID = fmt.Sprintf("item_reasoning_%d", state.OutputIndex)
				events = append(events, geminiMakeResponsesEvent(state, "response.output_item.added", &openAIResponsesStreamEvent{
					OutputIndex: state.OutputIndex,
					Item: &openAIResponsesOutput{
						Type: "reasoning",
						ID:   state.CurrentReasonItemID,
					},
				}))
			}
			// 发送推理摘要文本增量
			if part.Text != "" {
				events = append(events, geminiMakeResponsesEvent(state, "response.reasoning_summary_text.delta", &openAIResponsesStreamEvent{
					OutputIndex:  state.OutputIndex,
					SummaryIndex: 0,
					ItemID:       state.CurrentReasonItemID,
					Delta:        part.Text,
				}))
			}
		case part.FunctionCall != nil:
			// 函数调用：先关闭推理项和消息项（如果有）
			if state.CurrentReasonItemID != "" {
				events = append(events, geminiCloseReasoningItem(state)...)
			}
			if state.CurrentMessageItemID != "" {
				events = append(events, geminiCloseMessageItem(state)...)
			}
			// 创建函数调用输出项
			itemID := fmt.Sprintf("item_func_%d", state.OutputIndex)
			callID := firstNonEmptyGemini(part.FunctionCall.ID, fmt.Sprintf("call_%d", state.OutputIndex))
			// 添加函数调用项
			events = append(events, geminiMakeResponsesEvent(state, "response.output_item.added", &openAIResponsesStreamEvent{
				OutputIndex: state.OutputIndex,
				Item: &openAIResponsesOutput{
					Type:   "function_call",
					ID:     itemID,
					CallID: callID,
					Name:   part.FunctionCall.Name,
					Status: "in_progress",
				},
			}))
			// 发送函数参数增量
			if part.FunctionCall.Args != nil {
				if b, err := json.Marshal(part.FunctionCall.Args); err == nil {
					events = append(events, geminiMakeResponsesEvent(state, "response.function_call_arguments.delta", &openAIResponsesStreamEvent{
						OutputIndex: state.OutputIndex,
						ItemID:      itemID,
						CallID:      callID,
						Name:        part.FunctionCall.Name,
						Delta:       string(b),
					}))
				}
			}
			// 函数参数完成
			events = append(events, geminiMakeResponsesEvent(state, "response.function_call_arguments.done", &openAIResponsesStreamEvent{
				OutputIndex: state.OutputIndex,
				ItemID:      itemID,
				CallID:      callID,
				Name:        part.FunctionCall.Name,
			}))
			// 函数调用项完成
			events = append(events, geminiMakeResponsesEvent(state, "response.output_item.done", &openAIResponsesStreamEvent{
				OutputIndex: state.OutputIndex,
				Item: &openAIResponsesOutput{
					Type:   "function_call",
					ID:     itemID,
					Status: "completed",
				},
			}))
			state.OutputIndex++
		default:
			// 文本或图片内容
			text := part.Text
			// 图片内容转换为 Markdown 图片格式
			if part.InlineData != nil && part.InlineData.Data != "" {
				text = fmt.Sprintf("![image](data:%s;base64,%s)", part.InlineData.MimeType, part.InlineData.Data)
			}
			if text == "" {
				continue
			}
			// 先关闭推理项（如果有）
			if state.CurrentReasonItemID != "" {
				events = append(events, geminiCloseReasoningItem(state)...)
			}
			// 如果没有消息项，创建新的
			if state.CurrentMessageItemID == "" {
				state.CurrentMessageItemID = fmt.Sprintf("item_message_%d", state.OutputIndex)
				events = append(events, geminiMakeResponsesEvent(state, "response.output_item.added", &openAIResponsesStreamEvent{
					OutputIndex: state.OutputIndex,
					Item: &openAIResponsesOutput{
						Type:   "message",
						ID:     state.CurrentMessageItemID,
						Role:   "assistant",
						Status: "in_progress",
					},
				}))
			}
			// 发送文本增量
			events = append(events, geminiMakeResponsesEvent(state, "response.output_text.delta", &openAIResponsesStreamEvent{
				OutputIndex: state.OutputIndex,
				ItemID:      state.CurrentMessageItemID,
				Delta:       text,
			}))
		}
	}
	// 处理完成原因
	if candidate.FinishReason != "" {
		// 如果有 grounding 文本（搜索结果），添加到消息中
		if grounding := buildGeminiGroundingText(candidate.GroundingMetadata); grounding != "" {
			if state.CurrentReasonItemID != "" {
				events = append(events, geminiCloseReasoningItem(state)...)
			}
			if state.CurrentMessageItemID == "" {
				state.CurrentMessageItemID = fmt.Sprintf("item_message_%d", state.OutputIndex)
				events = append(events, geminiMakeResponsesEvent(state, "response.output_item.added", &openAIResponsesStreamEvent{
					OutputIndex: state.OutputIndex,
					Item: &openAIResponsesOutput{
						Type:   "message",
						ID:     state.CurrentMessageItemID,
						Role:   "assistant",
						Status: "in_progress",
					},
				}))
			}
			events = append(events, geminiMakeResponsesEvent(state, "response.output_text.delta", &openAIResponsesStreamEvent{
				OutputIndex: state.OutputIndex,
				ItemID:      state.CurrentMessageItemID,
				Delta:       grounding,
			}))
		}
		// 关闭所有未关闭的输出项
		events = append(events, geminiCloseReasoningItem(state)...)
		events = append(events, geminiCloseMessageItem(state)...)
		// 确定最终状态
		status := "completed"
		var incomplete *openAIResponsesIncompleteDetail
		if candidate.FinishReason == "MAX_TOKENS" {
			status = "incomplete" // 达到最大 Token 数
			incomplete = &openAIResponsesIncompleteDetail{Reason: "max_output_tokens"}
		}
		events = append(events, geminiMakeResponsesCompletedEvent(state, status, incomplete))
		state.CompletedSent = true
	}
	return events
}

// geminiCloseReasoningItem 关闭当前推理输出项
// 发送推理摘要完成事件和输出项完成事件，然后递增输出索引
func geminiCloseReasoningItem(state *geminiResponsesStreamState) []openAIResponsesStreamEvent {
	if state.CurrentReasonItemID == "" {
		return nil
	}
	itemID := state.CurrentReasonItemID
	state.CurrentReasonItemID = ""
	events := []openAIResponsesStreamEvent{
		geminiMakeResponsesEvent(state, "response.reasoning_summary_text.done", &openAIResponsesStreamEvent{
			OutputIndex:  state.OutputIndex,
			SummaryIndex: 0,
			ItemID:       itemID,
		}),
		geminiMakeResponsesEvent(state, "response.output_item.done", &openAIResponsesStreamEvent{
			OutputIndex: state.OutputIndex,
			Item: &openAIResponsesOutput{
				Type:   "reasoning",
				ID:     itemID,
				Status: "completed",
			},
		}),
	}
	state.OutputIndex++
	return events
}

// geminiCloseMessageItem 关闭当前消息输出项
// 发送输出文本完成事件和输出项完成事件，然后递增输出索引
func geminiCloseMessageItem(state *geminiResponsesStreamState) []openAIResponsesStreamEvent {
	if state.CurrentMessageItemID == "" {
		return nil
	}
	itemID := state.CurrentMessageItemID
	state.CurrentMessageItemID = ""
	events := []openAIResponsesStreamEvent{
		geminiMakeResponsesEvent(state, "response.output_text.done", &openAIResponsesStreamEvent{
			OutputIndex: state.OutputIndex,
			ItemID:      itemID,
		}),
		geminiMakeResponsesEvent(state, "response.output_item.done", &openAIResponsesStreamEvent{
			OutputIndex: state.OutputIndex,
			Item: &openAIResponsesOutput{
				Type:   "message",
				ID:     itemID,
				Status: "completed",
			},
		}),
	}
	state.OutputIndex++
	return events
}

// geminiMakeResponsesCreatedEvent 创建 response.created 事件
func geminiMakeResponsesCreatedEvent(state *geminiResponsesStreamState) openAIResponsesStreamEvent {
	seq := state.SequenceNumber
	state.SequenceNumber++
	return openAIResponsesStreamEvent{
		Type:           "response.created",
		SequenceNumber: seq,
		Response: &openAIResponsesResponse{
			ID:     state.ResponseID,
			Object: "response",
			Model:  state.Model,
			Status: "in_progress",
			Output: []openAIResponsesOutput{},
		},
	}
}

// geminiMakeResponsesCompletedEvent 创建 response.completed/incomplete/failed 事件
// 包含最终的 Token 用量和不完整详情
func geminiMakeResponsesCompletedEvent(state *geminiResponsesStreamState, status string, incomplete *openAIResponsesIncompleteDetail) openAIResponsesStreamEvent {
	seq := state.SequenceNumber
	state.SequenceNumber++
	// 构建用量信息
	usage := &openAIResponsesUsage{
		InputTokens:  state.InputTokens,
		OutputTokens: state.OutputTokens,
		TotalTokens:  state.InputTokens + state.OutputTokens,
	}
	if state.CacheReadTokens > 0 {
		usage.InputTokensDetails = &openAIResponsesInputTokenDetails{CachedTokens: state.CacheReadTokens}
	}
	return openAIResponsesStreamEvent{
		Type:           geminiTerminalResponseEventType(status),
		SequenceNumber: seq,
		Response: &openAIResponsesResponse{
			ID:                state.ResponseID,
			Object:            "response",
			Model:             state.Model,
			Status:            status,
			Output:            []openAIResponsesOutput{},
			Usage:             usage,
			IncompleteDetails: incomplete,
		},
	}
}

// geminiMakeResponsesEvent 创建通用 Responses 流式事件
// 自动分配递增的序列号
func geminiMakeResponsesEvent(state *geminiResponsesStreamState, eventType string, template *openAIResponsesStreamEvent) openAIResponsesStreamEvent {
	seq := state.SequenceNumber
	state.SequenceNumber++
	evt := *template
	evt.Type = eventType
	evt.SequenceNumber = seq
	return evt
}

// geminiFinalizeResponsesEvents 生成 Responses 流的终止事件
// 如果流已创建但未完成，关闭所有未关闭的输出项并发送完成事件
func geminiFinalizeResponsesEvents(state *geminiResponsesStreamState) []openAIResponsesStreamEvent {
	if state.CompletedSent || !state.CreatedSent {
		return nil
	}
	var events []openAIResponsesStreamEvent
	events = append(events, geminiCloseReasoningItem(state)...)
	events = append(events, geminiCloseMessageItem(state)...)
	events = append(events, geminiMakeResponsesCompletedEvent(state, "completed", nil))
	state.CompletedSent = true
	return events
}

// geminiResponsesEventToChatChunks 将 Responses 流式事件转换为 Chat Completions 流式块
// 处理以下事件类型：
// - response.created → 发送角色信息
// - response.output_text.delta → 发送文本增量
// - response.output_item.added (function_call) → 发送工具调用开始
// - response.function_call_arguments.delta → 发送工具参数增量
// - response.reasoning_summary_text.delta → 发送推理内容增量
// - response.completed/incomplete/failed → 发送终止块
func geminiResponsesEventToChatChunks(evt *openAIResponsesStreamEvent, state *geminiChatStreamState) []openAIChatCompletionsChunk {
	switch evt.Type {
	case "response.created":
		// 提取响应 ID 和模型名称
		if evt.Response != nil {
			if evt.Response.ID != "" {
				state.ID = evt.Response.ID
			}
			if state.Model == "" {
				state.Model = evt.Response.Model
			}
		}
		// 首次创建时发送角色信息
		if state.SentRole {
			return nil
		}
		state.SentRole = true
		role := "assistant"
		return []openAIChatCompletionsChunk{geminiMakeChatDeltaChunk(state, openAIChatDelta{Role: role})}
	case "response.output_text.delta":
		// 文本增量
		if evt.Delta == "" {
			return nil
		}
		content := evt.Delta
		return []openAIChatCompletionsChunk{geminiMakeChatDeltaChunk(state, openAIChatDelta{Content: &content})}
	case "response.output_item.added":
		// 函数调用项添加：只处理 function_call 类型
		if evt.Item == nil || evt.Item.Type != "function_call" {
			return nil
		}
		state.SawToolCall = true
		idx := state.NextToolCallIndex
		state.OutputIndexToToolIndex[evt.OutputIndex] = idx
		state.NextToolCallIndex++
		return []openAIChatCompletionsChunk{geminiMakeChatDeltaChunk(state, openAIChatDelta{
			ToolCalls: []openAIChatToolCall{{
				Index: &idx,
				ID:    evt.Item.CallID,
				Type:  "function",
				Function: openAIChatFunctionCall{
					Name: evt.Item.Name,
				},
			}},
		})}
	case "response.function_call_arguments.delta":
		// 函数参数增量
		if evt.Delta == "" {
			return nil
		}
		idx, ok := state.OutputIndexToToolIndex[evt.OutputIndex]
		if !ok {
			return nil
		}
		return []openAIChatCompletionsChunk{geminiMakeChatDeltaChunk(state, openAIChatDelta{
			ToolCalls: []openAIChatToolCall{{
				Index: &idx,
				Function: openAIChatFunctionCall{
					Arguments: evt.Delta,
				},
			}},
		})}
	case "response.reasoning_summary_text.delta":
		// 推理内容增量
		if evt.Delta == "" {
			return nil
		}
		reasoning := evt.Delta
		return []openAIChatCompletionsChunk{geminiMakeChatDeltaChunk(state, openAIChatDelta{ReasoningContent: &reasoning})}
	case "response.completed":
		fallthrough
	case "response.incomplete":
		fallthrough
	case "response.failed":
		// 终止事件：发送完成块
		state.Finalized = true
		finishReason := "stop"
		if evt.Response != nil {
			// 提取用量信息
			if evt.Response.Usage != nil {
				u := evt.Response.Usage
				state.Usage = &openAIChatUsage{
					PromptTokens:     u.InputTokens,
					CompletionTokens: u.OutputTokens,
					TotalTokens:      u.TotalTokens,
				}
				if u.InputTokensDetails != nil && u.InputTokensDetails.CachedTokens > 0 {
					state.Usage.PromptTokensDetails = &openAIChatTokenDetail{CachedTokens: u.InputTokensDetails.CachedTokens}
				}
				if u.OutputTokenDetails != nil && u.OutputTokenDetails.ReasoningTokens > 0 {
					state.Usage.CompletionTokenInfo = &openAIChatTokenDetail{
						ReasoningTokens: u.OutputTokenDetails.ReasoningTokens,
					}
				}
			}
			// 确定完成原因
			if evt.Response.Status == "incomplete" {
				finishReason = "length"
			} else if state.SawToolCall {
				finishReason = "tool_calls"
			}
		}
		chunks := []openAIChatCompletionsChunk{geminiMakeChatFinishChunk(state, finishReason)}
		// 如果请求包含用量信息，追加一个单独的用量块
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
	default:
		return nil
	}
}

// geminiMakeChatDeltaChunk 创建 Chat Completions 增量块
func geminiMakeChatDeltaChunk(state *geminiChatStreamState, delta openAIChatDelta) openAIChatCompletionsChunk {
	return openAIChatCompletionsChunk{
		ID:      state.ID,
		Object:  "chat.completion.chunk",
		Created: state.Created,
		Model:   state.Model,
		Choices: []openAIChatChunkChoice{{Index: 0, Delta: delta}},
	}
}

// geminiMakeChatFinishChunk 创建 Chat Completions 终止块
// 包含空内容和完成原因
func geminiMakeChatFinishChunk(state *geminiChatStreamState, finishReason string) openAIChatCompletionsChunk {
	empty := ""
	return openAIChatCompletionsChunk{
		ID:      state.ID,
		Object:  "chat.completion.chunk",
		Created: state.Created,
		Model:   state.Model,
		Choices: []openAIChatChunkChoice{{Index: 0, Delta: openAIChatDelta{Content: &empty}, FinishReason: &finishReason}},
	}
}

// geminiFinalizeChatChunks 生成 Chat Completions 流的终止块
// 如果流未终止，发送完成块和用量信息
func geminiFinalizeChatChunks(state *geminiChatStreamState) []openAIChatCompletionsChunk {
	if state.Finalized {
		return nil
	}
	state.Finalized = true
	finishReason := "stop"
	if state.SawToolCall {
		finishReason = "tool_calls"
	}
	chunks := []openAIChatCompletionsChunk{geminiMakeChatFinishChunk(state, finishReason)}
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

// isGeminiTerminalResponsesEvent 判断是否为终止类型的 Responses 事件
func isGeminiTerminalResponsesEvent(eventType string) bool {
	return eventType == "response.completed" || eventType == "response.incomplete" || eventType == "response.failed"
}

// buildGeminiGroundingText 构建 grounding 文本（搜索结果引用）
// 包含搜索查询和引用来源链接
func buildGeminiGroundingText(grounding *GeminiGroundingMetadata) string {
	if grounding == nil {
		return ""
	}
	var b strings.Builder
	// 添加搜索查询
	if len(grounding.WebSearchQueries) > 0 {
		b.WriteString("\n\n---\nWeb search queries: ")
		b.WriteString(strings.Join(grounding.WebSearchQueries, ", "))
	}
	if len(grounding.GroundingChunks) > 0 {
		var links []string
		// 构建引用来源链接列表
		for i, chunk := range grounding.GroundingChunks {
			if chunk.Web == nil {
				continue
			}
			title := strings.TrimSpace(chunk.Web.Title)
			if title == "" {
				title = "Source" // 默认标题
			}
			uri := strings.TrimSpace(chunk.Web.URI)
			if uri == "" {
				uri = "#" // 默认链接
			}
			links = append(links, fmt.Sprintf("[%d] [%s](%s)", i+1, title, uri))
		}
		if len(links) > 0 {
			b.WriteString("\n\nSources:\n")
			b.WriteString(strings.Join(links, "\n"))
		}
	}
	return b.String()
}

// firstGeminiGroundingQuery 获取第一个搜索查询
func firstGeminiGroundingQuery(grounding *GeminiGroundingMetadata) string {
	if grounding == nil || len(grounding.WebSearchQueries) == 0 {
		return ""
	}
	return strings.TrimSpace(grounding.WebSearchQueries[0])
}

// geminiTerminalResponseEventType 根据状态返回对应的终止事件类型
func geminiTerminalResponseEventType(status string) string {
	switch status {
	case "incomplete":
		return "response.incomplete"
	case "failed":
		return "response.failed"
	default:
		return "response.completed"
	}
}

// ImageOutputTokens 获取图片输出的 Token 数
func (m *GeminiUsageMetadata) ImageOutputTokens() int {
	for _, d := range m.CandidatesTokensDetails {
		if d.Modality == "IMAGE" {
			return d.TokenCount
		}
	}
	return 0
}

// defaultGeminiSafetySettings 返回默认安全设置（全部关闭限制）
func defaultGeminiSafetySettings() []GeminiSafetySetting {
	return []GeminiSafetySetting{
		{Category: "HARM_CATEGORY_HARASSMENT", Threshold: "OFF"},
		{Category: "HARM_CATEGORY_HATE_SPEECH", Threshold: "OFF"},
		{Category: "HARM_CATEGORY_SEXUALLY_EXPLICIT", Threshold: "OFF"},
		{Category: "HARM_CATEGORY_DANGEROUS_CONTENT", Threshold: "OFF"},
		{Category: "HARM_CATEGORY_CIVIC_INTEGRITY", Threshold: "OFF"},
	}
}

// cloneGeminiHeader 深拷贝 HTTP 头
func cloneGeminiHeader(src http.Header) http.Header {
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

// firstNonEmptyGemini 返回第一个非空字符串
func firstNonEmptyGemini(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

// geminiParseArguments 解析函数参数字符串
// 空字符串返回空对象，有效 JSON 直接解析，无效 JSON 包装为 {"value": raw}
func geminiParseArguments(raw string) any {
	if strings.TrimSpace(raw) == "" {
		return map[string]any{}
	}
	var out any
	if err := json.Unmarshal([]byte(raw), &out); err == nil {
		return out
	}
	return map[string]any{"value": raw}
}

// geminiDataURIToInlineData 将 Data URI（data:mime;base64,xxx）转换为 Gemini 的内联数据
// 格式：data:<mediaType>;base64,<data>
func geminiDataURIToInlineData(dataURI string) *GeminiInlineData {
	if !strings.HasPrefix(dataURI, "data:") {
		return nil
	}
	rest := strings.TrimPrefix(dataURI, "data:")
	semicolonIdx := strings.Index(rest, ";")
	if semicolonIdx < 0 {
		return nil
	}
	mediaType := rest[:semicolonIdx]
	rest = rest[semicolonIdx+1:]
	if !strings.HasPrefix(rest, "base64,") {
		return nil
	}
	data := strings.TrimPrefix(rest, "base64,")
	// 验证 base64 数据有效性
	if _, err := base64.StdEncoding.DecodeString(data); err != nil {
		return nil
	}
	return &GeminiInlineData{MimeType: mediaType, Data: data}
}

// geminiParseAssistantContent 解析助手消息内容
// 支持纯字符串和内容数组格式
// 内容数组中的 thinking/reasoning 类型用 <thinking> 标签包裹
func geminiParseAssistantContent(raw json.RawMessage) (string, error) {
	if len(raw) == 0 {
		return "", nil
	}
	var s string
	if err := json.Unmarshal(raw, &s); err == nil {
		return s, nil
	}
	var parts []map[string]any
	if err := json.Unmarshal(raw, &parts); err != nil {
		return "", nil
	}
	var b strings.Builder
	for _, p := range parts {
		typ, _ := p["type"].(string)
		text, _ := p["text"].(string)
		switch typ {
		case "thinking", "reasoning":
			// 思维/推理内容用 <thinking> 标签包裹
			if text != "" {
				b.WriteString("<thinking>")
				b.WriteString(text)
				b.WriteString("</thinking>")
			}
		default:
			b.WriteString(text)
		}
	}
	return b.String(), nil
}

// geminiParseChatContent 解析 Chat 消息内容
// 支持纯字符串和内容数组格式，提取所有文本内容
func geminiParseChatContent(raw json.RawMessage) (string, error) {
	if len(raw) == 0 {
		return "", nil
	}
	var s string
	if err := json.Unmarshal(raw, &s); err == nil {
		return s, nil
	}
	var parts []openAIChatContentPart
	if err := json.Unmarshal(raw, &parts); err != nil {
		return "", err
	}
	var texts []string
	for _, part := range parts {
		if part.Type == "text" && part.Text != "" {
			texts = append(texts, part.Text)
		}
	}
	return strings.Join(texts, ""), nil
}

// geminiMarshalChatInputContent 将 Chat 消息内容转换为 Responses 输入格式
// 纯字符串保持不变，内容数组中的 text→input_text, image_url→input_image
func geminiMarshalChatInputContent(raw json.RawMessage) (json.RawMessage, error) {
	if len(raw) == 0 {
		return json.Marshal("") // 空内容返回空字符串
	}
	// 尝试解析为纯字符串，直接透传
	var s string
	if err := json.Unmarshal(raw, &s); err == nil {
		return json.Marshal(s)
	}
	// 解析为内容数组，逐项转换类型
	var parts []openAIChatContentPart
	if err := json.Unmarshal(raw, &parts); err != nil {
		return nil, err
	}
	out := make([]openAIResponsesContentPart, 0, len(parts))
	for _, part := range parts {
		switch part.Type {
		case "text":
			// 文本类型：text → input_text
			if part.Text != "" {
				out = append(out, openAIResponsesContentPart{Type: "input_text", Text: part.Text})
			}
		case "image_url":
			// 图片类型：image_url → input_image
			if part.ImageURL != nil && part.ImageURL.URL != "" {
				out = append(out, openAIResponsesContentPart{Type: "input_image", ImageURL: part.ImageURL.URL})
			}
		}
	}
	return json.Marshal(out)
}

// geminiConvertChatToolsToResponses 将 Chat Completions 的工具列表转换为 Responses 格式
// 兼容新版 tools（含 type 字段）和旧版 functions（直接函数定义）
func geminiConvertChatToolsToResponses(tools []openAIChatTool, functions []openAIChatFunction) []openAIResponsesTool {
	var out []openAIResponsesTool
	// 处理新版 tools 格式，只提取 function 类型的工具
	for _, t := range tools {
		if t.Type == "function" && t.Function != nil {
			out = append(out, openAIResponsesTool{
				Type:        "function",
				Name:        t.Function.Name,
				Description: t.Function.Description,
				Parameters:  t.Function.Parameters,
			})
		}
	}
	// 处理旧版 functions 格式，直接转换
	for _, f := range functions {
		out = append(out, openAIResponsesTool{
			Type:        "function",
			Name:        f.Name,
			Description: f.Description,
			Parameters:  f.Parameters,
		})
	}
	return out
}

// geminiConvertChatFunctionCallToToolChoice 将旧版 function_call 转换为 tool_choice 格式
// 支持两种输入格式：
// - 字符串（"auto"/"none"/"required"）→ 直接透传
// - 对象 {"name":"xxx"} → 转换为 {"type":"function","name":"xxx"}
func geminiConvertChatFunctionCallToToolChoice(raw json.RawMessage) (json.RawMessage, error) {
	// 尝试解析为字符串
	var s string
	if err := json.Unmarshal(raw, &s); err == nil {
		return json.Marshal(s)
	}
	// 解析为对象，提取函数名并转换为标准格式
	var obj struct {
		Name string `json:"name"`
	}
	if err := json.Unmarshal(raw, &obj); err != nil {
		return nil, err
	}
	return json.Marshal(map[string]any{"type": "function", "name": obj.Name})
}

// CleanJSONSchema 清理 JSON Schema，使其兼容 Gemini API
// 处理步骤：
// 1. 提取并内联 $defs/definitions 定义
// 2. 递归清理 Schema 结构（合并 allOf、处理 anyOf/oneOf、清理不允许的字段等）
func CleanJSONSchema(schema map[string]any) map[string]any {
	if schema == nil {
		return nil
	}
	// 第一步：提取定义并内联 $ref 引用
	flattenRefs(schema, extractDefs(schema))
	// 第二步：递归清理 Schema
	cleaned := cleanJSONSchemaRecursive(schema)
	result, ok := cleaned.(map[string]any)
	if !ok {
		return nil
	}
	return result
}

// extractDefs 提取并移除 Schema 中的 $defs 和 definitions 定义
// Gemini 不支持 $ref 引用，需要将定义提取出来后内联到引用处
func extractDefs(schema map[string]any) map[string]any {
	defs := make(map[string]any)
	// 提取 JSON Schema 的 $defs（新版）
	if d, ok := schema["$defs"].(map[string]any); ok {
		for k, v := range d {
			defs[k] = v
		}
		delete(schema, "$defs") // 移除，Gemini 不支持
	}
	// 提取 JSON Schema 的 definitions（旧版）
	if d, ok := schema["definitions"].(map[string]any); ok {
		for k, v := range d {
			defs[k] = v
		}
		delete(schema, "definitions") // 移除，Gemini 不支持
	}
	return defs
}

// flattenRefs 递归内联 $ref 引用
// 将 {"$ref": "#/$defs/MyDef"} 替换为 MyDef 的实际定义内容
// 处理顺序：先处理当前节点的 $ref，再递归处理子节点
func flattenRefs(schema map[string]any, defs map[string]any) {
	if len(defs) == 0 {
		return // 没有定义，无需内联
	}
	// 如果当前节点有 $ref，替换为实际定义
	if ref, ok := schema["$ref"].(string); ok {
		delete(schema, "$ref")
		// 从引用路径中提取定义名称（如 "#/$defs/MyDef" → "MyDef"）
		parts := strings.Split(ref, "/")
		refName := parts[len(parts)-1]
		// 查找定义并内联
		if defSchema, exists := defs[refName]; exists {
			if defMap, ok := defSchema.(map[string]any); ok {
				// 将定义中的字段复制到当前节点（不覆盖已有字段）
				for k, v := range defMap {
					if _, has := schema[k]; !has {
						schema[k] = deepCopy(v)
					}
				}
				// 内联后可能还有新的 $ref，递归处理
				flattenRefs(schema, defs)
			}
		}
	}
	// 递归处理所有子节点
	for _, v := range schema {
		if subMap, ok := v.(map[string]any); ok {
			flattenRefs(subMap, defs)
		} else if subArr, ok := v.([]any); ok {
			for _, item := range subArr {
				if itemMap, ok := item.(map[string]any); ok {
					flattenRefs(itemMap, defs)
				}
			}
		}
	}
}

// deepCopy 深拷贝一个值
// 支持 map[string]any、[]any 和基本类型的深拷贝
func deepCopy(src any) any {
	if src == nil {
		return nil
	}
	switch v := src.(type) {
	case map[string]any:
		dst := make(map[string]any, len(v))
		for k, val := range v {
			dst[k] = deepCopy(val) // 递归拷贝每个值
		}
		return dst
	case []any:
		dst := make([]any, len(v))
		for i, val := range v {
			dst[i] = deepCopy(val) // 递归拷贝每个元素
		}
		return dst
	default:
		return src // 基本类型直接返回
	}
}

// cleanJSONSchemaRecursive 递归清理 JSON Schema，使其兼容 Gemini API
// 主要处理：
// 1. 合并 allOf 结构
// 2. 递归清理 properties 和 items
// 3. 从 anyOf/oneOf 中选择最佳分支并合并
// 4. 清理不允许的字段，迁移约束到 description
// 5. 处理 nullable 类型、enum 值等
func cleanJSONSchemaRecursive(value any) any {
	schemaMap, ok := value.(map[string]any)
	if !ok {
		return value // 非 map 类型直接返回
	}
	// 合并 allOf 结构（Gemini 不支持 allOf）
	mergeAllOf(schemaMap)
	// 递归清理 properties 中的每个属性
	if props, ok := schemaMap["properties"].(map[string]any); ok {
		for _, v := range props {
			cleanJSONSchemaRecursive(v)
		}
	} else if items, ok := schemaMap["items"]; ok {
		// 处理数组类型的 items
		if itemsArr, ok := items.([]any); ok {
			// items 是数组（联合类型），选择最佳 Schema
			best := extractBestSchemaFromUnion(itemsArr)
			if best == nil {
				best = map[string]any{"type": "string"} // 默认字符串类型
			}
			schemaMap["items"] = cleanJSONSchemaRecursive(best)
		} else {
			// items 是单个对象，递归清理
			cleanJSONSchemaRecursive(items)
		}
	} else {
		// 其他情况：递归清理所有 map 和 array 子节点
		for _, v := range schemaMap {
			if _, isMap := v.(map[string]any); isMap {
				cleanJSONSchemaRecursive(v)
			} else if arr, isArr := v.([]any); isArr {
				for _, item := range arr {
					cleanJSONSchemaRecursive(item)
				}
			}
		}
	}

	// 处理 anyOf/oneOf 联合类型：选择最佳分支并合并到当前 Schema
	var unionArray []any
	typeStr, _ := schemaMap["type"].(string)
	// 只在类型为空或 object 时处理联合类型
	if typeStr == "" || typeStr == "object" {
		if anyOf, ok := schemaMap["anyOf"].([]any); ok {
			unionArray = anyOf
		} else if oneOf, ok := schemaMap["oneOf"].([]any); ok {
			unionArray = oneOf
		}
	}
	if len(unionArray) > 0 {
		// 从联合类型中选择最佳分支（优先选择有 properties 的 object 类型）
		if bestBranch := extractBestSchemaFromUnion(unionArray); bestBranch != nil {
			if bestMap, ok := bestBranch.(map[string]any); ok {
				for k, v := range bestMap {
					if k == "properties" {
						// 合并属性到当前 Schema
						targetProps, _ := schemaMap["properties"].(map[string]any)
						if targetProps == nil {
							targetProps = make(map[string]any)
							schemaMap["properties"] = targetProps
						}
						if sourceProps, ok := v.(map[string]any); ok {
							for pk, pv := range sourceProps {
								if _, exists := targetProps[pk]; !exists {
									targetProps[pk] = deepCopy(pv) // 不覆盖已有属性
								}
							}
						}
					} else if k == "required" {
						// 合并必填字段到当前 Schema
						targetReq, _ := schemaMap["required"].([]any)
						if sourceReq, ok := v.([]any); ok {
							for _, rv := range sourceReq {
								// 去重：检查是否已存在
								exists := false
								for _, tr := range targetReq {
									if tr == rv {
										exists = true
										break
									}
								}
								if !exists {
									targetReq = append(targetReq, rv)
								}
							}
							schemaMap["required"] = targetReq
						}
					} else if _, exists := schemaMap[k]; !exists {
						// 其他字段直接复制（不覆盖已有字段）
						schemaMap[k] = deepCopy(v)
					}
				}
			}
		}
	}

	// 判断当前节点是否像一个 JSON Schema（而非普通对象）
	looksLikeSchema := hasKey(schemaMap, "type") ||
		hasKey(schemaMap, "properties") ||
		hasKey(schemaMap, "items") ||
		hasKey(schemaMap, "enum") ||
		hasKey(schemaMap, "anyOf") ||
		hasKey(schemaMap, "oneOf") ||
		hasKey(schemaMap, "allOf")

	if looksLikeSchema {
		// 将不支持的约束条件迁移到 description 中
		migrateConstraints(schemaMap)
		// 只保留 Gemini 允许的字段，删除其余字段
		allowedFields := map[string]bool{
			"type":        true,
			"description": true,
			"properties":  true,
			"required":    true,
			"items":       true,
			"enum":        true,
			"title":       true,
		}
		for k := range schemaMap {
			if !allowedFields[k] {
				delete(schemaMap, k) // 删除不支持的字段
			}
		}

		// 如果是 object 类型但没有 properties，添加一个默认属性
		// Gemini 要求 object 类型必须有 properties
		if t, _ := schemaMap["type"].(string); t == "object" {
			hasProps := false
			if props, ok := schemaMap["properties"].(map[string]any); ok && len(props) > 0 {
				hasProps = true
			}
			if !hasProps {
				schemaMap["properties"] = map[string]any{
					"reason": map[string]any{
						"type":        "string",
						"description": "Reason for calling this tool",
					},
				}
				schemaMap["required"] = []any{"reason"}
			}
		}

		// 清理 required 列表：移除不存在的属性名
		if props, ok := schemaMap["properties"].(map[string]any); ok {
			if req, ok := schemaMap["required"].([]any); ok {
				var validReq []any
				for _, r := range req {
					if rStr, ok := r.(string); ok {
						if _, exists := props[rStr]; exists {
							validReq = append(validReq, r) // 只保留存在的属性
						}
					}
				}
				if len(validReq) > 0 {
					schemaMap["required"] = validReq
				} else {
					delete(schemaMap, "required") // 无有效必填字段则删除
				}
			}
		}

		// 处理 nullable 类型和 type 字段的规范化
		isNullable := false
		if typeVal, exists := schemaMap["type"]; exists {
			var selectedType string
			switch v := typeVal.(type) {
			case string:
				// type 为字符串：直接使用
				lower := strings.ToLower(v)
				if lower == "null" {
					isNullable = true
					selectedType = "string" // null 类型回退为 string
				} else {
					selectedType = lower
				}
			case []any:
				// type 为数组（如 ["string", "null"]）：选择第一个非 null 类型
				for _, t := range v {
					if ts, ok := t.(string); ok {
						lower := strings.ToLower(ts)
						if lower == "null" {
							isNullable = true // 标记为可空
						} else if selectedType == "" {
							selectedType = lower // 选择第一个非 null 类型
						}
					}
				}
				if selectedType == "" {
					selectedType = "string" // 全部为 null 时回退为 string
				}
			}
			schemaMap["type"] = selectedType
		} else if hasKey(schemaMap, "properties") {
			// 有 properties 但没有 type，默认为 object
			schemaMap["type"] = "object"
		} else {
			// 既没有 type 也没有 properties，默认为 object
			schemaMap["type"] = "object"
		}

		// 如果是可空类型，在 description 中标注
		if isNullable {
			desc, _ := schemaMap["description"].(string)
			if !strings.Contains(desc, "nullable") {
				if desc != "" {
					desc += " "
				}
				desc += "(nullable)"
				schemaMap["description"] = desc
			}
		}

		// 处理 enum 值：Gemini 只支持字符串类型的 enum
		if enumVals, ok := schemaMap["enum"].([]any); ok {
			hasNonString := false
			for i, val := range enumVals {
				if _, isStr := val.(string); !isStr {
					hasNonString = true
					if val == nil {
						enumVals[i] = "null" // null 转为字符串 "null"
					} else {
						enumVals[i] = fmt.Sprintf("%v", val) // 其他类型转为字符串
					}
				}
			}
			if hasNonString {
				schemaMap["type"] = "string" // 有非字符串值时强制 type 为 string
			}
		}
	}
	return schemaMap
}

// hasKey 检查 map 中是否存在指定键
func hasKey(m map[string]any, k string) bool {
	_, ok := m[k]
	return ok
}

// migrateConstraints 将 Gemini 不支持的约束条件迁移到 description 中
// Gemini 不支持 minLength、maxLength、pattern、minimum、maximum 等约束
// 将这些约束以文本形式附加到 description 中，以便模型理解
func migrateConstraints(m map[string]any) {
	constraints := []struct {
		key   string
		label string
	}{
		{"minLength", "minLen"},
		{"maxLength", "maxLen"},
		{"pattern", "pattern"},
		{"minimum", "min"},
		{"maximum", "max"},
		{"multipleOf", "multipleOf"},
		{"exclusiveMinimum", "exclMin"},
		{"exclusiveMaximum", "exclMax"},
		{"minItems", "minItems"},
		{"maxItems", "maxItems"},
		{"propertyNames", "propertyNames"},
		{"format", "format"},
	}
	var hints []string
	// 遍历所有约束，收集存在的约束值
	for _, c := range constraints {
		if val, ok := m[c.key]; ok && val != nil {
			hints = append(hints, fmt.Sprintf("%s: %v", c.label, val))
		}
	}
	// 将约束信息附加到 description 中
	if len(hints) > 0 {
		suffix := fmt.Sprintf(" [Constraint: %s]", strings.Join(hints, ", "))
		desc, _ := m["description"].(string)
		if !strings.Contains(desc, suffix) {
			m["description"] = desc + suffix
		}
	}
}

// mergeAllOf 合并 allOf 结构到当前 Schema
// Gemini 不支持 allOf，需要将所有子 Schema 的 properties、required 和其他字段合并
func mergeAllOf(m map[string]any) {
	allOf, ok := m["allOf"].([]any)
	if !ok {
		return // 没有 allOf，无需处理
	}
	delete(m, "allOf") // 移除 allOf 字段
	mergedProps := make(map[string]any)
	mergedReq := make(map[string]bool)
	otherFields := make(map[string]any)
	// 遍历 allOf 中的每个子 Schema，合并内容
	for _, sub := range allOf {
		if subMap, ok := sub.(map[string]any); ok {
			// 合并 properties
			if props, ok := subMap["properties"].(map[string]any); ok {
				for k, v := range props {
					mergedProps[k] = v
				}
			}
			// 合并 required（用 map 去重）
			if reqs, ok := subMap["required"].([]any); ok {
				for _, r := range reqs {
					if s, ok := r.(string); ok {
						mergedReq[s] = true
					}
				}
			}
			// 合并其他字段（不覆盖已有字段）
			for k, v := range subMap {
				if k != "properties" && k != "required" && k != "allOf" {
					if _, exists := otherFields[k]; !exists {
						otherFields[k] = v
					}
				}
			}
		}
	}
	// 将合并的其他字段写入当前 Schema
	for k, v := range otherFields {
		if _, exists := m[k]; !exists {
			m[k] = v
		}
	}
	// 将合并的 properties 写入当前 Schema
	if len(mergedProps) > 0 {
		existProps, _ := m["properties"].(map[string]any)
		if existProps == nil {
			existProps = make(map[string]any)
			m["properties"] = existProps
		}
		for k, v := range mergedProps {
			if _, exists := existProps[k]; !exists {
				existProps[k] = v // 不覆盖已有属性
			}
		}
	}
	// 将合并的 required 写入当前 Schema
	if len(mergedReq) > 0 {
		existReq, _ := m["required"].([]any)
		var validReqs []any
		// 保留已有的 required 项
		for _, r := range existReq {
			if s, ok := r.(string); ok {
				validReqs = append(validReqs, s)
				delete(mergedReq, s) // 去重
			}
		}
		for r := range mergedReq {
			validReqs = append(validReqs, r) // 添加新的 required 项
		}
		m["required"] = validReqs
	}
}

// extractBestSchemaFromUnion 从联合类型（anyOf/oneOf）中选择最佳 Schema
// 评分规则：有 properties/object > 有 items/array > 有非 null type > 其他
func extractBestSchemaFromUnion(unionArray []any) any {
	var bestOption any
	bestScore := -1
	for _, item := range unionArray {
		score := scoreSchemaOption(item)
		if score > bestScore {
			bestScore = score
			bestOption = item
		}
	}
	return bestOption
}

// scoreSchemaOption 为 Schema 选项评分
// 评分规则：
// - 有 properties 或 type=object：3 分（最优先，因为包含结构化信息）
// - 有 items 或 type=array：2 分（次优先，数组类型有结构）
// - 有非 null 的 type：1 分（基本类型）
// - 其他：0 分（null 或无类型信息）
func scoreSchemaOption(val any) int {
	m, ok := val.(map[string]any)
	if !ok {
		return 0
	}
	typeStr, _ := m["type"].(string)
	if hasKey(m, "properties") || typeStr == "object" {
		return 3 // 对象类型最优先
	}
	if hasKey(m, "items") || typeStr == "array" {
		return 2 // 数组类型次优先
	}
	if typeStr != "" && typeStr != "null" {
		return 1 // 基本类型
	}
	return 0 // null 或无类型
}

// DeepCleanUndefined 深度清理 JSON 中的 [undefined] 占位符
// 递归遍历 map 和 array，删除值为 "[undefined]" 的键
func DeepCleanUndefined(value any) {
	if value == nil {
		return
	}
	switch v := value.(type) {
	case map[string]any:
		for k, val := range v {
			if s, ok := val.(string); ok && s == "[undefined]" {
				delete(v, k) // 删除 undefined 占位符
				continue
			}
			DeepCleanUndefined(val) // 递归清理子节点
		}
	case []any:
		for _, val := range v {
			DeepCleanUndefined(val) // 递归清理数组元素
		}
	}
}
