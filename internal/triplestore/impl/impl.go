//spellchecker:words impl
package impl

//spellchecker:words encoding json
import (
	"encoding/json"
	"fmt"
)

// Label represents the label of individual triple members.
// A label is a uri.
type Label string

// LabelAsByte encodes a label as a set of bytes.
func LabelAsByte(label Label) []byte {
	return []byte(label)
}

// ByteAsLabel returns a label from a []byte.
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
	bytes, err := json.Marshal(datum)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal datum: %w", err)
	}
	return bytes, nil
}

// ByteAsDatum decodes a datum from a set of bytes.
func ByteAsDatum(dest *Datum, src []byte) error {
	if err := json.Unmarshal(src, dest); err != nil {
		return fmt.Errorf("failed to unmarshal datum: %w", err)
	}
	return nil
}

// Source represents source information for a triple.
type Source struct {
	Graph      Label
	Identifier string
}
