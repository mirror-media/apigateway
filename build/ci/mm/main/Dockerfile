FROM golang:1.17-bullseye AS build

WORKDIR /api-gateway
RUN apt update && apt install build-essential -y

COPY . .
RUN go mod download && \
    make all

FROM debian:bullseye

WORKDIR /api-gateway

RUN apt update && apt install ca-certificates -y && apt clean 

COPY --from=build ./api-gateway/bin/ .
COPY graph/member/*.graphql ./graph/member/
COPY scripts/start.sh .

EXPOSE 8080
CMD ["./start.sh"]
