package memory

import (
	"context"
	"errors"
	"fmt"
	"net"
	"strings"
	"sync"
	"time"

	"go-backend/internal/logger"
	"go-backend/internal/memory/memorypb"
	"go-backend/shared"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/protobuf/types/known/structpb"
)

const grpcDialTimeout = 10 * time.Second

// Client communicates with the Python MemU sidecar API.
type Client struct {
	target string
	logger *logger.Logger

	mu         sync.Mutex
	grpcConn   *grpc.ClientConn
	grpcClient memorypb.MemoryServiceClient
}

// NewClient creates a new memory client.
func NewClient(target string, log *logger.Logger) *Client {
	return &Client{
		target: target,
		logger: log,
	}
}

// Memorize stores text in memory via the sidecar API.
func (c *Client) Memorize(ctx context.Context, req *shared.MemorizeRequest) (*shared.MemorizeResponse, error) {
	return c.memorizeGRPC(ctx, req)
}

func (c *Client) memorizeGRPC(ctx context.Context, req *shared.MemorizeRequest) (*shared.MemorizeResponse, error) {
	c.logger.Debug("memorize request", "user_id", req.UserID, "category", req.Category)

	grpcClient, err := c.ensureGRPCClient(ctx)
	if err != nil {
		return nil, err
	}

	resp, err := grpcClient.Memorize(ctx, &memorypb.MemorizeRequest{
		UserId:        req.UserID,
		Text:          req.Text,
		AssistantText: req.AssistantText,
		MemoryType:    req.MemoryType,
		Category:      req.Category,
	})
	if err != nil {
		c.logger.Error("memorize request failed", "error", err.Error())
		return nil, err
	}

	result := shared.MemorizeResponse{
		UserID:     resp.GetUserId(),
		ItemID:     resp.GetItemId(),
		Category:   resp.GetCategory(),
		Success:    resp.GetSuccess(),
		Saved:      resp.GetSaved(),
		SkipReason: resp.GetSkipReason(),
		Error:      resp.GetError(),
	}

	if !result.Success {
		errMsg := result.Error
		if errMsg == "" {
			errMsg = "sidecar returned success=false"
		}
		return nil, fmt.Errorf("memorize failed: %s", errMsg)
	}

	if !result.Saved {
		reason := result.SkipReason
		if reason == "" {
			reason = "unspecified"
		}
		c.logger.Info("memorize skipped", "reason", reason)
		return &result, nil
	}

	c.logger.Debug("memorize success", "item_id", result.ItemID)
	return &result, nil
}

// Recall retrieves relevant memories via the sidecar API.
func (c *Client) Recall(ctx context.Context, req *shared.RecallRequest) (*shared.RecallResponse, error) {
	return c.recallGRPC(ctx, req)
}

func (c *Client) recallGRPC(ctx context.Context, req *shared.RecallRequest) (*shared.RecallResponse, error) {
	runes := []rune(req.Query)
	queryPreview := string(runes[:min(50, len(runes))])
	c.logger.Debug("recall request", "user_id", req.UserID, "query", queryPreview, "method", req.Method)

	grpcClient, err := c.ensureGRPCClient(ctx)
	if err != nil {
		return nil, err
	}

	grpcReq := &memorypb.RecallRequest{
		UserId: req.UserID,
		Query:  req.Query,
		TopK:   int32(req.TopK),
		Method: req.Method,
	}

	if req.Where != nil {
		where, err := structpb.NewStruct(req.Where)
		if err != nil {
			return nil, fmt.Errorf("invalid recall where payload: %w", err)
		}
		grpcReq.Where = where
	}

	resp, err := grpcClient.Recall(ctx, grpcReq)
	if err != nil {
		c.logger.Error("recall request failed", "error", err.Error())
		return nil, err
	}

	items := make([]shared.RecallItem, 0, len(resp.GetItems()))
	for _, item := range resp.GetItems() {
		items = append(items, shared.RecallItem{
			ID:       item.GetId(),
			Content:  item.GetContent(),
			Category: item.GetCategory(),
			Salience: float64(item.GetSalience()),
		})
	}

	result := shared.RecallResponse{
		UserID:  resp.GetUserId(),
		Items:   items,
		Success: resp.GetSuccess(),
		Error:   resp.GetError(),
	}

	if !result.Success {
		errMsg := result.Error
		if errMsg == "" {
			errMsg = "sidecar returned success=false"
		}
		return nil, fmt.Errorf("recall failed: %s", errMsg)
	}

	c.logger.Debug("recall success", "items_count", len(result.Items))
	return &result, nil
}

// Health checks if the memory service is available.
func (c *Client) Health(ctx context.Context) (bool, error) {
	return c.healthGRPC(ctx)
}

func (c *Client) healthGRPC(ctx context.Context) (bool, error) {
	grpcClient, err := c.ensureGRPCClient(ctx)
	if err != nil {
		return false, err
	}

	resp, err := grpcClient.Health(ctx, &memorypb.HealthRequest{})
	if err != nil {
		return false, err
	}

	if !resp.GetHealthy() {
		if msg := strings.TrimSpace(resp.GetError()); msg != "" {
			return false, errors.New(msg)
		}
		return false, nil
	}

	return true, nil
}

func (c *Client) ensureGRPCClient(ctx context.Context) (memorypb.MemoryServiceClient, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.grpcClient != nil {
		return c.grpcClient, nil
	}

	conn, err := dialGRPCConn(ctx, c.target)
	if err != nil {
		return nil, err
	}

	c.grpcConn = conn
	c.grpcClient = memorypb.NewMemoryServiceClient(conn)
	return c.grpcClient, nil
}

func dialGRPCConn(ctx context.Context, endpoint string) (*grpc.ClientConn, error) {
	dialCtx, cancel := context.WithTimeout(ctx, grpcDialTimeout)
	defer cancel()

	options := []grpc.DialOption{
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithBlock(),
	}

	target := endpoint
	switch {
	case strings.HasPrefix(endpoint, "npipe://"):
		dialer, err := newNamedPipeDialer(endpoint)
		if err != nil {
			return nil, err
		}
		options = append(options, grpc.WithContextDialer(func(ctx context.Context, _ string) (net.Conn, error) {
			return dialer(ctx, endpoint)
		}))
		target = "passthrough:///memory-sidecar-npipe"
	case strings.HasPrefix(endpoint, "unix:"):
		socketPath, err := parseUnixEndpoint(endpoint)
		if err != nil {
			return nil, err
		}
		options = append(options, grpc.WithContextDialer(func(ctx context.Context, _ string) (net.Conn, error) {
			var d net.Dialer
			return d.DialContext(ctx, "unix", socketPath)
		}))
		target = "passthrough:///memory-sidecar-unix"
	}

	return grpc.DialContext(dialCtx, target, options...)
}

func parseUnixEndpoint(endpoint string) (string, error) {
	if strings.HasPrefix(endpoint, "unix://") {
		path := strings.TrimSpace(strings.TrimPrefix(endpoint, "unix://"))
		if path == "" {
			return "", fmt.Errorf("invalid unix endpoint %q", endpoint)
		}
		return path, nil
	}

	if strings.HasPrefix(endpoint, "unix:") {
		path := strings.TrimSpace(strings.TrimPrefix(endpoint, "unix:"))
		if path == "" {
			return "", fmt.Errorf("invalid unix endpoint %q", endpoint)
		}
		return path, nil
	}

	return "", fmt.Errorf("invalid unix endpoint %q", endpoint)
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
