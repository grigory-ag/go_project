package handler

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"go_project/internal/domain"
)

type TestHandler struct {
	uc  domain.TestUsecase
	log *slog.Logger
}

func NewTestHandler(uc domain.TestUsecase, log *slog.Logger) *TestHandler {
	return &TestHandler{uc: uc, log: log}
}

func (h *TestHandler) Test() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		msg, err := h.uc.GetTestMessage(r.Context())
		if err != nil {
			h.log.Error("failed to get test message", slog.Any("error", err))
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(msg))
	}
}

func (h *TestHandler) DbTest() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		var req struct {
			Message string `json:"message"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			h.log.Warn("invalid dbtest request", slog.Any("error", err))
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}
		if err := h.uc.SaveMessage(r.Context(), req.Message); err != nil {
			h.log.Error("failed to save dbtest message", slog.Any("error", err))
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
	}
}
