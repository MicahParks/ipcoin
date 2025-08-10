package server

import (
	"context"
	"net/netip"
	"slices"

	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/MicahParks/ipcoin/proto"
	"github.com/MicahParks/ipcoin/storage"
)

func (s *server) GetFeed(ctx context.Context, request *proto.GetFeedRequest) (*proto.GetFeedResponse, error) { // TODO Cache similar to leaderboard.
	now := s.clock.Now()
	var address *netip.Addr
	if request.GetAddress() != nil {
		a, ok := netip.AddrFromSlice(request.GetAddress())
		if !ok {
			return nil, status.Error(codes.InvalidArgument, "invalid address")
		}
		address = &a
	}

	storageRequest := storage.GetFeedRequest{
		Now:     now,
		Address: address,
	}
	storageResponse, err := storage.GetFeed(ctx, s.pool, storageRequest)
	if err != nil {
		return nil, status.Error(codes.Internal, "unable to get feed")
	}

	censoredIDs := make([]uuid.UUID, 0)
	if len(storageResponse.Feed.Comment) > 0 {
		commentIDs := make([]uuid.UUID, len(storageResponse.Feed.Comment))
		for i, c := range storageResponse.Feed.Comment {
			commentIDs[i] = c.ID
		}
		censoredIDs, err = storage.ReadCommentCensored(ctx, s.pool, commentIDs)
		if err != nil {
			return nil, status.Error(codes.Internal, "unable to read comment moderation")
		}
	}

	comment := make([]*proto.Comment, len(storageResponse.Feed.Comment))
	for i, c := range storageResponse.Feed.Comment {
		protoComment := &proto.Comment{
			Created:  timestamppb.New(c.Created),
			Id:       c.ID.String(),
			Address:  c.Address.AsSlice(),
			Message:  c.Message,
			Censored: slices.Contains(censoredIDs, c.ID),
		}
		comment[i] = protoComment
	}
	transfer := make([]*proto.Transfer, len(storageResponse.Feed.Transfer))
	for i, t := range storageResponse.Feed.Transfer {
		transfer[i] = &proto.Transfer{
			Created:          timestamppb.New(t.Created),
			Id:               t.ID.String(),
			SenderAddress:    t.Sender.AsSlice(),
			RecipientAddress: t.Recipient.AsSlice(),
			Amount:           t.Amount,
		}
	}
	response := &proto.GetFeedResponse{
		Feed: &proto.Feed{
			Timestamp: timestamppb.New(storageResponse.Feed.Timestamp),
			Comment:   comment,
			Transfer:  transfer,
		},
	}
	return response, nil
}
