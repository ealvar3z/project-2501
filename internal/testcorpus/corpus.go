package testcorpus

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
)

type Suite string

const (
	SuiteCharset Suite = "charset"
	SuiteGo      Suite = "go"
	SuiteJS      Suite = "js"
	SuiteLayout  Suite = "layout"
	SuiteMD      Suite = "md"
	SuiteNet     Suite = "net"
	SuitePager   Suite = "pager"
)

var AllSuites = []Suite{
	SuiteCharset,
	SuiteGo,
	SuiteJS,
	SuiteLayout,
	SuiteMD,
	SuiteNet,
	SuitePager,
}

type DisplayOptions struct {
	Columns         int
	Lines           int
	PixelsPerColumn int
	PixelsPerLine   int
	ColorMode       string
	FormatMode      []string
}

func DefaultDisplayOptions() DisplayOptions {
	return DisplayOptions{
		Columns:         80,
		Lines:           24,
		PixelsPerColumn: 9,
		PixelsPerLine:   18,
		ColorMode:       "true-color",
		FormatMode:      []string{"bold", "italic", "underline", "reverse", "strike"},
	}
}

type Fixture struct {
	Suite    Suite
	Name     string
	Input    string
	Expected string
}

func Root() (string, error) {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		return "", errors.New("cannot locate test corpus package")
	}
	root := filepath.Clean(filepath.Join(filepath.Dir(file), "..", "..", "testdata", "pm"))
	if _, err := os.Stat(root); err != nil {
		return "", err
	}
	return root, nil
}

func CountTopLevelFiles(root string) (map[Suite]int, error) {
	counts := make(map[Suite]int)
	for _, suite := range AllSuites {
		dir := filepath.Join(root, string(suite))
		entries, err := os.ReadDir(dir)
		if err != nil {
			return nil, err
		}
		for _, entry := range entries {
			if !entry.Type().IsRegular() {
				continue
			}
			counts[suite]++
		}
	}
	return counts, nil
}

func Discover(root string, suite Suite) ([]Fixture, error) {
	switch suite {
	case SuiteLayout:
		return discoverPaired(root, suite, ".html")
	case SuiteMD:
		return discoverPaired(root, suite, ".md")
	case SuiteJS:
		return discoverSharedExpected(root, suite, ".html", "all.expected", nil)
	case SuiteNet:
		skip := func(name string) bool {
			return name == "cookie.css.http" || name == "headers.http" ||
				strings.HasPrefix(name, "module") && strings.HasSuffix(name, ".http")
		}
		return discoverSharedExpected(root, suite, ".html .http", "all.expected", skip)
	case SuiteCharset:
		dir := filepath.Join(root, string(suite))
		return []Fixture{{
			Suite:    suite,
			Name:     "x",
			Input:    filepath.Join(dir, "x"),
			Expected: filepath.Join(dir, "x.expected"),
		}}, nil
	case SuitePager:
		dir := filepath.Join(root, string(suite))
		return []Fixture{
			{Suite: suite, Name: "test2", Input: filepath.Join(dir, "test2.test2"), Expected: filepath.Join(dir, "test2.expected")},
			{Suite: suite, Name: "test3", Input: filepath.Join(dir, "test.toml"), Expected: filepath.Join(dir, "test3.expected")},
		}, nil
	default:
		return nil, fmt.Errorf("suite %q has no fixture discovery rule", suite)
	}
}

func Filter(fixtures []Fixture, name string) []Fixture {
	if name == "" {
		return fixtures
	}
	var out []Fixture
	for _, fixture := range fixtures {
		if strings.Contains(fixture.Name, name) {
			out = append(out, fixture)
		}
	}
	return out
}

func ReadExpected(fixture Fixture) (string, error) {
	b, err := os.ReadFile(fixture.Expected)
	if err != nil {
		return "", err
	}
	return normalizeNewlines(b), nil
}

func EqualGolden(expected, got string) bool {
	return normalizeString(expected) == normalizeString(got)
}

func normalizeString(s string) string {
	return normalizeNewlines([]byte(s))
}

func normalizeNewlines(b []byte) string {
	return string(bytes.ReplaceAll(b, []byte("\r\n"), []byte("\n")))
}

func discoverPaired(root string, suite Suite, ext string) ([]Fixture, error) {
	dir := filepath.Join(root, string(suite))
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	var fixtures []Fixture
	for _, entry := range entries {
		if !entry.Type().IsRegular() || filepath.Ext(entry.Name()) != ext {
			continue
		}
		name := strings.TrimSuffix(entry.Name(), ext)
		fixtures = append(fixtures, Fixture{
			Suite:    suite,
			Name:     name,
			Input:    filepath.Join(dir, entry.Name()),
			Expected: filepath.Join(dir, name+".expected"),
		})
	}
	slices.SortFunc(fixtures, func(a, b Fixture) int {
		return strings.Compare(a.Name, b.Name)
	})
	return fixtures, nil
}

func discoverSharedExpected(root string, suite Suite, exts string, expectedName string, skip func(string) bool) ([]Fixture, error) {
	dir := filepath.Join(root, string(suite))
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	expected := filepath.Join(dir, expectedName)
	allowed := strings.Fields(exts)
	var fixtures []Fixture
	for _, entry := range entries {
		if !entry.Type().IsRegular() {
			continue
		}
		name := entry.Name()
		if skip != nil && skip(name) {
			continue
		}
		if !slices.Contains(allowed, filepath.Ext(name)) {
			continue
		}
		fixtures = append(fixtures, Fixture{
			Suite:    suite,
			Name:     strings.TrimSuffix(name, filepath.Ext(name)),
			Input:    filepath.Join(dir, name),
			Expected: expected,
		})
	}
	slices.SortFunc(fixtures, func(a, b Fixture) int {
		return strings.Compare(a.Input, b.Input)
	})
	return fixtures, nil
}
