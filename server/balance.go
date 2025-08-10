package server

import (
	"context"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/MicahParks/ipcoin/proto"
	"github.com/MicahParks/ipcoin/storage"
)

func (s *server) GetBalance(ctx context.Context, _ *proto.GetBalanceRequest) (*proto.GetBalanceResponse, error) {
	from, err := s.getPeer(ctx)
	if err != nil {
		return nil, err
	}
	err = s.readLimiter.Wait(ctx, from)
	if err != nil {
		return nil, status.Error(codes.ResourceExhausted, "rate limit exceeded")
	}

	tx, err := s.tx(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	now := s.clock.Now()
	dbReq := storage.GetBalanceRequest{
		Address: from,
		Now:     now,
	}
	balance, err := storage.GetBalance(ctx, tx, dbReq)
	if err != nil {
		return nil, status.Error(codes.Internal, "unable to transfer to address")
	}

	err = tx.Commit(ctx)
	if err != nil {
		return nil, status.Error(codes.Internal, "unable to commit database transaction")
	}

	response := &proto.GetBalanceResponse{
		Balance: &proto.Balance{
			Timestamp: timestamppb.New(now),
			Address:   from.AsSlice(),
			Available: balance,
		},
	}
	return response, nil
}
