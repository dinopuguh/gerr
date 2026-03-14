package spec

// GenOptions holds the options passed to a Generator.
type GenOptions struct {
	SchemaFile string
	OutDir     string
	LocalesDir string
}

// Generator is the interface implemented by each language backend.
// Adding support for a new language means implementing this interface.
type Generator interface {
	// Lang returns the language identifier, e.g. "go", "typescript".
	Lang() string
	// Generate produces output files from the parsed schema.
	Generate(schema Schema, opts GenOptions) error
}
