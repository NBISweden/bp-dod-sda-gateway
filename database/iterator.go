package database

import "database/sql"

type Iterator[T any] struct {
	rows *sql.Rows
	scan func(*sql.Rows) (T, error)
}

func NewIterator[T any](rows *sql.Rows, scanFunc func(*sql.Rows) (T, error)) *Iterator[T] {
	return &Iterator[T]{rows: rows, scan: scanFunc}
}

func (it *Iterator[T]) Next() (T, bool, error) {
	var zero T

	if !it.rows.Next() {
		return zero, false, it.rows.Err()
	}

	v, err := it.scan(it.rows)
	return v, true, err
}

func (it *Iterator[T]) Close() error {
	return it.rows.Close()
}
