package gerr

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/nicksnyder/go-i18n/v2/i18n"
	"golang.org/x/text/language"
	"gopkg.in/yaml.v3"
)

// LoadBundleFromFiles creates an i18n.Bundle with the given defaultLang (e.g. "en")
// and loads all provided message files. Supported file formats: JSON and YAML.
// The filenames' extensions (.json, .yaml, .yml) are used to determine the format.
//
// Example:
//
//	bundle, err := LoadBundleFromFiles("en", "example/locales/en.json", "example/locales/id.json")
func LoadBundleFromFiles(defaultLang language.Tag, files ...string) (*i18n.Bundle, error) {
	if defaultLang.String() == "" {
		return nil, fmt.Errorf("defaultLang must be provided")
	}

	bundle := i18n.NewBundle(defaultLang)
	// register json and yaml unmarshalers
	bundle.RegisterUnmarshalFunc("json", json.Unmarshal)
	bundle.RegisterUnmarshalFunc("yaml", yaml.Unmarshal)
	bundle.RegisterUnmarshalFunc("yml", yaml.Unmarshal)

	for _, f := range files {
		if f == "" {
			continue
		}
		data, err := os.ReadFile(f)
		if err != nil {
			return nil, fmt.Errorf("read locale file %q: %w", f, err)
		}

		// ParseMessageFileBytes returns (*MessageFile, error). We only need the error.
		// Use filepath.Base to ensure the extension is visible for format detection.
		if _, err := bundle.ParseMessageFileBytes(data, filepath.Base(f)); err != nil {
			// Some callers may pass full paths; ParseMessageFileBytes relies on the
			// filename extension to choose the unmarshal func. Passing Base keeps it simple.
			// If parsing fails, include the file path in the error for easier debugging.
			return nil, fmt.Errorf("parse locale file %q: %w", f, err)
		}
	}

	return bundle, nil
}

func localizeMessage(lang string, key string, args map[string]any) (string, error) {
	if defaultBundle == nil {
		return "", fmt.Errorf("i18n bundle is not set; call SetBundle to configure the bundle")
	}
	if key == "" {
		return "", fmt.Errorf("message key is empty")
	}

	localizer := i18n.NewLocalizer(defaultBundle, lang)
	cfg := &i18n.LocalizeConfig{
		MessageID:    key,
		TemplateData: args,
	}

	// Auto-detect plural count from args if present (support "Count" and "count").
	// We don't enforce the numeric type strictly here; go-i18n accepts numeric types.
	if args != nil {
		if v, ok := args["Count"]; ok {
			cfg.PluralCount = v
		} else if v, ok := args["count"]; ok {
			cfg.PluralCount = v
		}
	}

	msg, err := localizer.Localize(cfg)
	if err != nil {
		return "", fmt.Errorf("localize key %q for lang %q: %w", key, lang, err)
	}
	return msg, nil
}
