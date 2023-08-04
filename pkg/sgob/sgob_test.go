package sgob

import (
	"bytes"
	"encoding/gob"
	"math/rand"
	"reflect"
	"testing"
)

func TestSGob_Long(t *testing.T) {
	const N = 1_000_000 // number of elements

	{
		source := rand.New(rand.NewSource(N))

		// generate "random" map values
		ints := make(map[int]int, N)
		for i := 0; i < N; i++ {
			ints[source.Int()] = source.Int()
		}

		assertRoundTrip(t, ints)
	}

	{
		source := rand.New(rand.NewSource(N))

		// generate a random list
		ints := make([]int, N)
		for i := range ints {
			ints[i] = source.Int()
		}

		assertRoundTrip(t, ints)
	}
}

func TestSGob(t *testing.T) {
	tests := []struct {
		name  string
		value any
	}{
		{
			name:  "slice",
			value: []string{"hello", "world"},
		},
		{
			name:  "map",
			value: map[string]string{"hello": "world"},
		},
		{
			name:  "map of map",
			value: map[string]map[string]string{"hello": {"hello": "world"}},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assertRoundTrip(t, tt.value)
		})
	}
}

func assertRoundTrip(t *testing.T, value any) {
	var buffer bytes.Buffer
	encoder := gob.NewEncoder(&buffer)

	rvalue := reflect.ValueOf(value)

	if err := EncodeValue(encoder, rvalue); err != nil {
		t.Errorf("Encode() error = %v, wantErr %v", err, nil)
	}

	dest := reflect.New(rvalue.Type())

	decoder := gob.NewDecoder(&buffer)
	if err := DecodeValue(decoder, dest); err != nil {
		t.Errorf("Decode() error = %v, wantErr %v", err, nil)
	}

	got := dest.Elem().Interface()

	if !reflect.DeepEqual(got, value) {
		t.Errorf("sgob round trip got = %v, want %v", got, value)
	}
}
