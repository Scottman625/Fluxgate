FROM golang:1.25-alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o queue-server ./cmd/server

FROM alpine:latest
RUN apk --no-cache add ca-certificates tzdata
WORKDIR /root/

COPY --from=builder /app/queue-server .
COPY --from=builder /app/web ./web
COPY --from=builder /app/internal/config/config.yaml ./config.yaml

EXPOSE 8080 9090
CMD ["./queue-server"]
