# Build stage
FROM golang:alpine AS builder

WORKDIR /app

RUN apk add --no-cache git ca-certificates

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o /bin/api ./cmd/api
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o /bin/worker ./cmd/worker

# API runtime stage
FROM alpine:latest AS api

WORKDIR /app

RUN apk add --no-cache ca-certificates tzdata

COPY --from=builder /bin/api /app/api
COPY migrations /app/migrations

EXPOSE 8080

CMD ["/app/api"]

# Worker runtime stage
FROM alpine:latest AS worker

WORKDIR /app

RUN apk add --no-cache ca-certificates tzdata

COPY --from=builder /bin/worker /app/worker
COPY migrations /app/migrations

CMD ["/app/worker"]

