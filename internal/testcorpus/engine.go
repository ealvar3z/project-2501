package testcorpus

import (
	"context"
	"fmt"
	"strings"
)

type Engine interface {
	RenderHTML(context.Context, []byte, RenderOptions) (string, error)
	RenderMarkdown(context.Context, []byte, RenderOptions) (string, error)
	RunHTMLWithJS(context.Context, []byte, RenderOptions) (string, error)
	FetchAndRender(context.Context, string, RenderOptions) (string, error)
}

type RenderOptions struct {
	Display DisplayOptions
	Config  string
	BaseURL string
	Charset string
}

func DefaultRenderOptions() RenderOptions {
	return RenderOptions{
		Display: DefaultDisplayOptions(),
	}
}

type GoldenMismatch struct {
	Fixture  Fixture
	Expected string
	Got      string
}

func (m GoldenMismatch) Error() string {
	return fmt.Sprintf("%s/%s output mismatch\n%s", m.Fixture.Suite, m.Fixture.Name, UnifiedDiff(m.Expected, m.Got))
}

func CompareGolden(fixture Fixture, expected, got string) error {
	expected = normalizeString(expected)
	got = normalizeString(got)
	if expected == got {
		return nil
	}
	return GoldenMismatch{Fixture: fixture, Expected: expected, Got: got}
}

func UnifiedDiff(expected, got string) string {
	wantLines := splitLines(expected)
	gotLines := splitLines(got)
	var b strings.Builder
	b.WriteString("--- expected\n+++ got\n")
	n := max(len(wantLines), len(gotLines))
	for i := 0; i < n; i++ {
		var want, have string
		if i < len(wantLines) {
			want = wantLines[i]
		}
		if i < len(gotLines) {
			have = gotLines[i]
		}
		if want == have {
			continue
		}
		fmt.Fprintf(&b, "@@ line %d @@\n", i+1)
		if i < len(wantLines) {
			fmt.Fprintf(&b, "-%s\n", want)
		}
		if i < len(gotLines) {
			fmt.Fprintf(&b, "+%s\n", have)
		}
	}
	return b.String()
}

func splitLines(s string) []string {
	s = strings.TrimSuffix(s, "\n")
	if s == "" {
		return nil
	}
	return strings.Split(s, "\n")
}
