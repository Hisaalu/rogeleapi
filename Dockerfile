FROM golang:1.27-alpine AS builder
WORKDIR /app
RUN apk add --no-cache git
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o /app/bin/rogele ./cmd/api

FROM alpine:3.19
WORKDIR /app
RUN apk add --no-cache ca-certificates tzdata curl
COPY --from=builder /app/bin/rogele /app/rogele
COPY --from=builder /app/migrations /app/migrations
COPY --from=builder /app/configs /app/configs

RUN curl -L https://github.com/golang-migrate/migrate/releases/download/v4.17.0/migrate.linux-amd64.tar.gz | tar xvz \
    && mv migrate /usr/local/bin/migrate

EXPOSE 8080

# CMD ["sh", "-c", "migrate -path ./migrations -database \"postgres://${ROGELEDB_DB_USER}:${ROGELEDB_DB_PASSWORD}@${ROGELEDB_DB_HOST}:${ROGELEDB_DB_PORT}/${ROGELEDB_DB_NAME}?sslmode=require\" up && ./rogele"]
CMD ["./rogele"]