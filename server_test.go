package resp

import (
	"net"
	"os"
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
	val, err := conn.ReadValue()
	if err != nil {
		t.Fatal(err)
	}
	if val.String() != "PONG" {
		t.Fatalf("expecting 'PONG', got '%s'", val)
	}

	// SET
	if err := conn.WriteMultiBulk("SET", "test", 123.4); err != nil {
		t.Fatal(err)
	}
	val, err = conn.ReadValue()
	if err != nil {
		t.Fatal(err)
	}
	if val.String() != "OK" {
		t.Fatalf("expecting 'OK', got '%s'", val)
	}

	// GET
	if err := conn.WriteMultiBulk("GET", "test"); err != nil {
		t.Fatal(err)
	}
	val, err = conn.ReadValue()
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
	val, err = conn.ReadValue()
	if err != nil {
		t.Fatal(err)
	}
	if val.String() != "OK" {
		t.Fatalf("expecting 'OK', got '%s'", val)
	}

}
