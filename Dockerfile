# Stage 1: Build Geth and install Node.js and Hardhat
FROM golang:1.16 as builder

# Install Node.js and npm
RUN curl -sL https://deb.nodesource.com/setup_14.x | bash - && \
    apt-get install -y nodejs

# Install Hardhat globally
RUN npm install -g hardhat

# Set the working directory
WORKDIR /go-ethereum

# Copy go.mod and go.sum to the working directory
COPY go.mod .
COPY go.sum .

# Download Go dependencies
RUN go mod download

# Copy the rest of the project files
COPY . .

# Build Geth
RUN go run build/ci.go install -static ./cmd/geth

# Stage 2: Create a lightweight deployment container
FROM alpine:latest

# Install necessary packages
RUN apk add --no-cache ca-certificates

# Copy the Geth binary from the builder stage
COPY --from=builder /go-ethereum/build/bin/geth /usr/local/bin/

# Copy the project files (including Node.js and Hardhat)
COPY --from=builder /go-ethereum /app

# Set the working directory
WORKDIR /app

# Expose necessary ports
EXPOSE 8545 30303

# Command to run Geth
CMD ["geth", "--networkid", "1337", "--nodiscover", "--http", "--http.addr", "0.0.0.0", "--http.port", "8545"]