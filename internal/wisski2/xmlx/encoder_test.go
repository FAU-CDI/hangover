package xmlx_test

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"io"
	"strings"
	"testing"

	"github.com/FAU-CDI/hangover/internal/wisski2/xmlx"
)

type EncoderTestObj struct {
	First  []byte `xmlxcodec:"bytes"`
	Second []byte `xmlxcodec:"bytes" xmlxtag:"test"`
}

func TestXMLEncode_single(t *testing.T) {
	t.Parallel()

	for _, tt := range []struct {
		Name string

		Obj       EncoderTestObj
		WantError bool
		WantXML   string
	}{
		{
			Name: "regular encode in order",
			Obj: EncoderTestObj{
				First:  []byte("some bytes here"),
				Second: []byte("more"),
			},

			WantXML:   "<EncoderTestObj><first>some bytes here</first><test>more</test></EncoderTestObj>",
			WantError: false,
		},
	} {
		t.Run(tt.Name, func(t *testing.T) {
			t.Parallel()

			var encoder xmlx.Encoder
			encoder.MustRegister("bytes", xmlx.EncodeBytesFunc(WriteBytes))

			// create the new decoder
			var strings strings.Builder
			e := xml.NewEncoder(&strings)

			err := encoder.Encode(&tt.Obj, e, xml.StartElement{Name: xml.Name{Local: "EncoderTestObj"}})
			gotErr := (err != nil)
			if gotErr != tt.WantError {
				t.Errorf("wantError = %v, gotError = %v", tt.WantError, err)
			}

			if gotErr {
				return
			}

			gotString := strings.String()
			if gotString != tt.WantXML {
				t.Errorf("want target.First = %q, got = %q", tt.WantXML, gotString)
			}
		})
	}
}

func WriteBytes(dest io.Writer, source *[]byte) error {
	_, err := bytes.NewReader(*source).WriteTo(dest)
	if err != nil {
		return fmt.Errorf("failed to write to reader: %w", err)
	}
	return nil
}
