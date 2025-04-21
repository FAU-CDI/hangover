package storages

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"sync"

	"github.com/FAU-CDI/hangover/internal/triplestore/igraph"
	"github.com/FAU-CDI/hangover/internal/triplestore/impl"
	"github.com/FAU-CDI/hangover/internal/wisski"
)

// sEntityPool is a pool of stored entities.
var sEntityPool = sync.Pool{
	New: func() any {
		return new(sEntity)
	},
}

var bufferPool = sync.Pool{
	New: func() any {
		return new(bytes.Buffer)
	},
}

// sEntity represents a stored entity that does not hold references to child entities.
type sEntity struct {
	Fields   map[string][]wisski.FieldValue
	Children map[string][]impl.Label // child entities
	URI      impl.Label
	Path     []impl.Label
	Triples  []igraph.Triple
}

// Reset resets this stored entity.
func (s *sEntity) Reset() {
	s.Path = nil
	s.Children = nil
	s.Triples = nil
	s.Fields = nil
	s.URI = ""
}

// Encode encodes this stored entity into a stream of bytes.
func (s *sEntity) Encode() ([]byte, error) {
	// take a buffer
	buffer := bufferPool.Get().(*bytes.Buffer)
	defer bufferPool.Put(buffer)

	buffer.Reset()

	// encode the entity
	err := gob.NewEncoder(buffer).Encode(s)
	if err != nil {
		return nil, fmt.Errorf("failed to encode with gob: %w", err)
	}

	// return a copy of the buffer!
	bytes := buffer.Bytes()
	data := make([]byte, len(bytes))
	copy(data, bytes)

	return data, nil
}

// Decode decodes this stored entity from a stream of bytes.
func (s *sEntity) Decode(data []byte) error {
	// take a buffer
	buffer := bufferPool.Get().(*bytes.Buffer)
	defer bufferPool.Put(buffer)

	// fill it with data
	buffer.Reset()
	buffer.Write(data)

	// and decode from it
	if err := gob.NewDecoder(buffer).Decode(s); err != nil {
		return fmt.Errorf("failed to decode Entity from gob: %w", err)
	}
	return nil
}

func init() {
	gob.Register(impl.Label(""))
	gob.Register(impl.Datum{})
	gob.Register(igraph.Role(0))

	gob.Register(wisski.FieldValue{})
	gob.Register(igraph.Triple{})
	gob.Register(sEntity{})
}
