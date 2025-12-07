package handlers

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

type ProductHandler struct {
	db    *sql.DB
	redis *redis.Client
}

type Product struct {
	ID          int       `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Price       float64   `json:"price"`
	Stock       int       `json:"stock"`
	Category    string    `json:"category"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

const ProductCacheTTL = 5 * time.Minute

func NewProductHandler(db *sql.DB, redis *redis.Client) *ProductHandler {
	return &ProductHandler{
		db:    db,
		redis: redis,
	}
}

// SearchProducts returns products with optional category filter (cached)
func (h *ProductHandler) SearchProducts(c *gin.Context) {
	category := c.Query("category")
	name := c.Query("name")

	// Create cache key based on query parameters
	cacheKey := "products"
	if category != "" {
		cacheKey += ":category:" + category
	}
	if name != "" {
		cacheKey += ":name:" + name
	}

	ctx := context.Background()

	// Try to get from cache first
	cachedData, err := h.redis.Get(ctx, cacheKey).Result()
	if err == nil {
		// Cache hit
		var products []Product
		if err := json.Unmarshal([]byte(cachedData), &products); err == nil {
			c.JSON(http.StatusOK, gin.H{
				"success": true,
				"data":    products,
				"cached":  true,
			})
			return
		}
	}

	// Cache miss - query database
	query := "SELECT id, name, description, price, stock, category, created_at, updated_at FROM products WHERE 1=1"
	args := []interface{}{}
	argCount := 1

	if category != "" {
		query += " AND category = $" + strconv.Itoa(argCount)
		args = append(args, category)
		argCount++
	}

	if name != "" {
		query += " AND name ILIKE $" + strconv.Itoa(argCount)
		args = append(args, "%"+name+"%")
		argCount++
	}

	query += " ORDER BY id"

	rows, err := h.db.Query(query, args...)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Database error",
		})
		return
	}
	defer rows.Close()

	products := []Product{}
	for rows.Next() {
		var p Product
		err := rows.Scan(&p.ID, &p.Name, &p.Description, &p.Price, &p.Stock, &p.Category, &p.CreatedAt, &p.UpdatedAt)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"success": false,
				"error":   "Failed to scan product",
			})
			return
		}
		products = append(products, p)
	}

	// Store in cache for 5 minutes
	productsJSON, _ := json.Marshal(products)
	h.redis.Set(ctx, cacheKey, productsJSON, ProductCacheTTL)

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    products,
		"cached":  false,
	})
}

// GetProductByID returns a single product by ID
func (h *ProductHandler) GetProductByID(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Invalid product ID",
		})
		return
	}

	// Try cache first
	cacheKey := "product:" + idStr
	ctx := context.Background()

	cachedData, err := h.redis.Get(ctx, cacheKey).Result()
	if err == nil {
		var product Product
		if err := json.Unmarshal([]byte(cachedData), &product); err == nil {
			c.JSON(http.StatusOK, gin.H{
				"success": true,
				"data":    product,
				"cached":  true,
			})
			return
		}
	}

	// Query database
	var product Product
	err = h.db.QueryRow(
		`SELECT id, name, description, price, stock, category, created_at, updated_at 
		 FROM products WHERE id = $1`,
		id,
	).Scan(&product.ID, &product.Name, &product.Description, &product.Price, &product.Stock, &product.Category, &product.CreatedAt, &product.UpdatedAt)

	if err == sql.ErrNoRows {
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"error":   "Product not found",
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

	// Cache the product
	productJSON, _ := json.Marshal(product)
	h.redis.Set(ctx, cacheKey, productJSON, ProductCacheTTL)

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    product,
		"cached":  false,
	})
}
