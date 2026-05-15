# ---- Build stage ----
FROM golang:1.21-alpine AS builder

ARG VERSION=dev
ARG COMMIT=unknown
ARG BUILD_DATE=unknown

WORKDIR /src

# Cache dependencies
COPY go.mod go.sum ./
RUN go mod download

# Build the binary
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build \
    -ldflags "-s -w -X main.version=${VERSION} -X main.commit=${COMMIT} -X main.date=${BUILD_DATE}" \
    -o /usr/local/bin/pingboard \
    ./cmd/pingboard

# ---- Runtime stage ----
FROM alpine:3.19

RUN apk add --no-cache ca-certificates tzdata

COPY --from=builder /usr/local/bin/pingboard /usr/local/bin/pingboard

# Default config mount point
VOLUME ["/etc/pingboard"]

ENTRYPOINT ["pingboard"]
CMD ["-config", "/etc/pingboard/config.yaml"]
