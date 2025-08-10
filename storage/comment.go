package storage

import (
	"context"
	"fmt"
	"net/netip"
	"time"

	"github.com/google/uuid"
)

type Comment struct {
	Created time.Time  `db:"created"`
	ID      uuid.UUID  `db:"id"`
	Address netip.Addr `db:"address"`
	Message string     `db:"message"`
}

type CreateCommentRequest struct {
	Addr    netip.Addr
	Message string
	Now     time.Time
}

type CreateCommentResponse struct {
	Comment Comment
}

func CreateComment(ctx context.Context, db dbConn, request CreateCommentRequest) (CreateCommentResponse, error) {
	query := `
INSERT INTO comment (created, id, address, message)
VALUES ($1, $2, $3, $4)
`
	id := uuid.New()
	_, err := db.Exec(ctx, query, request.Now, id, request.Addr, request.Message)
	if err != nil {
		return CreateCommentResponse{}, fmt.Errorf("failed to insert new comment: %w", err)
	}
	response := CreateCommentResponse{
		Comment: Comment{
			Created: request.Now,
			ID:      id,
			Address: request.Addr,
			Message: request.Message,
		},
	}
	return response, nil
}
