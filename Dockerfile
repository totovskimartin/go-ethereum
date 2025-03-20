# Support setting various labels on the final image
ARG COMMIT=""
ARG VERSION=""
ARG BUILDNUM=""

# Build Geth in a stock Go builder container
FROM golang:1.24-alpine AS builder

RUN apk add --no-cache gcc musl-dev linux-headers git

# Get dependencies - will also be cached if we won't change go.mod/go.sum
COPY go.mod /go-ethereum/
COPY go.sum /go-ethereum/
RUN cd /go-ethereum && go mod download

ADD . /go-ethereum
RUN cd /go-ethereum && go run build/ci.go install -static ./cmd/geth

# Pull Geth into a second stage deploy alpine container
FROM node:16-alpine

# Update to a Node.js base image that includes npm
RUN apk add --no-cache ca-certificates

# Copy the Geth binary from the builder
COPY --from=builder /go-ethereum/build/bin/geth /usr/local/bin/

# Set the working directory for your hardhat project
WORKDIR /app

# Copy package.json and other necessary files for npm install
COPY ./hardhat/package.json ./hardhat/package-lock.json ./

# Install npm dependencies
RUN npm install

# Copy the rest of your Hardhat files
COPY ./hardhat/ ./hardhat

EXPOSE 8545 8546 30303 30303/udp

# Initiate Geth when container starts, adjust this to your needs e.g., for Hardhat
ENTRYPOINT ["geth"]

LABEL commit="$COMMIT" version="$VERSION" buildnum="$BUILDNUM"
