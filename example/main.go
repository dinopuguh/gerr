package main

import (
	"errors"
	"net/http"
	"os"
	"time"

	"github.com/dinopuguh/gerr"
	"github.com/go-playground/validator/v10"
	"github.com/rs/zerolog"
	zlog "github.com/rs/zerolog/log"
	"golang.org/x/text/language"
)

type Payload struct {
	Email string `validate:"required,email"`
	Name  string `validate:"required"`
}

func main() {
	// Configure zerolog to pretty-print to stderr for the example (console output).
	// In production you might prefer JSON output.
	zerolog.TimeFieldFormat = time.RFC3339
	zlog.Logger = zlog.Output(zerolog.ConsoleWriter{Out: os.Stderr, TimeFormat: time.RFC3339})

	// Load i18n bundle from example locale files (client-managed).
	bundle, err := gerr.LoadBundleFromFiles(language.English,
		"example/locales/en.json",
		"example/locales/id.json",
		"example/locales/ar.json",
	)
	if err != nil {
		zlog.Fatal().Err(err).Msg("failed to load i18n bundle")
	}
	// Configure package-level bundle so Localize can be called without passing a bundle.
	gerr.SetBundle(bundle)

	// -------------------------
	// Example: create a simple error
	// -------------------------
	rawErr := gerr.New("USER-001", "error.user_not_found", http.StatusNotFound,
		gerr.WithArgs(map[string]interface{}{"username": "alice"}),
		gerr.WithMetadata(gerr.Metadata{
			"username": "alice",
			"user_id":  123,
			"email":    "alice@example.com",
		}),
	)

	// Localize the user-facing message for logging / display
	msg, _ := gerr.GetUserMessage(rawErr, "en")
	// Structured log: include the error object (our *gerr implements MarshalZerologObject)
	zlog.Error().Err(rawErr).
		Str("operation", "lookup_user").
		Str("user", "alice").
		Str("user_message", msg).
		Msg("lookup finished with error")

	// -------------------------
	// Example: wrap a lower-level error and log it
	// -------------------------
	dbErr := errors.New("sql: no rows in result set")
	rawWrapped := gerr.Wrap(dbErr, "GENERIC-001", "error.unavailable", http.StatusServiceUnavailable,
		gerr.WithArgs(map[string]interface{}{"Resource": "payment"}),
		// explicit override of retryable (optional)
		gerr.WithRetryable(true),
	)
	userMsg, _ := gerr.GetUserMessage(rawWrapped, "en")
	zlog.Error().Err(rawWrapped).
		Str("operation", "query_db").
		Str("service", "payment").
		Str("user_message", userMsg).
		Msg("database query failed")

	// -------------------------
	// Example: validator conversion and logging
	// -------------------------
	validate := validator.New()
	payload := Payload{
		Email: "not-an-email", // invalid email
		Name:  "",             // missing required
	}
	if err := validate.Struct(payload); err != nil {
		vErr := gerr.FromValidatorErr(err) // returns error
		if vErr != nil {
			enMsg, _ := gerr.GetUserMessage(vErr, "en")
			idMsg, _ := gerr.GetUserMessage(vErr, "id")
			arMsg, _ := gerr.GetUserMessage(vErr, "ar")

			zlog.Warn().Err(vErr).
				Str("operation", "validate_payload").
				Str("message_en", enMsg).
				Str("message_id", idMsg).
				Str("message_ar", arMsg).
				Msg("payload validation failed")
		}
	}
}
