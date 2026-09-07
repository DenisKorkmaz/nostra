package gateway_test

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/jackc/pgx/v5/pgxpool"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"

	systemv1 "github.com/DenisKorkmaz/nostra/gen/nostra/system/v1"
	"github.com/DenisKorkmaz/nostra/internal/gateway"
)

// databaseURL is filled by TestMain and points at the throwaway container.
var databaseURL string

// TestMain starts one PostgreSQL container for the whole package, applies the
// production migrations to it and tears it down afterwards. Sharing a single
// container keeps the suite fast; every test isolates itself by truncating.
func TestMain(m *testing.M) {
	ctx := context.Background()

	container, err := postgres.Run(ctx,
		"postgres:18-alpine",
		postgres.WithDatabase("nostra"),
		postgres.WithUsername("nostra"),
		postgres.WithPassword("test"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(90*time.Second),
		),
	)
	if err != nil {
		panic("start postgres container: " + err.Error())
	}

	databaseURL, err = container.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		panic("connection string: " + err.Error())
	}

	if err := migrate(databaseURL); err != nil {
		panic("migrate: " + err.Error())
	}

	code := m.Run()

	if err := testcontainers.TerminateContainer(container); err != nil {
		panic("terminate container: " + err.Error())
	}

	os.Exit(code)
}

// migrate applies the same goose migrations that run in production, so the test
// verifies against the real schema rather than a hand-written copy.
func migrate(url string) error {
	db, err := sql.Open("pgx", url)
	if err != nil {
		return err
	}
	defer func() { _ = db.Close() }()

	if err := goose.SetDialect("postgres"); err != nil {
		return err
	}

	goose.SetLogger(goose.NopLogger())

	return goose.Up(db, filepath.Join("..", "..", "db", "migrations"))
}

// newPool returns a pool for one test and empties ping_log beforehand. Tests in
// this package run sequentially on purpose: they share one container, so a
// parallel truncate would wipe another test's rows mid-run.
func newPool(t *testing.T) *pgxpool.Pool {
	t.Helper()

	ctx := t.Context()

	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		t.Fatalf("create pool: %v", err)
	}

	t.Cleanup(pool.Close)

	if _, err := pool.Exec(ctx, "truncate ping_log restart identity"); err != nil {
		t.Fatalf("truncate ping_log: %v", err)
	}

	return pool
}

func TestPingRecordsCallAndReportsDatabaseTime(t *testing.T) {

	pool := newPool(t)
	handler := gateway.NewSystemHandler(pool, "test-version")

	resp, err := handler.Ping(t.Context(), connect.NewRequest(&systemv1.PingRequest{Message: "hallo"}))
	if err != nil {
		t.Fatalf("ping: %v", err)
	}

	if got := resp.Msg.GetEcho(); got != "hallo" {
		t.Errorf("echo = %q, want %q", got, "hallo")
	}

	if got := resp.Msg.GetVersion(); got != "test-version" {
		t.Errorf("version = %q, want %q", got, "test-version")
	}

	if got := resp.Msg.GetPingCount(); got != 1 {
		t.Errorf("ping count = %d, want 1", got)
	}

	if !resp.Msg.GetDbTime().IsValid() {
		t.Error("db time is not set")
	}

	// The database time must come from the database, not from the process.
	// A few seconds of slack covers clock skew inside the container.
	drift := resp.Msg.GetServerTime().AsTime().Sub(resp.Msg.GetDbTime().AsTime())
	if drift < -5*time.Second || drift > 5*time.Second {
		t.Errorf("server and database time differ by %v", drift)
	}
}

func TestPingCountIncrementsPerCall(t *testing.T) {

	pool := newPool(t)
	handler := gateway.NewSystemHandler(pool, "test-version")

	for want := int64(1); want <= 3; want++ {
		resp, err := handler.Ping(t.Context(), connect.NewRequest(&systemv1.PingRequest{Message: "x"}))
		if err != nil {
			t.Fatalf("ping %d: %v", want, err)
		}

		if got := resp.Msg.GetPingCount(); got != want {
			t.Fatalf("ping count = %d, want %d", got, want)
		}
	}
}

func TestPingPersistsRow(t *testing.T) {

	pool := newPool(t)
	handler := gateway.NewSystemHandler(pool, "test-version")

	if _, err := handler.Ping(t.Context(), connect.NewRequest(&systemv1.PingRequest{Message: "persisted"})); err != nil {
		t.Fatalf("ping: %v", err)
	}

	var message string
	if err := pool.QueryRow(t.Context(), "select message from ping_log order by id desc limit 1").Scan(&message); err != nil {
		t.Fatalf("read row: %v", err)
	}

	if message != "persisted" {
		t.Errorf("stored message = %q, want %q", message, "persisted")
	}
}
