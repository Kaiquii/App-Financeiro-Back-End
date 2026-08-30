package migrations

import "testing"

func TestRegistryStartsWithFrozenBaseline(t *testing.T) {
	migrations := sortedRegistry()
	if len(migrations) == 0 {
		t.Fatal("expected at least one migration")
	}
	if migrations[0].Version != 1 || migrations[0].Name != "baseline_current_schema" {
		t.Fatalf("unexpected baseline: %+v", migrations[0])
	}

	seen := make(map[int64]bool, len(migrations))
	for _, migration := range migrations {
		if seen[migration.Version] {
			t.Fatalf("duplicate migration version: %d", migration.Version)
		}
		seen[migration.Version] = true
		if migration.Up == nil {
			t.Fatalf("migration %d has no Up function", migration.Version)
		}
		if len(migration.Checksum) != 64 {
			t.Fatalf("migration %d has invalid checksum %q", migration.Version, migration.Checksum)
		}
	}
}

func TestRegistryIncludesUserLastActiveMigration(t *testing.T) {
	for _, migration := range sortedRegistry() {
		if migration.Version == 5 {
			if migration.Name != "add_user_last_active" {
				t.Fatalf("unexpected migration 5: %+v", migration)
			}
			return
		}
	}
	t.Fatal("migration 5 was not registered")
}

func TestUnknownAppliedMigrationIsRejected(t *testing.T) {
	err := validateAppliedMigrations(map[int64]AppliedMigration{
		999: {Version: 999, Name: "future"},
	})
	if err == nil {
		t.Fatal("expected unknown migration to be rejected")
	}
}

func TestMatchingAppliedMigrationsAreAccepted(t *testing.T) {
	migration := sortedRegistry()[0]
	err := validateAppliedMigrations(map[int64]AppliedMigration{
		migration.Version: {
			Version:  migration.Version,
			Name:     migration.Name,
			Checksum: migration.Checksum,
		},
	})
	if err != nil {
		t.Fatalf("expected matching migration to be accepted: %v", err)
	}
}

func TestAppliedMigrationNameMismatchIsRejected(t *testing.T) {
	migration := sortedRegistry()[0]
	err := validateAppliedMigrations(map[int64]AppliedMigration{
		migration.Version: {
			Version:  migration.Version,
			Name:     "renamed_migration",
			Checksum: migration.Checksum,
		},
	})
	if err == nil {
		t.Fatal("expected migration name mismatch to be rejected")
	}
}

func TestAppliedMigrationChecksumMismatchIsRejected(t *testing.T) {
	migration := sortedRegistry()[0]
	err := validateAppliedMigrations(map[int64]AppliedMigration{
		migration.Version: {
			Version:  migration.Version,
			Name:     migration.Name,
			Checksum: "different-checksum",
		},
	})
	if err == nil {
		t.Fatal("expected migration checksum mismatch to be rejected")
	}
}

func TestPendingMigrationsAreDetected(t *testing.T) {
	pending := pendingMigrations(map[int64]AppliedMigration{})
	if len(pending) != len(registry) {
		t.Fatalf("expected %d pending migrations, got %d", len(registry), len(pending))
	}

	migration := sortedRegistry()[0]
	applied := map[int64]AppliedMigration{
		migration.Version: {
			Version:  migration.Version,
			Name:     migration.Name,
			Checksum: migration.Checksum,
		},
	}
	if remaining := pendingMigrations(applied); len(remaining) != len(registry)-1 {
		t.Fatalf("expected %d pending migrations, got %d", len(registry)-1, len(remaining))
	}
}

func TestBaselineDefinitionCoversCurrentTables(t *testing.T) {
	expected := []string{
		"users",
		"password_reset_tokens",
		"registration_codes",
		"expenses",
		"incomes",
		"categories",
		"conversations",
		"messages",
		"app_versions",
	}
	for _, table := range expected {
		if len(baselineColumns[table]) == 0 {
			t.Fatalf("baseline does not define table %s", table)
		}
	}
}

func TestBaselineSQLDefinitionIsImmutable(t *testing.T) {
	const expected = "3eaee9311646e1fbcf6ecd7450056ed527180bad715cf5f448c07db822f9a066"
	if got := baselineDefinitionChecksum(); got != expected {
		t.Fatalf("baseline SQL changed: expected %s, got %s", expected, got)
	}
}
