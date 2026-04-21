package service

import (
	"context"

	"github.com/remitly-task/stocksim/internal/domain"
	"github.com/remitly-task/stocksim/internal/store"
)

// LogService retrieves audit log entries.
type LogService struct {
	store store.Store
}

// NewLogService creates a new LogService.
func NewLogService(s store.Store) *LogService {
	return &LogService{store: s}
}

// GetLog returns all audit log entries in insertion order.
func (l *LogService) GetLog(ctx context.Context) ([]domain.LogEntry, error) {
	return l.store.GetLog(ctx)
}
