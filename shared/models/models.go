package models

import "time"

// User represents a user in the system
type User struct {
	ID        int       `json:"id" db:"id"`
	Name      string    `json:"name" db:"name"`
	Email     string    `json:"email" db:"email"`
	Password  string    `json:"-" db:"password"` // Never return password in JSON
	Phone     string    `json:"phone" db:"phone"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
}

// Product represents a product in the catalog
type Product struct {
	ID          int       `json:"id" db:"id"`
	Name        string    `json:"name" db:"name"`
	Description string    `json:"description" db:"description"`
	Price       float64   `json:"price" db:"price"`
	Stock       int       `json:"stock" db:"stock"`
	Category    string    `json:"category" db:"category"`
	CreatedAt   time.Time `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time `json:"updated_at" db:"updated_at"`
}

// Order represents an order
type Order struct {
	ID          int         `json:"id" db:"id"`
	UserID      int         `json:"user_id" db:"user_id"`
	Status      string      `json:"status" db:"status"` // PENDING, CONFIRMED, CANCELLED, FAILED
	TotalAmount float64     `json:"total_amount" db:"total_amount"`
	CreatedAt   time.Time   `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time   `json:"updated_at" db:"updated_at"`
	Items       []OrderItem `json:"items,omitempty" db:"-"`
}

// OrderItem represents an item in an order
type OrderItem struct {
	ID        int       `json:"id" db:"id"`
	OrderID   int       `json:"order_id" db:"order_id"`
	ProductID int       `json:"product_id" db:"product_id"`
	Quantity  int       `json:"quantity" db:"quantity"`
	Price     float64   `json:"price" db:"price"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
}

// Request/Response DTOs

// RegisterRequest for user registration
type RegisterRequest struct {
	Name     string `json:"name" binding:"required"`
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=6"`
	Phone    string `json:"phone" binding:"required"`
}

// LoginRequest for user authentication
type LoginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

// LoginResponse returns JWT token
type LoginResponse struct {
	Token string `json:"token"`
	User  User   `json:"user"`
}

// UpdateProfileRequest for updating user profile
type UpdateProfileRequest struct {
	Name  string `json:"name,omitempty"`
	Phone string `json:"phone,omitempty"`
}

// CreateOrderRequest for placing an order
type CreateOrderRequest struct {
	Items []OrderItemRequest `json:"items" binding:"required,min=1"`
}

// OrderItemRequest represents an item in order request
type OrderItemRequest struct {
	ProductID int `json:"product_id" binding:"required"`
	Quantity  int `json:"quantity" binding:"required,min=1"`
}

// CreateOrderResponse returns order status
type CreateOrderResponse struct {
	OrderID int    `json:"order_id"`
	Status  string `json:"status"`
	Message string `json:"message"`
}

// RabbitMQ Message Structures

// OrderPlacedMessage sent when order is placed
type OrderPlacedMessage struct {
	OrderID     int                `json:"order_id"`
	UserID      int                `json:"user_id"`
	Items       []OrderItemRequest `json:"items"`
	TotalAmount float64            `json:"total_amount"`
	Timestamp   time.Time          `json:"timestamp"`
}

// OrderConfirmedMessage sent when order is confirmed
type OrderConfirmedMessage struct {
	OrderID   int       `json:"order_id"`
	UserID    int       `json:"user_id"`
	UserEmail string    `json:"user_email"`
	Timestamp time.Time `json:"timestamp"`
}

// OrderFailedMessage sent when order fails
type OrderFailedMessage struct {
	OrderID   int       `json:"order_id"`
	UserID    int       `json:"user_id"`
	UserEmail string    `json:"user_email"`
	Reason    string    `json:"reason"`
	Timestamp time.Time `json:"timestamp"`
}

// Response wrapper
type Response struct {
	Success bool        `json:"success"`
	Message string      `json:"message,omitempty"`
	Data    interface{} `json:"data,omitempty"`
	Error   string      `json:"error,omitempty"`
}

// ErrorResponse for error handling
type ErrorResponse struct {
	Success bool   `json:"success"`
	Error   string `json:"error"`
}
