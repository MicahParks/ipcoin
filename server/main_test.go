package server

import (
	"context"
	"log/slog"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/MicahParks/ipcoin"
	"github.com/MicahParks/ipcoin/ctxkey"
	"github.com/MicahParks/ipcoin/storage"
)

var (
	now  = time.Now()
	pool *pgxpool.Pool
	s    *server
)

func TestMain(m *testing.M) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	l := slog.Default()
	ctx = context.WithValue(ctx, ctxkey.Logger, l.With("defaultLogger", true))

	var err error
	pool, err = storage.NewPool(ctx, ipcoin.LocalDSN)
	if err != nil {
		l.ErrorContext(ctx, "Failed to connect to PostgreSQL.",
			ipcoin.LogErr, err,
		)
		return
	}
	defer pool.Close()

	s = New(ctx, ipcoin.Config{}, NewFakeClock(now), l, leaderboardNoOp{}, pool, "").(*server)

	m.Run()
}

func addTx(ctx context.Context, t *testing.T) (context.Context, pgx.Tx) {
	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatalf("Failed to begin transaction.  Error: %s", err)
	}
	return context.WithValue(ctx, ctxkey.TestingTx, tx), tx
}
