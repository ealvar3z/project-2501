package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"sort"
	"strings"
	"time"
)

type goModule struct {
	Path      string
	Version   string
	Main      bool
	Indirect  bool
	Replace   *goModule
	Time      *time.Time
	GoVersion string
	Sum       string
	GoModSum  string
}

type component struct {
	Type       string     `json:"type"`
	BOMRef     string     `json:"bom-ref"`
	Name       string     `json:"name"`
	Version    string     `json:"version,omitempty"`
	Scope      string     `json:"scope,omitempty"`
	PURL       string     `json:"purl,omitempty"`
	Properties []property `json:"properties,omitempty"`
}

type property struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

type dependency struct {
	Ref       string   `json:"ref"`
	DependsOn []string `json:"dependsOn,omitempty"`
}

type metadata struct {
	Timestamp string    `json:"timestamp"`
	Tools     []tool    `json:"tools"`
	Component component `json:"component"`
}

type tool struct {
	Vendor  string `json:"vendor"`
	Name    string `json:"name"`
	Version string `json:"version"`
}

type bom struct {
	Schema       string       `json:"$schema"`
	BOMFormat    string       `json:"bomFormat"`
	SpecVersion  string       `json:"specVersion"`
	SerialNumber string       `json:"serialNumber"`
	Version      int          `json:"version"`
	Metadata     metadata     `json:"metadata"`
	Components   []component  `json:"components"`
	Dependencies []dependency `json:"dependencies"`
}

func main() {
	out := flag.String("o", "bom.cdx.json", "output CycloneDX JSON file")
	flag.Parse()
	modules, err := listModules()
	if err != nil {
		fatal(err)
	}
	deps, err := listDeps()
	if err != nil {
		fatal(err)
	}
	doc, err := buildBOM(modules, deps)
	if err != nil {
		fatal(err)
	}
	b, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		fatal(err)
	}
	b = append(b, '\n')
	if err := os.WriteFile(*out, b, 0o644); err != nil {
		fatal(err)
	}
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, "sbom:", err)
	os.Exit(1)
}

func listModules() ([]goModule, error) {
	cmd := exec.Command("go", "list", "-m", "-json", "all")
	out, err := cmd.Output()
	if err != nil {
		return nil, commandError(cmd, err)
	}
	dec := json.NewDecoder(bytes.NewReader(out))
	var modules []goModule
	for dec.More() {
		var mod goModule
		if err := dec.Decode(&mod); err != nil {
			return nil, err
		}
		modules = append(modules, mod)
	}
	return modules, nil
}

func listDeps() (map[string][]string, error) {
	cmd := exec.Command("go", "mod", "graph")
	out, err := cmd.Output()
	if err != nil {
		return nil, commandError(cmd, err)
	}
	deps := map[string][]string{}
	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		if line == "" {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) != 2 {
			return nil, fmt.Errorf("unexpected go mod graph line %q", line)
		}
		from := refFromGraphNode(fields[0])
		to := refFromGraphNode(fields[1])
		deps[from] = append(deps[from], to)
	}
	for ref := range deps {
		sort.Strings(deps[ref])
		deps[ref] = compact(deps[ref])
	}
	return deps, nil
}

func commandError(cmd *exec.Cmd, err error) error {
	if ee, ok := err.(*exec.ExitError); ok {
		return fmt.Errorf("%s: %w\n%s", strings.Join(cmd.Args, " "), err, ee.Stderr)
	}
	return fmt.Errorf("%s: %w", strings.Join(cmd.Args, " "), err)
}

func buildBOM(modules []goModule, deps map[string][]string) (bom, error) {
	if len(modules) == 0 || !modules[0].Main {
		return bom{}, fmt.Errorf("go list did not return a main module first")
	}
	main := modules[0]
	mainComponent := component{
		Type:   "application",
		BOMRef: moduleRef(main),
		Name:   main.Path,
		Properties: []property{
			{Name: "go:main", Value: "true"},
			{Name: "go:version", Value: main.GoVersion},
		},
	}
	var components []component
	for _, mod := range modules[1:] {
		components = append(components, componentForModule(mod))
	}
	sort.Slice(components, func(i, j int) bool {
		return components[i].BOMRef < components[j].BOMRef
	})
	allRefs := map[string]bool{mainComponent.BOMRef: true}
	for _, c := range components {
		allRefs[c.BOMRef] = true
	}
	var dependencies []dependency
	for ref, dependsOn := range deps {
		if !allRefs[ref] {
			continue
		}
		var filtered []string
		for _, dep := range dependsOn {
			if allRefs[dep] {
				filtered = append(filtered, dep)
			}
		}
		dependencies = append(dependencies, dependency{Ref: ref, DependsOn: filtered})
	}
	sort.Slice(dependencies, func(i, j int) bool {
		return dependencies[i].Ref < dependencies[j].Ref
	})
	return bom{
		Schema:       "http://cyclonedx.org/schema/bom-1.5.schema.json",
		BOMFormat:    "CycloneDX",
		SpecVersion:  "1.5",
		SerialNumber: serialNumber(mainComponent, components),
		Version:      1,
		Metadata: metadata{
			Timestamp: time.Now().UTC().Format(time.RFC3339),
			Tools: []tool{{
				Vendor:  "project2501",
				Name:    "tools/sbom",
				Version: "0.1.0",
			}},
			Component: mainComponent,
		},
		Components:   components,
		Dependencies: dependencies,
	}, nil
}

func componentForModule(mod goModule) component {
	properties := []property{
		{Name: "go:indirect", Value: fmt.Sprint(mod.Indirect)},
	}
	if mod.GoVersion != "" {
		properties = append(properties, property{Name: "go:version", Value: mod.GoVersion})
	}
	if mod.Sum != "" {
		properties = append(properties, property{Name: "go:sum", Value: mod.Sum})
	}
	if mod.GoModSum != "" {
		properties = append(properties, property{Name: "go:modSum", Value: mod.GoModSum})
	}
	if mod.Time != nil {
		properties = append(properties, property{Name: "go:moduleTime", Value: mod.Time.UTC().Format(time.RFC3339)})
	}
	sort.Slice(properties, func(i, j int) bool {
		return properties[i].Name < properties[j].Name
	})
	return component{
		Type:       "library",
		BOMRef:     moduleRef(mod),
		Name:       mod.Path,
		Version:    mod.Version,
		Scope:      "required",
		PURL:       purl(mod),
		Properties: properties,
	}
}

func moduleRef(mod goModule) string {
	if mod.Version == "" {
		return mod.Path
	}
	return mod.Path + "@" + mod.Version
}

func purl(mod goModule) string {
	if mod.Version == "" {
		return "pkg:golang/" + mod.Path
	}
	return "pkg:golang/" + mod.Path + "@" + mod.Version
}

func refFromGraphNode(node string) string {
	if i := strings.IndexByte(node, '@'); i >= 0 {
		return node[:i] + "@" + node[i+1:]
	}
	return node
}

func compact(in []string) []string {
	out := in[:0]
	for i, s := range in {
		if i == 0 || s != in[i-1] {
			out = append(out, s)
		}
	}
	return out
}

func serialNumber(main component, components []component) string {
	h := sha256.New()
	_, _ = h.Write([]byte(main.BOMRef))
	for _, c := range components {
		_, _ = h.Write([]byte{0})
		_, _ = h.Write([]byte(c.BOMRef))
	}
	sum := hex.EncodeToString(h.Sum(nil))
	return fmt.Sprintf("urn:uuid:%s-%s-%s-%s-%s", sum[:8], sum[8:12], sum[12:16], sum[16:20], sum[20:32])
}
