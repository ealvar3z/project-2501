//go:build ignore

package main

import (
	"fmt"
	"os"
	"path/filepath"

	"golang.org/x/text/encoding/htmlindex"
)

func main() {
	dir := os.Getenv("CGS_TESTDIR")
	runTest(dir, "big5", "big5", false)
	runTest(dir, "euc_kr", "euc-kr", false)
	runTest(dir, "jis0208", "euc-jp", false)
	runTest(dir, "jis0212", "euc-jp", true)
	runTest(dir, "gb18030", "gb18030", false)
	runTest(dir, "iso_2022_jp", "iso-2022-jp", false)
	runTest(dir, "shift_jis", "shift_jis", false)
}

func runTest(dir, name, label string, noOut bool) {
	runDecodeCheck(dir, name, label)
	if !noOut {
		runEncodeCheck(dir, name, label)
	}
}

func runDecodeCheck(dir, name, label string) {
	enc, err := htmlindex.Get(label)
	check(err)
	in := read(filepath.Join(dir, name+"_in.txt"))
	got, err := enc.NewDecoder().String(string(in))
	check(err)
	want := string(read(filepath.Join(dir, name+"_in_ref.txt")))
	if got != want {
		_ = os.WriteFile(filepath.Join(dir, "fail_in_"+label), []byte(got), 0o644)
		panic(fmt.Sprintf("decode mismatch for %s: got %d bytes, want %d", label, len(got), len(want)))
	}
}

func runEncodeCheck(dir, name, label string) {
	enc, err := htmlindex.Get(label)
	check(err)
	in := read(filepath.Join(dir, name+"_out.txt"))
	got, err := enc.NewEncoder().Bytes(in)
	check(err)
	want := read(filepath.Join(dir, name+"_out_ref.txt"))
	if string(got) != string(want) {
		_ = os.WriteFile(filepath.Join(dir, "fail_out_"+label), got, 0o644)
		panic(fmt.Sprintf("encode mismatch for %s: got %d bytes, want %d", label, len(got), len(want)))
	}
}

func read(path string) []byte {
	b, err := os.ReadFile(path)
	check(err)
	return b
}

func check(err error) {
	if err != nil {
		panic(err)
	}
}
