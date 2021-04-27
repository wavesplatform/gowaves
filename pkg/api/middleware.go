package api

import (
	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	"go.uber.org/zap"
	"net/http"
	"strconv"
	"time"
)

// LoggerMiddleware is a middleware that logs the start and end of each request, along
// with some useful data about what was requested, what the response status was,
// and how long it took to return.
func LoggerMiddleware(l *zap.Logger) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		fn := func(w http.ResponseWriter, r *http.Request) {
			ww, ok := w.(middleware.WrapResponseWriter)
			if !ok {
				ww = middleware.NewWrapResponseWriter(w, r.ProtoMajor)
			}

			t1 := time.Now()
			defer func() {
				l.Info("ServedHttpRequest",
					zap.String("proto", r.Proto),
					zap.String("path", r.URL.Path),
					zap.Duration("lat", time.Since(t1)),
					zap.Int("status", ww.Status()),
					zap.Int("size", ww.BytesWritten()),
					zap.String("reqId", middleware.GetReqID(r.Context())))
			}()

			next.ServeHTTP(ww, r)
		}
		return http.HandlerFunc(fn)
	}
}

func chiHttpApiGeneralMetricsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		begin := time.Now()

		metricApiTotalRequests.Inc()

		ww, ok := w.(middleware.WrapResponseWriter)
		if !ok {
			ww = middleware.NewWrapResponseWriter(w, r.ProtoMajor)
		}

		defer func() {
			routePath := r.URL.Path
			if chiRouteContext := chi.RouteContext(r.Context()); chiRouteContext != nil {
				if updatedRoutePath := chiRouteContext.RoutePattern(); updatedRoutePath != "" {
					routePath = updatedRoutePath
				}
			}

			statusCode := ww.Status()
			metricApiHits.WithLabelValues(strconv.Itoa(statusCode), routePath).Inc()

			observer := metricApiRequestDuration.WithLabelValues(r.Method, routePath)
			observer.Observe(time.Since(begin).Seconds())
		}()

		next.ServeHTTP(ww, r)
	})
}

func createHeadersMiddleware(headers map[string]string) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			for k, v := range headers {
				w.Header().Set(k, v)
			}
			next.ServeHTTP(w, r)
		})
	}
}

func jsonContentTypeMiddleware(next http.Handler) http.Handler {
	return createHeadersMiddleware(map[string]string{
		"Content-Type": "application/json",
	})(next)
}
