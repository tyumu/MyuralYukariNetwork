//go:build windows

package memory

import (
	"context"
	"fmt"
	"net"
	"strings"

	"github.com/Microsoft/go-winio"
)

func newNamedPipeDialer(endpoint string) (func(context.Context, string) (net.Conn, error), error) {
	pipePath, err := normalizePipePath(endpoint)
	if err != nil {
		return nil, err
	}

	return func(ctx context.Context, _ string) (net.Conn, error) {
		return winio.DialPipeContext(ctx, pipePath)
	}, nil
}

func normalizePipePath(endpoint string) (string, error) {
	if strings.TrimSpace(endpoint) == "" {
		return "", fmt.Errorf("empty npipe endpoint")
	}

	if strings.HasPrefix(endpoint, `\\.\pipe\`) {
		return endpoint, nil
	}

	if !strings.HasPrefix(endpoint, "npipe://") {
		return "", fmt.Errorf("invalid npipe endpoint: %q", endpoint)
	}

	trimmed := strings.TrimPrefix(endpoint, "npipe://")
	trimmed = strings.TrimLeft(trimmed, "/")
	if trimmed == "" {
		return "", fmt.Errorf("invalid npipe endpoint: %q", endpoint)
	}

	trimmed = strings.ReplaceAll(trimmed, "/", `\`)
	if strings.HasPrefix(trimmed, `.\pipe\`) {
		return `\\` + trimmed, nil
	}

	if strings.HasPrefix(trimmed, `\\.\pipe\`) {
		return trimmed, nil
	}

	return "", fmt.Errorf("invalid npipe endpoint %q; expected npipe:////./pipe/<name>", endpoint)
}
