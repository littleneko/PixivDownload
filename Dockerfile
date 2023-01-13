# syntax=docker/dockerfile:1

## Build
FROM golang:latest AS build

WORKDIR /app

COPY go.mod ./
COPY go.sum ./
RUN go mod download

ADD pkg ./pkg
COPY main.go ./

RUN go build -o /pixiv-dl

## Deploy
FROM ubuntu:latest

RUN apt update && apt install ca-certificates -y && update-ca-certificates

WORKDIR /

COPY --from=build /pixiv-dl /pixiv-dl

ENTRYPOINT ["/pixiv-dl"]
