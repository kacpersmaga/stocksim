package service

import (
	"context"
	"fmt"

	"github.com/remitly-task/stocksim/internal/domain"
	"github.com/remitly-task/stocksim/internal/store"
)

// BankService manages bank stock inventory.
type BankService struct {
	store store.Store
}

// NewBankService creates a new BankService.
func NewBankService(s store.Store) *BankService {
	return &BankService{store: s}
}

// SetBankStocks validates and replaces the entire bank inventory atomically.
func (b *BankService) SetBankStocks(ctx context.Context, stocks []domain.Stock) error {
	for _, s := range stocks {
		if s.Quantity < 0 {
			return domain.NewValidationError(fmt.Sprintf("stock %q has negative quantity %d", s.Name, s.Quantity))
		}
	}
	return b.store.SetBankStocks(ctx, stocks)
}

// GetBankStocks retrieves all stocks currently in the bank.
func (b *BankService) GetBankStocks(ctx context.Context) ([]domain.Stock, error) {
	return b.store.GetBankStocks(ctx)
}
