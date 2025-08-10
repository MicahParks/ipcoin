package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"

	"github.com/google/uuid"
	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"

	"github.com/MicahParks/ipcoin"
	"github.com/MicahParks/ipcoin/ctxkey"
	"github.com/MicahParks/ipcoin/proto"
	"github.com/MicahParks/ipcoin/server"
	"github.com/MicahParks/ipcoin/storage"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	l := slog.Default()
	ctx = context.WithValue(ctx, ctxkey.Logger, l.With("defaultLogger", true))

	c, err := config()
	if err != nil {
		l.ErrorContext(ctx, "Failed to read config.",
			ipcoin.LogErr, err,
		)
		return
	}

	l.InfoContext(ctx, "Connecting to PostgreSQL.")
	pool, err := storage.NewPool(ctx, c.DBDSN)
	if err != nil {
		l.ErrorContext(ctx, "Failed to connect to PostgreSQL.",
			ipcoin.LogErr, err,
		)
		return
	}
	defer pool.Close()
	l.InfoContext(ctx, "Connected to PostgreSQL.")

	runtimeKey := uuid.New().String()
	s := server.New(ctx, c, server.NewRealClock(), l, server.NewLeaderboardMemCache(ctx, pool), pool, runtimeKey)

	gs := grpc.NewServer()
	proto.RegisterIPCoinServiceServer(gs, s)
	lis, err := net.Listen("tcp", ":8080")
	if err != nil {
		l.ErrorContext(ctx, "Failed to create listener.",
			ipcoin.LogErr, err,
		)
		return
	}

	go func() {
		defer cancel()
		l.InfoContext(ctx, "Serving gRPC on port 8080.")
		err = gs.Serve(lis)
		if err != nil {
			l.ErrorContext(ctx, "Failed to serve gRPC.",
				ipcoin.LogErr, err,
			)
			return
		}
	}()

	mux := runtime.NewServeMux(
		runtime.WithMetadata(func(ctx context.Context, request *http.Request) metadata.MD {
			pairs := metadata.Pairs(
				ipcoin.GPRCMetadataCFConnectingIP, request.Header.Get("CF-Connecting-IP"),
				ipcoin.GRPCMetadataKeyRuntimeKey, runtimeKey,
			)
			return pairs
		}),
	)
	opts := []grpc.DialOption{grpc.WithTransportCredentials(insecure.NewCredentials())}
	err = proto.RegisterIPCoinServiceHandlerFromEndpoint(ctx, mux, ":8080", opts)
	if err != nil {
		l.ErrorContext(ctx, "Failed to register gRPC gateway.",
			ipcoin.LogErr, err,
		)
		return
	}

	l.InfoContext(ctx, "Serving HTTP proxy on port 8081.")
	err = http.ListenAndServe(":8081", allCORS(mux))
	if err != nil {
		l.ErrorContext(ctx, "Failed to serve HTTP.",
			ipcoin.LogErr, err,
		)
		return
	}
}

func allCORS(handler http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Set CORS headers
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS, PATCH")
		w.Header().Set("Access-Control-Allow-Headers", "Accept, Content-Type, Content-Length, Accept-Encoding, Authorization, X-CSRF-Token")

		// Handle preflight requests
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		// Serve the actual request
		handler.ServeHTTP(w, r)
	})
}

func config() (ipcoin.Config, error) {
	b, err := os.ReadFile("config.json")
	if err != nil {
		return ipcoin.Config{}, fmt.Errorf("failed to read config JSON file: %w", err)
	}
	var c ipcoin.Config
	err = json.Unmarshal(b, &c)
	if err != nil {
		return ipcoin.Config{}, fmt.Errorf("failed to unmarshal config JSON: %w", err)
	}
	if c.DBDSN == "" {
		return ipcoin.Config{}, errors.New("config.json must contain a database DSN")
	}
	return c, nil
}
