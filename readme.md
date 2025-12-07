# Final Project - Backend Bootcamp

**Name:** Jovan Amarta Liem  
**Student ID:** 2602058932

## About This Project

This project was developed as the final assignment for the Backend Bootcamp program organized by **TFI (Teach For Indonesia)** and **Taman Belajar**, running from **October 11, 2025** to **November 29, 2025**.

This is a microservices-based e-commerce backend system built with Go, featuring separate services for authentication and order management.

## Live Deployment

The application is currently deployed and accessible at the following endpoints:

- **Authentication Service:** [https://jovapis.cloud/auth](https://jovapis.cloud/auth)  
  Handles user registration, login, and authentication-related operations.

- **Order Service:** [https://jovapis.cloud/order](https://jovapis.cloud/order)  
  Manages product orders and related requests.

- **Product Catalog:** [https://jovapis.cloud/order/products](https://jovapis.cloud/order/products)  
  View all available products in the database.

> **Note:** The deployment will remain active until **December 26, 2025**.

## Tech Stack

- **Language:** Go (Golang)
- **Framework:** Gin
- **Database:** PostgreSQL
- **Authentication:** JWT (JSON Web Tokens)
- **Architecture:** Microservices
- **Deployment:** VPS

## Features

### Authentication Service
- User registration
- User login
- JWT token generation and validation
- Password hashing and security

### Order Service
- Product catalog management
- Order creation and processing
- Product listing
- Order history

## Project Structure
```
├── auth-service/
│   ├── handlers/
│   ├── models/
│   ├── middleware/
│   └── main.go
├── order-service/
│   ├── handlers/
│   ├── models/
│   ├── middleware/
│   └── main.go
└── README.md
```

## Getting Started

### Prerequisites
- Go 1.21 or higher
- PostgreSQL / MySQL
- Git

### Installation

1. Clone the repository
```bash
git clone https://github.com/JovanLiem/Final-Project-For-Backend-Bootcamp.git
cd [project-folder]
```

2. Install dependencies
```bash
go mod download
```

3. Set up environment variables
```bash
# Create .env file for each service
cp .env.example .env
```

4. Run the services
```bash
# Run auth service
cd auth-service
go run main.go

# Run order service (in another terminal)
cd order-service
go run main.go
```

## API Documentation

### Authentication Endpoints

#### Register
```http
POST /auth/register
Content-Type: application/json

{
  "username": "string",
  "email": "string",
  "password": "string"
}
```

#### Login
```http
POST /auth/login
Content-Type: application/json

{
  "email": "string",
  "password": "string"
}
```

### Order Endpoints

#### Get All Products
```http
GET /orders/products
Authorization: Bearer {token}
```

#### Create Order
```http
POST /orders
Authorization: Bearer {token}
Content-Type: application/json

{
    "items": [
        {
            "product_id": "integer",
            "quantity": "integer"
        }
    ]
}
```

## Database Schema

*(Add your database schema here if needed)*

## Environment Variables
```env
# Auth Service
AUTH_PORT=8080
DB_HOST=localhost
DB_PORT=5432
DB_USER=postgres
DB_PASSWORD=password
DB_NAME=auth_db
JWT_SECRET=your_secret_key

# Order Service
ORDER_PORT=8081
DB_HOST=localhost
DB_PORT=5432
DB_USER=postgres
DB_PASSWORD=password
DB_NAME=order_db
```

## Contact

For questions or feedback, please contact:
- **Email:** *mail.atrama.07@gmail.com*
- **GitHub:** *https://github.com/JovanLiem*

---

© 2025 Jovan Amarta Liem.