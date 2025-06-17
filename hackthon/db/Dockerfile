# ビルドステージ
FROM golang:1.24 AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .
# For alpine, you might not strictly need CGO_ENABLED=0, but it's good practice for smaller binaries.
RUN go build -o /app/main .

# ランタイム
FROM alpine/alpine:latest # Or golang:1.24-alpine if you want more Go tools
WORKDIR /app
COPY --from=builder /app/main .

EXPOSE 8080
CMD ["/app/main"]
