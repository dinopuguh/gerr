package main

import (
	"errors"
	"os"
	"time"

	"github.com/dinopuguh/gerr"
	apierrors "github.com/dinopuguh/gerr/example/gen"
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
	zerolog.TimeFieldFormat = time.RFC3339
	zlog.Logger = zlog.Output(zerolog.ConsoleWriter{Out: os.Stderr, TimeFormat: time.RFC3339})

	if err := gerr.InitBundle(language.English, "example/gen/locales/"); err != nil {
		zlog.Fatal().Err(err).Msg("failed to init i18n bundle")
	}

	// -------------------------
	// Example: generated error with args
	// -------------------------
	rawErr := apierrors.UserNotFound("alice",
		gerr.WithMetadata(gerr.Metadata{"user_id": 123}),
	)
	msg, _ := gerr.GetUserMessage(rawErr, "en")
	zlog.Error().Err(rawErr).Str("user_message", msg).Msg("user lookup failed")

	// -------------------------
	// Example: wrap a lower-level error using a generated code constant
	// -------------------------
	dbErr := errors.New("sql: no rows in result set")
	wrapped := gerr.Wrap(dbErr, apierrors.CodeServiceUnavailable, "error.service_unavailable", 503,
		gerr.WithRetryable(true),
	)
	msg, _ = gerr.GetUserMessage(wrapped, "en")
	zlog.Error().Err(wrapped).Str("user_message", msg).Msg("database query failed")

	// -------------------------
	// Example: generated error without args
	// -------------------------
	authErr := apierrors.Unauthorized()
	msg, _ = gerr.GetUserMessage(authErr, "id")
	zlog.Warn().Err(authErr).Str("user_message", msg).Msg("unauthorized access")

	// -------------------------
	// Example: validator integration
	// -------------------------
	validate := validator.New()
	payload := Payload{Email: "not-an-email", Name: ""}
	if err := validate.Struct(payload); err != nil {
		vErr := gerr.FromValidatorErr(err)
		if vErr != nil {
			enMsg, _ := gerr.GetUserMessage(vErr, "en")
			idMsg, _ := gerr.GetUserMessage(vErr, "id")
			zlog.Warn().Err(vErr).
				Str("message_en", enMsg).
				Str("message_id", idMsg).
				Msg("validation failed")
		}
	}
}
