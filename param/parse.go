package param

import (
	"encoding"
	"errors"
	"fmt"
	"reflect"
	"strconv"
	"strings"
)

// Parser represents a function that can parse a value from a slice of params.
type Parser func(value any, params []string) error

type Parsers []Parser

// Parse calls each parser until one returns no error or any error other than ErrInvalidParamType.
func (p Parsers) Parse(value any, params []string) error {
	var err error
	for i := range p {
		err = p[i](value, params)
		if !errors.Is(err, ErrInvalidParamType) {
			return err
		}
	}
	return err
}

// ErrInvalidParamType represents an error when a type cannot be parsed as a param.
var ErrInvalidParamType = errors.New("invalid param type")

func ParseInt(value any, params []string) error {
	err := ErrInvalidParamType
	param := params[0]

	switch v := value.(type) {
	case *int:
		var i int64
		i, err = strconv.ParseInt(param, 10, 0)
		*v = int(i)
	case *int8:
		var i int64
		i, err = strconv.ParseInt(param, 10, 8)
		*v = int8(i)
	case *int16:
		var i int64
		i, err = strconv.ParseInt(param, 10, 16)
		*v = int16(i)
	case *int32:
		var i int64
		i, err = strconv.ParseInt(param, 10, 32)
		*v = int32(i)
	case *int64:
		*v, err = strconv.ParseInt(param, 10, 64)
	}

	return err
}

func ParseUint(value any, params []string) error {
	err := ErrInvalidParamType
	param := params[0]

	switch v := value.(type) {
	case *uint:
		var i uint64
		i, err = strconv.ParseUint(param, 10, 0)
		*v = uint(i)
	case *uint8:
		var i uint64
		i, err = strconv.ParseUint(param, 10, 8)
		*v = uint8(i)
	case *uint16:
		var i uint64
		i, err = strconv.ParseUint(param, 10, 16)
		*v = uint16(i)
	case *uint32:
		var i uint64
		i, err = strconv.ParseUint(param, 10, 32)
		*v = uint32(i)
	case *uint64:
		*v, err = strconv.ParseUint(param, 10, 64)
	}

	return err
}

func ParseFloat(value any, params []string) error {
	err := ErrInvalidParamType
	param := params[0]

	switch v := value.(type) {
	case *float32:
		var i float64
		i, err = strconv.ParseFloat(param, 32)
		*v = float32(i)
	case *float64:
		*v, err = strconv.ParseFloat(param, 64)
	}

	return err
}

func ParseTextUnmarshaller(value any, params []string) error {
	if v, ok := value.(encoding.TextUnmarshaler); ok {
		return v.UnmarshalText([]byte(params[0]))
	}
	return ErrInvalidParamType
}

func ParseBool(value any, params []string) error {
	err := ErrInvalidParamType
	if v, ok := value.(*bool); ok {
		*v, err = strconv.ParseBool(params[0])
	}
	return err
}

func ParseString(value any, params []string) error {
	err := ErrInvalidParamType
	if v, ok := value.(*string); ok {
		*v, err = params[0], nil
	}
	return err
}

func createSlice(parser Parser, params []string, typ reflect.Type) (reflect.Value, error) {
	if len(params) == 1 {
		params = strings.Split(params[0], ",")
	}

	l := len(params)
	s := reflect.MakeSlice(typ, l, l)

	for i := range l {
		item := s.Index(i).Addr().Interface()
		if err := parser(item, []string{params[i]}); err != nil {
			return reflect.Value{}, fmt.Errorf("error parsing array item: %w", err)
		}
	}

	return s, nil
}

// NewReflectParser returns a parser that uses reflection to set the value.
func NewReflectParser(parser Parser) Parser {
	return func(value any, params []string) error {
		// value should be a pointer to a value
		v := reflect.ValueOf(value).Elem()
		typ := v.Type()

		switch typ.Kind() {
		case reflect.Array, reflect.Slice:
			s, err := createSlice(parser, params, typ)
			if err == nil {
				v.Set(s)
			}
			return err
		}

		return ErrInvalidParamType
	}
}
