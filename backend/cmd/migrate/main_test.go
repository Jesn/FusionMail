package main

import (
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"testing"
)

func TestSQLMigrationPrefixesAreUnique(t *testing.T) {
	_, currentFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("failed to resolve test file path")
	}

	migrationsDir := filepath.Join(filepath.Dir(currentFile), "..", "..", "migrations")
	entries, err := os.ReadDir(migrationsDir)
	if err != nil {
		t.Fatalf("failed to read migrations dir: %v", err)
	}

	prefixPattern := regexp.MustCompile(`^(\d+)_.*\.sql$`)
	seen := make(map[string]string)

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		matches := prefixPattern.FindStringSubmatch(entry.Name())
		if matches == nil {
			continue
		}

		prefix := matches[1]
		if previous, exists := seen[prefix]; exists {
			t.Fatalf("duplicate migration prefix %s: %s and %s", prefix, previous, entry.Name())
		}
		seen[prefix] = entry.Name()
	}
}

func TestMigrationRootOnlyContainsVersionedSQLAndDocs(t *testing.T) {
	_, currentFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("failed to resolve test file path")
	}

	migrationsDir := filepath.Join(filepath.Dir(currentFile), "..", "..", "migrations")
	entries, err := os.ReadDir(migrationsDir)
	if err != nil {
		t.Fatalf("failed to read migrations dir: %v", err)
	}

	versionedSQLPattern := regexp.MustCompile(`^\d{3}_.+\.sql$`)
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		name := entry.Name()
		if filepath.Ext(name) == ".md" {
			continue
		}
		if versionedSQLPattern.MatchString(name) {
			continue
		}

		t.Fatalf("migrations root contains non-versioned file %s; move manual SQL to migrations/manual/ and maintenance SQL to migrations/maintenance/", name)
	}
}
