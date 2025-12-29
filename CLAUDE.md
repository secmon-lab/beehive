# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Beehive is an IoC (Indicator of Compromise) management system built with Go and React. The system collects IoCs from multiple sources including security blog RSS feeds and threat intelligence feeds, using LLM to extract indicators from unstructured text and storing them with semantic embeddings for advanced search capabilities.

### Core Capabilities

- **Automated IoC Collection**: Fetches data from blog RSS feeds and threat intelligence feeds (e.g., abuse.ch)
- **LLM-Powered Extraction**: Uses Gemini/OpenAI to extract IoCs from blog articles and security reports
- **Semantic Search**: Stores IoCs with vector embeddings for similarity-based search
- **Multi-Source Tracking**: Tracks IoCs across different sources with status management (active/inactive)
- **GraphQL API**: Provides a type-safe API for querying and managing IoCs
- **Web Interface**: React-based frontend for visualizing and analyzing threat intelligence

## Common Development Commands

### Building and Testing
- `task` - Run default task (GraphQL code generation)
- `task build` - Build the complete application (frontend + backend)
- `task build:frontend` - Build frontend only
- `task graphql` - Generate GraphQL code from schema
- `task run` - Build and run the server
- `task dev:frontend` - Run frontend development server
- `go build` - Build the main binary
- `go test ./...` - Run all tests
- `go test ./pkg/path/to/package` - Run tests for specific package

### Code Generation
- `go tool gqlgen generate` - Generate GraphQL resolvers and types from schema
- Mock generation planned for future when more interfaces are defined

## Important Development Guidelines

### Error Handling
- Use `github.com/m-mizutani/goerr/v2` for error handling
- Must wrap errors with `goerr.Wrap` to maintain error context
- Add helpful variables with `goerr.V` for debugging
- **NEVER check error messages using `strings.Contains(err.Error(), ...)`**
- **ALWAYS use `errors.Is(err, targetErr)` or `errors.As(err, &target)` for error type checking**
- Error discrimination must be done by error types, not by parsing error messages

### Testing Best Practices

**IMPORTANT**: All tests must use exact expected values. Never test "is not empty" or "count > 0". Always specify the exact values you expect.

#### Using the gt Library

We use `github.com/m-mizutani/gt` for all test assertions. This library provides type-safe, fluent assertions with generics.

**Core Principles**:
1. **Always test exact expected values** - Never use `gt.True(t, x != "")` or `gt.True(t, len(x) > 0)`
2. **Use appropriate gt methods** - `gt.S()` for strings, `gt.N()` for numbers, `gt.V()` for generic values, `gt.A()` for arrays
3. **Add descriptions** - Use `.Describe()` or `.Describef()` to explain what's being tested
4. **Use callbacks for array elements** - When testing array elements, use `.At(index, func(t testing.TB, v T) {...})`

**Good Examples**:
```go
// ✅ GOOD: Test exact string value
gt.S(t, article.Title).Equal("Denmark Accuses Russia of Conducting Two Cyberattacks").Describe("first article title")

// ✅ GOOD: Test exact array length
gt.A(t, entries).Length(11).Describe("should parse all 11 URLhaus entries from real data")

// ✅ GOOD: Test array element with callback
gt.A(t, entries).At(0, func(t testing.TB, first *feed.FeedEntry) {
    gt.V(t, first.ID).Equal("3741935").Describe("first entry ID")
    gt.V(t, first.Type).Equal(model.IoCTypeURL).Describe("first entry type")
})

// ✅ GOOD: Test exact number value
gt.N(t, stats.ItemsFetched).Equal(10).Describe("items fetched should be 10")

// ✅ GOOD: Test with Contains for partial string match
gt.S(t, first.Description).Contains("malware_download").Describe("description should contain threat type")

// ✅ GOOD: Error assertions
gt.NoError(t, err) // No .Describe() on NoError/Error
gt.Error(t, err)
```

**Bad Examples (FORBIDDEN)**:
```go
// ❌ BAD: Testing "not empty"
gt.False(t, article.Title == "")
gt.True(t, article.Title != "")

// ❌ BAD: Testing "count > 0"
gt.True(t, len(articles) > 0)
gt.True(t, len(articles) >= 3)

// ❌ BAD: Using .At() without callback
first := gt.Array(t, entries).At(0).Required()
gt.V(t, first.ID).Equal("123") // Wrong - .At() requires a callback

// ❌ BAD: Using .Describe() on NoError/Error
gt.NoError(t, err).Describe("should not error") // Error: NoErrorTest has no method Describe
gt.Error(t, err).Describe("should error")       // Error: ErrorTest has no method Describe
```

