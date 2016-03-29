package resp

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"strings"
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

func ExampleReader() {
	raw := "*3\r\n$3\r\nset\r\n$6\r\nleader\r\n$7\r\nCharlie\r\n"
	raw += "*3\r\n$3\r\nset\r\n$8\r\nfollower\r\n$6\r\nSkyler\r\n"
	rd := NewReader(bytes.NewBufferString(raw))
	for {
		v, _, err := rd.ReadValue()
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Fatal(err)
		}
		fmt.Printf("Read %s\n", v.Type())
		if v.Type() == Array {
			for i, v := range v.Array() {
				fmt.Printf("  #%d %s, value: '%s'\n", i, v.Type(), v)
			}
		}
	}
	// Output:
	// Read Array
	//   #0 BulkString, value: 'set'
	//   #1 BulkString, value: 'leader'
	//   #2 BulkString, value: 'Charlie'
	// Read Array
	//   #0 BulkString, value: 'set'
	//   #1 BulkString, value: 'follower'
	//   #2 BulkString, value: 'Skyler'
}

func ExampleWriter() {
	var buf bytes.Buffer
	wr := NewWriter(&buf)
	wr.WriteArray([]Value{StringValue("set"), StringValue("leader"), StringValue("Charlie")})
	wr.WriteArray([]Value{StringValue("set"), StringValue("follower"), StringValue("Skyler")})
	fmt.Printf("%s", strings.Replace(buf.String(), "\r\n", "\\r\\n", -1))
	// Output:
	// *3\r\n$3\r\nset\r\n$6\r\nleader\r\n$7\r\nCharlie\r\n*3\r\n$3\r\nset\r\n$8\r\nfollower\r\n$6\r\nSkyler\r\n
}

func ExampleAOF() {
	os.RemoveAll("appendonly.aof")

	// create and fill an appendonly file
	aof, err := OpenAOF("appendonly.aof")
	if err != nil {
		log.Fatal(err)
	}
	// append a couple values and close the file
	aof.Append(MultiBulkValue("set", "leader", "Charlie"))
	aof.Append(MultiBulkValue("set", "follower", "Skyler"))
	aof.Close()

	// reopen and scan all values
	aof, err = OpenAOF("appendonly.aof")
	if err != nil {
		log.Fatal(err)
	}
	defer aof.Close()
	aof.Scan(func(v Value) {
		fmt.Printf("%s\n", v.String())
	})

	// Output:
	// [set leader Charlie]
	// [set follower Skyler]
}
