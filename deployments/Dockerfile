# Build stage
FROM golang:1.24-alpine AS builder

WORKDIR /app

RUN apk add --no-cache git ca-certificates

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o /app/bin/ftfm ./cmd/ftfm

# Final minimal stage
FROM alpine:3.20

RUN apk --no-cache add ca-certificates tzdata

WORKDIR /app
COPY --from=builder /app/bin/ftfm /app/ftfm

EXPOSE 8080

ENTRYPOINT ["/app/ftfm"]
CMD ["server", "--port", "8080"]
