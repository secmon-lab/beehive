// Package http hosts the chi router and request handlers. We do not
// (yet) generate code from openapi.yaml — the handler set is small
// enough that hand-written wrappers keep the dependency graph simple.
// `task gen:openapi` will eventually populate `generated.go` when we
// reach for richer request validation.
package http

import (
	"context"
	"encoding/json"
	"io/fs"
	"log/slog"
	"net/http"
	"runtime/debug"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/m-mizutani/goerr/v2"
	"github.com/secmon-lab/beehive/frontend"
	"github.com/secmon-lab/beehive/pkg/domain/interfaces"
	"github.com/secmon-lab/beehive/pkg/domain/model"
	"github.com/secmon-lab/beehive/pkg/domain/types"
	"github.com/secmon-lab/beehive/pkg/service/source_catalog"
	"github.com/secmon-lab/beehive/pkg/usecase"
	"github.com/secmon-lab/beehive/pkg/utils/async"
	"github.com/secmon-lab/beehive/pkg/utils/errutil"
	"github.com/secmon-lab/beehive/pkg/utils/id"
	"github.com/secmon-lab/beehive/pkg/utils/logging"
	"github.com/secmon-lab/beehive/pkg/utils/safe"
)

// Version is injected by main.go before the router is built.
var Version = "dev"

// Deps is the use case dependencies the handlers need. Constructed in
// pkg/cli/serve.go and passed in here.
type Deps = usecase.Deps

// Server bundles the chi router and the use case dependencies.
type Server struct {
	Router  chi.Router
	Deps    Deps
	Catalog *source_catalog.Catalog
	Static  fs.FS
}

// New builds the router with the full set of v1 endpoints. The optional
// `deps` may be empty (Deps{}); handlers that require state then return
// `503 Service Unavailable`.
//
// Unless WithStaticFS overrides it, the embedded React build under
// frontend/dist is mounted at "/" with a catch-all SPA handler. chi's
// radix tree prefers the static "/api/v1" prefix over the "/*"
// wildcard, so API 404s still get the JSON problem response.
func New(opts ...Option) *Server {
	s := &Server{Static: defaultStaticFS()}
	for _, o := range opts {
		o(s)
	}
	r := chi.NewRouter()
	r.Use(middleware.RealIP)
	r.Use(requestLogger)
	r.Use(recoverer)

	r.Route("/api/v1", func(r chi.Router) {
		r.Get("/health", s.getHealth)
		r.Get("/sources", s.listSources)
		r.Get("/sources/{id}", s.getSource)
		r.Put("/sources/{id}/enabled_override", s.setEnabledOverride)
		r.Post("/sources/{id}/fetch", s.triggerSourceFetch)
		r.Post("/fetch", s.triggerFetchAll)
		r.Get("/runs", s.listRuns)
		r.Get("/runs/{id}", s.getRun)
		r.Get("/iocs", s.listRecentIoCs)
		r.Get("/iocs/stats", s.iocStats)
		r.Get("/iocs/lookup", s.lookupIoC)
	})

	if s.Static != nil {
		r.Get("/*", spaHandler(s.Static))
	}

	r.NotFound(notFound)
	s.Router = r
	return s
}

// defaultStaticFS returns the embedded frontend/dist sub-FS when it
// actually contains a built React app (i.e. index.html is present).
// In fresh checkouts where `pnpm build` has not run, dist/ holds only
// the .gitkeep placeholder; we return nil so New() simply skips the
// catch-all mount and unit tests stay focused on the API surface.
func defaultStaticFS() fs.FS {
	sub, err := fs.Sub(frontend.StaticFiles, "dist")
	if err != nil {
		return nil
	}
	if _, err := fs.Stat(sub, "index.html"); err != nil {
		return nil
	}
	return sub
}

// Option configures the Server during construction. Passed to New().
type Option func(*Server)

func WithDeps(d Deps) Option { return func(s *Server) { s.Deps = d } }
func WithCatalog(c *source_catalog.Catalog) Option {
	return func(s *Server) { s.Catalog = c }
}

