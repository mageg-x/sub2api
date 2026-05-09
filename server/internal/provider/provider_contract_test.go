package provider_test

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"

	"sub2api/server/internal/config"
	"sub2api/server/internal/model"
	"sub2api/server/internal/provider"
	claudepkg "sub2api/server/internal/provider/claude"
	geminipkg "sub2api/server/internal/provider/gemini"
	openaipkg "sub2api/server/internal/provider/openai"
)

func TestOpenAIProviderNormalizeGatewayForOAuthResponses(t *testing.T) {
	p := openaipkg.New(config.Config{})
	req := provider.GatewayRequest{
		Method:       http.MethodPost,
		PublicPath:   "/v1/responses",
		InternalPath: "/v1/responses",
		Body:         []byte(`{"model":"gpt-4.1","stream":true}`),
	}

	got, err := p.NormalizeGateway(model.Account{AuthType: "oauth"}, nil, req)
	if err != nil {
		t.Fatalf("NormalizeGateway() error = %v", err)
	}
	if got.Provider != "openai" {
		t.Fatalf("Provider = %q, want openai", got.Provider)
	}
	if got.InternalPath != "/backend-api/codex/responses" {
		t.Fatalf("InternalPath = %q, want /backend-api/codex/responses", got.InternalPath)
	}
	if got.Model != "gpt-4.1" {
		t.Fatalf("Model = %q, want gpt-4.1", got.Model)
	}
	if !got.Stream {
		t.Fatal("Stream = false, want true")
	}
}

func TestClaudeProviderNormalizeGatewayForOpenAICompatPaths(t *testing.T) {
	p := claudepkg.New(config.Config{})
	body := []byte(`{
		"model":"claude-sonnet-4",
		"stream":true,
		"stream_options":{"include_usage":true},
		"messages":[{"role":"user","content":"hello"}]
	}`)
	req := provider.GatewayRequest{
		Method:       http.MethodPost,
		PublicPath:   "/v1/chat/completions",
		InternalPath: "/v1/chat/completions",
		Body:         body,
	}

	got, err := p.NormalizeGateway(model.Account{}, nil, req)
	if err != nil {
		t.Fatalf("NormalizeGateway() error = %v", err)
	}
	if got.Provider != "claude" {
		t.Fatalf("Provider = %q, want claude", got.Provider)
	}
	if got.InternalPath != "/v1/messages" {
		t.Fatalf("InternalPath = %q, want /v1/messages", got.InternalPath)
	}
	if got.UpstreamMethod != http.MethodPost {
		t.Fatalf("UpstreamMethod = %q, want POST", got.UpstreamMethod)
	}
	if !got.Stream || !got.UpstreamStream {
		t.Fatalf("Stream flags = (%v,%v), want (true,true)", got.Stream, got.UpstreamStream)
	}
	if !got.IncludeUsage {
		t.Fatal("IncludeUsage = false, want true")
	}
	if got.Model != "claude-sonnet-4" {
		t.Fatalf("Model = %q, want claude-sonnet-4", got.Model)
	}

	var upstream struct {
		Model    string `json:"model"`
		Stream   bool   `json:"stream"`
		Messages []any  `json:"messages"`
	}
	if err := json.Unmarshal(got.Body, &upstream); err != nil {
		t.Fatalf("unmarshal upstream body: %v", err)
	}
	if upstream.Model != "claude-sonnet-4" {
		t.Fatalf("upstream model = %q, want claude-sonnet-4", upstream.Model)
	}
	if !upstream.Stream {
		t.Fatal("upstream stream = false, want true")
	}
	if len(upstream.Messages) == 0 {
		t.Fatal("upstream messages empty")
	}
}

