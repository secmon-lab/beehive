// Package fetcher hosts the provider Registry and the shared
// SSRF-protected HTTP client.
//
// There is intentionally NO global registry and NO init()-time side
// effects in this package. Providers are constructed by name in
// pkg/service/providers and handed to a *Registry at startup; the CLI
// / usecase layer carries the Registry through usecase.Deps.
//
// See CLAUDE.md §4 for how to add a new provider.
package fetcher
