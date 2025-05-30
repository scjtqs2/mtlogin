FROM golang:1.22-alpine AS builder
RUN go env -w GO111MODULE=auto \
  && go env -w CGO_ENABLED=0
#  && go env -w GOPROXY=https://goproxy.cn,direct

WORKDIR /build

COPY ./ .

RUN set -ex \
    && cd /build \
    && go build -ldflags "-s -w -extldflags '-static'" -o mtlogin

FROM alpine:latest

COPY docker-entrypoint.sh /docker-entrypoint.sh

RUN chmod +x /docker-entrypoint.sh && \
    apk add --no-cache --update \
      coreutils \
      shadow \
      su-exec \
      tzdata && \
    rm -rf /var/cache/apk/* && \
    mkdir -p /app && \
    mkdir -p /data && \
    mkdir -p /config && \
    useradd -d /config -s /bin/sh abc && \
    chown -R abc /config && \
    chown -R abc /data

ENV TZ="Asia/Shanghai"
ENV UID=99
ENV GID=100
ENV UMASK=002
ENV COOKIE_MODE="normal"

COPY --from=builder /build/mtlogin /app/

WORKDIR /data

VOLUME [ "/data" ]

ENTRYPOINT [ "/docker-entrypoint.sh" ]
CMD [ "/app/mtlogin" ]