package postgres

import (
	"testing"
	"time"

	"github.com/dharlanoliveira/norvii/apps/api/internal/snapshot/domain"
	"github.com/google/uuid"
)

func TestManifestSHA256IsIndependentOfMemberInputOrder(t *testing.T) {
	first := member()
	second := member()
	second.SourceID = uuid.New()

	forward := manifestSHA256([]domain.Member{first, second})
	reversed := manifestSHA256([]domain.Member{second, first})

	if forward != reversed {
		t.Fatalf("manifest hashes = %s/%s, want stable ordering", forward, reversed)
	}
}

func TestReplaceMemberKeepsAllOtherSnapshotMembers(t *testing.T) {
	first := member()
	second := member()
	second.SourceID = uuid.New()
	replacement := first
	replacement.DocumentID = uuid.New()

	updated := replaceMember([]domain.Member{first, second}, replacement)

	if len(updated) != 2 || updated[0].DocumentID != replacement.DocumentID || updated[1] != second {
		t.Fatalf("replaceMember() = %+v, want replacement and preserved member", updated)
	}
}

func member() domain.Member {
	return domain.Member{
		SourceID: uuid.New(), SourceRevisionID: uuid.New(), DocumentID: uuid.New(),
		OfficialOrigin: "https://example.org/law", CapturedAt: time.Date(2026, time.August, 24, 12, 0, 0, 0, time.UTC),
		ContentSHA256: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
	}
}
