# first stage
FROM golang:1.21-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build -o main ./cmd/main.go

# second stage
FROM alpine:latest
WORKDIR /app
COPY --from=builder /app/main .

# static
COPY internal/static/ internal/static/
COPY internal/templates/ internal/templates/
COPY ./config.yaml ./google_client_secret.json ./

CMD ["./main"]