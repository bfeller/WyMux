ARG BUILD_FROM=ghcr.io/home-assistant/amd64-base:latest

# Stage 1: Build the Go binary
FROM golang:1.21-alpine AS builder
WORKDIR /app
COPY go.mod ./
# COPY go.sum ./ # Uncomment when we add external dependencies
RUN go mod download

COPY . .
RUN go build -o wymux main.go

# Stage 2: Final Image
FROM $BUILD_FROM

WORKDIR /app
# Copy built binary from builder stage
COPY --from=builder /app/wymux /app/wymux
COPY run.sh /app/run.sh
RUN chmod a+x /app/run.sh

CMD [ "/app/run.sh" ]
