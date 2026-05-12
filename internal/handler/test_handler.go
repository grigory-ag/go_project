package handler

import (
	"encoding/json"
	"net/http"

	"go_project/internal/domain"
)

type TestHandler struct {
	uc domain.TestUsecase
}

func NewTestHandler(uc domain.TestUsecase) *TestHandler {
	return &TestHandler{uc: uc}
}

func (h *TestHandler) Test() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		msg, err := h.uc.GetTestMessage(r.Context())
		if err != nil {
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
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}
		if err := h.uc.SaveMessage(r.Context(), req.Message); err != nil {
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
	}
}
