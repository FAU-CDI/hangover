//spellchecker:words wisski
package wisski

//spellchecker:words github hangover internal triplestore impl
import "github.com/FAU-CDI/hangover/internal/triplestore/impl"

//spellchecker:words Wiss KI

const (
	SameAs             impl.Label = "http://www.w3.org/2002/07/owl#sameAs"
	EquivalentClass    impl.Label = "http://www.w3.org/2002/07/owl#equivalentClass"
	EquivalentProperty impl.Label = "http://www.w3.org/2002/07/owl#equivalentProperty"

	DefaultSameAsProperties = SameAs + "\n" + EquivalentClass + "\n" + EquivalentProperty

	InverseOf impl.Label = "http://www.w3.org/2002/07/owl#inverseOf"         // the default "InverseOf" Predicate
	Type      impl.Label = "http://www.w3.org/1999/02/22-rdf-syntax-ns#type" // the "Type" Predicate
)
