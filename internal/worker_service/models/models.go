package models

type NewStatus struct {
	OrderID   string `json:"orderID"`
	NewStatus string `json:"newStatus"`
}
