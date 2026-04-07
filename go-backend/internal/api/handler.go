package api

import (
	"go-backend/internal/logger"
	"go-backend/internal/memory"
	llm "go-backend/internal/ollama"
)

// Handler holds API handler state.
type Handler struct {
	memoryClient *memory.Client
	llmClient    *llm.Client
	logger       *logger.Logger
	llmModel     string
	recallTopK   int
}

// NewHandler creates a new API handler.
func NewHandler(memClient *memory.Client, llmClient *llm.Client, model string, recallTopK int, log *logger.Logger) *Handler {
	if recallTopK < 1 {
		recallTopK = 1
	}
	return &Handler{
		memoryClient: memClient,
		llmClient:    llmClient,
		logger:       log,
		llmModel:     model,
		recallTopK:   recallTopK,
	}
}
