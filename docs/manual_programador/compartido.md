# Paquetes compartidos

Parte del [Manual del programador](LEEME.md). Fichero generado con
`scripts/generar_manual_programador.py`; no editar a mano.

## Paquete `internal/shared/i18n`

> Catalogo de internacionalizacion compartido con fallback espanol.

### Constantes

```go
const (
	DefaultLocale = "es"
)
```

### Variables

```go
var (
	ErrNoLocalesFound = errors.New("i18n: no locale files found")
	ErrLocaleRequired = errors.New("i18n: locale is required")
	ErrKeyRequired    = errors.New("i18n: key is required")
)
```

### Tipos

```go
type Catalog struct {
	// Has unexported fields.
}

func Load(opts ...Option) (*Catalog, error)

func LoadDir(dir string, opts ...Option) (*Catalog, error)

func LoadFS(fsys fs.FS, dir string, opts ...Option) (*Catalog, error)

func New(defaultLocale string, messages map[string]map[string]string) (*Catalog, error)

func (c *Catalog) DefaultLocale() string

func (c *Catalog) Locales() []string

func (c *Catalog) Lookup(locale, key string) (string, bool)

func (c *Catalog) Message(locale, key string) (string, bool)

func (c *Catalog) T(locale, key string) string

func (c *Catalog) Translate(locale, key string) string

type Option func(*config)

func WithDefaultLocale(locale string) Option
```
