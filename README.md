# sub2api lite

面向旧版 `sub2api` 的精简重构起点。

当前目标不是复制旧项目全部能力，而是优先保住最核心闭环：

- 多用户 + API Key
- 账号池 + Provider 插件化
- OpenAI 兼容协议转发
- 模型价格与余额扣费
- SQLite + GORM 单机部署
- `gopay` 充值支付接入

## 启动

```bash
cd server
go run ./cmd/sub2api \
  -admin-token your-admin-token \
  -public-base-url http://127.0.0.1:8080 \
  -gopay-url http://127.0.0.1:8081 \
  -gopay-pid 10000 \
  -gopay-key your-gopay-key
```

也可以用环境变量：

```bash
export SUB2API_ADMIN_TOKEN=your-admin-token
export SUB2API_PUBLIC_BASE_URL=http://127.0.0.1:8080
export SUB2API_GOPAY_URL=http://127.0.0.1:8081
export SUB2API_GOPAY_PID=10000
export SUB2API_GOPAY_KEY=your-gopay-key
cd server && go run ./cmd/sub2api
```

## 首次初始化

1. `POST /api/admin/bootstrap` 创建首个管理员用户
2. 用 `X-Admin-Token` 调管理接口创建普通用户
3. 创建 API Key
4. 创建上游账户
5. 创建模型价格
6. 用用户 API Key 调 `/v1/chat/completions` 等代理接口

## 关键文档

- [重构简化指南](docs/重构简化指南.md)
- [一期重构实施说明](docs/一期重构实施说明.md)
