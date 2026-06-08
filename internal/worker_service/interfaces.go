package worker_service

type Repository interface {
	PublishOrderStatus(orderID, newStatus string) error
	StartNewOrdersConsumer(queueName string)
}
