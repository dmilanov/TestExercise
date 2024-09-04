# Support setting various labels on the final image
ARG COMMIT=""
ARG VERSION=""
ARG BUILDNUM=""

# Build Geth in a stock Go builder container
FROM golang:1.23-alpine as builder

RUN apk add --no-cache gcc musl-dev linux-headers git
RUN mkdir -p /app

# Install Node.js and npm
RUN curl -sL https://deb.nodesource.com/setup_14.x | bash - && \
    apt-get install -y nodejs

# Install Hardhat globally
RUN npm install -g hardhat

# Copy the Geth binary from the builder stage
COPY --from=builder /go-ethereum/build/bin/geth /usr/local/bin/

# Copy the project files (including Node.js and Hardhat)
COPY --from=builder /go-ethereum /app

# Get dependencies - will also be cached if we won't change go.mod/go.sum
COPY go.mod /go-ethereum/
COPY go.sum /go-ethereum/
RUN cd /go-ethereum && go mod download

ADD . /go-ethereum
RUN cd /go-ethereum && go run build/ci.go install -static ./cmd/geth

# Pull Geth into a second stage deploy alpine container
FROM alpine:latest

RUN apk add --no-cache ca-certificates
COPY --from=builder /go-ethereum/build/bin/geth /usr/local/bin/

EXPOSE 8545 8546 30303 30303/udp
ENTRYPOINT ["geth"]

# Add some metadata labels to help programmatic image consumption
ARG COMMIT=""
ARG VERSION=""
ARG BUILDNUM=""

LABEL commit="$COMMIT" version="$VERSION" buildnum="$BUILDNUM"
CMD ["geth", "--networkid", "1337", "--nodiscover", "--http", "--http.addr", "0.0.0.0", "--http.port", "8545"]