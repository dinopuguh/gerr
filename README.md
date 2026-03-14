# gerr

`gerr` is a small Go library that provides a structured error wrapper with:
- machine-readable error `Code`
- i18n-aware `MessageKey` and template `Args`
- arbitrary `Meta` and `Details`
- HTTP `Status` and `Retryable` flag
- seamless wrapping of underlying `error` values (supports `errors.Is` / `errors.As`)
- helpers to convert `validator/v10` validation errors (uses the first validation error)
- zerolog integration via `MarshalZerologObject` for structured logging

Module path: `github.com/dinopuguh/gerr`

Installation
------------
**Library** — add `gerr` to your project:

```bash
go get github.com/dinopuguh/gerr@latest
go mod tidy
```

**CLI (`gerr gen`)** — install the code generator:

```bash
go install github.com/dinopuguh/gerr/cmd/gerr@latest
```

Verify the install:

```bash
gerr gen -help
```

Usage (quick start)
-------------------
1) Load locales and set the package bundle at application startup:

```go
bundle, err := gerr.LoadBundleFromFiles("en", "example/locales/en.json", "example/locales/id.json", "example/locales/ar.json")
if err != nil {
    // handle error
}
gerr.SetBundle(bundle)
```

2) Create an error and obtain a localized user message:

```go
rawErr := gerr.New("USER-001", "error.user_not_found", 404,
    gerr.WithArgs(map[string]interface{}{"username": "alice"}),
)
ge, _ := gerr.AsError(rawErr)
userMsg, _ := ge.UserMessage("en")
fmt.Println(userMsg)
```

3) Wrap a lower-level error and log it with zerolog:

```go
raw := gerr.Wrap(fmt.Errorf("db: fail"), "GENERIC-001", "error.unavailable", 503)
log.Error().Err(raw).Msg("operation failed")
```

Overview
--------
`gerr` helps you:
- Create consistent, machine-friendly error objects for your services.
- Keep user-facing messages in i18n resource files (owned by your application).
- Convert validator errors into an error that can be localized.
- Attach HTTP-related info (status, retryable) to errors so HTTP handlers can map errors to responses.
- Log errors as structured objects (zerolog).

Design highlights
-----------------
- `Code` (string): machine-readable code like `GENERIC-001` or `USER-001`.
- `MessageKey` (string): i18n key used to fetch a localized user message (e.g. `error.user_not_found`).
- `Args` (map[string]any): arguments for message templates (e.g. `{"username":"alice"}`).
- `Details` ([]string): developer details or extra human info.
- `Meta` (map[string]any): arbitrary metadata (e.g., field error details).
- `Status` (int): HTTP status code; defaulting behavior (e.g. validation -> 400).
- `Retryable` (bool): indicates whether the client should retry (default true for 5xx).
- `Err` (error): wrapped underlying error; `Unwrap()` is supported.

Key runtime decisions
- Constructors (`New` / `Wrap`) return the built-in `error` interface. Use `gerr.AsError(err)` to obtain the richer API (accessors, localization, fluent methods).
- The package exposes Option helpers to configure `Args`, `Details`, `Meta`, and `Retryable` at construction time.
- A package-level i18n bundle is used by default for `UserMessage` / `Localize` calls; set it once with `gerr.SetBundle(...)` at startup. You can still localize with a specific bundle using `LocalizeMessage(...)`.

i18n
----
- Integration: `github.com/nicksnyder/go-i18n/v2/i18n` + `golang.org/x/text/language`.
- Best practice: the *client application* owns the localization message files for all locales. The library provides `LoadBundleFromFiles` to help load them.
- Example locale files are under `example/locales/` to demonstrate expected keys and pluralization; replace or extend them in your application.

Where to put your locale files
------------------------------
Place your message files anywhere your app prefers. The example uses `example/locales/`. Example file content (shows plural forms):

```gerr/example/locales/en.json#L1-80
{
  "validation.required": {
    "id": "validation.required",
    "other": "The field '{{.Field}}' is required"
  },
  "error.user_not_found": {
    "id": "error.user_not_found",
    "other": "User '{{.username}}' not found"
  },
  "items.count": {
    "id": "items.count",
    "one": "You have one item",
    "other": "You have {{.Count}} items"
  }
}
```

Pluralization and Count detection
-------------------------------
When you localize messages that require plural forms (e.g. `items.count`), provide a numeric `Count` in the `Args` when constructing the error. The library helper `LocalizeMessage` automatically detects `Args["Count"]` (or `args["count"]`) and passes it as `PluralCount` to go-i18n so the correct plural form is selected.

Usage examples
--------------

Notes before examples
- `New` and `Wrap` return `error`. Use `gerr.AsError(err)` to get the richer API back (it will return a `*gerr` instance even for unknown errors — `AsError` wraps unknown errors into a `UNKNOWN-000` gerr so callers can always inspect code/status/meta).
- Set the package i18n bundle at startup: `gerr.SetBundle(bundle)`.
- If you prefer to localize using a specific bundle for a one-off, use `LocalizeMessage(bundle, lang, key, args)`.

