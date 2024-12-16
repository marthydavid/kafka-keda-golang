# Build stage
FROM --platform=${BUILDPLATFORM:-linux/amd64} golang:1.23.4-alpine AS builder

ARG TARGETPLATFORM
ARG BUILDPLATFORM
ARG TARGETOS
ARG TARGETARCH
WORKDIR /app
COPY *.go go.* /app/
RUN apk add --no-cache build-base
RUN CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} go build -o consumer


# Final stage
FROM --platform=${TARGETPLATFORM:-linux/amd64} alpine:3.21
LABEL org.opencontainers.image.source	https://github.com/marthydavid/kafka-keda-golang
RUN apk add --no-cache ca-certificates

WORKDIR /app
COPY --from=builder /app/consumer /app/consumer

CMD ["/app/consumer"]