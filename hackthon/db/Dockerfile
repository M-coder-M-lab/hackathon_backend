# ビルドステージ
FROM golang:1.24 AS builder
WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN go build -o main .   # ← 注意：ここで「main」バイナリが本当に生成されるか

# ランタイムステージ
FROM scratch
WORKDIR /app
COPY --from=builder /app/main .   # ← このファイルが builder に存在している必要がある

EXPOSE 8080
CMD ["/app/main"]
