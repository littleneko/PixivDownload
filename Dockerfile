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

WORKDIR /

COPY --from=build /pixiv-dl /pixiv-dl

CMD ["/pixiv-dl", "download", "--service-mode"]
