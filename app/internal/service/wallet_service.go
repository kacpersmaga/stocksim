package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/remitly-task/stocksim/internal/domain"
	"github.com/remitly-task/stocksim/internal/store"
)

// WalletService handles wallet trade operations.
type WalletService struct {
	store store.Store
}

// NewWalletService creates a new WalletService.
func NewWalletService(s store.Store) *WalletService {
	return &WalletService{store: s}
}

// Trade executes a buy or sell operation for a wallet.
func (w *WalletService) Trade(ctx context.Context, walletID, stockName string, tradeType domain.TradeType) error {
	if tradeType != domain.TradeTypeBuy && tradeType != domain.TradeTypeSell {
		return domain.ErrInvalidTradeType
	}

	if err := w.store.ExecuteTrade(ctx, walletID, stockName, tradeType); err != nil {
		// Map store errors to domain errors
		if errors.Is(err, domain.ErrStockNotFound) ||
			errors.Is(err, domain.ErrBankOutOfStock) ||
			errors.Is(err, domain.ErrWalletOutOfStock) {
			return err
		}
		return fmt.Errorf("executing trade: %w", err)
	}

	entry := domain.LogEntry{
		WalletID:  walletID,
		StockName: stockName,
		Type:      tradeType,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	}
	if err := w.store.AppendLog(ctx, entry); err != nil {
		// Non-fatal: trade succeeded, log failure is acceptable
		return fmt.Errorf("appending log (trade succeeded): %w", err)
	}

	return nil
}

// GetWallet retrieves a wallet's full portfolio.
func (w *WalletService) GetWallet(ctx context.Context, walletID string) (domain.Wallet, error) {
	return w.store.GetWallet(ctx, walletID)
}

// GetWalletStock retrieves the quantity of a specific stock in a wallet.
func (w *WalletService) GetWalletStock(ctx context.Context, walletID, stockName string) (int, error) {
	return w.store.GetWalletStock(ctx, walletID, stockName)
}
