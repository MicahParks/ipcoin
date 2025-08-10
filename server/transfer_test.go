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

func TestServer_Transfer(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	fromIPs := []string{"127.0.0.1", "::1"}
	toIPs := []string{"127.0.0.2", "::2"}
	for _, sender := range fromIPs {
		for _, recipient := range toIPs {
			p := &peer.Peer{
				Addr: &net.TCPAddr{IP: net.ParseIP(sender)},
			}
			ctx = context.WithValue(ctx, ctxkey.TestingPeer, p)
			recipientAddr := netip.MustParseAddr(recipient).AsSlice()

			t.Run("Valid", func(t *testing.T) {
				ctx, tx := addTx(ctx, t)
				defer tx.Rollback(ctx)
				request := &proto.CreateTransferRequest{
					Amount:           1,
					RecipientAddress: recipientAddr,
				}
				_, err := s.CreateTransfer(ctx, request)
				if err != nil {
					t.Fatalf("Failed to transfer.\n  Error: %s", err)
				}
			})

			t.Run("InsufficientBalance", func(t *testing.T) {
				ctx, tx := addTx(ctx, t)
				defer tx.Rollback(ctx)
				request := &proto.CreateTransferRequest{
					Amount:           storage.BalanceUntouched(now) + 1,
					RecipientAddress: recipientAddr,
				}
				_, err := s.CreateTransfer(ctx, request)
				if status.Code(err) != codes.InvalidArgument {
					t.Fatalf("Should have invalid argument error.\n  Error: %s", err)
				}
			})

			t.Run("Negative", func(t *testing.T) {
				ctx, tx := addTx(ctx, t)
				defer tx.Rollback(ctx)
				request := &proto.CreateTransferRequest{
					Amount:           -1,
					RecipientAddress: recipientAddr,
				}
				_, err := s.CreateTransfer(ctx, request)
				if status.Code(err) != codes.InvalidArgument {
					t.Fatalf("Should have invalid argument error.\n  Error: %s", err)
				}
			})
		}
	}
}

func TestServer_TransferSelf(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	ctx, tx := addTx(ctx, t)
	defer tx.Rollback(ctx)

	p := &peer.Peer{
		Addr: &net.TCPAddr{IP: net.ParseIP("::1")},
	}
	ctx = context.WithValue(ctx, ctxkey.TestingPeer, p)

	request := &proto.CreateTransferRequest{
		Amount:           1,
		RecipientAddress: netip.MustParseAddr("::1").AsSlice(),
	}
	_, err := s.CreateTransfer(ctx, request)
	if status.Code(err) != codes.InvalidArgument {
		t.Fatalf("Should have invalid argument error.\n  Error: %s", err)
	}
}

func TestServer_TransferInvalidArguments(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	ctx, tx := addTx(ctx, t)
	defer tx.Rollback(ctx)

	request := &proto.CreateTransferRequest{
		Amount:           0,
		RecipientAddress: nil,
	}
	_, err := s.CreateTransfer(ctx, request)
	if status.Code(err) != codes.InvalidArgument {
		t.Fatalf("Should have invalid argument error.\n  Error: %s", err)
	}

	recipient := netip.MustParseAddr("192.168.0.1").AsSlice()
	request = &proto.CreateTransferRequest{
		Amount:           0,
		RecipientAddress: recipient,
	}
	_, err = s.CreateTransfer(ctx, request)
	if status.Code(err) != codes.InvalidArgument {
		t.Fatalf("Should have invalid argument error.\n  Error: %s", err)
	}

	const amount = 1
	request = &proto.CreateTransferRequest{
		Amount:           amount,
		RecipientAddress: nil,
	}
	_, err = s.CreateTransfer(ctx, request)
	if status.Code(err) != codes.InvalidArgument {
		t.Fatalf("Should have invalid argument error.\n  Error: %s", err)
	}

	request = &proto.CreateTransferRequest{
		Amount:           amount,
		RecipientAddress: recipient,
	}
	_, err = s.CreateTransfer(ctx, request)
	if status.Code(err) != codes.Unauthenticated {
		t.Fatalf("Should have invalid argument error.\n  Error: %s", err)
	}
}
