package xmlx_test

import (
	"encoding/xml"
	"fmt"
	"io"
	"strings"

	"github.com/FAU-CDI/hangover/internal/wisski2/xmlx"
)

func ExampleReadTagBytes() {
	// create the new decoder
	d := xml.NewDecoder(strings.NewReader(`<data>
<!-- some ignored comment -->
Some content here
<!-- another ignored comment -->
More content here
</data>
`))

	// read the start tag
	start, err := d.Token()
	if err != nil {
		panic(err)
	}

	value, err := xmlx.ReadTagBytes(d, start.(xml.StartElement), io.ReadAll)
	if err != nil {
		panic(err)
	}

	// Output: Some content here
	//
	// More content here
	fmt.Println(string(value))
}
