package api

import (
	"context"
	"fmt"
	"strings"
	"time"

	"go-backend/shared"
)

const (
	chatRequestTimeout        = 180 * time.Second
	statusRequestTimeout      = 5 * time.Second
	chatRecallPreviewItems    = 2
	chatRecallSnippetLimit    = 100
	chatSystemPromptPrefix    = "You are a helpful AI assistant. Relevant memory context:\n"
	chatSystemPromptSuffix    = "\n"
	chatResponseAssistantRole = "assistant"
	chatMemoryTypeEvent       = "event"
	chatMemoryCategory        = "conversation"
)

type recallResult struct {
	Prompt string
	Status string
	Error  string
}

func (h *Handler) buildRecallContext(ctx context.Context, req shared.ChatRequest) recallResult {
	recallReq := &shared.RecallRequest{
		UserID: req.UserID,
		Query:  req.Message,
		TopK:   h.recallTopK,
		Method: "rag",
	}

	recallResp, err := h.memoryClient.Recall(ctx, recallReq)
	if err != nil {
		return recallResult{
			Prompt: "[Memory recall failed]",
			Status: "error",
			Error:  err.Error(),
		}
	}

	if !recallResp.Success || len(recallResp.Items) == 0 {
		return recallResult{
			Prompt: "[No relevant memories found]",
			Status: "empty",
		}
	}

	var builder strings.Builder
	builder.WriteString(fmt.Sprintf("[Retrieved %d memory items]\n", len(recallResp.Items)))
	for i, item := range recallResp.Items {
		if i >= chatRecallPreviewItems {
			break
		}
		builder.WriteString(fmt.Sprintf("- [%s] %s\n", item.Category, previewText(item.Content, chatRecallSnippetLimit)))
	}

	return recallResult{
		Prompt: builder.String(),
		Status: "ok",
	}
}

func (h *Handler) generateResponse(ctx context.Context, userPrompt, recallInfo string) (string, error) {
	systemPrompt := fmt.Sprintf("%s%s%s", chatSystemPromptPrefix, recallInfo, chatSystemPromptSuffix)
	return h.llmClient.GenerateCompletion(ctx, h.llmModel, systemPrompt, userPrompt)
}

func (h *Handler) storeChatExchange(ctx context.Context, req shared.ChatRequest, assistantText string) (*shared.MemorizeResponse, error) {
	memorizeReq := &shared.MemorizeRequest{
		UserID:        req.UserID,
		Text:          req.Message,
		AssistantText: assistantText,
		MemoryType:    chatMemoryTypeEvent,
		Category:      chatMemoryCategory,
	}

	return h.memoryClient.Memorize(ctx, memorizeReq)
}

func (h *Handler) buildChatResponse(req shared.ChatRequest, assistantText string, recall recallResult, memorizeResp *shared.MemorizeResponse, memorizeErr error) shared.ChatResponse {
	now := time.Now()
	memoryWrite := map[string]interface{}{"status": "ok"}
	if memorizeErr != nil {
		memoryWrite["status"] = "error"
		memoryWrite["error"] = memorizeErr.Error()
	} else if memorizeResp != nil {
		memoryWrite["saved"] = memorizeResp.Saved
		if !memorizeResp.Saved {
			memoryWrite["status"] = "skipped"
			if memorizeResp.SkipReason != "" {
				memoryWrite["skip_reason"] = memorizeResp.SkipReason
			}
		}
	}

	memoryRecall := map[string]interface{}{"status": recall.Status}
	if recall.Error != "" {
		memoryRecall["error"] = recall.Error
	}

	return shared.ChatResponse{
		UserID: req.UserID,
		Message: shared.ChatMessage{
			Role:        chatResponseAssistantRole,
			Content:     assistantText,
			ID:          fmt.Sprintf("msg_%d", now.UnixMilli()),
			TimestampMs: now.UnixMilli(),
		},
		SessionID: req.SessionID,
		Metadata: map[string]interface{}{
			"memory_context": recall.Prompt,
			"memory_recall":  memoryRecall,
			"memory_write":   memoryWrite,
			"model":          h.llmModel,
			"timestamp":      now.Format(time.RFC3339),
		},
	}
}

func previewText(content string, limit int) string {
	if limit <= 0 || len(content) <= limit {
		return content
	}
	return content[:limit]
}
