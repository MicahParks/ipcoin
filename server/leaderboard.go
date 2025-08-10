package server

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/MicahParks/ipcoin"
	"github.com/MicahParks/ipcoin/ctxkey"
	"github.com/MicahParks/ipcoin/proto"
	"github.com/MicahParks/ipcoin/storage"
)

func (s *server) GetLeaderboard(ctx context.Context, request *proto.GetLeaderboardRequest) (*proto.GetLeaderboardResponse, error) {
	from, err := s.getPeer(ctx)
	if err != nil {
		return nil, err
	}
	err = s.readLimiter.Wait(ctx, from)
	if err != nil {
		return nil, status.Error(codes.ResourceExhausted, "rate limit exceeded")
	}
	return s.leaderboardGetter.Get(ctx, request) // TODO Refresh MATERIALIZED VIEW occasionally.
}

type LeaderboardGetter interface {
	Get(ctx context.Context, request *proto.GetLeaderboardRequest) (*proto.GetLeaderboardResponse, error)
}

type leaderboardDebug struct {
	s *server
}

func (l leaderboardDebug) Get(ctx context.Context, request *proto.GetLeaderboardRequest) (*proto.GetLeaderboardResponse, error) {
	now := l.s.clock.Now()
	storageRequest := storage.GetLeaderboardRequest{
		Now: now,
	}
	err := storage.RefreshLeaderboard(ctx, l.s.pool, false)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "unable to refresh leaderboard")
	}
	response, balanceUntouched, err := storage.GetLeaderboard(ctx, l.s.pool, storageRequest)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "unable to get leaderboard")
	}
	return protoBuildGetLeaderboardResponse(response, balanceUntouched, now), nil
}

type leaderboardNoOp struct{}

func (leaderboardNoOp) Get(_ context.Context, _ *proto.GetLeaderboardRequest) (*proto.GetLeaderboardResponse, error) {
	return nil, nil
}

type leaderboardMemCache struct {
	mux      sync.RWMutex
	response *proto.GetLeaderboardResponse
}

func NewLeaderboardMemCache(ctx context.Context, pool *pgxpool.Pool) LeaderboardGetter {
	l := ctx.Value(ctxkey.Logger).(*slog.Logger)
	cache := &leaderboardMemCache{
		response: protoBuildGetLeaderboardResponse(storage.GetLeaderboardResponse{}, 0, time.Now()),
	}
	go func() {
		for {
			now := time.Now()
			nextRequest := now.Truncate(time.Minute).Add(time.Minute)
			select {
			case <-ctx.Done():
				return
			case <-time.After(time.Until(nextRequest)):
				request := storage.GetLeaderboardRequest{
					Now: nextRequest,
				}
				response, balanceUntouched, err := storage.GetLeaderboard(ctx, pool, request)
				if err != nil {
					l.WarnContext(ctx, "Failed to get leaderboard.",
						ipcoin.LogErr, err,
					)
					continue
				}
				cache.mux.Lock()
				cache.response = protoBuildGetLeaderboardResponse(response, balanceUntouched, nextRequest)
				cache.mux.Unlock()
				l.DebugContext(ctx, "Leaderboard updated.")
			}
		}
	}()
	return cache
}

func (c *leaderboardMemCache) Get(_ context.Context, _ *proto.GetLeaderboardRequest) (*proto.GetLeaderboardResponse, error) {
	c.mux.RLock()
	defer c.mux.RUnlock()
	return c.response, nil
}

func protoBuildGetLeaderboardResponse(response storage.GetLeaderboardResponse, balanceUntouched int64, now time.Time) *proto.GetLeaderboardResponse {
	leaderboardBalanceEntries := make([]*proto.Glance, len(response.LeaderboardBalance))
	leaderboardTransferEntries := make([]*proto.Glance, len(response.LeaderboardTransfer))
	for i, entry := range response.LeaderboardBalance {
		leaderboardBalanceEntries[i] = &proto.Glance{
			Timestamp:        timestamppb.New(now),
			Address:          entry.Address.AsSlice(),
			BalanceAvailable: entry.BalanceAvailable,
			CommentCount:     entry.CommentCount,
			TransferCount:    entry.TransferCount,
		}
	}
	for i, entry := range response.LeaderboardTransfer {
		leaderboardTransferEntries[i] = &proto.Glance{
			Timestamp:        timestamppb.New(now),
			Address:          entry.Address.AsSlice(),
			BalanceAvailable: entry.BalanceAvailable,
			CommentCount:     entry.CommentCount,
			TransferCount:    entry.TransferCount,
		}
	}
	return &proto.GetLeaderboardResponse{
		Leaderboard: &proto.Leaderboard{
			Timestamp:        timestamppb.New(now),
			BalanceUntouched: balanceUntouched,
			LeaderboardBalance: &proto.LeaderboardBalance{
				Entries: leaderboardBalanceEntries,
			},
			LeaderboardTransfer: &proto.LeaderboardTransfer{
				Entries: leaderboardTransferEntries,
			},
		},
	}
}
