package pmapp

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

type recordingEngine struct {
	params Params
	config Config
	code   int
}

func (e *recordingEngine) Run(params Params, config Config) int {
	e.params = params
	e.config = config
	return e.code
}

func TestParsePMArgs(t *testing.T) {
	params, help, version, err := parsePMArgs([]string{
		"-dCtest.toml",
		"-I", "iso-2022-jp",
		"-M",
		"-Outf-8",
		"-T", "text/html",
		"-c", "a{color:red}",
		"-o", "buffer.images=true",
		"-r", "quit()",
		"--",
		"-literal",
	})
	if err != nil {
		t.Fatal(err)
	}
	if help || version {
		t.Fatalf("help=%v version=%v, want false", help, version)
	}
	if params.ConfigPath != "test.toml" {
		t.Fatalf("ConfigPath = %q", params.ConfigPath)
	}
	if !params.Dump || !params.Monochrome {
		t.Fatalf("Dump=%v Monochrome=%v, want true", params.Dump, params.Monochrome)
	}
	if params.InputCharset != "iso-2022-jp" || params.OutputCharset != "utf-8" {
		t.Fatalf("charsets = %q/%q", params.InputCharset, params.OutputCharset)
	}
	if params.ContentType != "text/html" || params.Stylesheet != "a{color:red}" || params.RunScript != "quit()" {
		t.Fatalf("params not populated: %#v", params)
	}
	if len(params.Pages) != 1 || params.Pages[0] != "-literal" {
		t.Fatalf("Pages = %#v", params.Pages)
	}
}

func TestRunInitializesConfigAndEngine(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.toml")
	if err := os.WriteFile(configPath, []byte("[start]\nvisual-home = 'about:test'\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	engine := &recordingEngine{}
	var stdout, stderr bytes.Buffer
	err := Run([]string{"-C", configPath, "-V"}, bytes.NewBuffer(nil), &stdout, &stderr, engine)
	if err != nil {
		t.Fatal(err)
	}
	if engine.config.VisualHome != "about:test" {
		t.Fatalf("VisualHome = %q", engine.config.VisualHome)
	}
	if got := os.Getenv("PM_DIR"); got != dir {
		t.Fatalf("PM_DIR = %q, want %q", got, dir)
	}
}

func TestHelpAndVersion(t *testing.T) {
	for _, args := range [][]string{{"-h"}, {"--version"}} {
		var stdout, stderr bytes.Buffer
		if err := Run(args, bytes.NewBuffer(nil), &stdout, &stderr, &recordingEngine{}); err != nil {
			t.Fatalf("Run(%v): %v", args, err)
		}
		if stdout.Len() == 0 {
			t.Fatalf("Run(%v) wrote no stdout", args)
		}
	}
}

func TestMain2ReportsNotImplementedForMissingEngine(t *testing.T) {
	err := main2(Runtime{}, []string{"-r", "notQuit()"}, bytes.NewBuffer(nil), &bytes.Buffer{}, &bytes.Buffer{}, nil)
	if !errors.Is(err, errNotImplemented) {
		t.Fatalf("main2 error = %v, want not-implemented", err)
	}
}

func TestRunReportsNotImplementedForStubEnginePageLoad(t *testing.T) {
	err := Run([]string{"https://example.test"}, bytes.NewBuffer(nil), &bytes.Buffer{}, &bytes.Buffer{}, stubEngine{})
	if !errors.Is(err, errNotImplemented) {
		t.Fatalf("Run error = %v, want not-implemented", err)
	}
}
