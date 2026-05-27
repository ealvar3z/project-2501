//go:build ignore

package main

import (
	"fmt"
	"os"
	"strconv"
	"time"

	"golang.org/x/text/encoding/htmlindex"
)

func main() {
	file := os.Getenv("BENCH_FILE")
	label := os.Getenv("BENCH_CHARSET")
	iter, err := strconv.Atoi(os.Getenv("BENCH_ITER"))
	check(err)
	failOutDir := os.Getenv("BENCH_ERROR_OUTDIR")
	enc, err := htmlindex.Get(label)
	check(err)
	input, err := os.ReadFile(file)
	check(err)
	decoded, err := enc.NewDecoder().Bytes(input)
	check(err)
	encoded, err := enc.NewEncoder().Bytes(decoded)
	check(err)
	if string(encoded) != string(input) {
		if failOutDir != "" {
			_ = os.WriteFile("/tmp/bench_fail_output0", decoded, 0o644)
			_ = os.WriteFile("/tmp/bench_fail_output1", encoded, 0o644)
		}
		fmt.Fprintln(os.Stderr, "ERROR: equivalence check failed")
		os.Exit(1)
	}
	fmt.Println("Starting benchmark for", file, "charset", label)
	startAll := time.Now()
	var total time.Duration
	var low time.Duration
	var high time.Duration
	for i := 0; i < iter; i++ {
		start := time.Now()
		res, err := enc.NewDecoder().Bytes(input)
		check(err)
		_, _ = os.Stdout.Write(nil)
		_ = res
		elapsed := time.Since(start)
		if i == 0 || elapsed < low {
			low = elapsed
		}
		if elapsed > high {
			high = elapsed
		}
		total += elapsed
	}
	fmt.Println("Done in", time.Since(startAll), "avg", total/time.Duration(iter), "lowest", low, "highest", high)
}

func check(err error) {
	if err != nil {
		panic(err)
	}
}
