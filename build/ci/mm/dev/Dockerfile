FROM golang:1.17-alpine AS build

WORKDIR /api-gateway
RUN apk add --update --no-cache make

COPY . .
RUN go mod download && \
    make all

FROM alpine:latest

WORKDIR /api-gateway

COPY --from=build ./api-gateway/bin/ .
COPY configs ./configs
COPY graph/member/*.graphql ./graph/member/
COPY scripts/start.sh .

EXPOSE 80
CMD ["./start.sh"]
