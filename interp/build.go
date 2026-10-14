package interp

import (
	"fmt"
	"go/ast"
	"go/build"
	"go/build/constraint"
	"go/parser"
	"path"
	"strconv"
	"strings"
)

// buildOk returns true if a file or script matches build constraints
// as specified in https://pkg.go.dev/cmd/go#hdr-Build_constraints.
// Both "//go:build" and legacy "// +build" lines are supported.
// If a "//go:build" line is present, "// +build" lines are ignored.
// An error from parser is returned as well.
func (interp *Interpreter) buildOk(ctx *build.Context, name, src string) (bool, error) {
	// Extract comments before the first clause
	f, err := parser.ParseFile(interp.fset, name, src, parser.PackageClauseOnly|parser.ParseComments)
	if err != nil {
		return false, err
	}

	var goBuild constraint.Expr
	var plusBuild []constraint.Expr
	for _, g := range f.Comments {
		for _, c := range g.List {
			// Build constraints are only valid in the file header.
			if c.Pos() >= f.Package {
				break
			}
			switch {
			case constraint.IsGoBuild(c.Text):
				if goBuild != nil {
					return false, fmt.Errorf("%s: multiple //go:build comments", interp.fset.Position(c.Pos()))
				}
				x, err := constraint.Parse(c.Text)
				if err != nil {
					return false, fmt.Errorf("%s: parsing //go:build line: %w", interp.fset.Position(c.Pos()), err)
				}
				goBuild = x
			case constraint.IsPlusBuild(c.Text):
				// As in go/build, invalid "// +build" lines are ignored.
				if x, err := constraint.Parse(c.Text); err == nil {
					plusBuild = append(plusBuild, x)
				}
			}
		}
	}

	tagOk := func(tag string) bool { return buildTagOk(ctx, tag) }
	if goBuild != nil {
		if !goBuild.Eval(tagOk) {
			return false, nil
		}
	} else {
		// In file, evaluate the AND of multiple "// +build" lines.
		for _, x := range plusBuild {
			if !x.Eval(tagOk) {
				return false, nil
			}
		}
	}

	setYaegiTags(ctx, f.Comments)
	return true, nil
}

// buildTagOk returns true if a build tag matches, false otherwise.
func buildTagOk(ctx *build.Context, s string) bool {
	switch {
	case contains(ctx.BuildTags, s):
		return true
	case s == ctx.GOOS:
		return true
	case s == ctx.GOARCH:
		return true
	case len(s) > 4 && s[:4] == "go1.":
		n, err := strconv.Atoi(s[4:])
		if err != nil {
			return false
		}
		return goMinorVersion(ctx) >= n
	default:
		return false
	}
}

// setYaegiTags scans a comment group for "yaegi:tags tag1 tag2 ..." lines
// and adds the corresponding tags to the interpreter build tags.
func setYaegiTags(ctx *build.Context, comments []*ast.CommentGroup) {
	for _, g := range comments {
		for _, line := range strings.Split(strings.TrimSpace(g.Text()), "\n") {
			if len(line) < 11 || line[:11] != "yaegi:tags " {
				continue
			}

			tags := strings.Split(strings.TrimSpace(line[10:]), " ")
			for _, tag := range tags {
				if !contains(ctx.BuildTags, tag) {
					ctx.BuildTags = append(ctx.BuildTags, tag)
				}
			}
		}
	}
}

func contains(tags []string, tag string) bool {
	for _, t := range tags {
		if t == tag {
			return true
		}
	}
	return false
}

// goMinorVersion returns the go minor version number.
func goMinorVersion(ctx *build.Context) int {
	current := ctx.ReleaseTags[len(ctx.ReleaseTags)-1]

	v := strings.Split(current, ".")
	if len(v) < 2 {
		panic("unsupported Go version: " + current)
	}

	m, err := strconv.Atoi(v[1])
	if err != nil {
		panic("unsupported Go version: " + current)
	}
	return m
}

// skipFile returns true if file should be skipped.
func skipFile(ctx *build.Context, p string, skipTest bool) bool {
	if !strings.HasSuffix(p, ".go") {
		return true
	}
	p = strings.TrimSuffix(path.Base(p), ".go")
	if pp := path.Base(p); strings.HasPrefix(pp, "_") || strings.HasPrefix(pp, ".") {
		return true
	}
	if skipTest && strings.HasSuffix(p, "_test") {
		return true
	}
	i := strings.Index(p, "_")
	if i < 0 {
		return false
	}
	a := strings.Split(p[i+1:], "_")
	last := len(a) - 1
	if last-1 >= 0 {
		switch x, y := a[last-1], a[last]; {
		case x == ctx.GOOS:
			if knownArch[y] {
				return y != ctx.GOARCH
			}
			return false
		case knownOs[x] && knownArch[y]:
			return true
		case knownArch[y] && y != ctx.GOARCH:
			return true
		default:
			return false
		}
	}
	if x := a[last]; knownOs[x] && x != ctx.GOOS || knownArch[x] && x != ctx.GOARCH {
		return true
	}
	return false
}

var knownOs = map[string]bool{
	"aix":       true,
	"android":   true,
	"darwin":    true,
	"dragonfly": true,
	"freebsd":   true,
	"illumos":   true,
	"ios":       true,
	"js":        true,
	"linux":     true,
	"netbsd":    true,
	"openbsd":   true,
	"plan9":     true,
	"solaris":   true,
	"wasip1":    true,
	"windows":   true,
}

var knownArch = map[string]bool{
	"386":      true,
	"amd64":    true,
	"arm":      true,
	"arm64":    true,
	"loong64":  true,
	"mips":     true,
	"mips64":   true,
	"mips64le": true,
	"mipsle":   true,
	"ppc64":    true,
	"ppc64le":  true,
	"s390x":    true,
	"wasm":     true,
}
