// gerr is the CLI tool for the gerr library.
//
// Usage:
//
//	gerr <command> [flags]
//
// Commands:
//
//	gen   Generate error code files from a YAML schema
//
// Install:
//
//	go install github.com/dinopuguh/gerr/cmd/gerr@latest
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"maps"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/dinopuguh/gerr/cmd/gerr/codegen"
	"github.com/dinopuguh/gerr/cmd/gerr/spec"
	"gopkg.in/yaml.v3"
)

// generators is the registry of supported language backends.
// To add a new language, implement Generator and register it here.
var generators = map[string]spec.Generator{
	"go":         &codegen.Go{},
	"typescript": &codegen.TypeScript{},
}

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(1)
	}

	switch os.Args[1] {
	case "gen":
		runGen(os.Args[2:])
	default:
		fmt.Fprintf(os.Stderr, "gerr: unknown command %q\n\n", os.Args[1])
		usage()
		os.Exit(1)
	}
}

func runGen(args []string) {
	fs := flag.NewFlagSet("gen", flag.ExitOnError)
	schemaFlag := fs.String("schema", "", `path to YAML schema file, or "-" to read from stdin`)
	schemaURLFlag := fs.String("schema-url", "", "URL to fetch YAML schema from (e.g. raw GitHub URL)")
	outFlag := fs.String("out", ".", "output directory for generated files")
	localesFlag := fs.String("locales", "locales", "output directory for locale JSON files")
	langFlag := fs.String("lang", "go", fmt.Sprintf("target language (%s)", supportedLangs()))
	fs.Usage = func() {
		fmt.Fprintln(os.Stderr, "Usage: gerr gen [-schema <file.yaml>|-schema-url <url>] [-out <dir>] [-locales <dir>] [-lang <lang>]")
		fmt.Fprintln(os.Stderr, "       gerr gen -schema - < errors.yaml  # read from stdin")
		fs.PrintDefaults()
	}
	_ = fs.Parse(args)

	if *schemaFlag != "" && *schemaURLFlag != "" {
		fmt.Fprintln(os.Stderr, "gerr gen: -schema and -schema-url are mutually exclusive")
		os.Exit(1)
	}
	if *schemaFlag == "" && *schemaURLFlag == "" {
		fs.Usage()
		os.Exit(1)
	}

	gen, ok := generators[*langFlag]
	if !ok {
		fmt.Fprintf(os.Stderr, "gerr gen: unsupported language %q (supported: %s)\n", *langFlag, supportedLangs())
		os.Exit(1)
	}

	var (
		data       []byte
		schemaName string
		err        error
	)

	switch {
	case *schemaURLFlag != "":
		data, err = fetchURL(*schemaURLFlag)
		schemaName = filepath.Base(*schemaURLFlag)
		if err != nil {
			fmt.Fprintf(os.Stderr, "gerr gen: %v\n", err)
			os.Exit(1)
		}
	case *schemaFlag == "-":
		data, err = io.ReadAll(os.Stdin)
		schemaName = "stdin"
		if err != nil {
			fmt.Fprintf(os.Stderr, "gerr gen: read stdin: %v\n", err)
			os.Exit(1)
		}
	default:
		data, err = os.ReadFile(*schemaFlag)
		schemaName = filepath.Base(*schemaFlag)
		if err != nil {
			fmt.Fprintf(os.Stderr, "gerr gen: read schema %q: %v\n", *schemaFlag, err)
			os.Exit(1)
		}
	}

	schema, err := parseSchema(data, schemaName)
	if err != nil {
		fmt.Fprintf(os.Stderr, "gerr gen: %v\n", err)
		os.Exit(1)
	}

	opts := spec.GenOptions{
		SchemaFile: schemaName,
		OutDir:     *outFlag,
		LocalesDir: *localesFlag,
	}

	if err := gen.Generate(schema, opts); err != nil {
		fmt.Fprintf(os.Stderr, "gerr gen: %v\n", err)
		os.Exit(1)
	}
	if err := generateLocales(schema, opts.LocalesDir); err != nil {
		fmt.Fprintf(os.Stderr, "gerr gen: %v\n", err)
		os.Exit(1)
	}
}

func fetchURL(url string) ([]byte, error) {
	resp, err := http.Get(url) //nolint:gosec
	if err != nil {
		return nil, fmt.Errorf("fetch %q: %w", url, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("fetch %q: unexpected status %s", url, resp.Status)
	}
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("fetch %q: read body: %w", url, err)
	}
	return data, nil
}

func parseSchema(data []byte, name string) (spec.Schema, error) {
	var schema spec.Schema
	if err := yaml.Unmarshal(data, &schema); err != nil {
		return spec.Schema{}, fmt.Errorf("parse schema %q: %w", name, err)
	}
	if schema.Package == "" {
		return spec.Schema{}, fmt.Errorf("schema %q: missing required field 'package'", name)
	}
	seenCodes := make(map[string]string) // code → name
	seenNames := make(map[string]string) // name → code
	for _, e := range schema.Errors {
		if prev, ok := seenCodes[e.Code]; ok {
			return spec.Schema{}, fmt.Errorf("schema %q: duplicate error code %q (used by %q and %q)", name, e.Code, prev, e.Name)
		}
		if prev, ok := seenNames[e.Name]; ok {
			return spec.Schema{}, fmt.Errorf("schema %q: duplicate error name %q (used by code %q and %q)", name, e.Name, prev, e.Code)
		}
		seenCodes[e.Code] = e.Name
		seenNames[e.Name] = e.Code
	}
	return schema, nil
}

func supportedLangs() string {
	langs := make([]string, 0, len(generators))
	for l := range generators {
		langs = append(langs, l)
	}
	return strings.Join(langs, ", ")
}

func usage() {
	fmt.Fprintln(os.Stderr, "Usage: gerr <command> [flags]")
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, "Commands:")
	fmt.Fprintln(os.Stderr, "  gen   Generate error code files from a YAML schema")
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, "Run 'gerr <command> -help' for command usage.")
}

// generateLocales merges schema messages into per-language JSON files.
// Existing keys are preserved; schema keys are added or overwritten.
func generateLocales(schema spec.Schema, localesDir string) error {
	langMessages := map[string]map[string]string{}
	for _, e := range schema.Errors {
		for lang, msg := range e.Messages {
			if langMessages[lang] == nil {
				langMessages[lang] = map[string]string{}
			}
			langMessages[lang][e.Key] = msg
		}
	}
	for tag, translations := range schema.Validators {
		for lang, msg := range translations {
			if langMessages[lang] == nil {
				langMessages[lang] = map[string]string{}
			}
			langMessages[lang]["validation."+tag] = msg
		}
	}

	if err := os.MkdirAll(localesDir, 0755); err != nil {
		return err
	}

	for lang, messages := range langMessages {
		path := filepath.Join(localesDir, lang+".json")

		existing := map[string]string{}
		if data, err := os.ReadFile(path); err == nil {
			_ = json.Unmarshal(data, &existing)
		}
		maps.Copy(existing, messages)

		out, err := json.MarshalIndent(existing, "", "\t")
		if err != nil {
			return err
		}
		if err := os.WriteFile(path, append(out, '\n'), 0644); err != nil {
			return err
		}
		fmt.Printf("gerr gen: merged %s\n", path)
	}
	return nil
}
