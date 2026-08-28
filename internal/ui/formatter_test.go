package ui

import (
	"testing"

	"github.com/mona-actions/gh-gitlab-stats/internal/models"
)

// TestConvertToCSVRowSizePrecision verifies size fields render with one decimal
// so values like 25.5 are preserved in the CSV output.
func TestConvertToCSVRowSizePrecision(t *testing.T) {
	tests := []struct {
		name        string
		repoSizeMB  float64
		lfsSizeMB   float64
		wantSize    string
		wantLFSSize string
	}{
		{"decimal preserved", 25.5, 10.2, "25.5", "10.2"},
		{"whole number gets one decimal", 250, 0, "250.0", "0.0"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stat := &models.RepositoryStats{RepoSizeMB: tt.repoSizeMB, LFSSizeMB: tt.lfsSizeMB}
			row := convertToCSVRow(stat)
			// Project_Size(mb) is column index 5, LFS_Size(mb) is column index 6.
			if got := row[5]; got != tt.wantSize {
				t.Errorf("Project_Size(mb) = %q, want %q", got, tt.wantSize)
			}
			if got := row[6]; got != tt.wantLFSSize {
				t.Errorf("LFS_Size(mb) = %q, want %q", got, tt.wantLFSSize)
			}
		})
	}
}

// TestCSVSchemaStable guards the CSV schema: the row must have the same number of
// columns as the header, and the two size columns must stay at their expected
// positions. This proves formatting changes do not alter column count or order.
func TestCSVSchemaStable(t *testing.T) {
	headers := getCSVHeaders()
	row := convertToCSVRow(&models.RepositoryStats{})

	if len(row) != len(headers) {
		t.Fatalf("row has %d columns, want %d (matching headers)", len(row), len(headers))
	}

	if headers[5] != "Project_Size(mb)" {
		t.Errorf("header[5] = %q, want %q", headers[5], "Project_Size(mb)")
	}
	if headers[6] != "LFS_Size(mb)" {
		t.Errorf("header[6] = %q, want %q", headers[6], "LFS_Size(mb)")
	}
}
