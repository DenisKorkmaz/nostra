package server

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"connectrpc.com/connect"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"

	"github.com/DenisKorkmaz/nostra/gen/nostra/system/v1/systemv1connect"
	"github.com/DenisKorkmaz/nostra/internal/platform/interceptor"
)

const readyTimeout = 2 * time.Second

func NewHandler(pool *pgxpool.Pool, logger *slog.Logger, version string) http.Handler {
	mux := http.NewServeMux()

	path, handler := systemv1connect.NewSystemServiceHandler(
		NewSystemHandler(pool, version),
		connect.WithInterceptors(
			interceptor.NewRequestID(),
			interceptor.NewLogging(logger),
		),
		connect.WithRecover(recoverPanic(logger)),
	)
	mux.Handle(path, handler)

	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) {
		writePlain(w, http.StatusOK, "ok")
	})

	mux.HandleFunc("GET /readyz", func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), readyTimeout)
		defer cancel()

		if err := pool.Ping(ctx); err != nil {
			logger.WarnContext(ctx, "readiness check failed", slog.String("error", err.Error()))
			writePlain(w, http.StatusServiceUnavailable, "database unavailable")

			return
		}

		writePlain(w, http.StatusOK, "ready")
	})

	return h2c.NewHandler(mux, &http2.Server{})
}

func recoverPanic(logger *slog.Logger) func(context.Context, connect.Spec, http.Header, any) error {
	return func(ctx context.Context, spec connect.Spec, _ http.Header, panicValue any) error {
		logger.ErrorContext(ctx, "recovered from panic",
			slog.String("procedure", spec.Procedure),
			slog.Any("panic", panicValue),
		)

		return connect.NewError(connect.CodeInternal, errors.New("internal error"))
	}
}

func writePlain(w http.ResponseWriter, status int, body string) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(status)
	_, _ = w.Write([]byte(body))
}
