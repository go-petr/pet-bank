# Build stage
FROM golang:1.18.10-alpine3.17 AS builder
WORKDIR /usr/app
COPY . .
RUN go build -o main cmd/server/main.go

# Run stage
FROM alpine:3.17
WORKDIR /usr/app
COPY --from=builder /usr/app/main .
COPY configs/app.env configs/

EXPOSE 8080
CMD ["/usr/app/main"]