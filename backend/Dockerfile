FROM golang:1.24-alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /api ./cmd/api

FROM alpine:3.20
RUN apk --no-cache add ca-certificates && \
    adduser -D -u 1001 appuser
WORKDIR /app
COPY --from=builder /api .

USER appuser
EXPOSE 8080
CMD ["./api"]
