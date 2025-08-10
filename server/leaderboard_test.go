package server

import (
	"context"
	"net/netip"
	"testing"
	"time"

	"github.com/MicahParks/ipcoin/proto"
	"github.com/MicahParks/ipcoin/storage"
)

func TestServer_GetLeaderboard(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	request := &proto.GetLeaderboardRequest{}

	ipA := netip.MustParseAddr("192.168.0.1")
	bal1 := int64(999)
	tCount1 := int64(1)
	ipB := netip.MustParseAddr("192.168.0.2")
	bal2 := int64(1)
	tCount2 := int64(999)
	storageResponse := storage.GetLeaderboardResponse{
		LeaderboardBalance: []storage.Glance{
			{
				Address:          ipA,
				BalanceAvailable: bal1,
				CommentCount:     0,
				TransferCount:    0,
			},
			{
				Address:          ipB,
				BalanceAvailable: bal2,
				CommentCount:     0,
				TransferCount:    0,
			},
		},
		LeaderboardTransfer: []storage.Glance{
			{
				Address:          ipB,
				BalanceAvailable: 0,
				CommentCount:     0,
				TransferCount:    tCount2,
			},
			{
				Address:          ipA,
				BalanceAvailable: 0,
				CommentCount:     0,
				TransferCount:    tCount1,
			},
		},
	}
	serv := s
	oldLeaderboardGetter := serv.leaderboardGetter
	serv.leaderboardGetter = &leaderboardMemCache{
		response: protoBuildGetLeaderboardResponse(storageResponse, 0, now),
	}
	defer func() {
		serv.leaderboardGetter = oldLeaderboardGetter
	}()

	response, err := s.GetLeaderboard(ctx, request)
	if err != nil {
		t.Fatalf("Failed to read leaderboard.\n  Error: %s", err)
	}
	balance := response.GetLeaderboard().GetLeaderboardBalance().GetEntries()
	transfer := response.GetLeaderboard().GetLeaderboardTransfer().GetEntries()
	if len(balance) != 2 {
		t.Fatalf("Expected 2 entries on balance leaderboard.\n  Actual: %d", len(balance))
	}
	if len(transfer) != 2 {
		t.Fatalf("Expected 2 entries on transfer leaderboard.\n  Actual: %d", len(transfer))
	}

	addr := netIPMustParseSlice(balance[0].GetAddress())
	if addr != ipA {
		t.Fatalf("Balance leaderboard incorrect address.\n  Expected: %s\n  Actual: %s", addr.String(), ipA.String())
	}
	if balance[0].GetBalanceAvailable() != bal1 {
		t.Fatalf("Balance leaderboard incorrect balance.\n  Expected: %d\n  Actual: %d", bal1, balance[0].GetBalanceAvailable())
	}
	addr = netIPMustParseSlice(balance[1].GetAddress())
	if addr != ipB {
		t.Fatalf("Balance leaderboard incorrect address.\n  Expected: %s\n  Actual: %s", addr.String(), ipB.String())
	}
	if balance[1].GetBalanceAvailable() != bal2 {
		t.Fatalf("Balance leaderboard incorrect balance.\n  Expected: %d\n  Actual: %d", bal2, balance[1].GetBalanceAvailable())
	}
	addr = netIPMustParseSlice(transfer[0].GetAddress())
	if addr != ipB {
		t.Fatalf("Transfer leaderboard incorrect address.\n  Expected: %s\n  Actual: %s", addr.String(), ipA.String())
	}
	if transfer[0].GetTransferCount() != tCount2 {
		t.Fatalf("Transfer leaderboard incorrect transfer count.\n  Expected: %d\n  Actual: %d", tCount2, transfer[0].GetTransferCount())
	}
	addr = netIPMustParseSlice(transfer[1].GetAddress())
	if addr != ipA {
		t.Fatalf("Transfer leaderboard incorrect address.\n  Expected: %s\n  Actual: %s", addr.String(), ipA.String())
	}
	if transfer[1].GetTransferCount() != tCount1 {
		t.Fatalf("Transfer leaderboard incorrect transfer count.\n  Expected: %d\n  Actual: %d", tCount1, transfer[1].GetTransferCount())
	}
}

func TestServer_GetLeaderboard_Empty(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	request := &proto.GetLeaderboardRequest{}
	response, err := s.GetLeaderboard(ctx, request)
	if err != nil {
		t.Fatalf("Failed to read leaderboard.\n  Error: %s", err)
	}
	balance := response.GetLeaderboard().GetLeaderboardBalance().GetEntries()
	transfer := response.GetLeaderboard().GetLeaderboardTransfer().GetEntries()
	if len(balance) != 0 {
		t.Fatalf("Expected 0 entries on balance leaderboard.\n  Actual: %d", len(balance))
	}
	if len(transfer) != 0 {
		t.Fatalf("Expected 0 entries on transfer leaderboard.\n  Actual: %d", len(transfer))
	}
}

func netIPMustParseSlice(b []byte) netip.Addr {
	addr, ok := netip.AddrFromSlice(b)
	if !ok {
		panic("Failed to parse address.")
	}
	return addr
}
