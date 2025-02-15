package sqlitey_test

import (
	"fmt"
	"sync"

	"github.com/FAU-CDI/hangover/internal/sqlitey"
	"zombiezen.com/go/sqlite"
	"zombiezen.com/go/sqlite/sqlitex"
)

type KeyValue struct {
	Key   string
	Value string
}

func ExampleStreamStatement() {
	// create a new in-memory db
	conn, err := sqlite.OpenConn(":memory:")
	if err != nil {
		panic(err)
	}
	defer conn.Close()

	// create an example table
	if err := sqlitex.ExecuteTransient(conn, `CREATE TABLE "Example" ("key" TEXT PRIMARY KEY, "value" TEXT NOT NULL);`, nil); err != nil {
		panic(err)
	}

	// prepare streaming of the statement
	insert, close, err := sqlitey.StreamStatement(conn, `INSERT INTO "EXAMPLE" ("key", "value") VALUES (?, ?);`, func(stmt *sqlite.Stmt, value *KeyValue) error {
		stmt.BindText(1, value.Key)
		stmt.BindText(2, value.Value)
		return nil
	}, 100)
	if err != nil {
		panic(err)
	}
	defer close()

	// generate a bunch of random values
	values := make([]KeyValue, 10000)
	for i := range values {
		values[i].Key = fmt.Sprint(i)
		values[i].Value = "test"
	}

	// do the actual inserts
	var wg sync.WaitGroup
	wg.Add(len(values))

	for _, kv := range values {
		go func() {
			defer wg.Done()

			if err := insert(&kv); err != nil {
				panic(err)
			}
		}()
	}
	wg.Wait()

	// check that we got the right amount of results
	gotResult := -1
	if err := sqlitex.ExecuteTransient(conn, `SELECT COUNT(*) FROM "Example"`, &sqlitex.ExecOptions{
		ResultFunc: func(stmt *sqlite.Stmt) error {
			gotResult = int(stmt.ColumnInt64(0))
			return nil
		},
	}); err != nil {
		panic(err)
	}

	fmt.Print(gotResult)

	// Output: 10000
}
