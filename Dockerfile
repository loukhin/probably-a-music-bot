FROM golang:1.23.4-alpine3.21 as builder

WORKDIR /opt/probably-music-bot

COPY . .

RUN go build -o probably-music-bot


FROM alpine:3.21

RUN apk add dumb-init

COPY --from=builder --chmod=0755 /opt/probably-music-bot/probably-music-bot /usr/local/bin/

ENTRYPOINT ["dumb-init"]
CMD ["probably-music-bot"]