#### gt Method Reference

- **`gt.V(t, value)`** - Generic value assertions for any type
  - `.Equal(expected)` - Test equality
  - `.NotEqual(unexpected)` - Test inequality
  - `.NotNil()` / `.Nil()` - Test nil/non-nil

- **`gt.S(t, string)`** - String-specific assertions
  - `.Equal(expected)` - Exact string match
  - `.Contains(substring)` - Substring match
  - `.HasPrefix(prefix)` / `.HasSuffix(suffix)` - Prefix/suffix match

- **`gt.N(t, number)`** - Number assertions
  - `.Equal(expected)` - Exact number match
  - `.Greater(min)` / `.Less(max)` - Comparison assertions
  - `.GreaterOrEqual(min)` / `.LessOrEqual(max)`

- **`gt.A(t, array)`** - Array/slice assertions
  - `.Length(n)` - Exact length match
  - `.At(index, func(t testing.TB, v T) {...})` - Test element at index with callback
  - `.Has(value)` - Check if array contains value

- **`gt.NoError(t, err)`** - Assert no error (does NOT support `.Describe()`)
- **`gt.Error(t, err)`** - Assert error exists (does NOT support `.Describe()`)

- **`.Describe(msg)`** - Add description to any assertion (except NoError/Error)
- **`.Describef(format, args...)`** - Add formatted description

#### Test Data Management

- **Use Real Data**: Always fetch real data from actual sources and store in `testdata/` directory
- **Never Modify Source Data**: If data format doesn't match, fix the parser code, not the test data
- **Use `//go:embed`**: Embed test data files for reliability
- **Exact Values in Tests**: Write exact expected values from the embedded real data

Example:
```go
//go:embed testdata/urlhaus_sample.csv
var urlhausSampleData []byte

func TestParsing(t *testing.T) {
    // Use real embedded data
    entries, err := ParseData(urlhausSampleData)
    gt.NoError(t, err)

    // Test exact values from the real data
    gt.A(t, entries).Length(11).Describe("should parse all 11 entries")
    gt.A(t, entries).At(0, func(t testing.TB, first *Entry) {
        gt.V(t, first.ID).Equal("3741935").Describe("first entry ID from real data")
        gt.V(t, first.URL).Equal("https://sivqen.a8riculmarb1e.ru/0dh149h0").Describe("first entry URL from real data")
    })
}
```

#### Repository Testing

- Use Memory repository from `pkg/repository/memory` for repository testing
- Test both memory and firestore implementations when applicable
- **CRITICAL: Firestore tests do NOT clean up data** (cost/performance reasons)
  - **ALWAYS use timestamp-based unique VALUES** to avoid ID conflicts across test runs
  - IoC values MUST be unique (not just sourceID/contextKey)
  - Example:
    ```go
    sourceID := time.Now().Format("source-20060102-150405.000000")
    value := time.Now().Format("192.168.1.150405000")  // Value must be unique!
    contextKey := model.IoCContextKey(time.Now().Format("entry-20060102-150405.000000"))
    ```
  - Wrong: Using static values like `"192.168.1.1"` - will conflict on repeated test runs
  - Right: Using timestamp-based values that are always unique

### Code Visibility
- Do not expose unnecessary methods, variables and types
- Use `export_test.go` to expose items needed only for testing

## Architecture

### Core Structure
The application follows Domain-Driven Design (DDD) with clean architecture:

- `pkg/cli/` - CLI commands and configuration
- `pkg/controller/` - Interface adapters
  - `graphql/` - GraphQL resolvers
  - `http/` - HTTP server and routing
- `pkg/domain/` - Domain layer
  - `interfaces/` - Repository and service interfaces
  - `model/` - Domain models (IoC data structures)
- `pkg/repository/` - Data persistence implementations
  - `firestore/` - Firestore backend
  - `memory/` - In-memory backend (testing/development)
- `pkg/usecase/` - Application use cases orchestrating domain operations
- `pkg/utils/` - Shared utilities (logging, etc.)
- `frontend/` - React TypeScript application
- `graphql/` - GraphQL schema definitions

### Key Components

#### GraphQL API
- Schema-first design using gqlgen
- GraphQL playground available at `/graphiql` (configurable)
- Type-safe resolvers in `pkg/controller/graphql/`

#### Frontend
- React with TypeScript
- Vite for development and building
- pnpm for package management (faster and more efficient than npm)
- Apollo Client for GraphQL integration
- Embedded into Go binary via `//go:embed`
- Development mode: Hot reload on port 5173
- Production mode: Served from embedded files

