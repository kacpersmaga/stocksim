package handler

import (
	"context"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/remitly-task/stocksim/internal/domain"
)

// WalletServicer is the interface the wallet handler depends on.
type WalletServicer interface {
	Trade(ctx context.Context, walletID, stockName string, tradeType domain.TradeType) error
	GetWallet(ctx context.Context, walletID string) (domain.Wallet, error)
	GetWalletStock(ctx context.Context, walletID, stockName string) (int, error)
}

// BankServicer is the interface the bank handler depends on.
type BankServicer interface {
	SetBankStocks(ctx context.Context, stocks []domain.Stock) error
	GetBankStocks(ctx context.Context) ([]domain.Stock, error)
}

// LogServicer is the interface the log handler depends on.
type LogServicer interface {
	GetLog(ctx context.Context) ([]domain.LogEntry, error)
}

// Services groups all service dependencies for handlers.
type Services struct {
	Wallet WalletServicer
	Bank   BankServicer
	Log    LogServicer
}

// NewRouter builds and returns the chi router with all routes and middleware.
func NewRouter(svc Services) http.Handler {
	r := chi.NewRouter()

	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(10 * time.Second))

	wh := &walletHandler{wallet: svc.Wallet}
	bh := &bankHandler{bank: svc.Bank}
	lh := &logHandler{log: svc.Log}
	ch := &chaosHandler{}

	r.Get("/healthz", healthzHandler)
	r.Get("/readyz", healthzHandler)

	r.Post("/stocks", bh.SetBankStocks)
	r.Get("/stocks", bh.GetBankStocks)

	r.Post("/wallets/{walletID}/stocks/{stockName}", wh.Trade)
	r.Get("/wallets/{walletID}", wh.GetWallet)
	r.Get("/wallets/{walletID}/stocks/{stockName}", wh.GetWalletStock)

	r.Get("/log", lh.GetLog)

	r.Post("/chaos", ch.Chaos)

	return r
}

func healthzHandler(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}
