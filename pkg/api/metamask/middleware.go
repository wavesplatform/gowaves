package metamask

import (
	"context"
	"encoding/json"
	"log/slog"
	"time"

	"github.com/semrush/zenrpc/v2"
)

func APILogMiddleware(handler zenrpc.InvokeFunc) zenrpc.InvokeFunc {
	return func(ctx context.Context, method string, params json.RawMessage) zenrpc.Response {
		var (
			start = time.Now()
			ip    = "<nil>"
		)
		if req, ok := zenrpc.RequestFromContext(ctx); ok && req != nil {
			ip = req.RemoteAddr
		}
		response := handler(ctx, method, params)
		slog.Debug("MetaMaskRPC", "ip", ip, "ns", zenrpc.NamespaceFromContext(ctx), "method", method,
			"duration", time.Since(start), "params", params, "response", response.JSON())
		return response
	}
}
