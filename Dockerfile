FROM golang:1.26-alpine3.23 AS builder

RUN apk add --no-cache git

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -o main -trimpath -ldflags="-s -w" ./cmd/server/

FROM alpine:latest

RUN apk --no-cache add ca-certificates tzdata

WORKDIR /root/

COPY --from=builder /app/main .
COPY --from=builder /app/internal/database/migrations ./migrations
COPY GeoLite2-City.mmdb ./GeoLite2-City.mmdb

CMD ["./main"]
