package cache

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/redis/go-redis/v9"
)

// Connect establishes connection to Redis with retry logic
func Connect() (*redis.Client, error) {
	host := os.Getenv("REDIS_HOST")
	port := os.Getenv("REDIS_PORT")

	client := redis.NewClient(&redis.Options{
		Addr:         fmt.Sprintf("%s:%s", host, port),
		Username:     "default",
		Password:     "1234", // no password set
		DB:           0,      // use default DB
		DialTimeout:  5 * time.Second,
		ReadTimeout:  3 * time.Second,
		WriteTimeout: 3 * time.Second,
		PoolSize:     10,
		MinIdleConns: 5,
	})

	ctx := context.Background()
	var err error

	// Retry connection up to 10 times
	for i := 0; i < 10; i++ {
		_, err = client.Ping(ctx).Result()
		if err == nil {
			break
		}
		log.Printf("Failed to connect to Redis (attempt %d/10): %v", i+1, err)
		time.Sleep(3 * time.Second)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to connect to Redis after 10 attempts: %w", err)
	}

	log.Println("Successfully connected to Redis")
	return client, nil
}