1) Basic creation and localization
- Create an error with HTTP status and options. Then use `gerr.AsError` to access details and user message.

```gerr/example/main.go#L1-40
package main

import (
  "fmt"
  "net/http"
  "github.com/dinopuguh/gerr"
)

func main() {
  bundle, _ := gerr.LoadBundleFromFiles("en", "example/locales/en.json", "example/locales/id.json")
  gerr.SetBundle(bundle)

  // New returns an error (built-in). Use WithArgs via options.
  rawErr := gerr.New("USER-001", "error.user_not_found", http.StatusNotFound,
    gerr.WithArgs(map[string]interface{}{"username": "alice"}),
  )

  // Obtain the richer gerr object
  ge, ok := gerr.AsError(rawErr)
  if ok {
    userMsg, _ := ge.UserMessage("en") // localized string using package bundle
    fmt.Println("EN:", userMsg)
  }
}
```

2) Wrapping an underlying error and logging
- Wrap a lower-level error and log it as a structured object with zerolog. `*gerr` implements `MarshalZerologObject`.

```gerr/example/main.go#L40-90
package main

import (
  "errors"
  "net/http"
  "github.com/dinopuguh/gerr"
  "github.com/rs/zerolog/log"
)

func main() {
  dbErr := errors.New("sql: no rows in result set")
  rawWrapped := gerr.Wrap(dbErr, "GENERIC-001", "error.unavailable", http.StatusServiceUnavailable,
    gerr.WithArgs(map[string]interface{}{"Resource": "payment"}),
    gerr.WithRetryable(true),
  )

  wrapped, _ := gerr.AsError(rawWrapped)
  userMsg, _ := wrapped.UserMessage("en")
  log.Error().
    Str("operation", "query_db").
    Str("user_message", userMsg).
    Object("error", wrapped). // triggers MarshalZerologObject
    Msg("database query failed")
}
```

3) Validator error conversion (first error)
- Convert `validator/v10` errors to `gerr` (first FieldError) and localize.

```gerr/example/main.go#L90-130
package main

import (
  "fmt"
  "github.com/dinopuguh/gerr"
  "github.com/go-playground/validator/v10"
)

func main() {
  validate := validator.New()
  p := struct{ Email string `validate:"required,email"` }{Email: "bad"}
  if err := validate.Struct(p); err != nil {
    vErr := gerr.FromValidatorErr(err) // returns error
    if ge, ok := gerr.AsError(vErr); ok {
      msg, _ := ge.UserMessage("en") // ge.MessageKey = "validation.<tag>"
      fmt.Println("Validation message:", msg)
    }
  }
}
```

4) Multiple machine codes mapping to the same message key
- The library separates `Code` (machine) and `MessageKey` (user message). The mapping is controlled by your application.

```gerr/example/main.go#L130-170
raw1 := gerr.New("GENERIC-002", "error.unavailable", 503, gerr.WithArgs(map[string]interface{}{"Resource":"service"}))
raw2 := gerr.New("SERVICE-003", "error.unavailable", 503, gerr.WithArgs(map[string]interface{}{"Resource":"auth"}))

e1, _ := gerr.AsError(raw1)
e2, _ := gerr.AsError(raw2)

fmt.Println("Code:", e1.Code(), "Message:", e1.UserMessage("en"))
fmt.Println("Code:", e2.Code(), "Message:", e2.UserMessage("en"))
```

Zerolog integration (structured logs)
-------------------------------------
- `*gerr` implements `zerolog.LogObjectMarshaler` via `MarshalZerologObject`.
- You can include the error as an object in the event: `log.Error().Object("error", ge).Msg("...")`.
- The marshaler emits: `code`, `message_key`, `status`, `retryable`, `cause`, `args`, `meta`, and `details`.

API summary (current)
---------------------
- Constructors (return built-in `error`):
  - `func New(code, messageKey string, httpCode int, opts ...Option) error`
  - `func Wrap(err error, code, messageKey string, httpCode int, opts ...Option) error`

- Options:
  - `type Option func(*gerr)`
  - `func WithArgs(args map[string]any) Option`
  - `func WithDetails(details ...string) Option`
  - `func WithMeta(key string, value any) Option`
  - `func WithRetryable(retryable bool) Option`

- Accessing gerr features:
  - `func AsError(err error) (*gerr, bool)` — always returns a `*gerr` for non-nil input (unknown errors are wrapped into `UNKNOWN-000`).
  - `func GetCode(err error) (string, bool)`
  - `func GetMessageKey(err error) (string, bool)`

- `*gerr` accessors:
  - `func (g *gerr) Code() string`
  - `func (g *gerr) MessageKey() string`
  - `func (g *gerr) Args() map[string]any`
  - `func (g *gerr) Meta() map[string]any`
  - `func (g *gerr) Status() int`
  - `func (g *gerr) Retryable() bool`
  - `func (g *gerr) Unwrap() error`
  - `func (g *gerr) Error() string` // `<Code> (<wrapped error>)`
  - `func (g *gerr) UserMessage(lang string) (string, error)` // uses package bundle via `SetBundle`

