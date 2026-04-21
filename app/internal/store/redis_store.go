package store

import (
	"context"
	"encoding/json"
	_ "embed"
	"errors"
	"fmt"
	"strconv"

	"github.com/redis/go-redis/v9"
	"github.com/remitly-task/stocksim/internal/domain"
)

//go:embed lua/buy_stock.lua
var buyStockLua string

//go:embed lua/sell_stock.lua
var sellStockLua string

const (
	keyBankStocks     = "bank:stocks"
	keyBankStockNames = "bank:stock_names"
	keyAuditLog       = "audit:log"
)

func walletKey(walletID string) string {
	return fmt.Sprintf("wallet:%s:stocks", walletID)
}

// RedisStore implements Store using Redis as the backend.
type RedisStore struct {
	client      *redis.Client
	buyScript   *redis.Script
	sellScript  *redis.Script
}

// NewRedisStore creates a new RedisStore and pre-loads Lua scripts.
func NewRedisStore(client *redis.Client) (*RedisStore, error) {
	s := &RedisStore{
		client:     client,
		buyScript:  redis.NewScript(buyStockLua),
		sellScript: redis.NewScript(sellStockLua),
	}

	ctx := context.Background()
	if err := s.buyScript.Load(ctx, client).Err(); err != nil {
		return nil, fmt.Errorf("loading buy script: %w", err)
	}
	if err := s.sellScript.Load(ctx, client).Err(); err != nil {
		return nil, fmt.Errorf("loading sell script: %w", err)
	}

	return s, nil
}

// SetBankStocks atomically replaces the entire bank stock inventory.
func (s *RedisStore) SetBankStocks(ctx context.Context, stocks []domain.Stock) error {
	_, err := s.client.TxPipelined(ctx, func(pipe redis.Pipeliner) error {
		pipe.Del(ctx, keyBankStocks, keyBankStockNames)

		if len(stocks) == 0 {
			return nil
		}

		stockMap := make(map[string]interface{}, len(stocks))
		names := make([]interface{}, len(stocks))
		for i, st := range stocks {
			stockMap[st.Name] = st.Quantity
			names[i] = st.Name
		}
		pipe.HMSet(ctx, keyBankStocks, stockMap)
		pipe.SAdd(ctx, keyBankStockNames, names...)
		return nil
	})
	return err
}

// GetBankStocks retrieves all stocks currently in the bank.
func (s *RedisStore) GetBankStocks(ctx context.Context) ([]domain.Stock, error) {
	fields, err := s.client.HGetAll(ctx, keyBankStocks).Result()
	if err != nil {
		return nil, err
	}

	stocks := make([]domain.Stock, 0, len(fields))
	for name, qtyStr := range fields {
		qty, _ := strconv.Atoi(qtyStr)
		stocks = append(stocks, domain.Stock{Name: name, Quantity: qty})
	}
	return stocks, nil
}

// ExecuteTrade runs the atomic buy or sell Lua script.
func (s *RedisStore) ExecuteTrade(ctx context.Context, walletID, stockName string, t domain.TradeType) error {
	wKey := walletKey(walletID)
	keys := []string{keyBankStocks, wKey, keyBankStockNames}
	args := []interface{}{stockName}

	var script *redis.Script
	switch t {
	case domain.TradeTypeBuy:
		script = s.buyScript
	case domain.TradeTypeSell:
		script = s.sellScript
	default:
		return domain.ErrInvalidTradeType
	}

	result, err := script.Run(ctx, s.client, keys, args...).Text()
	if err != nil {
		return fmt.Errorf("lua script error: %w", err)
	}

	switch result {
	case "OK":
		return nil
	case "STOCK_NOT_FOUND":
		return domain.ErrStockNotFound
	case "BANK_OUT_OF_STOCK":
		return domain.ErrBankOutOfStock
	case "WALLET_OUT_OF_STOCK":
		return domain.ErrWalletOutOfStock
	default:
		return fmt.Errorf("unexpected lua result: %s", result)
	}
}

// GetWallet retrieves a user's full wallet. Returns an empty wallet if none exists.
func (s *RedisStore) GetWallet(ctx context.Context, walletID string) (domain.Wallet, error) {
	fields, err := s.client.HGetAll(ctx, walletKey(walletID)).Result()
	if err != nil {
		return domain.Wallet{}, err
	}

	stocks := make([]domain.Stock, 0, len(fields))
	for name, qtyStr := range fields {
		qty, _ := strconv.Atoi(qtyStr)
		if qty > 0 {
			stocks = append(stocks, domain.Stock{Name: name, Quantity: qty})
		}
	}
	return domain.Wallet{ID: walletID, Stocks: stocks}, nil
}

// GetWalletStock retrieves the quantity of a specific stock in a wallet.
func (s *RedisStore) GetWalletStock(ctx context.Context, walletID, stockName string) (int, error) {
	val, err := s.client.HGet(ctx, walletKey(walletID), stockName).Result()
	if errors.Is(err, redis.Nil) {
		return 0, nil
	}
	if err != nil {
		return 0, err
	}
	qty, _ := strconv.Atoi(val)
	return qty, nil
}

// AppendLog appends a log entry to the audit trail.
func (s *RedisStore) AppendLog(ctx context.Context, entry domain.LogEntry) error {
	data, err := json.Marshal(entry)
	if err != nil {
		return err
	}
	return s.client.RPush(ctx, keyAuditLog, data).Err()
}

// GetLog retrieves all audit log entries in insertion order.
func (s *RedisStore) GetLog(ctx context.Context) ([]domain.LogEntry, error) {
	items, err := s.client.LRange(ctx, keyAuditLog, 0, -1).Result()
	if err != nil {
		return nil, err
	}

	entries := make([]domain.LogEntry, 0, len(items))
	for _, item := range items {
		var entry domain.LogEntry
		if err := json.Unmarshal([]byte(item), &entry); err != nil {
			return nil, fmt.Errorf("parsing log entry: %w", err)
		}
		entries = append(entries, entry)
	}
	return entries, nil
}
