package config

import (
	"crypto/sha256"
	"encoding/hex"
	"flag"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// Config 应用程序的完整配置结构
// 包含服务器配置、数据库配置、加密配置、支付配置和各AI Provider的配置
type Config struct {
	Addr           string            // HTTP服务器监听地址
	DBPath         string            // SQLite数据库文件路径
	AdminToken     string            // 管理员认证Token
	AllowBootstrap bool              // 是否允许引导模式
	AESKey         []byte            // AES加密密钥（32字节）
	PublicBaseURL  string            // 公共访问的基础URL（用于支付回调等）
	GopayURL       string            // Gopay支付网关URL
	GopayPID       uint64            // Gopay商户ID
	GopayKey       string            // Gopay商户密钥
	GopayType      int               // Gopay支付类型
	OpenAI         OpenAIConfig      // OpenAI Provider配置
	Claude         ClaudeConfig      // Claude Provider配置
	Gemini         GeminiConfig      // Gemini Provider配置
	Antigravity    AntigravityConfig // Antigravity Provider配置
}

// OpenAIConfig OpenAI Provider的配置
type OpenAIConfig struct {
	ClientID string // OAuth客户端ID
}

// ClaudeConfig Claude Provider的配置
type ClaudeConfig struct {
	ClientID string // OAuth客户端ID
}

// GeminiConfig Gemini Provider的配置
type GeminiConfig struct {
	ClientID            string // OAuth客户端ID
	ClientSecret        string // OAuth客户端密钥
	BuiltinClientSecret string // 内置客户端密钥（用于CLI）
}

// AntigravityConfig Antigravity Provider的配置
type AntigravityConfig struct {
	ClientSecret string // OAuth客户端密钥
}

// Load 从环境变量和命令行参数加载配置
// 优先级：命令行参数 > 环境变量 > 默认值
// 返回: 配置结构体
func Load() Config {
	addr := flag.String("addr", env("SUB2API_ADDR", ":8080"), "")
	dbPath := flag.String("db", env("SUB2API_DB", filepath.Join("data", "sub2api.db")), "")
	adminToken := flag.String("admin-token", env("SUB2API_ADMIN_TOKEN", "sub2api-admin-change-me"), "")
	publicBase := flag.String("public-base-url", env("SUB2API_PUBLIC_BASE_URL", ""), "")
	gopayURL := flag.String("gopay-url", env("SUB2API_GOPAY_URL", ""), "")
	gopayPID := flag.Uint64("gopay-pid", mustParseUint64(env("SUB2API_GOPAY_PID", "0")), "")
	gopayKey := flag.String("gopay-key", env("SUB2API_GOPAY_KEY", ""), "")
	gopayType := flag.Int("gopay-type", mustParseInt(env("SUB2API_GOPAY_TYPE", "1")), "")
	aesSeed := flag.String("aes-key", env("SUB2API_AES_KEY", "sub2api-dev-key"), "")
	// 解析命令行参数
	flag.Parse()

	// 返回完整配置
	return Config{
		Addr:           *addr,
		DBPath:         *dbPath,
		AdminToken:     *adminToken,
		AllowBootstrap: env("SUB2API_ALLOW_BOOTSTRAP", "false") == "true",
		AESKey:         derive32(*aesSeed),                  // 从种子派生32字节密钥
		PublicBaseURL:  strings.TrimRight(*publicBase, "/"), // 移除尾部斜杠
		GopayURL:       strings.TrimRight(*gopayURL, "/"),
		GopayPID:       *gopayPID,
		GopayKey:       *gopayKey,
		GopayType:      *gopayType,
		// OpenAI配置
		OpenAI: OpenAIConfig{
			ClientID: env("SUB2API_OPENAI_CLIENT_ID", "app_EMoamEEZ73f0CkXaXp7hrann"),
		},
		// Claude配置
		Claude: ClaudeConfig{
			ClientID: env("SUB2API_CLAUDE_CLIENT_ID", "9d1c250a-e61b-44d9-88ed-5944d1962f5e"),
		},
		// Gemini配置
		Gemini: GeminiConfig{
			ClientID:            env("SUB2API_GEMINI_CLIENT_ID", ""),
			ClientSecret:        env("SUB2API_GEMINI_CLIENT_SECRET", ""),
			BuiltinClientSecret: env("GEMINI_CLI_OAUTH_CLIENT_SECRET", env("SUB2API_GEMINI_BUILTIN_CLIENT_SECRET", "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")),
		},
		// Antigravity配置
		Antigravity: AntigravityConfig{
			ClientSecret: env("ANTIGRAVITY_OAUTH_CLIENT_SECRET", env("SUB2API_ANTIGRAVITY_CLIENT_SECRET", "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")),
		},
	}
}

// env 获取环境变量的值
// 如果环境变量存在且非空，返回其值；否则返回默认值
func env(key, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(key)); value != "" {
		return value
	}
	return fallback
}

func mustParseInt(raw string) int {
	value, _ := strconv.Atoi(raw)
	return value
}

func mustParseUint64(raw string) uint64 {
	value, _ := strconv.ParseUint(raw, 10, 64)
	return value
}

// derive32 从种子生成32字节的AES加密密钥
// 如果种子已经是32字节的十六进制字符串，直接使用
// 否则对种子进行SHA256哈希并取前32字节
func derive32(seed string) []byte {
	// 尝试直接解码十六进制字符串
	if raw, err := hex.DecodeString(seed); err == nil && len(raw) == 32 {
		return raw
	}
	// 否则对种子进行SHA256哈希
	sum := sha256.Sum256([]byte(seed))
	return sum[:]
}
