//go:build !windows

package memory

import (
	"context"
	"fmt"
	"net"
)

func newNamedPipeDialer(_ string) (func(context.Context, string) (net.Conn, error), error) {
	return nil, fmt.Errorf("npipe endpoint is only supported on windows")
}
