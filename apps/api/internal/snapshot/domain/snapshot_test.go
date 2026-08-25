package domain

import (
	"testing"

	"github.com/google/uuid"
)

func TestPublishCommandValidateRejectsIncompleteCommand(t *testing.T) {
	command := PublishCommand{CorpusID: uuid.New(), SourceID: uuid.New(), DocumentID: uuid.New(), Actor: "local-maintainer"}
	if err := command.Validate(); err != ErrInvalidInput {
		t.Fatalf("Validate() error = %v, want %v", err, ErrInvalidInput)
	}
}

func TestPublishCommandValidateAcceptsCompleteCommand(t *testing.T) {
	command := PublishCommand{
		CorpusID: uuid.New(), SourceID: uuid.New(), DocumentID: uuid.New(),
		ExpectedReleaseVersion: 1, SnapshotID: uuid.New(), Actor: "local-maintainer",
	}
	if err := command.Validate(); err != nil {
		t.Fatalf("Validate() error = %v", err)
	}
}
