package storage

import (
	"context"
	"fmt"
	"net/netip"
	"sync/atomic"
	"time"

	"github.com/jackc/pgx/v5"
)

type GetBalanceRequest struct {
	Address netip.Addr
	Now     time.Time
}

func GetBalance(ctx context.Context, db dbConn, request GetBalanceRequest) (int64, error) {
	batch := &pgx.Batch{}

	balanceDiff := &atomic.Int64{}
	addBalanceDiff(request.Address, batch, balanceDiff)

	err := db.SendBatch(ctx, batch).Close()
	if err != nil {
		return 0, fmt.Errorf("failed to check transfers for balance: %w", err)
	}
	available := BalanceUntouched(request.Now) + balanceDiff.Load()
	return available, nil
}

var projectStarted = time.Date(2025, 8, 10, 21, 0, 0, 0, time.UTC)

func BalanceUntouched(now time.Time) int64 {
	return int64(now.Sub(projectStarted) / time.Hour)
}

func addBalanceDiff(address netip.Addr, batch *pgx.Batch, balanceDiff *atomic.Int64) {
	//language=sql
	query := `
SELECT COALESCE(SUM(amount), 0)
FROM transfer
WHERE recipient = $1
`
	batch.Queue(query, address).QueryRow(func(row pgx.Row) error {
		var credits int64
		err := row.Scan(&credits)
		if err != nil {
			return fmt.Errorf("failed to check transfers of credit: %w", err)
		}
		balanceDiff.Add(credits)
		return nil
	})

	//language=sql
	query = `
SELECT COALESCE(SUM(amount), 0)
FROM transfer
WHERE sender = $1
`
	batch.Queue(query, address).QueryRow(func(row pgx.Row) error {
		var debits int64
		err := row.Scan(&debits)
		if err != nil {
			return fmt.Errorf("failed to check transfers of debit: %w", err)
		}
		balanceDiff.Add(-1 * debits)
		return nil
	})
}
