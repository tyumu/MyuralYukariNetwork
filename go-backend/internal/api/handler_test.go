package api

import (
	"bytes"
	"context"
	"encoding/json"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"

	"go-backend/internal/logger"
	"go-backend/internal/memory"
	"go-backend/internal/memory/memorypb"
	llm "go-backend/internal/ollama"
	"go-backend/shared"

	"google.golang.org/grpc"
)

type mockMemoryService struct {
	memorypb.UnimplementedMemoryServiceServer

	memorizeResp *memorypb.MemorizeResponse
	recallResp   *memorypb.RecallResponse
	healthResp   *memorypb.HealthResponse
}

func (m *mockMemoryService) Memorize(_ context.Context, req *memorypb.MemorizeRequest) (*memorypb.MemorizeResponse, error) {
	if m.memorizeResp != nil {
		return m.memorizeResp, nil
	}
	return &memorypb.MemorizeResponse{
		UserId:   req.GetUserId(),
		Category: "conversation",
		Success:  true,
		Saved:    true,
	}, nil
}

func (m *mockMemoryService) Recall(_ context.Context, req *memorypb.RecallRequest) (*memorypb.RecallResponse, error) {
	if m.recallResp != nil {
		return m.recallResp, nil
	}
	return &memorypb.RecallResponse{UserId: req.GetUserId(), Success: true}, nil
}

func (m *mockMemoryService) Health(context.Context, *memorypb.HealthRequest) (*memorypb.HealthResponse, error) {
	if m.healthResp != nil {
		return m.healthResp, nil
	}
	return &memorypb.HealthResponse{Healthy: true, Status: "ok"}, nil
}

func startMockMemoryGRPCServer(t *testing.T, svc *mockMemoryService) string {
	t.Helper()

	lis, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}

	server := grpc.NewServer()
	memorypb.RegisterMemoryServiceServer(server, svc)

	go func() {
		_ = server.Serve(lis)
	}()

	t.Cleanup(func() {
		server.Stop()
		_ = lis.Close()
	})

	return lis.Addr().String()
}

func TestChatSuccessIncludesMemoryStatuses(t *testing.T) {
	memoryAddr := startMockMemoryGRPCServer(t, &mockMemoryService{
		recallResp: &memorypb.RecallResponse{
			UserId: "u1",
			Items: []*memorypb.RecallItem{{
				Id:       "m1",
				Content:  "prefers short answers",
				Category: "conversation",
				Salience: 0.9,
			}},
			Success: true,
		},
		memorizeResp: &memorypb.MemorizeResponse{
			UserId:   "u1",
			ItemId:   "item1",
			Category: "conversation",
			Success:  true,
			Saved:    true,
		},
		healthResp: &memorypb.HealthResponse{Healthy: true, Status: "ok"},
	})

	llmServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/chat/completions":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"choices":[{"message":{"role":"assistant","content":"hello"}}]}`))
		case "/models":
			w.WriteHeader(http.StatusOK)
		default:
			http.NotFound(w, r)
		}
	}))
	defer llmServer.Close()

	log := logger.NewLogger(logger.ErrorLevel, false)
	h := NewHandler(memory.NewClient(memoryAddr, log), llm.NewClient(llmServer.URL, log), "mock-model", 3, log)

	body := []byte(`{"user_id":"u1","message":"hi","session_id":"s1"}`)
	req := httptest.NewRequest(http.MethodPost, "/chat", bytes.NewReader(body))
	rec := httptest.NewRecorder()

	h.Chat(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", rec.Code, rec.Body.String())
	}

	var got map[string]interface{}
	if err := json.NewDecoder(rec.Body).Decode(&got); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	metadata, ok := got["metadata"].(map[string]interface{})
	if !ok {
		t.Fatal("metadata missing")
	}

	recallStatus := metadata["memory_recall"].(map[string]interface{})["status"]
	if recallStatus != "ok" {
		t.Fatalf("expected memory_recall.status=ok, got %v", recallStatus)
	}

	writeStatus := metadata["memory_write"].(map[string]interface{})["status"]
	if writeStatus != "ok" {
		t.Fatalf("expected memory_write.status=ok, got %v", writeStatus)
	}
}

