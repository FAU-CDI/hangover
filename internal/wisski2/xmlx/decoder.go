package xmlx

import (
	"encoding/xml"
	"errors"
	"fmt"
	"reflect"
	"slices"
	"strings"
)

// Decoder decodes multiple different fields and writes them into a struct tag
type Decoder struct {
	decoders   map[string]reflect.Value
	decoderTyp map[string]reflect.Type

	IgnoreRepeats  bool // If true, ignore a repeated occurrence of an already seen field.
	IgnoreUnknowns bool // If true, ignore entirely unknown fields completely.
}

var (
	errNoDecoderFunc = errors.New("Register: Not a Decoder Function")
	errNoEncoderFunc = errors.New("Register: Not a Encoder Function")
)

// DecoderFunction is a function that decodes into an object of type T
type DecoderFunction[T any] = func(dest *T, d *xml.Decoder, start xml.StartElement) error

// type of arguments for decoder function
var (
	typeErr     = reflect.TypeFor[error]()
	typeDecoder = reflect.TypeFor[*xml.Decoder]()
	typeStart   = reflect.TypeFor[xml.StartElement]()
)

// MustRegister is like Register, but panic()s if something goes wrong
func (ed *Decoder) MustRegister(name string, decoder any) {
	if err := ed.Register(name, decoder); err != nil {
		panic(err)
	}
}

// Register registers a new decoder function, overwriting any previous calls to register.
// It must be of type [DecoderFunction].
func (ed *Decoder) Register(name string, decoder any) error {
	v := reflect.ValueOf(decoder)

	t := v.Type()
	if !(t.Kind() == reflect.Func &&
		t.NumIn() == 3 &&
		t.In(0).Kind() == reflect.Pointer &&
		t.In(1) == typeDecoder &&
		t.In(2) == typeStart &&
		t.NumOut() == 1 &&
		t.Out(0) == typeErr) {
		return errNoDecoderFunc
	}

	if ed.decoders == nil {
		ed.decoders = make(map[string]reflect.Value)
	}
	if ed.decoderTyp == nil {
		ed.decoderTyp = make(map[string]reflect.Type)
	}

	// store the decoder and its' type
	ed.decoders[name] = v
	ed.decoderTyp[name] = t.In(0).Elem()

	return nil
}

var (
	errNotPointerToStruct  = errors.New("target is not a pointer to a struct")
	errInvalidCloseElement = errors.New("invalid close element")
	errInvalidStartElement = errors.New("invalid start element")
)

// Decode decodes into the element target.
func (ed *Decoder) Decode(target any, d *xml.Decoder, start xml.StartElement) error {
	obj := reflect.ValueOf(target)
	if obj.Kind() != reflect.Pointer || obj.Elem().Kind() != reflect.Struct {
		return errNotPointerToStruct
	}

	var (
		elem = obj.Elem()
		typ  = elem.Type()
	)

	var (
		dVal = reflect.ValueOf(d)
	)

	// prepare the destination for all of the fields
	var (
		total    = elem.NumField()                       // total number of functions
		decoders = make(map[string]reflect.Value, total) // decoder functions to use for each field
		fields   = make(map[string]reflect.Value, total) // target to write element into
	)

	// to simplify work for the GC
	// ensure that none of the reflect values leak!
	defer clear(fields)
	defer clear(decoders)

	{
		for i := range total {
			field := typ.Field(i)

			// ignore private fields!
			if !field.IsExported() {
				continue
			}

			// extract the codec
			codec, ok := field.Tag.Lookup("xmlxcoder")
			if !ok {
				continue
			}

			// ensure that it is known
			if _, ok := ed.decoders[codec]; !ok {
				return fmt.Errorf("unknown codec %q", codec)
			}

			// and of the right type
			if field.Type != ed.decoderTyp[codec] {
				return fmt.Errorf("%q does not match type of field %q", codec, field.Name)
			}

			// check that we have a valid source field
			// or fall back to the name of the field itself
			source, ok := field.Tag.Lookup("xmlxtag")
			if !ok {
				source = field.Name
			}
			source = strings.ToLower(source)

			// check that we have a new source!
			if _, ok := fields[source]; ok {
				return fmt.Errorf("repeated codec source %q", source)
			}

			// store the decoder and its destination
			decoders[source] = ed.decoders[codec]
			fields[source] = elem.FieldByName(field.Name).Addr()
		}
	}

loop:
	for {
		token, err := d.Token()
		if err != nil {
			return fmt.Errorf("unexpected decode error: %w", err)
		}

		switch tt := token.(type) {
		case xml.EndElement:
			if tt.Name != start.Name {
				// shouldn't happen because the parser validates
				// but we'll handle it gracefully either way
				return errInvalidCloseElement
			}
			break loop
		case xml.StartElement:
			if tt.Name.Space != "" {
				return errInvalidStartElement
			}

			// normalize the tag
			local := strings.ToLower(tt.Name.Local)

			// check if we know the field
			field, fieldOK := fields[local]
			decoder, decoderOK := decoders[local]
			if !fieldOK {
				// if we still have a decoder, it is a repeated field!
				if decoderOK {
					if !ed.IgnoreRepeats {
						return fmt.Errorf("repeated tag %q", tt.Name.Local)

					}
					// we have an unknown field
				} else {
					if !ed.IgnoreUnknowns {
						return fmt.Errorf("unknown tag %q", tt.Name.Local)
					}
				}

				// skip the tag and move to the next one!
				if err := d.Skip(); err != nil {
					return fmt.Errorf("unexpected decode error: %w", err)
				}
				continue loop
			}

			if !decoderOK {
				panic("ElementDecoder: have field but no decoder (logic error)")
			}

			result := decoder.Call([]reflect.Value{field, dVal, reflect.ValueOf(tt)})
			if len(result) != 1 || !result[0].Type().Implements(typeErr) {
				panic("decoder didn't return exactly one error value (logic error)")
			}
			err := result[0].Interface()
			if err != nil {
				return fmt.Errorf("decoder for %q: %w", tt.Name.Local, err.(error))
			}
			delete(fields, local)

		}
	}

	// ensure that all the required fields were set, if not bail out with an error!
	if len(fields) != 0 {
		keys := make([]string, 0, len(fields))
		for key := range fields {
			keys = append(keys, key)
		}
		slices.Sort(keys)
		return fmt.Errorf("ElementDecoder.Decode: Required element %q missing", keys[0])
	}

	return nil
}
