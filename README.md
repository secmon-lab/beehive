# Beehive 🐝

An IoC (Indicator of Compromise) management system built with Go and React.

## Features

- **GraphQL API**: Modern API powered by gqlgen
- **React Frontend**: TypeScript-based UI with Apollo Client
- **Clean Architecture**: Following Domain-Driven Design principles
- **Multiple Storage Backends**: Support for Firestore and in-memory storage
- **Single Binary Deployment**: Frontend embedded in Go binary

## Architecture

```
beehive/
├── pkg/
│   ├── cli/              # CLI commands
│   ├── controller/       # HTTP/GraphQL controllers
│   │   ├── graphql/      # GraphQL resolvers
│   │   └── http/         # HTTP server
│   ├── domain/           # Domain layer
│   │   ├── interfaces/   # Repository interfaces
│   │   └── model/        # Domain models
│   ├── repository/       # Data persistence
│   │   ├── firestore/    # Firestore implementation
│   │   └── memory/       # In-memory implementation
│   └── usecase/          # Business logic
├── frontend/             # React application
│   ├── src/
│   │   ├── pages/        # Page components
│   │   ├── App.tsx
│   │   └── main.tsx
│   └── static.go         # Go embed file
└── graphql/              # GraphQL schema
```

## Getting Started

### Prerequisites

- Go 1.24+
- Node.js 18+
- Task (optional, for task automation)

### Development

1. **Install dependencies**:
```bash
go mod download
cd frontend && npm install
```

2. **Generate GraphQL code**:
```bash
go tool gqlgen generate
# or with task
task graphql
```

3. **Build frontend**:
```bash
cd frontend && npm run build
# or with task
task build:frontend
```

4. **Run the server**:
```bash
go run main.go serve
# or build and run
task run
```

5. **Access the application**:
- Frontend: http://localhost:8080
- GraphiQL: http://localhost:8080/graphiql

### Frontend Development

For hot-reload during frontend development:

```bash
# Terminal 1: Run backend
go run main.go serve

# Terminal 2: Run frontend dev server
cd frontend && npm run dev
```

Then access frontend at http://localhost:5173 (Vite dev server).

## Building

### Local Build

```bash
# Build everything
task build

# Or manually
go tool gqlgen generate
cd frontend && npm run build && cd ..
go build -o beehive main.go
```

### Docker Build

```bash
docker build -t beehive:latest .
docker run -p 8080:8080 beehive serve
```

## Configuration

Environment variables:

- `BEEHIVE_ADDR`: HTTP server address (default: `:8080`)
- `BEEHIVE_GRAPHIQL`: Enable GraphiQL playground (default: `true`)

## Development Commands

With Task:

```bash
task                    # Generate GraphQL code
task graphql           # Generate GraphQL code
task mock              # Generate mock files (when interfaces are defined)
task build             # Build application
task build:frontend    # Build frontend only
task run               # Build and run with GraphiQL enabled
task dev:frontend      # Run frontend dev server
```

Without Task:

```bash
go tool gqlgen generate                    # Generate GraphQL code
cd frontend && npm run build              # Build frontend
go build -o beehive main.go               # Build Go binary
./beehive serve                           # Run server
```

## Project Status

This is the initial implementation with the following structure in place:

- ✅ Go project with clean architecture
- ✅ GraphQL API with basic health check
- ✅ React frontend with dashboard
- ✅ HTTP server with embedded static files
- ✅ CLI interface
- ✅ Repository interfaces (empty implementations)
- ✅ Build automation with Taskfile

Future development will add:

- IoC data models (IP addresses, domains, file hashes, URLs, etc.)
- IoC ingestion and management APIs
- Authentication and authorization
- IoC enrichment and threat intelligence integration
- Search and query capabilities
- Dashboard with IoC analytics and visualizations
- Export and integration features

## License

See [LICENSE](LICENSE) file for details.
