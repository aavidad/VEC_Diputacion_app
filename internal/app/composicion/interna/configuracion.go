// Package interna define la raiz exclusiva del portal corporativo. C4 solo
// establece su frontera y permanece cerrada hasta que C5 y C6 aporten todas
// las dependencias productivas verificables.
package interna

import (
	"errors"
	"net"
	"os"
	"path/filepath"
	"strings"
	"time"
	"unicode"

	"vec-diputacion-granada/config"
	"vec-diputacion-granada/internal/vec/adapters/httpseguridad"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
)

const (
	EnvDireccionEscuchaInterna = "VEC_INTERNO_HTTP_ADDR"
	EnvRedesPermitidasInternas = "VEC_INTERNO_HTTP_ALLOWED_CIDRS"
	EnvCertificadoTLSInterno   = "VEC_INTERNO_TLS_CERT_FILE"
	EnvClaveTLSInterna         = "VEC_INTERNO_TLS_KEY_FILE"
	EnvAutoridadClientesTLS    = "VEC_INTERNO_TLS_CLIENT_CA_FILE"
	EnvNombreServidorTLS       = "VEC_INTERNO_TLS_SERVER_NAME"
	EnvAudienciaInterna        = "VEC_INTERNO_IDENTITY_AUDIENCE"
	EnvEmisorIdentidadInterna  = "VEC_INTERNO_IDENTITY_ISSUER"
	EnvHuellasProxyTLSInternas = "VEC_INTERNO_PROXY_TLS_SHA256"
	EnvIdentidadesSANProxy     = "VEC_INTERNO_PROXY_SAN_IDENTITIES"
)

var (
	ErrConfiguracionInternaInvalida = errors.New("composicion interna: configuracion invalida")
	ErrConfiguracionTLSIncompleta   = errors.New("composicion interna: referencias TLS mutuo incompletas")
	ErrSelectorHeredadoProhibido    = errors.New("composicion interna: selector heredado prohibido")
)

const (
	duracionMaximaAsercionInterna  = 3 * time.Minute
	edadMaximaAutenticacionInterna = 15 * time.Minute
	toleranciaRelojInterna         = 20 * time.Second
)

// Configuracion contiene solo transporte, limite de red y politica de
// identidad de la superficie interna. No contiene DSN, claves KMS/TSA ni
// credenciales: C5/C6 deberan inyectar proveedores ya construidos.
type Configuracion struct {
	DireccionEscucha     string
	RedesPermitidas      []string
	TiempoCabeceras      time.Duration
	TiempoLectura        time.Duration
	TiempoEscritura      time.Duration
	TiempoInactividad    time.Duration
	MaximoBytesCabeceras int
	MaximoBytesPeticion  int64

	CertificadoServidorTLS string
	ClaveServidorTLS       string
	AutoridadClientesTLS   string
	NombreServidorTLS      string
	Audiencia              string
	EmisorIdentidad        string
	HuellasProxyTLS        []string
	IdentidadesSANProxy    []string

	SelectorPerfilHeredado        string
	SelectorAutenticacionHeredado string
	SelectorAlmacenHeredado       string
	GuardaDesarrolloHeredada      string
	MaterialDesarrolloHeredado    string
	SelectorPresentacionHeredado  bool
}

// CargarConfiguracion lee una lista positiva de variables. Los selectores
// globales heredados se observan unicamente para rechazarlos; nunca seleccionan
// un proveedor de la superficie interna.
func CargarConfiguracion() Configuracion {
	return Configuracion{
		DireccionEscucha:       valorEntorno(EnvDireccionEscuchaInterna),
		RedesPermitidas:        listaEntorno(EnvRedesPermitidasInternas),
		TiempoCabeceras:        config.DefaultReadHeaderLimit,
		TiempoLectura:          config.DefaultReadTimeout,
		TiempoEscritura:        config.DefaultWriteTimeout,
		TiempoInactividad:      config.DefaultIdleTimeout,
		MaximoBytesCabeceras:   config.DefaultMaxHeaderBytes,
		MaximoBytesPeticion:    config.DefaultMaxRequestBodyBytes,
		CertificadoServidorTLS: valorEntorno(EnvCertificadoTLSInterno),
		ClaveServidorTLS:       valorEntorno(EnvClaveTLSInterna),
		AutoridadClientesTLS:   valorEntorno(EnvAutoridadClientesTLS),
		NombreServidorTLS:      valorEntorno(EnvNombreServidorTLS),
		Audiencia:              valorEntorno(EnvAudienciaInterna),
		EmisorIdentidad:        valorEntorno(EnvEmisorIdentidadInterna),
		HuellasProxyTLS:        listaEntorno(EnvHuellasProxyTLSInternas),
		IdentidadesSANProxy:    listaEntorno(EnvIdentidadesSANProxy),
		SelectorPerfilHeredado: valorEntorno(config.EnvExecutionProfile),
		SelectorAutenticacionHeredado: primerValorEntorno(
			config.EnvAuthMode,
			config.LegacyEnvAuthMode,
		),
		SelectorAlmacenHeredado: primerValorEntorno(
			config.EnvStorageMode,
			config.LegacyEnvStorageMode,
		),
		GuardaDesarrolloHeredada:   valorEntorno(config.EnvDevelopmentGuard),
		MaterialDesarrolloHeredado: valorEntorno(config.EnvDevelopmentMaterialDir),
		SelectorPresentacionHeredado: algunValorEntorno(
			config.EnvRRHHPresentationEnabled,
			config.EnvRRHHPresentationGuardOne,
			config.EnvRRHHPresentationGuardTwo,
		),
	}.normalizar()
}

