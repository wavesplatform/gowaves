FROM ubuntu:22.04

ENV TZ=Etc/UTC
ENV APP_USER=gowaves

# Установка зависимостей
RUN apt-get update && apt-get install -y \
    ca-certificates \
    curl \
    dnsutils \
    tzdata \
    && rm -rf /var/lib/apt/lists/*

# Создание пользователя
RUN useradd --system --create-home --shell /usr/sbin/nologin $APP_USER

# Подготовка путей
RUN mkdir -p /home/gowaves/state && chown -R $APP_USER:$APP_USER /home/gowaves/state

# Копирование entrypoint и бинарника
COPY docker-entrypoint.sh /app/
COPY build/bin/linux-amd64/node /app/node

RUN chmod +x /app/docker-entrypoint.sh /app/node

USER $APP_USER

EXPOSE 6868 6869 7470
VOLUME /home/gowaves/state

HEALTHCHECK CMD ["curl", "--fail", "--silent", "http://localhost:6869/node/status"]

STOPSIGNAL SIGINT

ENTRYPOINT ["/app/docker-entrypoint.sh"]
CMD ["-state-path=/home/gowaves/state", "-bind-address=0.0.0.0:6868", "-api-address=0.0.0.0:6869", "-build-extended-api", "-serve-extended-api", "-build-state-hashes", "-enable-grpc-api", "-grpc-address=0.0.0.0:7470"]