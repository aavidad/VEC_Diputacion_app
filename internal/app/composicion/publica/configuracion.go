// Package publica contiene la raiz de composicion exclusiva de la superficie
// publica anonima. No debe importar dominios internos ni cargar sus secretos.
package publica

import (
	"os"
	"strconv"
	"strings"
	"time"

	"vec-diputacion-granada/config"
)

// Configuracion es la proyeccion minima que puede conocer el proceso publico.
// Los selectores no publicos se conservan solo para detectarlos y fallar de
// forma cerrada; nunca activan autenticacion ni proveedores internos.
type Configuracion struct {
	Direccion               string
	TiempoCabeceras         time.Duration
	TiempoLectura           time.Duration
	TiempoEscritura         time.Duration
	TiempoInactividad       time.Duration
	MaximoBytesCabeceras    int
	MaximoBytesPeticion     int64
	RedesPermitidas         []string
	CertificadoTLS          string
	ClaveTLS                string
	PerfilEjecucion         string
	AutenticacionSolicitada string
	GuardaDesarrollo        string
	MaterialDesarrollo      string

	FuenteConvocatorias        string
	FuenteCategorias           string
	CatalogoCategorias         string
	VersionCategorias          int
	HuellaCategorias           string
	HuellaProyeccionCategorias string
	HuellaManifiesto           string
	PostgreSQL                 config.ConfiguracionPostgreSQLPublica

	PresentacionHabilitada bool
	GuardaPresentacionUno  string
	GuardaPresentacionDos  string
}

// CargarConfiguracion lee exclusivamente opciones de transporte, fuentes
// publicas y selectores necesarios para cerrar configuraciones inseguras. No
// consulta DSN, KMS, credenciales, almacenamiento interno ni cabeceras de
// identidad.
func CargarConfiguracion() Configuracion {
	postgresql, _ := config.NuevaConfiguracionPostgreSQLPublica(
		primerValorEntorno(config.EnvBolsaPublicaDatabaseURL),
	)
	return Configuracion{
		Direccion:                  primerValorEntorno(config.EnvAddress, config.LegacyEnvAddress),
		TiempoCabeceras:            config.DefaultReadHeaderLimit,
		TiempoLectura:              config.DefaultReadTimeout,
		TiempoEscritura:            config.DefaultWriteTimeout,
		TiempoInactividad:          config.DefaultIdleTimeout,
		MaximoBytesCabeceras:       config.DefaultMaxHeaderBytes,
		MaximoBytesPeticion:        config.DefaultMaxRequestBodyBytes,
		RedesPermitidas:            separarLista(primerValorEntorno(config.EnvHTTPAllowedCIDRs)),
		CertificadoTLS:             primerValorEntorno(config.EnvTLSCertFile),
		ClaveTLS:                   primerValorEntorno(config.EnvTLSKeyFile),
		PerfilEjecucion:            primerValorEntorno(config.EnvExecutionProfile),
		AutenticacionSolicitada:    primerValorEntorno(config.EnvAuthMode, config.LegacyEnvAuthMode),
		GuardaDesarrollo:           primerValorEntorno(config.EnvDevelopmentGuard),
		MaterialDesarrollo:         primerValorEntorno(config.EnvDevelopmentMaterialDir),
		FuenteConvocatorias:        primerValorEntorno(config.EnvBolsaPublicSourcePath),
		FuenteCategorias:           primerValorEntorno(config.EnvBolsaCategoriesSourcePath),
		CatalogoCategorias:         primerValorEntorno(config.EnvBolsaCategoriesCatalogID),
		VersionCategorias:          enteroPositivoEntorno(config.EnvBolsaCategoriesVersion),
		HuellaCategorias:           primerValorEntorno(config.EnvBolsaCategoriesSHA256),
		HuellaProyeccionCategorias: primerValorEntorno(config.EnvBolsaCategoriesPublicProjectionSHA256),
		HuellaManifiesto:           primerValorEntorno(config.EnvBolsaPublicaManifiestoSHA256),
		PostgreSQL:                 postgresql,
		PresentacionHabilitada:     booleanoEntorno(config.EnvRRHHPresentationEnabled),
		GuardaPresentacionUno:      primerValorEntorno(config.EnvRRHHPresentationGuardOne),
		GuardaPresentacionDos:      primerValorEntorno(config.EnvRRHHPresentationGuardTwo),
	}.normalizar()
}

