package handlers

import (
	"database/sql"
	"net/http"
	"order-service/rabbitmq"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

type OrderHandler struct {
	db  *sql.DB
	rmq *rabbitmq.RabbitMQ
}

type Order struct {
	ID          int         `json:"id"`
	UserID      int         `json:"user_id"`
	Status      string      `json:"status"`
	TotalAmount float64     `json:"total_amount"`
	CreatedAt   time.Time   `json:"created_at"`
	UpdatedAt   time.Time   `json:"updated_at"`
	Items       []OrderItem `json:"items,omitempty"`
}

type OrderItem struct {
	ID        int       `json:"id"`
	OrderID   int       `json:"order_id"`
	ProductID int       `json:"product_id"`
	Quantity  int       `json:"quantity"`
	Price     float64   `json:"price"`
	CreatedAt time.Time `json:"created_at"`
}

type CreateOrderRequest struct {
	Items []OrderItemRequest `json:"items" binding:"required,min=1"`
}

type OrderItemRequest struct {
	ProductID int `json:"product_id" binding:"required"`
	Quantity  int `json:"quantity" binding:"required,min=1"`
}

type OrderPlacedMessage struct {
	OrderID     int                `json:"order_id"`
	UserID      int                `json:"user_id"`
	Items       []OrderItemRequest `json:"items"`
	TotalAmount float64            `json:"total_amount"`
	Timestamp   time.Time          `json:"timestamp"`
}

func NewOrderHandler(db *sql.DB, rmq *rabbitmq.RabbitMQ) *OrderHandler {
	return &OrderHandler{
		db:  db,
		rmq: rmq,
	}
}

// CreateOrder creates a new order (async processing with RabbitMQ)
func (h *OrderHandler) CreateOrder(c *gin.Context) {
	userID := c.GetInt("user_id")

	var req CreateOrderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	// Start transaction
	tx, err := h.db.Begin()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to start transaction",
		})
		return
	}
	defer tx.Rollback()

	// Validate products exist and calculate total
	totalAmount := 0.0
	for _, item := range req.Items {
		var price float64
		var stock int
		err := tx.QueryRow(
			"SELECT price, stock FROM products WHERE id = $1",
			item.ProductID,
		).Scan(&price, &stock)

		if err == sql.ErrNoRows {
			c.JSON(http.StatusBadRequest, gin.H{
				"success": false,
				"error":   "Product ID " + strconv.Itoa(item.ProductID) + " not found",
			})
			return
		}

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"success": false,
				"error":   "Database error",
			})
			return
		}

		totalAmount += price * float64(item.Quantity)
	}

	// Create order with PENDING status
	var orderID int
	err = tx.QueryRow(
		`INSERT INTO orders (user_id, status, total_amount) 
		 VALUES ($1, $2, $3) 
		 RETURNING id`,
		userID, "PENDING", totalAmount,
	).Scan(&orderID)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to create order",
		})
		return
	}

	// Insert order items
	for _, item := range req.Items {
		var price float64
		tx.QueryRow("SELECT price FROM products WHERE id = $1", item.ProductID).Scan(&price)

		_, err := tx.Exec(
			`INSERT INTO order_items (order_id, product_id, quantity, price) 
			 VALUES ($1, $2, $3, $4)`,
			orderID, item.ProductID, item.Quantity, price,
		)

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"success": false,
				"error":   "Failed to create order items",
			})
			return
		}
	}

	// Commit transaction
	if err := tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to commit transaction",
		})
		return
	}

	// Publish message to RabbitMQ for async processing
	message := OrderPlacedMessage{
		OrderID:     orderID,
		UserID:      userID,
		Items:       req.Items,
		TotalAmount: totalAmount,
		Timestamp:   time.Now(),
	}

	err = h.rmq.Publish(rabbitmq.QueueOrderPlaced, message)
	if err != nil {
		// Order created but failed to publish - log error
		// In production, you might want to implement retry logic
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Order created but failed to queue for processing",
		})
		return
	}

	// Return 202 Accepted - order is being processed
	c.JSON(http.StatusAccepted, gin.H{
		"success": true,
		"message": "Order received and is being processed",
		"data": gin.H{
			"order_id": orderID,
			"status":   "PENDING",
		},
	})
}

// GetOrderByID returns order details
func (h *OrderHandler) GetOrderByID(c *gin.Context) {
	userID := c.GetInt("user_id")
	orderIDStr := c.Param("id")
	orderID, err := strconv.Atoi(orderIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Invalid order ID",
		})
		return
	}

	// Get order
	var order Order
	err = h.db.QueryRow(
		`SELECT id, user_id, status, total_amount, created_at, updated_at 
		 FROM orders WHERE id = $1 AND user_id = $2`,
		orderID, userID,
	).Scan(&order.ID, &order.UserID, &order.Status, &order.TotalAmount, &order.CreatedAt, &order.UpdatedAt)

	if err == sql.ErrNoRows {
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"error":   "Order not found",
		})
		return
	}

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Database error",
		})
		return
	}

	// Get order items
	rows, err := h.db.Query(
		`SELECT id, order_id, product_id, quantity, price, created_at 
		 FROM order_items WHERE order_id = $1`,
		orderID,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to get order items",
		})
		return
	}
	defer rows.Close()

	items := []OrderItem{}
	for rows.Next() {
		var item OrderItem
		err := rows.Scan(&item.ID, &item.OrderID, &item.ProductID, &item.Quantity, &item.Price, &item.CreatedAt)
		if err != nil {
			continue
		}
		items = append(items, item)
	}

	order.Items = items

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    order,
	})
}

// GetUserOrders returns all orders for the logged-in user
func (h *OrderHandler) GetUserOrders(c *gin.Context) {
	userID := c.GetInt("user_id")

	rows, err := h.db.Query(
		`SELECT id, user_id, status, total_amount, created_at, updated_at 
		 FROM orders WHERE user_id = $1 ORDER BY created_at DESC`,
		userID,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Database error",
		})
		return
	}
	defer rows.Close()

	orders := []Order{}
	for rows.Next() {
		var order Order
		err := rows.Scan(&order.ID, &order.UserID, &order.Status, &order.TotalAmount, &order.CreatedAt, &order.UpdatedAt)
		if err != nil {
			continue
		}
		orders = append(orders, order)
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    orders,
	})
}
