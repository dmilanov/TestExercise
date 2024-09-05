# Build Geth in a stock Go builder container
FROM golang:1.23-alpine as builder

RUN apk add --no-cache gcc musl-dev linux-headers git nodejs npm

# Get dependencies - will also be cached if we won't change go.mod/go.sum
COPY go.mod /go-ethereum/
COPY go.sum /go-ethereum/
RUN cd /go-ethereum && go mod download

ADD . /go-ethereum
RUN cd /go-ethereum && go run build/ci.go install -static ./cmd/geth

# Create the /app directory
RUN mkdir -p /app

# Copy only package.json and package-lock.json (if present) to the /app directory
COPY package*.json /app/

# Debugging step: List the contents of the /app directory
RUN ls -la /app


# Install npm dependencies for the Hardhat project
RUN cd /app 
RUN npm install

# Copy the rest of the project files to the /app directory
COPY . /app

# Pull Geth into a second stage deploy alpine container
FROM alpine:latest

RUN apk add --no-cache ca-certificates
COPY --from=builder /app /app
COPY --from=builder /go-ethereum/build/bin/geth /usr/local/bin/

EXPOSE 8545 8546 30303 30303/udp
ENTRYPOINT ["geth"]
