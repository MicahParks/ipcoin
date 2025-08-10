package server

import (
	"context"
	"net/netip"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/MicahParks/ipcoin/proto"
	"github.com/MicahParks/ipcoin/storage"
)

func (s *server) GetGlance(ctx context.Context, request *proto.GetGlanceRequest) (*proto.GetGlanceResponse, error) {
	from, err := s.getPeer(ctx)
	if err != nil {
		return nil, err
	}
	err = s.readLimiter.Wait(ctx, from)
	if err != nil {
		return nil, status.Error(codes.ResourceExhausted, "rate limit exceeded")
	}

	var address netip.Addr
	a := request.GetAddress()
	if a == nil {
		address = from
	} else {
		var ok bool
		address, ok = netip.AddrFromSlice(a)
		if !ok {
			return nil, status.Error(codes.InvalidArgument, "invalid address")
		}
	}

	tx, err := s.tx(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	now := s.clock.Now()
	dbReq := storage.GetGlanceRequest{
		Address: address,
		Now:     now,
	}
	glance, balanceUntouched, err := storage.GetGlance(ctx, tx, dbReq)
	if err != nil {
		return nil, status.Error(codes.Internal, "unable to transfer to address")
	}

	err = tx.Commit(ctx)
	if err != nil {
		return nil, status.Error(codes.Internal, "unable to commit database transaction")
	}

	response := &proto.GetGlanceResponse{
		Glance: &proto.Glance{
			Timestamp:        timestamppb.New(now),
			Address:          glance.Address.AsSlice(),
			BalanceAvailable: glance.BalanceAvailable,
			CommentCount:     glance.CommentCount,
			TransferCount:    glance.TransferCount,
		},
		BalanceUntouched: balanceUntouched,
	}
	return response, nil
}
