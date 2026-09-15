// Package settings declares a type that no constructor in this package
// produces. It must never be imported by generated code, because a variable
// holding a *settings.Settings is passed by value and the type is never
// spelled out.
package settings

// Settings holds configuration.
type Settings struct{}
