# syntax=docker/dockerfile:1

## Build
FROM golang:latest AS build

WORKDIR /app

COPY go.mod ./
COPY go.sum ./
RUN go mod download

ADD app ./app
ADD cmd ./cmd
COPY main.go ./

RUN CGO_ENABLED=0 go build -o /pixiv-dl

## Deploy
FROM alpine:latest

ENV TZ Asia/Shanghai
RUN apk update && apk --no-cache add tzdata && cp /usr/share/zoneinfo/${TZ} /etc/localtime && echo ${TZ} > /etc/timezone

RUN apk update && apk --no-cache add ca-certificates && update-ca-certificates

RUN apk update && apk --no-cache add su-exec


WORKDIR /

COPY --from=build /pixiv-dl /pixiv-dl
COPY entrypoint.sh /entrypoint.sh
RUN chmod +x /pixiv-dl
RUN chmod +x /entrypoint.sh

ENTRYPOINT ["/entrypoint.sh", "/pixiv-dl", "download", "--service-mode"]
