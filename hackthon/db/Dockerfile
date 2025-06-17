# ビルドステージ
FROM golang:1.24 AS builder
WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
# distroless/static は完全に静的である必要はありませんが、CGO_ENABLED=0 は一般的に良い習慣です。
RUN CGO_ENABLED=0 go build -o main .

# Final stage (distroless)
FROM gcr.io/distroless/static # <-- ここを変更
WORKDIR /app
COPY --from=builder /app/main .

EXPOSE 8080
CMD ["/app/main"]
