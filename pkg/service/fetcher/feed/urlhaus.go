package feed

import (
	"context"
	"encoding/csv"
	"io"
	"strings"
	"time"

	"github.com/m-mizutani/goerr/v2"
	"github.com/secmon-lab/beehive/pkg/domain/interfaces"
	"github.com/secmon-lab/beehive/pkg/domain/model"
	"github.com/secmon-lab/beehive/pkg/domain/types"
	"github.com/secmon-lab/beehive/pkg/service/fetcher"
)

const URLhausTypeID = "urlhaus_recent"
const urlhausURL = "https://urlhaus.abuse.ch/downloads/csv_recent/"

type URLhausProvider struct {
	http *fetcher.HTTPClient
}

func NewURLhaus(c *fetcher.HTTPClient) *URLhausProvider {
	return &URLhausProvider{http: c}
}

func (p *URLhausProvider) Kind() types.SourceKind { return types.KindFeed }

func (p *URLhausProvider) Fetch(ctx context.Context, _ *model.Source) ([]*interfaces.IoCSeed, error) {
	body, err := p.http.Get(ctx, urlhausURL, nil)
	if err != nil {
		return nil, err
	}
	return ParseURLhaus(body)
}

// ParseURLhaus parses the CSV body. The expected columns are documented
// at https://urlhaus.abuse.ch/api/#csv. We only care about the URL
// column ("url"); other fields are reserved for future use.
func ParseURLhaus(raw []byte) ([]*interfaces.IoCSeed, error) {
	r := csv.NewReader(strings.NewReader(string(raw)))
	r.Comment = '#'
	r.FieldsPerRecord = -1 // tolerate ragged rows

	now := time.Now().UTC()
	var out []*interfaces.IoCSeed
	for {
		rec, err := r.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, goerr.Wrap(err, "parse urlhaus csv")
		}
		// Column layout (per URLhaus docs):
		//   0: id, 1: dateadded, 2: url, 3: url_status, 4: ...
		if len(rec) < 3 {
			continue
		}
		seed := seedURL(rec[2], 0.85, now)
		// Best-effort SeenAt from the dateadded column.
		if t, err := time.Parse("2006-01-02 15:04:05", strings.TrimSpace(rec[1])); err == nil {
			seed.SeenAt = t.UTC()
		}
		out = append(out, seed)
	}
	return out, nil
}
