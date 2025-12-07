package main

import (
	"inventory-worker/consumers"
	"inventory-worker/database"
	"inventory-worker/rabbitmq"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/joho/godotenv"
)

func main() {
	// Load environment variables
	godotenv.Load()

	log.Println("Starting Inventory Worker Service...")

	// Initialize database
	db, err := database.Connect()
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	// Initialize database schema
	if err := database.InitDB(db); err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}

	// Initialize RabbitMQ
	rmq, err := rabbitmq.Connect(
		os.Getenv("RABBITMQ_HOST"),
		os.Getenv("RABBITMQ_PORT"),
		os.Getenv("RABBITMQ_USER"),
		os.Getenv("RABBITMQ_PASSWORD"),
	)
	if err != nil {
		log.Fatalf("Failed to connect to RabbitMQ: %v", err)
	}
	defer rmq.Close()

	// Initialize consumers
	inventoryConsumer := consumers.NewInventoryConsumer(db, rmq)
	notificationConsumer := consumers.NewNotificationConsumer(db)

	// Start consuming order_placed messages
	err = rmq.Consume(rabbitmq.QueueOrderPlaced, inventoryConsumer.ProcessOrder)
	if err != nil {
		log.Fatalf("Failed to start inventory consumer: %v", err)
	}

	// Start consuming order_confirmed messages
	err = rmq.Consume(rabbitmq.QueueOrderConfirmed, notificationConsumer.ProcessConfirmed)
	if err != nil {
		log.Fatalf("Failed to start notification consumer for confirmed orders: %v", err)
	}

	// Start consuming order_failed messages
	err = rmq.Consume(rabbitmq.QueueOrderFailed, notificationConsumer.ProcessFailed)
	if err != nil {
		log.Fatalf("Failed to start notification consumer for failed orders: %v", err)
	}

	log.Println("Inventory Worker Service started successfully")
	log.Println("Waiting for messages. Press CTRL+C to exit.")

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down Inventory Worker Service...")
}
