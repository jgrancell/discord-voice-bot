# Stage 1: Build the Go application using the official Golang image
FROM golang:1.23-alpine3.20 AS builder

# Set the working directory for the build
WORKDIR /app

# Copy go.mod and go.sum files to the working directory
COPY go.mod go.sum ./

# Download Go module dependencies
RUN go mod download

# Copy the rest of the application code to the container
COPY . .

# Build the Go application binary
RUN CGO_ENABLED=0 GOOS=linux go build -o discord-bot .

# Stage 2: Create the final minimal container image using Alpine Linux
FROM alpine:3.20

# Set environment variable for the Discord token
ENV BOT_DISCORD_TOKEN=""
ENV BOT_ENVIRONMENT="production"
ENV BOT_LOG_LEVEL="info"
ENV BOT_GUILD_IDS=""
ENV BOT_CATEGORY_ENABLED=""

# Install necessary CA certificates (if your bot makes HTTPS requests)
RUN apk --no-cache add ca-certificates

# Set the working directory in the final container
WORKDIR /app

# Copy the built Go binary from the builder stage to the final container
COPY --from=builder /app/discord-bot /app/discord-bot

# Expose the port for Prometheus metrics (adjust if necessary)
EXPOSE 2112

# Set the entrypoint to the binary, so the container will run the bot on startup
ENTRYPOINT ["/app/discord-bot"]
