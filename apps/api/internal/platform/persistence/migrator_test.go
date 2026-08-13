package persistence

import (
	"context"
	"errors"
	"strings"
	"testing"
)

func TestMigrationServiceAppliesPendingMigrationsAndReportsVersion(t *testing.T) {
	engine := &fakeMigrationEngine{version: 1}
	service := NewMigrationService(engine)

	status, err := service.Apply(context.Background())

	if err != nil {
		t.Fatalf("Apply() error = %v", err)
	}
	if !engine.migrated {
		t.Fatal("Apply() did not migrate to the latest version")
	}
	if status.CurrentVersion != 1 {
		t.Fatalf("Apply() version = %d, want 1", status.CurrentVersion)
	}
}

func TestMigrationServiceReportsCurrentVersionWithoutApplying(t *testing.T) {
	engine := &fakeMigrationEngine{version: 1}
	service := NewMigrationService(engine)

	status, err := service.Status(context.Background())

	if err != nil {
		t.Fatalf("Status() error = %v", err)
	}
	if engine.migrated {
		t.Fatal("Status() applied a migration")
	}
	if status.CurrentVersion != 1 {
		t.Fatalf("Status() version = %d, want 1", status.CurrentVersion)
	}
}

func TestMigrationServiceAttributesMigrationFailure(t *testing.T) {
	engine := &fakeMigrationEngine{migrationError: errors.New("migration 001_enable_vector failed")}
	service := NewMigrationService(engine)

	_, err := service.Apply(context.Background())

	if err == nil {
		t.Fatal("Apply() error = nil, want migration failure")
	}
	if !strings.Contains(err.Error(), "apply PostgreSQL migrations") {
		t.Fatalf("Apply() error = %q, want operation context", err)
	}
}

type fakeMigrationEngine struct {
	version        int32
	migrated       bool
	migrationError error
	statusError    error
}

func (engine *fakeMigrationEngine) Migrate(context.Context) error {
	engine.migrated = true
	return engine.migrationError
}

func (engine *fakeMigrationEngine) GetCurrentVersion(context.Context) (int32, error) {
	return engine.version, engine.statusError
}
