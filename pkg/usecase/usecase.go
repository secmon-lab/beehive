// Package usecase wires together the repository, fetcher, extractor
// and source catalog into request-scoped flows. Each use case is a
// small constructor + a method; no global state and no scheduling
// (request-scoped only — see CLAUDE.md §5).
package usecase

import (
	"github.com/secmon-lab/beehive/pkg/domain/interfaces"
	"github.com/secmon-lab/beehive/pkg/service/fetcher"
	"github.com/secmon-lab/beehive/pkg/service/source_catalog"
)

// Deps is what every use case needs to do its job. The struct is small
// so wiring stays explicit at the cli / controller layer.
//
// Registry is the provider registry built at startup (see
// pkg/service/providers.All). It is the only place the fetch use case
// resolves Source.Type to a concrete Provider — there is no package-
// level registry anywhere in the codebase.
type Deps struct {
	Repo      interfaces.Repository
	Lock      interfaces.LockManager
	Extractor interfaces.Extractor
	Catalog   *source_catalog.Catalog
	Registry  *fetcher.Registry
}
