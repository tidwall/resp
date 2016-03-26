package resp

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"strconv"
)

var nullValue = Value{0, nil}

type errProtocol struct{ msg string }

func (err errProtocol) Error() string {
	return "Protocol error: " + err.msg
}

// Value represents the data of a valid RESP type.
type Value struct {
	typ  byte
	base interface{}
}

// String converts Value to a string.
func (v Value) String() string {
	switch v := v.base.(type) {
	default:
		return fmt.Sprintf("%v", v)
	case nil:
		return ""
	case string:
		return v
	}
}

// Float converts Value to a float64. If Value cannot be converted, Zero is returned.
func (v Value) Float() float64 {
	switch v := v.base.(type) {
	case int:
		return float64(v)
	}
	f, _ := strconv.ParseFloat(v.String(), 64)
	return f
}

// Integer converts Value to an int. If Value cannot be converted, Zero is returned.
func (v Value) Integer() int {
	switch v := v.base.(type) {
	case int:
		return v
	}
	return 0
}

// Bytes converts the Value to a byte array. An empty string is converted to a non-nil empty byte array. If it's a RESP Null value, nil is returned.
func (v Value) Bytes() []byte {
	if v.base == nil {
		return nil
	}
	return []byte(v.String())
}

// Error converts the Value to an error. If Value is not an error, nil is returned.
func (v Value) Error() error {
	switch v := v.base.(type) {
	case error:
		return v
	}
	return nil
}

// Array converts the Value to a an array. If Value is not an array or when it's is a RESP Null value, nil is returned.
func (v Value) Array() []Value {
	switch v := v.base.(type) {
	case []Value:
		return v
	}
	return nil
}

// Type returns the underlying RESP type. The following types are represent valid RESP values.
//   '+'  SimpleString
//   '-'  Error
//   ':'  Integer
//   '$'  BulkString
//   '*'  Array
func (v Value) Type() byte {
	return v.typ
}

// MarshalRESP returns the original serialized byte representation of Value.
// For more information on this format please see http://redis.io/topics/protocol.
func (v Value) MarshalRESP() ([]byte, error) {
	switch v.typ {
	default:
		return nil, errors.New("unknown resp type encountered")
	case '+', '-', ':':
		return []byte(string(v.typ) + v.String() + "\r\n"), nil
	case '$':
		if v.base == nil {
			return []byte("$-1\r\n"), nil
		}
		s := v.String()
		return []byte("$" + strconv.FormatInt(int64(len(s)), 10) + "\r\n" + s + "\r\n"), nil
	case '*':
		if v.base == nil {
			return []byte("*-1\r\n"), nil
		}
		a := v.Array()
		var buf bytes.Buffer
		buf.WriteString("*" + strconv.FormatInt(int64(len(a)), 10) + "\r\n")
		for _, v := range a {
			b, err := v.MarshalRESP()
			if err != nil {
				return nil, err
			}
			buf.Write(b)
		}
		return buf.Bytes(), nil
	}
}

// Reader is a specialized RESP Value type reader.
type Reader struct {
	bufrd *bufio.Reader
}

// NewReader returns a Reader for reading Value types.
func NewReader(rd io.Reader) *Reader {
	return &Reader{bufio.NewReader(rd)}
}

// ReadValue reads the next Value from Reader.
func (rd *Reader) ReadValue() (Value, error) {
	b, err := rd.bufrd.ReadByte()
	if err != nil {
		return nullValue, err
	}
	var v Value
	switch b {
	default:
		if err := rd.bufrd.UnreadByte(); err != nil {
			return nullValue, err
		}
		v, err = rd.readTelnetMultiBulk()
	case '+':
		v, err = rd.readSimpleString()
	case '-':
		v, err = rd.readError()
	case ':':
		v, err = rd.readInteger()
	case '$':
		v, err = rd.readBulkString()
	case '*':
		v, err = rd.readArray()
	}
	if err == io.EOF {
		return nullValue, io.ErrUnexpectedEOF
	}
	return v, err
}

// ReadMultiBulk reads the next multi bulk Value from Reader.
// A multi bulk value is a RESP array that contains one or more bulk strings.
// For more information on RESP arrays and strings please see http://redis.io/topics/protocol.
func (rd *Reader) ReadMultiBulk() (Value, error) {
	b, err := rd.bufrd.ReadByte()
	if err != nil {
		return nullValue, err
	}
	var v Value
	switch b {
	default:
		if err := rd.bufrd.UnreadByte(); err != nil {
			return nullValue, err
		}
		v, err = rd.readTelnetMultiBulk()
	case '*':
		v, err = rd.readMultiBulk()
	}
	if err == io.EOF {
		return nullValue, io.ErrUnexpectedEOF
	}
	return v, err
}

