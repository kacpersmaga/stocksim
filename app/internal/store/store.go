package store

import (
	"context"

	"github.com/remitly-task/stocksim/internal/domain"
)

// Store defines the persistence interface for the stock market simulation.
type Store interface {
	SetBankStocks(ctx context.Context, stocks []domain.Stock) error
	GetBankStocks(ctx context.Context) ([]domain.Stock, error)
	ExecuteTrade(ctx context.Context, walletID, stockName string, t domain.TradeType) error
	GetWallet(ctx context.Context, walletID string) (domain.Wallet, error)
	GetWalletStock(ctx context.Context, walletID, stockName string) (int, error)
	AppendLog(ctx context.Context, entry domain.LogEntry) error
	GetLog(ctx context.Context) ([]domain.LogEntry, error)
}
