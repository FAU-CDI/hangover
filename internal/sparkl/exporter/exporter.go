package exporter

import (
	"io"

	"github.com/FAU-CDI/drincw/pathbuilder"
	"github.com/FAU-CDI/hangover/internal/wisski"
)

// cspell:words WissKI

// Exporter handles WissKI Entities
type Exporter interface {
	io.Closer

	// Begin signals that count entities will be transmitted for the given bundle
	Begin(bundle *pathbuilder.Bundle, count int64) error

	// Add adds entities for the given bundle
	Add(bundle *pathbuilder.Bundle, entity *wisski.Entity) error

	// End signals that no more entities will be submitted for the given bundle
	End(bundle *pathbuilder.Bundle) error
}
