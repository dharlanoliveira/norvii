package persistence

import (
	"context"
	"errors"
	"fmt"
	"time"
)

const storeCloseTimeout = 2 * time.Second

type persistenceStore interface {
	Name() string
	Verify(context.Context) error
	Close(context.Context) error
}

// VerificationResult identifies one successfully checked persistence store.
type VerificationResult struct {
	Store    string
	Duration time.Duration
}

// Verifier checks an ordered set of stores and owns their cleanup.
type Verifier struct {
	stores []persistenceStore
}

// NewVerifier creates a one-shot verifier for the provided stores.
func NewVerifier(stores ...persistenceStore) *Verifier {
	return &Verifier{stores: stores}
}

// Verify checks each store in order and closes every store in reverse order.
func (verifier *Verifier) Verify(ctx context.Context) (results []VerificationResult, returnedError error) {
	defer func() {
		for index := len(verifier.stores) - 1; index >= 0; index-- {
			closeCtx, cancel := context.WithTimeout(context.Background(), storeCloseTimeout)
			closeErr := verifier.stores[index].Close(closeCtx)
			cancel()
			if closeErr != nil {
				returnedError = errors.Join(
					returnedError,
					fmt.Errorf("close %s persistence connection: %w", verifier.stores[index].Name(), closeErr),
				)
			}
		}
	}()

	results = make([]VerificationResult, 0, len(verifier.stores))
	for _, store := range verifier.stores {
		startedAt := time.Now()
		if err := store.Verify(ctx); err != nil {
			return results, fmt.Errorf("verify %s connectivity: %w", store.Name(), err)
		}
		results = append(results, VerificationResult{Store: store.Name(), Duration: time.Since(startedAt)})
	}
	return results, nil
}
