package xmlx

import (
	"encoding/xml"
	"fmt"
	"reflect"
	"strings"
)

//nolint:recvcheck
type Encoder struct {
	encoders   map[string]reflect.Value
	encoderTyp map[string]reflect.Type
}

// DecoderFunction is a function that decodes into an object of type T.
type EncoderFunction[T any] = func(source *T, d *xml.Encoder, start xml.StartElement) error

// MustRegister is like Register, but panic()s if something goes wrong.
func (e *Encoder) MustRegister(name string, decoder any) {
	if err := e.Register(name, decoder); err != nil {
		panic(err)
	}
}

var (
	typeEncoder = reflect.TypeFor[*xml.Encoder]()
)

// Register registers a new encoder function, overwriting any previous calls to register.
// It must be of type [EncoderFunction].
func (e *Encoder) Register(name string, encoder any) error {
	v := reflect.ValueOf(encoder)

	t := v.Type()
	if t.Kind() != reflect.Func ||
		t.NumIn() != 3 ||
		t.In(0).Kind() != reflect.Pointer ||
		t.In(1) != typeEncoder ||
		t.In(2) != typeStart ||
		t.NumOut() != 1 ||
		t.Out(0) != typeErr {
		return errNoEncoderFunc
	}

	if e.encoders == nil {
		e.encoders = make(map[string]reflect.Value)
	}
	if e.encoderTyp == nil {
		e.encoderTyp = make(map[string]reflect.Type)
	}

	// store the decoder and its' type
	e.encoders[name] = v
	e.encoderTyp[name] = t.In(0).Elem()

	return nil
}

// Decode decodes into the element target.
func (ee Encoder) Encode(source any, e *xml.Encoder, start xml.StartElement) error {
	obj := reflect.ValueOf(source)
	if obj.Kind() != reflect.Pointer || obj.Elem().Kind() != reflect.Struct {
		return errNotPointerToStruct
	}

	var (
		elem = obj.Elem()
		typ  = elem.Type()
	)

	var (
		encoder = reflect.ValueOf(e)
	)

	// encode the start token
	if err := e.EncodeToken(start); err != nil {
		return fmt.Errorf("error encoding start: %w", err)
	}

	for i := range typ.NumField() {
		field := typ.Field(i)

		// ignore private fields!
		if !field.IsExported() {
			continue
		}

		// extract the decoder
		codec, ok := field.Tag.Lookup("xmlxcodec")
		if !ok {
			continue
		}

		// ensure that it is known
		code, ok := ee.encoders[codec]
		if !ok {
			return fmt.Errorf("unknown codec %q", codec)
		}

		// and of the right type
		if field.Type != ee.encoderTyp[codec] {
			return fmt.Errorf("%q does not match type of field %q", codec, field.Name)
		}

		// check that we have a valid source field
		// or fall back to the name of the field itself
		source, ok := field.Tag.Lookup("xmlxtag")
		if !ok {
			source = field.Name
		}
		source = strings.ToLower(source)

		// do the encoding!
		start := xml.StartElement{Name: xml.Name{Local: source}}
		result := code.Call([]reflect.Value{
			elem.FieldByName(field.Name).Addr(),
			encoder,
			reflect.ValueOf(start),
		})
		if len(result) != 1 || !result[0].Type().Implements(typeErr) {
			panic("encoder didn't return exactly one error value (logic error)")
		}
		if err := result[0].Interface(); err != nil {
			return fmt.Errorf("encoder for %q: %w", field.Name, err.(error))
		}
	}

	if err := e.EncodeToken(start.End()); err != nil {
		return fmt.Errorf("error encoding end: %w", err)
	}

	// force a flush!
	if err := e.Flush(); err != nil {
		return fmt.Errorf("failed to flush: %w", err)
	}
	return nil
}
