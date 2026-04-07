package memory

import (
	"context"
	"testing"

	"go-backend/internal/logger"
	"go-backend/internal/memory/memorypb"
	"go-backend/shared"
)

func TestMemorizeReturnsErrorOnSuccessFalse(t *testing.T) {
	addr := startMockGRPCServer(t, &mockMemoryService{
		memorizeResp: &memorypb.MemorizeResponse{
			UserId:   "u1",
			Category: "conversation",
			Success:  false,
			Error:    "db down",
		},
		recallResp: &memorypb.RecallResponse{UserId: "u1", Success: true},
		healthResp: &memorypb.HealthResponse{Healthy: true, Status: "ok"},
	})

	c := NewClient(addr, logger.NewLogger(logger.ErrorLevel, false))
	_, err := c.Memorize(context.Background(), &shared.MemorizeRequest{UserID: "u1", Text: "hello", MemoryType: "event", Category: "conversation"})
	if err == nil {
		t.Fatal("expected error when sidecar returns success=false")
	}
}

func TestMemorizeReturnsNoErrorWhenSkipped(t *testing.T) {
	addr := startMockGRPCServer(t, &mockMemoryService{
		memorizeResp: &memorypb.MemorizeResponse{
			UserId:     "u1",
			Category:   "conversation",
			Success:    true,
			Saved:      false,
			SkipReason: "no_memory_items_created",
		},
		recallResp: &memorypb.RecallResponse{UserId: "u1", Success: true},
		healthResp: &memorypb.HealthResponse{Healthy: true, Status: "ok"},
	})

	c := NewClient(addr, logger.NewLogger(logger.ErrorLevel, false))
	resp, err := c.Memorize(context.Background(), &shared.MemorizeRequest{UserID: "u1", Text: "hello", MemoryType: "event", Category: "conversation"})
	if err != nil {
		t.Fatalf("expected skipped memorize to be non-error, got: %v", err)
	}
	if resp.Saved {
		t.Fatal("expected Saved=false for skipped memorize response")
	}
}

func TestRecallReturnsErrorOnSuccessFalse(t *testing.T) {
	addr := startMockGRPCServer(t, &mockMemoryService{
		memorizeResp: &memorypb.MemorizeResponse{UserId: "u1", Category: "conversation", Success: true, Saved: true},
		recallResp:   &memorypb.RecallResponse{UserId: "u1", Success: false, Error: "query failed"},
		healthResp:   &memorypb.HealthResponse{Healthy: true, Status: "ok"},
	})

	c := NewClient(addr, logger.NewLogger(logger.ErrorLevel, false))
	_, err := c.Recall(context.Background(), &shared.RecallRequest{UserID: "u1", Query: "q", TopK: 3, Method: "rag"})
	if err == nil {
		t.Fatal("expected error when sidecar returns success=false")
	}
}
