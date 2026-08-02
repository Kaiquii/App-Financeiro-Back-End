FROM golang:1.26-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -o main ./cmd/api
RUN CGO_ENABLED=0 GOOS=linux go build -o migrate ./cmd/migrate

FROM alpine:latest

RUN apk add --no-cache tzdata

WORKDIR /root/

COPY --from=builder /app/main .
COPY --from=builder /app/migrate .

EXPOSE 8080

CMD ["./main"]
