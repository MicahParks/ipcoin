package storage

import (
	"context"
	"fmt"
	"net/netip"
	"strings"
	"testing"
	"time"
)

func TestLeaderboard(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatalf("Failed to begin transaction.\n  Error: %s", err)
	}
	defer tx.Rollback(ctx)

	untouched := BalanceUntouched(now)

	addrs := make([]netip.Addr, leaderboardLimit)
	for i := range leaderboardLimit {
		addrs[i] = netip.MustParseAddr(fmt.Sprintf("192.168.0.%d", i+1))
	}

	ip1 := netip.MustParseAddr("::1")
	ip2 := netip.MustParseAddr("::2")
	for _, addr := range addrs {
		tRequest := CreateTransferRequest{
			Amount:    1,
			Sender:    addr,
			Now:       now,
			Recipient: ip1,
		}
		_, err = CreateTransfer(ctx, tx, tRequest)
		if err != nil {
			t.Fatalf("Failed to transfer.\n  Error: %s", err)
		}
	}
	for _, addr := range addrs {
		tRequest := CreateTransferRequest{
			Amount:    2,
			Sender:    addr,
			Now:       now,
			Recipient: ip2,
		}
		_, err = CreateTransfer(ctx, tx, tRequest)
		if err != nil {
			t.Fatalf("Failed to transfer.\n  Error: %s", err)
		}
	}

	err = RefreshLeaderboard(ctx, tx, false)
	if err != nil {
		t.Fatalf("Failed to refresh leaderboard.\n  Error: %s", err)
	}
	request := GetLeaderboardRequest{
		Now: now,
	}
	response, _, err := GetLeaderboard(ctx, tx, request)
	if err != nil {
		t.Fatalf("Failed to read leaderboard.\n  Error: %s", err)
	}
	if response.LeaderboardBalance == nil || len(response.LeaderboardBalance) != leaderboardLimit {
		t.Fatalf("Balance leaderboard should have %d entries but has %d.", leaderboardLimit, len(response.LeaderboardBalance))
	}
	if response.LeaderboardTransfer == nil || len(response.LeaderboardTransfer) != leaderboardLimit {
		t.Fatalf("Transfer leaderboard should have %d entries but has %d.", leaderboardLimit, len(response.LeaderboardTransfer))
	}
	for i, entry := range response.LeaderboardBalance {
		var expectedBalance int64
		switch i {
		case 0:
			if entry.Address != ip2 {
				t.Fatalf("Balance leaderboard entry %d should have address %s but has %s.", i, ip2.String(), entry.Address.String())
			}
			expectedBalance = untouched + 2*leaderboardLimit
		case 1:
			if entry.Address != ip1 {
				t.Fatalf("Balance leaderboard entry %d should have address %s but has %s.", i, ip1.String(), entry.Address.String())
			}
			expectedBalance = untouched + leaderboardLimit
		default:
			if !strings.HasPrefix(entry.Address.String(), "192.168.0.") {
				t.Fatalf("Invalid address on balance leaderboard %d.\n  Address: %s", i, entry.Address.String())
			}
			expectedBalance = untouched - 3
		}
		if entry.BalanceAvailable != expectedBalance {
			t.Fatalf("Balance leaderboard entry %d should have balance %d but has %d.\n  Address: %s", i, expectedBalance, entry.BalanceAvailable, entry.Address.String())
		}
	}
	for i, entry := range response.LeaderboardTransfer {
		var transferCount int64
		switch i {
		case 0, 1:
			if !(entry.Address == ip1 || entry.Address == ip2) {
				t.Fatalf("Transfer leaderboard entry %d should have address %s or %s but has %s.", i, ip1.String(), ip2.String(), entry.Address.String())
			}
			transferCount = leaderboardLimit
		default:
			if !strings.HasPrefix(entry.Address.String(), "192.168.0.") {
				t.Fatalf("Invalid address on transfer leaderboard %d.\n  Address: %s", i, entry.Address.String())
			}
			transferCount = 2
		}
		if entry.TransferCount != transferCount {
			t.Fatalf("Transfer leaderboard entry %d should have transfer count %d but has %d.\n  Address: %s", i, transferCount, entry.TransferCount, entry.Address.String())
		}
	}

	now = now.Add(time.Minute)
	for _, addr := range addrs {
		tRequest := CreateTransferRequest{
			Amount:    3,
			Sender:    ip2,
			Now:       now,
			Recipient: addr,
		}
		_, err = CreateTransfer(ctx, tx, tRequest)
		if err != nil {
			t.Fatalf("Failed to transfer.\n  Error: %s", err)
		}
	}

	err = RefreshLeaderboard(ctx, tx, false)
	if err != nil {
		t.Fatalf("Failed to refresh leaderboard.\n  Error: %s", err)
	}
	request = GetLeaderboardRequest{
		Now: now,
	}
	response, _, err = GetLeaderboard(ctx, tx, request)
	if err != nil {
		t.Fatalf("Failed to read leaderboard.\n  Error: %s", err)
	}
	if response.LeaderboardBalance == nil || len(response.LeaderboardBalance) != leaderboardLimit {
		t.Fatalf("Balance leaderboard should have %d entries but has %d.", leaderboardLimit, len(response.LeaderboardBalance))
	}
	if response.LeaderboardTransfer == nil || len(response.LeaderboardTransfer) != leaderboardLimit {
		t.Fatalf("Transfer leaderboard should have %d entries but has %d.", leaderboardLimit, len(response.LeaderboardTransfer))
	}
	for i, entry := range response.LeaderboardBalance {
		var expectedBalance int64
		switch i {
		case 0:
			if entry.Address != ip1 {
				t.Fatalf("Balance leaderboard entry %d should have address %s but has %s.", i, ip2.String(), entry.Address.String())
			}
			expectedBalance = untouched + leaderboardLimit
		default:
			if !strings.HasPrefix(entry.Address.String(), "192.168.0.") {
				t.Fatalf("Invalid address on balance leaderboard %d.\n  Address: %s", i, entry.Address.String())
			}
			expectedBalance = untouched
		}
		if entry.BalanceAvailable != expectedBalance {
			t.Fatalf("Balance leaderboard entry %d should have balance %d but has %d.\n  Address: %s", i, expectedBalance, entry.BalanceAvailable, entry.Address.String())
		}
	}
	for i, entry := range response.LeaderboardTransfer {
		var transferCount int64
		switch i {
		case 0:
			transferCount = 2 * leaderboardLimit
			if entry.Address != ip2 {
				t.Fatalf("Transfer leaderboard entry %d should have address %s but has %s.", i, ip2.String(), entry.Address.String())
			}
		case 1:
			transferCount = leaderboardLimit
			if entry.Address != ip1 {
				t.Fatalf("Transfer leaderboard entry %d should have address %s but has %s.", i, ip1.String(), entry.Address.String())
			}
		default:
			transferCount = 3
			if !strings.HasPrefix(entry.Address.String(), "192.168.0.") {
				t.Fatalf("Invalid address on transfer leaderboard %d.\n  Address: %s", i, entry.Address.String())
			}
		}
		if entry.TransferCount != transferCount {
			t.Fatalf("Transfer leaderboard entry %d should have transfer count %d but has %d.\n  Address: %s", i, transferCount, entry.TransferCount, entry.Address.String())
		}
	}
}

func TestLeaderboardEmpty(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatalf("Failed to begin transaction.\n  Error: %s", err)
	}
	defer tx.Rollback(ctx)

	err = RefreshLeaderboard(ctx, tx, false)
	if err != nil {
		t.Fatalf("Failed to refresh leaderboard.\n  Error: %s", err)
	}
	response, _, err := GetLeaderboard(ctx, tx, GetLeaderboardRequest{})
	if err != nil {
		t.Fatalf("Failed to read leaderboard.\n  Error: %s", err)
	}
	if response.LeaderboardBalance == nil || len(response.LeaderboardBalance) != 0 {
		t.Fatalf("Balance leaderboard should be empty but non-nil.")
	}
	if response.LeaderboardTransfer == nil || len(response.LeaderboardTransfer) != 0 {
		t.Fatalf("Transfer leaderboard should be empty but non-nil.")
	}
}
