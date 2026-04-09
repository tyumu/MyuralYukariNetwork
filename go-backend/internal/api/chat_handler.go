package api

import (
	"context"
	"net/http"

	"go-backend/shared"
)

// Chat handles a chat request.
func (h *Handler) Chat(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	req, err := h.decodeChatRequest(r)
	if err != nil {
		h.logger.Error("chat request decode error", "error", err.Error())
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	h.logger.Info("chat request", "user_id", req.UserID, "message_len", len(req.Message))

	ctx, cancel := context.WithTimeout(context.Background(), chatRequestTimeout)
	defer cancel()

	recallInfo := h.buildRecallContext(ctx, req)
	if recallInfo.Status == "error" {
		h.logger.Warn("recall failed", "error", recallInfo.Error)
	}

	response, err := h.generateResponse(ctx, req.Message, recallInfo.Prompt)
	if err != nil {
		h.logger.Error("llama.cpp generation failed", "error", err.Error())
		message, statusCode := classifyGenerationError(err)
		writeJSONError(w, message, statusCode)
		return
	}

	memorizeResp, memorizeErr := h.storeChatExchange(ctx, req, response)
	if memorizeErr != nil {
		h.logger.Warn("memorize failed", "error", memorizeErr.Error())
	}

	chatResp := h.buildChatResponse(req, response, recallInfo, memorizeResp, memorizeErr)
	writeJSON(w, chatResp, http.StatusOK)
}

func (h *Handler) decodeChatRequest(r *http.Request) (shared.ChatRequest, error) {
	var req shared.ChatRequest
	if err := decodeJSON(r, &req); err != nil {
		return shared.ChatRequest{}, err
	}
	return req, nil
}
