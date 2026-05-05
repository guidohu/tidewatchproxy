# Build stage
FROM golang:1.25-alpine AS builder

WORKDIR /app

# Copy go.mod and go.sum and download dependencies
COPY go.mod go.sum ./
RUN go mod download

# Copy the source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -o proxy ./cmd/proxy/main.go

# Run stage
FROM alpine:latest

# Install ca-certificates for HTTPS requests
RUN apk --no-cache add ca-certificates

WORKDIR /root/

# Copy the binary from the builder stage
COPY --from=builder /app/proxy .

# Setup directory structure
RUN mkdir -p /app/database

# Expose the port the app runs on
EXPOSE 8080

# Command to run the application
ENTRYPOINT ["./proxy"]
CMD ["--use-cache=true", "--db-path=/app/database/metrics.db"]
