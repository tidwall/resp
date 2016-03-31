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

func TestIntegers(t *testing.T) {
	var n, rn int
	var v Value
	var err error
	data := []byte(":1234567\r\n:-90898\r\n:0\r\n")
	r := NewReader(bytes.NewBuffer(data))
	v, rn, err = r.ReadValue()
	n += rn
	if err != nil {
		t.Fatal(err)
	}
	if v.Integer() != 1234567 {
		t.Fatalf("invalid integer: expected %d, got %d", 1234567, v.Integer())
	}
	v, rn, err = r.ReadValue()
	n += rn
	if err != nil {
		t.Fatal(err)
	}
	if v.Integer() != -90898 {
		t.Fatalf("invalid integer: expected %d, got %d", -90898, v.Integer())
	}
	v, rn, err = r.ReadValue()
	n += rn
	if err != nil {
		t.Fatal(err)
	}
	if v.Integer() != 0 {
		t.Fatalf("invalid integer: expected %d, got %d", 0, v.Integer())
	}
	v, rn, err = r.ReadValue()
	n += rn
	if err != io.EOF {
		t.Fatalf("invalid error: expected %v, got %v", io.EOF, err)
	}
	if n != len(data) {
		t.Fatalf("invalid read count: expected %d, got %d", len(data), n)
	}
}

func TestFloats(t *testing.T) {
	var n, rn int
	var v Value
	var err error
	data := []byte(":1234567\r\n+-90898\r\n$6\r\n12.345\r\n-90284.987\r\n")
	r := NewReader(bytes.NewBuffer(data))
	v, rn, err = r.ReadValue()
	n += rn
	if err != nil {
		t.Fatal(err)
	}
	if v.Float() != 1234567 {
		t.Fatalf("invalid integer: expected %v, got %v", 1234567, v.Float())
	}
	v, rn, err = r.ReadValue()
	n += rn
	if err != nil {
		t.Fatal(err)
	}
	if v.Float() != -90898 {
		t.Fatalf("invalid integer: expected %v, got %v", -90898, v.Float())
	}
	v, rn, err = r.ReadValue()
	n += rn
	if err != nil {
		t.Fatal(err)
	}
	if v.Float() != 12.345 {
		t.Fatalf("invalid integer: expected %v, got %v", 12.345, v.Float())
	}
	v, rn, err = r.ReadValue()
	n += rn
	if err != nil {
		t.Fatal(err)
	}
	if v.Float() != 90284.987 {
		t.Fatalf("invalid integer: expected %v, got %v", 90284.987, v.Float())
	}
	v, rn, err = r.ReadValue()
	n += rn
	if err != io.EOF {
		t.Fatalf("invalid error: expected %v, got %v", io.EOF, err)
	}
	if n != len(data) {
		t.Fatalf("invalid read count: expected %d, got %d", len(data), n)
	}
}

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

func BenchmarkRead(b *testing.B) {
	n := 1000
	var buf bytes.Buffer
	for k := 0; k < n; k++ {
		buf.WriteString(randRESPAny())
	}
	bb := buf.Bytes()
	b.ResetTimer()
	var j int
	var r *Reader
	//start := time.Now()
	var k int
	for i := 0; i < b.N; i++ {
		if j == 0 {
			r = NewReader(bytes.NewBuffer(bb))
			j = n
		}
		_, _, err := r.ReadValue()
		if err != nil {
			b.Fatal(err)
		}
		j--
		k++
	}
	//fmt.Printf("\n%f\n", float64(k)/(float64(time.Now().Sub(start))/float64(time.Second)))
}
