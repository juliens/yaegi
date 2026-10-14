package interp

import (
	"go/build"
	"testing"
)

type testBuild struct {
	src string
	res bool
}

func TestBuildTag(t *testing.T) {
	// Assume a specific OS, arch and go version no matter the real underlying system
	ctx := build.Context{
		GOARCH:      "amd64",
		GOOS:        "linux",
		BuildTags:   []string{"foo"},
		ReleaseTags: []string{"go1.11"},
	}

	tests := []testBuild{
		{"// +build linux", true},
		{"// +build windows", false},
		{"// +build go1.9", true},
		{"// +build go1.11", true},
		{"// +build go1.12", false},
		{"// +build !go1.10", false},
		{"// +build !go1.12", true},
		{"// +build ignore", false},
		{"// +build linux,amd64", true},
		{"// +build linux,i386", false},
		{"// +build linux,i386 go1.11", true},
		{"// +build linux\n// +build amd64", true},
		{"// +build linux\n\n\n// +build amd64", true},
		{"// +build linux\n// +build i386", false},
		{"// +build foo", true},
		{"// +build !foo", false},
		{"// +build bar", false},
		{"//go:build linux", true},
		{"//go:build windows", false},
		{"//go:build ignore", false},
		{"//go:build go1.9", true},
		{"//go:build go1.11", true},
		{"//go:build go1.12", false},
		{"//go:build !go1.12", true},
		{"//go:build !go1.10", false},
		{"//go:build foo", true},
		{"//go:build !foo", false},
		{"//go:build linux && amd64", true},
		{"//go:build linux && 386", false},
		{"//go:build windows || linux", true},
		{"//go:build windows || darwin", false},
		{"//go:build !windows", true},
		{"//go:build !linux", false},
		{"//go:build (linux && 386) || go1.11", true},
		{"//go:build (linux || windows) && !amd64", false},
		{"//go:build !(windows || darwin)", true},
		{"//go:build linux\n// +build windows", true},
		{"//go:build windows\n// +build linux", false},
		{"// +build windows\n\n//go:build linux", true},
		{"// Copyright notice.\n\n//go:build ignore", false},
		{"/* block comment */\n//go:build ignore", false},
	}

	i := New(Options{})
	for _, test := range tests {
		test := test
		src := test.src + "\npackage x"
		t.Run(test.src, func(t *testing.T) {
			r, err := i.buildOk(&ctx, "", src)
			if err != nil {
				t.Fatal(err)
			}
			if r != test.res {
				t.Errorf("got %v, want %v", r, test.res)
			}
		})
	}
}

func TestBuildTag_afterPackageClause(t *testing.T) {
	ctx := build.Context{
		GOARCH:      "amd64",
		GOOS:        "linux",
		ReleaseTags: []string{"go1.11"},
	}

	i := New(Options{})

	ok, err := i.buildOk(&ctx, "", "package x\n\n//go:build ignore\n")
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Error("got false, want true: constraints after the package clause must be ignored")
	}
}

func TestBuildTag_errors(t *testing.T) {
	ctx := build.Context{
		GOARCH:      "amd64",
		GOOS:        "linux",
		ReleaseTags: []string{"go1.11"},
	}

	tests := []struct {
		desc string
		src  string
	}{
		{
			desc: "invalid expression",
			src:  "//go:build linux &&\n\npackage x",
		},
		{
			desc: "multiple go:build lines",
			src:  "//go:build linux\n//go:build amd64\n\npackage x",
		},
	}

	i := New(Options{})
	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			if _, err := i.buildOk(&ctx, "", test.src); err == nil {
				t.Error("got no error, want one")
			}
		})
	}
}

func TestSkipFile(t *testing.T) {
	// Assume a specific OS, arch and go pattern no matter the real underlying system
	ctx := build.Context{
		GOARCH: "amd64",
		GOOS:   "linux",
	}

	tests := []testBuild{
		{"foo/bar_linux_amd64.go", false},
		{"foo/bar.go", false},
		{"bar.go", false},
		{"bar_linux.go", false},
		{"bar_maix.go", false},
		{"bar_mlinux.go", false},

		{"bar_aix_foo.go", false},
		{"bar_linux_foo.go", false},
		{"bar_foo_amd64.go", false},
		{"bar_foo_arm.go", true},

		{"bar_aix_s390x.go", true},
		{"bar_aix_amd64.go", true},
		{"bar_linux_arm.go", true},

		{"bar_amd64.go", false},
		{"bar_arm.go", true},
	}

	for _, test := range tests {
		test := test
		t.Run(test.src, func(t *testing.T) {
			if r := skipFile(&ctx, test.src, NoTest); r != test.res {
				t.Errorf("got %v, want %v", r, test.res)
			}
		})
	}
}

func Test_goMinorVersion(t *testing.T) {
	tests := []struct {
		desc     string
		context  build.Context
		expected int
	}{
		{
			desc: "stable",
			context: build.Context{ReleaseTags: []string{
				"go1.1", "go1.2", "go1.3", "go1.4", "go1.5", "go1.6", "go1.7", "go1.8", "go1.9", "go1.10", "go1.11", "go1.12",
			}},
			expected: 12,
		},
		{
			desc: "devel/beta/rc",
			context: build.Context{ReleaseTags: []string{
				"go1.1", "go1.2", "go1.3", "go1.4", "go1.5", "go1.6", "go1.7", "go1.8", "go1.9", "go1.10", "go1.11", "go1.12", "go1.13",
			}},
			expected: 13,
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.desc, func(t *testing.T) {
			minor := goMinorVersion(&test.context)

			if minor != test.expected {
				t.Errorf("got %v, want %v", minor, test.expected)
			}
		})
	}
}
