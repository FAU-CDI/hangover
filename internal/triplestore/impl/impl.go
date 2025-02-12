package impl

import "encoding/json"

// Label represents the label of individual triple members.
// A label is a uri.
type Label string

// LabelAsByte encodes a label as a set of bytes.
func LabelAsByte(label Label) []byte {
	return []byte(label)
}

// ByteAsLabel returns a label from a []byte
func ByteAsLabel(label []byte) Label {
	return Label(label)
}

// Datum is the type of data used across the implementation.
// It may or may not be comparable.
type Datum struct {
	Value    string
	Language string
}

// DatumAsByte encodes a datum as a set of bytes.
func DatumAsByte(datum Datum) ([]byte, error) {
	return json.Marshal(&datum)
}

// ByteAsDatum decodes a datum from a set of bytes.
func ByteAsDatum(dest *Datum, src []byte) error {
	return json.Unmarshal(src, dest)
}

// Source represents source information for a triple
type Source struct {
	Graph      Label
	Identifier string
}
