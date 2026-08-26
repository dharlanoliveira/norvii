package application

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/dharlanoliveira/norvii/apps/api/internal/suggestions/domain"
	"github.com/google/uuid"
)

func TestServicePublish(t *testing.T) {
	t.Parallel()

	for _, testCase := range []publishServiceCase{
		{name: "publishes reviewed compatible projection", configure: func(_ *catalogFake, _ *activeSnapshotFake, _ *projectionAppenderFake, _ *PublishCommand) {}, wantWrite: 1},
		{
			name: "rejects invalid command before reads", configure: func(_ *catalogFake, _ *activeSnapshotFake, _ *projectionAppenderFake, command *PublishCommand) {
				command.SnapshotManifestSHA256 = "invalid"
			}, want: ErrInvalidCommand,
		},
		{
			name: "rejects stale requested snapshot before catalog reads", configure: func(_ *catalogFake, active *activeSnapshotFake, _ *projectionAppenderFake, _ *PublishCommand) {
				active.snapshots[0].ID = fixtureID(77)
			}, want: ErrStaleActiveRelease,
		},
		{
			name: "rejects unavailable dataset before append", configure: func(catalog *catalogFake, _ *activeSnapshotFake, _ *projectionAppenderFake, _ *PublishCommand) {
				catalog.publication.PublicationState = "draft"
			}, want: domain.ErrUnavailableDataset,
		},
		{
			name: "rejects publication for another dataset revision before append", configure: func(catalog *catalogFake, _ *activeSnapshotFake, _ *projectionAppenderFake, _ *PublishCommand) {
				catalog.publication.DatasetRevisionID = fixtureID(98)
			}, want: domain.ErrInvalidInput,
		},
		{
			name: "rejects foreign selected case before append", configure: func(catalog *catalogFake, _ *activeSnapshotFake, _ *projectionAppenderFake, _ *PublishCommand) {
				catalog.starterCases[0].CorpusID = fixtureID(99)
			}, want: domain.ErrCorpusMismatch,
		},
		{
			name: "rejects missing evidence resolution before append", configure: func(catalog *catalogFake, _ *activeSnapshotFake, _ *projectionAppenderFake, _ *PublishCommand) {
				catalog.resolvedEvidence = catalog.resolvedEvidence[:1]
			}, want: domain.ErrUnresolvedExpectedEvidence,
		},
		{
			name: "rejects ambiguous evidence resolution before append", configure: func(catalog *catalogFake, _ *activeSnapshotFake, _ *projectionAppenderFake, _ *PublishCommand) {
				catalog.resolvedEvidence = append(catalog.resolvedEvidence, catalog.resolvedEvidence[0])
			}, want: domain.ErrUnresolvedExpectedEvidence,
		},
		{
			name: "rejects active release changed before append", configure: func(_ *catalogFake, active *activeSnapshotFake, _ *projectionAppenderFake, _ *PublishCommand) {
				later := active.snapshots[0]
				later.ID = fixtureID(77)
				active.snapshots = append(active.snapshots, later)
			}, want: ErrStaleActiveRelease,
		},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			assertPublishServiceCase(t, testCase)
		})
	}
}

type publishServiceCase struct {
	name      string
	configure func(*catalogFake, *activeSnapshotFake, *projectionAppenderFake, *PublishCommand)
	want      error
	wantWrite int
}

func assertPublishServiceCase(t *testing.T, testCase publishServiceCase) {
	t.Helper()
	catalog, active, writer, command := validServiceInputs()
	testCase.configure(&catalog, &active, &writer, &command)
	service := NewService(&catalog, &active, &writer, func() uuid.UUID { return fixtureID(30) })
	projection, err := service.Publish(context.Background(), command)
	if !errors.Is(err, testCase.want) {
		t.Fatalf("Publish() error = %v, want %v", err, testCase.want)
	}
	if writer.calls != testCase.wantWrite {
		t.Fatalf("AppendProjection() calls = %d, want %d", writer.calls, testCase.wantWrite)
	}
	assertPublishProjection(t, testCase.wantWrite, projection, writer)
	assertPublishDependencyCalls(t, testCase.want, active, catalog)
}

func assertPublishProjection(t *testing.T, wantWrites int, projection domain.Projection, writer projectionAppenderFake) {
	t.Helper()
	if wantWrites == 1 && (projection.Set.ID != fixtureID(30) || len(projection.Items) != 2 || !reflect.DeepEqual(writer.projection, projection)) {
		t.Fatalf("Publish() projection = %#v, writer projection = %#v", projection, writer.projection)
	}
}

func assertPublishDependencyCalls(t *testing.T, want error, active activeSnapshotFake, catalog catalogFake) {
	t.Helper()
	if errors.Is(want, ErrInvalidCommand) && (active.calls != 0 || catalog.calls != 0) {
		t.Fatalf("invalid command called dependencies: active=%d catalog=%d", active.calls, catalog.calls)
	}
	if errors.Is(want, ErrStaleActiveRelease) && active.calls == 0 {
		t.Fatal("stale release was not checked against active snapshot")
	}
}

func TestServicePublishPropagatesPortErrorsWithoutAppend(t *testing.T) {
	t.Parallel()

	catalog, active, writer, command := validServiceInputs()
	catalog.publicationErr = errors.New("catalog unavailable")
	service := NewService(&catalog, &active, &writer, func() uuid.UUID { return fixtureID(30) })
	_, err := service.Publish(context.Background(), command)
	if !errors.Is(err, catalog.publicationErr) {
		t.Fatalf("Publish() error = %v, want wrapped catalog error", err)
	}
	if writer.calls != 0 {
		t.Fatalf("AppendProjection() calls = %d, want 0", writer.calls)
	}
}

