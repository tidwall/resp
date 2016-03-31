package resp

import (
	"fmt"
	"os"
	"testing"
)

func TestAOF(t *testing.T) {
	defer func() {
		os.RemoveAll("aof.tmp")
	}()
	os.RemoveAll("aof.tmp")
	f, err := OpenAOF("aof.tmp")
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		f.Close()
	}()
	for i := 0; i < 12345; i++ {
		if err := f.Append(StringValue(fmt.Sprintf("hello world #%d\n", i))); err != nil {
			t.Fatal(err)
		}
	}
	i := 0
	if err := f.Scan(func(v Value) {
		s := v.String()
		e := fmt.Sprintf("hello world #%d\n", i)
		if s != e {
			t.Fatalf("#%d is '%s', expect '%s'", i, s, e)
		}
		i++
	}); err != nil {
		t.Fatal(err)
	}
	f.Close()
	f, err = OpenAOF("aof.tmp")
	if err != nil {
		t.Fatal(err)
	}
	c := i
	for i := c; i < c+12345; i++ {
		if err := f.Append(StringValue(fmt.Sprintf("hello world #%d\n", i))); err != nil {
			t.Fatal(err)
		}
	}
	i = 0
	if err := f.Scan(func(v Value) {
		s := v.String()
		e := fmt.Sprintf("hello world #%d\n", i)
		if s != e {
			t.Fatalf("#%d is '%s', expect '%s'", i, s, e)
		}
		i++
	}); err != nil {
		t.Fatal(err)
	}
}
