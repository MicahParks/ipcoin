package storage

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type CommentModeration struct {
	Created   time.Time
	ID        uuid.UUID
	CommentID uuid.UUID
	Censored  bool
	Note      string
}

func CreateCommentModeration(ctx context.Context, db dbConn, moderations []CommentModeration, now time.Time) error {
	batch := &pgx.Batch{}
	//language=sql
	query := `
INSERT INTO comment_moderation (created, id, comment_id, censored, note)
VALUES ($1, $2, $3, $4, $5)
RETURNING created, id
`
	for _, m := range moderations {
		u := uuid.New()
		batch.Queue(query, now, u, m.CommentID, m.Censored, m.Note)
	}
	err := db.SendBatch(ctx, batch).Close()
	if err != nil {
		return fmt.Errorf("failed to create comment moderation: %w", err)
	}
	return nil
}

func ReadCommentUnmoderated(ctx context.Context, db dbConn) ([]Comment, error) {
	//language=sql
	query := `
SELECT DISTINCT ON (c.id) c.created, c.id, c.address, c.message
FROM comment c
         LEFT JOIN comment_moderation m ON c.id = m.comment_id
WHERE m.id IS NULL
` // TODO Add limit?
	rows, err := db.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to read unmoderated comments: %w", err)
	}
	defer rows.Close()
	unmoderated := make([]Comment, 0)
	for rows.Next() {
		var c Comment
		err = rows.Scan(&c.Created, &c.ID, &c.Address, &c.Message)
		if err != nil {
			return nil, fmt.Errorf("failed to scan unmoderated comments: %w", err)
		}
		unmoderated = append(unmoderated, c)
	}
	err = rows.Err()
	if err != nil {
		return nil, fmt.Errorf("failed to iterate over unmoderated comments rows: %w", err)
	}
	return unmoderated, nil
}

func ReadCommentCensored(ctx context.Context, db dbConn, commentIDs []uuid.UUID) ([]uuid.UUID, error) {
	//language=sql
	query := `
SELECT DISTINCT comment_id
FROM comment_moderation
WHERE comment_id = ANY ($1)
  AND censored IS TRUE
`
	rows, err := db.Query(ctx, query, commentIDs)
	if err != nil {
		return nil, fmt.Errorf("failed to read comment moderation: %w", err)
	}
	defer rows.Close()

	censored := make([]uuid.UUID, 0)
	for rows.Next() {
		var u uuid.UUID
		err = rows.Scan(&u)
		if err != nil {
			return nil, fmt.Errorf("failed to scan comment moderation: %w", err)
		}
		censored = append(censored, u)
	}
	err = rows.Err()
	if err != nil {
		return nil, fmt.Errorf("failed to iterate over comment moderation rows: %w", err)
	}

	return censored, nil
}