#### Storage Backends
- **Firestore**: Production-ready persistent storage
- **Memory**: In-memory storage for testing and development
- Repository pattern allows easy switching between backends
- Interface defined in `pkg/domain/interfaces/`

### Application Modes
- `serve` - HTTP server mode with GraphQL API and frontend

### Future Features (Planned)
The following features are planned but not yet implemented:
- IoC data models (IP addresses, domains, file hashes, URLs)
- IoC ingestion and management APIs
- Authentication and authorization
- Threat intelligence integration and enrichment
- Search and query capabilities
- Dashboard analytics and visualizations
- Export and integration features

## Configuration

The application is configured via CLI flags or environment variables:

- `BEEHIVE_ADDR` - HTTP server address (default: `:8080`)
- `BEEHIVE_GRAPHIQL` - Enable GraphiQL playground (default: `true`)
- Logger configuration (format, level, output destination)

## Testing

Test files follow Go conventions (`*_test.go`). The codebase includes:
- Unit tests for individual components
- Integration tests with repository implementations
- Repository tests use both memory and firestore backends for verification
- E2E tests for threat intelligence feeds (controlled by `TEST_E2E` environment variable)

### Running Tests

```bash
# Run all unit tests
zenv go test ./...

# Run E2E tests (fetches data from live threat intelligence feeds)
TEST_E2E=1 zenv go test ./pkg/service/feed -timeout 5m

# Run specific E2E test
TEST_E2E=1 zenv go test ./pkg/service/feed -run TestService_FetchAbuseCHURLhaus_E2E -v
```

### E2E Test Pattern

E2E tests check the `TEST_E2E` environment variable to determine whether to run:

```go
func TestService_FetchFeed_E2E(t *testing.T) {
    if os.Getenv("TEST_E2E") == "" {
        t.Skip("E2E test skipped (set TEST_E2E=1 to run)")
    }
    // ... actual test code
}
```

- E2E tests fetch data from actual live feeds to verify parsers work with real data
- Use environment variable (not CLI flag) so `go test ./...` works across all packages
- E2E tests should verify minimal data integrity (e.g., successful fetch, basic format validation)

## Restrictions and Rules

### Directory

- When you are mentioned about `tmp` directory, you SHOULD NOT see `/tmp`. You need to check `./tmp` directory from root of the repository.

### Exposure policy

In principle, do not trust developers who use this library from outside

- Do not export unnecessary methods, structs, and variables
- Assume that exposed items will be changed. Never expose fields that would be problematic if changed
- Use `export_test.go` for items that need to be exposed for testing purposes

### Check

When making changes, before finishing the task, always:
- Run `go vet ./...`, `go fmt ./...` to format the code
- Run `golangci-lint run ./...` to check lint error
- Run `gosec -exclude-generated -quiet ./...` to check security issue
- Run tests to ensure no impact on other code

### Language

All comment and character literal in source code must be in English

### Struct Tags

**NEVER use Firestore struct tags** - The repository layer handles field mapping, not the domain models.

- ❌ BAD: `type IoC struct { ID string `firestore:"id"` }`
- ✅ GOOD: `type IoC struct { ID string }`

Rationale:
- Domain models should be infrastructure-agnostic
- Firestore tags couple domain layer to infrastructure layer
- Repository implementations handle serialization/deserialization

### Testing

- Test files should have `package {name}_test`. Do not use same package name
- Test file name convention is: `xyz.go` → `xyz_test.go`. Other test file names (e.g., `xyz_e2e_test.go`) are not allowed.
- Repository Tests Location:
  - NEVER create test files in `pkg/repository/firestore/` or `pkg/repository/memory/` subdirectories
  - ALL repository tests MUST be placed directly in `pkg/repository/*_test.go`
  - Use `runRepositoryTest()` helper to test against both memory and firestore implementations
- Repository Tests Best Practices:
  - Always use random IDs (e.g., using `time.Now().UnixNano()`) to avoid test conflicts
  - Never use hardcoded IDs like "msg-001", "user-001" as they cause test failures when running in parallel
  - Always verify ALL fields of returned values, not just checking for nil/existence
  - Compare expected values properly - don't just check if something exists, verify it matches what was saved
  - For timestamp comparisons, use tolerance (e.g., `< time.Second`) to account for storage precision
- Test Skip Policy:
  - **NEVER use `t.Skip()` for anything other than missing environment variables**
  - If a test requires infrastructure (like Firestore index), fix the infrastructure, don't skip the test
  - If a feature is not implemented, write the code, don't skip the test
  - The only acceptable skip pattern: checking for missing environment variables at the beginning of a test

