package storage

import (
	"context"
	"fmt"
	"net/netip"
	"sync/atomic"
	"time"

	"github.com/jackc/pgx/v5"
)

type Glance struct {
	Address          netip.Addr
	BalanceAvailable int64
	CommentCount     int64
	TransferCount    int64
}

type GetGlanceRequest struct {
	Address netip.Addr
	Now     time.Time
}

func GetGlance(ctx context.Context, db dbConn, request GetGlanceRequest) (Glance, int64, error) {
	batch := &pgx.Batch{}

	balanceDiff := &atomic.Int64{}
	addBalanceDiff(request.Address, batch, balanceDiff)

	//language=sql
	query := `
SELECT COUNT(*)
FROM comment
WHERE address = $1
`
	var commentCount int64
	batch.Queue(query, request.Address).QueryRow(func(row pgx.Row) error {
		err := row.Scan(&commentCount)
		if err != nil {
			return fmt.Errorf("failed to count comments: %w", err)
		}
		return nil
	})

	//language=sql
	query = `
SELECT COUNT(*)
FROM transfer
WHERE recipient = $1
   OR sender = $1
`
	var transferCount int64
	batch.Queue(query, request.Address).QueryRow(func(row pgx.Row) error {
		err := row.Scan(&transferCount)
		if err != nil {
			return fmt.Errorf("failed to count transfers: %w", err)
		}
		return nil
	})

	err := db.SendBatch(ctx, batch).Close()
	if err != nil {
		return Glance{}, 0, err
	}
	balanceUntouched := BalanceUntouched(request.Now)
	glance := Glance{
		Address:          request.Address,
		BalanceAvailable: balanceUntouched + balanceDiff.Load(),
		CommentCount:     commentCount,
		TransferCount:    transferCount,
	}
	return glance, balanceUntouched, nil
}
