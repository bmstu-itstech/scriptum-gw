package logs

import (
	"log/slog"
	"net/http"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
)

type Middleware struct {
	l *slog.Logger
}

func NewMiddleware(l *slog.Logger) *Middleware {
	return &Middleware{l: l}
}

func (m *Middleware) Handler(next runtime.HandlerFunc) runtime.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request, pathParams map[string]string) {
		l := m.l.With(
			slog.String("method", r.Method),
			slog.String("path", r.URL.Path),
		)
		l.Info("handle request")
		next(w, r, pathParams)
		l.Info("request handled")
	}
}
