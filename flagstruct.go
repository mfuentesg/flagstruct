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
	ErrInvalidAnnotation = errors.New("flagstruct: could not specify 'default' and 'required' in the same annotation")
	ErrInvalidType       = errors.New("flagstruct: non-pointer passed to decode")
)

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

func Decode(v interface{}) error {
	// prevent any validation without flags
	if len(os.Args[1:]) == 0 {
		return nil
	}

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
			Decode(ss)
		}
		if !f.CanSet() {
			continue
		}
		tag := t.Field(i).Tag.Get("flag")
		if tag == "" {
			continue
		}
		flagVal, err := parse(tag)
		if err != nil {
			return err
		}
		if flagVal == "" {
			continue
		}
		decoder, custom := f.Addr().Interface().(Decoder)
		if custom {
			if err := decoder.Decode(flagVal); err != nil {
				return fmt.Errorf("flagstruct: could not decode value: %v", err)
			}
		} else if f.Kind() == reflect.Slice {
			decodeSlice(&f, flagVal)
		} else {
			if err := decodePrimitiveType(&f, flagVal); err != nil {
				return err
			}
		}
	}
	return nil
}

func parse(tag string) (string, error) {
	args := os.Args[1:]
	parts := strings.Split(tag, ",")
	flagVal := lookup(args, parts[0])

	required := false
	hasDefault := false
	defaultValue := ""

	for _, o := range parts[1:] {
		if !required {
			required = strings.HasPrefix(o, "required")
		}
		if strings.HasPrefix(o, "default=") {
			hasDefault = true
			defaultValue = o[8:]
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
	return flagVal, nil
}

func decodeSlice(f *reflect.Value, flagVal string) error {
	parts := strings.Split(flagVal, ";")
	values := parts[:0]
	for _, x := range parts {
		if x != "" {
			values = append(values, strings.TrimSpace(x))
		}
	}
	valuesCount := len(values)
	slice := reflect.MakeSlice(f.Type(), valuesCount, valuesCount)
	if valuesCount > 0 {
		for i := 0; i < valuesCount; i++ {
			e := slice.Index(i)
			return decodePrimitiveType(&e, values[i])
		}
	}
	f.Set(slice)
	return nil
}

func decodePrimitiveType(f *reflect.Value, flagVal string) error {
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
