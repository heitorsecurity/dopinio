# ==========================================
# Dopinio - Docker-native Developer Platform
# ==========================================

# Stage 1: Build the binary
FROM golang:1.23-alpine AS builder

WORKDIR /src

# Install build dependencies
RUN apk add --no-cache git make ca-certificates tzdata

# Cache dependencies
COPY go.mod go.sum ./
RUN go mod download

# Copy source tree
COPY . .

# Build dopinio binary
ARG VERSION=dev
ARG COMMIT=none
ARG DATE=unknown

RUN CGO_ENABLED=0 GOOS=linux go build \
    -ldflags="-s -w -X 'github.com/epinio/epinio/internal/version.Version=${VERSION}'" \
    -o /bin/dopinio main.go

# Stage 2: Runtime image
FROM alpine:3.20

LABEL maintainer="heitorsecurity"
LABEL description="Dopinio - Docker-native Developer Platform (PaaS)"

# Install runtime utilities and Docker CLI to interact with the daemon
RUN apk add --no-cache \
    ca-certificates \
    curl \
    bash \
    git \
    docker-cli \
    tzdata

# Create non-root user and required folders
RUN addgroup -g 1000 dopinio && \
    adduser -u 1000 -G dopinio -h /home/dopinio -D dopinio && \
    mkdir -p /var/lib/dopinio /etc/dopinio && \
    chown -R dopinio:dopinio /var/lib/dopinio /etc/dopinio /home/dopinio

# Copy binary from builder
COPY --from=builder /bin/dopinio /usr/local/bin/dopinio

# Default environment variables
ENV DOPINIO_RUNTIME=docker \
    PORT=8443 \
    DOPINIO_DOMAIN=dopinio.localhost \
    DOCKER_HOST=unix:///var/run/docker.sock

EXPOSE 8443

WORKDIR /home/dopinio

ENTRYPOINT ["/usr/local/bin/dopinio"]
CMD ["server"]

