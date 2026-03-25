FROM golang:1.21-alpine AS builder

# Install build dependencies
RUN apk add --no-cache gcc musl-dev sqlite-dev

WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source
COPY main.go ./

# Build
ENV CGO_CFLAGS="-D_LARGEFILE64_SOURCE"
RUN CGO_ENABLED=1 GOOS=linux go build -ldflags="-s -w" -o budget-tracker main.go

# Final stage
FROM alpine:latest

RUN apk add --no-cache sqlite-libs

WORKDIR /app

# Copy binary and web files
COPY --from=builder /app/budget-tracker .
COPY index.html ./

# Create volume for database
VOLUME ["/app/data"]

# Expose port
EXPOSE 8080

# Run
CMD ["./budget-tracker"]
