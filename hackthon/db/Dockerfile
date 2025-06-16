# ビルドステージ
FROM golang:1.24 AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .
# CGO_ENABLED=0 is important for truly static binaries with scratch or distroless
# If your app needs CGO, you'd link it dynamically which distroless/static helps with
RUN CGO_ENABLED=0 go build -o /app/main .

# ランタイム
FROM gcr.io/distroless/static
WORKDIR /app
COPY --from=builder /app/main .

EXPOSE 8080
CMD ["/app/main"]
