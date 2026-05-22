# syntax=docker/dockerfile:1.7

# ---- Stage 1: build frontend (pnpm) ----
FROM node:22-alpine AS frontend
WORKDIR /app/frontend

RUN corepack enable

COPY frontend/package.json frontend/pnpm-lock.yaml ./
RUN pnpm install --frozen-lockfile

COPY frontend/ ./
RUN pnpm run build

# ---- Stage 2: build go binary ----
FROM golang:1.26-alpine AS backend
WORKDIR /src

ARG BUILD_VERSION=dev

# Copy go module files first for layer caching
COPY go.mod go.sum* ./
RUN go mod download || true

# Copy backend sources
COPY . .

# Bring in the freshly built frontend dist
COPY --from=frontend /app/frontend/dist /src/frontend/dist

RUN CGO_ENABLED=0 GOOS=linux go build -trimpath \
    -ldflags="-s -w -X main.version=${BUILD_VERSION}" \
    -o /out/beehive .

# ---- Stage 3: runtime ----
FROM gcr.io/distroless/static-debian12:nonroot
WORKDIR /app

COPY --from=backend /out/beehive /app/beehive

# Default port (Cloud Run also sets PORT env, but the spec uses BEEHIVE_HTTP_ADDR)
EXPOSE 8080

USER nonroot:nonroot
ENTRYPOINT ["/app/beehive"]
CMD ["serve"]