func TestClaudeProviderAdaptGatewayResponseForResponses(t *testing.T) {
	p := claudepkg.New(config.Config{})
	resp := &http.Response{
		StatusCode: http.StatusOK,
		Header:     http.Header{"Content-Type": []string{"application/json"}},
	}
	body := []byte(`{
		"id":"msg_123",
		"type":"message",
		"role":"assistant",
		"model":"claude-sonnet-4",
		"content":[{"type":"text","text":"hello from claude"}],
		"stop_reason":"end_turn",
		"usage":{"input_tokens":11,"output_tokens":7}
	}`)

	gotResp, gotBody, err := p.AdaptGatewayResponse(provider.GatewayRequest{
		PublicPath: "/v1/responses",
	}, resp, body)
	if err != nil {
		t.Fatalf("AdaptGatewayResponse() error = %v", err)
	}
	if ct := gotResp.Header.Get("Content-Type"); !strings.Contains(ct, "application/json") {
		t.Fatalf("Content-Type = %q, want application/json", ct)
	}

	var out struct {
		Object string `json:"object"`
		Model  string `json:"model"`
		Status string `json:"status"`
		Output []struct {
			Type    string `json:"type"`
			Role    string `json:"role"`
			Status  string `json:"status"`
			Content []struct {
				Type string `json:"type"`
				Text string `json:"text"`
			} `json:"content"`
		} `json:"output"`
		Usage struct {
			InputTokens  int `json:"input_tokens"`
			OutputTokens int `json:"output_tokens"`
			TotalTokens  int `json:"total_tokens"`
		} `json:"usage"`
	}
	if err := json.Unmarshal(gotBody, &out); err != nil {
		t.Fatalf("unmarshal adapted body: %v", err)
	}
	if out.Object != "response" {
		t.Fatalf("Object = %q, want response", out.Object)
	}
	if out.Model != "claude-sonnet-4" {
		t.Fatalf("Model = %q, want claude-sonnet-4", out.Model)
	}
	if out.Status != "completed" {
		t.Fatalf("Status = %q, want completed", out.Status)
	}
	if len(out.Output) == 0 || len(out.Output[len(out.Output)-1].Content) == 0 {
		t.Fatal("message output missing")
	}
	if out.Output[len(out.Output)-1].Content[0].Text != "hello from claude" {
		t.Fatalf("message text = %q, want hello from claude", out.Output[len(out.Output)-1].Content[0].Text)
	}
	if out.Usage.InputTokens != 11 || out.Usage.OutputTokens != 7 || out.Usage.TotalTokens != 18 {
		t.Fatalf("usage = %+v, want input=11 output=7 total=18", out.Usage)
	}
}

func TestGeminiProviderNormalizeGatewayForResponses(t *testing.T) {
	p := geminipkg.New(config.Config{})
	body := []byte(`{
		"model":"gemini-2.5-pro",
		"stream":true,
		"input":"hello"
	}`)
	req := provider.GatewayRequest{
		Method:       http.MethodPost,
		PublicPath:   "/backend-api/codex/responses",
		InternalPath: "/backend-api/codex/responses",
		Body:         body,
	}

	got, err := p.NormalizeGateway(model.Account{}, nil, req)
	if err != nil {
		t.Fatalf("NormalizeGateway() error = %v", err)
	}
	if got.Provider != "gemini" {
		t.Fatalf("Provider = %q, want gemini", got.Provider)
	}
	if got.InternalPath != "/v1beta/models/gemini-2.5-pro:streamGenerateContent" {
		t.Fatalf("InternalPath = %q", got.InternalPath)
	}
	if !got.Stream || !got.UpstreamStream {
		t.Fatalf("Stream flags = (%v,%v), want (true,true)", got.Stream, got.UpstreamStream)
	}
	if got.Model != "gemini-2.5-pro" {
		t.Fatalf("Model = %q, want gemini-2.5-pro", got.Model)
	}

	var upstream struct {
		Contents []struct {
			Role string `json:"role"`
		} `json:"contents"`
	}
	if err := json.Unmarshal(got.Body, &upstream); err != nil {
		t.Fatalf("unmarshal upstream body: %v", err)
	}
	if len(upstream.Contents) == 0 || upstream.Contents[0].Role != "user" {
		t.Fatalf("unexpected upstream contents: %+v", upstream.Contents)
	}
}

