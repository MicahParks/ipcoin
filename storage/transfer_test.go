package storage

import (
	"context"
	"errors"
	"net/netip"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestCreateTransfer(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatalf("Failed to begin transaction.\n  Error: %s", err)
	}
	defer tx.Rollback(ctx)

	addr1 := netip.MustParseAddr("192.168.0.1")
	addr2 := netip.MustParseAddr("192.168.0.2")
	addr3 := netip.MustParseAddr("192.168.0.3")

	expectedBalance := BalanceUntouched(now)
	for _, addr := range []netip.Addr{addr1, addr2, addr3} {
		request := GetBalanceRequest{
			Address: addr,
			Now:     now,
		}
		balance, err := GetBalance(ctx, tx, request)
		if err != nil {
			t.Fatalf("Failed to read balance.\n  Error: %s", err)
		}
		if balance != expectedBalance {
			t.Fatalf("Balance does not match expected balance.\n  Expected: %v\n  Actual: %v", expectedBalance, balance)
		}
	}

	transferAmount := int64(100)
	{
		now = now.Add(time.Minute)
		request := CreateTransferRequest{
			Amount:    transferAmount,
			Sender:    addr1,
			Now:       now,
			Recipient: addr2,
		}
		response, err := CreateTransfer(ctx, tx, request)
		if err != nil {
			t.Fatalf("Failed to transfer.\n  Error: %s", err)
		}
		if response.SenderBalance != expectedBalance-transferAmount {
			t.Fatalf("Balance does not match expected balance.\n  Expected: %v\n  Actual: %v", transferAmount, response.SenderBalance)
		}
		if response.Transfer.ID == uuid.Nil {
			t.Fatalf("ID is nil.")
		}
	}

	for _, addr := range []netip.Addr{addr1, addr2, addr3} {
		switch addr.String() {
		case "192.168.0.1":
			expectedBalance = BalanceUntouched(now) - transferAmount
		case "192.168.0.2":
			expectedBalance = BalanceUntouched(now) + transferAmount
		case "192.168.0.3":
			expectedBalance = BalanceUntouched(now)
		default:
			panic("programmer error")
		}
		request := GetBalanceRequest{
			Address: addr,
			Now:     now,
		}
		balance, err := GetBalance(ctx, tx, request)
		if err != nil {
			t.Fatalf("Failed to read balance.\n  Error: %s", err)
		}
		if balance != expectedBalance {
			t.Fatalf("Balance does not match expected balance.\n  Expected: %v\n  Actual: %v", expectedBalance, balance)
		}
	}

	transferAmount = int64(100) + BalanceUntouched(now)
	{
		now = now.Add(time.Minute)
		request := CreateTransferRequest{
			Amount:    transferAmount,
			Sender:    addr3,
			Now:       now,
			Recipient: addr1,
		}
		_, err = CreateTransfer(ctx, tx, request)
		if !errors.Is(err, ErrInsufficientBalance) {
			t.Fatalf("Should have failed transfer with insufficient balance.\n  Error: %s", err)
		}
	}
}
