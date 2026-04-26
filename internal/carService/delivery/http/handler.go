package http

import (
	carService "go_project/internal/carService"
	"net/http"
)

type handler struct {
	uc carService.UseCase
}

func New(uc carService.UseCase) *handler {
	return &handler{uc: uc}
}

func (h *handler) Test() http.HandlerFunc {
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
