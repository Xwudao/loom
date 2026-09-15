// Package provider supplies a settings.Settings value from a different package,
// which is the shape that used to trigger an unused import.
package provider

import "github.com/Xwudao/loom/internal/testdata/gen/naming/settings"

// NewSettings loads settings.
func NewSettings() *settings.Settings { return &settings.Settings{} }
