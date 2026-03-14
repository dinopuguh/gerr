package gerr

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/nicksnyder/go-i18n/v2/i18n"
	"golang.org/x/text/language"
	"gopkg.in/yaml.v3"
)

// LoadBundleFromFiles creates an i18n.Bundle with the given defaultLang (e.g. "en")
// and loads all provided paths. Each path may be a file or a directory; directories
// are walked recursively and all .json, .yaml, and .yml files are loaded.
//
// Example:
//
//	bundle, err := LoadBundleFromFiles(language.English, "locales/")
//	bundle, err := LoadBundleFromFiles(language.English, "locales/en.json", "locales/id.json")
func LoadBundleFromFiles(defaultLang language.Tag, paths ...string) (*i18n.Bundle, error) {
	if defaultLang.String() == "" {
		return nil, fmt.Errorf("defaultLang must be provided")
	}

	bundle := i18n.NewBundle(defaultLang)
	bundle.RegisterUnmarshalFunc("json", json.Unmarshal)
	bundle.RegisterUnmarshalFunc("yaml", yaml.Unmarshal)
	bundle.RegisterUnmarshalFunc("yml", yaml.Unmarshal)

	for _, p := range paths {
		if p == "" {
			continue
		}
		if err := loadPath(bundle, p); err != nil {
			return nil, err
		}
	}

	return bundle, nil
}

func loadPath(bundle *i18n.Bundle, path string) error {
	info, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("stat locale path %q: %w", path, err)
	}

	if !info.IsDir() {
		return loadFile(bundle, path)
	}

	return filepath.WalkDir(path, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		ext := filepath.Ext(p)
		if ext != ".json" && ext != ".yaml" && ext != ".yml" {
			return nil
		}
		return loadFile(bundle, p)
	})
}

func loadFile(bundle *i18n.Bundle, path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read locale file %q: %w", path, err)
	}
	if _, err := bundle.ParseMessageFileBytes(data, filepath.Base(path)); err != nil {
		return fmt.Errorf("parse locale file %q: %w", path, err)
	}
	return nil
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