func (rd *Reader) readLine(requireCRLF bool) (string, error) {
	if requireCRLF {
		var line []byte
		for {
			bline, err := rd.bufrd.ReadBytes('\r')
			if err != nil {
				return "", err
			}
			if line == nil {
				line = bline
			} else {
				line = append(line, bline...)
			}
			b, err := rd.bufrd.ReadByte()
			if err != nil {
				return "", err
			}
			if b == '\n' {
				return string(line[:len(line)-1]), nil
			}
			if err := rd.bufrd.UnreadByte(); err != nil {
				return "", err
			}
		}
	}
	line, err := rd.bufrd.ReadBytes('\n')
	if err != nil {
		return "", err
	}
	if len(line) > 1 && line[len(line)-2] == '\r' {
		return string(line[:len(line)-2]), nil
	}
	return string(line[:len(line)-1]), nil
}

func (rd *Reader) readTelnetMultiBulk() (Value, error) {
	values := make([]Value, 0, 8)
	var bline []byte
	var quote, mustspace bool
	for {
		b, err := rd.bufrd.ReadByte()
		if err != nil {
			return nullValue, err
		}
		if b == '\n' {
			if len(bline) > 0 && bline[len(bline)-1] == '\r' {
				bline = bline[:len(bline)-1]
			}
			break
		}
		if mustspace && b != ' ' {
			return nullValue, &errProtocol{"unbalanced quotes in request"}
		}
		if b == ' ' {
			if quote {
				bline = append(bline, b)
			} else {
				values = append(values, Value{'$', string(bline)})
				bline = nil
			}
		} else if b == '"' {
			if quote {
				mustspace = true
			} else {
				if len(bline) > 0 {
					return nullValue, &errProtocol{"unbalanced quotes in request"}
				}
				quote = true
			}
		} else {
			bline = append(bline, b)
		}
	}
	if quote {
		return nullValue, &errProtocol{"unbalanced quotes in request"}
	}
	if len(bline) > 0 {
		values = append(values, Value{'$', string(bline)})
	}

	return Value{'*', values}, nil
}

func (rd *Reader) readSimpleString() (Value, error) {
	line, err := rd.readLine(true)
	if err != nil {
		return nullValue, err
	}
	return Value{'+', line}, nil
}

func (rd *Reader) readError() (Value, error) {
	line, err := rd.readLine(true)
	if err != nil {
		return nullValue, err
	}
	return Value{'-', errors.New(line)}, nil
}

func (rd *Reader) readInteger() (Value, error) {
	line, err := rd.readLine(true)
	if err != nil {
		return nullValue, err
	}
	n, err := strconv.ParseInt(line, 10, 64)
	if err != nil {
		return nullValue, &errProtocol{"invalid integer"}
	}
	return Value{':', int(n)}, nil
}

func (rd *Reader) readBulkString() (Value, error) {
	line, err := rd.readLine(true)
	if err != nil {
		return nullValue, err
	}
	n, err := strconv.ParseInt(line, 10, 64)
	if err != nil || n > 512*1024*1024 {
		return nullValue, &errProtocol{"invalid bulk length"}
	}
	if n < 0 {
		return Value{'$', nil}, nil
	}
	bline := make([]byte, int(n))
	if _, err := io.ReadFull(rd.bufrd, bline); err != nil {
		return nullValue, err
	}
	if b, err := rd.bufrd.ReadByte(); err != nil {
		return nullValue, err
	} else if b != '\r' {
		return nullValue, &errProtocol{"invalid bulk line ending"}
	}
	if b, err := rd.bufrd.ReadByte(); err != nil {
		return nullValue, err
	} else if b != '\n' {
		return nullValue, &errProtocol{"invalid bulk line ending"}
	}
	return Value{'$', string(bline)}, nil
}

func (rd *Reader) readArray() (Value, error) {
	return rd.readArrayOrMultiBulk(false)
}
func (rd *Reader) readMultiBulk() (Value, error) {
	return rd.readArrayOrMultiBulk(true)
}

