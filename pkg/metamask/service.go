package metamask

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/pkg/errors"
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

func RunMetaMaskService(ctx context.Context, address string, service RPCService, enableRpcServiceLog bool) error {
	// TODO(nickeskov): what about `BatchMaxLen` option?
	rpc := zenrpc.NewServer(zenrpc.Options{ExposeSMD: true, AllowCORS: true})
	rpc.Register("", service) // public

	if enableRpcServiceLog {
		rpc.Use(zenrpcZapLoggerMiddleware)
	}

	http.Handle("/eth", rpc)

	server := &http.Server{Addr: address, Handler: nil}

	go func() {
		<-ctx.Done()
		zap.S().Info("shutting down metamask service...")
		err := server.Shutdown(ctx)
		if err != nil && !errors.Is(err, context.Canceled) {
			zap.S().Errorf("failed to shutdown metamask service: %v", err)
		}
	}()
	err := server.ListenAndServe()

	if err != nil && err != http.ErrServerClosed {
		return err
	}

	return nil
}