func TestChatWhenMemorizeSkippedMarksMetadata(t *testing.T) {
	memoryAddr := startMockMemoryGRPCServer(t, &mockMemoryService{
		recallResp: &memorypb.RecallResponse{UserId: "u1", Success: true},
		memorizeResp: &memorypb.MemorizeResponse{
			UserId:     "u1",
			Category:   "conversation",
			Success:    true,
			Saved:      false,
			SkipReason: "no_memory_items_created",
		},
		healthResp: &memorypb.HealthResponse{Healthy: true, Status: "ok"},
	})

	llmServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/chat/completions":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"choices":[{"message":{"role":"assistant","content":"hello"}}]}`))
		case "/models":
			w.WriteHeader(http.StatusOK)
		default:
			http.NotFound(w, r)
		}
	}))
	defer llmServer.Close()

	log := logger.NewLogger(logger.ErrorLevel, false)
	h := NewHandler(memory.NewClient(memoryAddr, log), llm.NewClient(llmServer.URL, log), "mock-model", 3, log)

	body := []byte(`{"user_id":"u1","message":"hi","session_id":"s1"}`)
	req := httptest.NewRequest(http.MethodPost, "/chat", bytes.NewReader(body))
	rec := httptest.NewRecorder()

	h.Chat(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", rec.Code, rec.Body.String())
	}

	var got map[string]interface{}
	if err := json.NewDecoder(rec.Body).Decode(&got); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	metadata, ok := got["metadata"].(map[string]interface{})
	if !ok {
		t.Fatal("metadata missing")
	}

	memoryWrite, ok := metadata["memory_write"].(map[string]interface{})
	if !ok {
		t.Fatal("memory_write missing")
	}

	if memoryWrite["status"] != "skipped" {
		t.Fatalf("expected memory_write.status=skipped, got %v", memoryWrite["status"])
	}
	if memoryWrite["saved"] != false {
		t.Fatalf("expected memory_write.saved=false, got %v", memoryWrite["saved"])
	}
}

func TestChatWhenLLMUnavailableReturnsBadGateway(t *testing.T) {
	memoryAddr := startMockMemoryGRPCServer(t, &mockMemoryService{
		recallResp: &memorypb.RecallResponse{UserId: "u1", Success: true},
		healthResp: &memorypb.HealthResponse{Healthy: true, Status: "ok"},
	})

	llmServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/chat/completions" {
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte(`{"error":"model down"}`))
			return
		}
		if r.URL.Path == "/models" {
			w.WriteHeader(http.StatusOK)
			return
		}
		http.NotFound(w, r)
	}))
	defer llmServer.Close()

	log := logger.NewLogger(logger.ErrorLevel, false)
	h := NewHandler(memory.NewClient(memoryAddr, log), llm.NewClient(llmServer.URL, log), "mock-model", 3, log)

	body := []byte(`{"user_id":"u1","message":"hi","session_id":"s1"}`)
	req := httptest.NewRequest(http.MethodPost, "/chat", bytes.NewReader(body))
	rec := httptest.NewRecorder()

	h.Chat(rec, req)

	if rec.Code != http.StatusBadGateway {
		t.Fatalf("expected 502, got %d body=%s", rec.Code, rec.Body.String())
	}
}

func TestClassifyGenerationErrorTimeout(t *testing.T) {
	msg, code := classifyGenerationError(context.DeadlineExceeded)
	if code != http.StatusGatewayTimeout {
		t.Fatalf("expected 504, got %d", code)
	}
	if msg != "Generation timeout" {
		t.Fatalf("unexpected message: %s", msg)
	}
}

func TestStatusHealthyWhenAllServicesOK(t *testing.T) {
	h := newStatusTestHandler(t, true, http.StatusOK)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/status", nil)

	h.Status(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	var resp shared.StatusResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if resp.Status != "healthy" {
		t.Fatalf("expected healthy, got %s", resp.Status)
	}
}

func TestStatusDegradedWhenOneServiceDown(t *testing.T) {
	h := newStatusTestHandler(t, true, http.StatusServiceUnavailable)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/status", nil)

	h.Status(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	var resp shared.StatusResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if resp.Status != "degraded" {
		t.Fatalf("expected degraded, got %s", resp.Status)
	}
}

func TestStatusErrorWhenAllServicesDown(t *testing.T) {
	h := newStatusTestHandler(t, false, http.StatusServiceUnavailable)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/status", nil)

	h.Status(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	var resp shared.StatusResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if resp.Status != "error" {
		t.Fatalf("expected error, got %s", resp.Status)
	}
}

func newStatusTestHandler(t *testing.T, memoryHealthy bool, llmHealthCode int) *Handler {
	t.Helper()
	memoryAddr := startMockMemoryGRPCServer(t, &mockMemoryService{
		healthResp: &memorypb.HealthResponse{Healthy: memoryHealthy, Status: "ok"},
	})

	llmServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/models":
			w.WriteHeader(llmHealthCode)
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(llmServer.Close)

	log := logger.NewLogger(logger.ErrorLevel, false)
	return NewHandler(memory.NewClient(memoryAddr, log), llm.NewClient(llmServer.URL, log), "mock-model", 3, log)
}
