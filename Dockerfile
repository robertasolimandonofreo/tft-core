FROM golang:1.24.0-alpine AS builder
WORKDIR /app
RUN apk add --no-cache upx
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" ./cmd/main.go; upx main

FROM alpine:3.20
WORKDIR /root/
COPY --from=builder /app/main .
EXPOSE 8000
CMD ["./main"]