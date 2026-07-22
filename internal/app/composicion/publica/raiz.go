package publica

import (
	"context"
	"errors"
	"net/http"
	"time"

	"vec-diputacion-granada/config"
	"vec-diputacion-granada/internal/app/server"
	bolsapostgrespublico "vec-diputacion-granada/internal/modules/bolsa/adapters/postgrespublico"
	bolsaapp "vec-diputacion-granada/internal/modules/bolsa/publico/aplicacion"
	bolsahttp "vec-diputacion-granada/internal/modules/bolsa/publico/httpapi"
)

var (
	ErrComposicionProductivaNoDisponible = errors.New("composicion publica: proveedor autoritativo no disponible")
	ErrPerfilEjecucionDesconocido        = errors.New("composicion publica: perfil de ejecucion desconocido")
	ErrModoAutenticacionDesconocido      = errors.New("composicion publica: modo de autenticacion desconocido")
	ErrAutenticacionNoAdmitida           = errors.New("composicion publica: la superficie anonima exige autenticacion deshabilitada")
	ErrActivacionDesarrolloInvalida      = errors.New("composicion publica: selectores de desarrollo prohibidos")
	ErrSelectoresPresentacionProhibidos  = errors.New("composicion publica: selectores de presentacion prohibidos")
	ErrConfiguracionAutoritativaInvalida = errors.New("composicion publica: configuracion autoritativa invalida")
)

// NuevoServidor compone exclusivamente la proyeccion PostgreSQL publica. Los
// adaptadores JSON de desarrollo no estan importados por este paquete ni
// pueden seleccionarse mediante configuracion.
func NuevoServidor(cfg Configuracion) (*http.Server, error) {
	cfg = cfg.normalizar()
	if err := validarSelectoresConocidos(cfg); err != nil {
		return nil, err
	}
	if cfg.AutenticacionSolicitada != config.AuthModeDisabled {
		return nil, ErrAutenticacionNoAdmitida
	}
	if tieneSelectoresPresentacion(cfg) {
		return nil, ErrSelectoresPresentacionProhibidos
	}
	if cfg.PerfilEjecucion == config.ExecutionProfileDevelopment ||
		cfg.GuardaDesarrollo != "" || cfg.MaterialDesarrollo != "" ||
		cfg.FuenteConvocatorias != "" || cfg.FuenteCategorias != "" {
		return nil, ErrActivacionDesarrolloInvalida
	}
	if cfg.PerfilEjecucion != config.ExecutionProfileProduction {
		return nil, ErrConfiguracionAutoritativaInvalida
	}
	dsn, err := cfg.PostgreSQL.DSN()
	if err != nil {
		return nil, errors.Join(ErrConfiguracionAutoritativaInvalida, err)
	}
	if cfg.CatalogoCategorias == "" || cfg.VersionCategorias < 1 || cfg.HuellaCategorias == "" ||
		cfg.HuellaProyeccionCategorias == "" ||
		config.ValidarHuellaManifiestoPublico(cfg.HuellaManifiesto) != nil {
		return nil, ErrConfiguracionAutoritativaInvalida
	}
	ctxConexion, cancelarConexion := context.WithTimeout(context.Background(), 15*time.Second)
	fuente, err := bolsapostgrespublico.Abrir(
		ctxConexion, dsn, cfg.CatalogoCategorias, cfg.VersionCategorias, cfg.HuellaCategorias,
		cfg.HuellaProyeccionCategorias, cfg.HuellaManifiesto,
	)
	cancelarConexion()
	if err != nil {
		return nil, err
	}
	servicio, err := bolsaapp.NuevoServicioConsultaPublicaConsistente(
		fuente, bolsaapp.RelojSistemaConsultaPublica{},
	)
	if err != nil {
		fuente.Cerrar()
		return nil, err
	}
	// La validación recorre y coteja todas las huellas publicadas por lotes.
	// No comparte el presupuesto corto reservado al establecimiento TLS.
	ctxValidacion, cancelarValidacion := context.WithTimeout(context.Background(), 2*time.Minute)
	err = servicio.ValidarConfiguracion(ctxValidacion)
	cancelarValidacion()
	if err != nil {
		fuente.Cerrar()
		return nil, errors.Join(ErrConfiguracionAutoritativaInvalida, err)
	}
	manejador, err := bolsahttp.NuevoHandler(servicio)
	if err != nil {
		fuente.Cerrar()
		return nil, err
	}
	api := http.NewServeMux()
	api.Handle(bolsahttp.RutaConvocatorias, manejador)
	api.Handle(bolsahttp.RutaConvocatorias+"/", manejador)
	api.Handle(bolsahttp.RutaCategorias, manejador)
	servidor, err := server.NewHTTPServerPublicoConComprobadorDisponibilidad(configuracionHTTP(cfg), api, fuente)
	if err != nil {
		fuente.Cerrar()
		return nil, err
	}
	servidor.RegisterOnShutdown(fuente.Cerrar)
	return servidor, nil
}

func configuracionHTTP(cfg Configuracion) config.Config {
	return config.Config{
		Address: cfg.Direccion, ReadHeaderTimeout: cfg.TiempoCabeceras,
		ReadTimeout: cfg.TiempoLectura, WriteTimeout: cfg.TiempoEscritura,
		IdleTimeout: cfg.TiempoInactividad, MaxHeaderBytes: cfg.MaximoBytesCabeceras,
		MaxRequestBodyBytes: cfg.MaximoBytesPeticion,
		HTTPAllowedCIDRs:    append([]string(nil), cfg.RedesPermitidas...),
		TLSCertFile:         cfg.CertificadoTLS, TLSKeyFile: cfg.ClaveTLS,
		ExecutionProfile: cfg.PerfilEjecucion, AuthMode: config.AuthModeDisabled,
	}.NormalizePublicTransport()
}

func validarSelectoresConocidos(cfg Configuracion) error {
	switch cfg.PerfilEjecucion {
	case config.ExecutionProfileProduction, config.ExecutionProfileDevelopment, config.ExecutionProfileRRHHPresentation:
	default:
		return ErrPerfilEjecucionDesconocido
	}
	switch cfg.AutenticacionSolicitada {
	case config.AuthModeDisabled, config.AuthModeFake, config.AuthModeTrustedHeaders, config.AuthModeDevelopment:
	default:
		return ErrModoAutenticacionDesconocido
	}
	return nil
}

func tieneSelectoresPresentacion(cfg Configuracion) bool {
	return cfg.PerfilEjecucion == config.ExecutionProfileRRHHPresentation ||
		cfg.PresentacionHabilitada || cfg.GuardaPresentacionUno != "" ||
		cfg.GuardaPresentacionDos != ""
}
