package metamask

import (
	"context"
	"encoding/json"
	"time"

	"github.com/semrush/zenrpc/v2"
	"go.uber.org/zap"
)

func zenrpcZapLoggerMiddleware(handler zenrpc.InvokeFunc) zenrpc.InvokeFunc {
	return func(ctx context.Context, method string, params json.RawMessage) zenrpc.Response {
		var (
			start = time.Now()
			ip    = "<nil>"
		)
		if req, ok := zenrpc.RequestFromContext(ctx); ok && req != nil {
			ip = req.RemoteAddr
		}
		response := handler(ctx, method, params)
		zap.S().Debugf(
			"MetaMaskRPC: ip=%s method=%s.%s duration=%v params=%s err=%v",
			ip, zenrpc.NamespaceFromContext(ctx), method, time.Since(start), params, response.Error,
		)
		return response
	}
}
