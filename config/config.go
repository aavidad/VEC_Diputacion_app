package config

import (
	"os"
	"path/filepath"
	"strconv"
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
	EnvBolsaCategoriesSourcePath = "VEC_BOLSA_CATEGORIES_SOURCE_PATH"
	EnvBolsaCategoriesCatalogID  = "VEC_BOLSA_CATEGORIES_CATALOG_ID"
	EnvBolsaCategoriesVersion    = "VEC_BOLSA_CATEGORIES_CATALOG_VERSION"
	EnvBolsaCategoriesSHA256     = "VEC_BOLSA_CATEGORIES_CATALOG_SHA256"
	EnvOSRMBaseURL               = "VEC_OSRM_BASE_URL"
	EnvOSRMScopeName             = "VEC_OSRM_SCOPE_NAME"
	EnvOSRMScopeBounds           = "VEC_OSRM_SCOPE_BOUNDS"
	EnvOSRMAllowedCIDRs          = "VEC_OSRM_ALLOWED_CIDRS"
	EnvRRHHPresentationEnabled   = "VEC_RRHH_PRESENTATION_ENABLED"
	EnvRRHHPresentationGuardOne  = "VEC_RRHH_PRESENTATION_GUARD_ONE"
	EnvRRHHPresentationGuardTwo  = "VEC_RRHH_PRESENTATION_GUARD_TWO"

	StorageModeMemory       = "memory"
	StorageModeFile         = "file"
	StorageModeLocalDurable = "local_durable"

	AuthModeDisabled       = "disabled"
	AuthModeFake           = "fake"
	AuthModeTrustedHeaders = "trusted_headers"

	DefaultAddress                   = "127.0.0.1:8080"
	DefaultAPIBasePath               = "/api"
	DefaultReadHeaderLimit           = 5 * time.Second
	DefaultReadTimeout               = 30 * time.Second
	DefaultWriteTimeout              = 60 * time.Second
	DefaultIdleTimeout               = 2 * time.Minute
	DefaultMaxHeaderBytes            = 1 << 20
	DefaultMaxRequestBodyBytes       = int64(2 << 20)
	DefaultStorageMode               = StorageModeMemory
	DefaultDataDir                   = "var/bolsa"
	DefaultDataFileName              = "bolsa_store.json"
	DefaultPersonalCatalogPath       = "var/vec/personal_catalog.json"
	DefaultBolsaPublicSourcePath     = "data/demo/convocatorias_publicas.demo.json"
	DefaultBolsaCategoriesSourcePath = "data/catalogos/categorias-profesionales/v1.demo.json"
	DefaultBolsaCategoriesCatalogID  = "categorias-profesionales"
	DefaultBolsaCategoriesVersion    = 1
	DefaultBolsaCategoriesSHA256     = "b800a7e9c306fa8027709cfb4304cc8ccf8065f888673da71bd73a138c519233"
	DefaultAuthMode                  = AuthModeDisabled
	DefaultTrustedHeaderSubject      = "X-VEC-Subject"
	DefaultTrustedHeaderRoles        = "X-VEC-Roles"
	DefaultTrustedHeaderMechanism    = "X-VEC-Auth-Mechanism"
)

type Config struct {
	Address                   string
	APIBasePath               string
	ReadHeaderTimeout         time.Duration
	ReadTimeout               time.Duration
	WriteTimeout              time.Duration
	IdleTimeout               time.Duration
	MaxHeaderBytes            int
	MaxRequestBodyBytes       int64
	StorageMode               string
	DataDir                   string
	DataPath                  string
	AuthMode                  string
	ExecutionProfile          string
	DevelopmentGuard          string
	DevelopmentMaterialDir    string
	FakeCredentialsPath       string
	TrustedHeaderSubject      string
	TrustedHeaderRoles        string
	TrustedHeaderMechanism    string
	TrustedProxyCIDRs         []string
	HTTPAllowedCIDRs          []string
	TLSCertFile               string
	TLSKeyFile                string
	PersonalCatalogPath       string
	PersonalCatalogInMemory   bool
	BolsaPublicSourcePath     string
	BolsaCategoriesSourcePath string
	BolsaCategoriesCatalogID  string
	BolsaCategoriesVersion    int
	BolsaCategoriesSHA256     string
	OSRMBaseURL               string
	OSRMScopeName             string
	OSRMScopeBounds           string
	OSRMAllowedCIDRs          []string
	RRHHPresentationEnabled   bool
	RRHHPresentationGuardOne  string
	RRHHPresentationGuardTwo  string
}

