# Dockerfile
FROM golang:1.23 as builder
WORKDIR /app
COPY . .
RUN go mod download
RUN CGO_ENABLED=0 GOOS=linux go build -o mqtt-client

FROM alpine:latest
WORKDIR /root/
COPY --from=builder /app/mqtt-client .
CMD ["./mqtt-client"]
