package i18n

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
)

const (
	DefaultLocale = "es"
	localesDir    = "locales"
)

var (
	ErrNoLocalesFound = errors.New("i18n: no locale files found")
	ErrLocaleRequired = errors.New("i18n: locale is required")
	ErrKeyRequired    = errors.New("i18n: key is required")
)

type Catalog struct {
	defaultLocale string
	messages      map[string]map[string]string
}

type Option func(*config)

type config struct {
	defaultLocale string
}

func WithDefaultLocale(locale string) Option {
	return func(cfg *config) {
		cfg.defaultLocale = normalize(locale)
	}
}

func Load(opts ...Option) (*Catalog, error) {
	return LoadFS(os.DirFS("."), localesDir, opts...)
}

func LoadDir(dir string, opts ...Option) (*Catalog, error) {
	clean := filepath.Clean(strings.TrimSpace(dir))
	if clean == "." || clean == "" {
		return LoadFS(os.DirFS("."), ".", opts...)
	}
	return LoadFS(os.DirFS(filepath.Dir(clean)), filepath.Base(clean), opts...)
}

func LoadFS(fsys fs.FS, dir string, opts ...Option) (*Catalog, error) {
	cfg := config{defaultLocale: DefaultLocale}
	for _, opt := range opts {
		if opt != nil {
			opt(&cfg)
		}
	}
	if cfg.defaultLocale == "" {
		return nil, ErrLocaleRequired
	}

	pattern := path.Join(strings.Trim(strings.TrimSpace(dir), "/"), "*.json")
	if pattern == "*.json" {
		pattern = "*.json"
	}
	files, err := fs.Glob(fsys, pattern)
	if err != nil {
		return nil, fmt.Errorf("i18n: glob locales: %w", err)
	}
	if len(files) == 0 {
		return nil, ErrNoLocalesFound
	}
	sort.Strings(files)

	catalog := &Catalog{
		defaultLocale: cfg.defaultLocale,
		messages:      make(map[string]map[string]string, len(files)),
	}
	for _, file := range files {
		locale := localeFromFile(file)
		if locale == "" {
			continue
		}
		messages, err := readLocale(fsys, file)
		if err != nil {
			return nil, err
		}
		catalog.messages[locale] = messages
	}
	if len(catalog.messages) == 0 {
		return nil, ErrNoLocalesFound
	}
	return catalog, nil
}

func New(defaultLocale string, messages map[string]map[string]string) (*Catalog, error) {
	defaultLocale = normalize(defaultLocale)
	if defaultLocale == "" {
		return nil, ErrLocaleRequired
	}
	catalog := &Catalog{
		defaultLocale: defaultLocale,
		messages:      make(map[string]map[string]string, len(messages)),
	}
	for locale, localeMessages := range messages {
		normalizedLocale := normalize(locale)
		if normalizedLocale == "" {
			continue
		}
		catalog.messages[normalizedLocale] = cloneMessages(localeMessages)
	}
	if len(catalog.messages) == 0 {
		return nil, ErrNoLocalesFound
	}
	return catalog, nil
}

func (c *Catalog) T(locale, key string) string {
	message, ok := c.Message(locale, key)
	if !ok {
		return strings.TrimSpace(key)
	}
	return message
}

func (c *Catalog) Translate(locale, key string) string {
	return c.T(locale, key)
}

func (c *Catalog) Message(locale, key string) (string, bool) {
	if c == nil {
		return "", false
	}
	key = strings.TrimSpace(key)
	if key == "" {
		return "", false
	}
	if message, ok := c.messageFor(normalize(locale), key); ok {
		return message, true
	}
	return c.messageFor(c.defaultLocale, key)
}

func (c *Catalog) Lookup(locale, key string) (string, bool) {
	return c.Message(locale, key)
}

func (c *Catalog) DefaultLocale() string {
	if c == nil {
		return ""
	}
	return c.defaultLocale
}

func (c *Catalog) Locales() []string {
	if c == nil {
		return nil
	}
	locales := make([]string, 0, len(c.messages))
	for locale := range c.messages {
		locales = append(locales, locale)
	}
	sort.Strings(locales)
	return locales
}

func (c *Catalog) messageFor(locale, key string) (string, bool) {
	messages, ok := c.messages[locale]
	if !ok {
		return "", false
	}
	message, ok := messages[key]
	return message, ok
}

func readLocale(fsys fs.FS, file string) (map[string]string, error) {
	data, err := fs.ReadFile(fsys, file)
	if err != nil {
		return nil, fmt.Errorf("i18n: read %s: %w", file, err)
	}
	var messages map[string]string
	if err := json.Unmarshal(data, &messages); err != nil {
		return nil, fmt.Errorf("i18n: parse %s: %w", file, err)
	}
	return cloneMessages(messages), nil
}

func cloneMessages(source map[string]string) map[string]string {
	clone := make(map[string]string, len(source))
	for key, message := range source {
		key = strings.TrimSpace(key)
		if key == "" {
			continue
		}
		clone[key] = message
	}
	return clone
}

func localeFromFile(file string) string {
	name := path.Base(file)
	ext := path.Ext(name)
	return normalize(strings.TrimSuffix(name, ext))
}

func normalize(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}
