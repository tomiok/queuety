FROM golang:1.24-alpine AS builder

RUN apk add --no-cache git ca-certificates tzdata
RUN adduser -D -s /bin/sh -u 1001 appuser
RUN mkdir -p /tmp/data && chown 1001:1001 /tmp/data

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download && go mod verify

COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags='-w -s -extldflags "-static"' \
    -a -installsuffix cgo \
    -o queuety ./server/main/main.go

FROM scratch

COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /usr/share/zoneinfo /usr/share/zoneinfo
COPY --from=builder /etc/passwd /etc/passwd
COPY --from=builder --chown=1001:1001 /tmp/data /data
COPY --from=builder /app/queuety /queuety

EXPOSE 9845

ENV BADGER_PATH=/data/badger
ENV PROTOCOL=tcp
ENV PORT=:9845

VOLUME ["/data"]

USER appuser

ENTRYPOINT ["/queuety"]

HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD nc -z localhost 9845 || exit 1