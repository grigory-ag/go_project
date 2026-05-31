package utils

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strings"

	"github.com/go-playground/validator/v10"
)

func WriteJSONError(w http.ResponseWriter, status int, message string, log *slog.Logger) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	resp := map[string]string{"error": message}
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		log.Error("Failed to encode response error", slog.Any("error", err))
	}
}

func WriteJSONResponse(w http.ResponseWriter, status int, body interface{}, log *slog.Logger) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	if err := json.NewEncoder(w).Encode(body); err != nil {
		log.Error("Failed to encode response body", slog.Any("error", err))
	}
}

func FormatValidationErrors(errs validator.ValidationErrors) string {
	messages := make([]string, 0, len(errs))
	for _, err := range errs {
		messages = append(messages, fmt.Sprintf("%s failed on %s validation", err.Field(), err.Tag()))
	}

	return strings.Join(messages, "; ")
}
