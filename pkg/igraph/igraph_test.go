package igraph

import (
	"math/rand"
	"reflect"
	"strconv"
	"testing"
)

// graphTest implements an integration test for an IGraph with the given engine.
//
// It first constructs a graph with O(N) nodes, and forms various connections.
// It makes use of both inverses and identical relationships.
//
// It then forms a single query against this graph, ensuring that the correct result set is returned.
func graphTest(t *testing.T, engine Engine[int, string], N int) {

	var g IGraph[int, string]
	defer g.Close()

	g.Reset(engine)
	{
		// mark some inverses
		g.MarkInverse(0, -1)
		g.MarkInverse(1, -2)

		// mark some identical labels
		// by using the negatives
		for i := 0; i < N; i++ {
			if i%2 == 0 {
				g.MarkIdentical(3*i+8, -(3*i + 8))
			}
		}

		for i := 0; i < N; i++ {
			// add triple (3i+6, 0, 3i + 7) or the inverse
			if i%4 == 0 || i%4 == 1 {
				g.AddTriple(3*i+6, 0, 3*i+7)
			} else {
				g.AddTriple(3*i+7, -1, 3*i+6)
			}

			// add triple (3i+7, 1, 3i + 8) or the inverse
			if i%4 == 0 || i%4 == 2 {
				g.AddTriple(3*i+7, 1, 3*i+8)
			} else {
				g.AddTriple(3*i+8, -2, 3*i+7)
			}

			// add labels to 3i + 6 and 3i+7
			g.AddTriple(3*i+6, 2, 2)

			g.AddTriple(3*i+7, 3, 3)

			// add some data (namely the i) to 3i+8
			// (or the inverse)
			if i%4 == 0 {
				// i %4 == 0 ==> i % 2 == 0 ==> we can just use the identical label
				g.AddData(-(3*i + 8), 3, strconv.Itoa(i))
			} else {
				g.AddData(3*i+8, 3, strconv.Itoa(i))
			}
		}

		// randomly fill 100 more elements
		source := rand.New(rand.NewSource(int64(N)))
		for i := 0; i < 100; i++ {
			g.AddTriple(source.Intn(N), 4, source.Intn(N))
			g.AddTriple(source.Intn(N), 5, source.Intn(N))
		}
	}
	g.Finalize()

	// query for all of the paths we have just created
	query, err := g.PathsStarting(2, 2)
	if err != nil {
		t.Fatalf("Unable to start paths: %s", err)
	}
	if err := query.Connected(0); err != nil {
		t.Fatalf("Unable to continue paths: %s", err)
	}
	if err := query.Ending(3, 3); err != nil {
		t.Fatalf("Unable to filter ending paths: %s", err)
	}
	if err := query.Connected(1); err != nil {
		t.Fatalf("Unable to continue paths: %s", err)
	}
	if err := query.Connected(3); err != nil {
		t.Fatalf("Unable to continue paths: %s", err)
	}

	// check that the paths are correct
	paths := query.Paths()

	encountered := make(map[int]struct{})
	for paths.Next() {
		path := paths.Datum()

		// extract the datum
		datum, ok, err := path.Datum()
		if err != nil || !ok {
			t.Errorf("Unable to retrieve Datum: %s", err)
		}
		i64, err := strconv.ParseInt(datum, 10, 64)
		if err != nil {
			t.Errorf("Unable to parse datum: %s", err)
		}

		// find the i and mark that we saw it!
		i := int(i64)
		encountered[i] = struct{}{}

		// determine the nodes and edges we expect
		wantNodes := []int{3*i + 6, 3*i + 7, 3*i + 8}
		wantEdges := []int{0, 1, 3}

		wantTriples := make([]Triple[int, string], 0, 5)
		{
			wantTriples = append(wantTriples, Triple[int, string]{
				Role: Regular,

				Subject:   3*i + 6,
				Predicate: 2,
				Object:    2,

				SSubject:   3*i + 6,
				SPredicate: 2,
				SObject:    2,
			})

			wantTriples = append(wantTriples, Triple[int, string]{
				Role: Regular,

				Subject:   3*i + 7,
				Predicate: 3,
				Object:    3,

				SSubject:   3*i + 7,
				SPredicate: 3,
				SObject:    3,
			})

			if i%4 == 0 || i%4 == 1 {
				wantTriples = append(wantTriples, Triple[int, string]{
					Role: Regular,

					Subject:   3*i + 6,
					Predicate: 0,
					Object:    3*i + 7,

					SSubject:   3*i + 6,
					SPredicate: 0,
					SObject:    3*i + 7,
				})
			} else {
				wantTriples = append(wantTriples, Triple[int, string]{
					Role: Inverse,

					Subject:   3*i + 7,
					Predicate: -1,
					Object:    3*i + 6,

					SSubject:   3*i + 6,
					SPredicate: 0,
					SObject:    3*i + 7,
				})
			}

			if i%4 == 0 || i%4 == 2 {
				wantTriples = append(wantTriples, Triple[int, string]{
					Role: Regular,

					Subject:   3*i + 7,
					Predicate: 1,
					Object:    3*i + 8,

					SSubject:   3*i + 7,
					SPredicate: 1,
					SObject:    3*i + 8,
				})
			} else {
				wantTriples = append(wantTriples, Triple[int, string]{
					Role: Inverse,

					Subject:   3*i + 8,
					Predicate: -2,
					Object:    3*i + 7,

					SSubject:   3*i + 7,
					SPredicate: 1,
					SObject:    3*i + 8,
				})
			}
		}

		if i%4 == 0 {
			wantTriples = append(wantTriples, Triple[int, string]{
				Role: Data,

				Subject:   -(3*i + 8),
				Predicate: 3,

				SSubject:   3*i + 8,
				SPredicate: 3,

				Datum: strconv.Itoa(i),
			})
		} else {
			wantTriples = append(wantTriples, Triple[int, string]{
				Role: Data,

				Subject:   3*i + 8,
				Predicate: 3,
				Object:    0,

				SSubject:   3*i + 8,
				SPredicate: 3,
				SObject:    0,

				Datum: strconv.Itoa(i),
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
		_, ok := encountered[i]
		if !ok {
			t.Errorf("missing index %d", i)
		}
	}
	if len(encountered) != counter {
		t.Error("too few paths encounted")
	}
}
