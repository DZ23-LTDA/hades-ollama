package agent

import "testing"

// TestPostgresSchemaMigrationChecksumIsStableAndDriftSensitive verifies the
// migration-ledger fingerprint: identical definitions produce the same
// checksum (so idempotent re-applies do not look like drift), any change
// produces a different one (so drift is recorded), and the grouping/ordering
// is significant (two statements cannot be merged into one without changing it).
func TestPostgresSchemaMigrationChecksumIsStableAndDriftSensitive(t *testing.T) {
	base := postgresSchemaMigrationChecksum(
		[]string{"CREATE TABLE a (...)", "CREATE TABLE b (...)"},
		[]string{"GRANT x"},
	)
	same := postgresSchemaMigrationChecksum(
		[]string{"CREATE TABLE a (...)", "CREATE TABLE b (...)"},
		[]string{"GRANT x"},
	)
	if base != same {
		t.Fatalf("identical migration definitions must share a checksum: %s vs %s", base, same)
	}

	changed := postgresSchemaMigrationChecksum(
		[]string{"CREATE TABLE a (...)", "CREATE TABLE b (...)", "ALTER TABLE a ADD COLUMN c"},
		[]string{"GRANT x"},
	)
	if base == changed {
		t.Fatal("adding a statement must change the checksum (drift must be visible)")
	}

	// Separator significance: the same bytes split differently must differ.
	merged := postgresSchemaMigrationChecksum(
		[]string{"CREATE TABLE a (...)CREATE TABLE b (...)"},
		[]string{"GRANT x"},
	)
	if base == merged {
		t.Fatal("statement boundaries must be significant in the checksum")
	}

	if len(base) != 64 {
		t.Fatalf("checksum must be a 64-char sha256 hex string, got %d chars", len(base))
	}
}
