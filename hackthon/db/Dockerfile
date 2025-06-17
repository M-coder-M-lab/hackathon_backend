# Build stage
FROM golang:1.24 AS builder
WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o main .

# Final stage using distroless
FROM gcr.io/distroless/static:nonroot
WORKDIR /app
COPY --from=builder /app/main .

USER nonroot
EXPOSE 8080
CMD ["/app/main"]

COPY ./server-ca.pem ./client-cert.pem ./client-key.pem ./
