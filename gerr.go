package gerr

import (
	"errors"
	"fmt"

	"github.com/nicksnyder/go-i18n/v2/i18n"
)

// defaultBundle is the package-level i18n bundle used by Localize when a
// bundle parameter is not provided. Call SetBundle at application startup to
// configure it (for example after loading locale files).
var defaultBundle *i18n.Bundle

// SetBundle sets the package-level i18n bundle used by Localize.
func SetBundle(b *i18n.Bundle) error {
	if b == nil {
		return errors.New("bundle cannot be nil")
	}
	defaultBundle = b
	return nil
}

// GetBundle returns the currently-set package-level i18n bundle (may be nil).
func GetBundle() *i18n.Bundle {
	return defaultBundle
}

type Gerr interface {
	Code() string
	MessageKey() string
	Args() map[string]any
	Metadata() map[string]any
	Status() int
	Retryable() bool
	Unwrap() error
	Error() string
	UserMessage(lang string) (string, error)
}

// gerr is the unexported concrete implementation of the Error interface.
type gerr struct {
	code       string
	messageKey string
	args       map[string]any
	metadata   map[string]any
	status     int
	retryable  bool
	err        error
}

// ensure gerr implements built-in error interface
var _ error = (*gerr)(nil)

// newGerr creates a *gerr instance with defaults.
func newGerr(code, messageKey string, httpCode int, opts ...Option) *gerr {
	g := &gerr{
		code:       code,
		messageKey: messageKey,
		args:       make(map[string]any),
		metadata:   make(map[string]any),
		status:     httpCode,
		retryable:  isRetryableByStatus(httpCode),
		err:        nil,
	}

	// Apply options
	for _, opt := range opts {
		if opt != nil {
			opt(g)
		}
	}

	return g
}

// isRetryableByStatus returns true for 5xx HTTP status codes.
func isRetryableByStatus(status int) bool {
	return status >= 500 && status <= 599
}

// New constructs a new error (built-in error interface) representing a gerr.
// Parameters:
//   - code: machine-friendly code, e.g. "GENERIC-001" or "USER-001"
//   - messageKey: i18n message key, e.g. "error.user_not_found"
//   - httpCode: suggested HTTP status code
//   - opts: optional Option helpers (WithArgs, WithMeta, WithRetryable)
//
// The function returns the standard built-in error interface. Callers that
// need access to gerr-specific features (Localize, accessors) should use
// AsError / errors.As to obtain the exported Error interface.
func New(code, messageKey string, httpCode int, opts ...Option) error {
	return newGerr(code, messageKey, httpCode, opts...)
}

// Wrap constructs a new error that wraps an underlying err. Other parameters
// are the same as New.
func Wrap(err error, code, messageKey string, httpCode int, opts ...Option) error {
	g := newGerr(code, messageKey, httpCode, opts...)
	g.err = err
	return g
}

func (g *gerr) Code() string {
	if g == nil {
		return ""
	}
	return g.code
}

func (g *gerr) MessageKey() string {
	if g == nil {
		return ""
	}
	return g.messageKey
}

func (g *gerr) Args() map[string]any {
	if g == nil {
		return nil
	}
	return g.args
}

func (g *gerr) Metadata() map[string]any {
	if g == nil {
		return nil
	}
	return g.metadata
}

func (g *gerr) Status() int {
	if g == nil {
		return 0
	}
	return g.status
}

func (g *gerr) Retryable() bool {
	if g == nil {
		return false
	}
	return g.retryable
}

// Unwrap returns the wrapped error (may be nil).
func (g *gerr) Unwrap() error {
	if g == nil {
		return nil
	}
	return g.err
}

// UserMessage returns the localized error message for this Error using the package-level
// i18n bundle and language. The package bundle must be set with SetBundle
// (e.g. during application startup). If the bundle is not set, this method
// returns an error explaining that the bundle must be configured.
func (g *gerr) UserMessage(lang string) (string, error) {
	if g == nil {
		return "", fmt.Errorf("error is nil")
	}

	if g.messageKey == "" {
		return "", fmt.Errorf("message key is empty on error")
	}

	return localizeMessage(lang, g.messageKey, g.args)
}

// Error implements the standard error interface. It prints:
//
//	<Code> (<wrapped error>)
//
// and intentionally omits the MessageKey.
func (g *gerr) Error() string {
	if g == nil {
		return "<nil>"
	}
	if g.err != nil {
		return fmt.Sprintf("%s (%v)", g.code, g.err)
	}
	return g.code
}

// --------------------
// helpers for consumers
// --------------------

// AsError tries to extract our Error interface from an arbitrary error chain.
// It returns the Error interface and true on success.
func AsError(err error) (Gerr, bool) {
	if err == nil {
		return nil, false
	}
	// First try direct type assertion to our exported interface
	if e, ok := err.(*gerr); ok {
		return e, true
	}
	// Then try errors.As for the concrete *gerr type
	var ge *gerr
	if errors.As(err, &ge) {
		return ge, true
	}
	return nil, false
}

// GetCode extracts a machine-friendly code from any error chain, if present.
func GetCode(err error) (string, bool) {
	if err == nil {
		return "", false
	}
	if e, ok := AsError(err); ok {
		return e.Code(), e.Code() != ""
	}
	return "", false
}

// GetMessageKey extracts the MessageKey from any error chain, if present.
func GetMessageKey(err error) (string, bool) {
	if err == nil {
		return "", false
	}
	if e, ok := AsError(err); ok {
		return e.MessageKey(), e.MessageKey() != ""
	}
	return "", false
}

// GetUserMessage extracts the UserMessage from any error chain, if present.
func GetUserMessage(err error, lang string) (string, error) {
	if err == nil {
		return "", fmt.Errorf("error is nil")
	}
	if e, ok := AsError(err); ok {
		return e.UserMessage(lang)
	}
	return "", fmt.Errorf("error is not a gerr.Error")
}
