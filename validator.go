package gerr

import (
	"errors"
	"fmt"

	"github.com/go-playground/validator/v10"
	"github.com/nicksnyder/go-i18n/v2/i18n"
	"golang.org/x/text/language"
)

// FromValidatorErr converts an error produced by github.com/go-playground/validator/v10
// into a gerr error (returns the built-in error interface).
// Behavior:
//   - If err is nil, returns nil.
//   - If err is a validator.ValidationErrors (or contains it in its chain),
//     this uses only the first FieldError to construct an error with:
//   - Code = "validation.failed"
//   - MessageKey = "validation.<tag>" (e.g. "validation.required")
//   - HTTP status = 400
//   - Retryable = false
//   - Args and Meta populated with field/tag/param info
//   - If err is not a validation error, returns nil.
func FromValidatorErr(err error) error {
	if err == nil {
		return nil
	}

	// Direct assertion: validator.ValidationErrors is a slice of FieldError.
	if ve, ok := err.(validator.ValidationErrors); ok {
		if len(ve) == 0 {
			return nil
		}
		fe := ve[0]
		return buildErrorFromFieldError(err, fe)
	}

	// Try to find ValidationErrors in the error chain.
	var vePtr validator.ValidationErrors
	if errors.As(err, &vePtr) {
		if len(vePtr) == 0 {
			return nil
		}
		fe := vePtr[0]
		return buildErrorFromFieldError(err, fe)
	}

	// Not a validator.ValidationErrors instance.
	return nil
}

// ValidatorMessageKeyForFieldError returns the message key that should be used
// for the provided FieldError. The convention used is "validation.<tag>".
// If the tag is empty, falls back to "validation.failed".
func ValidatorMessageKeyForFieldError(fe validator.FieldError) string {
	if fe == nil {
		return "validation.failed"
	}
	tag := fe.Tag()
	if tag == "" {
		return "validation.failed"
	}
	return "validation." + tag
}

// buildErrorFromFieldError constructs an error (gerr) from a single
// validator.FieldError. It sets Code="validation.failed", MessageKey based on
// the tag (e.g. "validation.required"), populates Args with useful template
// data, fills Meta with a small subset for programmatic checks, sets Status=400,
// and attaches the original validation error as the wrapped error.
func buildErrorFromFieldError(origErr error, fe validator.FieldError) error {
	if fe == nil {
		g := newGerr("validation.failed", "validation.failed", 400)
		g.err = origErr
		g.retryable = false
		return g
	}

	args := map[string]any{
		"Field":           fe.Field(),
		"Tag":             fe.Tag(),
		"Param":           fe.Param(),
		"Value":           fe.Value(),
		"Namespace":       fe.Namespace(),
		"StructNamespace": fe.StructNamespace(),
		"ActualTag":       fe.ActualTag(),
	}

	msgKey := ValidatorMessageKeyForFieldError(fe)

	// Use options to populate args; newGerr returns *gerr which implements error.
	g := newGerr("validation.failed", msgKey, 400, WithArgs(args))
	g.err = origErr
	g.retryable = false

	// populate metadata
	if g.metadata == nil {
		g.metadata = make(map[string]any)
	}
	g.metadata["field"] = fe.Field()
	g.metadata["tag"] = fe.Tag()
	g.metadata["param"] = fe.Param()

	return g
}

// RegisterValidatorTagMessages registers a map of validator tag -> template text
// into the provided i18n.Bundle for the given language. Each entry will be
// registered under the message ID "validation.<tag>".
//
// Example:
//
//	RegisterValidatorTagMessages(bundle, "en", map[string]string{
//	  "required": "{{.Field}} is required",
//	  "email": "{{.Field}} must be a valid email address",
//	})
func RegisterValidatorTagMessages(bundle *i18n.Bundle, lang string, tagMessages map[string]string) error {
	if bundle == nil {
		return fmt.Errorf("bundle is nil")
	}
	if lang == "" {
		return fmt.Errorf("lang must be provided")
	}
	if len(tagMessages) == 0 {
		return nil
	}

	langs := language.Make(lang)

	for tag, tmpl := range tagMessages {
		if tag == "" {
			continue
		}
		id := "validation." + tag
		msg := &i18n.Message{
			ID:    id,
			Other: tmpl,
		}
		if err := bundle.AddMessages(langs, msg); err != nil {
			return fmt.Errorf("add validator message %q: %w", id, err)
		}
	}
	return nil
}

// RegisterMessages registers arbitrary messageID->template pairs into the
// provided bundle for the given language. This helper is generic and can be
// used to register any message IDs.
//
// Example:
//
//	RegisterMessages(bundle, "en", map[string]string{
//	  "error.unavailable": "Service '{{.Resource}}' is unavailable",
//	})
func RegisterMessages(bundle *i18n.Bundle, lang string, messages map[string]string) error {
	if bundle == nil {
		return fmt.Errorf("bundle is nil")
	}
	if lang == "" {
		return fmt.Errorf("lang must be provided")
	}
	if len(messages) == 0 {
		return nil
	}

	langs := language.Make(lang)
	for id, tmpl := range messages {
		if id == "" {
			continue
		}
		msg := &i18n.Message{
			ID:    id,
			Other: tmpl,
		}
		if err := bundle.AddMessages(langs, msg); err != nil {
			return fmt.Errorf("add message %q: %w", id, err)
		}
	}
	return nil
}
