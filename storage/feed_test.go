package storage

import (
	"context"
	"net/netip"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestGetFeed(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatalf("Failed to begin transaction.\n  Error: %s", err)
	}
	defer tx.Rollback(ctx)
	// var err error
	// tx := pool

	ipA := netip.MustParseAddr("192.168.0.1")
	ipB := netip.MustParseAddr("192.168.0.2")
	ipC := netip.MustParseAddr("192.168.0.3")
	commentA := "test comment A"
	commentC := "test comment C"

	tNow := now
	tRequest := CreateTransferRequest{
		Amount:    1,
		Sender:    ipA,
		Now:       now,
		Recipient: ipB,
	}
	_, err = CreateTransfer(ctx, tx, tRequest)
	if err != nil {
		t.Fatalf("Failed to transfer.\n  Error: %s", err)
	}
	tNow = tNow.Add(time.Minute)
	tRequest = CreateTransferRequest{
		Amount:    2,
		Sender:    ipB,
		Now:       tNow,
		Recipient: ipC,
	}
	_, err = CreateTransfer(ctx, tx, tRequest)
	if err != nil {
		t.Fatalf("Failed to transfer.\n  Error: %s", err)
	}
	tNow = tNow.Add(time.Minute)
	cRequest := CreateCommentRequest{
		Addr:    ipC,
		Message: commentC,
		Now:     tNow,
	}
	_, err = CreateComment(ctx, tx, cRequest)
	if err != nil {
		t.Fatalf("Failed to write comment.\n  Error: %s", err)
	}
	tNow = tNow.Add(time.Minute)
	tRequest = CreateTransferRequest{
		Amount:    3,
		Sender:    ipC,
		Now:       tNow,
		Recipient: ipA,
	}
	_, err = CreateTransfer(ctx, tx, tRequest)
	if err != nil {
		t.Fatalf("Failed to transfer.\n  Error: %s", err)
	}
	tNow = tNow.Add(time.Minute)
	cRequest = CreateCommentRequest{
		Addr:    ipA,
		Message: commentA,
		Now:     tNow,
	}
	_, err = CreateComment(ctx, tx, cRequest)
	if err != nil {
		t.Fatalf("Failed to write comment.\n  Error: %s", err)
	}

	tNow = tNow.Add(time.Minute)
	request := GetFeedRequest{
		Now: tNow,
	}
	response, err := GetFeed(ctx, tx, request)
	if err != nil {
		t.Fatalf("Failed to read feed.\n  Error: %s", err)
	}
	if !response.Feed.Timestamp.Equal(tNow) {
		t.Fatalf("Feed timestamp incorrect.")
	}
	if len(response.Feed.Comment) != 2 {
		t.Fatalf("Comment feed should have 2 entries but has %d.", len(response.Feed.Comment))
	}
	if len(response.Feed.Transfer) != 3 {
		t.Fatalf("Transfer feed should have 3 entries but has %d.", len(response.Feed.Transfer))
	}
	comment := response.Feed.Comment[0]
	feedCommentCheck(t, "Comment[0]: ", comment, now.Add(4*time.Minute), ipA, commentA)
	transfer := response.Feed.Transfer[0]
	feedTransferCheck(t, "Transfer[0]: ", transfer, now.Add(3*time.Minute), ipC, ipA, 3)
	comment = response.Feed.Comment[1]
	feedCommentCheck(t, "Comment[1]: ", comment, now.Add(2*time.Minute), ipC, commentC)
	transfer = response.Feed.Transfer[1]
	feedTransferCheck(t, "Transfer[1]: ", transfer, now.Add(time.Minute), ipB, ipC, 2)
	transfer = response.Feed.Transfer[2]
	feedTransferCheck(t, "Transfer[2]: ", transfer, now, ipA, ipB, 1)
}

func TestGetFeed_Empty(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatalf("Failed to begin transaction.\n  Error: %s", err)
	}
	defer tx.Rollback(ctx)

	request := GetFeedRequest{
		Now: now,
	}
	feed, err := GetFeed(ctx, tx, request)
	if err != nil {
		t.Fatalf("Failed to read feed.\n  Error: %s", err)
	}

	if !feed.Feed.Timestamp.Equal(request.Now) { // TODO Test timestamp/created in other storage operations.
		t.Fatalf("Feed timestamp incorrect.")
	}
	if len(feed.Feed.Comment) != 0 {
		t.Fatalf("Comment feed should be empty.")
	}
	if len(feed.Feed.Transfer) != 0 {
		t.Fatalf("Transfer feed should be empty.")
	}
}

func feedCommentCheck(t *testing.T, prefix string, comment Comment, created time.Time, address netip.Addr, message string) {
	if comment.ID == uuid.Nil {
		t.Fatal(prefix + "Comment ID must be non-nil.")
	}
	if !comment.Created.Equal(created.Truncate(time.Microsecond)) {
		t.Log(created.Sub(comment.Created).String())
		t.Fatal(prefix + "Incorrect comment created time.")
	}
	if comment.Address != address {
		t.Fatal(prefix + "Incorrect comment address.")
	}
	if comment.Message != message {
		t.Fatal(prefix + "Incorrect comment message.")
	}
}

func feedTransferCheck(t *testing.T, prefix string, transfer Transfer, created time.Time, sender, recipient netip.Addr, amount int64) {
	if transfer.ID == uuid.Nil {
		t.Fatal(prefix + "Transfer ID must be non-nil.")
	}
	if !transfer.Created.Equal(created.Truncate(time.Microsecond)) {
		t.Fatal(prefix + "Incorrect transfer created time.")
	}
	if transfer.Sender != sender {
		t.Fatal(prefix + "Incorrect transfer sender.")
	}
	if transfer.Recipient != recipient {
		t.Fatal(prefix + "Incorrect transfer recipient.")
	}
	if transfer.Amount != amount {
		t.Fatal(prefix + "Incorrect transfer amount.")
	}
}
