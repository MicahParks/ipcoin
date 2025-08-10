package server

import (
	"context"
	"net/netip"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestAddrLocker_WithLock_ExecutesFunction(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	locker := newAddrLocker(10 * time.Millisecond)
	for _, ip := range []string{"192.168.0.1", "::1"} {
		addr := netip.MustParseAddr(ip)

		var executed atomic.Bool
		locker.WithLock(ctx, addr, func() {
			executed.Store(true)
		})

		if !executed.Load() {
			t.Fatal("Function did not execute.")
		}
	}
}

func TestAddrLocker_WithLock_ContextCancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	locker := newAddrLocker(10 * time.Millisecond)
	for _, ip := range []string{"192.168.0.1", "::1"} {
		addr := netip.MustParseAddr(ip)

		var executed atomic.Bool
		locker.WithLock(ctx, addr, func() {
			executed.Store(true)
		})

		if executed.Load() {
			t.Fatal("Function with cancelled context should not execute.")
		}
	}
}

func TestAddrLocker_WithLock_ConcurrentAccess(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	locker := newAddrLocker(10 * time.Millisecond)
	for _, ip := range []string{"192.168.0.1", "::1"} {
		addr := netip.MustParseAddr(ip)

		var counter atomic.Int64
		var wgFinish, wgStart sync.WaitGroup
		const atTheSameTime = 1000
		wgFinish.Add(atTheSameTime)
		wgStart.Add(1)

		for i := 0; i < atTheSameTime; i++ {
			go func() {
				defer wgFinish.Done()
				wgStart.Wait()
				locker.WithLock(ctx, addr, func() {
					counter.Add(1)
				})
			}()
		}
		wgStart.Done()
		wgFinish.Wait()

		if counter.Load() != atTheSameTime {
			t.Fatalf("Expected all functions to execute.")
		}
	}
}

func TestAddrLocker_WithLock_EntryDeletedAfterDuration(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	locker := newAddrLocker(10 * time.Millisecond)
	for _, ip := range []string{"192.168.0.1", "::1"} {
		addr := netip.MustParseAddr(ip)

		locker.WithLock(ctx, addr, func() {})
		time.Sleep(15 * time.Millisecond)

		locker.mux.Lock()
		_, exists := locker.m[addr]
		locker.mux.Unlock()

		if exists {
			t.Fatal("Expected lock to have been deleted.")
		}
	}
}

func TestAddrLocker_WithLock_ResetDeleteTimer(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	locker := newAddrLocker(10 * time.Millisecond)
	for _, ip := range []string{"192.168.0.1", "::1"} {
		addr := netip.MustParseAddr(ip)

		locker.WithLock(ctx, addr, func() {})
		time.Sleep(5 * time.Millisecond)
		locker.WithLock(ctx, addr, func() {})
		time.Sleep(8 * time.Millisecond)

		locker.mux.Lock()
		_, exists := locker.m[addr]
		locker.mux.Unlock()

		if !exists {
			t.Fatal("Expected lock to exist due to timer reset.")
		}
	}
}
