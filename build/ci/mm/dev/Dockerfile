FROM golang:1.15-alpine AS build

WORKDIR /api-gateway

COPY . .

RUN apk add --update --no-cache make && \
    go get ./... && \
    make all

FROM alpine:latest

WORKDIR /api-gateway

COPY --from=build /api-gateway/bin/ .
COPY firebaseCredential.json .
COPY config.yaml .

EXPOSE 8080
CMD ["./apigateway"]
