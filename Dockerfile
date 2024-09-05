FROM golang:1.22.2 as builder

ENV GO111MODULE=on \
    CGO_ENABLED=0 \
    GOOS=linux \
    GOARCH=amd64 \
    GOPROXY=https://proxy.golang.org,direct

ARG BUILDNO=local

RUN apt-get update && apt-get install -y \
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

# build the maos-core-server and maos-core-migrate
RUN cd /app/src \
    && echo $BUILDNO > BUILD \
    && go build -o /go/bin/maos-core-server ./app/server \
    && go build -o /go/bin/maos-core-migrate ./app/migrate

# FROM scratch
FROM gcr.io/distroless/static-debian11

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

USER nonroot:nonroot
ENV PORT 5000
EXPOSE 5000

CMD ["./maos-core-server"]
