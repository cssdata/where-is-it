# Build stage
FROM golang:1.22.5-alpine AS builder

WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o main .

# Final stage
FROM alpine:latest

# Install ca-certificates for HTTPS requests
RUN apk --no-cache add ca-certificates

WORKDIR /root/

# Copy the binary from builder stage
COPY --from=builder /app/main .

# Copy web assets
COPY --from=builder /app/web ./web

# Create data directory
RUN mkdir -p ./data

# Expose port
EXPOSE 8080

# Set environment variable for server binding
ENV SERVER_BIND=0.0.0.0:8080

# Run the application
CMD ["./main"]