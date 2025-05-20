FROM alpine:3.21.3@sha256:a8560b36e8b8210634f77d9f7f9efd7ffa463e380b75e2e74aff4511df3ef88c

ARG TARGETOS
ARG TARGETARCH

ARG UID=1000
ARG GID=1000

ENV TZ=Etc/UTC
ENV APP_USER=gowaves

RUN apk add --no-cache bind-tools curl

RUN addgroup -S $APP_USER && adduser -S $APP_USER -G $APP_USER

RUN mkdir -p /home/gowaves/state
RUN chown -R $APP_USER:$APP_USER /home/gowaves/state

ENV CONFIG_PATH=/home/gowaves/config/gowaves-it.json \
    STATE_PATH=/home/gowaves/state \
    WALLET_PATH=/home/gowaves/wallet/go.wallet \
    BIND_ADDR=0.0.0.0:6868 \
    API_ADDR=0.0.0.0:6869 \
    GRPC_ADDR=0.0.0.0:7470

# Копирование бинарника и entrypoint
COPY docker-entrypoint.sh /app/
COPY build/bin/$TARGETOS-$TARGETARCH/node /app/node

RUN chmod +x /app/docker-entrypoint.sh

USER $APP_USER

HEALTHCHECK CMD ["curl", "--fail", "--silent", "http://localhost:6869/node/status"]

EXPOSE 6868 6869 7470
VOLUME /home/gowaves/state

STOPSIGNAL SIGINT

ENTRYPOINT ["/app/docker-entrypoint.sh"]
CMD []