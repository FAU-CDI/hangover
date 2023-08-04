// Package sgob wraps the gob package to stream encoding of lists and maps of objects.
//
// By default, the [gob] package encodes a large slice of map into a buffer and then makes a single Write call
// to the underlying buffer.
// This has the disadvantage that large objects have to be encoded entirely in memory before being written to the stream.
//
// This package works around this restriction by allowing top-level slices and maps to be encoded piece by piece, meaning
// that each object is first encoded in memory and then written to the stream.
// A disadvantage is that the corresponding decode also has to be performed using this package.
package sgob

import (
	"encoding/gob"
	"errors"
	"reflect"
)

// Encode encodes object into the given encoder.
func Encode(encoder *gob.Encoder, obj any) error {
	return EncodeValue(encoder, reflect.ValueOf(obj))
}

// Decode decodes object from this decoder.
func Decode(decoder *gob.Decoder, obj any) error {
	return DecodeValue(decoder, reflect.ValueOf(obj))
}

// EncodeValue is like Encode, but takes a relect.Value.
func EncodeValue(encoder *gob.Encoder, value reflect.Value) error {
	switch value.Type().Kind() {
	case reflect.Map:
		return encodeMap(encoder, value)
	case reflect.Slice:
		return encodeSlice(encoder, value)
	default:
		return encoder.EncodeValue(value)
	}
}

// DecodeValue is like Decode, but takes a relect.Value.
func DecodeValue(decoder *gob.Decoder, value reflect.Value) error {
	tp := value.Type()
	if tp.Kind() != reflect.Pointer {
		return errors.New("gobs: attempt to decode into a non-pointer")
	}
	switch tp.Elem().Kind() {
	case reflect.Map:
		return decodeMap(decoder, value)
	case reflect.Slice:
		return decodeSlice(decoder, value)
	default:
		return decoder.DecodeValue(value)
	}
}

func encodeMap(encoder *gob.Encoder, obj reflect.Value) error {
	// encode the object
	count := uint64(obj.Len())
	if err := encoder.Encode(count); err != nil {
		return err
	}

	// iterate through each key-value pair
	iterator := obj.MapRange()
	for iterator.Next() {
		if err := EncodeValue(encoder, iterator.Key()); err != nil {
			return err
		}
		if err := EncodeValue(encoder, iterator.Value()); err != nil {
			return err
		}
	}
	return nil
}

func decodeMap(decoder *gob.Decoder, obj reflect.Value) error {
	// decode the count
	var count uint64
	if err := decoder.Decode(&count); err != nil {
		return err
	}

	tp := obj.Type().Elem()

	// create a map key and value
	mp := reflect.MakeMapWithSize(tp, int(count))

	for i := 0; i < int(count); i++ {
		key := reflect.New(tp.Key())
		value := reflect.New(tp.Elem())

		if err := DecodeValue(decoder, key); err != nil {
			return err
		}
		if err := DecodeValue(decoder, value); err != nil {
			return err
		}
		mp.SetMapIndex(key.Elem(), value.Elem())
	}

	obj.Elem().Set(mp)
	return nil
}

func encodeSlice(encoder *gob.Encoder, obj reflect.Value) error {
	// encode the object
	count := int(obj.Len())
	if err := encoder.Encode(uint64(count)); err != nil {
		return err
	}

	// encode each value in order
	for i := 0; i < count; i++ {
		if err := EncodeValue(encoder, obj.Index(i)); err != nil {
			return err
		}
	}
	return nil
}

func decodeSlice(decoder *gob.Decoder, obj reflect.Value) error {
	// decode the count
	var count uint64
	if err := decoder.Decode(&count); err != nil {
		return err
	}

	tp := obj.Type().Elem()

	// create a map key and value
	slice := reflect.MakeSlice(tp, int(count), int(count))

	for i := 0; i < int(count); i++ {
		value := reflect.New(tp.Elem())
		if err := DecodeValue(decoder, value); err != nil {
			return err
		}
		slice.Index(i).Set(value.Elem())
	}

	obj.Elem().Set(slice)
	return nil
}