// Validar exige una configuracion productiva cerrada y la politica corporativa
// Kerberos+certificado ya fijada por el contrato de superficies.
func (cfg Configuracion) Validar() error {
	cfg = cfg.normalizar()
	if cfg.SelectorPerfilHeredado != "" &&
		cfg.SelectorPerfilHeredado != config.ExecutionProfileProduction {
		return ErrSelectorHeredadoProhibido
	}
	if cfg.SelectorAutenticacionHeredado != "" || cfg.SelectorAlmacenHeredado != "" ||
		cfg.GuardaDesarrolloHeredada != "" || cfg.MaterialDesarrolloHeredado != "" ||
		cfg.SelectorPresentacionHeredado {
		return ErrSelectorHeredadoProhibido
	}
	if err := validarReferenciasTLS(cfg); err != nil {
		return err
	}
	if !nombreTLSValido(cfg.NombreServidorTLS) {
		return ErrConfiguracionTLSIncompleta
	}
	if err := cfg.configuracionSuperficie().Validar(); err != nil {
		// El contrato compartido incluye en algunas causas el literal recibido.
		// Esta raiz no lo propaga: direccion, CIDR e identificadores proceden del
		// entorno y no deben acabar en logs, respuestas ni trazas de arranque.
		return ErrConfiguracionInternaInvalida
	}
	return nil
}

func (cfg Configuracion) normalizar() Configuracion {
	cfg.DireccionEscucha = strings.TrimSpace(cfg.DireccionEscucha)
	cfg.RedesPermitidas = normalizarLista(cfg.RedesPermitidas)
	if cfg.TiempoCabeceras <= 0 {
		cfg.TiempoCabeceras = config.DefaultReadHeaderLimit
	}
	if cfg.TiempoLectura <= 0 {
		cfg.TiempoLectura = config.DefaultReadTimeout
	}
	if cfg.TiempoEscritura <= 0 {
		cfg.TiempoEscritura = config.DefaultWriteTimeout
	}
	if cfg.TiempoInactividad <= 0 {
		cfg.TiempoInactividad = config.DefaultIdleTimeout
	}
	if cfg.MaximoBytesCabeceras <= 0 {
		cfg.MaximoBytesCabeceras = config.DefaultMaxHeaderBytes
	}
	if cfg.MaximoBytesPeticion <= 0 {
		cfg.MaximoBytesPeticion = config.DefaultMaxRequestBodyBytes
	}
	cfg.CertificadoServidorTLS = strings.TrimSpace(cfg.CertificadoServidorTLS)
	cfg.ClaveServidorTLS = strings.TrimSpace(cfg.ClaveServidorTLS)
	cfg.AutoridadClientesTLS = strings.TrimSpace(cfg.AutoridadClientesTLS)
	cfg.NombreServidorTLS = strings.ToLower(strings.TrimSpace(cfg.NombreServidorTLS))
	cfg.Audiencia = strings.TrimSpace(cfg.Audiencia)
	cfg.EmisorIdentidad = strings.TrimSpace(cfg.EmisorIdentidad)
	cfg.HuellasProxyTLS = normalizarLista(cfg.HuellasProxyTLS)
	cfg.IdentidadesSANProxy = normalizarLista(cfg.IdentidadesSANProxy)
	cfg.SelectorPerfilHeredado = strings.ToLower(strings.TrimSpace(cfg.SelectorPerfilHeredado))
	cfg.SelectorAutenticacionHeredado = strings.TrimSpace(cfg.SelectorAutenticacionHeredado)
	cfg.SelectorAlmacenHeredado = strings.TrimSpace(cfg.SelectorAlmacenHeredado)
	cfg.GuardaDesarrolloHeredada = strings.TrimSpace(cfg.GuardaDesarrolloHeredada)
	cfg.MaterialDesarrolloHeredado = strings.TrimSpace(cfg.MaterialDesarrolloHeredado)
	return cfg
}

