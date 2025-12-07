package consumers

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/smtp"
	"time"
)

type NotificationConsumer struct {
	db *sql.DB
}

func NewNotificationConsumer(db *sql.DB) *NotificationConsumer {
	return &NotificationConsumer{db: db}
}

const (
	smtpHost = "qweqwe"
	smtpPort = "qweqwe"
	smtpUser = "vsddsvsvd" // ganti dengan emailmu
	smtpPass = "asfdgsdvs" // ganti dengan App Password
)

// ProcessConfirmed handles order_confirmed messages and simulates sending email
func (c *NotificationConsumer) ProcessConfirmed(body []byte) error {
	var msg OrderConfirmedMessage
	if err := json.Unmarshal(body, &msg); err != nil {
		return fmt.Errorf("failed to unmarshal message: %w", err)
	}

	log.Printf("Processing order confirmation notification for order #%d", msg.OrderID)

	// Get order details
	var totalAmount float64
	err := c.db.QueryRow(
		`SELECT total_amount FROM orders WHERE id = $1`,
		msg.OrderID,
	).Scan(&totalAmount)

	if err != nil {
		return fmt.Errorf("failed to get order details: %w", err)
	}

	// Simulate sending confirmation email
	email_err := c.sendConfirmationEmail(msg.UserEmail, msg.OrderID, totalAmount)
	if email_err != nil {
		return fmt.Errorf("failed to send email: %w", email_err)
	}

	log.Printf("Confirmation email sent to %s for order #%d", msg.UserEmail, msg.OrderID)

	return nil
}

// ProcessFailed handles order_failed messages and simulates sending email
func (c *NotificationConsumer) ProcessFailed(body []byte) error {
	var msg OrderFailedMessage
	if err := json.Unmarshal(body, &msg); err != nil {
		return fmt.Errorf("failed to unmarshal message: %w", err)
	}

	log.Printf("Processing order failure notification for order #%d", msg.OrderID)

	// Simulate sending failure email
	if err := c.sendFailureEmail(msg.UserEmail, msg.OrderID, msg.Reason); err != nil {
		return fmt.Errorf("failed to send failure email: %w", err)
	}

	log.Printf("Failure email sent to %s for order #%d", msg.UserEmail, msg.OrderID)

	return nil
}

func (c *NotificationConsumer) HandleOrderConfirmed(body []byte) error {
	var order OrderConfirmedMessage
	if err := json.Unmarshal(body, &order); err != nil {
		return fmt.Errorf("failed to unmarshal order: %w", err)
	}

	log.Printf("📧 Sending confirmation email to %s for order %d", order.UserEmail, order.OrderID)

	if err := c.sendConfirmationEmail(order.UserEmail, order.OrderID, 0); err != nil {
		return err
	}

	return nil
}

func (c *NotificationConsumer) HandleOrderFailed(body []byte) error {
	var msg OrderFailedMessage
	if err := json.Unmarshal(body, &msg); err != nil {
		return fmt.Errorf("failed to unmarshal failed order: %w", err)
	}

	return c.sendFailureEmail(msg.UserEmail, msg.OrderID, msg.Reason)
}

// sendConfirmationEmail simulates sending a confirmation email
//
//	func (c *NotificationConsumer) sendConfirmationEmail(email string, orderID int, totalAmount float64) {
//		// In production, this would integrate with an email service (SendGrid, AWS SES, etc.)
//		log.Println("==============================================")
//		log.Println("📧 SENDING CONFIRMATION EMAIL")
//		log.Println("==============================================")
//		log.Printf("To: %s", email)
//		log.Printf("Subject: Order #%d Confirmed! 🎉", orderID)
//		log.Println("----------------------------------------------")
//		log.Println("Dear Customer,")
//		log.Println("")
//		log.Printf("Your order #%d has been confirmed successfully!", orderID)
//		log.Printf("Total Amount: Rp %.2f", totalAmount)
//		log.Println("")
//		log.Println("We will process your order shortly and keep you updated.")
//		log.Println("")
//		log.Println("Thank you for shopping with us!")
//		log.Println("")
//		log.Printf("Order Date: %s", time.Now().Format("2006-01-02 15:04:05"))
//		log.Println("==============================================")
//		log.Println("")
//	}
func (c *NotificationConsumer) sendConfirmationEmail(email string, orderID int, totalAmount float64) error {
	// Buat subject dan body email
	subject := fmt.Sprintf("Order #%d Confirmed! 🎉", orderID)
	body := fmt.Sprintf(
		`Dear Customer,

Your order #%d has been confirmed successfully!
Total Amount: Rp %.2f

We will process your order shortly and keep you updated.

Thank you for shopping with us!

Order Date: %s
`,
		orderID,
		totalAmount,
		time.Now().Format("2006-01-02 15:04:05"),
	)

	// Buat message email lengkap
	msg := "From: " + smtpUser + "\n" +
		"To: " + email + "\n" +
		"Subject: " + subject + "\n\n" +
		body

	// Setup SMTP auth
	addr := smtpHost + ":" + smtpPort
	auth := smtp.PlainAuth("", smtpUser, smtpPass, smtpHost)

	// Kirim email
	err := smtp.SendMail(addr, auth, smtpUser, []string{email}, []byte(msg))
	if err != nil {
		return fmt.Errorf("failed to send email: %w", err)
	}

	log.Printf("✅ Email sent to %s successfully!", email)
	return nil
}

// sendFailureEmail simulates sending a failure email
func (c *NotificationConsumer) sendFailureEmail(email string, orderID int, reason string) error {
	// Subject dan body email
	subject := fmt.Sprintf("Order #%d Cancelled ❌", orderID)
	body := fmt.Sprintf(
		`Dear Customer,

Unfortunately, your order #%d could not be processed.
Reason: %s

We apologize for the inconvenience. Please try placing your order again
or contact our customer service for assistance.

Order Date: %s
`,
		orderID,
		reason,
		time.Now().Format("2006-01-02 15:04:05"),
	)

	// Buat message email lengkap
	msg := "From: " + smtpUser + "\n" +
		"To: " + email + "\n" +
		"Subject: " + subject + "\n\n" +
		body

	// Setup SMTP auth
	addr := smtpHost + ":" + smtpPort
	auth := smtp.PlainAuth("", smtpUser, smtpPass, smtpHost)

	// Kirim email
	if err := smtp.SendMail(addr, auth, smtpUser, []string{email}, []byte(msg)); err != nil {
		return fmt.Errorf("failed to send failure email: %w", err)
	}

	log.Printf("❌ Failure email sent to %s successfully!", email)
	return nil
}
