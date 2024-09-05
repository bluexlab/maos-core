FROM golang:1.22.2-alpine3.19 as builder

ENV GO111MODULE=on \
    CGO_ENABLED=0 \
    GOOS=linux \
    GOARCH=amd64 \
    GOPROXY=https://proxy.golang.org,direct

RUN apk add --no-cache \
    binutils \
    ca-certificates \
    tzdata

# setup the working directory
WORKDIR /app/src

# Download dependencies
COPY go.mod /app/src/
COPY go.sum /app/src/
RUN go mod download

# add source code
COPY . /app/src/

# build the maos-core-server
RUN cd /app/src \
    && echo $BUILDNO > BUILD \
    && go build -o /go/bin/maos-core-server ./app/server \
    && go build -o /go/bin/maos-core-migrate ./app/migrate

# FROM scratch
FROM alpine:3.19

ARG BUILDNO=local
ARG REV=unknown
ARG APP_HOME=/app
ENV PATH=$APP_HOME:$PATH

WORKDIR $APP_HOME

COPY --from=builder /usr/local/go/lib/time/zoneinfo.zip /usr/local/go/lib/time/zoneinfo.zip
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /go/bin/maos-core-server $APP_HOME/maos-core-server
COPY --from=builder /go/bin/maos-core-migrate $APP_HOME/maos-core-migrate
COPY migrate.sh $APP_HOME/migrate.sh

# add launch shell command
COPY docker-entrypoint.sh /usr/bin/

RUN echo $BUILDNO > $APP_HOME/BUILD \
    && echo $REV > $APP_HOME/REV \
    && echo "INFO: BLUEX-RELEASE maos-core-server -- Rev: $REV -- Build: $BUILDNO" > /MANIFEST

RUN addgroup -S appgroup && adduser -h $APP_HOME -G appgroup -S -D -H appuser
RUN chown -R appuser:appgroup $APP_HOME

USER appuser

ENV PORT 5000

EXPOSE 5000

ENTRYPOINT ["docker-entrypoint.sh"]
