package testcorpus

import (
	"context"
	"path/filepath"
	"testing"

	"golang.org/x/tools/txtar"
	"rsc.io/script"
	"rsc.io/script/scripttest"
)

func TestCorpusManifest(t *testing.T) {
	root, err := Root()
	if err != nil {
		t.Fatal(err)
	}
	counts, err := CountTopLevelFiles(root)
	if err != nil {
		t.Fatal(err)
	}
	want := map[Suite]int{
		SuiteCharset: 7,
		SuiteGo:      1,
		SuiteJS:      48,
		SuiteLayout:  567,
		SuiteMD:      27,
		SuiteNet:     15,
		SuitePager:   8,
	}
	for suite, n := range want {
		if counts[suite] != n {
			t.Fatalf("%s: got %d top-level files, want %d", suite, counts[suite], n)
		}
	}
}

func TestPairedExpectedFiles(t *testing.T) {
	root, err := Root()
	if err != nil {
		t.Fatal(err)
	}
	for _, suite := range []Suite{SuiteLayout, SuiteMD} {
		fixtures, err := Discover(root, suite)
		if err != nil {
			t.Fatal(err)
		}
		for _, fixture := range fixtures {
			if fixture.Expected == "" {
				t.Fatalf("%s/%s has no expected path", suite, fixture.Name)
			}
			if _, err := ReadExpected(fixture); err != nil {
				t.Fatalf("%s/%s: %v", suite, fixture.Name, err)
			}
		}
	}
}

func TestSharedExpectedSuites(t *testing.T) {
	root, err := Root()
	if err != nil {
		t.Fatal(err)
	}
	for _, suite := range []Suite{SuiteJS, SuiteNet} {
		fixtures, err := Discover(root, suite)
		if err != nil {
			t.Fatal(err)
		}
		if len(fixtures) == 0 {
			t.Fatalf("%s: no fixtures discovered", suite)
		}
		for _, fixture := range fixtures {
			if filepath.Base(fixture.Expected) != "all.expected" {
				t.Fatalf("%s/%s: got expected %q, want all.expected", suite, fixture.Name, fixture.Expected)
			}
			if _, err := ReadExpected(fixture); err != nil {
				t.Fatalf("%s/%s: %v", suite, fixture.Name, err)
			}
		}
	}
}

func TestFixtureFiltering(t *testing.T) {
	root, err := Root()
	if err != nil {
		t.Fatal(err)
	}
	fixtures, err := Discover(root, SuiteLayout)
	if err != nil {
		t.Fatal(err)
	}
	filtered := Filter(fixtures, "flex-flow")
	if len(filtered) != 1 || filtered[0].Name != "flex-flow" {
		t.Fatalf("Filter(..., flex-flow) = %#v", filtered)
	}
}

func TestGoldenMismatchIsReadable(t *testing.T) {
	fixture := Fixture{Suite: SuiteLayout, Name: "example"}
	err := CompareGolden(fixture, "one\ntwo\n", "one\nthree\n")
	if err == nil {
		t.Fatal("CompareGolden unexpectedly succeeded")
	}
	const want = "layout/example output mismatch\n--- expected\n+++ got\n@@ line 2 @@\n-two\n+three\n"
	if err.Error() != want {
		t.Fatalf("mismatch text:\n%s", err)
	}
}

func TestTxtarCaseShape(t *testing.T) {
	const archive = `pm multi-file case

-- index.html --
<!doctype html>
<link rel="stylesheet" href="style.css">
<p>Hello</p>
-- style.css --
p { color: red }
-- expected.txt --
Hello
`
	a := txtar.Parse([]byte(archive))
	if len(a.Files) != 3 {
		t.Fatalf("txtar file count = %d, want 3", len(a.Files))
	}
	if string(a.Files[2].Data) != "Hello\n" {
		t.Fatalf("expected.txt = %q", a.Files[2].Data)
	}
}

func TestScriptCaseShape(t *testing.T) {
	engine := script.NewEngine()
	engine.Cmds = scripttest.DefaultCmds()
	engine.Conds = scripttest.DefaultConds()
	scripttest.Test(t, context.Background(), engine, nil, "testdata/script/*.txt")
}
