package store

import (
	"database/sql"
	"errors"
	"path/filepath"
	"testing"

	_ "modernc.org/sqlite"
)

func TestMigrateRecordsInitialMigrationForFreshDatabase(t *testing.T) {
	db, err := Open(filepath.Join(t.TempDir(), "feedctl.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	assertAppliedMigrationVersions(t, db, []int{1})
}

func TestMigrateIsIdempotentForAlreadyAppliedInitialMigration(t *testing.T) {
	path := filepath.Join(t.TempDir(), "feedctl.db")
	db, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	assertAppliedMigrationVersions(t, db, []int{1})
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}

	db, err = Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	assertAppliedMigrationVersions(t, db, []int{1})
}

func TestApplyMigrationsFailureDoesNotRecordVersion(t *testing.T) {
	sqlDB, err := sql.Open("sqlite", filepath.Join(t.TempDir(), "feedctl.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer sqlDB.Close()
	db := &DB{db: sqlDB}
	boom := errors.New("boom")
	migrations := []migration{{version: 1, up: func(*sql.Tx) error { return boom }}}
	if err := db.applyMigrations(migrations); !errors.Is(err, boom) {
		t.Fatalf("applyMigrations error=%v want boom", err)
	}
	assertAppliedMigrationVersions(t, db, nil)
}

func TestOpenReturnsNilDBWhenMigrationFails(t *testing.T) {
	db, err := Open(t.TempDir())
	if err == nil {
		if db != nil {
			_ = db.Close()
		}
		t.Fatal("Open unexpectedly succeeded for directory database path")
	}
	if db != nil {
		t.Fatalf("Open returned usable DB on migration failure: %#v", db)
	}
}

func TestApplyMigrationsRunsPendingVersionsInOrder(t *testing.T) {
	sqlDB, err := sql.Open("sqlite", filepath.Join(t.TempDir(), "feedctl.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer sqlDB.Close()
	db := &DB{db: sqlDB}
	if _, err := sqlDB.Exec(`CREATE TABLE schema_migrations (version INTEGER PRIMARY KEY, applied_at TEXT NOT NULL)`); err != nil {
		t.Fatal(err)
	}
	if _, err := sqlDB.Exec(`INSERT INTO schema_migrations(version, applied_at) VALUES (1, 'already')`); err != nil {
		t.Fatal(err)
	}
	if _, err := sqlDB.Exec(`CREATE TABLE migration_log (position INTEGER PRIMARY KEY AUTOINCREMENT, version INTEGER NOT NULL)`); err != nil {
		t.Fatal(err)
	}

	migrations := []migration{
		{version: 1, up: func(*sql.Tx) error { t.Fatal("migration 1 should be skipped"); return nil }},
		{version: 2, up: func(tx *sql.Tx) error { _, err := tx.Exec(`INSERT INTO migration_log(version) VALUES (2)`); return err }},
		{version: 3, up: func(tx *sql.Tx) error { _, err := tx.Exec(`INSERT INTO migration_log(version) VALUES (3)`); return err }},
	}
	if err := db.applyMigrations(migrations); err != nil {
		t.Fatal(err)
	}

	rows, err := sqlDB.Query(`SELECT version FROM migration_log ORDER BY position`)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	var got []int
	for rows.Next() {
		var version int
		if err := rows.Scan(&version); err != nil {
			t.Fatal(err)
		}
		got = append(got, version)
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}
	assertInts(t, got, []int{2, 3})
	assertAppliedMigrationVersions(t, db, []int{1, 2, 3})
}

func assertAppliedMigrationVersions(t *testing.T, db *DB, want []int) {
	t.Helper()
	rows, err := db.Raw().Query(`SELECT version FROM schema_migrations ORDER BY version`)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	var got []int
	for rows.Next() {
		var version int
		if err := rows.Scan(&version); err != nil {
			t.Fatal(err)
		}
		got = append(got, version)
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}
	assertInts(t, got, want)
}

func assertInts(t *testing.T, got, want []int) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("versions=%v want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("versions=%v want %v", got, want)
		}
	}
}
