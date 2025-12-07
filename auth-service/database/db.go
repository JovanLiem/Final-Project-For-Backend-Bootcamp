package database

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"time"

	_ "github.com/lib/pq"
)

// Connect establishes connection to PostgreSQL with retry logic
func Connect() (*sql.DB, error) {
	host := os.Getenv("DB_HOST")
	port := os.Getenv("DB_PORT")
	user := os.Getenv("DB_USER")
	password := os.Getenv("DB_PASSWORD")
	dbname := os.Getenv("DB_NAME")

	connStr := fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		host, port, user, password, dbname,
	)

	var db *sql.DB
	var err error

	// Retry connection up to 10 times
	for i := 0; i < 10; i++ {
		db, err = sql.Open("postgres", connStr)
		if err == nil {
			err = db.Ping()
			if err == nil {
				break
			}
		}
		log.Printf("Failed to connect to database (attempt %d/10): %v", i+1, err)
		time.Sleep(3 * time.Second)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to connect to database after 10 attempts: %w", err)
	}

	// Set connection pool settings
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)

	log.Println("Successfully connected to database")
	return db, nil
}

// InitDB creates database schema if not exists
func InitDB(db *sql.DB) error {
	log.Println("Initializing database schema...")

	schema := `
	-- Create users table
	CREATE TABLE IF NOT EXISTS users (
		id SERIAL PRIMARY KEY,
		name VARCHAR(255) NOT NULL,
		email VARCHAR(255) UNIQUE NOT NULL,
		password VARCHAR(255) NOT NULL,
		phone VARCHAR(50),
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	);

	-- Create products table
	CREATE TABLE IF NOT EXISTS products (
		id SERIAL PRIMARY KEY,
		name VARCHAR(255) NOT NULL,
		description TEXT,
		price DECIMAL(10, 2) NOT NULL,
		stock INTEGER NOT NULL DEFAULT 0,
		category VARCHAR(100),
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		CONSTRAINT positive_price CHECK (price >= 0),
		CONSTRAINT positive_stock CHECK (stock >= 0)
	);

	-- Create orders table
	CREATE TABLE IF NOT EXISTS orders (
		id SERIAL PRIMARY KEY,
		user_id INTEGER NOT NULL REFERENCES users(id),
		status VARCHAR(50) NOT NULL DEFAULT 'PENDING',
		total_amount DECIMAL(10, 2) NOT NULL DEFAULT 0,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		CONSTRAINT valid_status CHECK (status IN ('PENDING', 'CONFIRMED', 'CANCELLED', 'FAILED'))
	);

	-- Create order_items table
	CREATE TABLE IF NOT EXISTS order_items (
		id SERIAL PRIMARY KEY,
		order_id INTEGER NOT NULL REFERENCES orders(id) ON DELETE CASCADE,
		product_id INTEGER NOT NULL REFERENCES products(id),
		quantity INTEGER NOT NULL,
		price DECIMAL(10, 2) NOT NULL,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		CONSTRAINT positive_quantity CHECK (quantity > 0)
	);
	`

	if _, err := db.Exec(schema); err != nil {
		return fmt.Errorf("failed to create schema: %w", err)
	}

	log.Println("Creating indexes...")

	indexes := []string{
		"CREATE INDEX IF NOT EXISTS idx_users_email ON users(email)",
		"CREATE INDEX IF NOT EXISTS idx_products_category ON products(category)",
		"CREATE INDEX IF NOT EXISTS idx_orders_user_id ON orders(user_id)",
		"CREATE INDEX IF NOT EXISTS idx_orders_status ON orders(status)",
		"CREATE INDEX IF NOT EXISTS idx_order_items_order_id ON order_items(order_id)",
		"CREATE INDEX IF NOT EXISTS idx_order_items_product_id ON order_items(product_id)",
	}

	for _, index := range indexes {
		if _, err := db.Exec(index); err != nil {
			log.Printf("Warning: failed to create index: %v", err)
		}
	}

	log.Println("Seeding sample data...")

	// Check if products already exist
	var count int
	err := db.QueryRow("SELECT COUNT(*) FROM products").Scan(&count)
	if err != nil {
		return err
	}

	if count == 0 {
		products := `
		INSERT INTO products (name, description, price, stock, category) VALUES
		('Laptop Dell XPS 13', 'Ultra-portable laptop with 13-inch display', 15999000, 10, 'Electronics'),
		('iPhone 15 Pro', 'Latest iPhone with A17 Pro chip', 18999000, 15, 'Electronics'),
		('Sony WH-1000XM5', 'Premium noise-cancelling headphones', 4999000, 20, 'Electronics'),
		('Samsung 55" QLED TV', '4K QLED Smart TV', 12999000, 8, 'Electronics'),
		('Mechanical Keyboard', 'RGB gaming mechanical keyboard', 1299000, 30, 'Accessories'),
		('Logitech MX Master 3', 'Wireless productivity mouse', 1499000, 25, 'Accessories'),
		('USB-C Hub', '7-in-1 USB-C multiport adapter', 499000, 50, 'Accessories'),
		('Portable SSD 1TB', 'Fast external SSD storage', 1999000, 40, 'Storage'),
		('Nintendo Switch', 'Hybrid gaming console', 4499000, 12, 'Gaming'),
		('PS5 Controller', 'DualSense wireless controller', 999000, 35, 'Gaming')
		ON CONFLICT DO NOTHING
		`
		if _, err := db.Exec(products); err != nil {
			log.Printf("Warning: failed to seed products: %v", err)
		} else {
			log.Println("Sample products inserted")
		}
	}

	log.Println("Creating triggers...")

	// Create function for updated_at trigger
	triggerFunction := `
	CREATE OR REPLACE FUNCTION update_updated_at_column()
	RETURNS TRIGGER AS $$
	BEGIN
		NEW.updated_at = CURRENT_TIMESTAMP;
		RETURN NEW;
	END;
	$$ language 'plpgsql';
	`

	if _, err := db.Exec(triggerFunction); err != nil {
		log.Printf("Warning: failed to create trigger function: %v", err)
	}

	// Create triggers
	triggers := []string{
		"DROP TRIGGER IF EXISTS update_users_updated_at ON users",
		"CREATE TRIGGER update_users_updated_at BEFORE UPDATE ON users FOR EACH ROW EXECUTE FUNCTION update_updated_at_column()",
		"DROP TRIGGER IF EXISTS update_products_updated_at ON products",
		"CREATE TRIGGER update_products_updated_at BEFORE UPDATE ON products FOR EACH ROW EXECUTE FUNCTION update_updated_at_column()",
		"DROP TRIGGER IF EXISTS update_orders_updated_at ON orders",
		"CREATE TRIGGER update_orders_updated_at BEFORE UPDATE ON orders FOR EACH ROW EXECUTE FUNCTION update_updated_at_column()",
	}

	for _, trigger := range triggers {
		if _, err := db.Exec(trigger); err != nil {
			log.Printf("Warning: failed to create trigger: %v", err)
		}
	}

	log.Println("Database initialization completed successfully!")
	return nil
}
