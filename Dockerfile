FROM golang:1.26-alpine AS build

WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /out/telegram-public-mcp ./cmd/telegram-public-mcp

FROM alpine:3.22
RUN adduser -D -H app
USER app
COPY --from=build /out/telegram-public-mcp /usr/local/bin/telegram-public-mcp
EXPOSE 8080
ENV ADDR=:8080
ENTRYPOINT ["telegram-public-mcp"]
