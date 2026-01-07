package avro

import (
	"context"
	"fmt"
	"io"
)

// Manifest represents a parsed Iceberg manifest file.
type Manifest struct {
	Entries []ManifestEntry
}

// ReadManifest reads and parses an Iceberg manifest file.
func ReadManifest(ctx context.Context, r io.ReadCloser) (*Manifest, error) {
	reader, err := NewReader[ManifestEntry](r)
	if err != nil {
		return nil, fmt.Errorf("opening manifest: %w", err)
	}
	defer reader.Close()

	entries, err := reader.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("reading manifest entries: %w", err)
	}

	return &Manifest{Entries: entries}, nil
}

// LiveDataFiles returns data files that are currently live (not deleted).
func (m *Manifest) LiveDataFiles() []DataFile {
	var files []DataFile
	for _, entry := range m.Entries {
		// Skip deleted entries
		if entry.Status == StatusDeleted {
			continue
		}
		// Only include data files
		if entry.DataFile.IsDataFile() {
			files = append(files, entry.DataFile)
		}
	}
	return files
}

// LiveDeleteFiles returns delete files that are currently live.
func (m *Manifest) LiveDeleteFiles() []DataFile {
	var files []DataFile
	for _, entry := range m.Entries {
		// Skip deleted entries
		if entry.Status == StatusDeleted {
			continue
		}
		// Only include delete files
		if entry.DataFile.IsDeleteFile() {
			files = append(files, entry.DataFile)
		}
	}
	return files
}

// AllLiveFiles returns all files (data + delete) that are currently live.
func (m *Manifest) AllLiveFiles() []DataFile {
	var files []DataFile
	for _, entry := range m.Entries {
		if entry.Status != StatusDeleted {
			files = append(files, entry.DataFile)
		}
	}
	return files
}

// DataFileCount returns the count of live data files.
func (m *Manifest) DataFileCount() int {
	count := 0
	for _, entry := range m.Entries {
		if entry.Status != StatusDeleted && entry.DataFile.IsDataFile() {
			count++
		}
	}
	return count
}

// DeleteFileCount returns the count of live delete files.
func (m *Manifest) DeleteFileCount() int {
	count := 0
	for _, entry := range m.Entries {
		if entry.Status != StatusDeleted && entry.DataFile.IsDeleteFile() {
			count++
		}
	}
	return count
}

// TotalRecordCount returns the sum of record counts for all live data files.
func (m *Manifest) TotalRecordCount() int64 {
	var total int64
	for _, entry := range m.Entries {
		if entry.Status != StatusDeleted && entry.DataFile.IsDataFile() {
			total += entry.DataFile.RecordCount
		}
	}
	return total
}

// TotalFileSizeBytes returns the sum of file sizes for all live data files.
func (m *Manifest) TotalFileSizeBytes() int64 {
	var total int64
	for _, entry := range m.Entries {
		if entry.Status != StatusDeleted && entry.DataFile.IsDataFile() {
			total += entry.DataFile.FileSizeBytes
		}
	}
	return total
}
