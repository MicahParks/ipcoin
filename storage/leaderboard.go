package storage

import (
	"context"
	"fmt"
	"net/netip"
	"time"

	"github.com/jackc/pgx/v5"
)

const leaderboardLimit = 10

type GetLeaderboardRequest struct {
	Now time.Time
}

type GetLeaderboardResponse struct {
	LeaderboardBalance  []Glance
	LeaderboardTransfer []Glance
}

type LeaderboardGlance struct {
	Address       netip.Addr `db:"address"`
	BalanceDiff   int64      `db:"balance_diff"`
	CommentCount  int64      `db:"comment_count"`
	TransferCount int64      `db:"transfer_count"`
}

func RefreshLeaderboard(ctx context.Context, db dbConn, concurrently bool) (err error) {
	//language=sql
	query := `
REFRESH MATERIALIZED VIEW CONCURRENTLY leaderboard_glance;
`
	if !concurrently {
		query = `
REFRESH MATERIALIZED VIEW leaderboard_glance;
`
	}
	_, err = db.Exec(ctx, query)
	if err != nil {
		return fmt.Errorf("failed to refresh leaderboard: %w", err)
	}
	return nil
}

func GetLeaderboard(ctx context.Context, db dbConn, request GetLeaderboardRequest) (response GetLeaderboardResponse, untouchedBalance int64, err error) {
	batch := &pgx.Batch{}

	untouchedBalance = BalanceUntouched(request.Now)

	//language=sql
	query := `
SELECT address, balance_diff, comment_count, transfer_count
FROM leaderboard_glance
ORDER BY balance_diff DESC
`
	batch.Queue(query + fmt.Sprintf(" LIMIT %d", leaderboardLimit)).Query(func(rows pgx.Rows) error {
		lGlances, err := pgx.CollectRows(rows, pgx.RowToStructByName[LeaderboardGlance])
		if err != nil {
			return fmt.Errorf("failed to collect leaderboard balance: %w", err)
		}
		response.LeaderboardBalance = make([]Glance, len(lGlances))
		for i, g := range lGlances {
			response.LeaderboardBalance[i] = Glance{
				Address:          g.Address,
				BalanceAvailable: untouchedBalance + g.BalanceDiff,
				CommentCount:     g.CommentCount,
				TransferCount:    g.TransferCount,
			}
		}
		return nil
	})

	//language=sql
	query = `
SELECT address, balance_diff, comment_count, transfer_count
FROM leaderboard_glance
ORDER BY transfer_count DESC
`
	batch.Queue(query + fmt.Sprintf(" LIMIT %d", leaderboardLimit)).Query(func(rows pgx.Rows) error {
		lGlances, err := pgx.CollectRows(rows, pgx.RowToStructByName[LeaderboardGlance])
		if err != nil {
			return fmt.Errorf("failed to collect leaderboard transfer: %w", err)
		}
		response.LeaderboardTransfer = make([]Glance, len(lGlances))
		for i, g := range lGlances {
			response.LeaderboardTransfer[i] = Glance{
				Address:          g.Address,
				BalanceAvailable: untouchedBalance + g.BalanceDiff,
				CommentCount:     g.CommentCount,
				TransferCount:    g.TransferCount,
			}
		}
		return nil
	})

	err = db.SendBatch(ctx, batch).Close()
	if err != nil {
		return response, 0, fmt.Errorf("failed to collect leaderboard: %w", err)
	}
	return response, untouchedBalance, nil
}
