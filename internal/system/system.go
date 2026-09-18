package system

import (
	"context"
	"fmt"

	"connectrpc.com/connect"
	"github.com/jackc/pgx/v5/pgxpool"
	"google.golang.org/protobuf/types/known/timestamppb"

	systemv1 "github.com/DenisKorkmaz/nostra/gen/nostra/system/v1"
	"github.com/DenisKorkmaz/nostra/internal/system/internal/store"
)

type Service struct {
	pool    *pgxpool.Pool
	version string
}

func New(pool *pgxpool.Pool, version string) *Service {
	return &Service{pool: pool, version: version}
}

func (s *Service) Ping(
	ctx context.Context,
	req *connect.Request[systemv1.PingRequest],
) (*connect.Response[systemv1.PingResponse], error) {
	message := req.Msg.GetMessage()

	stats, err := s.recordPing(ctx, message)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	return connect.NewResponse(&systemv1.PingResponse{
		Echo:       message,
		ServerTime: timestamppb.Now(),
		DbTime:     timestamppb.New(stats.DbTime.Time),
		PingCount:  stats.PingCount,
		Version:    s.version,
	}), nil
}

func (s *Service) recordPing(ctx context.Context, message string) (store.PingStatsRow, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return store.PingStatsRow{}, fmt.Errorf("begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	queries := store.New(tx)

	if err := queries.InsertPing(ctx, message); err != nil {
		return store.PingStatsRow{}, fmt.Errorf("insert ping: %w", err)
	}

	stats, err := queries.PingStats(ctx)
	if err != nil {
		return store.PingStatsRow{}, fmt.Errorf("read ping stats: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return store.PingStatsRow{}, fmt.Errorf("commit transaction: %w", err)
	}

	return stats, nil
}
