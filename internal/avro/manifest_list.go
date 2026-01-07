package avro

import (
	"context"
	"fmt"
	"io"
)

// ManifestList represents a parsed Iceberg manifest list.
type ManifestList struct {
	Entries []ManifestListEntry
}

// ReadManifestList reads and parses an Iceberg manifest list file.
func ReadManifestList(ctx context.Context, r io.ReadCloser) (*ManifestList, error) {
	reader, err := NewReader[ManifestListEntry](r)
	if err != nil {
		return nil, fmt.Errorf("opening manifest list: %w", err)
	}
	defer reader.Close()

	entries, err := reader.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("reading manifest list entries: %w", err)
	}

	return &ManifestList{Entries: entries}, nil
}

// DataManifests returns manifest entries that contain data files.
func (m *ManifestList) DataManifests() []ManifestListEntry {
	var manifests []ManifestListEntry
	for _, entry := range m.Entries {
		if entry.Content == ContentData {
			manifests = append(manifests, entry)
		}
	}
	return manifests
}

// DeleteManifests returns manifest entries that contain delete files.
func (m *ManifestList) DeleteManifests() []ManifestListEntry {
	var manifests []ManifestListEntry
	for _, entry := range m.Entries {
		if entry.Content == ContentPositionDeletes || entry.Content == ContentEqualityDeletes {
			manifests = append(manifests, entry)
		}
	}
	return manifests
}

// TotalFiles returns the estimated total number of files across all manifests.
func (m *ManifestList) TotalFiles() int64 {
	var total int64
	for _, entry := range m.Entries {
		if entry.AddedFilesCount != nil {
			total += int64(*entry.AddedFilesCount)
		}
		if entry.ExistingFilesCount != nil {
			total += int64(*entry.ExistingFilesCount)
		}
	}
	return total
}

// TotalDataFiles returns the estimated number of data files.
func (m *ManifestList) TotalDataFiles() int64 {
	var total int64
	for _, entry := range m.Entries {
		if entry.Content != ContentData {
			continue
		}
		if entry.AddedFilesCount != nil {
			total += int64(*entry.AddedFilesCount)
		}
		if entry.ExistingFilesCount != nil {
			total += int64(*entry.ExistingFilesCount)
		}
	}
	return total
}

// ManifestPaths returns all manifest file paths.
func (m *ManifestList) ManifestPaths() []string {
	paths := make([]string, len(m.Entries))
	for i, entry := range m.Entries {
		paths[i] = entry.ManifestPath
	}
	return paths
}
