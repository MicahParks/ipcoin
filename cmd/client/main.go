package main

import (
	"context"
	"log/slog"
	"net/netip"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/MicahParks/ipcoin"
	"github.com/MicahParks/ipcoin/proto"
)

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	l := slog.Default()

	conn, err := grpc.NewClient("localhost:8080", grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		l.ErrorContext(ctx, "Failed to create gRPC client.",
			ipcoin.LogErr, err,
		)
		return
	}
	defer conn.Close()

	client := proto.NewIPCoinServiceClient(conn)
	balanceResponse, err := client.GetBalance(ctx, &proto.GetBalanceRequest{})
	if err != nil {
		l.ErrorContext(ctx, "Failed to read balance.",
			ipcoin.LogErr, err,
		)
		return
	}
	addr, ok := netip.AddrFromSlice(balanceResponse.GetBalance().GetAddress())
	if !ok {
		l.ErrorContext(ctx, "Failed to parse address.")
		return
	}
	l.InfoContext(ctx, "Balance read.",
		"addr", addr.String(),
		"balance", balanceResponse.Balance,
	)

	transferResponse, err := client.CreateTransfer(context.Background(), &proto.CreateTransferRequest{
		Amount:           1,
		RecipientAddress: netip.MustParseAddr("192.168.0.1").AsSlice(),
	})
	if err != nil {
		l.ErrorContext(ctx, "Failed to transfer.",
			ipcoin.LogErr, err,
		)
		return
	}
	addr, ok = netip.AddrFromSlice(transferResponse.GetSenderBalance().GetAddress())
	if !ok {
		l.ErrorContext(ctx, "Failed to parse address.",
			ipcoin.LogErr, err,
		)
		return
	}
	l.InfoContext(ctx, "Transfer complete.",
		"addr", addr.String(),
		"balance", balanceResponse.GetBalance().GetAvailable(),
	)
}
