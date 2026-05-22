// Package refang converts between fanged (clickable / dangerous) and
// defanged (safe to share) representations of IoCs. Reused both at
// ingest time (refanging incoming text before normalization) and at
// display time (defanging for the UI).
package refang

import "strings"

// Refang reverses common defang transforms ([.] -> ., hxxp -> http,
// [@] -> @, [:] -> :, etc.) so we can normalize the value. Idempotent.
func Refang(s string) string {
	repl := strings.NewReplacer(
		"[.]", ".", "(.)", ".", "{.}", ".",
		"[:]", ":", "(:)", ":",
		"[/]", "/", "(/)", "/",
		"[@]", "@", "(@)", "@", "{@}", "@",
		"hxxp://", "http://",
		"hxxps://", "https://",
		"hxxp[://]", "http://",
		"hxxps[://]", "https://",
		"hXXp://", "http://",
		"hXXps://", "https://",
	)
	return repl.Replace(s)
}

// Defang returns a presentation-safe form for the UI. We only escape
// the dot and scheme: that's enough for analysts to spot the IoC at a
// glance without making it clickable.
func Defang(s string) string {
	out := strings.ReplaceAll(s, ".", "[.]")
	out = strings.ReplaceAll(out, "http[.]", "hxxp[.]")
	out = strings.ReplaceAll(out, "http://", "hxxp://")
	out = strings.ReplaceAll(out, "https://", "hxxps://")
	return out
}
