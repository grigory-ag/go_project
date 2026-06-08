package handler

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"go_project/internal/domain"
	appjwt "go_project/internal/jwt"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

type UserHandler struct {
	uc  domain.UserUsecase
	log *slog.Logger
}

func NewUserHandler(uc domain.UserUsecase, log *slog.Logger) *UserHandler {
	return &UserHandler{uc: uc, log: log}
}

func (h *UserHandler) RegisterUser() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		h.log.Info("received user registration request")

		var req struct {
			Login    string `json:"login"`
			Password string `json:"password"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			h.log.Warn("invalid registration request body", slog.Any("error", err))
			http.Error(w, "invalid request body", http.StatusBadRequest)
			return
		}
		if req.Login == "" || req.Password == "" {
			h.log.Warn("registration request has empty required fields")
			http.Error(w, "login and password are required", http.StatusBadRequest)
			return
		}

		user, err := h.uc.RegisterUser(r.Context(), req.Login, req.Password)
		if err != nil {
			h.log.Warn("user registration failed", slog.String("login", req.Login), slog.Any("error", err))
			http.Error(w, err.Error(), http.StatusConflict)
			return
		}

		h.log.Info("user registered successfully", slog.String("user_id", user.ID), slog.String("login", user.Login))
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(user)
	}
}

func (h *UserHandler) LoginUser() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		h.log.Info("received login request")

		var req struct {
			Login    string `json:"login"`
			Password string `json:"password"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			h.log.Warn("invalid login request body", slog.Any("error", err))
			http.Error(w, "invalid request body", http.StatusBadRequest)
			return
		}

		user, err := h.uc.GetUserByLogin(r.Context(), req.Login)
		if err != nil || user == nil {
			h.log.Warn("login failed", slog.String("login", req.Login))
			http.Error(w, "invalid login or password", http.StatusUnauthorized)
			return
		}

		if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Password)); err != nil {
			h.log.Warn("invalid password attempt", slog.String("login", req.Login))
			http.Error(w, "invalid login or password", http.StatusUnauthorized)
			return
		}

		token, err := appjwt.GenerateToken(user.ID, user.Login)
		if err != nil {
			h.log.Error("failed to generate JWT", slog.String("user_id", user.ID), slog.Any("error", err))
			http.Error(w, "failed to generate token", http.StatusInternalServerError)
			return
		}

		sessionID := uuid.New().String()
		if err := h.uc.CreateSession(r.Context(), sessionID, user.ID); err != nil {
			h.log.Error("failed to create session", slog.String("user_id", user.ID), slog.Any("error", err))
			http.Error(w, "failed to create session", http.StatusInternalServerError)
			return
		}

		h.log.Info("user logged in successfully", slog.String("user_id", user.ID), slog.String("login", user.Login))
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{"token": token})
	}
}

func (h *UserHandler) Register(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	h.RegisterUser()(w, r)
}
