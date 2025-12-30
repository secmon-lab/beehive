package graphql

import (
	"github.com/secmon-lab/beehive/pkg/domain/interfaces"
	"github.com/secmon-lab/beehive/pkg/usecase"
)

// This file will not be regenerated automatically.
//
// It serves as dependency injection for your app, add any dependencies you require here.

type Resolver struct {
	repo              interfaces.Repository
	uc                *usecase.UseCases
	sourcesConfigPath string
}

func NewResolver(repo interfaces.Repository, uc *usecase.UseCases, sourcesConfigPath string) *Resolver {
	return &Resolver{
		repo:              repo,
		uc:                uc,
		sourcesConfigPath: sourcesConfigPath,
	}
}

// Repository returns the repository instance
func (r *Resolver) Repository() interfaces.Repository {
	return r.repo
}
