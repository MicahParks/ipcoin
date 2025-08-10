package server

import (
	"context"
	"errors"
	"net/netip"
	"time"

	"github.com/jackc/pgx/v5"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/MicahParks/ipcoin/proto"
	"github.com/MicahParks/ipcoin/storage"
)

func (s *server) CreateTransfer(ctx context.Context, request *proto.CreateTransferRequest) (*proto.CreateTransferResponse, error) {
	amount := request.GetAmount()
	if amount < 1 {
		return nil, status.Error(codes.InvalidArgument, "invalid amount")
	}
	toBytes := request.GetRecipientAddress()
	var recipient netip.Addr
	switch len(toBytes) {
	case 4:
		recipient = netip.AddrFrom4([4]byte(toBytes))
	case 16:
		recipient = netip.AddrFrom16([16]byte(toBytes))
	default:
		return nil, status.Error(codes.InvalidArgument, "invalid to address")
	}
	sender, err := s.getPeer(ctx)
	if err != nil {
		return nil, err
	}
	if sender == recipient {
		return nil, status.Error(codes.InvalidArgument, "cannot transfer to self")
	}
	err = s.writeLimiter.Wait(ctx, sender)
	if err != nil {
		return nil, status.Error(codes.ResourceExhausted, "rate limit exceeded")
	}

	var innerErr error
	var storageResponse storage.CreateTransferResponse
	var now time.Time
	s.addrLocker.WithLock(ctx, sender, func() {
		var tx pgx.Tx
		tx, innerErr = s.tx(ctx)
		if innerErr != nil {
			innerErr = status.Error(codes.Internal, "")
			return
		}
		defer tx.Rollback(ctx)

		now = s.clock.Now()
		dbReq := storage.CreateTransferRequest{
			Amount:    amount,
			Sender:    sender,
			Now:       now,
			Recipient: recipient,
		}
		storageResponse, innerErr = storage.CreateTransfer(ctx, tx, dbReq)
		switch {
		case errors.Is(innerErr, storage.ErrInsufficientBalance):
			innerErr = status.Error(codes.InvalidArgument, "insufficient balance")
			return
		case innerErr != nil:
			innerErr = status.Error(codes.Internal, "unable to transfer to address")
			return
		}

		innerErr = tx.Commit(ctx)
		if innerErr != nil {
			innerErr = status.Error(codes.Internal, "unable to commit database transaction")
			return
		}
	})
	if innerErr != nil {
		return nil, innerErr
	}

	pbNow := timestamppb.New(storageResponse.Transfer.Created)
	response := &proto.CreateTransferResponse{
		Transfer: &proto.Transfer{
			Created:          pbNow,
			Id:               storageResponse.Transfer.ID.String(),
			SenderAddress:    storageResponse.Transfer.Sender.AsSlice(),
			RecipientAddress: storageResponse.Transfer.Recipient.AsSlice(),
			Amount:           storageResponse.Transfer.Amount,
		},
		SenderBalance: &proto.Balance{
			Timestamp: pbNow,
			Address:   storageResponse.Transfer.Sender.AsSlice(),
			Available: storageResponse.SenderBalance,
		},
	}
	return response, nil
}
