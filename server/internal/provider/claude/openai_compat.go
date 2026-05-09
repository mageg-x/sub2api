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

type AnthropicRequest struct {
	Model        string                 `json:"model"`
	MaxTokens    int                    `json:"max_tokens"`
	System       json.RawMessage        `json:"system,omitempty"`
	Messages     []AnthropicMessage     `json:"messages"`
	Tools        []AnthropicTool        `json:"tools,omitempty"`
	Stream       bool                   `json:"stream,omitempty"`
	Temperature  *float64               `json:"temperature,omitempty"`
	TopP         *float64               `json:"top_p,omitempty"`
	StopSeqs     []string               `json:"stop_sequences,omitempty"`
	Thinking     *AnthropicThinking     `json:"thinking,omitempty"`
	ToolChoice   json.RawMessage        `json:"tool_choice,omitempty"`
	Metadata     json.RawMessage        `json:"metadata,omitempty"`
	OutputConfig *AnthropicOutputConfig `json:"output_config,omitempty"`
}

type AnthropicOutputConfig struct {
	Effort string `json:"effort,omitempty"`
}

type AnthropicThinking struct {
	Type         string `json:"type"`
	BudgetTokens int    `json:"budget_tokens,omitempty"`
}

type AnthropicMessage struct {
	Role    string          `json:"role"`
	Content json.RawMessage `json:"content"`
}

type AnthropicContentBlock struct {
	Type string `json:"type"`

	Text      string                `json:"text,omitempty"`
	Thinking  string                `json:"thinking,omitempty"`
	Signature string                `json:"signature,omitempty"`
	Source    *AnthropicImageSource `json:"source,omitempty"`

	ID    string          `json:"id,omitempty"`
	Name  string          `json:"name,omitempty"`
	Input json.RawMessage `json:"input,omitempty"`

	ToolUseID string          `json:"tool_use_id,omitempty"`
	Content   json.RawMessage `json:"content,omitempty"`
	IsError   bool            `json:"is_error,omitempty"`
}

type AnthropicImageSource struct {
	Type      string `json:"type"`
	MediaType string `json:"media_type"`
	Data      string `json:"data"`
}

type AnthropicTool struct {
	Type        string          `json:"type,omitempty"`
	Name        string          `json:"name"`
	Description string          `json:"description,omitempty"`
	InputSchema json.RawMessage `json:"input_schema"`
}

type AnthropicResponse struct {
	ID           string                  `json:"id"`
	Type         string                  `json:"type"`
	Role         string                  `json:"role"`
	Content      []AnthropicContentBlock `json:"content"`
	Model        string                  `json:"model"`
	StopReason   string                  `json:"stop_reason"`
	StopSequence *string                 `json:"stop_sequence,omitempty"`
	Usage        AnthropicUsage          `json:"usage"`
}

type AnthropicUsage struct {
	InputTokens              int `json:"input_tokens"`
	OutputTokens             int `json:"output_tokens"`
	CacheCreationInputTokens int `json:"cache_creation_input_tokens,omitempty"`
	CacheReadInputTokens     int `json:"cache_read_input_tokens,omitempty"`
}

type AnthropicStreamEvent struct {
	Type string `json:"type"`

	Message *AnthropicResponse `json:"message,omitempty"`

	Index        *int                   `json:"index,omitempty"`
	ContentBlock *AnthropicContentBlock `json:"content_block,omitempty"`

	Delta *AnthropicDelta `json:"delta,omitempty"`
	Usage *AnthropicUsage `json:"usage,omitempty"`
}

type AnthropicDelta struct {
	Type string `json:"type,omitempty"`

	Text         string  `json:"text,omitempty"`
	PartialJSON  string  `json:"partial_json,omitempty"`
	Thinking     string  `json:"thinking,omitempty"`
	Signature    string  `json:"signature,omitempty"`
	StopReason   string  `json:"stop_reason,omitempty"`
	StopSequence *string `json:"stop_sequence,omitempty"`
}

type openAIResponsesRequest struct {
	Model           string                    `json:"model"`
	Instructions    string                    `json:"instructions,omitempty"`
	Input           json.RawMessage           `json:"input"`
	MaxOutputTokens *int                      `json:"max_output_tokens,omitempty"`
	Temperature     *float64                  `json:"temperature,omitempty"`
	TopP            *float64                  `json:"top_p,omitempty"`
	Stream          bool                      `json:"stream,omitempty"`
	Tools           []openAIResponsesTool     `json:"tools,omitempty"`
	Include         []string                  `json:"include,omitempty"`
	Store           *bool                     `json:"store,omitempty"`
	Reasoning       *openAIResponsesReasoning `json:"reasoning,omitempty"`
	ToolChoice      json.RawMessage           `json:"tool_choice,omitempty"`
	ServiceTier     string                    `json:"service_tier,omitempty"`
}

