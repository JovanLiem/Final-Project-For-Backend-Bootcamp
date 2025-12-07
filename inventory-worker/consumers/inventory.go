package consumers

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"inventory-worker/rabbitmq"
	"log"
	"time"
)

type InventoryConsumer struct {
	db  *sql.DB
	rmq *rabbitmq.RabbitMQ
}

type OrderPlacedMessage struct {
	OrderID     int                `json:"order_id"`
	UserID      int                `json:"user_id"`
	Items       []OrderItemRequest `json:"items"`
	TotalAmount float64            `json:"total_amount"`
	Timestamp   time.Time          `json:"timestamp"`
}

type OrderItemRequest struct {
	ProductID int `json:"product_id"`
	Quantity  int `json:"quantity"`
}

type OrderConfirmedMessage struct {
	OrderID   int       `json:"order_id"`
	UserID    int       `json:"user_id"`
	UserEmail string    `json:"user_email"`
	Timestamp time.Time `json:"timestamp"`
}

type OrderFailedMessage struct {
	OrderID   int       `json:"order_id"`
	UserID    int       `json:"user_id"`
	UserEmail string    `json:"user_email"`
	Reason    string    `json:"reason"`
	Timestamp time.Time `json:"timestamp"`
}

func NewInventoryConsumer(db *sql.DB, rmq *rabbitmq.RabbitMQ) *InventoryConsumer {
	return &InventoryConsumer{
		db:  db,
		rmq: rmq,
	}
}

// ProcessOrder processes order_placed messages with atomic inventory updates
func (c *InventoryConsumer) ProcessOrder(body []byte) error {
	var msg OrderPlacedMessage
	if err := json.Unmarshal(body, &msg); err != nil {
		return fmt.Errorf("failed to unmarshal message: %w", err)
	}

	log.Printf("Processing order #%d for user #%d", msg.OrderID, msg.UserID)

	// Start database transaction for atomic operations
	tx, err := c.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to start transaction: %w", err)
	}
	defer func() {
		if r := tx.Rollback(); r != nil && r != sql.ErrTxDone {
			log.Printf("tx rollback failed: %v", r)
		}
	}()

	// Check and update inventory for each item
	for _, item := range msg.Items {
		var currentStock int
		var productName string

		// Lock the row for update (prevents race conditions)
		err := tx.QueryRow(
			`SELECT name, stock FROM products WHERE id = $1 FOR UPDATE`,
			item.ProductID,
		).Scan(&productName, &currentStock)

		if err == sql.ErrNoRows {
			log.Printf("Product #%d not found for order #%d", item.ProductID, msg.OrderID)
			c.failOrder(tx, msg.OrderID, msg.UserID, fmt.Sprintf("Product #%d not found", item.ProductID))
			return nil
		}

		if err != nil {
			return fmt.Errorf("failed to check stock: %w", err)
		}

		// Check if sufficient stock is available
		if currentStock < item.Quantity {
			log.Printf("Insufficient stock for product #%d (%s). Available: %d, Requested: %d",
				item.ProductID, productName, currentStock, item.Quantity)
			c.failOrder(tx, msg.OrderID, msg.UserID,
				fmt.Sprintf("Insufficient stock for %s. Available: %d, Requested: %d",
					productName, currentStock, item.Quantity))
			return nil
		}

		// Decrement stock atomically
		result, err := tx.Exec(
			`UPDATE products SET stock = stock - $1 WHERE id = $2`,
			item.Quantity, item.ProductID,
		)

		if err != nil {
			return fmt.Errorf("failed to update stock: %w", err)
		}

		rowsAffected, _ := result.RowsAffected()
		if rowsAffected == 0 {
			return fmt.Errorf("failed to update stock for product #%d", item.ProductID)
		}

		log.Printf("Decremented stock for product #%d (%s) by %d units. New stock: %d",
			item.ProductID, productName, item.Quantity, currentStock-item.Quantity)
	}

	// All items have sufficient stock - update order status to CONFIRMED
	_, err = tx.Exec(
		`UPDATE orders SET status = $1, updated_at = CURRENT_TIMESTAMP WHERE id = $2`,
		"CONFIRMED", msg.OrderID,
	)

	if err != nil {
		return fmt.Errorf("failed to update order status: %w", err)
	}

	// Commit the transaction
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	log.Printf("Order #%d confirmed successfully", msg.OrderID)

	// Get user email for notification
	var userEmail string
	email_err := c.db.QueryRow("SELECT email FROM users WHERE id = $1", msg.UserID).Scan(&userEmail)
	if email_err != nil {
		return fmt.Errorf("failed to send email: %w", email_err)
	}

	// Publish order_confirmed message
	confirmedMsg := OrderConfirmedMessage{
		OrderID:   msg.OrderID,
		UserID:    msg.UserID,
		UserEmail: userEmail,
		Timestamp: time.Now(),
	}

	err = c.rmq.Publish(rabbitmq.QueueOrderConfirmed, confirmedMsg)
	if err != nil {
		log.Printf("Failed to publish order_confirmed message: %v", err)
		// Don't return error - order is already confirmed
	}

	return nil
}

// failOrder marks order as CANCELLED and publishes failure message
func (c *InventoryConsumer) failOrder(tx *sql.Tx, orderID, userID int, reason string) {
	log.Printf("Failing order #%d: %s", orderID, reason)

	// Update order status to CANCELLED
	_, err := tx.Exec(
		`UPDATE orders SET status = $1, updated_at = CURRENT_TIMESTAMP WHERE id = $2`,
		"CANCELLED", orderID,
	)

	if err != nil {
		log.Printf("Failed to update order status to CANCELLED: %v", err)
		return
	}

	// Commit the transaction
	if err := tx.Commit(); err != nil {
		log.Printf("Failed to commit transaction: %v", err)
		return
	}

	// Get user email
	var userEmail string
	if err := c.db.QueryRow("SELECT email FROM users WHERE id = $1", userID).Scan(&userEmail); err != nil {
		log.Printf("failed to get email for user %d: %v", userID, err)
		return // cukup keluar dari fungsi
	}

	// Publish order_failed message
	failedMsg := OrderFailedMessage{
		OrderID:   orderID,
		UserID:    userID,
		UserEmail: userEmail,
		Reason:    reason,
		Timestamp: time.Now(),
	}

	err = c.rmq.Publish(rabbitmq.QueueOrderFailed, failedMsg)
	if err != nil {
		log.Printf("Failed to publish order_failed message: %v", err)
	}
}
