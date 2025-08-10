package server

import (
	"context"
	"log/slog"
	"net"
	"net/netip"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/MicahParks/ipcoin/storage"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/openai/openai-go/v2"
	"github.com/openai/openai-go/v2/option"
	"golang.org/x/time/rate"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/peer"
	"google.golang.org/grpc/status"

	"github.com/MicahParks/ipcoin"
	"github.com/MicahParks/ipcoin/ctxkey"
	"github.com/MicahParks/ipcoin/proto"
)

type server struct {
	addrLocker        *addrLocker
	c                 ipcoin.Config
	clock             Clock
	l                 *slog.Logger
	leaderboardGetter LeaderboardGetter
	openai            openai.Client
	pool              *pgxpool.Pool
	readLimiter       AddressLimiter
	runtimeKey        string
	writeLimiter      AddressLimiter
	proto.UnimplementedIPCoinServiceServer
}

func New(ctx context.Context, c ipcoin.Config, clock Clock, l *slog.Logger, leaderboardGetter LeaderboardGetter, pool *pgxpool.Pool, runtimeKey string) proto.IPCoinServiceServer {
	s := &server{
		addrLocker:        newAddrLocker(2 * time.Hour),
		c:                 c,
		clock:             clock,
		l:                 l,
		leaderboardGetter: leaderboardGetter,
		openai:            openai.NewClient(option.WithAPIKey(c.OpenAIAPIKey)),
		pool:              pool,
		readLimiter:       NewAddressLimiterMem(60, rate.Every(time.Second)),
		runtimeKey:        runtimeKey,
		writeLimiter:      NewAddressLimiterMem(6, rate.Every(10*time.Second)),
	}
	s.leaderboardGetter = leaderboardDebug{s: s}

	if c.OpenAIAPIKey != "" {
		go s.openaiModeration(ctx)
	}

	return s
}

type addrLock struct {
	ch          chan struct{}
	deleteTimer *time.Timer
}
type addrLocker struct {
	deleteAfter time.Duration
	m           map[netip.Addr]addrLock
	mux         sync.Mutex
}

func newAddrLocker(deleteAfter time.Duration) *addrLocker {
	return &addrLocker{
		deleteAfter: deleteAfter,
		m:           make(map[netip.Addr]addrLock),
	}
}

func (l *addrLocker) WithLock(ctx context.Context, addr netip.Addr, f func()) {
	l.mux.Lock()
	lock, ok := l.m[addr]
	if !ok {
		lock = addrLock{
			ch: make(chan struct{}, 1),
			deleteTimer: time.AfterFunc(l.deleteAfter, func() {
				l.mux.Lock()
				defer l.mux.Unlock()
				delete(l.m, addr)
			}),
		}
		l.m[addr] = lock
	} else {
		lock.deleteTimer.Reset(l.deleteAfter)
	}
	l.mux.Unlock()

	// Confirm context is not canceled to prevent randomized execution of function.
	select {
	case <-ctx.Done():
		return
	default:
	}

	select {
	case <-ctx.Done():
	case lock.ch <- struct{}{}:
		f()
		<-lock.ch
	}
}

func (s *server) tx(ctx context.Context) (pgx.Tx, error) {
	var err error
	tx, ok := ctx.Value(ctxkey.TestingTx).(pgx.Tx)
	if ok {
		tx, err = tx.Begin(ctx)
		if err != nil {
			return nil, status.Error(codes.Internal, "unable to start nested testing database transaction")
		}
	} else {
		tx, err = s.pool.Begin(ctx)
		if err != nil {
			return nil, status.Error(codes.Internal, "unable to start database transaction")
		}
	}
	return tx, nil
}

var cloudflareIPRanges = []netip.Prefix{ // https://www.cloudflare.com/ips/
	netip.MustParsePrefix("173.245.48.0/20"),
	netip.MustParsePrefix("103.21.244.0/22"),
	netip.MustParsePrefix("103.22.200.0/22"),
	netip.MustParsePrefix("103.31.4.0/22"),
	netip.MustParsePrefix("141.101.64.0/18"),
	netip.MustParsePrefix("108.162.192.0/18"),
	netip.MustParsePrefix("190.93.240.0/20"),
	netip.MustParsePrefix("188.114.96.0/20"),
	netip.MustParsePrefix("197.234.240.0/22"),
	netip.MustParsePrefix("198.41.128.0/17"),
	netip.MustParsePrefix("162.158.0.0/15"),
	netip.MustParsePrefix("104.16.0.0/13"),
	netip.MustParsePrefix("104.24.0.0/14"),
	netip.MustParsePrefix("172.64.0.0/13"),
	netip.MustParsePrefix("131.0.72.0/22"),
	netip.MustParsePrefix("2400:cb00::/32"),
	netip.MustParsePrefix("2606:4700::/32"),
	netip.MustParsePrefix("2803:f800::/32"),
	netip.MustParsePrefix("2405:b500::/32"),
	netip.MustParsePrefix("2405:8100::/32"),
	netip.MustParsePrefix("2a06:98c0::/29"),
	netip.MustParsePrefix("2c0f:f248::/32"),
}

