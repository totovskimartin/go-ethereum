ARG COMMIT=""
ARG VERSION=""
ARG BUILDNUM=""

# First stage: Build the Geth binary
FROM golang:1.20-alpine AS builder

RUN apk add --no-cache gcc musl-dev linux-headers git

# Get Go dependencies
COPY go.mod /go-ethereum/
COPY go.sum /go-ethereum/
RUN cd /go-ethereum && go mod download

# Build Geth
ADD . /go-ethereum
RUN cd /go-ethereum && go run build/ci.go install -static ./cmd/geth

# Second stage: Node.js setup with Geth
FROM node:16-alpine

# Install necessary packages
RUN apk add --no-cache ca-certificates

# Copy the Geth binary into the runtime image
COPY --from=builder /go-ethereum/build/bin/geth /usr/local/bin/

# Set the working directory for your application
WORKDIR /app

# Copy package.json and package-lock.json and install dependencies
COPY ./hardhat/package.json ./hardhat/package-lock.json ./
RUN npm install

# Copy the remaining Hardhat project files
COPY ./hardhat/ .

# Expose application ports
EXPOSE 8545 8546 30303 30303/udp

# Start Geth or replace with the needed CMD to start your application
ENTRYPOINT ["geth"]

LABEL commit="$COMMIT" version="$VERSION" buildnum="$BUILDNUM"