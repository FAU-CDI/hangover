//spellchecker:words hangover
package hangover

//spellchecker:words embed
import _ "embed"

//spellchecker:words gogenlicense

//go:generate go tool gogenlicense -m -skip-no-license

//go:embed LICENSE
var License string

// LegalText returns legal text to be included in human-readable output using hangover.
func LegalText() string {
	return `
================================================================================
Hangover - A WissKI Viewer
================================================================================
` + License + "\n" + ""
}
