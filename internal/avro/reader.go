package avro

import (
	"fmt"
	"io"

	"github.com/hamba/avro/v2"
	"github.com/hamba/avro/v2/ocf"
)

// Reader provides generic Avro OCF (Object Container File) reading.
type Reader[T any] struct {
	reader *ocf.Decoder
	closer io.Closer
}

// NewReader creates a new Avro OCF reader for the given type.
// The reader uses schema resolution to handle schema evolution.
func NewReader[T any](r io.ReadCloser) (*Reader[T], error) {
	decoder, err := ocf.NewDecoder(r)
	if err != nil {
		r.Close()
		return nil, fmt.Errorf("creating avro decoder: %w", err)
	}

	return &Reader[T]{
		reader: decoder,
		closer: r,
	}, nil
}

// ReadAll reads all records from the Avro file.
func (r *Reader[T]) ReadAll() ([]T, error) {
	var records []T

	for r.reader.HasNext() {
		var record T
		if err := r.reader.Decode(&record); err != nil {
			return nil, fmt.Errorf("decoding avro record: %w", err)
		}
		records = append(records, record)
	}

	if err := r.reader.Error(); err != nil {
		return nil, fmt.Errorf("reading avro file: %w", err)
	}

	return records, nil
}

// Close releases resources held by the reader.
func (r *Reader[T]) Close() error {
	return r.closer.Close()
}

// Metadata returns the file metadata from the Avro header.
func (r *Reader[T]) Metadata() map[string][]byte {
	return r.reader.Metadata()
}

// Schema returns the Avro schema string from the file header.
func (r *Reader[T]) Schema() string {
	if schema := r.reader.Metadata()["avro.schema"]; schema != nil {
		return string(schema)
	}
	return ""
}

// ParseSchema parses an Avro schema string.
// This is useful for dynamic schema handling.
func ParseSchema(schemaJSON string) (avro.Schema, error) {
	return avro.Parse(schemaJSON)
}
