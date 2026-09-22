FROM golang:1.22-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build -o server ./cmd/server

FROM alpine:3.19
WORKDIR /app
COPY --from=builder /app/server .
COPY internal/config/config.yaml ./internal/config/config.yaml
COPY public public
EXPOSE 8080
CMD ["./server"]
