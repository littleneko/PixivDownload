# syntax=docker/dockerfile:1

## Build
FROM golang:latest AS build

WORKDIR /app

COPY go.mod ./
COPY go.sum ./
RUN go mod download

ADD pkg ./pkg
COPY main.go ./

RUN go build -o pixiv-dl

## Deploy
FROM ubuntu:latest

RUN apt update && apt install ca-certificates -y && update-ca-certificates

WORKDIR /pixivdl

COPY --from=build /app/pixiv-dl pixiv-dl

ENTRYPOINT ["/pixivdl/pixiv-dl"]
