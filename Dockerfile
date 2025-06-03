FROM alpine:3.22.0@sha256:8a1f59ffb675680d47db6337b49d22281a139e9d709335b492be023728e11715

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
