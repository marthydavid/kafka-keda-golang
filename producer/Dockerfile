# Build stage
FROM golang:1.21.3 AS builder

WORKDIR /app
COPY *.go go.* /app/
RUN CGO_ENABLED=1 GOOS=linux go build -o producer

# Final stage
FROM alpine:3.14
RUN apk add --no-cache ca-certificates

WORKDIR /app
COPY --from=builder /app/producer /app/producer

CMD ["/app/producer"]