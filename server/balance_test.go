package server

import (
	"context"
	"net"
	"net/netip"
	"testing"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/peer"
	"google.golang.org/grpc/status"

	"github.com/MicahParks/ipcoin/ctxkey"
	"github.com/MicahParks/ipcoin/proto"
	"github.com/MicahParks/ipcoin/storage"
)

func TestServer_GetBalance(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	ctx, tx := addTx(ctx, t)
	defer tx.Rollback(ctx)

	for ip, isV4 := range map[string]bool{"192.168.0.1": true, "::1": false} {
		parsed := net.ParseIP(ip)
		if isV4 {
			parsed = parsed.To4()
		}
		p := &peer.Peer{
			Addr: &net.TCPAddr{IP: parsed},
		}
		ctx = context.WithValue(ctx, ctxkey.TestingPeer, p)

		request := &proto.GetBalanceRequest{}
		response, err := s.GetBalance(ctx, request)
		if err != nil {
			t.Fatalf("Failed to read balance.\n  Error: %s", err)
		}

		expectedBalanced := storage.BalanceUntouched(now)

		if response.GetBalance().GetAvailable() != expectedBalanced {
			t.Fatalf("Unexpected balanced.\n  Expected: %d\n  Actual: %d", expectedBalanced, response.GetBalance().GetAvailable())
		}

		addr, ok := netip.AddrFromSlice(response.GetBalance().GetAddress())
		if !ok {
			t.Fatalf("Failed to parse address.\n  Error: %s", err)
		}
		if addr.String() != ip {
			t.Fatalf("Unexpected address.\n  Expected: %s\n  Actual: %s", ip, addr.String())
		}
	}
}

func TestServer_GetBalanceNoPeer(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	request := &proto.GetBalanceRequest{}
	_, err := s.GetBalance(ctx, request)
	if status.Code(err) != codes.Unauthenticated {
		t.Fatalf("Should have internal error.\n  Error: %s", err)
	}
}

func Test_getPeer(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	p := &peer.Peer{
		Addr: &net.TCPAddr{IP: net.ParseIP("")},
	}
	ctx2 := context.WithValue(ctx, ctxkey.TestingPeer, p)
	_, err := s.getPeer(ctx2)
	if status.Code(err) != codes.Unauthenticated {
		t.Fatalf("Should have unauthenticated error.\n  Error: %s", err)
	}
}
