package codegen

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/dinopuguh/gerr/cmd/gerr/spec"
)

func TestGoGenerator_Lang(t *testing.T) {
	g := &Go{}
	if g.Lang() != "go" {
		t.Errorf("Lang() = %q, want %q", g.Lang(), "go")
	}
}

func TestGoGenerator_Generate(t *testing.T) {
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

	g := &Go{}
	if err := g.Generate(schema, opts); err != nil {
		t.Fatalf("Generate() error: %v", err)
	}

	goFile := filepath.Join(dir, "errors.go")
	data, err := os.ReadFile(goFile)
	if err != nil {
		t.Fatalf("generated file not found: %v", err)
	}
	src := string(data)

	for _, want := range []string{
		"package apierrors",
		`CodeNotFound`,
		`"NF-001"`,
		`CodeUnauthorized`,
		`"AUTH-001"`,
		"func NotFound(id string",
		"func Unauthorized(",
		"http.StatusNotFound",
		"http.StatusUnauthorized",
	} {
		if !strings.Contains(src, want) {
			t.Errorf("generated file missing %q", want)
		}
	}
}
