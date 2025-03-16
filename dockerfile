FROM golang:1.23.4-alpine AS builder
RUN apk add --no-cache git
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /job-worker ./cmd/main.go

FROM alpine:3.18
COPY --from=builder /job-worker /job-worker
EXPOSE 8080
ENTRYPOINT ["./job-worker"]
CMD ["--workers_count=5", "--port=8080", "--retries=3", "--timeout=3"]