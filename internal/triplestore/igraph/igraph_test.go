package igraph

import (
	"math/rand"
	"reflect"
	"strconv"
	"testing"

	"github.com/FAU-CDI/hangover/internal/triplestore/impl"
)

// l returns a label from an int
func l(i int) impl.Label {
	return impl.Label(strconv.Itoa(i))
}

// d returns a datum from an int
func d(i int) impl.Datum {
	return impl.Datum{
		Value: strconv.Itoa(i),
	}
}

// di is the inverse of the [d] function
func di(d impl.Datum) int {
	i64, err := strconv.ParseInt(d.Value, 10, 64)
	if err != nil {
		panic("di: failed to parse")
	}
	return int(i64)
}

// graphTest implements an integration test for an IGraph with the given engine.
//
// It first constructs a graph with O(N) nodes, and forms various connections.
// It makes use of both inverses and identical relationships.
//
// It then forms a single query against this graph, ensuring that the correct result set is returned.
func graphTest(t *testing.T, engine Engine, N int) {

	var g Index
	defer g.Close()

	if err := g.Reset(engine); err != nil {
		t.Errorf("unable to reset: %s", engine)
	}
	{
		// mark some inverses
		g.MarkInverse(l(0), l(-1))
		g.MarkInverse(l(1), l(-2))

		// mark some identical labels
		// by using the negatives
		for i := 0; i < N; i++ {
			if i%2 == 0 {
				g.MarkIdentical(l(3*i+8), l(-(3*i + 8)))
			}
		}

		for i := 0; i < N; i++ {
			// add triple (3i+6, 0, 3i + 7) or the inverse
			if i%4 == 0 || i%4 == 1 {
				g.AddTriple(l(3*i+6), l(0), l(3*i+7))
			} else {
				g.AddTriple(l(3*i+7), l(-1), l(3*i+6))
			}

			// add triple (3i+7, 1, 3i + 8) or the inverse
			if i%4 == 0 || i%4 == 2 {
				g.AddTriple(l(3*i+7), l(1), l(3*i+8))
			} else {
				g.AddTriple(l(3*i+8), l(-2), l(3*i+7))
			}

			// add labels to 3i + 6 and 3i+7
			g.AddTriple(l(3*i+6), l(2), l(2))

			g.AddTriple(l(3*i+7), l(3), l(3))

			// add some data (namely the i) to 3i+8
			// (or the inverse)
			if i%4 == 0 {
				// i %4 == 0 ==> i % 2 == 0 ==> we can just use the identical label
				g.AddData(l(-(3*i + 8)), l(3), d(i))
			} else {
				g.AddData(l(3*i+8), l(3), d(i))
			}
		}

		// randomly fill 100 more elements
		source := rand.New(rand.NewSource(int64(N)))
		for i := 0; i < 100; i++ {
			g.AddTriple(l(source.Intn(N)), l(4), l(source.Intn(N)))
			g.AddTriple(l(source.Intn(N)), l(5), l(source.Intn(N)))
		}
	}
	if err := g.Finalize(); err != nil {
		t.Fatalf("Unable to finalize: %s", err)
	}

	// query for all of the paths we have just created
	query, err := g.PathsStarting(l(2), l(2))
	if err != nil {
		t.Fatalf("Unable to start paths: %s", err)
	}
	if err := query.Connected(l(0)); err != nil {
		t.Fatalf("Unable to continue paths: %s", err)
	}
	if err := query.Ending(l(3), l(3)); err != nil {
		t.Fatalf("Unable to filter ending paths: %s", err)
	}
	if err := query.Connected(l(1)); err != nil {
		t.Fatalf("Unable to continue paths: %s", err)
	}
	if err := query.Connected(l(3)); err != nil {
		t.Fatalf("Unable to continue paths: %s", err)
	}

	// check that the paths are correct
	paths := query.Paths()

	encountered := make(map[impl.Datum]struct{})
	for paths.Next() {
		path := paths.Datum()

		// extract the datum
		if !path.HasDatum {
			t.Errorf("Unable to retrieve Datum: %s", err)
		}
		encountered[path.Datum] = struct{}{}

		// find the integer!
		i := di(path.Datum)

		// determine the nodes and edges we expect
		wantNodes := []impl.Label{l(3*i + 6), l(3*i + 7), l(3*i + 8)}
		wantEdges := []impl.Label{l(0), l(1), l(3)}

		wantTriples := make([]Triple, 0, 5)
		{
			wantTriples = append(wantTriples, Triple{
				Role: Regular,

				Subject:   l(3*i + 6),
				Predicate: l(2),
				Object:    l(2),

				SSubject:   l(3*i + 6),
				SPredicate: l(2),
				SObject:    l(2),
			})

			wantTriples = append(wantTriples, Triple{
				Role: Regular,

				Subject:   l(3*i + 7),
				Predicate: l(3),
				Object:    l(3),

				SSubject:   l(3*i + 7),
				SPredicate: l(3),
				SObject:    l(3),
			})

			if i%4 == 0 || i%4 == 1 {
				wantTriples = append(wantTriples, Triple{
					Role: Regular,

					Subject:   l(3*i + 6),
					Predicate: l(0),
					Object:    l(3*i + 7),

					SSubject:   l(3*i + 6),
					SPredicate: l(0),
					SObject:    l(3*i + 7),
				})
			} else {
				wantTriples = append(wantTriples, Triple{
					Role: Inverse,

					Subject:   l(3*i + 7),
					Predicate: l(-1),
					Object:    l(3*i + 6),

					SSubject:   l(3*i + 6),
					SPredicate: l(0),
					SObject:    l(3*i + 7),
				})
			}

			if i%4 == 0 || i%4 == 2 {
				wantTriples = append(wantTriples, Triple{
					Role: Regular,

					Subject:   l(3*i + 7),
					Predicate: l(1),
					Object:    l(3*i + 8),

					SSubject:   l(3*i + 7),
					SPredicate: l(1),
					SObject:    l(3*i + 8),
				})
			} else {
				wantTriples = append(wantTriples, Triple{
					Role: Inverse,

					Subject:   l(3*i + 8),
					Predicate: l(-2),
					Object:    l(3*i + 7),

					SSubject:   l(3*i + 7),
					SPredicate: l(1),
					SObject:    l(3*i + 8),
				})
			}
		}

		if i%4 == 0 {
			wantTriples = append(wantTriples, Triple{
				Role: Data,

				Subject:   l(-(3*i + 8)),
				Predicate: l(3),

				SSubject:   l(3*i + 8),
				SPredicate: l(3),

				Datum: d(i),
			})
		} else {
			wantTriples = append(wantTriples, Triple{
				Role: Data,

				Subject:   l(3*i + 8),
				Predicate: l(3),

				SSubject:   l(3*i + 8),
				SPredicate: l(3),

				Datum: d(i),
			})
		}

		// actually extract them all
		nodes := path.Nodes
		edges := path.Edges
		triples := path.Triples

		// reset the ids, as those are an implementation detail
		// and may change in the future
		for i := range triples {
			triples[i].ID.Reset()
		}

		if !reflect.DeepEqual(nodes, wantNodes) {
			t.Errorf("nodes = %v, want = %v", nodes, wantNodes)
		}
		if !reflect.DeepEqual(edges, wantEdges) {
			t.Errorf("edges = %v, want = %v", edges, wantEdges)
		}

		if !reflect.DeepEqual(triples, wantTriples) {
			t.Errorf("triples = %v, want = %v", triples, wantTriples)
		}

	}

	counter := 0
	for i := 0; i < N; i++ {
		counter++
		_, ok := encountered[d(i)]
		if !ok {
			t.Errorf("missing index %d", i)
		}
	}
	if len(encountered) != counter {
		t.Error("too few paths encounted")
	}
}
