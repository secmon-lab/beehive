package extractor

// Exposed for tests in package extractor_test.

// LLMConfigOf returns the LLMConfig captured by NewLLMClient, so tests
// can assert that user-supplied Model / Args / APIKey survive without
// being silently replaced by provider-internal defaults.
func LLMConfigOf(c any) (LLMConfig, bool) {
	g, ok := c.(*gollemClient)
	if !ok {
		return LLMConfig{}, false
	}
	return g.cfg, true
}
