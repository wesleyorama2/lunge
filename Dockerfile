# Build stage
FROM golang:1.23.5-alpine AS builder

# Install git (needed for go mod download)
RUN apk add --no-cache git

# Set working directory
WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the binary
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o lunge ./cmd/lunge

# Runtime stage
FROM alpine:latest

# Install ca-certificates for HTTPS requests
RUN apk --no-cache add ca-certificates

# Create non-root user
RUN addgroup -g 1001 -S lunge && \
    adduser -u 1001 -S lunge -G lunge

WORKDIR /home/lunge

# Copy the binary from builder stage
COPY --from=builder /app/lunge .

# Copy examples directory for testing
COPY --from=builder /app/examples ./examples

# Change ownership to lunge user
RUN chown -R lunge:lunge /home/lunge

# Switch to non-root user
USER lunge

# Set the binary as entrypoint
ENTRYPOINT ["./lunge"]

# Default command shows help
CMD ["--help"]