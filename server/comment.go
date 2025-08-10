package server

import (
	"context"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/MicahParks/ipcoin/proto"
	"github.com/MicahParks/ipcoin/storage"
)

const maxCommentLength = 1_000

func (s *server) CreateComment(ctx context.Context, request *proto.CreateCommentRequest) (*proto.CreateCommentResponse, error) {
	if len(request.GetComment()) == 0 {
		return nil, status.Error(codes.InvalidArgument, "comment must not be empty")
	}
	if len(request.GetComment()) > maxCommentLength {
		return nil, status.Errorf(codes.InvalidArgument, "comment must not be longer than %d characters", maxCommentLength)
	}

	address, err := s.getPeer(ctx)
	if err != nil {
		return nil, err
	}
	err = s.writeLimiter.Wait(ctx, address)
	if err != nil {
		return nil, status.Error(codes.ResourceExhausted, "rate limit exceeded")
	}

	tx, err := s.tx(ctx)
	if err != nil {
		return nil, status.Error(codes.Internal, "")
	}
	defer tx.Rollback(ctx)

	now := s.clock.Now()
	storageRequest := storage.CreateCommentRequest{
		Addr:    address,
		Message: request.GetComment(),
		Now:     now,
	}
	storageResponse, err := storage.CreateComment(ctx, tx, storageRequest)
	if err != nil {
		return nil, status.Error(codes.Internal, "unable to create comment")
	}

	err = tx.Commit(ctx)
	if err != nil {
		return nil, status.Error(codes.Internal, "unable to commit database transaction")
	}

	response := &proto.CreateCommentResponse{
		Comment: &proto.Comment{
			Created: timestamppb.New(storageResponse.Comment.Created),
			Id:      storageResponse.Comment.ID.String(),
			Address: storageResponse.Comment.Address.AsSlice(),
			Message: storageResponse.Comment.Message,
		},
	}
	return response, nil
}
