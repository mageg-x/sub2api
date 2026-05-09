# Provider 规范与模板

本项目新增 provider 时，目标不是“能编译就行”，而是必须符合统一职责边界、统一目录结构、统一能力声明。

## 一、目录规范

每个 provider 目录默认只允许这两类主文件：

1. `<provider>.go`
2. `openai_compat.go`

说明：

- `<provider>.go` 负责 provider 基础能力：
  - `Name`
  - `Contract`
  - `NormalizeGateway`
  - `BuildUpstreamURL`
  - `ApplyRequest`
  - `ParseUsage`
  - `SupportsPath`
  - `ParseStreamUsage`
  - `ParseCacheUsage`
  - `AccountCapability`
  - OAuth 相关实现（如果支持 OAuth）
- `openai_compat.go` 负责协议转换：
  - OpenAI 请求 -> 上游原生请求
  - 上游原生响应 -> OpenAI 响应
  - 上游原生 SSE -> OpenAI SSE
  - Chat / Responses 双协议适配
  - provider 专属 schema/tool/stream 处理

允许的额外文件：

- `*_test.go`

不推荐再拆很多小文件，除非单文件超过 2500-3000 行且已经明显影响维护。

## 二、接口规范

每个 provider 必须实现：

1. `provider.Provider`
2. `provider.ContractProvider`
3. `provider.AccountCapabilityProvider`

按契约要求选择实现：

1. `provider.GatewayResponseAdapter`
2. `provider.StreamUsageParser`
3. `provider.CacheUsageParser`
4. `provider.OAuthStarter`
5. `provider.OAuthExchanger`
6. `provider.OAuthRefresher`

## 三、能力契约规范

每个 provider 必须实现：

```go
func (p *Provider) Contract() provider.Contract
```

用途：

- 明确这个 provider 应该承担哪些能力
- 在注册阶段自动校验
- 防止“半成品 provider”混入运行时

示例：

```go
func (p *Provider) Contract() provider.Contract {
	return provider.Contract{
		Name:                           p.Name(),
		RequiresGatewayResponseAdapter: true,
		RequiresStreamUsageParser:      true,
		RequiresCacheUsageParser:       true,
		SupportsOAuth:                  true,
	}
}
```

## 四、职责边界规范

### 1. core 层职责

`core` 只负责：

- 鉴权
- 选账号
- 发上游请求
- 计费
- 调用 provider 暴露的标准钩子

`core` 不负责：

- provider 私有协议细节
- OpenAI <-> 上游原生协议转换细节
- provider 私有 tool/schema/SSE 处理

### 2. provider 层职责

provider 必须自己完成：

- OpenAI 公开协议到上游原生协议的请求转换
- 上游原生响应回写为 OpenAI 公开协议
- provider 私有的 tool/schema/stream 适配

### 3. 公共协议目标

对外统一暴露 OpenAI 兼容协议。

provider 内部负责：

- `Claude` -> Anthropic 原生协议
- `Gemini` -> Gemini 原生协议
- 其他 provider 同理

## 五、中文注释规范

建议在两个主文件中至少保留这些中文分段注释：

- provider 基础能力
- OpenAI 请求适配
- 原生响应适配
- 流式事件适配
- tool/schema 适配
- OAuth 认证流程

不要写废话注释，要解释“为什么这一段存在”和“承担什么职责”。

## 六、新增 Provider 步骤

1. 复制 `server/internal/provider/template/`
2. 改目录名
3. 改 `Name()`
4. 改 `Contract()`
5. 实现基础请求能力
6. 实现 `openai_compat.go`
7. 在 `server/internal/app/app.go` 注册
8. 运行：

```bash
cd server && go test ./...
```

## 七、最小检查清单

新增 provider 前，至少确认：

- 是否实现 `Contract()`
- 是否声明清楚是否需要 `GatewayResponseAdapter`
- 是否支持流式 usage 解析
- 是否支持缓存 usage 解析
- 是否支持 OAuth
- 是否完成 OpenAI Chat / Responses 的请求转换
- 是否完成非流式响应转换
- 是否完成 SSE 转换
- 是否加了 `*_test.go`
