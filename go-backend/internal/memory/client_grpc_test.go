package memory

import (
	"context"
	"net"
	"testing"

	"go-backend/internal/logger"
	"go-backend/internal/memory/memorypb"
	"go-backend/shared"

	"google.golang.org/grpc"
)

type mockMemoryService struct {
	memorypb.UnimplementedMemoryServiceServer

	memorizeResp *memorypb.MemorizeResponse
	recallResp   *memorypb.RecallResponse
	healthResp   *memorypb.HealthResponse
}

func (m *mockMemoryService) Memorize(context.Context, *memorypb.MemorizeRequest) (*memorypb.MemorizeResponse, error) {
	return m.memorizeResp, nil
}

func (m *mockMemoryService) Recall(context.Context, *memorypb.RecallRequest) (*memorypb.RecallResponse, error) {
	return m.recallResp, nil
}

func (m *mockMemoryService) Health(context.Context, *memorypb.HealthRequest) (*memorypb.HealthResponse, error) {
	return m.healthResp, nil
}

func startMockGRPCServer(t *testing.T, svc *mockMemoryService) string {
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

func TestGRPCMemorizeAndRecall(t *testing.T) {
	addr := startMockGRPCServer(t, &mockMemoryService{
		memorizeResp: &memorypb.MemorizeResponse{
			UserId:   "u1",
			ItemId:   "item-1",
			Category: "conversation",
			Success:  true,
			Saved:    true,
		},
		recallResp: &memorypb.RecallResponse{
			UserId: "u1",
			Items: []*memorypb.RecallItem{
				{Id: "m1", Content: "likes mugicha", Category: "conversation", Salience: 0.99},
			},
			Success: true,
		},
		healthResp: &memorypb.HealthResponse{Healthy: true, Status: "ok"},
	})

	client := NewClient(addr, logger.NewLogger(logger.ErrorLevel, false))

	memResp, err := client.Memorize(context.Background(), &shared.MemorizeRequest{
		UserID:     "u1",
		Text:       "hello",
		MemoryType: "event",
		Category:   "conversation",
	})
	if err != nil {
		t.Fatalf("memorize error: %v", err)
	}
	if !memResp.Saved || memResp.ItemID != "item-1" {
		t.Fatalf("unexpected memorize response: %+v", memResp)
	}

	recallResp, err := client.Recall(context.Background(), &shared.RecallRequest{
		UserID: "u1",
		Query:  "what does user like?",
		TopK:   5,
		Method: "rag",
	})
	if err != nil {
		t.Fatalf("recall error: %v", err)
	}
	if len(recallResp.Items) != 1 || recallResp.Items[0].ID != "m1" {
		t.Fatalf("unexpected recall response: %+v", recallResp)
	}

	healthy, err := client.Health(context.Background())
	if err != nil {
		t.Fatalf("health error: %v", err)
	}
	if !healthy {
		t.Fatal("expected healthy=true")
	}
}

func TestGRPCMemorizeFailure(t *testing.T) {
	addr := startMockGRPCServer(t, &mockMemoryService{
		memorizeResp: &memorypb.MemorizeResponse{
			UserId:   "u1",
			Category: "conversation",
			Success:  false,
			Error:    "memory down",
		},
		recallResp: &memorypb.RecallResponse{UserId: "u1", Success: true},
		healthResp: &memorypb.HealthResponse{Healthy: true, Status: "ok"},
	})

	client := NewClient(addr, logger.NewLogger(logger.ErrorLevel, false))
	_, err := client.Memorize(context.Background(), &shared.MemorizeRequest{UserID: "u1", Text: "hello", MemoryType: "event", Category: "conversation"})
	if err == nil {
		t.Fatal("expected error when grpc memorize returns success=false")
	}
}
