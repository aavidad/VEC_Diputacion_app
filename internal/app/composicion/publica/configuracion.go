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

	FuenteConvocatorias string
	FuenteCategorias    string
	CatalogoCategorias  string
	VersionCategorias   int
	HuellaCategorias    string

	PresentacionHabilitada bool
	GuardaPresentacionUno  string
	GuardaPresentacionDos  string
}

// CargarConfiguracion lee exclusivamente opciones de transporte, fuentes
// publicas y selectores necesarios para cerrar configuraciones inseguras. No
// consulta DSN, KMS, credenciales, almacenamiento interno ni cabeceras de
// identidad.
func CargarConfiguracion() Configuracion {
	return Configuracion{
		Direccion:               primerValorEntorno(config.EnvAddress, config.LegacyEnvAddress),
		TiempoCabeceras:         config.DefaultReadHeaderLimit,
		TiempoLectura:           config.DefaultReadTimeout,
		TiempoEscritura:         config.DefaultWriteTimeout,
		TiempoInactividad:       config.DefaultIdleTimeout,
		MaximoBytesCabeceras:    config.DefaultMaxHeaderBytes,
		MaximoBytesPeticion:     config.DefaultMaxRequestBodyBytes,
		RedesPermitidas:         separarLista(primerValorEntorno(config.EnvHTTPAllowedCIDRs)),
		CertificadoTLS:          primerValorEntorno(config.EnvTLSCertFile),
		ClaveTLS:                primerValorEntorno(config.EnvTLSKeyFile),
		PerfilEjecucion:         primerValorEntorno(config.EnvExecutionProfile),
		AutenticacionSolicitada: primerValorEntorno(config.EnvAuthMode, config.LegacyEnvAuthMode),
		GuardaDesarrollo:        primerValorEntorno(config.EnvDevelopmentGuard),
		MaterialDesarrollo:      primerValorEntorno(config.EnvDevelopmentMaterialDir),
		FuenteConvocatorias:     primerValorEntorno(config.EnvBolsaPublicSourcePath),
		FuenteCategorias:        primerValorEntorno(config.EnvBolsaCategoriesSourcePath),
		CatalogoCategorias:      primerValorEntorno(config.EnvBolsaCategoriesCatalogID),
		VersionCategorias:       enteroPositivoEntorno(config.EnvBolsaCategoriesVersion),
		HuellaCategorias:        primerValorEntorno(config.EnvBolsaCategoriesSHA256),
		PresentacionHabilitada:  booleanoEntorno(config.EnvRRHHPresentationEnabled),
		GuardaPresentacionUno:   primerValorEntorno(config.EnvRRHHPresentationGuardOne),
		GuardaPresentacionDos:   primerValorEntorno(config.EnvRRHHPresentationGuardTwo),
	}.normalizar()
}

// DesdeConfiguracionGeneral adapta llamadas heredadas sin introducir el
// bootstrap monolitico en el grafo del binario publico.
func DesdeConfiguracionGeneral(cfg config.Config) Configuracion {
	cfg = cfg.Normalize()
	return Configuracion{
		Direccion:               cfg.Address,
		TiempoCabeceras:         cfg.ReadHeaderTimeout,
		TiempoLectura:           cfg.ReadTimeout,
		TiempoEscritura:         cfg.WriteTimeout,
		TiempoInactividad:       cfg.IdleTimeout,
		MaximoBytesCabeceras:    cfg.MaxHeaderBytes,
		MaximoBytesPeticion:     cfg.MaxRequestBodyBytes,
		RedesPermitidas:         append([]string(nil), cfg.HTTPAllowedCIDRs...),
		CertificadoTLS:          cfg.TLSCertFile,
		ClaveTLS:                cfg.TLSKeyFile,
		PerfilEjecucion:         cfg.ExecutionProfile,
		AutenticacionSolicitada: cfg.AuthMode,
		GuardaDesarrollo:        cfg.DevelopmentGuard,
		MaterialDesarrollo:      cfg.DevelopmentMaterialDir,
		FuenteConvocatorias:     cfg.BolsaPublicSourcePath,
		FuenteCategorias:        cfg.BolsaCategoriesSourcePath,
		CatalogoCategorias:      cfg.BolsaCategoriesCatalogID,
		VersionCategorias:       cfg.BolsaCategoriesVersion,
		HuellaCategorias:        cfg.BolsaCategoriesSHA256,
		PresentacionHabilitada:  cfg.RRHHPresentationEnabled,
		GuardaPresentacionUno:   cfg.RRHHPresentationGuardOne,
		GuardaPresentacionDos:   cfg.RRHHPresentationGuardTwo,
	}
}

func (cfg Configuracion) normalizar() Configuracion {
	general := config.Config{
		Address:                   cfg.Direccion,
		ReadHeaderTimeout:         cfg.TiempoCabeceras,
		ReadTimeout:               cfg.TiempoLectura,
		WriteTimeout:              cfg.TiempoEscritura,
		IdleTimeout:               cfg.TiempoInactividad,
		MaxHeaderBytes:            cfg.MaximoBytesCabeceras,
		MaxRequestBodyBytes:       cfg.MaximoBytesPeticion,
		HTTPAllowedCIDRs:          append([]string(nil), cfg.RedesPermitidas...),
		TLSCertFile:               cfg.CertificadoTLS,
		TLSKeyFile:                cfg.ClaveTLS,
		ExecutionProfile:          cfg.PerfilEjecucion,
		AuthMode:                  cfg.AutenticacionSolicitada,
		DevelopmentGuard:          cfg.GuardaDesarrollo,
		DevelopmentMaterialDir:    cfg.MaterialDesarrollo,
		BolsaPublicSourcePath:     cfg.FuenteConvocatorias,
		BolsaCategoriesSourcePath: cfg.FuenteCategorias,
		BolsaCategoriesCatalogID:  cfg.CatalogoCategorias,
		BolsaCategoriesVersion:    cfg.VersionCategorias,
		BolsaCategoriesSHA256:     cfg.HuellaCategorias,
		RRHHPresentationEnabled:   cfg.PresentacionHabilitada,
		RRHHPresentationGuardOne:  cfg.GuardaPresentacionUno,
		RRHHPresentationGuardTwo:  cfg.GuardaPresentacionDos,
	}.Normalize()
	return DesdeConfiguracionGeneral(general)
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