// getPeer should only return addresses from valid peers. Valid peers include tests or Cloudflare via Caddy via gRPC
// Gateway.
func (s *server) getPeer(ctx context.Context) (from netip.Addr, err error) {
	fromTest := false
	p, ok := peer.FromContext(ctx)
	if !ok {
		p, ok = ctx.Value(ctxkey.TestingPeer).(*peer.Peer)
		if !ok {
			return netip.Addr{}, status.Error(codes.Unauthenticated, "no peer or metadata")
		}
		fromTest = true
	}
	switch t := p.Addr.(type) {
	case *net.TCPAddr:
		from, ok = netip.AddrFromSlice(t.IP)
	case *net.UDPAddr:
		from, ok = netip.AddrFromSlice(t.IP)
	default:
		return netip.Addr{}, status.Error(codes.Unauthenticated, "unhandled peer address type")
	}
	if !ok {
		return netip.Addr{}, status.Error(codes.Unauthenticated, "invalid IP address")
	}

	if fromTest {
		return from, nil // Allow tests.
	}

	// All other requests must be from Cloudflare via Caddy via gRPC Gateway.
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return netip.Addr{}, status.Error(codes.Unauthenticated, "missing required metadata in gRPC request")
	}
	slice := md.Get("x-forwarded-for") // Added automatically by gRPC gateway, originally from Caddy.
	if len(slice) == 0 {
		return netip.Addr{}, status.Error(codes.Unauthenticated, "no x-forwarded-for in metadata")
	}
	forwardedFor, err := netip.ParseAddr(strings.SplitN(slice[0], ",", 2)[0])
	if err != nil {
		return netip.Addr{}, status.Error(codes.Unauthenticated, "invalid IP address in x-forwarded-for")
	}
	if !s.c.CloudflareRequired {
		return from, nil
	}
	fromCloudflare := false
	for _, prefix := range cloudflareIPRanges {
		if prefix.Contains(forwardedFor) {
			fromCloudflare = true
			break
		}
	}
	if !fromCloudflare {
		return netip.Addr{}, status.Error(codes.Unauthenticated, "forwarded IP address not from Cloudflare")
	}
	slice = md.Get(ipcoin.GRPCMetadataKeyRuntimeKey)
	if len(slice) == 0 {
		return netip.Addr{}, status.Error(codes.Unauthenticated, "no runtime key in metadata from Cloudflare")
	}
	if slice[0] != s.runtimeKey {
		return netip.Addr{}, status.Error(codes.Unauthenticated, "invalid runtime key")
	}
	slice = md.Get(ipcoin.GPRCMetadataCFConnectingIP)
	if len(slice) == 0 {
		return netip.Addr{}, status.Error(codes.Unauthenticated, "no connecting IP in metadata from Cloudflare")
	}
	from, err = netip.ParseAddr(strings.SplitN(slice[0], ",", 2)[0])
	if err != nil {
		return netip.Addr{}, status.Error(codes.Unauthenticated, "invalid IP address in metadata from Cloudflare")
	}

	return from, nil
}

func (s *server) openaiModeration(ctx context.Context) {
	ticker := time.NewTicker(time.Minute)
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			unmoderated, err := storage.ReadCommentUnmoderated(ctx, s.pool)
			if err != nil {
				s.l.ErrorContext(ctx, "Failed to read unmoderated comments.",
					ipcoin.LogErr, err,
				)
				continue
			}
			if len(unmoderated) == 0 {
				continue
			}
			s.l.InfoContext(ctx, "Read unmoderated comments.",
				"count", len(unmoderated),
			)
			openaiCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
			response, err := s.openai.Moderations.New(openaiCtx, openai.ModerationNewParams{
				Input: openai.ModerationNewParamsInputUnion{
					OfStringArray: slices.Collect(func(yield func(string) bool) {
						for _, c := range unmoderated {
							if !yield(c.Message) {
								return
							}
						}
					}),
				},
				Model: openai.ModerationModelOmniModerationLatest,
			})
			cancel()
			if err != nil {
				s.l.ErrorContext(ctx, "Failed to get OpenAI moderation response.",
					ipcoin.LogErr, err,
				)
				continue
			}
			moderations := make([]storage.CommentModeration, 0)
			for i, result := range response.Results {
				if i >= len(unmoderated) {
					s.l.ErrorContext(ctx, "OpenAI moderation response has more results than unmoderated comments.",
						"unmoderated_count", len(unmoderated),
						"results_count", len(response.Results),
					)
					break
				}
				m := storage.CommentModeration{
					CommentID: unmoderated[i].ID,
					Censored:  result.Flagged,
					Note:      "Moderated by OpenAI.",
				}
				moderations = append(moderations, m)
			}
			// TODO Use DB transaction?
			err = storage.CreateCommentModeration(ctx, s.pool, moderations, s.clock.Now())
			if err != nil {
				s.l.ErrorContext(ctx, "Failed to create comment moderations.",
					ipcoin.LogErr, err,
				)
				continue
			}
		}
	}
}
