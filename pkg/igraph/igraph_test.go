package igraph

import (
	"math/rand"
	"reflect"
	"strconv"
	"testing"

	"github.com/FAU-CDI/hangover/pkg/imap"
)

func l(i int) imap.Label {
	return strconv.Itoa(i)
}

func d(i int) imap.Datum {
	return strconv.Itoa(i)
}

// graphTest implements an integration test for an IGraph with the given engine.
//
// It first constructs a graph with O(N) nodes, and forms various connections.
// It makes use of both inverses and identical relationships.
//
// It then forms a single query against this graph, ensuring that the correct result set is returned.
func graphTest(t *testing.T, engine Engine[imap.Label, imap.Datum], N int) {

	var g IGraph[imap.Label, imap.Datum]
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

	encountered := make(map[imap.Datum]struct{})
	for paths.Next() {
		path := paths.Datum()

		// extract the datum
		datum, ok, err := path.Datum()
		if err != nil || !ok {
			t.Errorf("Unable to retrieve Datum: %s", err)
		}
		encountered[datum] = struct{}{}

		// find the integer!
		i64, err := strconv.ParseInt(datum, 10, 64)
		if err != nil {
			t.Errorf("Unable to parse datum: %s", err)
		}
		i := int(i64)

		// determine the nodes and edges we expect
		wantNodes := []imap.Label{l(3*i + 6), l(3*i + 7), l(3*i + 8)}
		wantEdges := []imap.Label{l(0), l(1), l(3)}

		wantTriples := make([]Triple[imap.Label, imap.Datum], 0, 5)
		{
			wantTriples = append(wantTriples, Triple[imap.Label, imap.Datum]{
				Role: Regular,

				Subject:   l(3*i + 6),
				Predicate: l(2),
				Object:    l(2),

				SSubject:   l(3*i + 6),
				SPredicate: l(2),
				SObject:    l(2),
			})

			wantTriples = append(wantTriples, Triple[imap.Label, imap.Datum]{
				Role: Regular,

				Subject:   l(3*i + 7),
				Predicate: l(3),
				Object:    l(3),

				SSubject:   l(3*i + 7),
				SPredicate: l(3),
				SObject:    l(3),
			})

			if i%4 == 0 || i%4 == 1 {
				wantTriples = append(wantTriples, Triple[imap.Label, imap.Datum]{
					Role: Regular,

					Subject:   l(3*i + 6),
					Predicate: l(0),
					Object:    l(3*i + 7),

					SSubject:   l(3*i + 6),
					SPredicate: l(0),
					SObject:    l(3*i + 7),
				})
			} else {
				wantTriples = append(wantTriples, Triple[imap.Label, imap.Datum]{
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
				wantTriples = append(wantTriples, Triple[imap.Label, imap.Datum]{
					Role: Regular,

					Subject:   l(3*i + 7),
					Predicate: l(1),
					Object:    l(3*i + 8),

					SSubject:   l(3*i + 7),
					SPredicate: l(1),
					SObject:    l(3*i + 8),
				})
			} else {
				wantTriples = append(wantTriples, Triple[imap.Label, imap.Datum]{
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
			wantTriples = append(wantTriples, Triple[imap.Label, imap.Datum]{
				Role: Data,

				Subject:   l(-(3*i + 8)),
				Predicate: l(3),

				SSubject:   l(3*i + 8),
				SPredicate: l(3),

				Datum: d(i),
			})
		} else {
			wantTriples = append(wantTriples, Triple[imap.Label, imap.Datum]{
				Role: Data,

				Subject:   l(3*i + 8),
				Predicate: l(3),

				SSubject:   l(3*i + 8),
				SPredicate: l(3),

				Datum: d(i),
			})
		}

		// actually extract them all
		nodes, err := path.Nodes()
		if err != nil {
			t.Errorf("Unable to retrieve Nodes: %s", err)
		}
		edges, err := path.Edges()
		if err != nil {
			t.Errorf("Unable to retrieve Edges: %s", err)
		}
		triples, err := path.Triples()
		if err != nil {
			t.Errorf("Unable to retrieve Triples: %s", err)
		}

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
		_, ok := encountered[l(i)]
		if !ok {
			t.Errorf("missing index %d", i)
		}
	}
	if len(encountered) != counter {
		t.Error("too few paths encounted")
	}
}
