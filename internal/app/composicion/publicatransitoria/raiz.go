// Package publicatransitoria contiene exclusivamente la consulta de ficheros
// no autoritativos usada por desarrollo y presentacion. No puede ser importado
// por cmd/vec-publico ni formar parte de su artefacto productivo.
package publicatransitoria

import (
	"context"
	"crypto/subtle"
	"errors"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"vec-diputacion-granada/config"
	"vec-diputacion-granada/internal/app/composicion/publica"
	bolsacatalogosvec "vec-diputacion-granada/internal/modules/bolsa/adapters/catalogosvec"
	bolsafichero "vec-diputacion-granada/internal/modules/bolsa/adapters/fichero"
	bolsahttp "vec-diputacion-granada/internal/modules/bolsa/adapters/httppublico"
	bolsaapp "vec-diputacion-granada/internal/modules/bolsa/application"
	vecfichero "vec-diputacion-granada/internal/vec/adapters/fichero"
)

var (
	ErrFueraEntornoNoAutoritativo       = errors.New("composicion publica transitoria: limitada a entornos no autoritativos")
	ErrAutenticacionNoAdmitida          = publica.ErrAutenticacionNoAdmitida
	ErrActivacionNoAutoritativaInvalida = errors.New("composicion publica transitoria: activacion no autoritativa invalida")
)

// NuevaAPI compone la consulta de ficheros de desarrollo o presentacion. La
// seleccion de un perfil productivo falla antes de abrir las fuentes.
func NuevaAPI(cfg config.Config) (http.Handler, error) {
	cfg = cfg.Normalize()
	if err := validarEntorno(cfg); err != nil {
		return nil, err
	}
	consultaCatalogos, err := nuevaConsultaCatalogos(cfg)
	if err != nil {
		return nil, err
	}
	return NuevaAPIConCatalogos(cfg, consultaCatalogos)
}

// NuevaAPIConCatalogos permite que una composicion no autoritativa use la
// misma instantanea inmutable para Bolsa y Personal.
func NuevaAPIConCatalogos(
	cfg config.Config,
	consultaCatalogos *vecfichero.ConsultaCatalogos,
) (http.Handler, error) {
	cfg = cfg.Normalize()
	if err := validarEntorno(cfg); err != nil {
		return nil, err
	}
	if err := validarCatalogoCategorias(cfg, consultaCatalogos); err != nil {
		return nil, err
	}
	ruta, err := resolverRutaFuente(cfg.BolsaPublicSourcePath)
	if err != nil {
		return nil, err
	}
	convocatorias, err := bolsafichero.NuevaConsultaConvocatorias(ruta)
	if err != nil {
		return nil, err
	}
	categorias, err := bolsacatalogosvec.NuevaConsultaCategorias(
		consultaCatalogos,
		cfg.BolsaCategoriesCatalogID,
		cfg.BolsaCategoriesVersion,
	)
	if err != nil {
		return nil, err
	}
	servicio, err := bolsaapp.NuevoServicioConsultaPublica(
		convocatorias,
		categorias,
		bolsaapp.RelojSistemaConsultaPublica{},
	)
	if err != nil {
		return nil, err
	}
	ctx, cancelar := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancelar()
	if err := servicio.ValidarConfiguracion(ctx); err != nil {
		return nil, errors.Join(errors.New("composicion publica transitoria: fuentes publicas de Bolsa incompatibles"), err)
	}
	manejador, err := bolsahttp.NuevoHandler(servicio)
	if err != nil {
		return nil, err
	}
	mux := http.NewServeMux()
	RegistrarRutas(mux, manejador)
	return mux, nil
}

// RegistrarRutas aplica una lista positiva; no existe ruta comodin.
func RegistrarRutas(mux *http.ServeMux, manejador http.Handler) {
	if mux == nil || manejador == nil {
		return
	}
	mux.Handle(bolsahttp.RutaConvocatorias, manejador)
	mux.Handle(bolsahttp.RutaConvocatorias+"/", manejador)
	mux.Handle(bolsahttp.RutaCategorias, manejador)
}

