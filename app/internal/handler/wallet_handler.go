package handler

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/remitly-task/stocksim/internal/domain"
)

type walletHandler struct {
	wallet WalletServicer
}

type tradeRequest struct {
	Type domain.TradeType `json:"type"`
}

func (h *walletHandler) Trade(w http.ResponseWriter, r *http.Request) {
	walletID := chi.URLParam(r, "walletID")
	stockName := chi.URLParam(r, "stockName")

	var req tradeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Type != domain.TradeTypeBuy && req.Type != domain.TradeTypeSell {
		writeError(w, http.StatusBadRequest, domain.ErrInvalidTradeType.Error())
		return
	}

	err := h.wallet.Trade(r.Context(), walletID, stockName, req.Type)
	if err == nil {
		writeJSON(w, http.StatusOK, map[string]any{})
		return
	}

	switch {
	case errors.Is(err, domain.ErrStockNotFound):
		writeError(w, http.StatusNotFound, err.Error())
	case errors.Is(err, domain.ErrBankOutOfStock):
		writeError(w, http.StatusBadRequest, err.Error())
	case errors.Is(err, domain.ErrWalletOutOfStock):
		writeError(w, http.StatusBadRequest, err.Error())
	case errors.Is(err, domain.ErrInvalidTradeType):
		writeError(w, http.StatusBadRequest, err.Error())
	default:
		writeError(w, http.StatusInternalServerError, "internal server error")
	}
}

func (h *walletHandler) GetWallet(w http.ResponseWriter, r *http.Request) {
	walletID := chi.URLParam(r, "walletID")

	wallet, err := h.wallet.GetWallet(r.Context(), walletID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	// Ensure stocks is always an array, never null
	if wallet.Stocks == nil {
		wallet.Stocks = []domain.Stock{}
	}
	writeJSON(w, http.StatusOK, wallet)
}

func (h *walletHandler) GetWalletStock(w http.ResponseWriter, r *http.Request) {
	walletID := chi.URLParam(r, "walletID")
	stockName := chi.URLParam(r, "stockName")

	qty, err := h.wallet.GetWalletStock(r.Context(), walletID, stockName)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	// Spec says: returns a single number, like: 99
	json.NewEncoder(w).Encode(qty) //nolint:errcheck
}
