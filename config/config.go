package config

import (
	"os"
	"path/filepath"
	"strings"
	"time"
)

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

func Load() Config {
	return Config{
		Address:                envFirst(EnvAddress, LegacyEnvAddress),
		APIBasePath:            DefaultAPIBasePath,
		ReadHeaderTimeout:      DefaultReadHeaderLimit,
		ReadTimeout:            DefaultReadTimeout,
		WriteTimeout:           DefaultWriteTimeout,
		IdleTimeout:            DefaultIdleTimeout,
		MaxHeaderBytes:         DefaultMaxHeaderBytes,
		MaxRequestBodyBytes:    DefaultMaxRequestBodyBytes,
		StorageMode:            envFirst(EnvStorageMode, LegacyEnvStorageMode),
		DataDir:                envFirst(EnvDataDir, LegacyEnvDataDir),
		DataPath:               envFirst(EnvDataPath, LegacyEnvDataPath),
		AuthMode:               envFirst(EnvAuthMode, LegacyEnvAuthMode),
		FakeCredentialsPath:    envFirst(EnvFakeCredentialsPath),
		TrustedHeaderSubject:   envFirst(EnvTrustedHeaderSubject, LegacyTrustedHeaderSubject),
		TrustedHeaderRoles:     envFirst(EnvTrustedHeaderRoles, LegacyTrustedHeaderRoles),
		TrustedHeaderMechanism: envFirst(EnvTrustedHeaderMechanism, LegacyTrustedHeaderMechanism),
		TrustedProxyCIDRs:      splitCSV(envFirst(EnvTrustedProxyCIDRs, LegacyEnvTrustedProxyCIDRs)),
		HTTPAllowedCIDRs:       splitCSV(envFirst(EnvHTTPAllowedCIDRs)),
		TLSCertFile:            envFirst(EnvTLSCertFile),
		TLSKeyFile:             envFirst(EnvTLSKeyFile),
		PersonalCatalogPath:    envFirst(EnvPersonalCatalogPath),
		BolsaPublicSourcePath:  envFirst(EnvBolsaPublicSourcePath),
		OSRMBaseURL:            envFirst(EnvOSRMBaseURL),
		OSRMScopeName:          envFirst(EnvOSRMScopeName),
		OSRMScopeBounds:        envFirst(EnvOSRMScopeBounds),
		OSRMAllowedCIDRs:       splitCSV(envFirst(EnvOSRMAllowedCIDRs)),
	}.Normalize()
}

func (c Config) Normalize() Config {
	if strings.TrimSpace(c.Address) == "" {
		c.Address = DefaultAddress
	} else {
		c.Address = strings.TrimSpace(c.Address)
	}
	c.APIBasePath = normalizePath(c.APIBasePath)
	if c.ReadHeaderTimeout <= 0 {
		c.ReadHeaderTimeout = DefaultReadHeaderLimit
	}
	if c.ReadTimeout <= 0 {
		c.ReadTimeout = DefaultReadTimeout
	}
	if c.WriteTimeout <= 0 {
		c.WriteTimeout = DefaultWriteTimeout
	}
	if c.IdleTimeout <= 0 {
		c.IdleTimeout = DefaultIdleTimeout
	}
	if c.MaxHeaderBytes <= 0 {
		c.MaxHeaderBytes = DefaultMaxHeaderBytes
	}
	if c.MaxRequestBodyBytes <= 0 {
		c.MaxRequestBodyBytes = DefaultMaxRequestBodyBytes
	}
	c.StorageMode = normalizeStorageMode(c.StorageMode)
	c.DataDir = defaultString(c.DataDir, DefaultDataDir)
	c.DataPath = defaultDataPath(c.DataPath, c.DataDir)
	c.AuthMode = normalizeAuthMode(c.AuthMode)
	c.FakeCredentialsPath = strings.TrimSpace(c.FakeCredentialsPath)
	c.TrustedHeaderSubject = defaultString(c.TrustedHeaderSubject, DefaultTrustedHeaderSubject)
	c.TrustedHeaderRoles = defaultString(c.TrustedHeaderRoles, DefaultTrustedHeaderRoles)
	c.TrustedHeaderMechanism = defaultString(c.TrustedHeaderMechanism, DefaultTrustedHeaderMechanism)
	c.TrustedProxyCIDRs = normalizeCIDRs(c.TrustedProxyCIDRs)
	// El listener general arranca limitado a loopback. Exponerlo a otra red,
	// incluida Internet, exige enumerarla de forma expresa.
	c.HTTPAllowedCIDRs = normalizeCIDRs(c.HTTPAllowedCIDRs)
	c.TLSCertFile = strings.TrimSpace(c.TLSCertFile)
	c.TLSKeyFile = strings.TrimSpace(c.TLSKeyFile)
	c.PersonalCatalogPath = normalizeOptionalPath(c.PersonalCatalogPath, DefaultPersonalCatalogPath)
	c.BolsaPublicSourcePath = defaultString(c.BolsaPublicSourcePath, DefaultBolsaPublicSourcePath)
	c.OSRMBaseURL = strings.TrimRight(strings.TrimSpace(c.OSRMBaseURL), "/")
	c.OSRMScopeName = strings.TrimSpace(c.OSRMScopeName)
	c.OSRMScopeBounds = strings.TrimSpace(c.OSRMScopeBounds)
	// No se infiere ninguna red para el motor de rutas. Configurar la URL sin
	// enumerar tambien sus redes de destino deja la integracion incompleta y el
	// adaptador HTTP rechazara el arranque.
	c.OSRMAllowedCIDRs = normalizeOptionalCIDRs(c.OSRMAllowedCIDRs)
	return c
}

