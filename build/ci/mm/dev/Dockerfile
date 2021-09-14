FROM golang:1.17-bullseye AS build

WORKDIR /api-gateway
RUN apt update && apt install build-essential

COPY . .
RUN go mod download && \
    make all

FROM debian:bullseye

WORKDIR /api-gateway

COPY --from=build ./api-gateway/bin/ .
COPY configs ./configs
COPY graph/member/*.graphql ./graph/member/
COPY scripts/start.sh .

EXPOSE 80
CMD ["./start.sh"]
