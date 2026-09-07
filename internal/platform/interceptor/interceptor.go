package interceptor

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"log/slog"
	"time"

	"connectrpc.com/connect"
)

const HeaderRequestID = "X-Request-Id"

type requestIDKey struct{}

func RequestID(ctx context.Context) string {
	id, _ := ctx.Value(requestIDKey{}).(string)
	return id
}

func NewRequestID() connect.UnaryInterceptorFunc {
	return func(next connect.UnaryFunc) connect.UnaryFunc {
		return func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
			id := req.Header().Get(HeaderRequestID)
			if id == "" {
				id = generateID()
			}

			ctx = context.WithValue(ctx, requestIDKey{}, id)

			resp, err := next(ctx, req)
			if resp != nil {
				resp.Header().Set(HeaderRequestID, id)
			}

			return resp, err
		}
	}
}

func NewLogging(logger *slog.Logger) connect.UnaryInterceptorFunc {
	return func(next connect.UnaryFunc) connect.UnaryFunc {
		return func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
			start := time.Now()
			resp, err := next(ctx, req)

			attrs := []any{
				slog.String("procedure", req.Spec().Procedure),
				slog.Duration("duration", time.Since(start)),
				slog.String("request_id", RequestID(ctx)),
			}

			if err != nil {
				attrs = append(attrs,
					slog.String("code", connect.CodeOf(err).String()),
					slog.String("error", err.Error()),
				)
				logger.ErrorContext(ctx, "rpc failed", attrs...)

				return resp, err
			}

			logger.InfoContext(ctx, "rpc handled", attrs...)

			return resp, nil
		}
	}
}

func generateID() string {
	var buf [8]byte
	if _, err := rand.Read(buf[:]); err != nil {
		return "unknown"
	}

	return hex.EncodeToString(buf[:])
}
