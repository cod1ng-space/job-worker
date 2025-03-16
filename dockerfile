FROM golang:1.23.4-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /job-worker ./cmd/main.go

FROM alpine:3.18
WORKDIR /app
COPY --from=builder /job-worker .
EXPOSE 8081
CMD ["./job-worker"]