// WithStaticFS replaces the default embedded React build with an
// arbitrary filesystem. Production callers never need this — the
// default already wires frontend.StaticFiles. It exists so tests can
// inject an in-memory fstest.MapFS without depending on `pnpm build`.
// Pass nil to disable the catch-all mount entirely.
func WithStaticFS(fsys fs.FS) Option {
	return func(s *Server) { s.Static = fsys }
}

// ---- handlers ----

func (s *Server) getHealth(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{
		"status":  "ok",
		"version": Version,
	})
}

func (s *Server) listSources(w http.ResponseWriter, r *http.Request) {
	if s.Catalog == nil || s.Deps.Repo == nil {
		writeProblem(w, http.StatusServiceUnavailable, "not_ready", "catalog not initialised", nil)
		return
	}
	views, err := usecase.ListSources(r.Context(), s.Catalog, s.Deps.Repo)
	if err != nil {
		writeErr(r.Context(), w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"sources": toSourceViewPayload(views)})
}

func (s *Server) getSource(w http.ResponseWriter, r *http.Request) {
	if s.Catalog == nil || s.Deps.Repo == nil {
		writeProblem(w, http.StatusServiceUnavailable, "not_ready", "catalog not initialised", nil)
		return
	}
	id := types.SourceID(chi.URLParam(r, "id"))
	view, err := usecase.GetSource(r.Context(), s.Catalog, s.Deps.Repo, id)
	if err != nil {
		writeErr(r.Context(), w, err)
		return
	}
	writeJSON(w, http.StatusOK, toSourceViewPayload([]*usecase.SourceView{view})[0])
}

