package main

import (
  "os"
  "database/sql"
  "encoding/json"
  "encoding/hex"
  _ "github.com/mattn/go-sqlite3"
  "log"
  "strings"
  "strconv"
	"net/http"
	"github.com/googollee/go-socket.io"
  "github.com/vaughan0/go-zmq"
)

//https://forum.golangbridge.org/t/database-rows-scan-unknown-number-of-columns-json/7378/2
func queryToJson(db *sql.DB, query string, args ...interface{}) ([]byte, error) {
	// an array of JSON objects
	// the map key is the field name
	var objects []map[string]interface{}

	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		// figure out what columns were returned
		// the column names will be the JSON object field keys
		columns, err := rows.ColumnTypes()
		if err != nil {
			return nil, err
		}

		// Scan needs an array of pointers to the values it is setting
		// This creates the object and sets the values correctly
		values := make([]interface{}, len(columns))
		object := map[string]interface{}{}
		for i, column := range columns {
			var v interface{}

			switch column.DatabaseTypeName() {
			case "TEXT":
				v = new(sql.NullString)
			default:
				v = new(interface{})
			}

			object[column.Name()] = v
	    values[i] = v
		}

		err = rows.Scan(values...)
		if err != nil {
			return nil, err
		}

    for _, column := range columns {
      if column.DatabaseTypeName() == "TEXT" {
        name := column.Name()
        ob := object[name].(*sql.NullString)

        if ob.Valid {
          object[name] = ob.String
        } else {
          object[name] = nil
        }
      }
    }

		objects = append(objects, object)
	}

	// indent because I want to read the output
	return json.MarshalIndent(objects, "", "\t")
}

func zmqwatch(endpoint string, blockhash chan<- string, txhash chan<- string) {
  log.Println("Connecting to ZMQ", endpoint)
  context, zerr := zmq.NewContext()
  defer context.Close()

  if zerr != nil {
    log.Println("Couldn't connect to ZMQ, won't be able to announce new blocks or TXs")
    return
  }

  requester, rerr := context.Socket(zmq.Sub)
	defer requester.Close()

  if rerr != nil {
    log.Println("Couldn't create ZMQ subber, won't be able to announce new blocks or TXs")
    return
  }

  requester.Connect(endpoint)

  chans := requester.Channels()
  defer chans.Close()

  requester.Subscribe([]byte("hashblock"))
  requester.Subscribe([]byte("hashtx"))

  for {
    select {
    case msg := <-chans.In():
      go func() {
        topic, messg := string(msg[0]), hex.EncodeToString(msg[1])
        //resp := doSomething(msg)
        if topic == "hashblock" {
          blockhash <- messg
        } else if topic == "hashtx" {
          txhash <- messg
        }
      }()
    case err := <-chans.Errors():
      log.Fatal(err)
      return
    }
  }
}

func main() {

  log.Println("Running....")

  if len(os.Args) < 4 {
    panic("Needs 3 parameters: addr:port db_path zeromq_url")
  }

  server, err := socketio.NewServer(nil)
  if err != nil {
  	log.Fatal(err)
  }

  blockhash, txhash := make(chan string), make(chan string)

  server.OnConnect("/", func(s socketio.Conn) error {

    s.SetContext("")
  	log.Println("client connected")

    go func(block <-chan string, tx <-chan string) {
      for {
        select {
        case bhx := <-block:
          s.Emit("blocks", "hashblock", bhx)
        case thx := <-tx:
          s.Emit("txs", "hashtx", thx)
        }
      }
    }(blockhash, txhash)

    return nil

  })

  server.OnEvent("/", "chat message", func(so socketio.Conn, msg string) {
    db, err := sql.Open("sqlite3", os.Args[2])
    checkErr(err)
    defer db.Close()

    seq := -1

    if strings.HasPrefix(msg, "/*") {
      idx := strings.Index(msg, "*/")

      if idx >= 0 {
        n, err := strconv.Atoi(msg[2:idx])

        if err == nil {
          seq = n
        }

        msg = msg[idx + 2:]
      }
    }

    b, err := queryToJson(db, msg) // Super insecure if DB isn't readonly, which is our case
    if err != nil {
      so.Emit("chat message", "Error: " + err.Error())
    } else {
      query := string(b[:])
      so.Emit("chat message", strconv.Itoa(seq) + "|" + query)
    }
  })


  server.OnDisconnect("/", func(s socketio.Conn, msg string) {
    log.Println("client disconnected")
  })

  server.OnError("error", func(s socketio.Conn, err error) {
    log.Println("error:", err)
  })

  go server.Serve()
  defer server.Close()

  http.Handle("/socket.io/", server)
  http.Handle("/", http.FileServer(http.Dir("./asset")))
  log.Println("Serving at", os.Args[1])
  go zmqwatch(os.Args[3], blockhash, txhash)
  log.Fatal(http.ListenAndServe(os.Args[1], nil))


}

func checkErr(err error) {
    if err != nil {
        panic(err)
    }
}
