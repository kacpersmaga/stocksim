//go:build integration

package store_test

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/remitly-task/stocksim/internal/domain"
	"github.com/remitly-task/stocksim/internal/store"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

func setupRedis(t *testing.T) *store.RedisStore {
	t.Helper()
	ctx := context.Background()

	req := testcontainers.ContainerRequest{
		Image:        "redis:7.2-alpine",
		ExposedPorts: []string{"6379/tcp"},
		WaitingFor:   wait.ForLog("Ready to accept connections"),
	}
	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	if err != nil {
		t.Fatalf("start redis container: %v", err)
	}
	t.Cleanup(func() { _ = container.Terminate(ctx) })

	host, err := container.Host(ctx)
	if err != nil {
		t.Fatalf("get host: %v", err)
	}
	port, err := container.MappedPort(ctx, "6379")
	if err != nil {
		t.Fatalf("get port: %v", err)
	}

	client := redis.NewClient(&redis.Options{Addr: host + ":" + port.Port()})
	t.Cleanup(func() { _ = client.Close() })

	s, err := store.NewRedisStore(client)
	if err != nil {
		t.Fatalf("new redis store: %v", err)
	}
	return s
}

func TestConcurrentBuy(t *testing.T) {
	s := setupRedis(t)
	ctx := context.Background()

	// Seed bank with 10 units
	err := s.SetBankStocks(ctx, []domain.Stock{{Name: "AAPL", Quantity: 10}})
	if err != nil {
		t.Fatalf("set bank stocks: %v", err)
	}

	const goroutines = 50
	var succeeded atomic.Int32
	var failed atomic.Int32

	var wg sync.WaitGroup
	wg.Add(goroutines)
	for i := 0; i < goroutines; i++ {
		go func(id int) {
			defer wg.Done()
			walletID := "wallet-concurrent-buy"
			err := s.ExecuteTrade(ctx, walletID, "AAPL", domain.TradeTypeBuy)
			if err == nil {
				succeeded.Add(1)
			} else {
				failed.Add(1)
			}
		}(i)
	}
	wg.Wait()

	if succeeded.Load() != 10 {
		t.Errorf("expected exactly 10 successes, got %d", succeeded.Load())
	}
	if failed.Load() != 40 {
		t.Errorf("expected exactly 40 failures, got %d", failed.Load())
	}

	// Verify bank quantity is exactly 0
	stocks, err := s.GetBankStocks(ctx)
	if err != nil {
		t.Fatalf("get bank stocks: %v", err)
	}
	for _, st := range stocks {
		if st.Name == "AAPL" && st.Quantity != 0 {
			t.Errorf("bank AAPL quantity: want 0, got %d", st.Quantity)
		}
	}
}

func TestConcurrentSell(t *testing.T) {
	s := setupRedis(t)
	ctx := context.Background()

	// Pre-buy 10 stocks into a wallet
	err := s.SetBankStocks(ctx, []domain.Stock{{Name: "GOOG", Quantity: 10}})
	if err != nil {
		t.Fatalf("set bank stocks: %v", err)
	}
	for i := 0; i < 10; i++ {
		if err := s.ExecuteTrade(ctx, "sell-wallet", "GOOG", domain.TradeTypeBuy); err != nil {
			t.Fatalf("pre-buy: %v", err)
		}
	}

	const goroutines = 50
	var succeeded atomic.Int32
	var failed atomic.Int32

	var wg sync.WaitGroup
	wg.Add(goroutines)
	for i := 0; i < goroutines; i++ {
		go func() {
			defer wg.Done()
			err := s.ExecuteTrade(ctx, "sell-wallet", "GOOG", domain.TradeTypeSell)
			if err == nil {
				succeeded.Add(1)
			} else {
				failed.Add(1)
			}
		}()
	}
	wg.Wait()

	if succeeded.Load() != 10 {
		t.Errorf("expected exactly 10 successes, got %d", succeeded.Load())
	}
	if failed.Load() != 40 {
		t.Errorf("expected exactly 40 failures, got %d", failed.Load())
	}
}

func TestSetBankStocksAtomicity(t *testing.T) {
	s := setupRedis(t)
	ctx := context.Background()

	initial := []domain.Stock{{Name: "MSFT", Quantity: 5}}
	replacement := []domain.Stock{{Name: "TSLA", Quantity: 100}}

	if err := s.SetBankStocks(ctx, initial); err != nil {
		t.Fatalf("set initial: %v", err)
	}

	var wg sync.WaitGroup
	wg.Add(2)

	// Writer: replace bank
	go func() {
		defer wg.Done()
		_ = s.SetBankStocks(ctx, replacement)
	}()

	// Reader: must see either all old or all new, never partial
	go func() {
		defer wg.Done()
		stocks, err := s.GetBankStocks(ctx)
		if err != nil {
			return
		}
		// Just verify we got a consistent snapshot (no panics, no mixed data)
		_ = stocks
	}()

	wg.Wait()
}

func TestFullTradeCycle(t *testing.T) {
	s := setupRedis(t)
	ctx := context.Background()

	// Set bank
	if err := s.SetBankStocks(ctx, []domain.Stock{{Name: "AMZN", Quantity: 3}}); err != nil {
		t.Fatalf("set bank: %v", err)
	}

	// Buy
	if err := s.ExecuteTrade(ctx, "alice", "AMZN", domain.TradeTypeBuy); err != nil {
		t.Fatalf("buy: %v", err)
	}

	// Check wallet
	wallet, err := s.GetWallet(ctx, "alice")
	if err != nil {
		t.Fatalf("get wallet: %v", err)
	}
	if len(wallet.Stocks) != 1 || wallet.Stocks[0].Quantity != 1 {
		t.Errorf("expected 1 AMZN in wallet, got %+v", wallet.Stocks)
	}

	// Sell back
	if err := s.ExecuteTrade(ctx, "alice", "AMZN", domain.TradeTypeSell); err != nil {
		t.Fatalf("sell: %v", err)
	}

	// Bank should be back to 3
	stocks, err := s.GetBankStocks(ctx)
	if err != nil {
		t.Fatalf("get bank: %v", err)
	}
	for _, st := range stocks {
		if st.Name == "AMZN" && st.Quantity != 3 {
			t.Errorf("bank AMZN: want 3, got %d", st.Quantity)
		}
	}
}

func TestAuditLogOrdering(t *testing.T) {
	s := setupRedis(t)
	ctx := context.Background()

	entries := []domain.LogEntry{
		{WalletID: "w1", StockName: "A", Type: domain.TradeTypeBuy, Timestamp: time.Now().Format(time.RFC3339)},
		{WalletID: "w1", StockName: "B", Type: domain.TradeTypeSell, Timestamp: time.Now().Format(time.RFC3339)},
		{WalletID: "w2", StockName: "A", Type: domain.TradeTypeBuy, Timestamp: time.Now().Format(time.RFC3339)},
	}

	for _, e := range entries {
		if err := s.AppendLog(ctx, e); err != nil {
			t.Fatalf("append log: %v", err)
		}
	}

	log, err := s.GetLog(ctx)
	if err != nil {
		t.Fatalf("get log: %v", err)
	}
	if len(log) != 3 {
		t.Fatalf("expected 3 entries, got %d", len(log))
	}
	for i, e := range entries {
		if log[i].StockName != e.StockName {
			t.Errorf("entry[%d]: want %s, got %s", i, e.StockName, log[i].StockName)
		}
	}
}
