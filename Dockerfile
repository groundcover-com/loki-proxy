FROM alpine:3.16.2

ENV GIN_MODE=release

WORKDIR /app

COPY loki-proxy .

ENTRYPOINT [ "/app/loki-proxy" ]
