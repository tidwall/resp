package resp

import (
	"fmt"
	"net"
	"os"
	"sync"
	"testing"
	"time"
)

func TestServer(t *testing.T) {
	// Use the example server in example_test
	go func() {
		ExampleServer()
	}()
	if os.Getenv("WAIT_ON_TEST_SERVER") == "1" {
		select {}
	}
	time.Sleep(time.Millisecond * 50)

	n := 75

	// Open N connections and do a bunch of stuff.
	var wg sync.WaitGroup
	wg.Add(n)
	for i := 0; i < n; i++ {
		go func(i int) {
			defer func() {
				wg.Done()
			}()
			nconn, err := net.Dial("tcp", ":6380")
			if err != nil {
				t.Fatal(err)
			}
			defer nconn.Close()
			conn := NewConn(nconn)

			// PING
			if err := conn.WriteMultiBulk("PING"); err != nil {
				t.Fatal(err)
			}
			val, _, err := conn.ReadValue()
			if err != nil {
				t.Fatal(err)
			}
			if val.String() != "PONG" {
				t.Fatalf("expecting 'PONG', got '%s'", val)
			}

			key := fmt.Sprintf("key:%d", i)

			// SET
			if err := conn.WriteMultiBulk("SET", key, 123.4); err != nil {
				t.Fatal(err)
			}
			val, _, err = conn.ReadValue()
			if err != nil {
				t.Fatal(err)
			}
			if val.String() != "OK" {
				t.Fatalf("expecting 'OK', got '%s'", val)
			}

			// GET
			if err := conn.WriteMultiBulk("GET", key); err != nil {
				t.Fatal(err)
			}
			val, _, err = conn.ReadValue()
			if err != nil {
				t.Fatal(err)
			}
			if val.Float() != 123.4 {
				t.Fatalf("expecting '123.4', got '%s'", val)
			}

			// QUIT
			if err := conn.WriteMultiBulk("QUIT"); err != nil {
				t.Fatal(err)
			}
			val, _, err = conn.ReadValue()
			if err != nil {
				t.Fatal(err)
			}
			if val.String() != "OK" {
				t.Fatalf("expecting 'OK', got '%s'", val)
			}

		}(i)
	}
	wg.Wait()
}