type openAIResponsesReasoning struct {
	Effort  string `json:"effort"`
	Summary string `json:"summary,omitempty"`
}

type openAIResponsesInputItem struct {
	Type      string          `json:"type,omitempty"`
	Role      string          `json:"role,omitempty"`
	Content   json.RawMessage `json:"content,omitempty"`
	CallID    string          `json:"call_id,omitempty"`
	Name      string          `json:"name,omitempty"`
	Arguments string          `json:"arguments,omitempty"`
	ID        string          `json:"id,omitempty"`
	Output    string          `json:"output,omitempty"`
}

type openAIResponsesContentPart struct {
	Type     string `json:"type"`
	Text     string `json:"text,omitempty"`
	ImageURL string `json:"image_url,omitempty"`
}

type openAIResponsesTool struct {
	Type        string          `json:"type"`
	Name        string          `json:"name,omitempty"`
	Description string          `json:"description,omitempty"`
	Parameters  json.RawMessage `json:"parameters,omitempty"`
	Strict      *bool           `json:"strict,omitempty"`
}

type openAIResponsesResponse struct {
	ID                string                            `json:"id"`
	Object            string                            `json:"object"`
	Model             string                            `json:"model"`
	Status            string                            `json:"status"`
	Output            []openAIResponsesOutput           `json:"output"`
	Usage             *openAIResponsesUsage             `json:"usage,omitempty"`
	IncompleteDetails *openAIResponsesIncompleteDetails `json:"incomplete_details,omitempty"`
	Error             *openAIResponsesError             `json:"error,omitempty"`
}

type openAIResponsesError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type openAIResponsesIncompleteDetails struct {
	Reason string `json:"reason"`
}

type openAIResponsesOutput struct {
	Type      string                       `json:"type"`
	ID        string                       `json:"id,omitempty"`
	Role      string                       `json:"role,omitempty"`
	Content   []openAIResponsesContentPart `json:"content,omitempty"`
	Status    string                       `json:"status,omitempty"`
	Summary   []openAIResponsesSummary     `json:"summary,omitempty"`
	CallID    string                       `json:"call_id,omitempty"`
	Name      string                       `json:"name,omitempty"`
	Arguments string                       `json:"arguments,omitempty"`
	Action    *openAIWebSearchAction       `json:"action,omitempty"`
}

type openAIWebSearchAction struct {
	Type  string `json:"type,omitempty"`
	Query string `json:"query,omitempty"`
}

type openAIResponsesSummary struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

type openAIResponsesUsage struct {
	InputTokens         int                                `json:"input_tokens"`
	OutputTokens        int                                `json:"output_tokens"`
	TotalTokens         int                                `json:"total_tokens"`
	InputTokensDetails  *openAIResponsesInputTokenDetails  `json:"input_tokens_details,omitempty"`
	OutputTokensDetails *openAIResponsesOutputTokenDetails `json:"output_tokens_details,omitempty"`
}

type openAIResponsesInputTokenDetails struct {
	CachedTokens int `json:"cached_tokens,omitempty"`
}

type openAIResponsesOutputTokenDetails struct {
	ReasoningTokens int `json:"reasoning_tokens,omitempty"`
}

type openAIResponsesStreamEvent struct {
	Type           string                   `json:"type"`
	Response       *openAIResponsesResponse `json:"response,omitempty"`
	Item           *openAIResponsesOutput   `json:"item,omitempty"`
	OutputIndex    int                      `json:"output_index,omitempty"`
	ContentIndex   int                      `json:"content_index,omitempty"`
	Delta          string                   `json:"delta,omitempty"`
	Text           string                   `json:"text,omitempty"`
	ItemID         string                   `json:"item_id,omitempty"`
	CallID         string                   `json:"call_id,omitempty"`
	Name           string                   `json:"name,omitempty"`
	Arguments      string                   `json:"arguments,omitempty"`
	SummaryIndex   int                      `json:"summary_index,omitempty"`
	Code           string                   `json:"code,omitempty"`
	Param          string                   `json:"param,omitempty"`
	SequenceNumber int                      `json:"sequence_number,omitempty"`
}

