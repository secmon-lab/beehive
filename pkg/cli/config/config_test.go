package config_test

import (
	"testing"

	"github.com/m-mizutani/gt"
	"github.com/secmon-lab/beehive/pkg/cli/config"
)

func TestLLM_ArgsMap_Empty(t *testing.T) {
	c := config.LLM{}
	m, err := c.ArgsMap()
	gt.NoError(t, err)
	gt.A(t, mapKeys(m)).Length(0)
}

func TestLLM_ArgsMap_Parses(t *testing.T) {
	c := config.LLM{Args: "project_id=my-proj , location=us-central1 ,foo=bar"}
	m, err := c.ArgsMap()
	gt.NoError(t, err)
	gt.Equal(t, m["project_id"], "my-proj")
	gt.Equal(t, m["location"], "us-central1")
	gt.Equal(t, m["foo"], "bar")
}

func TestLLM_ArgsMap_RejectsInvalidEntry(t *testing.T) {
	c := config.LLM{Args: "project_id=foo,broken"}
	_, err := c.ArgsMap()
	gt.Error(t, err)
}

func TestFirestore_Validate_OnlyAppliesWhenBackendIsFirestore(t *testing.T) {
	c := config.Firestore{}
	gt.NoError(t, c.Validate("memory"))
}

func TestFirestore_Validate_RequiresProjectID(t *testing.T) {
	c := config.Firestore{Database: "beehive"}
	err := c.Validate("firestore")
	gt.Error(t, err)
}

func TestFirestore_Validate_RequiresDatabase(t *testing.T) {
	c := config.Firestore{ProjectID: "p"}
	err := c.Validate("firestore")
	gt.Error(t, err)
}

func TestFirestore_Validate_Passes(t *testing.T) {
	c := config.Firestore{ProjectID: "p", Database: "beehive"}
	gt.NoError(t, c.Validate("firestore"))
}

func TestSources_DefaultPath(t *testing.T) {
	c := config.Sources{Path: "./config.toml"}
	gt.Equal(t, c.Path, "./config.toml")
}

func TestLogger_Configure_Auto(t *testing.T) {
	// stderr is normally a TTY in interactive runs and a pipe in CI;
	// either way the resolver must not error out.
	c := config.Logger{Level: "info", FormatName: "auto", Output: "stderr"}
	closer, err := c.Configure()
	gt.NoError(t, err)
	t.Cleanup(closer)
}

func TestLogger_Configure_RejectsBadFormat(t *testing.T) {
	c := config.Logger{Level: "info", FormatName: "xml", Output: "stderr"}
	_, err := c.Configure()
	gt.Error(t, err)
}

func TestLogger_Configure_RejectsBadLevel(t *testing.T) {
	c := config.Logger{Level: "loud", FormatName: "json", Output: "stderr"}
	_, err := c.Configure()
	gt.Error(t, err)
}

func TestLogger_Configure_QuietWins(t *testing.T) {
	c := config.Logger{Quiet: true, Level: "garbage", FormatName: "garbage"}
	closer, err := c.Configure()
	gt.NoError(t, err)
	t.Cleanup(closer)
}

func mapKeys[K comparable, V any](m map[K]V) []K {
	out := make([]K, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}