func Load() Config {
	return Config{
		Address:                   envFirst(EnvAddress, LegacyEnvAddress),
		APIBasePath:               DefaultAPIBasePath,
		ReadHeaderTimeout:         DefaultReadHeaderLimit,
		ReadTimeout:               DefaultReadTimeout,
		WriteTimeout:              DefaultWriteTimeout,
		IdleTimeout:               DefaultIdleTimeout,
		MaxHeaderBytes:            DefaultMaxHeaderBytes,
		MaxRequestBodyBytes:       DefaultMaxRequestBodyBytes,
		StorageMode:               envFirst(EnvStorageMode, LegacyEnvStorageMode),
		DataDir:                   envFirst(EnvDataDir, LegacyEnvDataDir),
		DataPath:                  envFirst(EnvDataPath, LegacyEnvDataPath),
		AuthMode:                  envFirst(EnvAuthMode, LegacyEnvAuthMode),
		ExecutionProfile:          envFirst(EnvExecutionProfile),
		DevelopmentGuard:          envFirst(EnvDevelopmentGuard),
		DevelopmentMaterialDir:    envFirst(EnvDevelopmentMaterialDir),
		FakeCredentialsPath:       envFirst(EnvFakeCredentialsPath),
		TrustedHeaderSubject:      envFirst(EnvTrustedHeaderSubject, LegacyTrustedHeaderSubject),
		TrustedHeaderRoles:        envFirst(EnvTrustedHeaderRoles, LegacyTrustedHeaderRoles),
		TrustedHeaderMechanism:    envFirst(EnvTrustedHeaderMechanism, LegacyTrustedHeaderMechanism),
		TrustedProxyCIDRs:         splitCSV(envFirst(EnvTrustedProxyCIDRs, LegacyEnvTrustedProxyCIDRs)),
		HTTPAllowedCIDRs:          splitCSV(envFirst(EnvHTTPAllowedCIDRs)),
		TLSCertFile:               envFirst(EnvTLSCertFile),
		TLSKeyFile:                envFirst(EnvTLSKeyFile),
		PersonalCatalogPath:       envFirst(EnvPersonalCatalogPath),
		BolsaPublicSourcePath:     envFirst(EnvBolsaPublicSourcePath),
		BolsaCategoriesSourcePath: envFirst(EnvBolsaCategoriesSourcePath),
		BolsaCategoriesCatalogID:  envFirst(EnvBolsaCategoriesCatalogID),
		BolsaCategoriesVersion:    envPositiveInt(EnvBolsaCategoriesVersion),
		BolsaCategoriesSHA256:     envFirst(EnvBolsaCategoriesSHA256),
		OSRMBaseURL:               envFirst(EnvOSRMBaseURL),
		OSRMScopeName:             envFirst(EnvOSRMScopeName),
		OSRMScopeBounds:           envFirst(EnvOSRMScopeBounds),
		OSRMAllowedCIDRs:          splitCSV(envFirst(EnvOSRMAllowedCIDRs)),
		RRHHPresentationEnabled:   envBool(EnvRRHHPresentationEnabled),
		RRHHPresentationGuardOne:  envFirst(EnvRRHHPresentationGuardOne),
		RRHHPresentationGuardTwo:  envFirst(EnvRRHHPresentationGuardTwo),
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
	c.ExecutionProfile = normalizeExecutionProfile(c.ExecutionProfile)
	c.DevelopmentGuard = strings.TrimSpace(c.DevelopmentGuard)
	c.DevelopmentMaterialDir = strings.TrimSpace(c.DevelopmentMaterialDir)
	if c.ExecutionProfile == ExecutionProfileDevelopment && c.AuthMode == AuthModeDevelopment &&
		c.DevelopmentGuard == DevelopmentGuardAcknowledgement && c.DevelopmentMaterialDir != "" {
		rutas := c.DevelopmentPaths()
		if strings.TrimSpace(c.TLSCertFile) == "" {
			c.TLSCertFile = rutas.ServerCertificate
		}
		if strings.TrimSpace(c.TLSKeyFile) == "" {
			c.TLSKeyFile = rutas.ServerPrivateKey
		}
	}
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
	if c.PersonalCatalogInMemory || isMemoryPath(c.PersonalCatalogPath) {
		// El booleano conserva la decisión a través de normalizaciones sucesivas.
		// Un string vacío por sí solo significa «aplicar el valor por defecto» y
		// no puede representar de forma idempotente el modo en memoria.
		c.PersonalCatalogInMemory = true
		c.PersonalCatalogPath = ""
	} else {
		c.PersonalCatalogPath = normalizeOptionalPath(c.PersonalCatalogPath, DefaultPersonalCatalogPath)
	}
	c.BolsaPublicSourcePath = defaultString(c.BolsaPublicSourcePath, DefaultBolsaPublicSourcePath)
	c.BolsaCategoriesSourcePath = defaultString(c.BolsaCategoriesSourcePath, DefaultBolsaCategoriesSourcePath)
	c.BolsaCategoriesCatalogID = defaultString(c.BolsaCategoriesCatalogID, DefaultBolsaCategoriesCatalogID)
	if c.BolsaCategoriesVersion == 0 {
		c.BolsaCategoriesVersion = DefaultBolsaCategoriesVersion
	}
	c.BolsaCategoriesSHA256 = defaultString(c.BolsaCategoriesSHA256, DefaultBolsaCategoriesSHA256)
	c.OSRMBaseURL = strings.TrimRight(strings.TrimSpace(c.OSRMBaseURL), "/")
	c.OSRMScopeName = strings.TrimSpace(c.OSRMScopeName)
	c.OSRMScopeBounds = strings.TrimSpace(c.OSRMScopeBounds)
	// No se infiere ninguna red para el motor de rutas. Configurar la URL sin
	// enumerar tambien sus redes de destino deja la integracion incompleta y el
	// adaptador HTTP rechazara el arranque.
	c.OSRMAllowedCIDRs = normalizeOptionalCIDRs(c.OSRMAllowedCIDRs)
	c.RRHHPresentationGuardOne = strings.TrimSpace(c.RRHHPresentationGuardOne)
	c.RRHHPresentationGuardTwo = strings.TrimSpace(c.RRHHPresentationGuardTwo)
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

// envPositiveInt distingue una variable ausente (cero, aplica el valor por
// defecto) de una configuracion invalida (negativo, el bootstrap falla
// cerrado). No corrige silenciosamente una version de catalogo mal escrita.
func envPositiveInt(key string) int {
	valor := strings.TrimSpace(os.Getenv(key))
	if valor == "" {
		return 0
	}
	numero, err := strconv.Atoi(valor)
	if err != nil || numero < 1 {
		return -1
	}
	return numero
}

// envBool solo concede la activacion para valores positivos conocidos. Un
// valor ausente, ambiguo o mal escrito conserva la superficie deshabilitada.
func envBool(key string) bool {
	switch strings.ToLower(strings.TrimSpace(os.Getenv(key))) {
	case "1", "true", "yes", "on":
		return true
	default:
		return false
	}
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
	normalizado := strings.ToLower(strings.TrimSpace(mode))
	switch normalizado {
	case "", StorageModeMemory:
		return DefaultStorageMode
	case StorageModeFile, StorageModeLocalDurable:
		return StorageModeFile
	default:
		// Igual que perfil y autenticacion: conservar lo desconocido para que
		// la raiz de composicion falle, en vez de degradarlo a memoria.
		return normalizado
	}
}

func normalizeAuthMode(mode string) string {
	normalizado := strings.ToLower(strings.TrimSpace(mode))
	switch normalizado {
	case "":
		return DefaultAuthMode
	case AuthModeFake:
		return AuthModeFake
	case AuthModeTrustedHeaders:
		return AuthModeTrustedHeaders
	case AuthModeDevelopment:
		return AuthModeDevelopment
	case AuthModeDisabled:
		return AuthModeDisabled
	default:
		// Conservar el valor permite que la raiz de composicion lo rechace de
		// forma explicita. Convertir un error tipografico en "disabled" oculta
		// una configuracion invalida y puede cambiar la frontera desplegada.
		return normalizado
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

func isMemoryPath(path string) bool {
	switch strings.ToLower(strings.TrimSpace(path)) {
	case "memory", "mem", "none", "off", "-", ":memory:":
		return true
	default:
		return false
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
