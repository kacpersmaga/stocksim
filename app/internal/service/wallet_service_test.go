package service_test

import (
	"context"
	"errors"
	"testing"

	"github.com/remitly-task/stocksim/internal/domain"
	"github.com/remitly-task/stocksim/internal/service"
)

// mockStore is a test double for the store.Store interface.
type mockStore struct {
	executeTradeErr error
	appendLogErr    error
	wallet          domain.Wallet
	walletStock     int
	logCalls        int
}

func (m *mockStore) SetBankStocks(_ context.Context, _ []domain.Stock) error { return nil }
func (m *mockStore) GetBankStocks(_ context.Context) ([]domain.Stock, error) {
	return nil, nil
}
func (m *mockStore) ExecuteTrade(_ context.Context, _, _ string, _ domain.TradeType) error {
	return m.executeTradeErr
}
func (m *mockStore) GetWallet(_ context.Context, id string) (domain.Wallet, error) {
	return m.wallet, nil
}
func (m *mockStore) GetWalletStock(_ context.Context, _, _ string) (int, error) {
	return m.walletStock, nil
}
func (m *mockStore) AppendLog(_ context.Context, _ domain.LogEntry) error {
	m.logCalls++
	return m.appendLogErr
}
func (m *mockStore) GetLog(_ context.Context) ([]domain.LogEntry, error) { return nil, nil }

func TestTrade_BuySuccess(t *testing.T) {
	ms := &mockStore{}
	svc := service.NewWalletService(ms)

	err := svc.Trade(context.Background(), "wallet1", "AAPL", domain.TradeTypeBuy)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if ms.logCalls != 1 {
		t.Errorf("expected 1 log call, got %d", ms.logCalls)
	}
}

func TestTrade_SellSuccess(t *testing.T) {
	ms := &mockStore{}
	svc := service.NewWalletService(ms)

	err := svc.Trade(context.Background(), "wallet1", "AAPL", domain.TradeTypeSell)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if ms.logCalls != 1 {
		t.Errorf("expected 1 log call, got %d", ms.logCalls)
	}
}

func TestTrade_InvalidType(t *testing.T) {
	ms := &mockStore{}
	svc := service.NewWalletService(ms)

	err := svc.Trade(context.Background(), "wallet1", "AAPL", domain.TradeType("hold"))
	if !errors.Is(err, domain.ErrInvalidTradeType) {
		t.Fatalf("expected ErrInvalidTradeType, got %v", err)
	}
	if ms.logCalls != 0 {
		t.Error("log should not be called on invalid type")
	}
}

func TestTrade_StockNotFound(t *testing.T) {
	ms := &mockStore{executeTradeErr: domain.ErrStockNotFound}
	svc := service.NewWalletService(ms)

	err := svc.Trade(context.Background(), "wallet1", "UNKNOWN", domain.TradeTypeBuy)
	if !errors.Is(err, domain.ErrStockNotFound) {
		t.Fatalf("expected ErrStockNotFound, got %v", err)
	}
	if ms.logCalls != 0 {
		t.Error("log should not be called on failure")
	}
}

func TestTrade_BankOutOfStock(t *testing.T) {
	ms := &mockStore{executeTradeErr: domain.ErrBankOutOfStock}
	svc := service.NewWalletService(ms)

	err := svc.Trade(context.Background(), "wallet1", "AAPL", domain.TradeTypeBuy)
	if !errors.Is(err, domain.ErrBankOutOfStock) {
		t.Fatalf("expected ErrBankOutOfStock, got %v", err)
	}
	if ms.logCalls != 0 {
		t.Error("log should not be called on failure")
	}
}

func TestTrade_WalletOutOfStock(t *testing.T) {
	ms := &mockStore{executeTradeErr: domain.ErrWalletOutOfStock}
	svc := service.NewWalletService(ms)

	err := svc.Trade(context.Background(), "wallet1", "AAPL", domain.TradeTypeSell)
	if !errors.Is(err, domain.ErrWalletOutOfStock) {
		t.Fatalf("expected ErrWalletOutOfStock, got %v", err)
	}
	if ms.logCalls != 0 {
		t.Error("log should not be called on failure")
	}
}

func TestTrade_LogNotCalledOnExecuteError(t *testing.T) {
	ms := &mockStore{executeTradeErr: errors.New("redis timeout")}
	svc := service.NewWalletService(ms)

	_ = svc.Trade(context.Background(), "wallet1", "AAPL", domain.TradeTypeBuy)
	if ms.logCalls != 0 {
		t.Error("log should not be called when trade fails")
	}
}

func TestGetWallet(t *testing.T) {
	expected := domain.Wallet{
		ID:     "wallet1",
		Stocks: []domain.Stock{{Name: "AAPL", Quantity: 3}},
	}
	ms := &mockStore{wallet: expected}
	svc := service.NewWalletService(ms)

	got, err := svc.GetWallet(context.Background(), "wallet1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.ID != expected.ID || len(got.Stocks) != 1 {
		t.Errorf("unexpected wallet: %+v", got)
	}
}

func TestGetWalletStock(t *testing.T) {
	ms := &mockStore{walletStock: 7}
	svc := service.NewWalletService(ms)

	qty, err := svc.GetWalletStock(context.Background(), "wallet1", "AAPL")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if qty != 7 {
		t.Errorf("expected 7, got %d", qty)
	}
}