func (cfg Configuracion) configuracionSuperficie() httpseguridad.ConfiguracionSuperficie {
	return httpseguridad.ConfiguracionSuperficie{
		Superficie:                          httpseguridad.SuperficieInternaCorporativa,
		ZonaRed:                             httpseguridad.ZonaRedInterna,
		DireccionEscucha:                    cfg.DireccionEscucha,
		Audiencia:                           cfg.Audiencia,
		EmisorIdentidad:                     cfg.EmisorIdentidad,
		RedesPermitidas:                     append([]string(nil), cfg.RedesPermitidas...),
		HuellasProxyTLSPermitidas:           append([]string(nil), cfg.HuellasProxyTLS...),
		IdentidadesSANProxyPermitidas:       append([]string(nil), cfg.IdentidadesSANProxy...),
		DuracionMaximaAsercion:              duracionMaximaAsercionInterna,
		EdadMaximaAutenticacion:             edadMaximaAutenticacionInterna,
		ToleranciaReloj:                     toleranciaRelojInterna,
		MetodosAdmitidos:                    []httpseguridad.MetodoAutenticacion{httpseguridad.MetodoKerberos, httpseguridad.MetodoCertificado},
		FactoresRequeridos:                  []httpseguridad.MetodoAutenticacion{httpseguridad.MetodoKerberos, httpseguridad.MetodoCertificado},
		MinimoFactoresVerificados:           2,
		MinimoGruposCriptograficosDistintos: 2,
		GarantiaMinima:                      dominiovec.AuthAssuranceHigh,
	}
}

func validarReferenciasTLS(cfg Configuracion) error {
	rutas := []string{cfg.CertificadoServidorTLS, cfg.ClaveServidorTLS, cfg.AutoridadClientesTLS}
	vistas := make(map[string]struct{}, len(rutas))
	for _, ruta := range rutas {
		if ruta == "" || ruta == string(filepath.Separator) || !filepath.IsAbs(ruta) || filepath.Clean(ruta) != ruta ||
			strings.IndexFunc(ruta, unicode.IsControl) >= 0 {
			return ErrConfiguracionTLSIncompleta
		}
		if _, repetida := vistas[ruta]; repetida {
			return ErrConfiguracionTLSIncompleta
		}
		vistas[ruta] = struct{}{}
	}
	return nil
}

func nombreTLSValido(nombre string) bool {
	if nombre == "" || len(nombre) > 253 || strings.IndexFunc(nombre, unicode.IsControl) >= 0 {
		return false
	}
	if net.ParseIP(nombre) != nil {
		return true
	}
	for _, etiqueta := range strings.Split(nombre, ".") {
		if len(etiqueta) == 0 || len(etiqueta) > 63 || etiqueta[0] == '-' || etiqueta[len(etiqueta)-1] == '-' {
			return false
		}
		for _, caracter := range etiqueta {
			if (caracter < 'a' || caracter > 'z') && (caracter < '0' || caracter > '9') && caracter != '-' {
				return false
			}
		}
	}
	return true
}

func valorEntorno(clave string) string {
	return primerValorEntorno(clave)
}

func primerValorEntorno(claves ...string) string {
	for _, clave := range claves {
		if valor := strings.TrimSpace(os.Getenv(clave)); valor != "" {
			return valor
		}
	}
	return ""
}

func algunValorEntorno(claves ...string) bool {
	for _, clave := range claves {
		if strings.TrimSpace(os.Getenv(clave)) != "" {
			return true
		}
	}
	return false
}

func listaEntorno(clave string) []string {
	return normalizarLista(strings.Split(os.Getenv(clave), ","))
}

func normalizarLista(valores []string) []string {
	resultado := make([]string, 0, len(valores))
	vistos := make(map[string]struct{}, len(valores))
	for _, valor := range valores {
		valor = strings.TrimSpace(valor)
		if valor == "" {
			continue
		}
		if _, repetido := vistos[valor]; repetido {
			continue
		}
		vistos[valor] = struct{}{}
		resultado = append(resultado, valor)
	}
	return resultado
}