### Test File Checklist (Use this EVERY time)
Before creating or modifying tests:
1. ✓ Is there a corresponding source file for this test file?
2. ✓ Does the test file name match exactly? (`xyz.go` → `xyz_test.go`)
3. ✓ Are all tests for a source file in ONE test file?
4. ✓ No standalone feature/e2e/integration test files?
5. ✓ For repository tests: placed in `pkg/repository/*_test.go`, NOT in firestore/ or memory/ subdirectories?

### Feed Implementation Testing Requirements

**CRITICAL**: When implementing new threat intelligence feeds in `pkg/service/feed/`, you MUST follow these testing requirements:

#### E2E Tests (with `-e2e` flag)
- **MUST implement E2E tests** that download data from the actual feed URL
- E2E tests verify that data is correctly fetched and parsed from the live source
- Use build tag or environment variable check to conditionally run E2E tests:
  ```go
  func TestFetchXXX_E2E(t *testing.T) {
      if os.Getenv("E2E") == "" {
          t.Skip("E2E test skipped (set E2E=1 to run)")
      }
      // Test with actual URL
  }
  ```
- E2E tests should verify minimum data integrity (e.g., entry count > 0, correct format)

#### Unit Tests with Embedded Test Data
- **MUST create test data files** in `pkg/service/feed/testdata/` directory
- Test data MUST be based on real feed data but with **dummy values**
- **NEVER commit actual feed data** to avoid redistribution issues
- Steps to create test data:
  1. Download actual data from the feed URL
  2. Understand the data format and structure
  3. Create sample data (10-20 entries) with the same format
  4. Replace all actual values with dummy data:
     - IPs: Use private IP ranges (192.168.x.x, 10.x.x.x) or documentation IPs (192.0.2.x, 198.51.100.x)
     - Domains: Use example.com, example.org, test.local, etc.
     - Hashes: Generate random hex strings
     - URLs: Use dummy domains with realistic paths
  5. Save to `testdata/{feed_name}_sample.{ext}`
- Use `//go:embed` to embed test data in tests:
  ```go
  //go:embed testdata/example_feed_sample.csv
  var exampleFeedSampleData []byte
  ```
- Unit tests MUST verify exact expected values from the dummy data
- Unit tests MUST use httptest server to serve embedded test data

#### E2E Test Flag Pattern

E2E tests MUST use the `-e2e` flag pattern with the `flag` package:

```go
package feed_test

import (
    "context"
    "flag"
    "testing"

    "github.com/m-mizutani/gt"
    "github.com/secmon-lab/beehive/pkg/service/feed"
)

// Define the flag at package level
var e2e = flag.Bool("e2e", false, "run E2E tests against live feeds")

// E2E test with actual URL
func TestE2E_FetchExampleFeed(t *testing.T) {
    if !*e2e {
        t.Skip("E2E test skipped (use -e2e flag to run)")
    }

    ctx := context.Background()
    svc := feed.New()
    entries, err := svc.FetchExampleFeed(ctx, "https://actual-feed-url.example.com/feed.txt")
    gt.NoError(t, err)

    // Verify minimum data integrity
    gt.True(t, len(entries) > 0).Describe("should fetch at least one entry from live feed")
}
```

Run E2E tests with: `go test -e2e ./pkg/service/feed -v`

#### Example Structure
```go
package feed_test

import (
    _ "embed"
    "context"
    "net/http"
    "net/http/httptest"
    "testing"

    "github.com/m-mizutani/gt"
    "github.com/secmon-lab/beehive/pkg/service/feed"
)

//go:embed testdata/example_feed_sample.txt
var exampleFeedSampleData []byte

// Unit test with embedded dummy data
func TestService_FetchExampleFeed(t *testing.T) {
    ctx := context.Background()
    server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        w.WriteHeader(http.StatusOK)
        _, _ = w.Write(exampleFeedSampleData)
    }))
    defer server.Close()

    svc := feed.New()
    entries, err := svc.FetchExampleFeed(ctx, server.URL)
    gt.NoError(t, err)

    // Test exact values from dummy data
    gt.A(t, entries).Length(10).Describe("should parse all 10 entries")
    gt.A(t, entries).At(0, func(t testing.TB, first *feed.FeedEntry) {
        gt.V(t, first.Value).Equal("192.0.2.1").Describe("first entry IP from dummy data")
    })
}
```

#### Rationale
- **Real data testing**: Ensures parser works with actual feed formats
- **Dummy data in repo**: Avoids legal/licensing issues with redistributing threat intelligence data
- **E2E tests**: Validates that feeds are still accessible and haven't changed format
- **Exact value testing**: Catches parser bugs and format changes immediately
