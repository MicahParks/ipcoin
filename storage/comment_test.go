package storage

import (
	"context"
	"net/netip"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestWriteComment(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatalf("Failed to begin transaction.\n  Error: %s", err)
	}
	defer tx.Rollback(ctx)

	request := CreateCommentRequest{
		Addr:    netip.MustParseAddr("192.168.0.1"),
		Message: "test",
		Now:     now,
	}
	response, err := CreateComment(ctx, tx, request)
	if err != nil {
		t.Fatalf("Failed to write comment.\n  Error: %s", err)
	}
	if response.Comment.ID == uuid.Nil {
		t.Fatalf("Comment ID is nil.")
	}
}
