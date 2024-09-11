# Step 1: Build the Go binary in a build stage
FROM golang:1.21.6-alpine AS build

# Install git for Go module download (if needed) and any other dependencies
RUN apk add --no-cache git

# Set the current working directory inside the container
WORKDIR /app

# Copy go.mod and go.sum files and download dependencies
COPY go.mod go.sum ./
RUN go mod download

# Copy the rest of the application code
COPY . .

# Build the Go application
RUN env GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o snippetApp ./cmd/api

# Step 2: Create a lightweight image with the built binary
FROM alpine:latest

# Set working directory for the final container
WORKDIR /app

# Copy the built Go binary from the build stage
COPY --from=build /app/snippetApp .

# Expose the port the service listens on (if needed)
EXPOSE 8081

# Command to run the broker-service
CMD ["/app/snippetApp"]
