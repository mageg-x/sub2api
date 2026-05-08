## 🔴 严重 Bug

### 1. 命令行参数完全失效 — [config.go:131-149](file:///d:/work/sub2api/server/internal/config/config.go#L131-L149)

```go
func stringValue(name, value string) string {
    return *flag.String(name, value, "")  // ← 在 flag.Parse() 之前就解引用了！
}
```

`stringValue`/`intValue`/`uint64Value` 在 `flag.Parse()` 调用之前就解引用了 flag 指针，此时指针指向的仍然是默认值。虽然后面调用了 `flag.Parse()`，但返回的已经是默认值的副本，命令行参数永远不会生效。

**修复方式**：先注册所有 flag，调用 `flag.Parse()`，再读取指针值。

---

### 2. UpdateAccount 的 `credentials_json` 字段名与数据库列名不匹配 — [core.go:1135](file:///d:/work/sub2api/server/internal/service/core.go#L1135)

```go
if in.CredentialsJSON != nil {
    if creds := strings.TrimSpace(*in.CredentialsJSON); creds != "" {
        updates["credentials_json"] = creds  // ← 数据库列名是 credentials_encrypted！
    }
}
```

Account 模型中字段是 `CredentialsEncrypted`，GORM 生成的列名为 `credentials_encrypted`。但更新 map 中用了 `credentials_json`，GORM 的 `Updates(map)` 会将 key 当作列名，导致该更新**静默失败**（SQLite 不会报错，只是不匹配任何列）。

**修复方式**：将 `"credentials_json"` 改为 `"credentials_encrypted"`，同时还需要对明文凭证进行加密后再存储。

---

### 3. 部分退款时错误地将订单状态设为 REFUNDED — [core.go:1528-1531](file:///d:/work/sub2api/server/internal/service/core.go#L1528-L1531)

```go
if err := tx.Model(&order).Updates(map[string]any{
    "status":          "REFUNDED",  // ← 即使只是部分退款也直接设为 REFUNDED
    "refunded_amount": gorm.Expr("refunded_amount + ?", amount),
}).Error; err != nil {
```

如果一笔订单做了部分退款，状态就被设为 `REFUNDED`。而退款入口有前置检查 `if order.Status != "PAID"`，导致**第二次部分退款永远无法执行**。

**修复方式**：只有当 `refunded_amount + amount >= credited_amount` 时才将状态设为 `REFUNDED`，否则保持 `PAID`。

---

### 4. 注册速率限制完全失效 — [http.go:218](file:///d:/work/sub2api/server/internal/handler/http.go#L218)

```go
req.ClientIP = r.RemoteAddr  // ← 包含端口号，如 "192.168.1.1:54321"
```

`r.RemoteAddr` 格式为 `ip:port`，每次连接端口不同，所以速率限制的 key 每次都不一样，**限流形同虚设**。而同文件中已有 `clientIP(r)` 辅助函数能正确提取 IP。

**修复方式**：改为 `req.ClientIP = clientIP(r)`。

---

## 🟡 中等 Bug

### 5. sync.Map 内存泄漏 — `accountLoads` 和 `refreshMu` 永不清理

[core.go:2653-2665](file:///d:/work/sub2api/server/internal/service/core.go#L2653-L2665)

删除账号后，`accountLoads`（`*atomic.Int64`）和 `refreshMu`（`*sync.Mutex`）中的条目永远不会被清理。长时间运行后，如果频繁增删账号，会造成缓慢的内存泄漏。

---

### 6. flushPendingKeys / flushPendingUsers 逐条写数据库 — 性能问题

[core.go:4019-4030](file:///d:/work/sub2api/server/internal/service/core.go#L4019-L4030)

```go
for keyID, usedAt := range pending {
    _ = c.db.Model(&model.APIKey{}).Where("id = ?", keyID).Updates(...)
}
```

每个 key/user 都是单独的 SQL UPDATE。在高并发场景下，如果有数百个活跃 key，每 5 秒就会产生数百次数据库写入。SQLite 的 `MaxOpenConns=2` 会让这成为瓶颈。

**建议**：使用 `CASE WHEN` 批量更新，或按 ID 分组后批量执行。

---

### 7. registerIPs 清理时机不够及时 — [core.go:613-621](file:///d:/work/sub2api/server/internal/service/core.go#L613-L621)

```go
if len(c.registerIPs) > 10000 {
    // 只在超过 10000 条时清理
}
```

如果所有条目都在冷却期内（1 分钟），清理不会删除任何条目，map 会持续增长。在遭受注册攻击时，map 可能增长到非常大。

---

### 8. calculateCost 费率计算截断而非四舍五入 — [core.go:3243](file:///d:/work/sub2api/server/internal/service/core.go#L3243)

```go
return int64(ratePercent) * base / 100, nil  // ← 截断除法，总是少收
```

整数除法截断会导致每次计费略微少收。虽然单次差异极小，但长期累积可能有影响。

---

## 🟢 轻微问题

### 9. Gemini Provider 的 SupportsPath 与 detectRoute 不一致

[gemini.go:56](file:///d:/work/sub2api/server/internal/provider/gemini/gemini.go#L56) 中 `SupportsPath` 包含 `/v1internal:`，但 [core.go:3320](file:///d:/work/sub2api/server/internal/service/core.go#L3320) 的 `detectRoute` 将 `/v1internal:` 路由到 `antigravity`。Gemini provider 永远不会收到 `/v1internal:` 请求，`SupportsPath` 中的判断是死代码。

---

### 10. logout handler 静默忽略 JSON 解析错误 — [http.go:197](file:///d:/work/sub2api/server/internal/handler/http.go#L197)

```go
var req service.RefreshTokenInput
_ = decodeJSON(r, &req)  // ← 错误被忽略
```

如果请求体格式错误，`req.RefreshToken` 为空，`LogoutUser` 对空 token 直接返回 nil。用户以为登出成功，实际上 token 没有被失效。


---

### 12. Bootstrap 默认开启 — [config.go:104](file:///d:/work/sub2api/server/internal/config/config.go#L104)

```go
AllowBootstrap: env("SUB2API_ALLOW_BOOTSTRAP", "true") == "true",
```

生产环境中如果忘记关闭，任何人都可以创建管理员账号。

---

## 总结

| 严重度 | Bug | 影响 |
|--------|-----|------|
| 🔴 严重 | flag.Parse 时机错误 | 命令行参数完全无效 |
| 🔴 严重 | credentials_json 列名不匹配 | 管理API更新凭证静默失败 |
| 🔴 严重 | 部分退款状态错误 | 第二次部分退款永远无法执行 |
| 🔴 严重 | 注册限流用 RemoteAddr | 速率限制完全失效 |
| 🟡 中等 | sync.Map 内存泄漏 | 长期运行内存增长 |
| 🟡 中等 | 逐条 flush key/user | 高并发下性能瓶颈 |
| 🟡 中等 | registerIPs 清理不及时 | 攻击时内存可能暴涨 |
| 🟡 中等 | 费率截断少收 | 累积计费偏差 |
| 🟢 轻微 | Gemini SupportsPath 死代码 | 无实际影响 |
| 🟢 轻微 | logout 静默成功 | 用户误以为已登出 |
| 🟢 轻微 | Bootstrap 默认开启 | 安全隐患 |

最需要优先修复的是前 4 个严重 Bug，尤其是 **#1 flag 失效** 和 **#2 凭证更新失败**，它们会导致核心功能完全不可用。
        
