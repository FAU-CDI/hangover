package sqlitey

import (
	"errors"
	"fmt"
	"math"
	"reflect"

	"zombiezen.com/go/sqlite"
)

var errResultTrailing = errors.New("Result: Trailing bytes found")

// Result returns the result of a single query.
// It expects no trailing bytes left in the query.
func Result[T any](conn *sqlite.Conn, query string, args []any, f func(stmt *sqlite.Stmt) (T, error)) (T, error) {
	var value T // for error returns

	// prepare the query
	stmt, tb, err := conn.PrepareTransient(query)
	if err != nil {
		return value, fmt.Errorf("failed to prepare query: %w", err)
	}
	defer stmt.Finalize()

	// check that there are no trailing bytes
	if tb != 0 {
		return value, errResultTrailing
	}

	// bind the arguments!
	if err := BindArgs(stmt, args); err != nil {
		return value, fmt.Errorf("unable to bind arguments: %w", err)
	}

	// execute the result
	return f(stmt)
}

// BindArgs binds numbered arguments inside a query.
func BindArgs(stmt *sqlite.Stmt, args []any) error {
	// parameter count
	paramCount := stmt.BindParamCount()
	if len(args) != paramCount {
		return fmt.Errorf("BindArgs: invalid argument count: Expected %d, but got %d", paramCount, len(args))
	}

	for i, value := range args {
		if err := setArg(stmt, i, reflect.ValueOf(value)); err != nil {
			return fmt.Errorf("failed to set argument %d: %w", i, err)
		}
	}

	return nil
}

var (
	errTooBig = errors.New("setArg: uint value too big")
)

func setArg(stmt *sqlite.Stmt, i int, v reflect.Value) error {
	switch v.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		stmt.BindInt64(i, v.Int())
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		val := v.Uint()
		if val > math.MaxInt64 {
			return errTooBig
		}
		stmt.BindInt64(i, int64(val))
	case reflect.Float32, reflect.Float64:
		stmt.BindFloat(i, v.Float())
	case reflect.String:
		stmt.BindText(i, v.String())
	case reflect.Bool:
		stmt.BindBool(i, v.Bool())
	case reflect.Invalid:
		stmt.BindNull(i)
	case reflect.Complex64, reflect.Complex128, reflect.Array, reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Pointer, reflect.Slice, reflect.Struct, reflect.UnsafePointer:
		if v.Kind() == reflect.Slice && v.Type().Elem().Kind() == reflect.Uint8 {
			stmt.BindBytes(i, v.Bytes())
		} else {
			stmt.BindText(i, fmt.Sprint(v.Interface()))
		}
	}
	return nil
}
