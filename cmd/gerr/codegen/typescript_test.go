package codegen

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/dinopuguh/gerr/cmd/gerr/spec"
)

func TestTypeScriptGenerator_Lang(t *testing.T) {
	g := &TypeScript{}
	if g.Lang() != "typescript" {
		t.Errorf("Lang() = %q, want %q", g.Lang(), "typescript")
	}
}

func TestTypeScriptGenerator_Generate(t *testing.T) {
	dir := t.TempDir()

	schema := spec.Schema{
		Package: "apierrors",
		Errors: []spec.Error{
			{
				Name: "NotFound",
				Code: "NF-001",
				Key:  "error.not_found",
				HTTP: 404,
				Args: []string{"id"},
				Messages: map[string]string{
					"en": "Resource {{.id}} not found",
					"id": "Sumber daya {{.id}} tidak ditemukan",
				},
			},
			{
				Name: "Unauthorized",
				Code: "AUTH-001",
				Key:  "error.unauthorized",
				HTTP: 401,
				Messages: map[string]string{
					"en": "Unauthorized",
				},
			},
		},
	}

	opts := spec.GenOptions{
		SchemaFile: "errors.yaml",
		OutDir:     dir,
		LocalesDir: filepath.Join(dir, "locales"),
	}

	g := &TypeScript{}
	if err := g.Generate(schema, opts); err != nil {
		t.Fatalf("Generate() error: %v", err)
	}

	tsFile := filepath.Join(dir, "errors.ts")
	data, err := os.ReadFile(tsFile)
	if err != nil {
		t.Fatalf("generated file not found: %v", err)
	}
	src := string(data)

	for _, want := range []string{
		"DO NOT EDIT",
		"export interface GerrError",
		`NotFound: "NF-001"`,
		`Unauthorized: "AUTH-001"`,
		"export function NotFound(id: string)",
		"export function Unauthorized()",
		`"error.not_found"`,
		`"error.unauthorized"`,
		"404",
		"401",
	} {
		if !strings.Contains(src, want) {
			t.Errorf("generated file missing %q\nfull output:\n%s", want, src)
		}
	}
}
