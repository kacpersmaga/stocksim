package handler_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/remitly-task/stocksim/internal/domain"
	"github.com/remitly-task/stocksim/internal/handler"
)

// --- Mock services ---

type mockWalletService struct {
	tradeErr      error
	wallet        domain.Wallet
	walletStock   int
}

func (m *mockWalletService) Trade(_ context.Context, _, _ string, _ domain.TradeType) error {
	return m.tradeErr
}
func (m *mockWalletService) GetWallet(_ context.Context, id string) (domain.Wallet, error) {
	if m.wallet.ID == "" {
		return domain.Wallet{ID: id, Stocks: []domain.Stock{}}, nil
	}
	return m.wallet, nil
}
func (m *mockWalletService) GetWalletStock(_ context.Context, _, _ string) (int, error) {
	return m.walletStock, nil
}

type mockBankService struct {
	setBankErr error
	stocks     []domain.Stock
}

func (m *mockBankService) SetBankStocks(_ context.Context, _ []domain.Stock) error {
	return m.setBankErr
}
func (m *mockBankService) GetBankStocks(_ context.Context) ([]domain.Stock, error) {
	return m.stocks, nil
}

type mockLogService struct {
	entries []domain.LogEntry
}

func (m *mockLogService) GetLog(_ context.Context) ([]domain.LogEntry, error) {
	return m.entries, nil
}

func newRouter(ws *mockWalletService, bs *mockBankService, ls *mockLogService) http.Handler {
	return handler.NewRouter(handler.Services{
		Wallet: ws,
		Bank:   bs,
		Log:    ls,
	})
}

// --- Tests ---

func TestTrade_200(t *testing.T) {
	router := newRouter(&mockWalletService{}, &mockBankService{}, &mockLogService{})
	body := bytes.NewBufferString(`{"type":"buy"}`)
	req := httptest.NewRequest(http.MethodPost, "/wallets/w1/stocks/AAPL", body)
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rr.Code)
	}
}

func TestTrade_404_StockNotFound(t *testing.T) {
	router := newRouter(&mockWalletService{tradeErr: domain.ErrStockNotFound}, &mockBankService{}, &mockLogService{})
	body := bytes.NewBufferString(`{"type":"buy"}`)
	req := httptest.NewRequest(http.MethodPost, "/wallets/w1/stocks/UNKNOWN", body)
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", rr.Code)
	}
}

func TestTrade_400_BankOutOfStock(t *testing.T) {
	router := newRouter(&mockWalletService{tradeErr: domain.ErrBankOutOfStock}, &mockBankService{}, &mockLogService{})
	body := bytes.NewBufferString(`{"type":"buy"}`)
	req := httptest.NewRequest(http.MethodPost, "/wallets/w1/stocks/AAPL", body)
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rr.Code)
	}
}

func TestTrade_400_WalletOutOfStock(t *testing.T) {
	router := newRouter(&mockWalletService{tradeErr: domain.ErrWalletOutOfStock}, &mockBankService{}, &mockLogService{})
	body := bytes.NewBufferString(`{"type":"sell"}`)
	req := httptest.NewRequest(http.MethodPost, "/wallets/w1/stocks/AAPL", body)
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rr.Code)
	}
}

func TestTrade_400_InvalidType(t *testing.T) {
	router := newRouter(&mockWalletService{}, &mockBankService{}, &mockLogService{})
	body := bytes.NewBufferString(`{"type":"hold"}`)
	req := httptest.NewRequest(http.MethodPost, "/wallets/w1/stocks/AAPL", body)
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rr.Code)
	}
}

func TestGetWallet_EmptyWallet(t *testing.T) {
	router := newRouter(&mockWalletService{}, &mockBankService{}, &mockLogService{})
	req := httptest.NewRequest(http.MethodGet, "/wallets/w1", nil)
	rr := httptest.NewRecorder()

	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rr.Code)
	}

	var resp domain.Wallet
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp.Stocks == nil {
		t.Error("stocks should not be nil")
	}
}

func TestGetWallet_WithStocks(t *testing.T) {
	ws := &mockWalletService{
		wallet: domain.Wallet{
			ID:     "w1",
			Stocks: []domain.Stock{{Name: "AAPL", Quantity: 3}},
		},
	}
	router := newRouter(ws, &mockBankService{}, &mockLogService{})
	req := httptest.NewRequest(http.MethodGet, "/wallets/w1", nil)
	rr := httptest.NewRecorder()

	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rr.Code)
	}
}

