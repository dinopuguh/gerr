package gerr

// Option configures a newly created gerr instance.
type Option func(*gerr)

// WithArgs sets the template args to be used for localization.
// It overwrites any existing args map.
func WithArgs(args map[string]any) Option {
	return func(g *gerr) {
		if g == nil {
			return
		}
		if args == nil {
			g.args = make(map[string]any)
			return
		}
		g.args = args
	}
}

type Metadata map[string]any

func (m *Metadata) Add(key string, value any) {
	if m == nil {
		return
	}
	if *m == nil {
		*m = make(map[string]any)
	}
	(*m)[key] = value
}

// WithMetadata adds a metadata key/value pair.
// It appends the metadata to the existing metadata map.
func WithMetadata(metadata Metadata) Option {
	return func(g *gerr) {
		if g == nil {
			return
		}
		if g.metadata == nil {
			g.metadata = make(Metadata)
		}
		for k, v := range metadata {
			g.metadata[k] = v
		}
	}
}

// WithRetryable explicitly sets retryable flag (overrides HTTP-based default).
func WithRetryable(retryable bool) Option {
	return func(g *gerr) {
		if g == nil {
			return
		}
		g.retryable = retryable
	}
}
