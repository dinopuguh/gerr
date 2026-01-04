package gerr

import (
	"encoding/json"

	"github.com/rs/zerolog"
)

// MarshalZerologObject implements the zerolog.LogObjectMarshaler interface.
func (g *gerr) MarshalZerologObject(evt *zerolog.Event) {
	if g == nil {
		return
	}

	evt.Str("err_code", g.code).Bool("err_retryable", g.retryable).Str("err_message_key", g.messageKey)

	if g.status > 0 {
		evt.Int("err_http_status", g.status)
	}

	// Wrapped underlying error (if any)
	if g.err != nil {
		evt.AnErr("err_cause", g.err)
	}

	if g.args != nil && len(g.args) > 0 {
		argsJson, err := json.Marshal(g.args)
		if err == nil {
			evt.RawJSON("err_args", argsJson)
		}
	}
	if g.metadata != nil && len(g.metadata) > 0 {
		metaJson, err := json.Marshal(g.metadata)
		if err == nil {
			evt.RawJSON("err_metadata", metaJson)
		}
	}
}
