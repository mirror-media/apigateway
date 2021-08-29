FROM golang:1.16-alpine AS build

WORKDIR /api-gateway

COPY . .

RUN apk add --update --no-cache make && \
    go get ./... && \
    make all

FROM alpine:latest

WORKDIR /api-gateway

COPY --from=build /api-gateway/bin/ .
COPY configs ./configs
COPY graph ./graphql

EXPOSE 8080
CMD ["./apigateway"]
