package domain

// TradeType represents a buy or sell operation.
type TradeType string

const (
	TradeTypeBuy  TradeType = "buy"
	TradeTypeSell TradeType = "sell"
)

// Stock represents a single stock entry with its name and quantity.
type Stock struct {
	Name     string `json:"name"`
	Quantity int    `json:"quantity"`
}

// Wallet holds a user's stock portfolio.
type Wallet struct {
	ID     string  `json:"id"`
	Stocks []Stock `json:"stocks"`
}

// LogEntry records a successful trade operation for audit purposes.
type LogEntry struct {
	WalletID  string    `json:"wallet_id"`
	StockName string    `json:"stock_name"`
	Type      TradeType `json:"type"`
	Timestamp string    `json:"timestamp"`
}
