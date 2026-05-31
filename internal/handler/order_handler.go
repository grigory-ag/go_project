package handler

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"time"

	"go_project/internal/domain"
	"go_project/internal/utils"

	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
)

type OrderHandler struct {
	uc  domain.OrderUsecase
	log *slog.Logger
}

var validate = validator.New()

const CtxTimeout = 5 * time.Second

func NewOrderHandler(uc domain.OrderUsecase) *OrderHandler {
	return &OrderHandler{uc: uc, log: slog.Default()}
}

func (h *OrderHandler) AddNewOrder() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		h.log.Info("Received new order request")

		userID, ok := r.Context().Value("userID").(string)
		if !ok || userID == "" {
			h.log.Error("User not authenticated, missing userID")
			utils.WriteJSONError(w, http.StatusUnauthorized, "User not authenticated", h.log)
			return
		}

		body, err := io.ReadAll(r.Body)
		if err != nil {
			h.log.Error("failed to read body", slog.Any("error", err))
			utils.WriteJSONError(w, http.StatusInternalServerError, "Failed to read request body", h.log)
			return
		}
		defer r.Body.Close()

		newOrderParams := domain.NewOrderData{}
		if err := json.Unmarshal(body, &newOrderParams); err != nil {
			utils.WriteJSONError(w, http.StatusBadRequest, "Invalid JSON", h.log)
			return
		}

		newOrderParams.UserID = userID

		var validationError validator.ValidationErrors
		validationErr := validate.Struct(newOrderParams)
		if validationErr != nil {
			errString := ""
			if errors.As(validationErr, &validationError) {
				errString = utils.FormatValidationErrors(validationError)
			} else {
				errString = validationErr.Error()
			}
			utils.WriteJSONError(w, http.StatusBadRequest, errString, h.log)
			return
		}

		orderIDs := []uuid.UUID{}
		for range newOrderParams.Amount {
			newOrderCtx, cancel := context.WithTimeout(context.Background(), CtxTimeout)
			defer cancel()

			orderID, err := h.uc.CreateOrder(newOrderCtx, newOrderParams.UserID)
			if err != nil {
				utils.WriteJSONError(w, http.StatusInternalServerError, "Failed to create order", h.log)
				return
			}

			orderIDs = append(orderIDs, orderID)
		}

		utils.WriteJSONResponse(w, http.StatusOK, orderIDs, h.log)
	}
}

func (h *OrderHandler) GetOrdersList() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		h.log.Info("Received get orders list request")

		active := r.URL.Query().Get("active")
		isActive := false
		if active != "" {
			switch active {
			case "true":
				isActive = true
			case "false":
				isActive = false
			default:
				utils.WriteJSONError(w, http.StatusBadRequest, "Forbidden order status, should be true/false", h.log)
				return
			}
		}

		userID, ok := r.Context().Value("userID").(string)
		if !ok || userID == "" {
			h.log.Error("User not authenticated, missing userID")
			utils.WriteJSONError(w, http.StatusUnauthorized, "User not authenticated", h.log)
			return
		}

		getOrdersCtx, cancel := context.WithTimeout(context.Background(), CtxTimeout)
		defer cancel()

		orders, err := h.uc.GetOrders(getOrdersCtx, userID, isActive)
		if err != nil {
			utils.WriteJSONError(w, http.StatusInternalServerError, "Failed to get orders", h.log)
			return
		}

		utils.WriteJSONResponse(w, http.StatusOK, map[string]interface{}{"count": len(orders), "items": orders}, h.log)
	}
}