// DesdeConfiguracionGeneral adapta llamadas heredadas sin introducir el
// bootstrap monolitico en el grafo del binario publico.
func DesdeConfiguracionGeneral(cfg config.Config) Configuracion {
	fuenteConvocatorias := strings.TrimSpace(cfg.BolsaPublicSourcePath)
	fuenteCategorias := strings.TrimSpace(cfg.BolsaCategoriesSourcePath)
	catalogoCategorias := strings.TrimSpace(cfg.BolsaCategoriesCatalogID)
	versionCategorias := cfg.BolsaCategoriesVersion
	huellaCategorias := strings.TrimSpace(cfg.BolsaCategoriesSHA256)
	huellaProyeccionCategorias := strings.TrimSpace(cfg.BolsaCategoriesPublicProjectionSHA256)
	huellaManifiesto := strings.TrimSpace(cfg.BolsaPublicaManifiestoSHA256)
	cfg = cfg.NormalizePublicTransport()
	return Configuracion{
		Direccion:                  cfg.Address,
		TiempoCabeceras:            cfg.ReadHeaderTimeout,
		TiempoLectura:              cfg.ReadTimeout,
		TiempoEscritura:            cfg.WriteTimeout,
		TiempoInactividad:          cfg.IdleTimeout,
		MaximoBytesCabeceras:       cfg.MaxHeaderBytes,
		MaximoBytesPeticion:        cfg.MaxRequestBodyBytes,
		RedesPermitidas:            append([]string(nil), cfg.HTTPAllowedCIDRs...),
		CertificadoTLS:             cfg.TLSCertFile,
		ClaveTLS:                   cfg.TLSKeyFile,
		PerfilEjecucion:            cfg.ExecutionProfile,
		AutenticacionSolicitada:    cfg.AuthMode,
		GuardaDesarrollo:           cfg.DevelopmentGuard,
		MaterialDesarrollo:         cfg.DevelopmentMaterialDir,
		FuenteConvocatorias:        fuenteConvocatorias,
		FuenteCategorias:           fuenteCategorias,
		CatalogoCategorias:         catalogoCategorias,
		VersionCategorias:          versionCategorias,
		HuellaCategorias:           huellaCategorias,
		HuellaProyeccionCategorias: huellaProyeccionCategorias,
		HuellaManifiesto:           huellaManifiesto,
		PostgreSQL:                 cfg.BolsaPublicaPostgreSQL,
		PresentacionHabilitada:     cfg.RRHHPresentationEnabled,
		GuardaPresentacionUno:      cfg.RRHHPresentationGuardOne,
		GuardaPresentacionDos:      cfg.RRHHPresentationGuardTwo,
	}
}

