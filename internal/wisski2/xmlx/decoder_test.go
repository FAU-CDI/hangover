package xmlx_test

import (
	"encoding/xml"
	"fmt"
	"io"
	"strings"
	"testing"

	"github.com/FAU-CDI/hangover/internal/wisski2/xmlx"
)

type DecoderTestObj struct {
	First  []byte `xmlxcoder:"bytes"`
	Second []byte `xmlxcoder:"bytes" xmlxtag:"test"`
}

func TestXMLDecoder_single(t *testing.T) {
	t.Parallel()

	for _, tt := range []struct {
		Name           string
		XML            string
		IgnoreRepeats  bool
		IgnoreUnknowns bool

		WantFirst       string
		WantSecond      string
		WantDecodeError bool
	}{
		{
			Name:           "regular decode in order",
			XML:            "<thing><First>some bytes here</First><test>more</test></thing>",
			IgnoreRepeats:  false,
			IgnoreUnknowns: false,

			WantDecodeError: false,
			WantFirst:       "some bytes here",
			WantSecond:      "more",
		},
		{
			Name:           "regular decode out of  order",
			XML:            "<thing><test>more</test><First>some bytes here</First></thing>",
			IgnoreRepeats:  false,
			IgnoreUnknowns: false,

			WantDecodeError: false,
			WantFirst:       "some bytes here",
			WantSecond:      "more",
		},
		{
			Name:           "decode in other case",
			XML:            "<thing><FiRsT>some bytes here</FiRsT><test>more</test></thing>",
			IgnoreRepeats:  false,
			IgnoreUnknowns: false,

			WantDecodeError: false,
			WantFirst:       "some bytes here",
			WantSecond:      "more",
		},
		{
			Name:           "missing field",
			XML:            "<thing><test>more</test></thing>",
			IgnoreRepeats:  false,
			IgnoreUnknowns: false,

			WantDecodeError: true,
			WantFirst:       "",
			WantSecond:      "",
		},
		{
			Name:           "repeated field not ignored",
			XML:            "<thing><test>more</test><first>first first</first><first>second first</first></thing>",
			IgnoreRepeats:  false,
			IgnoreUnknowns: false,

			WantDecodeError: true,
		},
		{
			Name:           "repeated field ignored",
			XML:            "<thing><test>more</test><first>first first</first><first>second first</first></thing>",
			IgnoreRepeats:  true,
			IgnoreUnknowns: false,

			WantDecodeError: false,
			WantFirst:       "first first",
			WantSecond:      "more",
		},
		{
			Name:           "unknown field not ignored",
			XML:            "<thing><test>more</test><first>first</first><unknown>unknown</unknown></thing>",
			IgnoreRepeats:  false,
			IgnoreUnknowns: false,

			WantDecodeError: true,
		},
		{
			Name:           "unknown field not ignored",
			XML:            "<thing><test>more</test><first>first</first><unknown>unknown</unknown></thing>",
			IgnoreRepeats:  false,
			IgnoreUnknowns: true,

			WantDecodeError: false,
			WantFirst:       "first",
			WantSecond:      "more",
		},
	} {
		t.Run(tt.Name, func(t *testing.T) {
			t.Parallel()

			var decoder xmlx.Decoder
			decoder.IgnoreRepeats = tt.IgnoreRepeats
			decoder.IgnoreUnknowns = tt.IgnoreUnknowns

			// register the decoder function
			decoder.MustRegister("bytes", ReadBytes)

			// create the new decoder
			d := xml.NewDecoder(strings.NewReader(tt.XML))

			// read the start tag
			start, err := d.Token()
			if err != nil {
				t.Error(err)
			}

			// do the decode
			var target DecoderTestObj
			err = decoder.Decode(&target, d, start.(xml.StartElement))
			gotErr := (err != nil)
			if gotErr != tt.WantDecodeError {
				t.Errorf("wantError = %v, gotError = %v", tt.WantDecodeError, err)
			}

			if gotErr {
				return
			}

			if string(target.First) != tt.WantFirst {
				t.Errorf("want target.First = %q, got = %q", tt.WantFirst, string(target.First))
			}

			if string(target.Second) != tt.WantSecond {
				t.Errorf("want target.Second = %q, got = %q", tt.WantSecond, string(target.Second))
			}
		})
	}
}

func ReadBytes(dest *[]byte, d *xml.Decoder, start xml.StartElement) error {
	if err := xmlx.DecodeTagBytes(dest, d, start, func(dest *[]byte, src io.Reader) (err error) {
		*dest, err = io.ReadAll(src)
		if err != nil {
			return fmt.Errorf("failed to read from source: %w", err)
		}
		return nil
	}); err != nil {
		return fmt.Errorf("failed to decode tag bytes: %w", err)
	}
	return nil
}
