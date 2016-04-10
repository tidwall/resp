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

	var multi []Value
	for i := 0; i < 50; i++ {
		multi = append(multi, StringValue(fmt.Sprintf("hello multi world #%d\n", i)))
	}
	if err := f.AppendMulti(multi); err != nil {
		t.Fatal(err)
	}

	skip := i
	i = 0
	j := 0
	if err := f.Scan(func(v Value) {
		if i >= skip {
			s := v.String()
			e := fmt.Sprintf("hello multi world #%d\n", j)
			if s != e {
				t.Fatalf("#%d is '%s', expect '%s'", j, s, e)
			}
			j++
		}
		i++
	}); err != nil {
		t.Fatal(err)
	}

}