func (rd *Reader) readArrayOrMultiBulk(multibulk bool) (Value, error) {
	line, err := rd.readLine(true)
	if err != nil {
		return nullValue, err
	}
	n, err := strconv.ParseInt(line, 10, 64)
	if err != nil || n > 1024*1024 {
		if multibulk {
			return nullValue, &errProtocol{"invalid multibulk length"}
		}
		return nullValue, &errProtocol{"invalid array length"}
	}
	if n < 0 {
		return Value{'*', nil}, nil
	}
	values := make([]Value, int(n))
	for i := 0; i < len(values); i++ {
		b, err := rd.bufrd.ReadByte()
		if err != nil {
			return nullValue, err
		}
		var v Value
		switch b {
		default:
			if multibulk {
				return nullValue, &errProtocol{"expected '$', got '" + string(b) + "'"}
			}
			switch b {
			default:
				return nullValue, &errProtocol{"unknown first byte"}
			case '+':
				v, err = rd.readSimpleString()
			case '-':
				v, err = rd.readError()
			case ':':
				v, err = rd.readInteger()
			case '*':
				v, err = rd.readArray()
			}
		case '$':
			v, err = rd.readBulkString()
		}
		if err != nil {
			return nullValue, err
		}
		values[i] = v
	}
	return Value{'*', values}, nil
}

func formSingleLine(s string) string {
	bs1 := []byte(s)
	for i := 0; i < len(bs1); i++ {
		switch bs1[i] {
		case '\r', '\n':
			bs2 := make([]byte, len(bs1))
			copy(bs2, bs1)
			bs2[i] = ' '
			i++
			for ; i < len(bs2); i++ {
				switch bs1[i] {
				case '\r', '\n':
					bs2[i] = ' '
				}
			}
			return string(bs2)
		}
	}
	return s
}

// SimpleStringValue returns a RESP simple string. A simple string has no new lines. The carriage return and new line characters are replaced with spaces.
func SimpleStringValue(s string) Value { return Value{'+', formSingleLine(s)} }

// BytesValue returns a RESP bulk string. A bulk string can represent any data.
func BytesValue(b []byte) Value { return Value{'$', string(b)} }

// StringValue returns a RESP bulk string. A bulk string can represent any data.
func StringValue(s string) Value { return Value{'$', s} }

// NullValue returns a RESP null bulk string.
func NullValue() Value { return Value{'$', nil} }

// ErrorValue returns a RESP error.
func ErrorValue(err error) Value { return Value{'-', err} }

// IntegerValue returns a RESP integer.
func IntegerValue(i int) Value { return Value{':', i} }

// ArrayValue returns a RESP array.
func ArrayValue(vals []Value) Value { return Value{'*', vals} }

// MultiBulkValue returns a RESP array which contains one or more bulk strings.
// For more information on RESP arrays and strings please see http://redis.io/topics/protocol.
func MultiBulkValue(commandName string, args ...interface{}) Value {
	vals := make([]Value, len(args)+1)
	vals[0] = StringValue(commandName)
	for i, arg := range args {
		switch arg := arg.(type) {
		default:
			vals[i+1] = StringValue(fmt.Sprintf("%v", arg))
		case []byte:
			vals[i+1] = StringValue(string(arg))
		case string:
			vals[i+1] = StringValue(arg)
		case nil:
			vals[i+1] = NullValue()
		}
	}
	return ArrayValue(vals)
}

// Writer is a specialized RESP Value type writer.
type Writer struct {
	wr io.Writer
}

// NewWriter returns a new Writer.
func NewWriter(wr io.Writer) *Writer {
	return &Writer{wr}
}

// WriteValue writes a RESP Value.
func (wr *Writer) WriteValue(v Value) error {
	b, err := v.MarshalRESP()
	if err != nil {
		return err
	}
	_, err = wr.wr.Write(b)
	return nil
}

// WriteSimpleString writes a RESP simple string. A simple string has no new lines. The carriage return and new line characters are replaced with spaces.
func (wr *Writer) WriteSimpleString(s string) error { return wr.WriteValue(SimpleStringValue(s)) }

// WriteBytes writes a RESP bulk string. A bulk string can represent any data.
func (wr *Writer) WriteBytes(b []byte) error { return wr.WriteValue(BytesValue(b)) }

// WriteString writes a RESP bulk string. A bulk string can represent any data.
func (wr *Writer) WriteString(s string) error { return wr.WriteValue(StringValue(s)) }

// WriteNull writes a RESP null bulk string.
func (wr *Writer) WriteNull() error { return wr.WriteValue(NullValue()) }

// WriteError writes a RESP error.
func (wr *Writer) WriteError(err error) error { return wr.WriteValue(ErrorValue(err)) }

// WriteInteger writes a RESP integer.
func (wr *Writer) WriteInteger(i int) error { return wr.WriteValue(IntegerValue(i)) }

// WriteArray writes a RESP array.
func (wr *Writer) WriteArray(vals []Value) error { return wr.WriteValue(ArrayValue(vals)) }

// WriteMultiBulk writes a RESP array which contains one or more bulk strings.
// For more information on RESP arrays and strings please see http://redis.io/topics/protocol.
func (wr *Writer) WriteMultiBulk(commandName string, args ...interface{}) error {
	return wr.WriteValue(MultiBulkValue(commandName, args...))
}
