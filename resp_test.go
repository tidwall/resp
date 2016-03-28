package resp

import (
	"bytes"
	"crypto/rand"
	"encoding/binary"
	"fmt"
	"io"
	"strconv"
	"strings"
	"testing"
)

// TestLotsaRandomness does generates N resp messages and reads the values though a Reader.
// It then marshals the values back to strings and compares to the original.
// All data and resp types are random.
func TestLotsaRandomness(t *testing.T) {
	n := 10000
	var anys []string
	var buf bytes.Buffer
	for i := 0; i < n; i++ {
		any := randRESPAny()
		anys = append(anys, any)
		buf.WriteString(any)
	}
	r := NewReader(bytes.NewBuffer(buf.Bytes()))
	for i := 0; i < n; i++ {
		v, _, err := r.ReadValue()
		if err != nil {
			t.Fatal(err)
		}
		resp, err := v.MarshalRESP()
		if err != nil {
			t.Fatal(err)
		}
		if string(resp) != anys[i] {
			t.Fatalf("resp failed to remarshal #%d\n-- original --\n%s\n-- remarshalled --\n%s\n-- done --", i, anys[i], string(resp))
		}
		if &resp[0] != &v.buf[0] {
			t.Fatalf("resp failed to used original buffer #%d\n", i)
		}
	}
}

func randRESPInteger() string {
	return fmt.Sprintf(":%d\r\n", (randInt()%1000000)-500000)
}
func randRESPSimpleString() string {
	return "+" + strings.Replace(randString(), "\r\n", "", -1) + "\r\n"
}
func randRESPError() string {
	return "-" + strings.Replace(randString(), "\r\n", "", -1) + "\r\n"
}
func randRESPBulkString() string {
	s := randString()
	if len(s)%1024 == 0 {
		return "$-1\r\n"
	}
	return "$" + strconv.FormatInt(int64(len(s)), 10) + "\r\n" + s + "\r\n"
}
func randRESPArray() string {
	n := randInt() % 10
	if n%10 == 0 {
		return "$-1\r\n"
	}
	s := "*" + strconv.FormatInt(int64(n), 10) + "\r\n"
	for i := 0; i < n; i++ {
		rn := randInt() % 100
		if rn == 0 {
			s += randRESPArray()
		} else {
			switch (rn - 1) % 4 {
			case 0:
				s += randRESPInteger()
			case 1:
				s += randRESPSimpleString()
			case 2:
				s += randRESPError()
			case 3:
				s += randRESPBulkString()
			}
		}
	}
	return s
}

func randInt() int {
	n := int(binary.LittleEndian.Uint64(randBytes(8)))
	if n < 0 {
		n *= -1
	}
	return n
}

func randBytes(n int) []byte {
	b := make([]byte, n)
	if _, err := io.ReadFull(rand.Reader, b); err != nil {
		panic("random error: " + err.Error())
	}
	return b
}

func randString() string {
	return string(randBytes(randInt() % 1024))
}

func randRESPAny() string {
	switch randInt() % 5 {
	case 0:
		return randRESPInteger()
	case 1:
		return randRESPSimpleString()
	case 2:
		return randRESPError()
	case 3:
		return randRESPBulkString()
	case 4:
		return randRESPArray()
	}
	panic("?")
}
