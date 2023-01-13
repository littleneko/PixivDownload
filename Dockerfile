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

RUN go build -o /pixiv-dl

## Deploy
FROM ubuntu:jammy

RUN apt update && apt install ca-certificates -y && update-ca-certificates

WORKDIR /

COPY --from=build /pixiv-dl /pixiv-dl

CMD ["/pixiv-dl", "download", "--service-mode"]
