package main

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/netip"
	"time"

	"github.com/MicahParks/ipcoin"
)

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	l := slog.Default()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://ipcoin.recentralized.nexus/v1/balance", nil)
	if err != nil {
		l.ErrorContext(ctx, "Failed to create request.",
			ipcoin.LogErr, err,
		)
		return
	}
	// req.Header.Set("CF-Connecting-IP", "0.0.0.0")
	client := getClient(false)
	resp, err := client.Do(req)
	if err != nil {
		l.ErrorContext(ctx, "Failed to make request.",
			ipcoin.LogErr, err,
		)
		return
	}
	defer resp.Body.Close()

	b, err := io.ReadAll(resp.Body)
	if err != nil {
		l.ErrorContext(ctx, "Failed to read response.",
			ipcoin.LogErr, err,
		)
		return
	}

	l.InfoContext(ctx, "Response.",
		"status", resp.Status,
		"body", string(b),
	)

	type response struct {
		Balance struct {
			Timestamp time.Time `json:"timestamp"`
			Address   []byte    `json:"address"`
			Available string    `json:"available"`
		} `json:"balance"`
	}

	var r response
	err = json.Unmarshal(b, &r)
	if err != nil {
		l.ErrorContext(ctx, "Failed to unmarshal response.",
			ipcoin.LogErr, err,
		)
		return
	}
	addr, ok := netip.AddrFromSlice(r.Balance.Address)
	if !ok {
		l.ErrorContext(ctx, "Failed to parse address.",
			ipcoin.LogErr, err,
		)
		return
	}
	l.InfoContext(ctx, "Response unmarshalled.",
		"timestamp", r.Balance.Timestamp,
		"address", addr.String(),
		"available", r.Balance.Available,
	)
}

func getClient(useIPv4 bool) *http.Client {
	if !useIPv4 {
		return http.DefaultClient
	}

	dialContext := func(ctx context.Context, network, addr string) (net.Conn, error) {
		d := &net.Dialer{
			Timeout:   30 * time.Second,
			KeepAlive: 30 * time.Second,
		}
		return d.DialContext(ctx, "tcp4", addr)
	}

	transport := &http.Transport{
		DialContext: dialContext,
	}

	client := &http.Client{
		Transport: transport,
	}
	return client
}