func (cfg Configuracion) normalizar() Configuracion {
	fuenteConvocatorias := strings.TrimSpace(cfg.FuenteConvocatorias)
	fuenteCategorias := strings.TrimSpace(cfg.FuenteCategorias)
	catalogoCategorias := strings.TrimSpace(cfg.CatalogoCategorias)
	versionCategorias := cfg.VersionCategorias
	huellaCategorias := strings.TrimSpace(cfg.HuellaCategorias)
	huellaProyeccionCategorias := strings.TrimSpace(cfg.HuellaProyeccionCategorias)
	huellaManifiesto := strings.TrimSpace(cfg.HuellaManifiesto)
	general := config.Config{
		Address:                  cfg.Direccion,
		ReadHeaderTimeout:        cfg.TiempoCabeceras,
		ReadTimeout:              cfg.TiempoLectura,
		WriteTimeout:             cfg.TiempoEscritura,
		IdleTimeout:              cfg.TiempoInactividad,
		MaxHeaderBytes:           cfg.MaximoBytesCabeceras,
		MaxRequestBodyBytes:      cfg.MaximoBytesPeticion,
		HTTPAllowedCIDRs:         append([]string(nil), cfg.RedesPermitidas...),
		TLSCertFile:              cfg.CertificadoTLS,
		TLSKeyFile:               cfg.ClaveTLS,
		ExecutionProfile:         cfg.PerfilEjecucion,
		AuthMode:                 cfg.AutenticacionSolicitada,
		DevelopmentGuard:         cfg.GuardaDesarrollo,
		DevelopmentMaterialDir:   cfg.MaterialDesarrollo,
		BolsaPublicaPostgreSQL:   cfg.PostgreSQL,
		RRHHPresentationEnabled:  cfg.PresentacionHabilitada,
		RRHHPresentationGuardOne: cfg.GuardaPresentacionUno,
		RRHHPresentationGuardTwo: cfg.GuardaPresentacionDos,
	}.NormalizePublicTransport()
	resultado := Configuracion{
		Direccion: general.Address, TiempoCabeceras: general.ReadHeaderTimeout,
		TiempoLectura: general.ReadTimeout, TiempoEscritura: general.WriteTimeout,
		TiempoInactividad: general.IdleTimeout, MaximoBytesCabeceras: general.MaxHeaderBytes,
		MaximoBytesPeticion: general.MaxRequestBodyBytes,
		RedesPermitidas:     append([]string(nil), general.HTTPAllowedCIDRs...),
		CertificadoTLS:      general.TLSCertFile, ClaveTLS: general.TLSKeyFile,
		PerfilEjecucion: general.ExecutionProfile, AutenticacionSolicitada: general.AuthMode,
		GuardaDesarrollo:   strings.TrimSpace(cfg.GuardaDesarrollo),
		MaterialDesarrollo: strings.TrimSpace(cfg.MaterialDesarrollo),
		PostgreSQL:         cfg.PostgreSQL, PresentacionHabilitada: cfg.PresentacionHabilitada,
		GuardaPresentacionUno: strings.TrimSpace(cfg.GuardaPresentacionUno),
		GuardaPresentacionDos: strings.TrimSpace(cfg.GuardaPresentacionDos),
	}
	resultado.FuenteConvocatorias = fuenteConvocatorias
	resultado.FuenteCategorias = fuenteCategorias
	resultado.CatalogoCategorias = catalogoCategorias
	resultado.VersionCategorias = versionCategorias
	resultado.HuellaCategorias = huellaCategorias
	resultado.HuellaProyeccionCategorias = huellaProyeccionCategorias
	resultado.HuellaManifiesto = huellaManifiesto
	return resultado
}

func primerValorEntorno(claves ...string) string {
	for _, clave := range claves {
		if valor := strings.TrimSpace(os.Getenv(clave)); valor != "" {
			return valor
		}
	}
	return ""
}

func separarLista(valor string) []string {
	var resultado []string
	for _, elemento := range strings.Split(valor, ",") {
		if elemento = strings.TrimSpace(elemento); elemento != "" {
			resultado = append(resultado, elemento)
		}
	}
	return resultado
}

func enteroPositivoEntorno(clave string) int {
	valor := strings.TrimSpace(os.Getenv(clave))
	if valor == "" {
		return 0
	}
	numero, err := strconv.Atoi(valor)
	if err != nil || numero < 1 {
		return -1
	}
	return numero
}

func booleanoEntorno(clave string) bool {
	switch strings.ToLower(strings.TrimSpace(os.Getenv(clave))) {
	case "1", "true", "yes", "on":
		return true
	default:
		return false
	}
}
