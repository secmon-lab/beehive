package feed_test

import (
	"testing"

	"github.com/m-mizutani/gt"
	"github.com/secmon-lab/beehive/pkg/domain/types"
	"github.com/secmon-lab/beehive/pkg/service/fetcher/feed"
)

func TestParseURLhaus(t *testing.T) {
	body := []byte(`# URLhaus CSV
"1","2026-01-01 12:00:00","http://malware.example/a","online","exe","tag1"
"2","2026-01-02 09:30:00","https://malware.example/b","online","js","tag2"
`)
	seeds, err := feed.ParseURLhaus(body)
	gt.NoError(t, err)
	gt.A(t, seeds).Length(2)
	gt.Equal(t, seeds[0].Type, types.IoCTypeURL)
	gt.Equal(t, seeds[0].Value, "http://malware.example/a")
}

func TestParseURLhaus_Empty(t *testing.T) {
	seeds, err := feed.ParseURLhaus([]byte(""))
	gt.NoError(t, err)
	gt.A(t, seeds).Length(0)
}
