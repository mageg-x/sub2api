package template

import (
	"net/http"

	"sub2api/server/internal/provider"
)

type openAIResponsesRequest struct{}
type openAIResponsesResponse struct{}
type openAIResponsesStreamEvent struct{}
type openAIChatCompletionsRequest struct{}
type openAIChatCompletionsResponse struct{}
type openAIChatCompletionsChunk struct{}
type openAIChatMessage struct{}
type GeminiRequest struct{}
type GeminiResponse struct{}

// 编译期断言：如果这个模板声明需要协议回写，后续实现时必须补齐。
var _ provider.GatewayResponseAdapter = (*Provider)(nil)

// AdaptGatewayResponse 负责把上游非流式响应转换为公开协议响应。
func (p *Provider) AdaptGatewayResponse(req provider.GatewayRequest, resp *http.Response, body []byte) (*http.Response, []byte, error) {
	_ = req
	return resp, body, nil
}

// AdaptGatewayStream 负责把上游 SSE 流转换为公开协议 SSE。
func (p *Provider) AdaptGatewayStream(req provider.GatewayRequest, resp *http.Response) (*http.Response, error) {
	_ = req
	return resp, nil
}

// 这里保留与真实 provider 一致的分组结构：
// 1. OpenAI 公开协议类型
// 2. 上游原生协议类型
// 3. 请求转换
// 4. 响应转换
// 5. SSE 转换
// 6. 辅助函数