- i18n helpers:
  - `func LoadBundleFromFiles(defaultLang string, files ...string) (*i18n.Bundle, error)`
  - `func SetBundle(b *i18n.Bundle)`
  - `func GetBundle() *i18n.Bundle`
  - `func LocalizeMessage(bundle *i18n.Bundle, lang string, key string, args map[string]any) (string, error)`
  - `func RegisterMessages(bundle *i18n.Bundle, lang string, messages map[string]string) error`
  - `func RegisterValidatorTagMessages(bundle *i18n.Bundle, lang string, tagMessages map[string]string) error`

- Validator conversion:
  - `func FromValidatorErr(err error) error` — converts validator errors into a gerr (first field error); returns the built-in `error`.

Best practices
--------------
- Keep user-facing messages under your application's control in locale files.
- Use `Code` for programmatic checks (monitoring, metrics, branching).
- Use `MessageKey` + `Args` for user-facing messages and translations.
- Prefer whole-sentence message templates for translators (avoid composing tokens across languages).
- Set the package bundle at startup with `SetBundle` and call `UserMessage` to get localized strings.
- For logging, include the structured `*gerr` object in zerolog events to capture rich context.

Code generation (`gerr gen`)
----------------------------
`gerr` ships a CLI tool that generates type-safe error constructors and locale JSON files from a YAML schema.

**Usage:**

```bash
gerr gen [-schema <file.yaml>|-schema-url <url>] [-out <dir>] [-locales <dir>] [-lang <lang>]
gerr gen -schema - < errors.yaml   # read from stdin
```

| Flag | Default | Description |
|------|---------|-------------|
| `-schema` | — | Path to YAML schema file, or `-` to read from stdin |
| `-schema-url` | — | URL to fetch YAML schema from (e.g. raw GitHub URL) |
| `-out` | `.` | Output directory for generated source files |
| `-locales` | `locales` | Output directory for locale JSON files |
| `-lang` | `go` | Target language (`go`) |

One of `-schema` or `-schema-url` is required.

**Schema format (`example/gen/errors.yaml`):**

```yaml
package: apierrors

validators:
  required:
    en: "The field '{{.Field}}' is required"
    id: "Kolom '{{.Field}}' wajib diisi"
  email:
    en: "The field '{{.Field}}' must be a valid email address"
    id: "Kolom '{{.Field}}' harus berupa alamat email yang valid"
  min:
    en: "The field '{{.Field}}' must be at least {{.Param}} characters"
    id: "Kolom '{{.Field}}' minimal {{.Param}} karakter"

errors:
  - name: UserNotFound
    code: USER-001
    key: error.user_not_found
    http: 404
    args: [username]
    messages:
      en: "User '{{.username}}' not found"
      id: "Pengguna '{{.username}}' tidak ditemukan"

  - name: Unauthorized
    code: AUTH-001
    key: error.unauthorized
    http: 401
    messages:
      en: "You are not authorized to perform this action"
      id: "Anda tidak memiliki izin untuk melakukan tindakan ini"
```

The `validators` section is optional. When present, `gerr gen` merges `validation.<tag>` keys into the locale files automatically — no manual locale files needed for validator errors.

**Run the generator:**

```bash
gerr gen -schema example/gen/errors.yaml -out example/gen -locales example/gen/locales
```

**Generated Go output (`example/gen/errors.go`):**

```go
// Code generated by gerr gen from errors.yaml. DO NOT EDIT.
package apierrors

import (
    "net/http"
    "github.com/dinopuguh/gerr"
)

const (
    CodeUserNotFound = "USER-001"
    CodeUnauthorized = "AUTH-001"
)

func UserNotFound(username string, opts ...gerr.Option) error {
    return gerr.New(CodeUserNotFound, "error.user_not_found", http.StatusNotFound,
        append([]gerr.Option{gerr.WithArgs(map[string]any{"username": username})}, opts...)...,
    )
}

func Unauthorized(opts ...gerr.Option) error {
    return gerr.New(CodeUnauthorized, "error.unauthorized", http.StatusUnauthorized, opts...)
}
```

The generator also writes or merges locale JSON files (one per language tag found in `messages`) into the `-locales` directory. Existing keys are preserved; schema keys are added or overwritten.

**Adding a new language backend:**

Implement `spec.Generator` in `cmd/gerr/lang/` and register it in `cmd/gerr/main.go`:

```go
// cmd/gerr/lang/typescript.go
package lang

type TsGenerator struct{}

func (g *TsGenerator) Lang() string { return "typescript" }
func (g *TsGenerator) Generate(schema spec.Schema, opts spec.GenOptions) error {
    // emit .ts files
}
```

```go
// cmd/gerr/main.go
var generators = map[string]spec.Generator{
    "go":         &lang.GoGenerator{},
    "typescript": &lang.TsGenerator{},
}
```

Then run: `gerr gen -schema errors.yaml -lang typescript`

Contributing and license
------------------------
This repository is intentionally minimal and intended to be adapted. If you want tests, additional helpers (e.g., a client-owned registry), or changes to defaults (e.g., unknown-code name or HTTP status mapping), open an issue or PR or ask for edits.

Module
------
Module path: `github.com/dinopuguh/gerr`
