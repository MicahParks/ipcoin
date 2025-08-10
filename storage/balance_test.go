package storage

import (
	"context"
	"net/netip"
	"testing"
	"time"
)

func TestGetBalanceUntouched(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatalf("Failed to begin transaction.\n  Error: %s", err)
	}
	defer tx.Rollback(ctx)

	expectedBalance := BalanceUntouched(now)

	request := GetBalanceRequest{
		Address: netip.Addr{},
		Now:     now,
	}
	balance, err := GetBalance(ctx, tx, request)
	if err != nil {
		t.Fatalf("Failed to read balance.\n  Error: %s", err)
	}
	if balance != expectedBalance {
		t.Errorf("GetBalance returned unexpected balance.\n  Expected: %v\n  Actual: %v", expectedBalance, balance)
	}
}
