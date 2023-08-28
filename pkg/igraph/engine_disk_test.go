package igraph

import (
	"testing"

	"github.com/FAU-CDI/hangover/pkg/imap"
)

func TestDiskEngine(t *testing.T) {
	dir := t.TempDir()
	graphTest(t, &DiskEngine[imap.Label, imap.Datum]{
		DiskMap: imap.DiskMap[imap.Label]{
			Path: dir,
		},
		MarshalDatum: func(datum imap.Datum) ([]byte, error) {
			return imap.DatumAsByte(datum), nil
		},
		UnmarshalDatum: func(dest *imap.Datum, src []byte) error {
			*dest = imap.ByteAsDatum(src)
			return nil
		},
	}, 100_000)
}
