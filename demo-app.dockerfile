# syntax=docker/dockerfile:1

# Stage 1: build binary
FROM golang:1.25.1-alpine AS builder

WORKDIR /app

# Cài các gói cần thiết
RUN apk add --no-cache gcc g++ make

# Copy go mod để cache dependency
COPY go.mod go.sum ./
RUN go mod download

# Copy toàn bộ source code
COPY . .

# Build binary từ main.go
RUN CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -o server main.go 

# Stage 2: runtime
FROM alpine:3.18

WORKDIR /app

COPY --from=builder /app/server .

RUN chmod +x /app/server

EXPOSE 9090 8080

