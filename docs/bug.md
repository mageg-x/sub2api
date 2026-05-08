## 🔴 严重 Bug

### 1. 余额扣费竞态条件 — 可导致余额为负

[core.go:1592](file:///d:/work/sub2api/server/internal/service/core.go#L1592) 中 `Proxy` 先检查 `auth.User.Balance <= 0`，但余额扣减在 [core.go:2710](file:///d:/work/sub2api/server/internal/service/core.go#L2710) 的 `recordUsage` 中异步执行。多个并发请求都能通过余额检查，导致实际扣费后余额为负数。

```go
// Proxy 中只做了一次读取时的检查
if auth.User.Balance <= 0 {
    return nil, nil, fmt.Errorf("insufficient balance")
}
// recordUsage 中直接扣减，没有 WHERE balance >= cost 的保护
c.db.Model(&model.User{}).Where("id = ?", auth.User.ID).
    Update("balance", gorm.Expr("balance - ?", cost))
```

**修复建议**：扣费时加条件 `Where("balance >= ?", cost)`，检查 `RowsAffected` 是否为 1。

---

### 2. `BootstrapAdmin` 未检查 `AllowBootstrap` 配置 — 任意创建管理员

[http.go:261](file:///d:/work/sub2api/server/internal/handler/http.go#L261) 的 `bootstrapAdmin` 处理器完全没有检查 `cfg.AllowBootstrap` 配置。配置中定义了该字段 [config.go:82](file:///d:/work/sub2api/server/internal/config/config.go#L82)，但从未使用。只要系统中没有管理员，任何人都可以调用此接口创建管理员。

```go
// config 中定义了但从未检查
AllowBootstrap: env("SUB2API_ALLOW_BOOTSTRAP", "false") == "true",
```

**修复建议**：在 `bootstrapAdmin` handler 中加入 `if !h.cfg.AllowBootstrap { return error }`。

---

### 3. `RedeemCoupon` 忽略了优惠券 `Kind` 字段 — "percent" 类型优惠券行为错误

[core.go:996](file:///d:/work/sub2api/server/internal/service/core.go#L996) 中，无论优惠券的 `Kind` 是 `"balance"` 还是 `"percent"`，都直接把 `coupon.Amount` 加到用户余额上。`"percent"` 类型的优惠券应该按百分比折扣计算，而不是直接加固定金额。

```go
// 不管 Kind 是什么，都直接加 Amount
return tx.Model(&model.User{}).Where("id = ?", userID).Updates(map[string]any{
    "balance": gorm.Expr("balance + ?", coupon.Amount),
    ...
}).Error
```

**修复建议**：根据 `coupon.Kind` 分别处理 `"balance"` 和 `"percent"` 逻辑。

---

### 4. `recordUsage` 错误被静默忽略 — 可导致免费使用

[core.go:1657-1660](file:///d:/work/sub2api/server/internal/service/core.go#L1657) 中，`Proxy` 调用 `recordUsage` 时用 `_` 忽略了错误。如果余额扣减失败（比如 DB 暂时不可用），用户仍然获得了 AI 服务，但没被扣费。

```go
_ = c.recordUsage(auth, &account, modelName, path, inTokens, outTokens, cacheCreateTokens, cacheReadTokens)
```

**修复建议**：至少记录错误日志，或考虑在扣费失败时返回错误。

---

## 🟠 中等 Bug

### 5. `flushUsageLogsBatch` 聚合失败导致数据不一致

[core.go:2780-2786](file:///d:/work/sub2api/server/internal/service/core.go#L2780) 中，如果 `CreateInBatches` 成功但 `upsertUsageAggregates` 失败，usage_log 已写入数据库但聚合表（minute/hour/day）缺失数据，且不会重试聚合。

```go
if err := c.db.CreateInBatches(items, 100).Error; err != nil {
    // 失败时回队列
    ...
    return
}
// 聚合失败只记录日志，数据丢失
if err := c.upsertUsageAggregates(items); err != nil {
    c.recordError("usage.aggregate", "upsert usage aggregates failed", err.Error())
}
```

**修复建议**：聚合失败时也应将 items 放回队列重试，或在聚合表中使用幂等 upsert 保证最终一致。

---

### 6. Token 签名和加密共用同一个 AESKey — 密码学反模式

[core.go:3630](file:///d:/work/sub2api/server/internal/service/core.go#L3630) 中，`signUserToken` 使用 `c.cfg.AESKey` 作为 HMAC-SHA256 的密钥来签名 token。同一个密钥既用于 AES-GCM 加密（凭证加密），又用于 HMAC 签名（token 认证）。违反了密钥分离原则。

```go
mac := hmac.New(sha256.New, c.cfg.AESKey)
mac.Write([]byte(rawPayload))
```

**修复建议**：从主密钥派生出两个子密钥（如 `HKDF(AESKey, "encryption")` 和 `HKDF(AESKey, "signing")`），分别用于不同用途。

---

### 7. `gopay` 支付金额单位注释与实际不一致

[payment.go:9](file:///d:/work/sub2api/server/internal/payment/payment.go#L9) 中 `Amount int64` 的注释写的是 `金额（分）`，但 [gopay.go:276](file:///d:/work/sub2api/server/internal/payment/gopay/gopay.go#L276) 的 `amountToYuan` 除以 10000，说明内部单位实际是 **万分之元（CNY_1E4）**，不是"分"。

```go
// payment.go 注释
Amount     int64  // 订单金额（分）  ← 错误，实际是万分之元

// gopay.go 实际转换
func amountToYuan(amount int64) string {
    return strconv.FormatFloat(float64(amount)/10000, 'f', 2, 64)
}
```

**修复建议**：修正注释为 `金额（万分之CNY）`，与 `ModelPrice.Currency` 的 `CNY_1E4` 保持一致。

---

### 8. `gopay` 签名函数 HMAC 密钥同时出现在消息中 — 安全隐患

[gopay.go:259-267](file:///d:/work/sub2api/server/internal/payment/gopay/gopay.go#L259) 中，签名时先把 `key=` 追加到消息末尾，又用同一个 key 作为 HMAC 密钥。这是冗余且不规范的：

```go
b.WriteString("key=")
b.WriteString(key)           // key 出现在消息中
mac := hmac.New(sha256.New, []byte(key))  // 同一个 key 又作为 HMAC 密钥
mac.Write([]byte(b.String()))
```

**说明**：如果这是 gopay 上游 API 规定的签名算法则无法修改，但应加注释说明。

---

## 🟡 轻微问题

### 9. `Gemini` Provider 的 `BuildUpstreamURL` 包含死代码

[gemini.go:30-33](file:///d:/work/sub2api/server/internal/provider/gemini/gemini.go#L30) 中，`BuildUpstreamURL` 处理了 `/v1internal:` 路径，但 `SupportsPath` 不包含该路径（[gemini.go:76](file:///d:/work/sub2api/server/internal/provider/gemini/gemini.go#L76)），且 `detectRoute` 将 `/v1internal:` 路由到 `antigravity` provider，所以这段代码永远不会执行。

---

### 10. `cacheItems` 淘汰策略是 O(n) 扫描

[core.go:3310-3320](file:///d:/work/sub2api/server/internal/service/core.go#L3310) 中，当缓存满 1000 条时，遍历所有条目找最旧的删除。高频场景下性能不佳。

**修复建议**：改用 LRU 缓存（如 `github.com/hashicorp/golang-lru`）。

---

### 11. `registerIPs` 在 DDoS 场景下可能内存膨胀

[core.go:660-672](file:///d:/work/sub2api/server/internal/service/core.go#L660) 中，`registerIPs` map 只在 `Register` 调用时清理 60 秒前的条目。大量不同 IP 的恶意注册请求会导致 map 膨胀。

**修复建议**：限制 map 最大容量，或改用布隆过滤器/滑动窗口计数器。

---

### 12. `hashPassword` 返回空 salt — 向后兼容隐患

[core.go:3446](file:///d:/work/sub2api/server/internal/service/core.go#L3446) 中，`hashPassword` 返回空字符串作为 salt，因为 bcrypt 自带 salt。但 `verifyPassword` 中仍有依赖 salt 的旧验证路径（`$hmac$` 和裸 SHA256）。如果未来有人误用这些路径，空 salt 会导致验证失败或安全漏洞。

```go
func hashPassword(password string) (string, string, error) {
    // ...
    return "", "$bcrypt$" + string(hash), nil  // salt 始终为空
}
```

---

### 13. `v1` 版本旧 Token 签名验证使用 SHA256 而非 HMAC — 安全性弱

[core.go:3660-3665](file:///d:/work/sub2api/server/internal/service/core.go#L3660) 中，旧版 token（无 `v2.` 前缀）使用 `sha256.Sum256(payload + hexKey)` 验证，这是普通的 hash 而非 HMAC，存在长度扩展攻击的理论风险。

```go
// 旧版签名 - 非 HMAC
mac := sha256.Sum256([]byte(payloadStr + "." + hex.EncodeToString(c.cfg.AESKey)))
```

**修复建议**：迁移所有旧 token 到 v2 格式，然后移除旧验证路径。

---

### 14. `gopayNotify` 不验证 HTTP 方法

[http.go:930](file:///d:/work/sub2api/server/internal/handler/http.go#L930) 中，`gopayNotify` 注册在 `POST` 路由上（Go 1.22 路由语法），所以实际上已经限制了方法。但如果上游支付平台用 GET 回调，会被路由匹配失败。这个实际上不是 bug，只是需要确认 gopay 的回调方式。

---

## 总结

| 严重程度 | 数量 | 关键问题 |
|---------|------|---------|
| 🔴 严重 | 4 | 余额竞态负数、Bootstrap 未鉴权、优惠券 Kind 未处理、recordUsage 静默失败 |
| 🟠 中等 | 4 | 聚合数据不一致、密钥复用、金额单位注释错误、签名冗余 |
| 🟡 轻微 | 6 | 死代码、缓存淘汰性能、内存膨胀、空 salt、旧 token 安全性等 |

