package server

import (
	"context"
	"net/netip"
	"sync"
	"time"

	"golang.org/x/time/rate"
)

type AddressLimiter interface {
	Wait(ctx context.Context, addr netip.Addr) error
}

type addressLimiterMemValue struct {
	l     *rate.Limiter
	timer *time.Timer
}

type addressLimiterMem struct {
	burst       int
	deleteAfter time.Duration
	m           map[netip.Addr]addressLimiterMemValue
	mux         sync.Mutex
	rate        rate.Limit
}

func NewAddressLimiterMem(burst int, rateLimit rate.Limit) AddressLimiter {
	return &addressLimiterMem{
		burst:       burst,
		deleteAfter: time.Hour,
		m:           make(map[netip.Addr]addressLimiterMemValue),
		rate:        rateLimit,
	}
}

func (a *addressLimiterMem) Wait(ctx context.Context, addr netip.Addr) error {
	a.mux.Lock()
	limiter, ok := a.m[addr]
	if !ok {
		after := time.AfterFunc(a.deleteAfter, func() {
			a.mux.Lock()
			delete(a.m, addr)
			a.mux.Unlock()
		})
		limiter = addressLimiterMemValue{
			l:     rate.NewLimiter(a.rate, a.burst),
			timer: after,
		}
		a.m[addr] = limiter
	}
	a.mux.Unlock()
	limiter.timer.Reset(a.deleteAfter)
	return limiter.l.Wait(ctx)
}
