package storage

import (
	"context"
	"fmt"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

var psql = sq.StatementBuilder.PlaceholderFormat(sq.Dollar)

type dbConn interface {
	Exec(ctx context.Context, sql string, arguments ...any) (pgconn.CommandTag, error)
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
	SendBatch(ctx context.Context, batch *pgx.Batch) pgx.BatchResults
}

func NewPool(ctx context.Context, dsn string) (*pgxpool.Pool, error) {
	c, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to parse PostgreSQL DSN: %w", err)
	}
	c.HealthCheckPeriod = 5 * time.Second
	c.MaxConnIdleTime = 5 * time.Minute
	c.MaxConnLifetime = 30 * time.Minute
	c.MaxConnLifetimeJitter = 5 * time.Minute
	c.MinConns = 2

	var conn *pgxpool.Pool
	const retries = 5
	for i := 0; i < retries; i++ {
		conn, err = pgxpool.NewWithConfig(ctx, c)
		if err != nil {
			return nil, fmt.Errorf("failed to create pool with given configuration: %w", err)
		}
		err = conn.Ping(ctx)
		if err != nil {
			select {
			case <-ctx.Done():
				return nil, fmt.Errorf("failed to connect to Postgres after waiting %d seconds: %w", retries, err)
			case <-time.After(time.Second):
			}
			continue
		}
		break
	}
	if err != nil {
		return nil, fmt.Errorf("failed to create database connection: %w", err)
	}

	return conn, nil
}
