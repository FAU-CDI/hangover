package sqlitey

import (
	"io"
	"sync"

	"zombiezen.com/go/sqlite"
)

// StreamStatement returns a function to execute a query repreatedly.
// bind may be nil and is called to bind the currently streamed value in the query.
// cacheSize determines caching of the internal message channel.
//
// It is safe to call execute and closer concurrently and repeatedly.
// Once closer has been called, it will always return the same value.
func StreamStatement[T any](conn *sqlite.Conn, query string, bind func(stmt *sqlite.Stmt, value T) error, cacheSize int) (execute func(t T) error, closer func() error, err error) {
	stmt, err := conn.Prepare(query)
	if err != nil {
		return nil, nil, err
	}

	// messages used for insertion
	type message = struct {
		value  T
		result chan error
	}

	var (
		done      = make(chan struct{}, cacheSize) // closed once everything is done
		doneError error                            // error for calls to return

		inserts = make(chan message)
		useM    sync.RWMutex // used to allow concurrent calls to closer() and execute()
	)

	go func() {
		defer close(done)

		// finalize the statement at the end
		// only send error if we didn't have anything yet.
		defer func() {
			err := stmt.Finalize()
			if err != nil && doneError == nil {
				doneError = err
			}
		}()

		for msg := range inserts {
			// had an error => consume and don't do anything
			if doneError != nil {
				msg.result <- doneError
				continue
			}

			// do all the binding
			if bind != nil {
				if doneError = bind(stmt, msg.value); doneError != nil {
					msg.result <- doneError
					continue
				}
			}

			// step through it!
			var returned bool
			for {
				returned, doneError = stmt.Step()
				if !returned {
					break
				}
			}

			// check that everything went ok
			if doneError != nil {
				msg.result <- doneError
				continue
			}

			// done
			close(msg.result)

			// prepare for the next invocation
			stmt.Reset()
			stmt.ClearBindings()
		}
	}()

	return func(value T) error {
			useM.RLock()
			defer useM.RUnlock()

			// if we are done, bail out!
			select {
			case <-done:
				return doneError
			default:
			}

			// create a message struct
			// create a new message and result chann
			result := make(chan error, 1)
			msg := message{
				value:  value,
				result: result,
			}

			select {
			case inserts <- msg:
				return <-msg.result
			case <-done:
				return doneError
			}
		}, sync.OnceValue(func() error {
			useM.Lock()
			defer useM.Unlock()

			close(inserts) // no more insert channels
			<-done

			if doneError == nil {
				doneError = io.EOF
				return nil
			}

			return doneError
		}), nil
}