func (s *Server) setEnabledOverride(w http.ResponseWriter, r *http.Request) {
	if s.Deps.Repo == nil {
		writeProblem(w, http.StatusServiceUnavailable, "not_ready", "repository not initialised", nil)
		return
	}
	var body struct {
		Value string `json:"value"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeProblem(w, http.StatusBadRequest, "invalid_input", "request body is not JSON", nil)
		return
	}
	id := types.SourceID(chi.URLParam(r, "id"))
	if err := usecase.SetEnabledOverride(r.Context(), s.Deps.Repo, id, types.EnabledOverride(body.Value)); err != nil {
		writeErr(r.Context(), w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) triggerSourceFetch(w http.ResponseWriter, r *http.Request) {
	if s.Catalog == nil {
		writeProblem(w, http.StatusServiceUnavailable, "not_ready", "catalog not initialised", nil)
		return
	}
	id := types.SourceID(chi.URLParam(r, "id"))
	if _, ok := s.Catalog.ByID(id); !ok {
		writeProblem(w, http.StatusNotFound, "not_found", "source not found", map[string]any{"id": id})
		return
	}
	run, err := usecase.FetchAll(r.Context(), depsWithCatalog(s.Deps, s.Catalog), "manual", id)
	if err != nil {
		writeErr(r.Context(), w, err)
		return
	}
	writeJSON(w, http.StatusOK, runSummary(run))
}

func (s *Server) triggerFetchAll(w http.ResponseWriter, r *http.Request) {
	if s.Catalog == nil {
		writeProblem(w, http.StatusServiceUnavailable, "not_ready", "catalog not initialised", nil)
		return
	}
	mode := r.URL.Query().Get("mode")

	// De-duplicate concurrent fetch calls (Cloud Scheduler retries, UI
	// double-clicks): if a Run is still in progress, hand the caller
	// its id instead of starting another one. The check is best-effort
	// — a true race between two near-simultaneous requests still ends
	// up with two Runs, which CLAUDE.md §6 (idempotent fetches)
	// explicitly accepts.
	if s.Deps.Repo != nil {
		recents, err := usecase.ListRecentRuns(r.Context(), s.Deps.Repo, 10)
		if err != nil {
			writeErr(r.Context(), w, err)
			return
		}
		for _, existing := range recents {
			if existing.Status == types.RunStatusRunning {
				writeJSON(w, http.StatusAccepted, map[string]any{
					"runId":      existing.ID,
					"mode":       mode,
					"inProgress": true,
				})
				return
			}
		}
	}

	if mode == "async" {
		// Pre-create the Run so the client can poll it immediately.
		if s.Deps.Repo == nil {
			writeProblem(w, http.StatusServiceUnavailable, "not_ready", "repository not initialised", nil)
			return
		}
		run := &model.Run{
			ID:        types.RunID("run-" + id.NewULID()),
			Trigger:   "api",
			Status:    types.RunStatusRunning,
			StartedAt: time.Now().UTC(),
		}
		if err := s.Deps.Repo.CreateRun(r.Context(), run); err != nil {
			writeErr(r.Context(), w, err)
			return
		}
		// Detach: HTTP request will be done by the time fetch finishes.
		// Pull the logger across so log lines stay attributable.
		bgCtx := logging.With(context.Background(), logging.From(r.Context()))
		logging.From(r.Context()).LogAttrs(r.Context(), slog.LevelInfo, "fetch all: dispatched",
			slog.String("run_id", string(run.ID)),
			slog.String("mode", "async"),
		)
		async.DispatchDetached(bgCtx, func(ctx context.Context) error {
			_, err := usecase.FetchAll(
				ctx,
				depsWithCatalog(s.Deps, s.Catalog),
				"api",
				"",
				usecase.WithRunID(run.ID),
			)
			return err
		})
		writeJSON(w, http.StatusAccepted, map[string]any{
			"runId": run.ID,
			"mode":  "async",
		})
		return
	}

	// Default: synchronous fetch (Cloud Scheduler entry path).
	run, err := usecase.FetchAll(r.Context(), depsWithCatalog(s.Deps, s.Catalog), "api", "")
	if err != nil {
		writeErr(r.Context(), w, err)
		return
	}
	writeJSON(w, http.StatusOK, runSummary(run))
}

func (s *Server) getRun(w http.ResponseWriter, r *http.Request) {
	if s.Deps.Repo == nil {
		writeProblem(w, http.StatusServiceUnavailable, "not_ready", "repository not initialised", nil)
		return
	}
	id := types.RunID(chi.URLParam(r, "id"))
	run, err := s.Deps.Repo.GetRun(r.Context(), id)
	if err != nil {
		writeErr(r.Context(), w, err)
		return
	}
	writeJSON(w, http.StatusOK, runPayload(run))
}

func (s *Server) listRuns(w http.ResponseWriter, r *http.Request) {
	if s.Deps.Repo == nil {
		writeProblem(w, http.StatusServiceUnavailable, "not_ready", "repository not initialised", nil)
		return
	}
	limit := 0
	if v := r.URL.Query().Get("limit"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil || n < 0 {
			writeProblem(w, http.StatusBadRequest, "invalid_input", "limit must be a non-negative integer", nil)
			return
		}
		limit = n
	}
	runs, err := usecase.ListRecentRuns(r.Context(), s.Deps.Repo, limit)
	if err != nil {
		writeErr(r.Context(), w, err)
		return
	}
	payload := make([]map[string]any, 0, len(runs))
	for _, run := range runs {
		payload = append(payload, runPayload(run))
	}
	writeJSON(w, http.StatusOK, map[string]any{"runs": payload})
}

func (s *Server) listRecentIoCs(w http.ResponseWriter, r *http.Request) {
	if s.Deps.Repo == nil {
		writeProblem(w, http.StatusServiceUnavailable, "not_ready", "repository not initialised", nil)
		return
	}
	q := r.URL.Query()
	limit := 0
	if v := q.Get("limit"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil || n < 0 {
			writeProblem(w, http.StatusBadRequest, "invalid_input", "limit must be a non-negative integer", nil)
			return
		}
		limit = n
	}
	var after *time.Time
	if v := q.Get("after"); v != "" {
		t, err := time.Parse(time.RFC3339Nano, v)
		if err != nil {
			writeProblem(w, http.StatusBadRequest, "invalid_input", "after must be an RFC 3339 timestamp", nil)
			return
		}
		after = &t
	}
	iocs, err := usecase.ListRecentIoCsAfter(r.Context(), s.Deps.Repo, limit, after)
	if err != nil {
		writeErr(r.Context(), w, err)
		return
	}
	payload := make([]map[string]any, 0, len(iocs))
	for _, i := range iocs {
		payload = append(payload, iocPayload(i))
	}
	resp := map[string]any{"iocs": payload}
	// Only emit nextCursor when the page is full — otherwise the
	// client knows it has reached the tail.
	if limit > 0 && len(iocs) == limit && len(iocs) > 0 {
		resp["nextCursor"] = iocs[len(iocs)-1].LastSeenAt.Format(time.RFC3339Nano)
	}
	writeJSON(w, http.StatusOK, resp)
}

func (s *Server) iocStats(w http.ResponseWriter, r *http.Request) {
	if s.Deps.Repo == nil {
		writeProblem(w, http.StatusServiceUnavailable, "not_ready", "repository not initialised", nil)
		return
	}
	counts, err := usecase.GetIoCCounts(r.Context(), s.Deps.Repo)
	if err != nil {
		writeErr(r.Context(), w, err)
		return
	}
	byType := make(map[string]int64, len(counts.ByType))
	for k, v := range counts.ByType {
		byType[string(k)] = v
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"total":     counts.Total,
		"byType":    byType,
		"updatedAt": counts.UpdatedAt,
	})
}

func (s *Server) lookupIoC(w http.ResponseWriter, r *http.Request) {
	if s.Deps.Repo == nil {
		writeProblem(w, http.StatusServiceUnavailable, "not_ready", "repository not initialised", nil)
		return
	}
	q := r.URL.Query()
	t := types.IoCType(q.Get("type"))
	v := q.Get("value")
	if !t.Valid() || v == "" {
		writeProblem(w, http.StatusBadRequest, "invalid_input", "type and value query params are required", nil)
		return
	}
	ioc, err := usecase.LookupIoC(r.Context(), s.Deps.Repo, t, v)
	if err != nil {
		writeErr(r.Context(), w, err)
		return
	}
	writeJSON(w, http.StatusOK, iocPayload(ioc))
}

// ---- payload helpers (snake_case omitted: JSON uses camelCase) ----

func toSourceViewPayload(views []*usecase.SourceView) []map[string]any {
	out := make([]map[string]any, 0, len(views))
	for _, v := range views {
		src := map[string]any{
			"id":       v.Source.ID,
			"name":     v.Source.Name,
			"kind":     v.Source.Kind,
			"type":     v.Source.Type,
			"interval": v.Source.Interval.String(),
			"disabled": v.Source.Disabled,
		}
		if v.Source.URL != "" {
			src["url"] = v.Source.URL
		}
		entry := map[string]any{"source": src}
		if v.State != nil {
			entry["state"] = stateToPayload(v.State)
		}
		out = append(out, entry)
	}
	return out
}

func stateToPayload(s *model.SourceState) map[string]any {
	return map[string]any{
		"sourceId":        s.ID,
		"enabledOverride": s.EnabledOverride,
		"lastFetchedAt":   s.LastFetchedAt,
		"lastStatus":      s.LastStatus,
		"lastError":       s.LastError,
		"lastRunId":       s.LastRunID,
		"totalIocCount":   s.TotalIoCCount,
		"updatedAt":       s.UpdatedAt,
	}
}

func runSummary(r *model.Run) map[string]any {
	return map[string]any{
		"runId":      r.ID,
		"total":      r.Total,
		"triggered":  r.Triggered,
		"skipped":    r.Skipped,
		"failed":     r.Failed,
		"durationMs": r.DurationMs,
		"status":     r.Status,
	}
}

func runPayload(r *model.Run) map[string]any {
	srcs := make([]map[string]any, 0, len(r.Sources))
	for _, s := range r.Sources {
		srcs = append(srcs, map[string]any{
			"sourceId":        s.SourceID,
			"sourceKind":      s.SourceKind,
			"status":          s.Status,
			"skipReason":      s.SkipReason,
			"startedAt":       s.StartedAt,
			"finishedAt":      s.FinishedAt,
			"articleCount":    s.ArticleCount,
			"newArticleCount": s.NewArticleCount,
			"iocCount":        s.IoCCount,
			"errorMessage":    s.ErrorMessage,
		})
	}
	return map[string]any{
		"id":           r.ID,
		"trigger":      r.Trigger,
		"status":       r.Status,
		"startedAt":    r.StartedAt,
		"finishedAt":   r.FinishedAt,
		"durationMs":   r.DurationMs,
		"total":        r.Total,
		"triggered":    r.Triggered,
		"skipped":      r.Skipped,
		"failed":       r.Failed,
		"errorMessage": r.ErrorMessage,
		"sources":      srcs,
		"sourceIds":    r.SourceIDs,
	}
}

func iocPayload(i *model.IoC) map[string]any {
	return map[string]any{
		"id":          i.ID,
		"type":        i.Type,
		"value":       i.Value,
		"raw":         i.Raw,
		"firstSeenAt": i.FirstSeenAt,
		"lastSeenAt":  i.LastSeenAt,
	}
}

func depsWithCatalog(d Deps, c *source_catalog.Catalog) Deps {
	d.Catalog = c
	return d
}

// ---- SPA static handler ----

// spaHandler serves files from the embedded React build. Existing
// assets are delegated to http.FileServer; anything else (including
// the root "/") falls back to index.html so client-side react-router
// routes (e.g. /sources, /iocs) work after a full page reload.
//
// Mirrors the pattern used by secmon-lab/warren and
// secmon-lab/hecatoncheires — keep them aligned when changing this.
func spaHandler(fsys fs.FS) http.HandlerFunc {
	fileServer := http.FileServer(http.FS(fsys))
	return func(w http.ResponseWriter, r *http.Request) {
		urlPath := strings.TrimPrefix(r.URL.Path, "/")
		if urlPath == "" {
			urlPath = "index.html"
		}

		file, err := fsys.Open(urlPath)
		if err != nil {
			// File not found — serve index.html for SPA routing.
			indexFile, ierr := fsys.Open("index.html")
			if ierr != nil {
				http.NotFound(w, r)
				return
			}
			defer safe.Close(r.Context(), indexFile)
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			safe.Copy(r.Context(), w, indexFile)
			return
		}
		// File exists — close our probe handle and let FileServer stream it.
		safe.Close(r.Context(), file)
		fileServer.ServeHTTP(w, r)
	}
}

// ---- response helpers ----

func notFound(w http.ResponseWriter, _ *http.Request) {
	writeProblem(w, http.StatusNotFound, "not_found", "route not found", nil)
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

func writeProblem(w http.ResponseWriter, status int, code, msg string, details map[string]any) {
	body := map[string]any{"code": code, "message": msg}
	if details != nil {
		body["details"] = details
	}
	writeJSON(w, status, body)
}

// writeErr renders an error as a problem+json response AND funnels the
// same error through errutil.Handle so the operator-facing log captures
// the full goerr metadata (tags, values, stack). Without the Handle
// call, server-internal failures (e.g. Firestore API disabled) would
// only surface to the HTTP client and stay invisible to whoever is
// watching the server logs.
func writeErr(ctx context.Context, w http.ResponseWriter, err error) {
	status := errutil.HTTPStatus(err)
	// 4xx errors are caller mistakes (bad input, not found) — not worth
	// logging at error level. 5xx errors are server-side and MUST be
	// logged so the operator can act.
	if status >= http.StatusInternalServerError {
		errutil.Handle(ctx, err)
	}
	code := errutil.AsCode(err)
	details := map[string]any{}
	if ge := goerr.Unwrap(err); ge != nil {
		for k, v := range ge.Values() {
			details[k] = v
		}
	}
	writeProblem(w, status, code, err.Error(), details)
}

// ---- middlewares ----

func requestLogger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log := logging.From(r.Context()).With(
			slog.String("method", r.Method),
			slog.String("path", r.URL.Path),
		)
		ctx := logging.With(r.Context(), log)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func recoverer(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if rec := recover(); rec != nil {
				logging.From(r.Context()).LogAttrs(r.Context(), slog.LevelError, "panic in handler",
					slog.Any("recover", rec),
					slog.String("stack", string(debug.Stack())),
				)
				writeProblem(w, http.StatusInternalServerError, "internal_error", "internal server error", nil)
			}
		}()
		next.ServeHTTP(w, r)
	})
}

var _ = time.Now // keep imports stable across future helpers
var _ = interfaces.ErrLockBusy
