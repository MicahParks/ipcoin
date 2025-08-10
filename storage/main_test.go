package storage

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/MicahParks/ipcoin"
)

var (
	now  = time.Now()
	pool *pgxpool.Pool
)

func TestMain(m *testing.M) {
	ctx, cancel := context.WithTimeoutCause(context.Background(), time.Minute, errors.New("storage test main timeout"))
	defer cancel()

	var err error
	pool, err = NewPool(ctx, ipcoin.LocalDSN)
	if err != nil {
		panic(fmt.Sprintf("Failed to connect to PostgreSQL.\n  Error: %s", err))
	}
	defer pool.Close()

	m.Run()
}
