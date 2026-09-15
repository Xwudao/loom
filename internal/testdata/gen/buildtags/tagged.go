//go:build loom_test_tag

package buildtags

// NewConfig is only compiled with the loom_test_tag build tag. If loom ignored
// build constraints this would be a duplicate provider for *Config.
func NewConfig() *Config { return &Config{Source: "tagged"} }
