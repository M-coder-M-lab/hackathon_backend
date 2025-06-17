# ビルドステージ
FROM golang:1.24 AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN go build -o /app/main . # This builds the executable at /app/main in the builder stage

# ランタイム
FROM scratch
WORKDIR /app
COPY --from=builder /app/main . # This copies /app/main from the builder stage to /app/main in the scratch stage

EXPOSE 8080
CMD ["/app/main"] # This tries to run /app/main in the scratch stage
