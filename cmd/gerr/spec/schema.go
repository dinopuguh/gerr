package spec

// Schema is the top-level structure of an error schema YAML file.
type Schema struct {
	Package    string                       `yaml:"package"`
	Errors     []Error                      `yaml:"errors"`
	Validators map[string]map[string]string `yaml:"validators"`
}

// Error defines a single error entry in the schema.
type Error struct {
	Name      string            `yaml:"name"`
	Code      string            `yaml:"code"`
	Key       string            `yaml:"key"`
	HTTP      int               `yaml:"http"`
	Args      []string          `yaml:"args"`
	Messages  map[string]string `yaml:"messages"`
	Retryable *bool             `yaml:"retryable"`
}
