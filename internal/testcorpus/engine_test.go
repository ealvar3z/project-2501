package testcorpus

import (
	"os"
	"testing"
)

func TestEngineCorpusPlaceholders(t *testing.T) {
	root, err := Root()
	if err != nil {
		t.Fatal(err)
	}
	suiteFilter := os.Getenv("PM_TEST_SUITE")
	fixtureFilter := os.Getenv("PM_TEST_FIXTURE")
	for _, suite := range []Suite{SuiteLayout, SuiteMD, SuiteJS, SuiteNet, SuiteCharset, SuitePager} {
		if suiteFilter != "" && string(suite) != suiteFilter {
			continue
		}
		fixtures, err := Discover(root, suite)
		if err != nil {
			t.Fatal(err)
		}
		fixtures = Filter(fixtures, fixtureFilter)
		if len(fixtures) == 0 {
			t.Fatalf("%s: no fixtures matched PM_TEST_FIXTURE=%q", suite, fixtureFilter)
		}
		for _, fixture := range fixtures {
			fixture := fixture
			t.Run(string(fixture.Suite)+"/"+fixture.Name, func(t *testing.T) {
				t.Skip("pm render engine is not implemented yet")
			})
		}
	}
}
