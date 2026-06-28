FROM golang:1.25-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /bin/service .

FROM alpine:3.21
RUN apk add --no-cache ca-certificates tzdata
WORKDIR /app
COPY --from=builder /bin/service .
RUN mkdir -p uploads
VOLUME ["/app/uploads"]
EXPOSE 8084
# Required env vars at runtime:
#   JWT_PUBLIC_KEY     — RSA public key PEM (newlines escaped as \n)
#   DATABASE_URL       — Postgres DSN
#   UPLOAD_DIR         — optional, default ./uploads (mount a volume here)
#   MAX_FILE_SIZE_MB   — optional, default 10
#   PORT               — optional, default 8084
CMD ["./service"]