func validarEntorno(cfg config.Config) error {
	if cfg.ExecutionProfile == config.ExecutionProfileProduction {
		return publica.ErrComposicionProductivaNoDisponible
	}
	if cfg.AuthMode != config.AuthModeDisabled {
		return ErrAutenticacionNoAdmitida
	}
	switch cfg.ExecutionProfile {
	case config.ExecutionProfileRRHHPresentation:
		if !cfg.RRHHPresentationEnabledByDoubleGuard() {
			return ErrActivacionNoAutoritativaInvalida
		}
		return nil
	case config.ExecutionProfileDevelopment:
		// El submanejador publico es anonimo y por eso recibe AuthMode disabled,
		// pero no puede convertir el nombre del perfil en una via alternativa
		// para omitir las otras dos llaves de la composicion de desarrollo.
		if cfg.DevelopmentGuard != config.DevelopmentGuardAcknowledgement ||
			cfg.DevelopmentMaterialDir == "" || !filepath.IsAbs(cfg.DevelopmentMaterialDir) {
			return ErrActivacionNoAutoritativaInvalida
		}
		return nil
	default:
		return ErrFueraEntornoNoAutoritativo
	}
}

func nuevaConsultaCatalogos(cfg config.Config) (*vecfichero.ConsultaCatalogos, error) {
	if cfg.BolsaCategoriesVersion < 1 {
		return nil, errors.New("composicion publica transitoria: version de catalogo de categorias no valida")
	}
	ruta, err := resolverRutaFuente(cfg.BolsaCategoriesSourcePath)
	if err != nil {
		return nil, err
	}
	consulta, err := vecfichero.NuevaConsultaCatalogos(ruta)
	if err != nil {
		return nil, err
	}
	return consulta, nil
}

func validarCatalogoCategorias(cfg config.Config, consulta *vecfichero.ConsultaCatalogos) error {
	if cfg.BolsaCategoriesVersion < 1 || consulta == nil {
		return errors.New("composicion publica transitoria: catalogo gobernado de categorias de Bolsa incompatible")
	}
	ctx, cancelar := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancelar()
	catalogo, err := consulta.ObtenerCatalogo(ctx, cfg.BolsaCategoriesCatalogID, cfg.BolsaCategoriesVersion)
	if err != nil {
		return errors.Join(errors.New("composicion publica transitoria: catalogo gobernado de categorias de Bolsa incompatible"), err)
	}
	huella, err := catalogo.HuellaSHA256()
	if err != nil || !huellasIguales(huella, cfg.BolsaCategoriesSHA256) {
		return errors.New("composicion publica transitoria: catalogo gobernado de categorias de Bolsa incompatible")
	}
	return nil
}

func huellasIguales(obtenida, esperada string) bool {
	return len(obtenida) == 64 && len(esperada) == 64 &&
		subtle.ConstantTimeCompare([]byte(obtenida), []byte(esperada)) == 1
}

func resolverRutaFuente(configurada string) (string, error) {
	configurada = strings.TrimSpace(configurada)
	if configurada == "" {
		return "", errors.New("composicion publica transitoria: fuente de Bolsa no disponible")
	}
	if filepath.IsAbs(configurada) {
		if info, err := os.Stat(configurada); err == nil && !info.IsDir() {
			return configurada, nil
		}
		return "", errors.New("composicion publica transitoria: fuente de Bolsa no disponible")
	}
	actual, err := os.Getwd()
	if err != nil {
		return "", errors.New("composicion publica transitoria: fuente de Bolsa no disponible")
	}
	for {
		candidata := filepath.Join(actual, configurada)
		if info, err := os.Stat(candidata); err == nil && !info.IsDir() {
			return candidata, nil
		}
		padre := filepath.Dir(actual)
		if padre == actual {
			break
		}
		actual = padre
	}
	return "", errors.New("composicion publica transitoria: fuente de Bolsa no disponible")
}
