package storage

import (
	"context"
	"fmt"
	"net/netip"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/jackc/pgx/v5"
)

const feedLimit = 100

type GetFeedRequest struct {
	Now     time.Time
	Address *netip.Addr
}

type GetFeedResponse struct {
	Feed Feed
}

type Feed struct {
	Timestamp time.Time
	Comment   []Comment
	Transfer  []Transfer
}

func GetFeed(ctx context.Context, db dbConn, request GetFeedRequest) (response GetFeedResponse, err error) {
	batch := &pgx.Batch{}

	q := psql.Select("created, id, address, message").From("comment")
	if request.Address != nil {
		q = q.Where(sq.Eq{"address": request.Address})
	}
	query, args, err := q.OrderBy("created DESC").Limit(feedLimit).ToSql()
	if err != nil {
		return response, fmt.Errorf("failed to build GetFeed comment SQL query: %w", err)
	}
	batch.Queue(query, args...).Query(func(rows pgx.Rows) error {
		response.Feed.Comment, err = pgx.CollectRows(rows, pgx.RowToStructByName[Comment])
		if err != nil {
			return fmt.Errorf("failed to collect feed comments: %w", err)
		}
		return nil
	})

	//language=sql
	q = psql.Select("created, id, sender, recipient, amount").From("transfer")
	if request.Address != nil {
		q = q.Where("sender = $1 OR recipient = $1", request.Address)
	}
	query, args, err = q.OrderBy("created DESC").Limit(feedLimit).ToSql()
	batch.Queue(query, args...).Query(func(rows pgx.Rows) error {
		response.Feed.Transfer, err = pgx.CollectRows(rows, pgx.RowToStructByName[Transfer])
		if err != nil {
			return fmt.Errorf("failed to collect feed transfers: %w", err)
		}
		return nil
	})

	err = db.SendBatch(ctx, batch).Close()
	if err != nil {
		return response, fmt.Errorf("failed to collect feed: %w", err)
	}
	response.Feed.Timestamp = request.Now
	return response, nil
}
