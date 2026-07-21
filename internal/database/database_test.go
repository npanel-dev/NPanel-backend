package database

import "testing"

func TestNormalizeDriver(t *testing.T) {
	tests := []struct {
		input string
		want  Driver
	}{
		{input: "mysql", want: MySQL},
		{input: " PostgreSQL ", want: Postgres},
		{input: "postgres", want: Postgres},
	}
	for _, tt := range tests {
		got, err := NormalizeDriver(tt.input)
		if err != nil {
			t.Fatalf("NormalizeDriver(%q): %v", tt.input, err)
		}
		if got != tt.want {
			t.Fatalf("NormalizeDriver(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}

	if _, err := NormalizeDriver("sqlite3"); err == nil {
		t.Fatal("NormalizeDriver(sqlite3) succeeded, want unsupported-driver error")
	}
}

func TestOpenSQLMapsLogicalDrivers(t *testing.T) {
	tests := []struct {
		driver Driver
		source string
	}{
		{driver: MySQL, source: "user:password@tcp(localhost:3306)/database"},
		{driver: Postgres, source: "postgres://user:password@localhost/database?sslmode=disable"},
	}
	for _, tt := range tests {
		db, err := OpenSQL(tt.driver, tt.source)
		if err != nil {
			t.Fatalf("OpenSQL(%s): %v", tt.driver, err)
		}
		if err := db.Close(); err != nil {
			t.Fatalf("Close(%s): %v", tt.driver, err)
		}
	}
}
