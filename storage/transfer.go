package storage

import (
	"context"
	"fmt"
	"net/netip"
	"time"

	"github.com/google/uuid"
)

type Transfer struct {
	Created   time.Time  `db:"created"`
	ID        uuid.UUID  `db:"id"`
	Sender    netip.Addr `db:"sender"`
	Recipient netip.Addr `db:"recipient"`
	Amount    int64      `db:"amount"`
}

type CreateTransferRequest struct {
	Amount    int64
	Sender    netip.Addr
	Now       time.Time
	Recipient netip.Addr
}

type CreateTransferResponse struct {
	Transfer      Transfer
	SenderBalance int64
}

func CreateTransfer(ctx context.Context, db dbConn, request CreateTransferRequest) (CreateTransferResponse, error) {
	checkBalanceRequest := GetBalanceRequest{
		Address: request.Sender,
		Now:     request.Now,
	}
	balance, err := GetBalance(ctx, db, checkBalanceRequest)
	if err != nil {
		return CreateTransferResponse{}, fmt.Errorf("failed to check balance for transfer: %w", err)
	}

	balance -= request.Amount
	if balance < 0 {
		return CreateTransferResponse{}, fmt.Errorf("cannot complete transfer: %w", ErrInsufficientBalance)
	}

	query := `
INSERT INTO transfer (created, id, sender, recipient, amount)
VALUES ($1, $2, $3, $4, $5)
`
	id := uuid.New()
	_, err = db.Exec(ctx, query, request.Now, id, request.Sender, request.Recipient, request.Amount)
	if err != nil {
		return CreateTransferResponse{}, fmt.Errorf("failed to insert new transfer: %w", err)
	}

	response := CreateTransferResponse{
		Transfer: Transfer{
			Created:   request.Now,
			ID:        id,
			Sender:    request.Sender,
			Recipient: request.Recipient,
			Amount:    request.Amount,
		},
		SenderBalance: balance,
	}
	return response, nil
}
