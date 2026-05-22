// Package repository hosts compatibility constructors and the shared
// test harness. The two concrete Repository implementations live under
// pkg/repository/memory and pkg/repository/firestore. The harness re-runs
// every test body against both implementations so that they cannot drift
// (see CLAUDE.md §10).
package repository

import (
	"github.com/secmon-lab/beehive/pkg/repository/memory"
)

// Memory is an alias of the in-memory backend, exposed here so tests can
// say `repository.NewMemory()` without importing the concrete package.
type Memory = memory.Memory

// NewMemory returns a fresh in-memory Repository.
func NewMemory() *Memory { return memory.New() }
