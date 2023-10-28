# Build stage
FROM --platform=${BUILDPLATFORM:-linux/amd64} golang:1.21.3-bullseye AS builder

ARG TARGETPLATFORM
ARG BUILDPLATFORM
ARG TARGETOS
ARG TARGETARCH
WORKDIR /app
COPY *.go go.* /app/
COPY build.sh /
RUN apt update -y && apt install -y musl-dev libc-dev gcc-aarch64-linux-gnu
RUN /build.sh

# Final stage
FROM --platform=${TARGETPLATFORM:-linux/amd64} alpine:3.18
LABEL org.opencontainers.image.source	https://github.com/marthydavid/kafka-keda-golang
RUN apk add --no-cache ca-certificates

WORKDIR /app
COPY --from=builder /app/consumer /app/consumer

CMD ["/app/consumer"]