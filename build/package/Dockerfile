# Build stage
FROM golang:1.18.10-alpine3.17 AS builder
WORKDIR /usr/app
COPY . .
RUN pwd
RUN go build -o main /usr/app/cmd/main.go

# Run stage
FROM alpine:3.17
WORKDIR /usr/app
COPY --from=builder /usr/app/main .
COPY configs ./configs

# Credentials for AWS image registry deploy workflow
ARG DB_DRIVER
ARG DB_SOURCE
ARG SERVER_ADDRESS
ARG TOKEN_SYMMETRIC_KEY
ARG ACCESS_TOKEN_DURATION

ENV DB_DRIVER=$DB_DRIVER
ENV DB_SOURCE=$DB_SOURCE
ENV SERVER_ADDRESS=$SERVER_ADDRESS
ENV TOKEN_SYMMETRIC_KEY=$TOKEN_SYMMETRIC_KEY
ENV ACCESS_TOKEN_DURATION=$ACCESS_TOKEN_DURATION

EXPOSE 8080
CMD ["/usr/app/main"]