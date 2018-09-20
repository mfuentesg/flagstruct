package flagstruct

import (
	"fmt"
	"reflect"
	"testing"
	"time"
)

type Database struct {
	Host     string
	Port     int    `flag:"db-port,default=5672"`
	User     string `flag:"db-user,required"`
	Password string `flag:"db-password"`
}

type Test struct {
	DB      *Database
	ignored string        `flag:"ignored,default=bla"`
	Timeout time.Duration `flag:"timeout,default=2s"`
}

func TestLookup(t *testing.T) {
	type test struct {
		args     []string
		expected string
		arg      string
	}

	tests := []*test{
		{
			args:     []string{"-a a", "-timeout=2s"},
			arg:      "a",
			expected: "",
		},
		{
			args:     []string{"-timeout="},
			arg:      "timeout",
			expected: "",
		},
		{
			args:     []string{" =2"},
			arg:      "timeout",
			expected: "",
		},
		{
			args:     []string{"-_=2"},
			arg:      "_",
			expected: "2",
		},
		{
			args:     []string{"-host=127.0.0.1"},
			arg:      "host",
			expected: "127.0.0.1",
		},
		{
			args:     []string{"--host=127.0.0.1"},
			arg:      "-host",
			expected: "127.0.0.1",
		},
	}

	for _, ts := range tests {
		if result := lookup(ts.args, ts.arg); result != ts.expected {
			t.Errorf("wrong result expected %s got %s", ts.expected, result)
		}
	}
}

func TestParse(t *testing.T) {
	type test struct {
		args     []string
		expected string
		tag      string
	}

	tests := []*test{
		{
			args:     []string{"-timeout=1s", "-host=127.0.0.1"},
			tag:      "time",
			expected: "",
		},
		{
			args:     []string{"-timeout=1s", "-host=127.0.0.1"},
			tag:      ",",
			expected: "",
		},
		{
			args:     []string{"-timeout=1s", "-host=127.0.0.1"},
			tag:      "timeout,required,default=",
			expected: "",
		},
		{
			args:     []string{"-timeout=1s", "-host=127.0.0.1"},
			tag:      "password",
			expected: "",
		},
		{
			args:     []string{"-timeout=1s", "-host=127.0.0.1"},
			tag:      "password,required",
			expected: "",
		},
		{
			args:     []string{"-timeout=1s", "-host=127.0.0.1"},
			tag:      "password,default=secret",
			expected: "secret",
		},
		{
			args:     []string{"-timeout=1s", "-host=127.0.0.1", "-password=secret"},
			tag:      "password",
			expected: "secret",
		},
	}

	for i, ts := range tests {
		if result, _ := parse(ts.args, ts.tag); result != ts.expected {
			t.Errorf("%d. wrong result expected %s got %s", i, ts.expected, result)
		}
	}
}

func TestDecodeSlice(t *testing.T) {
	type test struct {
		value    string
		expected []int
	}
	type Struct struct {
		Slice []int
	}

	tests := []*test{
		{value: "a", expected: []int{}},
		{value: "", expected: []int{}},
		{value: ";a;", expected: []int{}},
		{value: ";;a", expected: []int{}},
		{value: "1;a", expected: []int{1}},
		{value: "1;", expected: []int{1}},
		{value: "1;2", expected: []int{1, 2}},
		{value: "1;;3", expected: []int{1, 3}},
	}
	var s Struct
	f := reflect.ValueOf(&s).Elem().Field(0)
	for i, ts := range tests {
		decodeSlice(&f, ts.value)
		if len(ts.expected) >= 0 && !reflect.DeepEqual(ts.expected, s.Slice) {
			t.Errorf("%d. wrong slice expected %v got %v", i, ts.expected, s.Slice)
		}
	}
}

func TestDecodePrimitive(t *testing.T) {
	type fields struct {
		Bool      bool
		Float32   float32
		Int       int
		Duration  time.Duration
		Uint      uint
		String    string
		Interface interface{}
	}

	type test struct {
		value    string
		field    int
		expected string
	}

	tests := []*test{
		{field: 0, value: "asdf", expected: "false"},
		{field: 0, value: "false", expected: "false"},
		{field: 0, value: "true", expected: "true"},
		{field: 1, value: "false", expected: "0"},
		{field: 1, value: "1.5", expected: "1.5"},
		{field: 2, value: "a", expected: "0"},
		{field: 2, value: "1", expected: "1"},
		{field: 3, value: "10", expected: "0s"},
		{field: 3, value: "a", expected: "0s"},
		{field: 3, value: "1m", expected: "1m0s"},
		{field: 4, value: "-", expected: "0"},
		{field: 4, value: "5", expected: "5"},
		{field: 5, value: "", expected: ""},
		{field: 5, value: "asdf", expected: "asdf"},
		{field: 6, value: "nil", expected: "nil"},
		{field: 6, value: "empty", expected: "empty"},
	}

	var s fields
	for i, ts := range tests {
		f := reflect.ValueOf(&s).Elem().Field(ts.field)
		decodePrimitive(&f, ts.value)
		if ts.expected != fmt.Sprintf("%v", f) {
			t.Errorf("case #%d: expected %v got %v", i, ts.expected, f)
		}
	}

}

// func TestDecode(t *testing.T) {
// 	os.Args = []string{"./example"}
// 	var ts Test
// 	if err := Decode(nil); err == nil {
// 		t.Error("expected error for invalid argument type")
// 	}
// 	if err := Decode(ts); err == nil {
// 		t.Error("expected error for invalid argument type")
// 	}
// 	if err := Decode(&ts); err != nil {
// 		t.Error("expected error for command without arguments")
// 	}
// 	var str string
// 	if err := Decode(&str); err == nil {
// 		t.Error("expected error for invalid argument type")
// 	}
// 	os.Args = []string{"./example", "-field=ignored"}
// 	type ignored struct {
// 		unexported string `flag:"wrong,default=str"`
// 		field      string `flag:"field,default=field"`
// 	}
// 	var i ignored
// 	if err := Decode(&i); err != nil {
// 		t.Errorf("wrong result unexpected error, %v", err)
// 	}
// 	if i.field != "" {
// 		t.Errorf("wrong decode expected '' for unexported fields but got '%s'", i.field)
// 	}
// 	type pointer struct {
// 		Str *string `flag:"str"`
// 	}
// 	os.Args = []string{"./example", "-str=value"}
// 	var p pointer
// 	if Decode(&p); p.Str != nil {
// 		t.Errorf("wrong result for unsupported field type, expected '' got %+v", p.Str)
// 	}
//
// 	type test struct {
// 		Int   int   `flag:"int,default=hola"`
// 		Slice []int `flag:"slice,default=1;2;a"`
// 	}
// 	var t3 test
// 	if err := Decode(&t3); err == nil {
// 		t.Errorf("expect an error for invalid conversion type %+v", err)
// 	}
// 	os.Args = []string{"./example", "-int=5"}
// 	var t4 test
// 	if err := Decode(&t4); err == nil {
// 		t.Errorf("expect an error for invalid conversion type %+v", err)
// 	}
// }
