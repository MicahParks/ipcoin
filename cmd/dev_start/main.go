package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/netip"
	"os"
	"time"

	"github.com/MicahParks/ipcoin"
	"github.com/MicahParks/ipcoin/ctxkey"
	"github.com/MicahParks/ipcoin/storage"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	l := slog.Default()
	ctx = context.WithValue(ctx, ctxkey.Logger, l.With("defaultLogger", true))

	c, err := config()
	if err != nil {
		l.ErrorContext(ctx, "Failed to read config.",
			ipcoin.LogErr, err,
		)
		return
	}

	l.InfoContext(ctx, "Connecting to PostgreSQL.")
	pool, err := storage.NewPool(ctx, c.DBDSN)
	if err != nil {
		l.ErrorContext(ctx, "Failed to connect to PostgreSQL.",
			ipcoin.LogErr, err,
		)
		return
	}
	defer pool.Close()
	l.InfoContext(ctx, "Connected to PostgreSQL.")

	tx, err := pool.Begin(ctx)
	if err != nil {
		l.ErrorContext(ctx, "Failed to begin transaction.",
			ipcoin.LogErr, err,
		)
		return
	}
	defer tx.Rollback(ctx)

	now := time.Now().Add(-7 * 24 * time.Hour)
	commentRequest := storage.CreateCommentRequest{
		Addr:    netip.MustParseAddr("1.1.1.1"),
		Message: "First comment.",
		Now:     now,
	}
	_, err = storage.CreateComment(ctx, tx, commentRequest)
	if err != nil {
		l.ErrorContext(ctx, "Failed to write comment.",
			ipcoin.LogErr, err,
		)
		return
	}
	now = now.Add(24 * time.Hour)
	commentRequest = storage.CreateCommentRequest{
		Addr:    netip.MustParseAddr("8faf:8a80:a716:ae7a:93d3:9a98:8d6a:7946"),
		Message: "A really really really really really really really really really really really really really really really really really really really really really really really really really really really long comment.",
		Now:     now,
	}
	_, err = storage.CreateComment(ctx, tx, commentRequest)
	if err != nil {
		l.ErrorContext(ctx, "Failed to write comment.",
			ipcoin.LogErr, err,
		)
		return
	}
	now.Add(time.Minute)
	transferRequest := storage.CreateTransferRequest{
		Amount:    109_293,
		Sender:    netip.MustParseAddr("8faf:8a80:a716:ae7a:93d3:9a98:8d6a:7946"),
		Now:       now,
		Recipient: netip.MustParseAddr("49de:1ed6:37f3:ec53:c590:12f6:f10c:092c"),
	}
	_, err = storage.CreateTransfer(ctx, tx, transferRequest)
	if err != nil {
		l.ErrorContext(ctx, "Failed to write transfer.",
			ipcoin.LogErr, err,
		)
		return
	}

	err = tx.Commit(ctx)
	if err != nil {
		l.ErrorContext(ctx, "Failed to commit transaction.",
			ipcoin.LogErr, err,
		)
		return
	}

	l.InfoContext(ctx, "Dev data added.")
}

func config() (ipcoin.Config, error) {
	b, err := os.ReadFile("config.json")
	if err != nil {
		return ipcoin.Config{}, fmt.Errorf("failed to read config JSON file: %w", err)
	}
	var c ipcoin.Config
	err = json.Unmarshal(b, &c)
	if err != nil {
		return ipcoin.Config{}, fmt.Errorf("failed to unmarshal config JSON: %w", err)
	}
	if c.DBDSN == "" {
		return ipcoin.Config{}, errors.New("config.json must contain a database DSN")
	}
	return c, nil
}
