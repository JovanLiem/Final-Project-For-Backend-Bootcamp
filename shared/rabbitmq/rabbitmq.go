package rabbitmq

import (
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/streadway/amqp"
)

const (
	QueueOrderPlaced    = "order_placed"
	QueueOrderConfirmed = "order_confirmed"
	QueueOrderFailed    = "order_failed"
)

type RabbitMQ struct {
	conn    *amqp.Connection
	channel *amqp.Channel
}

// Connect establishes connection to RabbitMQ with retry logic
func Connect(host, port, user, password string) (*RabbitMQ, error) {
	url := fmt.Sprintf("amqp://%s:%s@%s:%s/", user, password, host, port)

	var conn *amqp.Connection
	var err error

	// Retry connection up to 10 times
	for i := 0; i < 10; i++ {
		conn, err = amqp.Dial(url)
		if err == nil {
			break
		}
		log.Printf("Failed to connect to RabbitMQ (attempt %d/10): %v", i+1, err)
		time.Sleep(3 * time.Second)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to connect to RabbitMQ after 10 attempts: %w", err)
	}

	channel, err := conn.Channel()
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to open channel: %w", err)
	}

	log.Println("Successfully connected to RabbitMQ")

	rmq := &RabbitMQ{
		conn:    conn,
		channel: channel,
	}

	// Declare all queues
	if err := rmq.declareQueues(); err != nil {
		rmq.Close()
		return nil, err
	}

	return rmq, nil
}

// declareQueues declares all required queues
func (r *RabbitMQ) declareQueues() error {
	queues := []string{QueueOrderPlaced, QueueOrderConfirmed, QueueOrderFailed}

	for _, queue := range queues {
		_, err := r.channel.QueueDeclare(
			queue, // name
			true,  // durable
			false, // delete when unused
			false, // exclusive
			false, // no-wait
			nil,   // arguments
		)
		if err != nil {
			return fmt.Errorf("failed to declare queue %s: %w", queue, err)
		}
		log.Printf("Queue declared: %s", queue)
	}

	return nil
}

// Publish publishes a message to a queue
func (r *RabbitMQ) Publish(queueName string, message interface{}) error {
	body, err := json.Marshal(message)
	if err != nil {
		return fmt.Errorf("failed to marshal message: %w", err)
	}

	err = r.channel.Publish(
		"",        // exchange
		queueName, // routing key (queue name)
		false,     // mandatory
		false,     // immediate
		amqp.Publishing{
			DeliveryMode: amqp.Persistent,
			ContentType:  "application/json",
			Body:         body,
		},
	)

	if err != nil {
		return fmt.Errorf("failed to publish message: %w", err)
	}

	log.Printf("Message published to queue %s", queueName)
	return nil
}

// Consume starts consuming messages from a queue
func (r *RabbitMQ) Consume(queueName string, handler func([]byte) error) error {
	// Set QoS to process one message at a time
	err := r.channel.Qos(
		1,     // prefetch count
		0,     // prefetch size
		false, // global
	)
	if err != nil {
		return fmt.Errorf("failed to set QoS: %w", err)
	}

	msgs, err := r.channel.Consume(
		queueName, // queue
		"",        // consumer
		false,     // auto-ack (manual ack for reliability)
		false,     // exclusive
		false,     // no-local
		false,     // no-wait
		nil,       // args
	)
	if err != nil {
		return fmt.Errorf("failed to register consumer: %w", err)
	}

	log.Printf("Started consuming from queue: %s", queueName)

	go func() {
		for msg := range msgs {
			log.Printf("Received message from %s", queueName)

			err := handler(msg.Body)
			if err != nil {
				log.Printf("Error handling message: %v", err)
				// Reject and requeue the message
				msg.Nack(false, true)
			} else {
				// Acknowledge successful processing
				msg.Ack(false)
				log.Printf("Message processed successfully from %s", queueName)
			}
		}
	}()

	return nil
}

// Close closes the RabbitMQ connection
func (r *RabbitMQ) Close() {
	if r.channel != nil {
		r.channel.Close()
	}
	if r.conn != nil {
		r.conn.Close()
	}
	log.Println("RabbitMQ connection closed")
}

// IsConnected checks if the connection is alive
func (r *RabbitMQ) IsConnected() bool {
	return r.conn != nil && !r.conn.IsClosed()
}