type catalogFake struct {
	publication      domain.DatasetPublication
	starterCases     []domain.StarterCase
	expectedEvidence []domain.ExpectedEvidenceRequirement
	resolvedEvidence []domain.ExpectedEvidenceResolution
	publicationErr   error
	calls            int
}

func (fake *catalogFake) DatasetPublication(_ context.Context, _ uuid.UUID, _ uuid.UUID) (domain.DatasetPublication, error) {
	fake.calls++
	return fake.publication, fake.publicationErr
}

func (fake *catalogFake) StarterCases(_ context.Context, _ uuid.UUID, _ uuid.UUID) ([]domain.StarterCase, error) {
	fake.calls++
	return fake.starterCases, nil
}

func (fake *catalogFake) ExpectedEvidence(_ context.Context, _ uuid.UUID, _ uuid.UUID) ([]domain.ExpectedEvidenceRequirement, error) {
	fake.calls++
	return fake.expectedEvidence, nil
}

func (fake *catalogFake) ResolvedExpectedEvidence(_ context.Context, _ uuid.UUID, _ uuid.UUID, _ uuid.UUID) ([]domain.ExpectedEvidenceResolution, error) {
	fake.calls++
	return fake.resolvedEvidence, nil
}

type activeSnapshotFake struct {
	snapshots []domain.Snapshot
	calls     int
}

func (fake *activeSnapshotFake) ActiveSnapshot(_ context.Context, _ uuid.UUID) (domain.Snapshot, error) {
	if len(fake.snapshots) == 0 {
		return domain.Snapshot{}, errors.New("missing active snapshot")
	}
	index := fake.calls
	if index >= len(fake.snapshots) {
		index = len(fake.snapshots) - 1
	}
	fake.calls++
	return fake.snapshots[index], nil
}

type projectionAppenderFake struct {
	calls      int
	projection domain.Projection
	err        error
}

func (fake *projectionAppenderFake) AppendProjection(_ context.Context, projection domain.Projection) error {
	fake.calls++
	fake.projection = projection
	return fake.err
}

func validServiceInputs() (catalogFake, activeSnapshotFake, projectionAppenderFake, PublishCommand) {
	corpusID, revisionID, snapshotID := fixtureID(1), fixtureID(2), fixtureID(3)
	snapshot := domain.Snapshot{ID: snapshotID, CorpusID: corpusID, ManifestSHA256: checksum("a")}
	english, portuguese := domain.StarterCase{
		ID: fixtureID(10), CorpusID: corpusID, DatasetRevisionID: revisionID, ExternalCaseID: "case-001-en",
		ReciprocalExternalCaseID: "case-001-pt", Checksum: checksum("b"), QueryLanguage: domain.QueryLanguageEnglish,
		Rank: 1, Question: "What is required?",
	}, domain.StarterCase{
		ID: fixtureID(11), CorpusID: corpusID, DatasetRevisionID: revisionID, ExternalCaseID: "case-001-pt",
		ReciprocalExternalCaseID: "case-001-en", Checksum: checksum("c"), QueryLanguage: domain.QueryLanguagePortuguese,
		Rank: 1, Question: "Qual e a exigencia?",
	}
	requirements := []domain.ExpectedEvidenceRequirement{
		{ID: fixtureID(20), CorpusID: corpusID, DatasetRevisionID: revisionID, CaseID: english.ID},
		{ID: fixtureID(21), CorpusID: corpusID, DatasetRevisionID: revisionID, CaseID: portuguese.ID},
	}
	return catalogFake{
			publication: domain.DatasetPublication{ID: fixtureID(40), CorpusID: corpusID, DatasetRevisionID: revisionID,
				DatasetContentSHA256: checksum("d"), ReviewDecision: "approved", PublicationState: "available"},
			starterCases: englishPortuguese(english, portuguese), expectedEvidence: requirements,
			resolvedEvidence: []domain.ExpectedEvidenceResolution{
				{ExpectedEvidenceID: requirements[0].ID, CorpusID: corpusID, DatasetRevisionID: revisionID, CaseID: english.ID, SnapshotID: snapshotID},
				{ExpectedEvidenceID: requirements[1].ID, CorpusID: corpusID, DatasetRevisionID: revisionID, CaseID: portuguese.ID, SnapshotID: snapshotID},
			},
		}, activeSnapshotFake{snapshots: []domain.Snapshot{snapshot}}, projectionAppenderFake{}, PublishCommand{
			CorpusID: corpusID, DatasetRevisionID: revisionID, SnapshotID: snapshotID, SnapshotManifestSHA256: snapshot.ManifestSHA256,
			SelectionPolicyVersion: "v1", PublishedBy: "test-maintainer",
		}
}

func englishPortuguese(english, portuguese domain.StarterCase) []domain.StarterCase {
	return []domain.StarterCase{english, portuguese}
}

func fixtureID(value int) uuid.UUID {
	return uuid.MustParse(fmt.Sprintf("00000000-0000-4000-8000-%012d", value))
}

func checksum(character string) domain.SHA256 { return domain.SHA256(strings.Repeat(character, 64)) }
