package config

import (
	"fmt"
	"net"
	"net/url"
	"os"
	"runtime"
	"strconv"
	"strings"

	"github.com/joho/godotenv"
)

// Config holds all application configuration.
type Config struct {
	Server   ServerConfig
	Memory   MemoryConfig
	LLM      LLMConfig
	Chat     ChatConfig
	Frontend FrontendConfig
	Log      LogConfig
}

// ServerConfig holds HTTP server settings.
type ServerConfig struct {
	Port int
	Host string
}

// MemoryConfig holds Python sidecar settings.
type MemoryConfig struct {
	Endpoint string // e.g., "127.0.0.1:50051" (Windows dev default) or "unix:///tmp/myural_yukari_memory.sock"
}

// LLMConfig holds llama.cpp / OpenAI-compatible settings.
type LLMConfig struct {
	BaseURL string // e.g., "http://localhost:11434/v1"
	Model   string // e.g., "unsloth/gemma-4-E4B-it-GGUF"
}

// ChatConfig holds chat flow tuning settings.
type ChatConfig struct {
	RecallTopK int // Number of memory items to request from sidecar
}

// FrontendConfig holds browser/Tauri origin settings.
type FrontendConfig struct {
	Origin string // e.g., "http://localhost:1420"
}

// LogConfig holds logging settings.
type LogConfig struct {
	Level   string // "debug", "info", "warn", "error"
	DevMode bool   // Enable verbose dev logging
}

// Load returns configuration from environment variables.
func Load() (*Config, error) {
	// Try to load .env file (optional)
	_ = godotenv.Load()

	cfg := &Config{
		Server: ServerConfig{
			Port: getEnvInt("SERVER_PORT", 8000),
			Host: getEnv("SERVER_HOST", "localhost"),
		},
		Memory: MemoryConfig{
			Endpoint: getMemoryEndpoint(),
		},
		LLM: LLMConfig{
			BaseURL: getEnv("LLM_BASE_URL", "http://localhost:11434/v1"),
			Model:   getEnv("CHAT_MODEL", "unsloth/gemma-4-E4B-it-GGUF"),
		},
		Chat: ChatConfig{
			RecallTopK: getEnvInt("RETRIEVAL_TOP_K", 5),
		},
		Frontend: FrontendConfig{
			Origin: getEnv("FRONTEND_ORIGIN", "http://localhost:1420"),
		},
		Log: LogConfig{
			Level:   getEnv("LOG_LEVEL", "info"),
			DevMode: getEnvBool("DEV_MODE", true),
		},
	}

	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	return cfg, nil
}

func getEnv(key, defaultVal string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return defaultVal
}

func getEnvInt(key string, defaultVal int) int {
	if value, exists := os.LookupEnv(key); exists {
		if intVal, err := strconv.Atoi(value); err == nil {
			return intVal
		}
	}
	return defaultVal
}

func getEnvBool(key string, defaultVal bool) bool {
	if value, exists := os.LookupEnv(key); exists {
		return value == "1" || value == "true" || value == "yes"
	}
	return defaultVal
}

// Validate checks the config values that must be well-formed before startup.
func (c *Config) Validate() error {
	if c.Server.Port <= 0 || c.Server.Port > 65535 {
		return fmt.Errorf("invalid SERVER_PORT: %d", c.Server.Port)
	}
	if err := validateMemoryEndpoint(c.Memory.Endpoint); err != nil {
		return err
	}
	if err := validateURL("LLM_BASE_URL", c.LLM.BaseURL); err != nil {
		return err
	}
	if c.Chat.RecallTopK < 1 || c.Chat.RecallTopK > 50 {
		return fmt.Errorf("invalid RETRIEVAL_TOP_K: %d", c.Chat.RecallTopK)
	}
	if err := validateURL("FRONTEND_ORIGIN", c.Frontend.Origin); err != nil {
		return err
	}
	return nil
}

func validateURL(name, value string) error {
	if strings.TrimSpace(value) == "" {
		return fmt.Errorf("%s must not be empty", name)
	}
	parsed, err := url.Parse(value)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return fmt.Errorf("invalid %s: %q", name, value)
	}
	return nil
}

func (c *Config) String() string {
	return fmt.Sprintf(`
=== Configuration ===
Server: %s:%d
Memory Endpoint: %s
LLM: %s (model: %s)
Recall TopK: %d
Frontend Origin: %s
Log Level: %s
Dev Mode: %v
`, c.Server.Host, c.Server.Port, c.Memory.Endpoint, c.LLM.BaseURL, c.LLM.Model, c.Chat.RecallTopK, c.Frontend.Origin, c.Log.Level, c.Log.DevMode)
}

func getMemoryEndpoint() string {
	if endpoint, exists := os.LookupEnv("MEMORY_GRPC_ENDPOINT"); exists {
		return endpoint
	}

	if runtime.GOOS == "windows" {
		return "127.0.0.1:50051"
	}

	return "unix:///tmp/myural_yukari_memory.sock"
}

func validateMemoryEndpoint(value string) error {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return fmt.Errorf("MEMORY_GRPC_ENDPOINT must not be empty")
	}

	if strings.HasPrefix(trimmed, "http://") || strings.HasPrefix(trimmed, "https://") {
		return fmt.Errorf("invalid MEMORY_GRPC_ENDPOINT: HTTP endpoints are no longer supported (%q)", value)
	}

	if strings.HasPrefix(trimmed, "unix:") || strings.HasPrefix(trimmed, "npipe://") {
		return nil
	}

	if host, port, err := net.SplitHostPort(trimmed); err == nil {
		if strings.TrimSpace(host) == "" {
			return fmt.Errorf("invalid MEMORY_GRPC_ENDPOINT: %q", value)
		}

		portVal, convErr := strconv.Atoi(port)
		if convErr != nil || portVal < 1 || portVal > 65535 {
			return fmt.Errorf("invalid MEMORY_GRPC_ENDPOINT: %q", value)
		}

		return nil
	}

	if strings.HasPrefix(trimmed, "dns:///") {
		host := strings.TrimSpace(strings.TrimPrefix(trimmed, "dns:///"))
		if host == "" {
			return fmt.Errorf("invalid MEMORY_GRPC_ENDPOINT: %q", value)
		}
		return nil
	}

	return fmt.Errorf("invalid MEMORY_GRPC_ENDPOINT: %q (expected unix:/, npipe://, or host:port endpoint)", value)
}
