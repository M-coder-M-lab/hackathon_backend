# ビルドステージ
FROM golang:1.24 AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .
# Ensure CGO_ENABLED=0 for a fully static binary suitable for distroless/static
RUN CGO_ENABLED=0 go build -o /app/main .

# ランタイム
FROM gcr.io/distroless/static # <-- CHANGE THIS LINE
WORKDIR /app
COPY --from=builder /app/main .

EXPOSE 8080
CMD ["/app/main"]
