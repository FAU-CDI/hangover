package imap

// Label is the type of labels used across the implementation.
// It must be comparable.
type Label = string

// ZeroLabel represents the zero label
const ZeroLabel Label = ""

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
type Datum = string

// DatumAsByte encodes a datum as a set of bytes.
func DatumAsByte(datum Datum) []byte {
	return []byte(datum)
}

// ByteAsDatum decodes a datum from a set of bytes.
func ByteAsDatum(datum []byte) Datum {
	return Datum(datum)
}
