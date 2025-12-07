# Build stage
FROM golang:1.25.3 AS builder

WORKDIR /final-project-backend-bootcamp
# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o order-service ./main.go

# Runtime stage
FROM alpine:latest

RUN apk --no-cache add ca-certificates

WORKDIR /root/

# Copy binary from builder
COPY --from=builder /final-project-backend-bootcamp/order-service .

# Run the application
CMD ["./order-service"]