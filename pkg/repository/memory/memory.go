// Package memory is the in-memory implementation of interfaces.Repository.
// It is used for unit tests and for local runs when a real Firestore is not
// available (BEEHIVE_REPO_BACKEND=memory).
//
// All methods are safe for concurrent use. The implementation is
// intentionally simple — no indexes, no pagination tokens — because the
// total dataset size in tests / local runs is small.
package memory

import (
	"os"
	"sync"
	"time"

	"github.com/m-mizutani/goerr/v2"
	"github.com/secmon-lab/beehive/pkg/domain/interfaces"
	"github.com/secmon-lab/beehive/pkg/domain/model"
	"github.com/secmon-lab/beehive/pkg/domain/types"
	"github.com/secmon-lab/beehive/pkg/utils/errutil"
)

// Memory is the in-memory repository implementation.
type Memory struct {
	mu sync.RWMutex

	states       map[types.SourceID]*model.SourceState
	articles     map[types.ArticleID]*model.Article
	articleByURL map[string]types.ArticleID
	iocs         map[types.IoCID]*model.IoC
	refs         map[types.IoCID]map[types.RefID]*model.IoCRef
	runs         map[types.RunID]*model.Run
	locks        map[string]*model.Lock // key: kind + "/" + targetID

	// holderID is used by the lock implementation so each Memory instance
	// looks like a distinct "holder". Real implementations derive this
	// from hostname + pid + goroutine id.
	holderID string
}

// New returns a fresh Memory repository.
func New() *Memory {
	host, _ := os.Hostname()
	return &Memory{
		states:       make(map[types.SourceID]*model.SourceState),
		articles:     make(map[types.ArticleID]*model.Article),
		articleByURL: make(map[string]types.ArticleID),
		iocs:         make(map[types.IoCID]*model.IoC),
		refs:         make(map[types.IoCID]map[types.RefID]*model.IoCRef),
		runs:         make(map[types.RunID]*model.Run),
		locks:        make(map[string]*model.Lock),
		holderID:     host + "/memory",
	}
}

// Compile-time check.
var _ interfaces.Repository = (*Memory)(nil)

// ---- helpers ----

func lockKey(kind, target string) string { return kind + "/" + target }

func cloneSourceState(s *model.SourceState) *model.SourceState {
	if s == nil {
		return nil
	}
	c := *s
	return &c
}

func cloneArticle(a *model.Article) *model.Article {
	if a == nil {
		return nil
	}
	c := *a
	return &c
}

func cloneIoC(i *model.IoC) *model.IoC {
	if i == nil {
		return nil
	}
	c := *i
	return &c
}

func cloneRun(r *model.Run) *model.Run {
	if r == nil {
		return nil
	}
	c := *r
	if r.Sources != nil {
		c.Sources = append([]model.RunSource(nil), r.Sources...)
	}
	if r.SourceIDs != nil {
		c.SourceIDs = append([]types.SourceID(nil), r.SourceIDs...)
	}
	return &c
}

func notFound(what string, key any) error {
	return goerr.New(what+" not found",
		goerr.V("key", key),
		goerr.T(errutil.TagNotFound),
	)
}

func nowUTC() time.Time { return time.Now().UTC() }
