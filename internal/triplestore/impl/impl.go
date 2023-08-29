package impl

// Label represents the label of individual triple members.
// A label is a uri.
type Label string

// AsDatum returns a Datum representing a label.
func (label Label) AsDatum() Datum {
	return Datum(label)
}

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
type Datum string

// DatumAsByte encodes a datum as a set of bytes.
func DatumAsByte(datum Datum) []byte {
	return []byte(datum)
}

// ByteAsDatum decodes a datum from a set of bytes.
func ByteAsDatum(datum []byte) Datum {
	return Datum(datum)
}
