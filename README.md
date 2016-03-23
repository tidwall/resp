RESP
====

[![Build Status](https://travis-ci.org/tidwall/resp.svg?branch=master)](https://travis-ci.org/tidwall/resp)
[![GoDoc](https://godoc.org/github.com/tidwall/resp?status.svg)](https://godoc.org/github.com/tidwall/resp)

RESP is a [Go](http://golang.org/) library that provides a reader, writer, and server implementation for the [Redis RESP Protocol](http://redis.io/topics/protocol).

RESP is short for **REdis Serialization Protocol**.
While the protocol was designed specifically for Redis, it can be used for other client-server software projects.

The RESP protocol has the advantages of being human readable and with performance of a binary protocol.

## Example Server
A Redis clone that implements the SET and GET commands.

- The server runs on port 6380.
- You can interact using the Redis CLI (redis-cli). http://redis.io/download
- Or, use the telnet by typing in "telnet localhost 6380" and type in "set key value" and "get key".
- Or, use a client library such as http://github.com/garyburd/redigo
- The "QUIT" command will close the connection.

```go
    var mu sync.RWMutex
    kvs := make(map[string]string)
    s := NewServer()
    s.HandleFunc("set", func(conn *Conn, args []Value) bool {
        if len(args) != 3 {
            conn.WriteError(errors.New("ERR wrong number of arguments for 'set' command"))
        } else {
            mu.Lock()
            kvs[args[1].String()] = args[2].String()
            mu.Unlock()
            conn.WriteSimpleString("OK")
        }
        return true
    })
    s.HandleFunc("get", func(conn *Conn, args []Value) bool {
        if len(args) != 2 {
            conn.WriteError(errors.New("ERR wrong number of arguments for 'get' command"))
        } else {
            mu.RLock()
            s, ok := kvs[args[1].String()]
            mu.RUnlock()
            if !ok {
                conn.WriteNull()
            } else {
                conn.WriteString(s)
            }
        }
        return true
    })
    if err := s.ListenAndServe(":6380"); err != nil {
        log.Fatal(err)
    }
```

## Clients

There are bunches of [RESP Clients](http://redis.io/clients). Most any client that supports Redis will support this implementation.

## Contact

Josh Baker [@tidwall](http://twitter.com/tidwall)

## License

Tile38 source code is available under the MIT [License](/LICENSE).