func envFirst(keys ...string) string {
	for _, key := range keys {
		if value := strings.TrimSpace(os.Getenv(key)); value != "" {
			return value
		}
	}
	return ""
}

func defaultDataPath(value, dataDir string) string {
	if trimmed := strings.TrimSpace(value); trimmed != "" {
		return trimmed
	}
	return filepath.Join(defaultString(dataDir, DefaultDataDir), DefaultDataFileName)
}

func normalizePath(path string) string {
	path = "/" + strings.Trim(strings.TrimSpace(path), "/")
	if path == "/" {
		return DefaultAPIBasePath
	}
	return path
}

func normalizeStorageMode(mode string) string {
	switch strings.ToLower(strings.TrimSpace(mode)) {
	case StorageModeFile, StorageModeLocalDurable:
		return StorageModeFile
	default:
		return DefaultStorageMode
	}
}

func normalizeAuthMode(mode string) string {
	switch strings.ToLower(strings.TrimSpace(mode)) {
	case AuthModeFake:
		return AuthModeFake
	case AuthModeTrustedHeaders:
		return AuthModeTrustedHeaders
	case AuthModeDisabled:
		return AuthModeDisabled
	default:
		return DefaultAuthMode
	}
}

func normalizeOptionalPath(path, fallback string) string {
	trimmed := strings.TrimSpace(path)
	switch strings.ToLower(trimmed) {
	case "memory", "mem", "none", "off", "-", ":memory:":
		return ""
	case "":
		return fallback
	default:
		return trimmed
	}
}

func defaultString(value, fallback string) string {
	if trimmed := strings.TrimSpace(value); trimmed != "" {
		return trimmed
	}
	return fallback
}

func normalizeCIDRs(values []string) []string {
	if len(values) == 0 {
		return []string{"127.0.0.1/32", "::1/128"}
	}
	normalized := make([]string, 0, len(values))
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			normalized = append(normalized, trimmed)
		}
	}
	if len(normalized) == 0 {
		return []string{"127.0.0.1/32", "::1/128"}
	}
	return normalized
}

func normalizeOptionalCIDRs(values []string) []string {
	normalized := make([]string, 0, len(values))
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			normalized = append(normalized, trimmed)
		}
	}
	return normalized
}

func splitCSV(value string) []string {
	parts := strings.Split(value, ",")
	values := make([]string, 0, len(parts))
	for _, part := range parts {
		if trimmed := strings.TrimSpace(part); trimmed != "" {
			values = append(values, trimmed)
		}
	}
	return values
}
