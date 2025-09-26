# Build stage
FROM golang:1.23-alpine AS builder
RUN apk add --no-cache gcc musl-dev
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
ENV CGO_ENABLED=1
RUN go build -o main ./cmd

# Run stage
FROM alpine:latest
WORKDIR /app
COPY --from=builder /app/main .
COPY cmd/cert.pem .
COPY cmd/key.pem .

EXPOSE 8080
CMD ["./main"]