type openAIChatCompletionsRequest struct {
	Model               string                   `json:"model"`
	Messages            []openAIChatMessage      `json:"messages"`
	Instructions        string                   `json:"instructions,omitempty"`
	MaxTokens           *int                     `json:"max_tokens,omitempty"`
	MaxCompletionTokens *int                     `json:"max_completion_tokens,omitempty"`
	Temperature         *float64                 `json:"temperature,omitempty"`
	TopP                *float64                 `json:"top_p,omitempty"`
	Stream              bool                     `json:"stream,omitempty"`
	StreamOptions       *openAIChatStreamOptions `json:"stream_options,omitempty"`
	Tools               []openAIChatTool         `json:"tools,omitempty"`
	ToolChoice          json.RawMessage          `json:"tool_choice,omitempty"`
	ReasoningEffort     string                   `json:"reasoning_effort,omitempty"`
	ServiceTier         string                   `json:"service_tier,omitempty"`
	Stop                json.RawMessage          `json:"stop,omitempty"`
	Functions           []openAIChatFunction     `json:"functions,omitempty"`
	FunctionCall        json.RawMessage          `json:"function_call,omitempty"`
}

type openAIChatStreamOptions struct {
	IncludeUsage bool `json:"include_usage,omitempty"`
}

type openAIChatMessage struct {
	Role             string                  `json:"role"`
	Content          json.RawMessage         `json:"content,omitempty"`
	ReasoningContent string                  `json:"reasoning_content,omitempty"`
	Name             string                  `json:"name,omitempty"`
	ToolCalls        []openAIChatToolCall    `json:"tool_calls,omitempty"`
	ToolCallID       string                  `json:"tool_call_id,omitempty"`
	FunctionCall     *openAIChatFunctionCall `json:"function_call,omitempty"`
}

type openAIChatContentPart struct {
	Type     string              `json:"type"`
	Text     string              `json:"text,omitempty"`
	ImageURL *openAIChatImageURL `json:"image_url,omitempty"`
}

type openAIChatImageURL struct {
	URL    string `json:"url"`
	Detail string `json:"detail,omitempty"`
}

type openAIChatTool struct {
	Type     string              `json:"type"`
	Function *openAIChatFunction `json:"function,omitempty"`
}

type openAIChatFunction struct {
	Name        string          `json:"name"`
	Description string          `json:"description,omitempty"`
	Parameters  json.RawMessage `json:"parameters,omitempty"`
	Strict      *bool           `json:"strict,omitempty"`
}

type openAIChatToolCall struct {
	Index    *int                   `json:"index,omitempty"`
	ID       string                 `json:"id,omitempty"`
	Type     string                 `json:"type,omitempty"`
	Function openAIChatFunctionCall `json:"function"`
}

type openAIChatFunctionCall struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

type openAIChatCompletionsResponse struct {
	ID                string             `json:"id"`
	Object            string             `json:"object"`
	Created           int64              `json:"created"`
	Model             string             `json:"model"`
	Choices           []openAIChatChoice `json:"choices"`
	Usage             *openAIChatUsage   `json:"usage,omitempty"`
	SystemFingerprint string             `json:"system_fingerprint,omitempty"`
	ServiceTier       string             `json:"service_tier,omitempty"`
}

type openAIChatChoice struct {
	Index        int               `json:"index"`
	Message      openAIChatMessage `json:"message"`
	FinishReason string            `json:"finish_reason"`
}

type openAIChatUsage struct {
	PromptTokens        int                    `json:"prompt_tokens"`
	CompletionTokens    int                    `json:"completion_tokens"`
	TotalTokens         int                    `json:"total_tokens"`
	PromptTokensDetails *openAIChatTokenDetail `json:"prompt_tokens_details,omitempty"`
}

type openAIChatTokenDetail struct {
	CachedTokens int `json:"cached_tokens,omitempty"`
}

type openAIChatCompletionsChunk struct {
	ID                string                  `json:"id"`
	Object            string                  `json:"object"`
	Created           int64                   `json:"created"`
	Model             string                  `json:"model"`
	Choices           []openAIChatChunkChoice `json:"choices"`
	Usage             *openAIChatUsage        `json:"usage,omitempty"`
	SystemFingerprint string                  `json:"system_fingerprint,omitempty"`
	ServiceTier       string                  `json:"service_tier,omitempty"`
}

type openAIChatChunkChoice struct {
	Index        int             `json:"index"`
	Delta        openAIChatDelta `json:"delta"`
	FinishReason *string         `json:"finish_reason"`
}

type openAIChatDelta struct {
	Role             string               `json:"role,omitempty"`
	Content          *string              `json:"content,omitempty"`
	ReasoningContent *string              `json:"reasoning_content,omitempty"`
	ToolCalls        []openAIChatToolCall `json:"tool_calls,omitempty"`
}

const openAIMinMaxOutputTokens = 128

type chatMessageContent struct {
	Text  *string
	Parts []openAIChatContentPart
}

