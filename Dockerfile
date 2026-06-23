FROM alpine:3.20
RUN ALPINE_VERSION="v$(cut -d. -f1,2 /etc/alpine-release)" \
 && printf 'https://mirror.arvancloud.ir/alpine/%s/main\nhttps://mirror.arvancloud.ir/alpine/%s/community\n' "$ALPINE_VERSION" "$ALPINE_VERSION" > /etc/apk/repositories \
 && apk add --no-cache ca-certificates tzdata
WORKDIR /app
COPY bin/server /usr/local/bin/server
COPY dist /app/static
RUN mkdir -p /app/data
ENV DATABASE_PATH=/app/data/expenses.db
ENV STATIC_DIR=/app/static
ENV ADDR=:8080
EXPOSE 8080
VOLUME ["/app/data"]
ENTRYPOINT ["/usr/local/bin/server"]
