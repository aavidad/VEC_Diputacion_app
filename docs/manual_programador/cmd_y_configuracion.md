# Arranque, composicion y configuracion

Parte del [Manual del programador](LEEME.md). Fichero generado con
`scripts/generar_manual_programador.py`; no editar a mano.

## Paquete `cmd/bolsa-server`

> Centinela retirado: falla cerrado y no arranca ningun servidor.

## Paquete `cmd/vec-emisor-capacidad-v4`

> Command vec-emisor-capacidad-v4 ejecuta exclusivamente el verificador COSE y emisor de capacidades V4.

Command vec-emisor-capacidad-v4 ejecuta exclusivamente el verificador COSE y
emisor de capacidades V4. No importa la fachada ejecutora y no debe recibir la
credencial PostgreSQL del portal.

## Paquete `cmd/vec-server`

> Composicion canonica y arranque del servidor HTTP del portal VEC.

## Paquete `config`

> Carga y validacion de la configuracion canonica por variables de entorno.

### Constantes

```go
const (
	EnvAddress                   = "VEC_HTTP_ADDR"
	LegacyEnvAddress             = "BOLSA_HTTP_ADDR"
	EnvStorageMode               = "VEC_BOLSA_STORAGE_MODE"
	LegacyEnvStorageMode         = "BOLSA_STORAGE_MODE"
	EnvDataDir                   = "VEC_BOLSA_DATA_DIR"
	LegacyEnvDataDir             = "BOLSA_DATA_DIR"
	EnvDataPath                  = "VEC_BOLSA_DATA_PATH"
	LegacyEnvDataPath            = "BOLSA_DATA_PATH"
	EnvAuthMode                  = "VEC_AUTH_MODE"
	LegacyEnvAuthMode            = "BOLSA_AUTH_MODE"
	EnvFakeCredentialsPath       = "VEC_FAKE_CREDENTIALS_FILE"
	EnvTrustedHeaderSubject      = "VEC_TRUSTED_HEADER_SUBJECT"
	LegacyTrustedHeaderSubject   = "BOLSA_TRUSTED_HEADER_SUBJECT"
	EnvTrustedHeaderRoles        = "VEC_TRUSTED_HEADER_ROLES"
	LegacyTrustedHeaderRoles     = "BOLSA_TRUSTED_HEADER_ROLES"
	EnvTrustedHeaderMechanism    = "VEC_TRUSTED_HEADER_MECHANISM"
	LegacyTrustedHeaderMechanism = "BOLSA_TRUSTED_HEADER_MECHANISM"
	EnvTrustedProxyCIDRs         = "VEC_TRUSTED_PROXY_CIDRS"
	LegacyEnvTrustedProxyCIDRs   = "BOLSA_TRUSTED_PROXY_CIDRS"
	EnvHTTPAllowedCIDRs          = "VEC_HTTP_ALLOWED_CIDRS"
	EnvTLSCertFile               = "VEC_TLS_CERT_FILE"
	EnvTLSKeyFile                = "VEC_TLS_KEY_FILE"
	EnvPersonalCatalogPath       = "VEC_PERSONAL_CATALOG_PATH"
	EnvBolsaPublicSourcePath     = "VEC_BOLSA_PUBLIC_SOURCE_PATH"
	EnvOSRMBaseURL               = "VEC_OSRM_BASE_URL"
	EnvOSRMScopeName             = "VEC_OSRM_SCOPE_NAME"
	EnvOSRMScopeBounds           = "VEC_OSRM_SCOPE_BOUNDS"
	EnvOSRMAllowedCIDRs          = "VEC_OSRM_ALLOWED_CIDRS"

	StorageModeMemory       = "memory"
	StorageModeFile         = "file"
	StorageModeLocalDurable = "local_durable"

	AuthModeDisabled       = "disabled"
	AuthModeFake           = "fake"
	AuthModeTrustedHeaders = "trusted_headers"

	DefaultAddress                = "127.0.0.1:8080"
	DefaultAPIBasePath            = "/api"
	DefaultReadHeaderLimit        = 5 * time.Second
	DefaultReadTimeout            = 30 * time.Second
	DefaultWriteTimeout           = 60 * time.Second
	DefaultIdleTimeout            = 2 * time.Minute
	DefaultMaxHeaderBytes         = 1 << 20
	DefaultMaxRequestBodyBytes    = int64(2 << 20)
	DefaultStorageMode            = StorageModeMemory
	DefaultDataDir                = "var/bolsa"
	DefaultDataFileName           = "bolsa_store.json"
	DefaultPersonalCatalogPath    = "var/vec/personal_catalog.json"
	DefaultBolsaPublicSourcePath  = "data/demo/convocatorias_publicas.demo.json"
	DefaultAuthMode               = AuthModeDisabled
	DefaultTrustedHeaderSubject   = "X-VEC-Subject"
	DefaultTrustedHeaderRoles     = "X-VEC-Roles"
	DefaultTrustedHeaderMechanism = "X-VEC-Auth-Mechanism"
)
```

### Tipos

```go
type Config struct {
	Address                string
	APIBasePath            string
	ReadHeaderTimeout      time.Duration
	ReadTimeout            time.Duration
	WriteTimeout           time.Duration
	IdleTimeout            time.Duration
	MaxHeaderBytes         int
	MaxRequestBodyBytes    int64
	StorageMode            string
	DataDir                string
	DataPath               string
	AuthMode               string
	FakeCredentialsPath    string
	TrustedHeaderSubject   string
	TrustedHeaderRoles     string
	TrustedHeaderMechanism string
	TrustedProxyCIDRs      []string
	HTTPAllowedCIDRs       []string
	TLSCertFile            string
	TLSKeyFile             string
	PersonalCatalogPath    string
	BolsaPublicSourcePath  string
	OSRMBaseURL            string
	OSRMScopeName          string
	OSRMScopeBounds        string
	OSRMAllowedCIDRs       []string
}

func Load() Config

func (c Config) Normalize() Config
```

## Paquete `internal/app/bootstrap`

> Composicion de la API y montaje de modulos para el arranque.

### Funciones

```go
func NewDemoAPI() (http.Handler, error)
func NewDemoAPIWithConfig(cfg config.Config) (http.Handler, error)
func NewHTTPServer() (*http.Server, error)
func NewHTTPServerWithConfig(cfg config.Config) (*http.Server, error)
func NewVECShellAPI() (http.Handler, error)
func NewVECShellAPIWithConfig(cfg config.Config) (http.Handler, error)
```

## Paquete `internal/app/server`

> Construccion del servidor HTTP con limites y tiempos canonicos.

### Funciones

```go
func NewHTTPServer(cfg config.Config, api http.Handler) (*http.Server, error)
func NewHandler(api http.Handler) http.Handler
func NewHandlerWithConfig(cfg config.Config, api http.Handler) http.Handler
```
