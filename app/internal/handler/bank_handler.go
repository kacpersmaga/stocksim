package handler

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/remitly-task/stocksim/internal/domain"
)

type bankHandler struct {
	bank BankServicer
}

type setBankRequest struct {
	Stocks []domain.Stock `json:"stocks"`
}

type getBankResponse struct {
	Stocks []domain.Stock `json:"stocks"`
}

func (h *bankHandler) SetBankStocks(w http.ResponseWriter, r *http.Request) {
	var req setBankRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Stocks == nil {
		req.Stocks = []domain.Stock{}
	}

	if err := h.bank.SetBankStocks(r.Context(), req.Stocks); err != nil {
		var ve *domain.ValidationError
		if errors.As(err, &ve) {
			writeError(w, http.StatusBadRequest, ve.Error())
			return
		}
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{})
}

func (h *bankHandler) GetBankStocks(w http.ResponseWriter, r *http.Request) {
	stocks, err := h.bank.GetBankStocks(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	if stocks == nil {
		stocks = []domain.Stock{}
	}
	writeJSON(w, http.StatusOK, getBankResponse{Stocks: stocks})
}

