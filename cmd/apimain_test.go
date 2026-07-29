package main

import "testing"

// TestIsValidDatabaseName covers valid and invalid database name validation examples.
func TestIsValidDatabaseName(t *testing.T) {
	valid := []string{
		"rustdesk",
		"rustdesk_api",
		"_internal",
		"db2024",
		"a",
		"Abc123_Xy",
	}
	for _, name := range valid {
		if !isValidDatabaseName(name) {
			t.Errorf("expected %q to be a valid database name", name)
		}
	}

	invalid := []string{
		"",          // empty
		"1db",       // starts with a digit
		"db-name",   // hyphen
		"db name",   // space
		"db;DROP",   // injection attempt
		"db`x",      // backtick
		"../../etc", // path traversal characters
		"db.name",   // dot
		"-db",       // starts with a symbol other than underscore
	}
	for _, name := range invalid {
		if isValidDatabaseName(name) {
			t.Errorf("expected %q to be an invalid database name", name)
		}
	}

	// Names longer than 64 bytes should be invalid.
	tooLong := make([]byte, 65)
	for i := range tooLong {
		tooLong[i] = 'a'
	}
	if isValidDatabaseName(string(tooLong)) {
		t.Errorf("expected 65-char name to be invalid (length limit 64)")
	}

	// Exactly 64 bytes should be valid.
	exact := make([]byte, 64)
	for i := range exact {
		exact[i] = 'a'
	}
	if !isValidDatabaseName(string(exact)) {
		t.Errorf("expected 64-char name to be valid")
	}
}
