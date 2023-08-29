package wisski

import "github.com/FAU-CDI/hangover/internal/imap"

// cspell:words WissKI

const (
	SameAs    imap.Label = "http://www.w3.org/2002/07/owl#sameAs"            // the default "SameAs" Predicate
	InverseOf imap.Label = "http://www.w3.org/2002/07/owl#inverseOf"         // the default "InverseOf" Predicate
	Type      imap.Label = "http://www.w3.org/1999/02/22-rdf-syntax-ns#type" // the "Type" Predicate
)
