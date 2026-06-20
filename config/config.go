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
	EnvTrustedHeaderSubject      = "VEC_TRUSTED_HEADER_SUBJECT"
	LegacyTrustedHeaderSubject   = "BOLSA_TRUSTED_HEADER_SUBJECT"
	EnvTrustedHeaderRoles        = "VEC_TRUSTED_HEADER_ROLES"
	LegacyTrustedHeaderRoles     = "BOLSA_TRUSTED_HEADER_ROLES"
	EnvTrustedHeaderMechanism    = "VEC_TRUSTED_HEADER_MECHANISM"
	LegacyTrustedHeaderMechanism = "BOLSA_TRUSTED_HEADER_MECHANISM"
	EnvTrustedProxyCIDRs         = "VEC_TRUSTED_PROXY_CIDRS"
	LegacyEnvTrustedProxyCIDRs   = "BOLSA_TRUSTED_PROXY_CIDRS"

	StorageModeMemory       = "memory"
	StorageModeFile         = "file"
	StorageModeLocalDurable = "local_durable"

	AuthModeFake           = "fake"
	AuthModeTrustedHeaders = "trusted_headers"

	DefaultAddress                = ":8080"
	DefaultAPIBasePath            = "/api"
	DefaultReadHeaderLimit        = 5 * time.Second
	DefaultStorageMode            = StorageModeMemory
	DefaultDataDir                = "var/bolsa"
	DefaultDataFileName           = "bolsa_store.json"
	DefaultAuthMode               = AuthModeFake
	DefaultTrustedHeaderSubject   = "X-VEC-Subject"
	DefaultTrustedHeaderRoles     = "X-VEC-Roles"
	DefaultTrustedHeaderMechanism = "X-VEC-Auth-Mechanism"
)

type Config struct {
	Address                string
	APIBasePath            string
	ReadHeaderTimeout      time.Duration
	StorageMode            string
	DataDir                string
	DataPath               string
	AuthMode               string
	TrustedHeaderSubject   string
	TrustedHeaderRoles     string
	TrustedHeaderMechanism string
	TrustedProxyCIDRs      []string
}

func Load() Config {
	return Config{
		Address:                envFirst(EnvAddress, LegacyEnvAddress),
		APIBasePath:            DefaultAPIBasePath,
		ReadHeaderTimeout:      DefaultReadHeaderLimit,
		StorageMode:            envFirst(EnvStorageMode, LegacyEnvStorageMode),
		DataDir:                envFirst(EnvDataDir, LegacyEnvDataDir),
		DataPath:               envFirst(EnvDataPath, LegacyEnvDataPath),
		AuthMode:               envFirst(EnvAuthMode, LegacyEnvAuthMode),
		TrustedHeaderSubject:   envFirst(EnvTrustedHeaderSubject, LegacyTrustedHeaderSubject),
		TrustedHeaderRoles:     envFirst(EnvTrustedHeaderRoles, LegacyTrustedHeaderRoles),
		TrustedHeaderMechanism: envFirst(EnvTrustedHeaderMechanism, LegacyTrustedHeaderMechanism),
		TrustedProxyCIDRs:      splitCSV(envFirst(EnvTrustedProxyCIDRs, LegacyEnvTrustedProxyCIDRs)),
	}.Normalize()
}

func (c Config) Normalize() Config {
	if strings.TrimSpace(c.Address) == "" {
		c.Address = DefaultAddress
	}
	c.APIBasePath = normalizePath(c.APIBasePath)
	if c.ReadHeaderTimeout <= 0 {
		c.ReadHeaderTimeout = DefaultReadHeaderLimit
	}
	c.StorageMode = normalizeStorageMode(c.StorageMode)
	c.DataDir = defaultString(c.DataDir, DefaultDataDir)
	c.DataPath = defaultDataPath(c.DataPath, c.DataDir)
	c.AuthMode = normalizeAuthMode(c.AuthMode)
	c.TrustedHeaderSubject = defaultString(c.TrustedHeaderSubject, DefaultTrustedHeaderSubject)
	c.TrustedHeaderRoles = defaultString(c.TrustedHeaderRoles, DefaultTrustedHeaderRoles)
	c.TrustedHeaderMechanism = defaultString(c.TrustedHeaderMechanism, DefaultTrustedHeaderMechanism)
	c.TrustedProxyCIDRs = normalizeCIDRs(c.TrustedProxyCIDRs)
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
	case AuthModeTrustedHeaders:
		return AuthModeTrustedHeaders
	default:
		return DefaultAuthMode
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
