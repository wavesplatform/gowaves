package api

import (
	"github.com/go-chi/chi"
	"go.uber.org/zap"
	"net/http"
	"strconv"
	"time"
)

type Middleware = func(next http.Handler) http.Handler

type responseWriterWrapper struct {
	http.ResponseWriter
	statusCode    int
	length        int
	headerWritten bool
}

func newResponseWriterWrapper(inner http.ResponseWriter, defaultStatusCode int) *responseWriterWrapper {
	return &responseWriterWrapper{
		ResponseWriter: inner,
		statusCode:     defaultStatusCode,
		length:         0,
		headerWritten:  false,
	}
}

func (w *responseWriterWrapper) WriteHeader(status int) {
	if w.headerWritten {
		zap.S().Warn("WriteHeader called more than once")
		return
	}
	w.statusCode = status
	w.ResponseWriter.WriteHeader(status)
	w.headerWritten = true
}

func (w *responseWriterWrapper) Write(data []byte) (int, error) {
	n, err := w.ResponseWriter.Write(data)
	if err == nil {
		w.length += n
	}
	return n, err
}

func (w *responseWriterWrapper) GetStatusCode() int {
	return w.statusCode
}

func (w *responseWriterWrapper) GetResponseLength() int {
	return w.length
}

func chiHttpApiGeneralMetricsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		begin := time.Now()

		metricApiTotalRequests.Inc()

		newWriter := newResponseWriterWrapper(w, http.StatusOK)

		defer func() {
			routePath := r.URL.Path
			if chiRouteContext := chi.RouteContext(r.Context()); chiRouteContext != nil {
				if updatedRoutePath := chiRouteContext.RoutePattern(); updatedRoutePath != "" {
					routePath = updatedRoutePath
				}
			}

			statusCode := newWriter.GetStatusCode()
			metricApiHits.WithLabelValues(strconv.Itoa(statusCode), routePath).Inc()

			observer := metricApiRequestDuration.WithLabelValues(r.Method, routePath)
			observer.Observe(time.Since(begin).Seconds())
		}()

		next.ServeHTTP(newWriter, r)
	})
}

func panicMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if e := recover(); e != nil {
				http.Error(
					w,
					http.StatusText(http.StatusInternalServerError),
					http.StatusInternalServerError,
				)
				zap.S().Errorf("panic: %+v", e)
			}
		}()

		next.ServeHTTP(w, r)
	})
}

func createHeadersMiddleware(headers map[string]string) Middleware {
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
