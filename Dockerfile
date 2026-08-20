# syntax=docker/dockerfile:1

## ---- build stage ----
FROM golang:1.25-alpine AS build

RUN apk add --no-cache git ca-certificates && update-ca-certificates

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY cmd/ cmd/
COPY internal/ internal/

ARG VERSION=dev
ARG COMMIT=none
ARG BUILD_DATE=unknown

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -trimpath \
    -ldflags="-s -w \
      -X github.com/alirezawaskari/cloudforge/internal/version.Version=${VERSION} \
      -X github.com/alirezawaskari/cloudforge/internal/version.Commit=${COMMIT} \
      -X github.com/alirezawaskari/cloudforge/internal/version.BuildDate=${BUILD_DATE}" \
    -o /out/cloudforge-api ./cmd/api

## ---- runtime stage ----
FROM alpine:3.20 AS runtime

RUN apk add --no-cache ca-certificates && \
    addgroup -S app && adduser -S -G app -u 10001 app

WORKDIR /

COPY --from=build /out/cloudforge-api /cloudforge-api

USER app:app

EXPOSE 8080

ENTRYPOINT ["/cloudforge-api"]