func TestGetWalletStock_ReturnsNumber(t *testing.T) {
	ws := &mockWalletService{walletStock: 42}
	router := newRouter(ws, &mockBankService{}, &mockLogService{})
	req := httptest.NewRequest(http.MethodGet, "/wallets/w1/stocks/AAPL", nil)
	rr := httptest.NewRecorder()

	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rr.Code)
	}

	var qty int
	if err := json.NewDecoder(rr.Body).Decode(&qty); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if qty != 42 {
		t.Errorf("expected 42, got %d", qty)
	}
}

func TestGetWalletStock_ZeroCase(t *testing.T) {
	router := newRouter(&mockWalletService{walletStock: 0}, &mockBankService{}, &mockLogService{})
	req := httptest.NewRequest(http.MethodGet, "/wallets/w1/stocks/UNKNOWN", nil)
	rr := httptest.NewRecorder()

	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rr.Code)
	}
}

func TestGetBankStocks_Empty(t *testing.T) {
	router := newRouter(&mockWalletService{}, &mockBankService{}, &mockLogService{})
	req := httptest.NewRequest(http.MethodGet, "/stocks", nil)
	rr := httptest.NewRecorder()

	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rr.Code)
	}

	var resp struct {
		Stocks []domain.Stock `json:"stocks"`
	}
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp.Stocks == nil {
		t.Error("stocks should not be nil")
	}
}

func TestGetBankStocks_Populated(t *testing.T) {
	bs := &mockBankService{stocks: []domain.Stock{{Name: "AAPL", Quantity: 10}}}
	router := newRouter(&mockWalletService{}, bs, &mockLogService{})
	req := httptest.NewRequest(http.MethodGet, "/stocks", nil)
	rr := httptest.NewRecorder()

	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rr.Code)
	}
}

func TestSetBankStocks_200(t *testing.T) {
	router := newRouter(&mockWalletService{}, &mockBankService{}, &mockLogService{})
	body := bytes.NewBufferString(`{"stocks":[{"name":"AAPL","quantity":10}]}`)
	req := httptest.NewRequest(http.MethodPost, "/stocks", body)
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rr.Code)
	}
}

func TestSetBankStocks_400_ValidationError(t *testing.T) {
	bs := &mockBankService{setBankErr: domain.NewValidationError("stock has negative quantity")}
	router := newRouter(&mockWalletService{}, bs, &mockLogService{})
	body := bytes.NewBufferString(`{"stocks":[{"name":"AAPL","quantity":-1}]}`)
	req := httptest.NewRequest(http.MethodPost, "/stocks", body)
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rr.Code)
	}
}

func TestGetLog_Empty(t *testing.T) {
	router := newRouter(&mockWalletService{}, &mockBankService{}, &mockLogService{})
	req := httptest.NewRequest(http.MethodGet, "/log", nil)
	rr := httptest.NewRecorder()

	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rr.Code)
	}

	var resp struct {
		Entries []domain.LogEntry `json:"entries"`
	}
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp.Entries == nil {
		t.Error("entries should not be nil")
	}
}

func TestGetLog_Populated(t *testing.T) {
	ls := &mockLogService{entries: []domain.LogEntry{
		{WalletID: "w1", StockName: "AAPL", Type: domain.TradeTypeBuy, Timestamp: "2024-01-01T00:00:00Z"},
		{WalletID: "w1", StockName: "AAPL", Type: domain.TradeTypeSell, Timestamp: "2024-01-02T00:00:00Z"},
	}}
	router := newRouter(&mockWalletService{}, &mockBankService{}, ls)
	req := httptest.NewRequest(http.MethodGet, "/log", nil)
	rr := httptest.NewRecorder()

	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rr.Code)
	}

	var resp struct {
		Entries []domain.LogEntry `json:"entries"`
	}
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(resp.Entries) != 2 {
		t.Errorf("expected 2 entries, got %d", len(resp.Entries))
	}
	if resp.Entries[0].StockName != "AAPL" {
		t.Errorf("expected AAPL first, got %s", resp.Entries[0].StockName)
	}
}

func TestHealthz(t *testing.T) {
	router := newRouter(&mockWalletService{}, &mockBankService{}, &mockLogService{})
	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	rr := httptest.NewRecorder()

	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rr.Code)
	}
}
