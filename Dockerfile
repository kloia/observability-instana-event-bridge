FROM golang:1.26-alpine AS builder

WORKDIR /app

COPY go.mod ./
COPY main.go ./

RUN go build -o instana-solarwinds-bridge


FROM alpine:latest

WORKDIR /app

COPY --from=builder /app/instana-event-bridge .
COPY config.json .

RUN apk add --no-cache tzdata

ENV TZ=Europe/Istanbul

RUN mkdir -p /app/events

EXPOSE 8080

CMD ["./instana-event-bridge"]