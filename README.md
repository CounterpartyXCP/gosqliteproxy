# gosqliteproxy

A sqlite3 + socket.io + zmq proxy, meant for readonly access to databases that are linked to bitcoin transactions (i.e. Counterparty).

## Running

### Clone

    git clone https://github.com/CounterpartyXCP/gosqliteproxy

### Run with docker

Assuming you have bitcoind running on `bitcoin_zmq_endpoint` on port 8833, run with docker:

    docker build -t gosqliteproxy:latest .
    docker run -v /path/to/yourdb/:/data:ro -p 5000:5000 gosqliteproxy:latest gosqliteproxy :5000 /data/database.sqlite?immutable=1 bitcoin_zmq_endpoint:8833

Important to note: Mounting volumes this way makes it read-only, so any change to the database is impossible.

### Build standalone

You need to setup golang correctly, and install dependencies. One dependency needs CGO_ENABLED=1

    CGO_ENABLED=1 go get github.com/mattn/go-sqlite3
    go get github.com/googollee/go-socket.io
    go get github.com/vaughan0/go-zmq
    go build .
    strip gosqliteproxy # no need for debug data
    gosqliteproxy :5000 /data/database.sqlite?immutable=1 bitcoin_zmq_endpoint:8833

Access the sql console via HTTP at port 5000

### Immutable?

The `immutable` flag on the file path is needed if you want to prevent unauthorized write access to the sqlite database.

When using docker, a read-only mount is advised.

## License [MIT](LICENSE)
