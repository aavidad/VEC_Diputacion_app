package config

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

const (
	EnvAddress = "VEC_HTTP_ADDR"
	// EnvHTTPIdleTimeout alarga el tiempo que el servidor mantiene abiertas las
	// conexiones inactivas. Las consultas RRHH ligan su cursor al canal TLS, así
	// que un cierre por inactividad obliga a rehacer la consulta; un despliegue
	// detrás de un proxy con una conexión persistente puede necesitar más margen.
	EnvHTTPIdleTimeout                             = "VEC_HTTP_IDLE_TIMEOUT"
	LegacyEnvAddress                               = "BOLSA_HTTP_ADDR"
	EnvStorageMode                                 = "VEC_BOLSA_STORAGE_MODE"
	LegacyEnvStorageMode                           = "BOLSA_STORAGE_MODE"
	EnvDataDir                                     = "VEC_BOLSA_DATA_DIR"
	LegacyEnvDataDir                               = "BOLSA_DATA_DIR"
	EnvDataPath                                    = "VEC_BOLSA_DATA_PATH"
	LegacyEnvDataPath                              = "BOLSA_DATA_PATH"
	EnvAuthMode                                    = "VEC_AUTH_MODE"
	LegacyEnvAuthMode                              = "BOLSA_AUTH_MODE"
	EnvFakeCredentialsPath                         = "VEC_FAKE_CREDENTIALS_FILE"
	EnvTrustedHeaderSubject                        = "VEC_TRUSTED_HEADER_SUBJECT"
	LegacyTrustedHeaderSubject                     = "BOLSA_TRUSTED_HEADER_SUBJECT"
	EnvTrustedHeaderRoles                          = "VEC_TRUSTED_HEADER_ROLES"
	LegacyTrustedHeaderRoles                       = "BOLSA_TRUSTED_HEADER_ROLES"
	EnvTrustedHeaderMechanism                      = "VEC_TRUSTED_HEADER_MECHANISM"
	LegacyTrustedHeaderMechanism                   = "BOLSA_TRUSTED_HEADER_MECHANISM"
	EnvTrustedProxyCIDRs                           = "VEC_TRUSTED_PROXY_CIDRS"
	LegacyEnvTrustedProxyCIDRs                     = "BOLSA_TRUSTED_PROXY_CIDRS"
	EnvHTTPAllowedCIDRs                            = "VEC_HTTP_ALLOWED_CIDRS"
	EnvTLSCertFile                                 = "VEC_TLS_CERT_FILE"
	EnvTLSKeyFile                                  = "VEC_TLS_KEY_FILE"
	EnvCTNumeroExpedienteSourcePath                = "VEC_CT_NUMERO_EXPEDIENTE_SOURCE_PATH"
	EnvCTCircuitoRRHHSourcePath                    = "VEC_CT_CIRCUITO_RRHH_SOURCE_PATH"
	EnvCTNecesidadesAltaSourcePath                 = "VEC_CT_NECESIDADES_ALTA_SOURCE_PATH"
	EnvPersonalCatalogPath                         = "VEC_PERSONAL_CATALOG_PATH"
	EnvIncorporacionV2File                         = "VEC_CT_INCORPORACION_V2_FILE"
	EnvContratacionTemporalSubsanacionPoliticaFile = "VEC_CT_SUBSANACION_POLITICA_FILE"
	EnvPersonalOrganizacionSourcePath              = "VEC_PERSONAL_ORGANIZACION_SOURCE_PATH"
	EnvRPTCatalogoPath                             = "VEC_RPT_CATALOGO_PATH"
	EnvPersonalOrganizacionVersion                 = "VEC_PERSONAL_ORGANIZACION_VERSION"
	EnvPersonalOrganizacionPostgreSQL              = "VEC_PERSONAL_ORGANIZACION_POSTGRESQL"
	EnvBolsaPublicSourcePath                       = "VEC_BOLSA_PUBLIC_SOURCE_PATH"
	EnvBolsaCategoriesSourcePath                   = "VEC_BOLSA_CATEGORIES_SOURCE_PATH"
	EnvBolsaCategoriesCatalogID                    = "VEC_BOLSA_CATEGORIES_CATALOG_ID"
	EnvBolsaCategoriesVersion                      = "VEC_BOLSA_CATEGORIES_CATALOG_VERSION"
	EnvBolsaCategoriesSHA256                       = "VEC_BOLSA_CATEGORIES_CATALOG_SHA256"
	EnvBolsaCategoriesPublicProjectionSHA256       = "VEC_BOLSA_CATEGORIES_PUBLIC_PROJECTION_SHA256"
	EnvBolsaImportacionConvocaCustodiaDir          = "VEC_BOLSA_IMPORTACION_CONVOCA_CUSTODIA_DIR"
	EnvOSRMBaseURL                                 = "VEC_OSRM_BASE_URL"
	EnvOSRMScopeName                               = "VEC_OSRM_SCOPE_NAME"
	EnvOSRMScopeBounds                             = "VEC_OSRM_SCOPE_BOUNDS"
	EnvOSRMAllowedCIDRs                            = "VEC_OSRM_ALLOWED_CIDRS"
	EnvOSRMGraphVersion                            = "VEC_OSRM_GRAPH_VERSION"
	EnvRRHHPresentationEnabled                     = "VEC_RRHH_PRESENTATION_ENABLED"
	EnvRRHHPresentationGuardOne                    = "VEC_RRHH_PRESENTATION_GUARD_ONE"
	EnvRRHHPresentationGuardTwo                    = "VEC_RRHH_PRESENTATION_GUARD_TWO"
	EnvSMTPHost                                    = "VEC_SMTP_HOST"
	EnvSMTPPort                                    = "VEC_SMTP_PORT"
	EnvSMTPFrom                                    = "VEC_SMTP_FROM"
	EnvSMTPCAFile                                  = "VEC_SMTP_CA_FILE"
	EnvSMTPModoTLS                                 = "VEC_SMTP_MODO_TLS"
	EnvFirmaVerificacionEnabled                    = "VEC_FIRMA_VERIFICACION_ENABLED"
	EnvFirmaVerificacionURL                        = "VEC_FIRMA_VERIFICACION_URL"
	EnvFirmaVerificacionCAFile                     = "VEC_FIRMA_VERIFICACION_CA_FILE"
	EnvFirmaVerificacionTokenFile                  = "VEC_FIRMA_VERIFICACION_TOKEN_FILE"
	EnvFirmaVerificacionCertFile                   = "VEC_FIRMA_VERIFICACION_CERT_FILE"
	EnvFirmaVerificacionKeyFile                    = "VEC_FIRMA_VERIFICACION_KEY_FILE"
	EnvFirmaVerificacionTimeout                    = "VEC_FIRMA_VERIFICACION_TIMEOUT"
	EnvFirmaVerificacionNombreServidorTLS          = "VEC_FIRMA_VERIFICACION_TLS_SERVER_NAME"

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

const EnvCTAnalisisMotivosSourcePath = "VEC_CT_ANALISIS_RECTIFICACION_MOTIVOS_SOURCE_PATH"

type Config struct {
	Address                                     string
	APIBasePath                                 string
	ReadHeaderTimeout                           time.Duration
	ReadTimeout                                 time.Duration
	WriteTimeout                                time.Duration
	IdleTimeout                                 time.Duration
	MaxHeaderBytes                              int
	MaxRequestBodyBytes                         int64
	StorageMode                                 string
	DataDir                                     string
	DataPath                                    string
	AuthMode                                    string
	ExecutionProfile                            string
	DevelopmentGuard                            string
	DevelopmentMaterialDir                      string
	PortalProceso                               string
	ExternoPreflightV3DatabaseURL               string
	ExternoBolsaPublicaPostgreSQL               ConfiguracionPostgreSQLPublica
	ExternoBolsaPostgreSQL                      ConfiguracionPostgreSQLExterna
	BolsaInscripcionesLectorPostgreSQL          ConfiguracionPostgreSQLExterna
	BolsaInscripcionesEmpleadoLectorPostgreSQL  ConfiguracionPostgreSQLExterna
	BolsaInscripcionesRRHHLectorPostgreSQL      ConfiguracionPostgreSQLExterna
	ExternoCalendariosPostgreSQL                ConfiguracionPostgreSQLExterna
	ExternoBolsaFronteraPostgreSQL              ConfiguracionPostgreSQLExterna
	ExternoAutorizacionFuentePostgreSQL         ConfiguracionPostgreSQLExterna
	ExternoAutorizacionRegistroPostgreSQL       ConfiguracionPostgreSQLExterna
	ExternoAutorizacionMotivosPostgreSQL        ConfiguracionPostgreSQLExterna
	ExternoIdentidadRegistroPostgreSQL          ConfiguracionPostgreSQLExterna
	ExternoIdentidadRevalidacionPostgreSQL      ConfiguracionPostgreSQLExterna
	ExternoContextoPostgreSQL                   ConfiguracionPostgreSQLExterna
	IncorporacionV2File                         string
	ContratacionTemporalSubsanacionPoliticaFile string
	CTAnalisisMotivosSourcePath                 string
	ReglasEjemplo                               ConfiguracionReglasEjemplo
	FakeCredentialsPath                         string
	TrustedHeaderSubject                        string
	TrustedHeaderRoles                          string
	TrustedHeaderMechanism                      string
	TrustedProxyCIDRs                           []string
	HTTPAllowedCIDRs                            []string
	TLSCertFile                                 string
	TLSKeyFile                                  string
	CTNumeroExpedienteSourcePath                string
	CTCircuitoRRHHSourcePath                    string
	CTNecesidadesAltaSourcePath                 string
	PersonalCatalogPath                         string
	PersonalCatalogInMemory                     bool
	PersonalOrganizacionSourcePath              string
	RPTCatalogoPath                             string
	PersonalOrganizacionVersion                 int
	PersonalOrganizacionPostgreSQL              bool
	BolsaPublicSourcePath                       string
	BolsaCategoriesSourcePath                   string
	BolsaCategoriesCatalogID                    string
	BolsaCategoriesVersion                      int
	BolsaCategoriesSHA256                       string
	BolsaCategoriesPublicProjectionSHA256       string
	BolsaImportacionConvocaCustodiaDir          string
	BolsaAprobacionProvisionMiBolsa             string
	BolsaPreimagenProvisionMiBolsa              string
	BolsaPublicaPostgreSQL                      ConfiguracionPostgreSQLPublica
	BolsaPublicaManifiestoSHA256                string
	OSRMBaseURL                                 string
	OSRMScopeName                               string
	OSRMScopeBounds                             string
	OSRMAllowedCIDRs                            []string
	OSRMGraphVersion                            string
	RRHHPresentationEnabled                     bool
	RRHHPresentationGuardOne                    string
	RRHHPresentationGuardTwo                    string
	SMTPHost                                    string
	SMTPPort                                    int
	SMTPFrom                                    string
	SMTPCAFile                                  string
	SMTPModoTLS                                 string
	BolsaBorradoresPostgreSQL                   ConfiguracionPostgreSQLBorradores
	BolsaBorradoresEnabled                      bool
	BolsaContratosCT                            ConfiguracionEntregaContratosCTBolsa
	DietasBorradoresEnabled                     string
	CronosEmpleadoEnabled                       string
	CronosResolucionEnabled                     string
	CTFirmaRegistroEnabled                      string
	BolsaPortalCandidatoEnabled                 string
	BolsaInscripcionesEnabled                   string
	CTSeguimientoCeseEnabled                    string
	CTCancelacionEnabled                        string
	CTIncorporacionAcreditadaEnabled            string
	CTAprobacionPerfilesCentro                  string
	CTPreimagenesPerfilesCentro                 string
	CTAprobacionPerfilesRRHH                    string
	CTPreimagenesPerfilesRRHH                   string
	CronosNotificacionesEnabled                 string
	DocumentosEnabled                           string
	PortalModulosVisiblesLista                  string
	FirmaVerificacionEnabled                    string
	FirmaVerificacionURL                        string
	FirmaVerificacionCAFile                     string
	FirmaVerificacionTokenFile                  string
	FirmaVerificacionCertFile                   string
	FirmaVerificacionKeyFile                    string
	FirmaVerificacionTimeout                    string
	FirmaVerificacionNombreServidorTLS          string
	PersonalEmpleadoEnabled                     string
	PersonalB2GobiernoEnabled                   string
	OrganizacionHistoricaGobiernoEnabled        string
	DietasBorradoresPostgreSQL                  ConfiguracionDietasBorradores
	BolsaAuditoriaFronteraPostgreSQL            ConfiguracionPostgreSQLBolsaAuditoriaFrontera
	BolsaConstitucionPostgreSQL                 ConfiguracionPostgreSQLBolsaConstitucion
	BolsaRelevoNoIncorporacionPostgreSQL        ConfiguracionPostgreSQLBolsaRelevoNoIncorporacion
	BolsaRelevoCesePostgreSQL                   ConfiguracionPostgreSQLBolsaRelevoCese
	AuditoriaSelladoPostgreSQL                  ConfiguracionPostgreSQLAuditoriaSellado
	BolsaPoliticaOfertasCalculadorPostgreSQL    ConfiguracionPostgreSQLBolsaPoliticaOfertasCalculador
	BolsaImportacionConvocaPostgreSQL           ConfiguracionPostgreSQLImportacionConvoca
	ContratacionTemporalPostgreSQL              ConfiguracionPostgreSQLContratacionTemporal
	CalendariosPostgreSQL                       ConfiguracionCalendarios
}

func Load() Config {
	return Config{
		Address:                envFirst(EnvAddress, LegacyEnvAddress),
		APIBasePath:            DefaultAPIBasePath,
		ReadHeaderTimeout:      DefaultReadHeaderLimit,
		ReadTimeout:            DefaultReadTimeout,
		WriteTimeout:           DefaultWriteTimeout,
		IdleTimeout:            idleTimeoutDesdeEntorno(),
		MaxHeaderBytes:         DefaultMaxHeaderBytes,
		MaxRequestBodyBytes:    DefaultMaxRequestBodyBytes,
		StorageMode:            envFirst(EnvStorageMode, LegacyEnvStorageMode),
		DataDir:                envFirst(EnvDataDir, LegacyEnvDataDir),
		DataPath:               envFirst(EnvDataPath, LegacyEnvDataPath),
		AuthMode:               envFirst(EnvAuthMode, LegacyEnvAuthMode),
		ExecutionProfile:       envFirst(EnvExecutionProfile),
		DevelopmentGuard:       envFirst(EnvDevelopmentGuard),
		DevelopmentMaterialDir: envFirst(EnvDevelopmentMaterialDir),
		PortalProceso:          os.Getenv(EnvPortalProceso),
		IncorporacionV2File:    envFirst(EnvIncorporacionV2File),
		ContratacionTemporalSubsanacionPoliticaFile: envFirst(EnvContratacionTemporalSubsanacionPoliticaFile),
		CTAnalisisMotivosSourcePath:                 envFirst(EnvCTAnalisisMotivosSourcePath),
		ExternoPreflightV3DatabaseURL:               os.Getenv(EnvExternoPreflightV3DatabaseURL),
		ExternoBolsaPublicaPostgreSQL: ConfiguracionPostgreSQLPublica{
			dsn: os.Getenv(EnvExternoBolsaPublicaDatabaseURL),
		},
		ExternoBolsaPostgreSQL:                     ConfiguracionPostgreSQLExterna{dsn: os.Getenv(EnvExternoBolsaDatabaseURL)},
		BolsaInscripcionesLectorPostgreSQL:         ConfiguracionPostgreSQLExterna{dsn: os.Getenv(EnvBolsaInscripcionesLectorDatabaseURL)},
		BolsaInscripcionesEmpleadoLectorPostgreSQL: ConfiguracionPostgreSQLExterna{dsn: os.Getenv(EnvBolsaInscripcionesEmpleadoLectorDatabaseURL)},
		BolsaInscripcionesRRHHLectorPostgreSQL:     ConfiguracionPostgreSQLExterna{dsn: os.Getenv(EnvBolsaInscripcionesRRHHLectorDatabaseURL)},
		ExternoBolsaFronteraPostgreSQL:             ConfiguracionPostgreSQLExterna{dsn: os.Getenv(EnvExternoBolsaFronteraDatabaseURL)},
		ExternoCalendariosPostgreSQL:               ConfiguracionPostgreSQLExterna{dsn: os.Getenv(EnvExternoCalendariosDatabaseURL)},
		ExternoAutorizacionFuentePostgreSQL:        ConfiguracionPostgreSQLExterna{dsn: os.Getenv(EnvExternoAutorizacionFuenteDatabaseURL)},
		ExternoAutorizacionRegistroPostgreSQL:      ConfiguracionPostgreSQLExterna{dsn: os.Getenv(EnvExternoAutorizacionRegistroDatabaseURL)},
		ExternoAutorizacionMotivosPostgreSQL:       ConfiguracionPostgreSQLExterna{dsn: os.Getenv(EnvExternoAutorizacionMotivosDatabaseURL)},
		ExternoIdentidadRegistroPostgreSQL:         ConfiguracionPostgreSQLExterna{dsn: os.Getenv(EnvExternoIdentidadRegistroDatabaseURL)},
		ExternoIdentidadRevalidacionPostgreSQL:     ConfiguracionPostgreSQLExterna{dsn: os.Getenv(EnvExternoIdentidadRevalidacionDatabaseURL)},
		ExternoContextoPostgreSQL:                  ConfiguracionPostgreSQLExterna{dsn: os.Getenv(EnvExternoContextoDatabaseURL)},
		ReglasEjemplo:                              cargarConfiguracionReglasEjemplo(),
		FakeCredentialsPath:                        envFirst(EnvFakeCredentialsPath),
		TrustedHeaderSubject:                       envFirst(EnvTrustedHeaderSubject, LegacyTrustedHeaderSubject),
		TrustedHeaderRoles:                         envFirst(EnvTrustedHeaderRoles, LegacyTrustedHeaderRoles),
		TrustedHeaderMechanism:                     envFirst(EnvTrustedHeaderMechanism, LegacyTrustedHeaderMechanism),
		TrustedProxyCIDRs:                          splitCSV(envFirst(EnvTrustedProxyCIDRs, LegacyEnvTrustedProxyCIDRs)),
		HTTPAllowedCIDRs:                           splitCSV(envFirst(EnvHTTPAllowedCIDRs)),
		TLSCertFile:                                envFirst(EnvTLSCertFile),
		TLSKeyFile:                                 envFirst(EnvTLSKeyFile),
		CTNumeroExpedienteSourcePath:               envFirst(EnvCTNumeroExpedienteSourcePath),
		CTCircuitoRRHHSourcePath:                   envFirst(EnvCTCircuitoRRHHSourcePath),
		CTNecesidadesAltaSourcePath:                envFirst(EnvCTNecesidadesAltaSourcePath),
		PersonalCatalogPath:                        envFirst(EnvPersonalCatalogPath),
		PersonalOrganizacionSourcePath:             envFirst(EnvPersonalOrganizacionSourcePath),
		RPTCatalogoPath:                            envFirst(EnvRPTCatalogoPath),
		PersonalOrganizacionVersion:                envPositiveInt(EnvPersonalOrganizacionVersion),
		PersonalOrganizacionPostgreSQL:             envBool(EnvPersonalOrganizacionPostgreSQL),
		BolsaPublicSourcePath:                      envFirst(EnvBolsaPublicSourcePath),
		BolsaCategoriesSourcePath:                  envFirst(EnvBolsaCategoriesSourcePath),
		BolsaCategoriesCatalogID:                   envFirst(EnvBolsaCategoriesCatalogID),
		BolsaCategoriesVersion:                     envPositiveInt(EnvBolsaCategoriesVersion),
		BolsaCategoriesSHA256:                      envFirst(EnvBolsaCategoriesSHA256),
		BolsaCategoriesPublicProjectionSHA256:      envFirst(EnvBolsaCategoriesPublicProjectionSHA256),
		BolsaImportacionConvocaCustodiaDir:         envFirst(EnvBolsaImportacionConvocaCustodiaDir),
		BolsaAprobacionProvisionMiBolsa:            envFirst(EnvBolsaProvisionMiBolsaAprobacion),
		BolsaPreimagenProvisionMiBolsa:             envFirst(EnvBolsaProvisionMiBolsaPreimagen),
		BolsaPublicaPostgreSQL: ConfiguracionPostgreSQLPublica{
			dsn: envFirst(EnvBolsaPublicaDatabaseURL),
		},
		BolsaPublicaManifiestoSHA256:      envFirst(EnvBolsaPublicaManifiestoSHA256),
		OSRMBaseURL:                       envFirst(EnvOSRMBaseURL),
		OSRMScopeName:                     envFirst(EnvOSRMScopeName),
		OSRMScopeBounds:                   envFirst(EnvOSRMScopeBounds),
		OSRMAllowedCIDRs:                  splitCSV(envFirst(EnvOSRMAllowedCIDRs)),
		OSRMGraphVersion:                  envFirst(EnvOSRMGraphVersion),
		RRHHPresentationEnabled:           envBool(EnvRRHHPresentationEnabled),
		RRHHPresentationGuardOne:          envFirst(EnvRRHHPresentationGuardOne),
		RRHHPresentationGuardTwo:          envFirst(EnvRRHHPresentationGuardTwo),
		SMTPHost:                          envFirst(EnvSMTPHost),
		SMTPPort:                          envPositiveInt(EnvSMTPPort),
		SMTPFrom:                          envFirst(EnvSMTPFrom),
		SMTPCAFile:                        envFirst(EnvSMTPCAFile),
		SMTPModoTLS:                       envFirst(EnvSMTPModoTLS),
		BolsaImportacionConvocaPostgreSQL: ConfiguracionPostgreSQLImportacionConvoca{dsn: envFirst(EnvBolsaImportacionConvocaDatabaseURL)},
		BolsaBorradoresPostgreSQL: ConfiguracionPostgreSQLBorradores{
			dsnEjecutorConsulta:  envFirst(EnvBolsaBorradoresEjecutorConsultaDatabaseURL),
			dsnProyectorGobierno: envFirst(EnvBolsaBorradoresProyectorGobiernoDatabaseURL),
			dsnVerificadorRecibo: envFirst(EnvBolsaBorradoresVerificadorReciboDatabaseURL),
		},
		BolsaBorradoresEnabled:             envBool(EnvBolsaBorradoresEnabled),
		BolsaContratosCT:                   cargarEntregaContratosCTBolsa(),
		DietasBorradoresEnabled:            envFirst(EnvDietasBorradoresEnabled),
		CronosEmpleadoEnabled:              envFirst(EnvCronosEmpleadoEnabled),
		CronosResolucionEnabled:            envFirst(EnvCronosResolucionEnabled),
		CTFirmaRegistroEnabled:             envFirst(EnvCTFirmaRegistroEnabled),
		BolsaPortalCandidatoEnabled:        envFirst(EnvBolsaPortalCandidatoEnabled),
		BolsaInscripcionesEnabled:          envFirst(EnvBolsaInscripcionesEnabled),
		CTSeguimientoCeseEnabled:           envFirst(EnvCTSeguimientoCeseEnabled),
		CTCancelacionEnabled:               envFirst(EnvCTCancelacionEnabled),
		CTIncorporacionAcreditadaEnabled:   envFirst(EnvCTIncorporacionAcreditadaEnabled),
		CTAprobacionPerfilesCentro:         envFirst(EnvCTProvisionPerfilesCentroAprobacion),
		CTPreimagenesPerfilesCentro:        envFirst(EnvCTProvisionPerfilesCentroPreimagenes),
		CTAprobacionPerfilesRRHH:           envFirst(EnvCTProvisionPerfilesRRHHAprobacion),
		CTPreimagenesPerfilesRRHH:          envFirst(EnvCTProvisionPerfilesRRHHPreimagenes),
		CronosNotificacionesEnabled:        envFirst(EnvCronosNotificacionesEnabled),
		PersonalEmpleadoEnabled:            envFirst(EnvPersonalEmpleadoEnabled),
		PersonalB2GobiernoEnabled:          envFirst(EnvPersonalB2GobiernoEnabled),
		DocumentosEnabled:                  envFirst(EnvDocumentosEnabled),
		PortalModulosVisiblesLista:         envFirst(EnvPortalModulosVisibles),
		FirmaVerificacionEnabled:           envFirst(EnvFirmaVerificacionEnabled),
		FirmaVerificacionURL:               envFirst(EnvFirmaVerificacionURL),
		FirmaVerificacionCAFile:            envFirst(EnvFirmaVerificacionCAFile),
		FirmaVerificacionTokenFile:         envFirst(EnvFirmaVerificacionTokenFile),
		FirmaVerificacionCertFile:          envFirst(EnvFirmaVerificacionCertFile),
		FirmaVerificacionKeyFile:           envFirst(EnvFirmaVerificacionKeyFile),
		FirmaVerificacionTimeout:           envFirst(EnvFirmaVerificacionTimeout),
		FirmaVerificacionNombreServidorTLS: envFirst(EnvFirmaVerificacionNombreServidorTLS),
		DietasBorradoresPostgreSQL: ConfiguracionDietasBorradores{
			dsnDietas:             envFirst(EnvDietasBorradoresDatabaseURL),
			dsnPersonal:           envFirst(EnvDietasPersonalRelacionesDatabaseURL),
			dsnAsignacionPersonal: envFirst(EnvDietasPersonalAsignacionDatabaseURL),
			dsnAuditoriaPersonal:  envFirst(EnvDietasPersonalAuditoriaFronteraDatabaseURL),
		},
		CalendariosPostgreSQL: NuevaConfiguracionCalendarios(envFirst(EnvCalendariosDatabaseURL)),
		BolsaAuditoriaFronteraPostgreSQL: ConfiguracionPostgreSQLBolsaAuditoriaFrontera{
			dsn: envFirst(EnvBolsaAuditoriaFronteraDatabaseURL),
		},
		BolsaConstitucionPostgreSQL: ConfiguracionPostgreSQLBolsaConstitucion{
			dsn: envFirst(EnvBolsaConstitucionDatabaseURL),
		},
		BolsaRelevoNoIncorporacionPostgreSQL:     NuevaConfiguracionPostgreSQLBolsaRelevoNoIncorporacion(envFirst(EnvBolsaRelevoNoIncorporacionDatabaseURL)),
		BolsaRelevoCesePostgreSQL:                NuevaConfiguracionPostgreSQLBolsaRelevoCese(envFirst(EnvBolsaRelevoCeseDatabaseURL)),
		AuditoriaSelladoPostgreSQL:               NuevaConfiguracionPostgreSQLAuditoriaSellado(envFirst(EnvAuditoriaSelladoDatabaseURL)),
		BolsaPoliticaOfertasCalculadorPostgreSQL: NuevaConfiguracionPostgreSQLBolsaPoliticaOfertasCalculador(envFirst(EnvBolsaPoliticaOfertasCalculadorDatabaseURL)),
		ContratacionTemporalPostgreSQL: ConfiguracionPostgreSQLContratacionTemporal{
			dsnEjecucion: envFirst(EnvContratacionTemporalDatabaseURL),
			dsnGobierno:  envFirst(EnvContratacionTemporalGobiernoDatabaseURL),
			dsnRegistroAutorizacion: envFirst(
				EnvContratacionTemporalRegistroAutorizacionDatabaseURL,
			),
			dsnConfirmador:           envFirst(EnvContratacionTemporalConfirmadorDatabaseURL),
			dsnLectorResultado:       envFirst(EnvContratacionTemporalLectorResultadoDatabaseURL),
			dsnBolsaLlamamientos:     envFirst(EnvBolsaLlamamientosDatabaseURL),
			dsnConsultasRRHH:         envFirst(EnvContratacionTemporalConsultasRRHHDatabaseURL),
			dsnMotivosRRHH:           envFirst(EnvContratacionTemporalMotivosRRHHDatabaseURL),
			dsnRegistroIdentidad:     envFirst(EnvContratacionTemporalRegistroIdentidadDatabaseURL),
			dsnRevalidacionIdentidad: envFirst(EnvContratacionTemporalRevalidacionIdentidadDatabaseURL),
			dsnContextoActor:         envFirst(EnvContratacionTemporalContextoActorDatabaseURL),
			dsnAuditoriaFrontera:     envFirst(EnvContratacionTemporalAuditoriaFronteraDatabaseURL),
		},
		OrganizacionHistoricaGobiernoEnabled: envFirst(EnvOrganizacionHistoricaGobiernoEnabled),
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
	c.IncorporacionV2File = strings.TrimSpace(c.IncorporacionV2File)
	c.CTAnalisisMotivosSourcePath = strings.TrimSpace(c.CTAnalisisMotivosSourcePath)
	c.ReglasEjemplo = c.ReglasEjemplo.normalizar()
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
	c.PersonalOrganizacionSourcePath = strings.TrimSpace(c.PersonalOrganizacionSourcePath)
	c.RPTCatalogoPath = strings.TrimSpace(c.RPTCatalogoPath)
	c.CTNumeroExpedienteSourcePath = strings.TrimSpace(c.CTNumeroExpedienteSourcePath)
	c.CTCircuitoRRHHSourcePath = strings.TrimSpace(c.CTCircuitoRRHHSourcePath)
	c.CTNecesidadesAltaSourcePath = strings.TrimSpace(c.CTNecesidadesAltaSourcePath)
	if c.PersonalOrganizacionVersion == 0 {
		c.PersonalOrganizacionVersion = 1
	}
	c.BolsaCategoriesSourcePath = defaultString(c.BolsaCategoriesSourcePath, DefaultBolsaCategoriesSourcePath)
	c.BolsaCategoriesCatalogID = defaultString(c.BolsaCategoriesCatalogID, DefaultBolsaCategoriesCatalogID)
	if c.BolsaCategoriesVersion == 0 {
		c.BolsaCategoriesVersion = DefaultBolsaCategoriesVersion
	}
	c.BolsaCategoriesSHA256 = defaultString(c.BolsaCategoriesSHA256, DefaultBolsaCategoriesSHA256)
	c.BolsaImportacionConvocaCustodiaDir = strings.TrimSpace(c.BolsaImportacionConvocaCustodiaDir)
	c.BolsaAprobacionProvisionMiBolsa = strings.TrimSpace(c.BolsaAprobacionProvisionMiBolsa)
	c.BolsaPreimagenProvisionMiBolsa = strings.TrimSpace(c.BolsaPreimagenProvisionMiBolsa)
	c.OSRMBaseURL = strings.TrimRight(strings.TrimSpace(c.OSRMBaseURL), "/")
	c.OSRMScopeName = strings.TrimSpace(c.OSRMScopeName)
	c.OSRMScopeBounds = strings.TrimSpace(c.OSRMScopeBounds)
	c.OSRMGraphVersion = strings.TrimSpace(c.OSRMGraphVersion)
	// No se infiere ninguna red para el motor de rutas. Configurar la URL sin
	// enumerar tambien sus redes de destino deja la integracion incompleta y el
	// adaptador HTTP rechazara el arranque.
	c.OSRMAllowedCIDRs = normalizeOptionalCIDRs(c.OSRMAllowedCIDRs)
	c.RRHHPresentationGuardOne = strings.TrimSpace(c.RRHHPresentationGuardOne)
	c.RRHHPresentationGuardTwo = strings.TrimSpace(c.RRHHPresentationGuardTwo)
	c.BolsaBorradoresPostgreSQL = c.BolsaBorradoresPostgreSQL.normalizar()
	c.DietasBorradoresEnabled = strings.TrimSpace(c.DietasBorradoresEnabled)
	c.CronosEmpleadoEnabled = strings.TrimSpace(c.CronosEmpleadoEnabled)
	c.CronosResolucionEnabled = strings.TrimSpace(c.CronosResolucionEnabled)
	c.CTFirmaRegistroEnabled = strings.TrimSpace(c.CTFirmaRegistroEnabled)
	c.BolsaPortalCandidatoEnabled = strings.TrimSpace(c.BolsaPortalCandidatoEnabled)
	c.BolsaInscripcionesEnabled = strings.TrimSpace(c.BolsaInscripcionesEnabled)
	c.CTSeguimientoCeseEnabled = strings.TrimSpace(c.CTSeguimientoCeseEnabled)
	c.CTCancelacionEnabled = strings.TrimSpace(c.CTCancelacionEnabled)
	c.CTIncorporacionAcreditadaEnabled = strings.TrimSpace(c.CTIncorporacionAcreditadaEnabled)
	c.CTAprobacionPerfilesCentro = strings.TrimSpace(c.CTAprobacionPerfilesCentro)
	c.CTPreimagenesPerfilesCentro = strings.TrimSpace(c.CTPreimagenesPerfilesCentro)
	c.CTAprobacionPerfilesRRHH = strings.TrimSpace(c.CTAprobacionPerfilesRRHH)
	c.CTPreimagenesPerfilesRRHH = strings.TrimSpace(c.CTPreimagenesPerfilesRRHH)
	c.CronosNotificacionesEnabled = strings.TrimSpace(c.CronosNotificacionesEnabled)
	c.DocumentosEnabled = strings.TrimSpace(c.DocumentosEnabled)
	c.PortalModulosVisiblesLista = strings.TrimSpace(c.PortalModulosVisiblesLista)
	c.FirmaVerificacionEnabled = strings.TrimSpace(c.FirmaVerificacionEnabled)
	c.FirmaVerificacionURL = strings.TrimSpace(c.FirmaVerificacionURL)
	c.FirmaVerificacionCAFile = strings.TrimSpace(c.FirmaVerificacionCAFile)
	c.FirmaVerificacionTokenFile = strings.TrimSpace(c.FirmaVerificacionTokenFile)
	c.FirmaVerificacionCertFile = strings.TrimSpace(c.FirmaVerificacionCertFile)
	c.FirmaVerificacionKeyFile = strings.TrimSpace(c.FirmaVerificacionKeyFile)
	c.FirmaVerificacionTimeout = strings.TrimSpace(c.FirmaVerificacionTimeout)
	c.FirmaVerificacionNombreServidorTLS = strings.TrimSpace(c.FirmaVerificacionNombreServidorTLS)
	c.PersonalEmpleadoEnabled = strings.TrimSpace(c.PersonalEmpleadoEnabled)
	c.PersonalB2GobiernoEnabled = strings.TrimSpace(c.PersonalB2GobiernoEnabled)
	c.OrganizacionHistoricaGobiernoEnabled = strings.TrimSpace(c.OrganizacionHistoricaGobiernoEnabled)
	c.DietasBorradoresPostgreSQL = c.DietasBorradoresPostgreSQL.normalizar()
	c.BolsaAuditoriaFronteraPostgreSQL = c.BolsaAuditoriaFronteraPostgreSQL.normalizar()
	c.BolsaConstitucionPostgreSQL = c.BolsaConstitucionPostgreSQL.normalizar()
	c.BolsaRelevoNoIncorporacionPostgreSQL = c.BolsaRelevoNoIncorporacionPostgreSQL.normalizar()
	c.BolsaRelevoCesePostgreSQL = c.BolsaRelevoCesePostgreSQL.normalizar()
	c.AuditoriaSelladoPostgreSQL = c.AuditoriaSelladoPostgreSQL.normalizar()
	c.BolsaPoliticaOfertasCalculadorPostgreSQL = c.BolsaPoliticaOfertasCalculadorPostgreSQL.normalizar()
	c.BolsaPublicaPostgreSQL = c.BolsaPublicaPostgreSQL.normalizar()
	c.ExternoBolsaPublicaPostgreSQL = c.ExternoBolsaPublicaPostgreSQL.normalizar()
	c.ExternoBolsaPostgreSQL = c.ExternoBolsaPostgreSQL.normalizar()
	c.BolsaInscripcionesLectorPostgreSQL = c.BolsaInscripcionesLectorPostgreSQL.normalizar()
	c.BolsaInscripcionesEmpleadoLectorPostgreSQL = c.BolsaInscripcionesEmpleadoLectorPostgreSQL.normalizar()
	c.BolsaInscripcionesRRHHLectorPostgreSQL = c.BolsaInscripcionesRRHHLectorPostgreSQL.normalizar()
	c.ExternoBolsaFronteraPostgreSQL = c.ExternoBolsaFronteraPostgreSQL.normalizar()
	c.ExternoCalendariosPostgreSQL = c.ExternoCalendariosPostgreSQL.normalizar()
	c.ExternoAutorizacionFuentePostgreSQL = c.ExternoAutorizacionFuentePostgreSQL.normalizar()
	c.ExternoAutorizacionRegistroPostgreSQL = c.ExternoAutorizacionRegistroPostgreSQL.normalizar()
	c.ExternoAutorizacionMotivosPostgreSQL = c.ExternoAutorizacionMotivosPostgreSQL.normalizar()
	c.ExternoIdentidadRegistroPostgreSQL = c.ExternoIdentidadRegistroPostgreSQL.normalizar()
	c.ExternoIdentidadRevalidacionPostgreSQL = c.ExternoIdentidadRevalidacionPostgreSQL.normalizar()
	c.ExternoContextoPostgreSQL = c.ExternoContextoPostgreSQL.normalizar()
	c.ContratacionTemporalPostgreSQL = c.ContratacionTemporalPostgreSQL.normalizar()
	c.BolsaPublicaManifiestoSHA256 = strings.TrimSpace(c.BolsaPublicaManifiestoSHA256)
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

// idleTimeoutDesdeEntorno acepta VEC_HTTP_IDLE_TIMEOUT como duración de Go entre
// 30 segundos y 2 horas. Un valor ausente, mal formado o fuera de rango conserva
// el valor por defecto: la configuración nunca amplía el margen sin límite.
func idleTimeoutDesdeEntorno() time.Duration {
	valor := strings.TrimSpace(os.Getenv(EnvHTTPIdleTimeout))
	if valor == "" {
		return DefaultIdleTimeout
	}
	duracion, err := time.ParseDuration(valor)
	if err != nil || duracion < 30*time.Second || duracion > 2*time.Hour {
		return DefaultIdleTimeout
	}
	return duracion
}
