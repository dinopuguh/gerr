package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/dinopuguh/gerr/cmd/gerr/spec"
)

func TestGenerateLocales(t *testing.T) {
	dir := t.TempDir()

	schema := spec.Schema{
		Package: "apierrors",
		Errors: []spec.Error{
			{
				Name: "NotFound",
				Code: "NF-001",
				Key:  "error.not_found",
				HTTP: 404,
				Messages: map[string]string{
					"en": "Resource not found",
					"id": "Sumber daya tidak ditemukan",
				},
			},
		},
		Validators: map[string]map[string]string{
			"required": {
				"en": "The field '{{.Field}}' is required",
			},
		},
	}

	if err := generateLocales(schema, dir); err != nil {
		t.Fatalf("generateLocales() error: %v", err)
	}

	for _, lang := range []string{"en", "id"} {
		path := filepath.Join(dir, lang+".json")
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("locale %s not found: %v", lang, err)
		}
		var messages map[string]string
		if err := json.Unmarshal(data, &messages); err != nil {
			t.Fatalf("locale %s invalid JSON: %v", lang, err)
		}
		if _, ok := messages["error.not_found"]; !ok {
			t.Errorf("locale %s missing key %q", lang, "error.not_found")
		}
	}

	enData, _ := os.ReadFile(filepath.Join(dir, "en.json"))
	var en map[string]string
	_ = json.Unmarshal(enData, &en)
	if en["validation.required"] == "" {
		t.Error("expected validation.required key in en.json")
	}
}

func TestGenerateLocales_Merge(t *testing.T) {
	dir := t.TempDir()

	existing := map[string]string{"existing.key": "existing value"}
	data, _ := json.MarshalIndent(existing, "", "\t")
	if err := os.WriteFile(filepath.Join(dir, "en.json"), append(data, '\n'), 0644); err != nil {
		t.Fatal(err)
	}

	schema := spec.Schema{
		Package: "apierrors",
		Errors: []spec.Error{
			{Name: "NotFound", Code: "NF-001", Key: "error.not_found", HTTP: 404,
				Messages: map[string]string{"en": "Not found"}},
		},
	}

	if err := generateLocales(schema, dir); err != nil {
		t.Fatalf("generateLocales() error: %v", err)
	}

	out, _ := os.ReadFile(filepath.Join(dir, "en.json"))
	var messages map[string]string
	if err := json.Unmarshal(out, &messages); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if messages["existing.key"] != "existing value" {
		t.Error("existing key was overwritten during merge")
	}
	if messages["error.not_found"] == "" {
		t.Error("new key error.not_found missing after merge")
	}
}

func TestParseSchema_Valid(t *testing.T) {
	yaml := []byte(`
package: mypkg
errors:
  - name: NotFound
    code: NF-001
    key: error.not_found
    http: 404
`)
	schema, err := parseSchema(yaml, "test.yaml")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if schema.Package != "mypkg" {
		t.Errorf("package = %q, want %q", schema.Package, "mypkg")
	}
	if len(schema.Errors) != 1 {
		t.Errorf("errors len = %d, want 1", len(schema.Errors))
	}
}

func TestParseSchema_MissingPackage(t *testing.T) {
	yaml := []byte(`
errors:
  - name: NotFound
    code: NF-001
    key: error.not_found
    http: 404
`)
	_, err := parseSchema(yaml, "test.yaml")
	if err == nil {
		t.Fatal("expected error for missing package, got nil")
	}
}

func TestParseSchema_DuplicateCode(t *testing.T) {
	yaml := []byte(`
package: mypkg
errors:
  - name: NotFound
    code: ERR-001
    key: error.not_found
    http: 404
  - name: Unauthorized
    code: ERR-001
    key: error.unauthorized
    http: 401
`)
	_, err := parseSchema(yaml, "test.yaml")
	if err == nil {
		t.Fatal("expected error for duplicate code, got nil")
	}
}

func TestParseSchema_DuplicateName(t *testing.T) {
	yaml := []byte(`
package: mypkg
errors:
  - name: NotFound
    code: ERR-001
    key: error.not_found
    http: 404
  - name: NotFound
    code: ERR-002
    key: error.not_found_2
    http: 404
`)
	_, err := parseSchema(yaml, "test.yaml")
	if err == nil {
		t.Fatal("expected error for duplicate name, got nil")
	}
}

func TestParseSchema_InvalidYAML(t *testing.T) {
	yaml := []byte(`{invalid: [yaml`)
	_, err := parseSchema(yaml, "test.yaml")
	if err == nil {
		t.Fatal("expected error for invalid YAML, got nil")
	}
}

func TestParseSchema_WithValidators(t *testing.T) {
	yaml := []byte(`
package: mypkg
validators:
  required:
    en: "The field '{{.Field}}' is required"
    id: "Kolom '{{.Field}}' wajib diisi"
errors:
  - name: NotFound
    code: NF-001
    key: error.not_found
    http: 404
`)
	schema, err := parseSchema(yaml, "test.yaml")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(schema.Validators) != 1 {
		t.Errorf("validators len = %d, want 1", len(schema.Validators))
	}
	if schema.Validators["required"]["en"] == "" {
		t.Error("expected validators.required.en to be set")
	}
}
