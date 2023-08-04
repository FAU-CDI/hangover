package wisski

// URI represents a URI inside of WissKI
type URI string

const (
	SameAs    URI = "http://www.w3.org/2002/07/owl#sameAs"            // the default "SameAs" Predicate
	InverseOf URI = "http://www.w3.org/2002/07/owl#inverseOf"         // the default "InverseOf" Predicate
	Type      URI = "http://www.w3.org/1999/02/22-rdf-syntax-ns#type" // the "Type" Predicate
)