func parseAssistantContent(raw json.RawMessage) (string, error) {
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
		thinking, _ := p["thinking"].(string)
		switch typ {
		case "thinking", "reasoning":
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

func parseChatContent(raw json.RawMessage) (string, error) {
	parsed, err := parseChatMessageContent(raw)
	if err != nil {
		return "", err
	}
	if parsed.Text != nil {
		return *parsed.Text, nil
	}
	return flattenChatContentParts(parsed.Parts), nil
}

func parseChatMessageContent(raw json.RawMessage) (chatMessageContent, error) {
	if len(raw) == 0 {
		return chatMessageContent{Text: stringPtr("")}, nil
	}
	var s string
	if err := json.Unmarshal(raw, &s); err == nil {
		return chatMessageContent{Text: &s}, nil
	}
	var parts []openAIChatContentPart
	if err := json.Unmarshal(raw, &parts); err == nil {
		return chatMessageContent{Parts: parts}, nil
	}
	return chatMessageContent{}, fmt.Errorf("parse content as string or parts array")
}

func marshalChatInputContent(content chatMessageContent) (json.RawMessage, error) {
	if content.Text != nil {
		return json.Marshal(*content.Text)
	}
	return json.Marshal(convertChatContentPartsToResponses(content.Parts))
}

func convertChatContentPartsToResponses(parts []openAIChatContentPart) []openAIResponsesContentPart {
	var responseParts []openAIResponsesContentPart
	for _, p := range parts {
		switch p.Type {
		case "text":
			if p.Text != "" {
				responseParts = append(responseParts, openAIResponsesContentPart{Type: "input_text", Text: p.Text})
			}
		case "image_url":
			if p.ImageURL != nil && p.ImageURL.URL != "" && !isEmptyBase64DataURI(p.ImageURL.URL) {
				responseParts = append(responseParts, openAIResponsesContentPart{Type: "input_image", ImageURL: p.ImageURL.URL})
			}
		}
	}
	return responseParts
}

func isEmptyBase64DataURI(raw string) bool {
	if !strings.HasPrefix(raw, "data:") {
		return false
	}
	rest := strings.TrimPrefix(raw, "data:")
	semicolonIdx := strings.Index(rest, ";")
	if semicolonIdx < 0 {
		return false
	}
	rest = rest[semicolonIdx+1:]
	if !strings.HasPrefix(rest, "base64,") {
		return false
	}
	return strings.TrimSpace(strings.TrimPrefix(rest, "base64,")) == ""
}

func flattenChatContentParts(parts []openAIChatContentPart) string {
	var textParts []string
	for _, p := range parts {
		if p.Type == "text" && p.Text != "" {
			textParts = append(textParts, p.Text)
		}
	}
	return strings.Join(textParts, "")
}

func stringPtr(s string) *string {
	return &s
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

func convertChatToolsToResponses(tools []openAIChatTool, functions []openAIChatFunction) []openAIResponsesTool {
	var out []openAIResponsesTool
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

func convertChatFunctionCallToToolChoice(raw json.RawMessage) (json.RawMessage, error) {
	var s string
	if err := json.Unmarshal(raw, &s); err == nil {
		return json.Marshal(s)
	}
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

func convertOpenAIRequest(publicPath string, body []byte) ([]byte, string, bool, bool, error) {
	switch normalizeOpenAIPath(publicPath) {
	case "/v1/chat/completions":
		var req openAIChatCompletionsRequest
		if err := json.Unmarshal(body, &req); err != nil {
			return nil, "", false, false, err
		}
		responsesReq, err := chatCompletionsToResponses(&req)
		if err != nil {
			return nil, "", false, false, err
		}
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
		return body, "", false, false, nil
	}
}

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

func detectChatIncludeUsage(req openAIChatCompletionsRequest) bool {
	return req.StreamOptions != nil && req.StreamOptions.IncludeUsage
}

func chatCompletionsToResponses(req *openAIChatCompletionsRequest) (*openAIResponsesRequest, error) {
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
		Include:      []string{"reasoning.encrypted_content"},
		ServiceTier:  req.ServiceTier,
	}
	storeFalse := false
	out.Store = &storeFalse
	maxTokens := 0
	if req.MaxTokens != nil {
		maxTokens = *req.MaxTokens
	}
	if req.MaxCompletionTokens != nil {
		maxTokens = *req.MaxCompletionTokens
	}
	if maxTokens > 0 {
		if maxTokens < openAIMinMaxOutputTokens {
			maxTokens = openAIMinMaxOutputTokens
		}
		out.MaxOutputTokens = &maxTokens
	}
	if req.ReasoningEffort != "" {
		out.Reasoning = &openAIResponsesReasoning{Effort: req.ReasoningEffort, Summary: "auto"}
	}
	if len(req.Tools) > 0 || len(req.Functions) > 0 {
		out.Tools = convertChatToolsToResponses(req.Tools, req.Functions)
	}
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
		return chatUserToResponses(m)
	}
}

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

func chatAssistantToResponses(m openAIChatMessage) ([]openAIResponsesInputItem, error) {
	var items []openAIResponsesInputItem
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
	for _, tc := range m.ToolCalls {
		args := tc.Function.Arguments
		if args == "" {
			args = "{}"
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

func chatToolToResponses(m openAIChatMessage) ([]openAIResponsesInputItem, error) {
	output, err := parseChatContent(m.Content)
	if err != nil {
		return nil, err
	}
	if output == "" {
		output = "(empty)"
	}
	return []openAIResponsesInputItem{{
		Type:   "function_call_output",
		CallID: m.ToolCallID,
		Output: output,
	}}, nil
}

func chatFunctionToResponses(m openAIChatMessage) ([]openAIResponsesInputItem, error) {
	output, err := parseChatContent(m.Content)
	if err != nil {
		return nil, err
	}
	if output == "" {
		output = "(empty)"
	}
	return []openAIResponsesInputItem{{
		Type:   "function_call_output",
		CallID: m.Name,
		Output: output,
	}}, nil
}

func responsesToAnthropicRequest(req *openAIResponsesRequest) (*AnthropicRequest, error) {
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
	if len(system) > 0 {
		out.System = system
	}
	if req.MaxOutputTokens != nil && *req.MaxOutputTokens > 0 {
		out.MaxTokens = *req.MaxOutputTokens
	}
	if out.MaxTokens == 0 {
		out.MaxTokens = 8192
	}
	if len(req.Tools) > 0 {
		out.Tools = convertResponsesToAnthropicTools(req.Tools)
	}
	if len(req.ToolChoice) > 0 {
		tc, err := convertResponsesToAnthropicToolChoice(req.ToolChoice)
		if err != nil {
			return nil, err
		}
		out.ToolChoice = tc
	}
	if req.Reasoning != nil && req.Reasoning.Effort != "" {
		effort := mapResponsesEffortToAnthropic(req.Reasoning.Effort)
		out.OutputConfig = &AnthropicOutputConfig{Effort: effort}
		if effort != "low" {
			out.Thinking = &AnthropicThinking{
				Type:         "enabled",
				BudgetTokens: defaultThinkingBudget(effort),
			}
		}
	}
	return out, nil
}

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

func mapResponsesEffortToAnthropic(effort string) string {
	if effort == "xhigh" {
		return "max"
	}
	return effort
}

func convertResponsesInputToAnthropic(inputRaw json.RawMessage) (json.RawMessage, []AnthropicMessage, error) {
	var inputStr string
	if err := json.Unmarshal(inputRaw, &inputStr); err == nil {
		content, _ := json.Marshal(inputStr)
		return nil, []AnthropicMessage{{Role: "user", Content: content}}, nil
	}
	var items []openAIResponsesInputItem
	if err := json.Unmarshal(inputRaw, &items); err != nil {
		return nil, nil, err
	}
	var system json.RawMessage
	var messages []AnthropicMessage
	for _, item := range items {
		switch {
		case item.Role == "system":
			text := extractTextFromContent(item.Content)
			if text != "" {
				system, _ = json.Marshal(text)
			}
		case item.Type == "function_call":
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
			content, err := convertResponsesUserToAnthropicContent(item.Content)
			if err != nil {
				return nil, nil, err
			}
			messages = append(messages, AnthropicMessage{Role: "user", Content: content})
		case item.Role == "assistant":
			content, err := convertResponsesAssistantToAnthropicContent(item.Content)
			if err != nil {
				return nil, nil, err
			}
			messages = append(messages, AnthropicMessage{Role: "assistant", Content: content})
		default:
			if item.Content != nil {
				messages = append(messages, AnthropicMessage{Role: "user", Content: item.Content})
			}
		}
	}
	return system, mergeConsecutiveMessages(messages), nil
}

func extractTextFromContent(raw json.RawMessage) string {
	if len(raw) == 0 {
		return ""
	}
	var s string
	if err := json.Unmarshal(raw, &s); err == nil {
		return s
	}
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

func convertResponsesUserToAnthropicContent(raw json.RawMessage) (json.RawMessage, error) {
	if len(raw) == 0 {
		return json.Marshal("")
	}
	var s string
	if err := json.Unmarshal(raw, &s); err == nil {
		return json.Marshal(s)
	}
	var parts []openAIResponsesContentPart
	if err := json.Unmarshal(raw, &parts); err != nil {
		return raw, nil
	}
	var blocks []AnthropicContentBlock
	for _, p := range parts {
		switch p.Type {
		case "input_text", "text":
			if p.Text != "" {
				blocks = append(blocks, AnthropicContentBlock{Type: "text", Text: p.Text})
			}
		case "input_image":
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

func convertResponsesAssistantToAnthropicContent(raw json.RawMessage) (json.RawMessage, error) {
	if len(raw) == 0 {
		return json.Marshal([]AnthropicContentBlock{{Type: "text", Text: ""}})
	}
	var s string
	if err := json.Unmarshal(raw, &s); err == nil {
		return json.Marshal([]AnthropicContentBlock{{Type: "text", Text: s}})
	}
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
		last := &merged[len(merged)-1]
		lastBlocks := parseContentBlocks(last.Content)
		newBlocks := parseContentBlocks(msg.Content)
		combined := append(lastBlocks, newBlocks...)
		last.Content, _ = json.Marshal(combined)
	}
	return merged
}

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

func convertResponsesToAnthropicTools(tools []openAIResponsesTool) []AnthropicTool {
	var out []AnthropicTool
	for _, t := range tools {
		switch t.Type {
		case "web_search", "google_search", "web_search_20250305":
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

func normalizeAnthropicInputSchema(schema json.RawMessage) json.RawMessage {
	if len(schema) == 0 || string(schema) == "null" {
		return json.RawMessage(`{"type":"object","properties":{}}`)
	}
	return schema
}

func convertResponsesToAnthropicToolChoice(raw json.RawMessage) (json.RawMessage, error) {
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

func fromResponsesCallIDToAnthropic(id string) string {
	if after, ok := strings.CutPrefix(id, "fc_"); ok {
		if strings.HasPrefix(after, "toolu_") || strings.HasPrefix(after, "call_") {
			return after
		}
	}
	if !strings.HasPrefix(id, "toolu_") && !strings.HasPrefix(id, "call_") {
		return "toolu_" + id
	}
	return id
}

func dataURIToAnthropicImageSource(dataURI string) *AnthropicImageSource {
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
	return &AnthropicImageSource{Type: "base64", MediaType: mediaType, Data: data}
}

func (p *Provider) AdaptGatewayResponse(req provider.GatewayRequest, resp *http.Response, body []byte) (*http.Response, []byte, error) {
	switch normalizeOpenAIPath(req.PublicPath) {
	case "/v1/chat/completions":
		return p.adaptClaudeToOpenAIChatResponse(req, resp, body)
	case "/v1/responses", "/backend-api/codex/responses":
		return p.adaptClaudeToOpenAIResponsesResponse(resp, body)
	default:
		return resp, body, nil
	}
}

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

func (p *Provider) adaptClaudeToOpenAIResponsesResponse(resp *http.Response, body []byte) (*http.Response, []byte, error) {
	if resp.StatusCode >= 400 {
		return resp, body, nil
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

func (p *Provider) adaptClaudeToOpenAIChatResponse(req provider.GatewayRequest, resp *http.Response, body []byte) (*http.Response, []byte, error) {
	resp, converted, err := p.adaptClaudeToOpenAIResponsesResponse(resp, body)
	if err != nil || resp.StatusCode >= 400 {
		return resp, converted, err
	}
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

func (p *Provider) wrapClaudeToOpenAIResponsesStream(resp *http.Response) *http.Response {
	if resp.StatusCode >= 400 || !isSSEHeader(resp.Header) {
		return resp
	}
	resp.Header = cloneHTTPHeader(resp.Header)
	resp.Header.Set("Content-Type", "text/event-stream; charset=utf-8")
	resp.Body = newClaudeResponsesStreamAdapter(resp.Body)
	resp.ContentLength = -1
	return resp
}

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

func anthropicToResponsesResponse(resp *AnthropicResponse) *openAIResponsesResponse {
	id := strings.TrimSpace(resp.ID)
	if id == "" {
		id = generateResponsesID()
	}
	out := &openAIResponsesResponse{
		ID:     id,
		Object: "response",
		Model:  resp.Model,
		Status: anthropicStopReasonToResponsesStatus(resp.StopReason),
	}
	var outputs []openAIResponsesOutput
	var msgParts []openAIResponsesContentPart
	for _, block := range resp.Content {
		switch block.Type {
		case "thinking":
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
			if block.Text != "" {
				msgParts = append(msgParts, openAIResponsesContentPart{Type: "output_text", Text: block.Text})
			}
		case "tool_use":
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
	if len(msgParts) > 0 {
		outputs = append(outputs, openAIResponsesOutput{
			Type:    "message",
			ID:      generateItemID(),
			Role:    "assistant",
			Content: msgParts,
			Status:  "completed",
		})
	}
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
	out.Usage = &openAIResponsesUsage{
		InputTokens:  resp.Usage.InputTokens,
		OutputTokens: resp.Usage.OutputTokens,
		TotalTokens:  resp.Usage.InputTokens + resp.Usage.OutputTokens,
	}
	if resp.Usage.CacheReadInputTokens > 0 {
		out.Usage.InputTokensDetails = &openAIResponsesInputTokenDetails{CachedTokens: resp.Usage.CacheReadInputTokens}
	}
	if out.Status == "incomplete" {
		out.IncompleteDetails = &openAIResponsesIncompleteDetails{Reason: "max_output_tokens"}
	}
	return out
}

func anthropicStopReasonToResponsesStatus(stopReason string) string {
	if stopReason == "max_tokens" {
		return "incomplete"
	}
	return "completed"
}

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
			for _, s := range item.Summary {
				if s.Type == "summary_text" && s.Text != "" {
					reasoningText += s.Text
				}
			}
		}
	}
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
	out.Choices = []openAIChatChoice{{
		Index:        0,
		Message:      msg,
		FinishReason: responsesStatusToChatFinishReason(resp.Status, resp.IncompleteDetails, len(toolCalls) > 0),
	}}
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

func isSSEHeader(header http.Header) bool {
	return strings.Contains(strings.ToLower(header.Get("Content-Type")), "event-stream")
}

type anthropicResponsesStreamState struct {
	ResponseID           string
	Model                string
	SequenceNumber       int
	CreatedSent          bool
	CompletedSent        bool
	OutputIndex          int
	CurrentItemID        string
	CurrentItemType      string
	CurrentCallID        string
	CurrentName          string
	ContentIndex         int
	InputTokens          int
	OutputTokens         int
	CacheReadInputTokens int
}

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

type responsesChatStreamState struct {
	ID                     string
	Model                  string
	Created                int64
	SentRole               bool
	SawToolCall            bool
	Finalized              bool
	NextToolCallIndex      int
	OutputIndexToToolIndex map[int]int
	IncludeUsage           bool
	Usage                  *openAIChatUsage
}

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

type sseTransformReadCloser struct {
	src        io.ReadCloser
	scanner    *bufio.Scanner
	transform  func(string, any) ([]string, error)
	state      any
	pending    bytes.Buffer
	eofHandled bool
	closeErr   error
}

func newSSETransformReadCloser(src io.ReadCloser, transform func(string, any) ([]string, error), state any) io.ReadCloser {
	scanner := bufio.NewScanner(src)
	scanner.Buffer(make([]byte, 0, 4096), 1024*1024)
	return &sseTransformReadCloser{src: src, scanner: scanner, transform: transform, state: state}
}

func (r *sseTransformReadCloser) Read(p []byte) (int, error) {
	for r.pending.Len() == 0 {
		if !r.scanner.Scan() {
			if err := r.scanner.Err(); err != nil {
				return 0, err
			}
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

func (r *sseTransformReadCloser) appendEOF() {
	switch state := r.state.(type) {
	case *anthropicResponsesStreamState:
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

func (r *sseTransformReadCloser) Close() error {
	if r.src != nil {
		r.closeErr = r.src.Close()
	}
	return r.closeErr
}

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

func anthToResHandleMessageStart(evt *AnthropicStreamEvent, state *anthropicResponsesStreamState) []openAIResponsesStreamEvent {
	if evt.Message != nil {
		state.ResponseID = evt.Message.ID
		if state.Model == "" {
			state.Model = evt.Message.Model
		}
		state.InputTokens = evt.Message.Usage.InputTokens
		state.OutputTokens = evt.Message.Usage.OutputTokens
		state.CacheReadInputTokens = evt.Message.Usage.CacheReadInputTokens
	}
	if state.CreatedSent {
		return nil
	}
	state.CreatedSent = true
	return []openAIResponsesStreamEvent{makeResponsesCreatedEvent(state)}
}

func anthToResHandleContentBlockStart(evt *AnthropicStreamEvent, state *anthropicResponsesStreamState) []openAIResponsesStreamEvent {
	if evt.ContentBlock == nil {
		return nil
	}
	var events []openAIResponsesStreamEvent
	switch evt.ContentBlock.Type {
	case "thinking":
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

func anthToResHandleContentBlockDelta(evt *AnthropicStreamEvent, state *anthropicResponsesStreamState) []openAIResponsesStreamEvent {
	if evt.Delta == nil {
		return nil
	}
	switch evt.Delta.Type {
	case "text_delta":
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

func anthToResHandleContentBlockStop(state *anthropicResponsesStreamState) []openAIResponsesStreamEvent {
	switch state.CurrentItemType {
	case "reasoning":
		events := []openAIResponsesStreamEvent{
			makeResponsesEvent(state, "response.reasoning_summary_text.done", &openAIResponsesStreamEvent{
				OutputIndex:  state.OutputIndex,
				SummaryIndex: 0,
				ItemID:       state.CurrentItemID,
			}),
		}
		return append(events, closeCurrentResponsesItem(state)...)
	case "function_call":
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

func anthToResHandleMessageDelta(evt *AnthropicStreamEvent, state *anthropicResponsesStreamState) []openAIResponsesStreamEvent {
	if evt.Usage != nil {
		state.OutputTokens = evt.Usage.OutputTokens
		if evt.Usage.CacheReadInputTokens > 0 {
			state.CacheReadInputTokens = evt.Usage.CacheReadInputTokens
		}
	}
	return nil
}

func anthToResHandleMessageStop(state *anthropicResponsesStreamState) []openAIResponsesStreamEvent {
	if state.CompletedSent {
		return nil
	}
	events := closeCurrentResponsesItem(state)
	events = append(events, makeResponsesCompletedEvent(state, "completed", nil))
	state.CompletedSent = true
	return events
}

func closeCurrentResponsesItem(state *anthropicResponsesStreamState) []openAIResponsesStreamEvent {
	if state.CurrentItemType == "" {
		return nil
	}
	itemType := state.CurrentItemType
	itemID := state.CurrentItemID
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

func makeResponsesCompletedEvent(state *anthropicResponsesStreamState, status string, incomplete *openAIResponsesIncompleteDetails) openAIResponsesStreamEvent {
	seq := state.SequenceNumber
	state.SequenceNumber++
	usage := &openAIResponsesUsage{
		InputTokens:  state.InputTokens,
		OutputTokens: state.OutputTokens,
		TotalTokens:  state.InputTokens + state.OutputTokens,
	}
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

func makeResponsesEvent(state *anthropicResponsesStreamState, eventType string, template *openAIResponsesStreamEvent) openAIResponsesStreamEvent {
	seq := state.SequenceNumber
	state.SequenceNumber++
	evt := *template
	evt.Type = eventType
	evt.SequenceNumber = seq
	return evt
}

func finalizeAnthropicResponsesStream(state *anthropicResponsesStreamState) []openAIResponsesStreamEvent {
	if !state.CreatedSent || state.CompletedSent {
		return nil
	}
	events := closeCurrentResponsesItem(state)
	events = append(events, makeResponsesCompletedEvent(state, "completed", nil))
	state.CompletedSent = true
	return events
}

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
		return nil
	}
	state.SentRole = true
	role := "assistant"
	return []openAIChatCompletionsChunk{makeChatDeltaChunk(state, openAIChatDelta{Role: role})}
}

func resToChatHandleTextDelta(evt *openAIResponsesStreamEvent, state *responsesChatStreamState) []openAIChatCompletionsChunk {
	if evt.Delta == "" {
		return nil
	}
	content := evt.Delta
	return []openAIChatCompletionsChunk{makeChatDeltaChunk(state, openAIChatDelta{Content: &content})}
}

func resToChatHandleOutputItemAdded(evt *openAIResponsesStreamEvent, state *responsesChatStreamState) []openAIChatCompletionsChunk {
	if evt.Item == nil || evt.Item.Type != "function_call" {
		return nil
	}
	state.SawToolCall = true
	idx := state.NextToolCallIndex
	state.OutputIndexToToolIndex[evt.OutputIndex] = idx
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

func resToChatHandleReasoningDelta(evt *openAIResponsesStreamEvent, state *responsesChatStreamState) []openAIChatCompletionsChunk {
	if evt.Delta == "" {
		return nil
	}
	reasoning := evt.Delta
	return []openAIChatCompletionsChunk{makeChatDeltaChunk(state, openAIChatDelta{ReasoningContent: &reasoning})}
}

func resToChatHandleCompleted(evt *openAIResponsesStreamEvent, state *responsesChatStreamState) []openAIChatCompletionsChunk {
	state.Finalized = true
	finishReason := "stop"
	if evt.Response != nil {
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
		if evt.Response.Status == "incomplete" && evt.Response.IncompleteDetails != nil && evt.Response.IncompleteDetails.Reason == "max_output_tokens" {
			finishReason = "length"
		} else if state.SawToolCall {
			finishReason = "tool_calls"
		}
	} else if state.SawToolCall {
		finishReason = "tool_calls"
	}
	chunks := []openAIChatCompletionsChunk{makeChatFinishChunk(state, finishReason)}
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

func isTerminalResponsesEvent(eventType string) bool {
	switch eventType {
	case "response.completed", "response.done", "response.incomplete", "response.failed":
		return true
	default:
		return false
	}
}

func generateResponsesID() string {
	b := make([]byte, 12)
	_, _ = rand.Read(b)
	return "resp_" + hex.EncodeToString(b)
}

func generateItemID() string {
	b := make([]byte, 12)
	_, _ = rand.Read(b)
	return "item_" + hex.EncodeToString(b)
}

func generateChatCmplID() string {
	b := make([]byte, 12)
	_, _ = rand.Read(b)
	return "chatcmpl-" + hex.EncodeToString(b)
}

func toResponsesCallID(id string) string {
	if strings.HasPrefix(id, "fc_") {
		return id
	}
	return "fc_" + id
}
