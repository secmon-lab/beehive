package refang_test

import (
	"testing"

	"github.com/m-mizutani/gt"
	"github.com/secmon-lab/beehive/pkg/service/refang"
)

func TestRefang(t *testing.T) {
	cases := []struct {
		in, want string
	}{
		{"evil[.]example", "evil.example"},
		{"hxxp://x[.]example", "http://x.example"},
		{"hxxps://y.com", "https://y.com"},
		{"a[@]b", "a@b"},
		{"already.normal.com", "already.normal.com"},
	}
	for _, c := range cases {
		gt.Equal(t, refang.Refang(c.in), c.want)
	}
}

func TestRefang_Idempotent(t *testing.T) {
	once := refang.Refang("evil[.]example")
	twice := refang.Refang(once)
	gt.Equal(t, once, twice)
}

func TestDefang(t *testing.T) {
	gt.Equal(t, refang.Defang("evil.example"), "evil[.]example")
	gt.Equal(t, refang.Defang("http://x.example"), "hxxp://x[.]example")
	gt.Equal(t, refang.Defang("https://y.com"), "hxxps://y[.]com")
}
