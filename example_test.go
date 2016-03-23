package resp

import (
	"errors"
	"log"
	"sync"
)

func ExampleServer() {
	// ExampleServer is a Redis clone that implements the SET and GET commands.
	// The server runs on port 6380.
	// You can interact using the Redis CLI (redis-cli) The http://redis.io/download.
	// Or, use the telnet by typing in "telnet localhost 6380" and type in "set key value" and "get key".
	// Or, use a client library such as "http://github.com/garyburd/redigo"
	// The "QUIT" command will close the connection.
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
}
