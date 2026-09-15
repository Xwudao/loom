// Package diag renders human-friendly diagnostics for graph problems.
//
// Loom's error messages are part of its product surface: the parser and
// resolver run without a debugger, so a diagnostic must explain what went
// wrong, where it was declared, and how the dependency path reached it.
package diag

import (
	"fmt"
	"go/token"
	"go/types"
	"path/filepath"
	"strings"
)

// Frame is one step of a dependency path.
type Frame struct {
	// Type is the type at this step.
	Type types.Type
	// Label identifies the provider that produces Type, such as
	// "NewUserService(UserStore)". It is empty for the unresolved step.
	Label string
	// Pos is the provider's declaration position.
	Pos token.Position
}

// Error is a diagnostic with a source position and an optional dependency
// path. It implements error.
type Error struct {
	// Msg is the headline, without the "loom: " prefix.
	Msg string
	// Note annotates the final path frame, such as "missing" or "cycle".
	Note string
	// Path is the dependency chain from the graph target to the problem.
	Path []Frame
	// Providers is an explicit provider list, used when the problem is not a
	// dependency chain, such as duplicate providers.
	Providers []Frame
	// Pos is the primary source position, used when there is no path.
	Pos token.Position
	// Hint is an optional actionable suggestion.
	Hint string
	// Fmt controls how types are rendered. It must be set before calling
	// Error or String.
	Fmt Formatter
}

// Formatter renders types relative to a current package, so that types from
// the package under generation print without a qualifier, and positions
// relative to a base directory.
type Formatter struct {
	Current *types.Package
	Base    string
}

// Type renders t without the qualifier for the current package.
func (f Formatter) Type(t types.Type) string {
	if t == nil {
		return "<nil>"
	}
	return types.TypeString(t, func(p *types.Package) string {
		if p == nil || p == f.Current {
			return ""
		}
		return p.Name()
	})
}

// Pos renders a position relative to Base when possible.
func (f Formatter) Pos(p token.Position) string {
	if !p.IsValid() {
		return ""
	}
	if f.Base != "" && filepath.IsAbs(p.Filename) {
		if rel, err := filepath.Rel(f.Base, p.Filename); err == nil && !strings.HasPrefix(rel, "..") {
			p.Filename = rel
		}
	}
	return p.String()
}

// Error implements error.
func (e *Error) Error() string {
	var b strings.Builder
	b.WriteString("loom: ")
	b.WriteString(e.Msg)

	if len(e.Path) > 0 {
		b.WriteString("\n\ndependency path:\n")
		for i, f := range e.Path {
			if i == 0 {
				fmt.Fprintf(&b, "    %s\n", e.Fmt.Type(f.Type))
				continue
			}
			fmt.Fprintf(&b, "      -> %s", e.Fmt.Type(f.Type))
			if i == len(e.Path)-1 && e.Note != "" {
				fmt.Fprintf(&b, "  (%s)", e.Note)
			}
			b.WriteString("\n")
		}
	}

	if provs := e.providers(); len(provs) > 0 {
		b.WriteString("\nproviders:\n")
		width := 0
		for _, f := range provs {
			if n := len(e.Fmt.Pos(f.Pos)); n > width {
				width = n
			}
		}
		for _, f := range provs {
			fmt.Fprintf(&b, "    %-*s  %s\n", width, e.Fmt.Pos(f.Pos), f.Label)
		}
	}

	if len(e.Path) == 0 && len(e.Providers) == 0 && e.Pos.IsValid() {
		fmt.Fprintf(&b, "\nat:\n    %s\n", e.Fmt.Pos(e.Pos))
	}
	if e.Hint != "" {
		b.WriteString("\nhint:\n    ")
		b.WriteString(e.Hint)
		b.WriteString("\n")
	}
	return b.String()
}

// providers returns the frames to list under "providers", collapsing
// consecutive duplicates.
func (e *Error) providers() []Frame {
	if len(e.Providers) > 0 {
		return e.Providers
	}
	var out []Frame
	for _, f := range e.Path {
		if f.Label == "" {
			continue
		}
		if n := len(out); n > 0 && out[n-1].Label == f.Label && out[n-1].Pos == f.Pos {
			continue
		}
		out = append(out, f)
	}
	return out
}

// ProviderLabel renders a provider signature for diagnostics, e.g.
// "NewUserService(UserStore)".
func ProviderLabel(name string, inputs []types.Type, f Formatter) string {
	if len(inputs) == 0 {
		return name + "()"
	}
	parts := make([]string, len(inputs))
	for i, in := range inputs {
		parts[i] = f.Type(in)
	}
	return name + "(" + strings.Join(parts, ", ") + ")"
}
