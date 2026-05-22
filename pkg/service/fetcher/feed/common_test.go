package feed

import (
	"testing"
	"time"

	"github.com/m-mizutani/gt"
	"github.com/secmon-lab/beehive/pkg/domain/types"
)

func TestSeedIP_AcceptsPublic(t *testing.T) {
	seed, ok := seedIP("8.8.8.8", 0.9, time.Now())
	gt.True(t, ok)
	gt.Equal(t, seed.Type, types.IoCTypeIPv4)
	gt.Equal(t, seed.Value, "8.8.8.8")
}

func TestSeedIP_RejectsPrivateLoopback(t *testing.T) {
	for _, addr := range []string{
		"127.0.0.1", "10.0.0.1", "192.168.0.1", "169.254.1.1",
		"::1", "fe80::1", "fc00::1",
	} {
		_, ok := seedIP(addr, 0.9, time.Now())
		gt.False(t, ok)
	}
}

func TestSeedIP_RejectsGarbage(t *testing.T) {
	_, ok := seedIP("not-an-ip", 0.5, time.Now())
	gt.False(t, ok)
}

func TestIterText_SkipsCommentsAndBlanks(t *testing.T) {
	body := "# comment\n\n   \n1.2.3.4\n# 5.6.7.8\n2.3.4.5\n"
	var got []string
	iterText([]byte(body), func(line string) { got = append(got, line) })
	gt.A(t, got).Length(2)
	gt.Equal(t, got[0], "1.2.3.4")
	gt.Equal(t, got[1], "2.3.4.5")
}
