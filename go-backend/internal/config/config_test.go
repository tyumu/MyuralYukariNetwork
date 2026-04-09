package config

import (
	"os"
	"testing"
)

func TestValidateRejectsInvalidValues(t *testing.T) {
	t.Run("invalid port", func(t *testing.T) {
		cfg := &Config{Server: ServerConfig{Port: 0}, Memory: MemoryConfig{Endpoint: "npipe:////./pipe/myural_yukari_memory"}, LLM: LLMConfig{BaseURL: "http://localhost:11434/v1"}, Frontend: FrontendConfig{Origin: "http://localhost:1420"}}
		if err := cfg.Validate(); err == nil {
			t.Fatal("expected validation error for invalid port")
		}
	})

	t.Run("invalid endpoint", func(t *testing.T) {
		cfg := &Config{Server: ServerConfig{Port: 8000}, Memory: MemoryConfig{Endpoint: "bad-endpoint"}, LLM: LLMConfig{BaseURL: "http://localhost:11434/v1"}, Chat: ChatConfig{RecallTopK: 5}, Frontend: FrontendConfig{Origin: "http://localhost:1420"}}
		if err := cfg.Validate(); err == nil {
			t.Fatal("expected validation error for invalid memory endpoint")
		}
	})

	t.Run("legacy http endpoint rejected", func(t *testing.T) {
		cfg := &Config{Server: ServerConfig{Port: 8000}, Memory: MemoryConfig{Endpoint: "http://localhost:8001"}, LLM: LLMConfig{BaseURL: "http://localhost:11434/v1"}, Chat: ChatConfig{RecallTopK: 5}, Frontend: FrontendConfig{Origin: "http://localhost:1420"}}
		if err := cfg.Validate(); err == nil {
			t.Fatal("expected validation error for legacy HTTP memory endpoint")
		}
	})

	t.Run("tcp endpoint accepted", func(t *testing.T) {
		cfg := &Config{Server: ServerConfig{Port: 8000}, Memory: MemoryConfig{Endpoint: "127.0.0.1:50051"}, LLM: LLMConfig{BaseURL: "http://localhost:11434/v1"}, Chat: ChatConfig{RecallTopK: 5}, Frontend: FrontendConfig{Origin: "http://localhost:1420"}}
		if err := cfg.Validate(); err != nil {
			t.Fatalf("expected TCP memory endpoint to be accepted, got: %v", err)
		}
	})

	t.Run("invalid tcp endpoint rejected", func(t *testing.T) {
		cfg := &Config{Server: ServerConfig{Port: 8000}, Memory: MemoryConfig{Endpoint: "127.0.0.1"}, LLM: LLMConfig{BaseURL: "http://localhost:11434/v1"}, Chat: ChatConfig{RecallTopK: 5}, Frontend: FrontendConfig{Origin: "http://localhost:1420"}}
		if err := cfg.Validate(); err == nil {
			t.Fatal("expected validation error for malformed TCP memory endpoint")
		}
	})

	t.Run("invalid retrieval top k", func(t *testing.T) {
		cfg := &Config{Server: ServerConfig{Port: 8000}, Memory: MemoryConfig{Endpoint: "npipe:////./pipe/myural_yukari_memory"}, LLM: LLMConfig{BaseURL: "http://localhost:11434/v1"}, Chat: ChatConfig{RecallTopK: 0}, Frontend: FrontendConfig{Origin: "http://localhost:1420"}}
		if err := cfg.Validate(); err == nil {
			t.Fatal("expected validation error for invalid retrieval top k")
		}
	})
}

func TestLoadReadsRetrievalTopK(t *testing.T) {
	_ = os.Setenv("SERVER_PORT", "8000")
	_ = os.Setenv("MEMORY_GRPC_ENDPOINT", "127.0.0.1:50051")
	_ = os.Setenv("LLM_BASE_URL", "http://localhost:11434/v1")
	_ = os.Setenv("FRONTEND_ORIGIN", "http://localhost:1420")
	_ = os.Setenv("RETRIEVAL_TOP_K", "7")
	defer func() {
		_ = os.Unsetenv("SERVER_PORT")
		_ = os.Unsetenv("MEMORY_GRPC_ENDPOINT")
		_ = os.Unsetenv("LLM_BASE_URL")
		_ = os.Unsetenv("FRONTEND_ORIGIN")
		_ = os.Unsetenv("RETRIEVAL_TOP_K")
	}()

	cfg, err := Load()
	if err != nil {
		t.Fatalf("expected load success, got error: %v", err)
	}
	if cfg.Chat.RecallTopK != 7 {
		t.Fatalf("expected RecallTopK=7, got %d", cfg.Chat.RecallTopK)
	}
}
