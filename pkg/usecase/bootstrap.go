package usecase

import (
	"context"
	"sync"

	"github.com/m-mizutani/goerr/v2"
	"github.com/secmon-lab/beehive/pkg/domain/interfaces"
	"github.com/secmon-lab/beehive/pkg/service/fetcher"
	"github.com/secmon-lab/beehive/pkg/service/source_catalog"
)

// Bootstrap loads the TOML catalog and runs an orphan-state check. It
// is the only place that does work between "container starts" and "we
// can serve requests".
//
// `Once` is `sync.Once` semantics so calling Ensure from every request
// is cheap after the first.
type Bootstrap struct {
	once    sync.Once
	cat     *source_catalog.Catalog
	loadErr error
}

// NewBootstrap returns a fresh Bootstrap. Pass nil-safe inputs in tests.
func NewBootstrap() *Bootstrap { return &Bootstrap{} }

// Ensure performs the one-time load. After the first call, subsequent
// calls return immediately with the cached catalog.
//
// `reg` is the provider registry built at startup. It is required —
// validation refuses to run without it because every Source.Type must
// be cross-checked against a real provider.
func (b *Bootstrap) Ensure(ctx context.Context, sourcesDir string, repo interfaces.StateRepository, reg *fetcher.Registry) (*source_catalog.Catalog, error) {
	b.once.Do(func() {
		cat, err := source_catalog.Load(sourcesDir)
		if err != nil {
			b.loadErr = goerr.Wrap(err, "load source catalog",
				goerr.V("sources_dir", sourcesDir))
			return
		}
		problems := source_catalog.Validate(cat, reg)
		if pe := source_catalog.AsError(problems); pe != nil {
			b.loadErr = pe
			return
		}
		b.cat = cat
		if repo != nil {
			// Orphan detection is informational — errors should not stop
			// the bootstrap.
			_, _ = source_catalog.DetectOrphanStates(ctx, repo, cat)
		}
	})
	return b.cat, b.loadErr
}

// Catalog returns the cached catalog. May be nil before Ensure runs.
func (b *Bootstrap) Catalog() *source_catalog.Catalog { return b.cat }
