package persistence

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"
)

func TestVerifierChecksAndClosesEveryStoreInOrder(t *testing.T) {
	events := make([]string, 0, 4)
	postgres := &fakeStore{name: "PostgreSQL", events: &events}
	neo4j := &fakeStore{name: "Neo4j", events: &events}
	verifier := NewVerifier(postgres, neo4j)

	results, err := verifier.Verify(context.Background())

	if err != nil {
		t.Fatalf("Verify() error = %v", err)
	}
	if len(results) != 2 || results[0].Store != "PostgreSQL" || results[1].Store != "Neo4j" {
		t.Fatalf("Verify() results = %#v, want ordered store results", results)
	}
	wantEvents := []string{"verify PostgreSQL", "verify Neo4j", "close Neo4j", "close PostgreSQL"}
	if !equalStrings(events, wantEvents) {
		t.Fatalf("Verify() events = %v, want %v", events, wantEvents)
	}
}

func TestVerifierStopsAfterServiceFailureAndStillClosesEveryStore(t *testing.T) {
	events := make([]string, 0, 4)
	postgres := &fakeStore{
		name:        "PostgreSQL",
		events:      &events,
		verifyError: errors.New("authentication rejected"),
	}
	neo4j := &fakeStore{name: "Neo4j", events: &events}
	verifier := NewVerifier(postgres, neo4j)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	_, err := verifier.Verify(ctx)

	if err == nil || !strings.Contains(err.Error(), "verify PostgreSQL connectivity") {
		t.Fatalf("Verify() error = %v, want service-scoped failure", err)
	}
	wantEvents := []string{"verify PostgreSQL", "close Neo4j", "close PostgreSQL"}
	if !equalStrings(events, wantEvents) {
		t.Fatalf("Verify() events = %v, want %v", events, wantEvents)
	}
	if !postgres.observedDeadline {
		t.Fatal("Verify() did not pass the caller deadline to the store")
	}
}

type fakeStore struct {
	name             string
	events           *[]string
	verifyError      error
	observedDeadline bool
}

func (store *fakeStore) Name() string {
	return store.name
}

func (store *fakeStore) Verify(ctx context.Context) error {
	*store.events = append(*store.events, "verify "+store.name)
	_, store.observedDeadline = ctx.Deadline()
	return store.verifyError
}

func (store *fakeStore) Close(context.Context) error {
	*store.events = append(*store.events, "close "+store.name)
	return nil
}

func equalStrings(left, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	for index := range left {
		if left[index] != right[index] {
			return false
		}
	}
	return true
}
