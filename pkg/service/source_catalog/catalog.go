// Package source_catalog loads Source definitions from TOML files into an
// in-memory catalog. The TOML files are the single source of truth for
// Source metadata (id / name / type / interval / disabled / url) — they
// are NOT mirrored to Firestore. See spec §2.7 and CLAUDE.md.
package source_catalog

import (
	"errors"
	"io/fs"
	"os"
	"time"

	"github.com/m-mizutani/goerr/v2"
	toml "github.com/pelletier/go-toml/v2"
	"github.com/secmon-lab/beehive/pkg/domain/model"
	"github.com/secmon-lab/beehive/pkg/domain/types"
	"github.com/secmon-lab/beehive/pkg/utils/errutil"
)

// rawSource mirrors the TOML schema. The TOML keys are lower-snake-case
// for human ergonomics; we translate to the PascalCase model.Source in
// Load.
type rawSource struct {
	ID       string `toml:"id"`
	Name     string `toml:"name"`
	Type     string `toml:"type"`
	URL      string `toml:"url"`
	Interval string `toml:"interval"`
	Disabled bool   `toml:"disabled"`
}

type rawFile struct {
	Source []rawSource `toml:"source"`
}

// LoadedSource is what Load returns for each TOML `[[source]]` entry,
// alongside diagnostic information (file path + line) that validators
// and `beehive validate` use to render readable errors.
type LoadedSource struct {
	Source *model.Source
	File   string // absolute path of the TOML file the entry came from
	Index  int    // index within the file (0-based, useful for messages)
}

// Catalog is the in-memory result of loading a config directory. Use
// Load() to construct it.
type Catalog struct {
	Sources []*model.Source
	Loaded  []LoadedSource
}

// ByID returns the loaded source for id, if any.
func (c *Catalog) ByID(id types.SourceID) (*model.Source, bool) {
	for _, s := range c.Sources {
		if s.ID == id {
			return s, true
		}
	}
	return nil, false
}

// Load reads a single TOML config file. A missing file yields an empty
// catalog (the operator may not have written one yet); other I/O or
// parse errors are returned. Semantic checks (required fields, unknown
// provider type, etc.) live in Validate.
func Load(path string) (*Catalog, error) {
	bs, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return &Catalog{}, nil
		}
		return nil, goerr.Wrap(err, "read config file",
			goerr.V("path", path))
	}
	var rf rawFile
	if err := toml.Unmarshal(bs, &rf); err != nil {
		return nil, goerr.Wrap(err, "parse toml",
			goerr.V("path", path),
			goerr.T(errutil.TagInvalidInput))
	}
	cat := &Catalog{}
	for i, rs := range rf.Source {
		src, err := convert(&rs)
		if err != nil {
			return nil, goerr.Wrap(err, "convert source",
				goerr.V("path", path),
				goerr.V("index", i),
				goerr.V("id", rs.ID))
		}
		cat.Sources = append(cat.Sources, src)
		cat.Loaded = append(cat.Loaded, LoadedSource{
			Source: src, File: path, Index: i,
		})
	}
	return cat, nil
}

func convert(rs *rawSource) (*model.Source, error) {
	if rs.Interval == "" {
		return nil, goerr.New("interval is required",
			goerr.T(errutil.TagInvalidInput))
	}
	d, err := time.ParseDuration(rs.Interval)
	if err != nil {
		return nil, goerr.Wrap(err, "parse interval",
			goerr.V("value", rs.Interval),
			goerr.T(errutil.TagInvalidInput))
	}
	return &model.Source{
		ID:       types.SourceID(rs.ID),
		Name:     rs.Name,
		Type:     rs.Type,
		URL:      rs.URL,
		Interval: d,
		Disabled: rs.Disabled,
		// Kind is filled by Validate after looking up the provider.
	}, nil
}
