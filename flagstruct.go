package flagstruct

import (
	"errors"
	"fmt"
	"os"
	"reflect"
	"strconv"
	"strings"
	"time"
)

var (
	// ErrInvalidAnnotation custom error for invalid annotations
	ErrInvalidAnnotation = errors.New("flagstruct: could not specify 'default' and 'required' in the same annotation")
	// ErrInvalidType custom error for unexpected element to decode
	ErrInvalidType = errors.New("flagstruct: non-pointer passed to decode")
)

// Decoder is the interface implemented by an object that can decode an
// environment variable string representation of itself.
type Decoder interface {
	Decode(string) error
}

func lookup(args []string, t string) string {
	for _, arg := range args {
		p := strings.Split(arg, "=")
		if len(p) < 2 {
			continue
		}
		if strings.HasSuffix(p[0], t) {
			return p[1]
		}
	}
	return ""
}

func inSlice(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

// Decode command line arguments into the provided target.
// The target must be a non-nil pointer to a struct.
// Fields in the struct must be exported, and tagged with an "flag"
// struct tag with a value containing the name of the command line argument.
//
// Default values may be provided by appending ",default=value" to the
// struct tag.
// Required values may be marked by appending ",required"
// to the struct tag.  It is an error to provide both "default" and
// "required".
func Decode(v interface{}) error {
	args := os.Args[1:]
	vl := reflect.ValueOf(v)
	if vl.Kind() != reflect.Ptr || vl.IsNil() {
		return ErrInvalidType
	}
	vl = vl.Elem()
	if vl.Kind() != reflect.Struct {
		return ErrInvalidType
	}
	t := vl.Type()
	for i := 0; i < vl.NumField(); i++ {
		ft := t.Field(i)
		if ft.PkgPath != "" {
			continue
		}
		f := vl.Field(i)
		switch f.Kind() {
		case reflect.Ptr:
			if f.Elem().Kind() != reflect.Struct {
				break
			}
			f = f.Elem()
			fallthrough
		case reflect.Struct:
			if !f.Addr().CanInterface() {
				continue
			}
			ss := f.Addr().Interface()
			_, custom := ss.(Decoder)
			if custom {
				break
			}
			if err := Decode(ss); err != nil {
				return err
			}
		}
		if !f.CanSet() {
			continue
		}
		tag := ft.Tag.Get("flag")
		if tag == "" {
			continue
		}
		flagVal, err := parse(args, tag)
		if err != nil {
			return err
		}
		if flagVal == "" {
			continue
		}
		decoder, custom := f.Addr().Interface().(Decoder)
		var decodeErr error
		if custom {
			decodeErr = decoder.Decode(flagVal)
		} else if f.Kind() == reflect.Slice {
			decodeSlice(&f, flagVal)
		} else {
			decodeErr = decodePrimitive(&f, flagVal)
		}
		if decodeErr != nil {
			return fmt.Errorf("flagstruct: could not decode value `%s` to kind `%v`: %v", flagVal, f.Kind(), decodeErr)
		}
	}
	return nil
}

func parse(args []string, tag string) (string, error) {
	parts := strings.Split(tag, ",")
	if parts[0] == "" {
		return "", errors.New("flagstruct: malformed annotation, `flag` name must be defined")
	}
	flagVal := lookup(args, parts[0])
	if len(parts) < 2 {
		return flagVal, nil
	}
	var required, hasDefault, hasAllowed bool
	var defaultValue, allowedValue string
	for _, o := range parts[1:] {
		if !required {
			required = strings.HasPrefix(o, "required")
		}
		if strings.HasPrefix(o, "default=") {
			hasDefault = true
			defaultValue = o[8:]
		}
		if strings.HasPrefix(o, "allowed=") {
			hasAllowed = true
			allowedValue = o[8:]
		}
	}
	if required && hasDefault {
		return "", ErrInvalidAnnotation
	}
	if flagVal == "" && required {
		return "", fmt.Errorf(`flagstruct: flag '%s' is missing`, parts[0])
	}
	if flagVal == "" {
		flagVal = defaultValue
	}
	if parts := strings.Split(allowedValue, ";"); flagVal != "" && hasAllowed && len(parts) != 0 {
		if !inSlice(parts, flagVal) {
			return "", fmt.Errorf("flagstruct: the provided value is not allowed, instead use %+v", parts)
		}
	}
	return flagVal, nil
}

func decodeSlice(f *reflect.Value, flagVal string) {
	var values []string
	var toReduce int
	parts := strings.Split(flagVal, ";")
	for _, x := range parts {
		if x != "" {
			values = append(values, strings.TrimSpace(x))
		}
	}
	length := len(values)
	slice := reflect.MakeSlice(f.Type(), length, length)
	if length > 0 {
		for i := 0; i < length; i++ {
			e := slice.Index(i)
			if err := decodePrimitive(&e, values[i]); err != nil {
				toReduce += 1
			}
		}
	}
	if toReduce > 0 {
		slice = slice.Slice(0, length-toReduce)
	}
	f.Set(slice)
}

func decodePrimitive(f *reflect.Value, flagVal string) error {
	switch f.Kind() {
	case reflect.Bool:
		v, err := strconv.ParseBool(flagVal)
		if err != nil {
			return err
		}
		f.SetBool(v)
	case reflect.Float32, reflect.Float64:
		bits := f.Type().Bits()
		v, err := strconv.ParseFloat(flagVal, bits)
		if err != nil {
			return err
		}
		f.SetFloat(v)
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		if t := f.Type(); t.PkgPath() == "time" && t.Name() == "Duration" {
			v, err := time.ParseDuration(flagVal)
			if err != nil {
				return err
			}
			f.SetInt(int64(v))
		} else {
			bits := f.Type().Bits()
			v, err := strconv.ParseInt(flagVal, 0, bits)
			if err != nil {
				return err
			}
			f.SetInt(v)
		}
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		bits := f.Type().Bits()
		v, err := strconv.ParseUint(flagVal, 0, bits)
		if err != nil {
			return err
		}
		f.SetUint(v)
	case reflect.String:
		f.SetString(flagVal)
	case reflect.Interface:
		f.Set(reflect.ValueOf(flagVal))
	}
	return nil
}