func TestGeminiProviderAdaptGatewayResponseForChatCompletions(t *testing.T) {
	p := geminipkg.New(config.Config{})
	resp := &http.Response{
		StatusCode: http.StatusOK,
		Header:     http.Header{"Content-Type": []string{"application/json"}},
	}
	body := []byte(`{
		"responseId":"resp_gem_1",
		"modelVersion":"gemini-2.5-pro",
		"candidates":[
			{
				"finishReason":"STOP",
				"content":{
					"role":"model",
					"parts":[{"text":"hello from gemini"}]
				}
			}
		],
		"usageMetadata":{
			"promptTokenCount":12,
			"candidatesTokenCount":8,
			"totalTokenCount":20
		}
	}`)

	gotResp, gotBody, err := p.AdaptGatewayResponse(provider.GatewayRequest{
		PublicPath: "/v1/chat/completions",
		Model:      "gemini-2.5-pro",
	}, resp, body)
	if err != nil {
		t.Fatalf("AdaptGatewayResponse() error = %v", err)
	}
	if ct := gotResp.Header.Get("Content-Type"); !strings.Contains(ct, "application/json") {
		t.Fatalf("Content-Type = %q, want application/json", ct)
	}

	var out struct {
		Object  string `json:"object"`
		Model   string `json:"model"`
		Choices []struct {
			FinishReason string `json:"finish_reason"`
			Message      struct {
				Role    string `json:"role"`
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
		Usage struct {
			PromptTokens     int `json:"prompt_tokens"`
			CompletionTokens int `json:"completion_tokens"`
			TotalTokens      int `json:"total_tokens"`
		} `json:"usage"`
	}
	if err := json.Unmarshal(gotBody, &out); err != nil {
		t.Fatalf("unmarshal adapted body: %v", err)
	}
	if out.Object != "chat.completion" {
		t.Fatalf("Object = %q, want chat.completion", out.Object)
	}
	if out.Model != "gemini-2.5-pro" {
		t.Fatalf("Model = %q, want gemini-2.5-pro", out.Model)
	}
	if len(out.Choices) != 1 {
		t.Fatalf("len(Choices) = %d, want 1", len(out.Choices))
	}
	if out.Choices[0].Message.Role != "assistant" {
		t.Fatalf("role = %q, want assistant", out.Choices[0].Message.Role)
	}
	if out.Choices[0].Message.Content != "hello from gemini" {
		t.Fatalf("content = %q, want hello from gemini", out.Choices[0].Message.Content)
	}
	if out.Choices[0].FinishReason != "stop" {
		t.Fatalf("finish_reason = %q, want stop", out.Choices[0].FinishReason)
	}
	if out.Usage.PromptTokens != 12 || out.Usage.CompletionTokens != 8 || out.Usage.TotalTokens != 20 {
		t.Fatalf("usage = %+v, want prompt=12 completion=8 total=20", out.Usage)
	}
}

func TestClaudeAndGeminiAdaptGatewayStreamWrapSSE(t *testing.T) {
	claudeProvider := claudepkg.New(config.Config{})
	geminiProvider := geminipkg.New(config.Config{})

	claudeResp, err := claudeProvider.AdaptGatewayStream(provider.GatewayRequest{
		PublicPath: "/v1/responses",
	}, &http.Response{
		StatusCode: http.StatusOK,
		Header:     http.Header{"Content-Type": []string{"text/event-stream"}},
		Body:       io.NopCloser(strings.NewReader("")),
	})
	if err != nil {
		t.Fatalf("Claude AdaptGatewayStream() error = %v", err)
	}
	if ct := claudeResp.Header.Get("Content-Type"); !strings.Contains(strings.ToLower(ct), "event-stream") {
		t.Fatalf("claude stream content-type = %q", ct)
	}
	if claudeResp.ContentLength != -1 {
		t.Fatalf("claude ContentLength = %d, want -1", claudeResp.ContentLength)
	}

	geminiResp, err := geminiProvider.AdaptGatewayStream(provider.GatewayRequest{
		PublicPath: "/v1/chat/completions",
		Model:      "gemini-2.5-pro",
	}, &http.Response{
		StatusCode: http.StatusOK,
		Header:     http.Header{"Content-Type": []string{"text/event-stream"}},
		Body:       io.NopCloser(strings.NewReader("")),
	})
	if err != nil {
		t.Fatalf("Gemini AdaptGatewayStream() error = %v", err)
	}
	if ct := geminiResp.Header.Get("Content-Type"); !strings.Contains(strings.ToLower(ct), "event-stream") {
		t.Fatalf("gemini stream content-type = %q", ct)
	}
	if geminiResp.ContentLength != -1 {
		t.Fatalf("gemini ContentLength = %d, want -1", geminiResp.ContentLength)
	}
}

