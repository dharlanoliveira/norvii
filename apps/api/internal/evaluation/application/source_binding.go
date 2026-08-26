package application

import (
	"context"
	"errors"
	"regexp"
	"strings"

	"github.com/google/uuid"
)

// ErrInvalidSourceBinding identifies a malformed maintainer source-binding request.
var ErrInvalidSourceBinding = errors.New("evaluation dataset source binding is invalid")

var sourceAliasPattern = regexp.MustCompile(`^[a-z0-9][a-z0-9-]{0,127}$`)

// SourceBinding fixes one manifest source alias to one source in the same dataset corpus.
type SourceBinding struct {
	DatasetRevisionID uuid.UUID
	CorpusID          uuid.UUID
	SourceAlias       string
	CorpusSourceID    uuid.UUID
}

// SourceBinder persists a source alias exactly once. Implementations must reject foreign corpus
// sources and may not update a prior binding.
type SourceBinder interface {
	BindDatasetSource(context.Context, SourceBinding) (SourceBinding, error)
}

// SourceBindingService validates the maintainer boundary before immutable binding persistence.
type SourceBindingService struct{ binder SourceBinder }

// NewSourceBindingService constructs source binding around caller-owned persistence.
func NewSourceBindingService(binder SourceBinder) *SourceBindingService {
	return &SourceBindingService{binder: binder}
}

// Bind rejects malformed input before it reaches persistence and returns the persisted binding.
func (service *SourceBindingService) Bind(ctx context.Context, binding SourceBinding) (SourceBinding, error) {
	if service == nil || service.binder == nil || binding.DatasetRevisionID == uuid.Nil || binding.CorpusID == uuid.Nil ||
		binding.CorpusSourceID == uuid.Nil || binding.SourceAlias != strings.TrimSpace(binding.SourceAlias) ||
		!sourceAliasPattern.MatchString(binding.SourceAlias) {
		return SourceBinding{}, ErrInvalidSourceBinding
	}
	return service.binder.BindDatasetSource(ctx, binding)
}
