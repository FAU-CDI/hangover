// Package wisski2 implements recovering entity data based on a graph db
package wisski2

import (
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"iter"
	"strconv"
	"strings"

	"github.com/FAU-CDI/hangover/internal/wisski2/xmlx"
)

type Path struct {
	ID                   string   `xmlxcoder:"str" xmlxtag:"id"`
	Weight               int      `xmlxcoder:"int" xmlxtag:"weight"`
	Enabled              bool     `xmlxcoder:"bool" xmlxtag:"enabled"`
	GroupID              string   `xmlxcoder:"str0" xmlxtag:"group_id"`
	Bundle               string   `xmlxcoder:"str" xmlxtag:"bundle"`
	Field                string   `xmlxcoder:"str" xmlxtag:"field"`
	FieldType            string   `xmlxcoder:"str" xmlxtag:"fieldtype"`
	FieldTypeInformative string   `xmlxcoder:"str" xmlxtag:"field_type_informative"`
	DisplayWidget        string   `xmlxcoder:"str" xmlxtag:"displaywidget"`
	FormatterWidget      string   `xmlxcoder:"str" xmlxtag:"formatterwidget"`
	Cardinality          int      `xmlxcoder:"int" xmlxtag:"cardinality"`
	PathArray            []string `xmlxcoder:"path" xmlxtag:"path_array"`
	DatatypeProperty     string   `xmlxcoder:"strEmpty" xmlxtag:"datatype_property"`
	ShortName            string   `xmlxcoder:"str" xmlxtag:"short_name"`
	Disambiguation       int      `xmlxcoder:"int" xmlxtag:"disamb"`
	Description          string   `xmlxcoder:"str" xmlxtag:"description"`
	UUID                 string   `xmlxcoder:"str" xmlxtag:"uuid"`
	IsGroup              bool     `xmlxcoder:"bool" xmlxtag:"is_group"`
	Name                 string   `xmlxcoder:"str" xmlxtag:"name"`
}

// Gets the number of concepts in this path.
func (path Path) ConceptCount() int {
	// Note: This should really be Math.ceil(len(...) / 2)
	// But that would require conversion to floats, and this is simpler.
	return (len(path.PathArray) + 1) / 2
}

// Returns an iterator over all URIs referenced by this path including
// concepts, (object) properties, and the datatype property (if any).
func (path Path) URIs() iter.Seq[string] {
	return func(yield func(string) bool) {
		for _, uri := range path.PathArray {
			if !yield(uri) {
				return
			}
		}
		if path.DatatypeProperty != "" {
			yield(path.DatatypeProperty)
		}
	}
}

// Returns FieldTypeInformative or FieldType, whichever is set.
func (path Path) InformativeFieldType() (tp string, ok bool) {
	if path.FieldTypeInformative != "" {
		return path.FieldTypeInformative, true
	}
	if path.FieldType == "" {
		return "", false
	}
	return path.FieldType, true
}

// Returns the index of the disambiguated concept in the pathArray, or null
func (path Path) DisambiguationIndex() (index int, ok bool) {
	index = 2*path.Disambiguation - 2
	if index < 0 || index >= len(path.PathArray) {
		return -1, false
	}
	return index, true
}

// The concept  disambiguated by this pathbuilder, if any
func (path Path) DisambiguatedConcept() (ceoncept string, ok bool) {
	index := 2*path.Disambiguation - 2
	if index < 0 || index >= len(path.PathArray) {
		return "", false
	}
	return path.PathArray[index], true
}

var (
	pathDecoder xmlx.Decoder
	pathEncoder xmlx.Encoder
)

func init() {
	pathDecoder.IgnoreRepeats = false
	pathDecoder.IgnoreUnknowns = true

	pathDecoder.MustRegister("str", xmlx.TagDecoderFunction(parseString))
	pathDecoder.MustRegister("str0", xmlx.TagDecoderFunction(parseString0))
	pathDecoder.MustRegister("strEmpty", xmlx.TagDecoderFunction(parseStringEmpty))
	pathDecoder.MustRegister("bool", xmlx.TagDecoderFunction(parseBool))
	pathDecoder.MustRegister("int", xmlx.TagDecoderFunction(parseInt))

	pathDecoder.MustRegister("path", parsePathArray)
}

func (path *Path) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
	if path == nil {
		panic("cannot unmarshal into nil path")
	}

	return pathDecoder.Decode(path, d, start)
}

func parseString(dst *string, src io.Reader) error {
	str, err := io.ReadAll(src)
	if err != nil {
		return err
	}
	*dst = string(str)
	return nil
}

func parseString0(dst *string, src io.Reader) error {
	if err := parseString(dst, src); err != nil {
		return err
	}
	if *dst == "0" {
		*dst = ""
	}
	return nil
}

func parseStringEmpty(dst *string, src io.Reader) error {
	if err := parseString(dst, src); err != nil {
		return err
	}
	if *dst == "empty" {
		*dst = ""
	}
	return nil
}

func parseInt(dst *int, src io.Reader) error {
	value, err := io.ReadAll(src)
	if err != nil {
		return err
	}
	str := strings.TrimSpace(string(value))
	if str == "" {
		*dst = 0
		return nil
	}

	i, err := strconv.Atoi(str)
	if err != nil {
		return err
	}
	*dst = i
	return nil
}

func parseBool(dst *bool, src io.Reader) error {
	var value int
	if err := parseInt(&value, src); err != nil {
		return err
	}
	*dst = value != 0
	return nil
}

var errUnexpectedElement = errors.New("unexpected element, must be alternating <x> or <y>")

func parsePathArray(dest *[]string, d *xml.Decoder, start xml.StartElement) error {
	*dest = nil

	next := "x"
	for {
		token, err := d.Token()
		if err != nil {
			return err
		}

		switch tt := token.(type) {
		case xml.EndElement:
			if tt.Name != start.Name {
				// shouldn't happen because the parser validates
				// but we'll handle it gracefully either way
				return fmt.Errorf("unexpected close tag %s (expected %s)", tt.Name, start.Name)
			}
			return nil
		case xml.StartElement:
			if tt.Name.Space != "" || len(tt.Name.Local) == 0 {
				return errUnexpectedElement
			}

			// check that we have an <x> or a <y>
			name := strings.ToLower(tt.Name.Local[:1])
			if name != next {
				return errUnexpectedElement
			}

			// pick the correct next element
			if next == "x" {
				next = "y"
			} else {
				next = "x"
			}

			// write the element into dest
			*dest = append(*dest, "")
			if err := xmlx.DecodeTagBytes(&(*dest)[len(*dest)-1], d, tt, parseString); err != nil {
				return err
			}

		}
	}
}
