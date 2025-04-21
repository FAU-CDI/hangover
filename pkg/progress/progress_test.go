//spellchecker:words progress
package progress_test

//spellchecker:words strings github hangover progress
import (
	"fmt"
	"io"
	"strings"

	"github.com/FAU-CDI/hangover/pkg/progress"
)

func ExampleReader() {
	source := strings.NewReader("hello world")
	var builder strings.Builder

	reader := &progress.Reader{
		Reader: source,

		Rewritable: progress.Rewritable{
			FlushInterval: 0,
			Writer:        &builder,
		},
	}

	_, _ = reader.Read([]byte("hello"))
	_, _ = reader.Read([]byte(" world"))

	// replace all the '\r's with '\n's for testing
	fmt.Println(strings.ReplaceAll(builder.String(), "\r", "\n"))

	// Output: Read 5 B
	// Read 11 B
}

func ExampleWriter() {
	var builder strings.Builder

	writer := &progress.Writer{
		Writer: io.Discard,

		Rewritable: progress.Rewritable{
			FlushInterval: 0,
			Writer:        &builder,
		},
	}

	_, _ = writer.Write([]byte("hello"))
	_, _ = writer.Write([]byte(" world"))

	// replace all the '\r's with '\n's for testing
	fmt.Println(strings.ReplaceAll(builder.String(), "\r", "\n"))

	// Output: Wrote 5 B
	// Wrote 11 B
}
