FROM alpine:3.24@sha256:a2d49ea686c2adfe3c992e47dc3b5e7fa6e6b5055609400dc2acaeb241c829f4

ARG TARGETOS
ARG TARGETARCH

ENV TZ=Etc/UTC
ENV APP_USER=gowaves

RUN apk add --no-cache bind-tools curl

RUN addgroup -S $APP_USER && adduser -S $APP_USER -G $APP_USER

RUN mkdir -p /home/gowaves/state
RUN chown -R $APP_USER:$APP_USER /home/gowaves/state

COPY docker-entrypoint.sh /app/
RUN chmod +x /app/docker-entrypoint.sh

USER $APP_USER

COPY build/bin/$TARGETOS-$TARGETARCH/node /app/node

HEALTHCHECK CMD ["curl", "--fail", "--silent", "http://localhost:6869/node/status"]

EXPOSE 6868 6869 7470
VOLUME /home/gowaves/state

STOPSIGNAL SIGINT

ENTRYPOINT ["/app/docker-entrypoint.sh"]
CMD ["-state-path=/home/gowaves/state", "-bind-address=0.0.0.0:6868","-api-address=0.0.0.0:6869", "-build-extended-api", "-serve-extended-api", "-build-state-hashes", "-enable-grpc-api", "-grpc-address=0.0.0.0:7470"]
