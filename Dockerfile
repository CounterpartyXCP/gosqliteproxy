FROM golang:1.11-alpine as base
RUN apk update && apk upgrade && apk add git gcc musl-dev libzmq zeromq-dev
RUN CGO_ENABLED=1 go get github.com/mattn/go-sqlite3
RUN go get github.com/googollee/go-socket.io
RUN go get github.com/vaughan0/go-zmq

WORKDIR /usr/src
COPY . .
RUN go build -o main
RUN strip /usr/src/main

FROM scratch
COPY --from=base /usr/src/main /gosqliteproxy
COPY ./asset /asset

EXPOSE 5000

CMD ["/gosqliteproxy", ":5000", "/data/database.sqlite?immutable=1", "bitcoin:8833"]
