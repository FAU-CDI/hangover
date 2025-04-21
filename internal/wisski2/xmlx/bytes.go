package xmlx

import (
	"encoding/xml"
	"errors"
	"fmt"
	"io"
)

var (
	errParseValueOpen = errors.New("no sub element allowed")
)

// DecodeTagBytes parses the contents of a tag into a value of type T.
//
// The bytes will be read from the char data content of the given xml element.
// Any other opening tags (or non-matching close tags) are an errror.
// Comments, parsing directives and processing instructions are ignored.
//
// decoder may not retain a reference to it's argument after returning.
func DecodeTagBytes[T any](dest *T, d *xml.Decoder, start xml.StartElement, decoder func(dest *T, src io.Reader) error) error {
	reader, writer := io.Pipe()

	errParse := make(chan error, 1)
	go func() {
		var e error
		defer func() {
			defer close(errParse)
			if e2 := reader.Close(); e2 != nil {
				e2 = fmt.Errorf("failed to close reader: %w", e2)
				if e == nil {
					e = e2
				} else {
					e = errors.Join(e, e2)
				}
			}

			errParse <- e
		}()
		e = decoder(dest, reader)
	}()

	errDecode := func() (e error) {
		defer func() {
			if e2 := writer.Close(); e2 != nil {
				e2 = fmt.Errorf("failed to close writer: %w", e2)
				if e == nil {
					e = e2
				} else {
					e = errors.Join(e, e2)
				}
			}
		}()

		for {
			token, err := d.Token()
			if err != nil {
				return fmt.Errorf("unexpected decode error: %w", err)
			}

			switch tt := token.(type) {
			case xml.CharData: // read some character data
				if _, err := writer.Write([]byte(tt)); err != nil {
					return fmt.Errorf("failed to write to writer: %w", err)
				}
			case xml.EndElement:
				if tt.Name != start.Name {
					// shouldn't happen because the parser validates
					// but we'll handle it gracefully either way
					return fmt.Errorf("unexpected close tag %s (expected %s)", tt.Name, start.Name)
				}
				return nil
			case xml.StartElement:
				// no other element should open within this one
				return errParseValueOpen
			}
		}
	}()

	errValue := <-errParse
	if errDecode != nil {
		return errDecode
	}
	return errValue
}

// ReadTagBytes is like DecodeTagBytes, except that the parsing function directly returns a value.
func ReadTagBytes[T any](d *xml.Decoder, start xml.StartElement, parser func(src io.Reader) (T, error)) (value T, err error) {
	err = DecodeTagBytes(&value, d, start, func(dest *T, src io.Reader) (err error) {
		*dest, err = parser(src)
		return err
	})
	if err != nil {
		var zero T
		return zero, err
	}
	return value, nil
}

// ParseMainTag is like ReadMainTag, except the parser takes a byte-slice instead of a stream.
func ParseMainTag[T any](d *xml.Decoder, start xml.StartElement, parser func(bytes []byte) (T, error)) (value T, err error) {
	return ReadTagBytes(d, start, func(src io.Reader) (T, error) {
		bytes, err := io.ReadAll(src)
		if err != nil {
			var zero T
			return zero, fmt.Errorf("failed to read from source: %w", err)
		}
		return parser(bytes)
	})
}

func TagDecoderFunction[T any](decoder func(dst *T, src io.Reader) error) DecoderFunction[T] {
	return func(dest *T, d *xml.Decoder, start xml.StartElement) error {
		return DecodeTagBytes(dest, d, start, decoder)
	}
}

var encodeBufferSize = 32 * 1024

func EncodeBytesFunc[T any](encoder func(dest io.Writer, source *T) error) EncoderFunction[T] {
	return func(source *T, xmlEncoder *xml.Encoder, start xml.StartElement) error {
		// TODO: Make this an io.Pipe!
		// and read from it bit by bit.
		reader, writer := io.Pipe()

		errEncode := make(chan error, 1)
		go func() {
			var e error

			defer func() {
				defer close(errEncode)

				if e2 := writer.Close(); e2 != nil {
					e2 = fmt.Errorf("failed to close writer: %w", e2)
					if e == nil {
						e = e2
					} else {
						e = errors.Join(e, e2)
					}
				}
				errEncode <- e
			}()

			e = encoder(writer, source)
		}()

		// encode the token
		if err := xmlEncoder.EncodeToken(start); err != nil {
			return fmt.Errorf("failed to encode start token: %w", err)
		}

		errWrite := (func() (e error) {
			defer func() {
				if e2 := reader.Close(); e2 != nil {
					e2 = fmt.Errorf("failed to close reader: %w", e2)
					if xmlEncoder == nil {
						e = e2
					} else {
						e = errors.Join(e, e2)
					}
				}
			}()

			buffer := make([]byte, encodeBufferSize)

			for {
				// read the next chunk from the bufer
				n, err := reader.Read(buffer)
				if err != nil && !errors.Is(err, io.EOF) {
					return fmt.Errorf("unexpected read error: %w", err)
				}

				// write the bytes back into the buffer
				if n >= 0 {
					if err := xmlEncoder.EncodeToken(xml.CharData(buffer[:n])); err != nil {
						return fmt.Errorf("unexpected encode token error: %w", err)
					}
				}

				if errors.Is(err, io.EOF) {
					return nil
				}
			}
		})()

		// check if an error occurred
		errValue := <-errEncode
		if errValue != nil {
			return errValue
		}
		if errWrite != nil {
			return errWrite
		}

		// encode the token
		if err := xmlEncoder.EncodeToken(start.End()); err != nil {
			return fmt.Errorf("failed to encode end token: %w", err)
		}

		return nil
	}
}
