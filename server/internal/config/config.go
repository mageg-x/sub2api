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

type Config struct {
	Addr          string
	DBPath        string
	AdminToken    string
	AESKey        []byte
	PublicBaseURL string
	GopayURL      string
	GopayPID      uint64
	GopayKey      string
	GopayType     int
	OpenAI        OpenAIConfig
	Claude        ClaudeConfig
	Gemini        GeminiConfig
	Antigravity   AntigravityConfig
}

type OpenAIConfig struct {
	ClientID string
}

type ClaudeConfig struct {
	ClientID string
}

type GeminiConfig struct {
	ClientID            string
	ClientSecret        string
	BuiltinClientSecret string
}

type AntigravityConfig struct {
	ClientSecret string
}

func Load() Config {
	addr := stringValue("addr", env("SUB2API_ADDR", ":8080"))
	dbPath := stringValue("db", env("SUB2API_DB", filepath.Join("data", "sub2api.db")))
	adminToken := stringValue("admin-token", env("SUB2API_ADMIN_TOKEN", "sub2api-admin-change-me"))
	publicBase := stringValue("public-base-url", env("SUB2API_PUBLIC_BASE_URL", ""))
	gopayURL := stringValue("gopay-url", env("SUB2API_GOPAY_URL", ""))
	gopayPID := uint64Value("gopay-pid", env("SUB2API_GOPAY_PID", "0"))
	gopayKey := stringValue("gopay-key", env("SUB2API_GOPAY_KEY", ""))
	gopayType := intValue("gopay-type", env("SUB2API_GOPAY_TYPE", "1"))
	aesSeed := stringValue("aes-key", env("SUB2API_AES_KEY", "sub2api-dev-key"))
	flag.Parse()

	return Config{
		Addr:          addr,
		DBPath:        dbPath,
		AdminToken:    adminToken,
		AESKey:        derive32(aesSeed),
		PublicBaseURL: strings.TrimRight(publicBase, "/"),
		GopayURL:      strings.TrimRight(gopayURL, "/"),
		GopayPID:      gopayPID,
		GopayKey:      gopayKey,
		GopayType:     gopayType,
		OpenAI: OpenAIConfig{
			ClientID: env("SUB2API_OPENAI_CLIENT_ID", "app_EMoamEEZ73f0CkXaXp7hrann"),
		},
		Claude: ClaudeConfig{
			ClientID: env("SUB2API_CLAUDE_CLIENT_ID", "9d1c250a-e61b-44d9-88ed-5944d1962f5e"),
		},
		Gemini: GeminiConfig{
			ClientID:            env("SUB2API_GEMINI_CLIENT_ID", ""),
			ClientSecret:        env("SUB2API_GEMINI_CLIENT_SECRET", ""),
			BuiltinClientSecret: env("GEMINI_CLI_OAUTH_CLIENT_SECRET", env("SUB2API_GEMINI_BUILTIN_CLIENT_SECRET", "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")),
		},
		Antigravity: AntigravityConfig{
			ClientSecret: env("ANTIGRAVITY_OAUTH_CLIENT_SECRET", env("SUB2API_ANTIGRAVITY_CLIENT_SECRET", "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")),
		},
	}
}

func env(key, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(key)); value != "" {
		return value
	}
	return fallback
}

func stringValue(name, value string) string {
	return *flag.String(name, value, "")
}

func intValue(name, raw string) int {
	value, _ := strconv.Atoi(raw)
	return *flag.Int(name, value, "")
}

func uint64Value(name, raw string) uint64 {
	value, _ := strconv.ParseUint(raw, 10, 64)
	return *flag.Uint64(name, value, "")
}

func derive32(seed string) []byte {
	if raw, err := hex.DecodeString(seed); err == nil && len(raw) == 32 {
		return raw
	}
	sum := sha256.Sum256([]byte(seed))
	return sum[:]
}
