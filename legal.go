package hangover

import _ "embed"

// cspell:words gogenlicense

//go:generate gogenlicense -m -t 0.5

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
