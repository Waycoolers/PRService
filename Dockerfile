# Альпайн + golang билд
FROM golang:1.25-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /out/PRService ./cmd/server

FROM alpine:3.18

COPY --from=build /out/PRService /PRService

ENTRYPOINT ["/PRService"]