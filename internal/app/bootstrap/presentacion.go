package bootstrap

import (
	"errors"
	"net/http"
	"path/filepath"
	"strings"

	"vec-diputacion-granada/config"
	"vec-diputacion-granada/internal/app/server"
	personalrpt "vec-diputacion-granada/internal/modules/personal/adapters/rptpublica"
	personalapp "vec-diputacion-granada/internal/modules/personal/application"
	vechttp "vec-diputacion-granada/internal/vec/adapters/httpapi"
)

var (
	ErrPresentacionRRHHEnComposicionNormal = errors.New("bootstrap: selectores de presentacion RRHH prohibidos en composicion normal")
	ErrComposicionPresentacionRRHHInvalida = errors.New("bootstrap: composicion de presentacion RRHH incompleta o insegura")
)

// NewHTTPServerPresentacionWithConfig es la raiz minima y exclusiva del
// artefacto de presentacion. Solo compone la consulta publica desde JSON
// sintetico y el servidor de estaticos allowlisted; no crea identidad,
// PostgreSQL, S3, firma, registro, pagos, comunicaciones ni clientes de red.
func NewHTTPServerPresentacionWithConfig(cfg config.Config) (*http.Server, error) {
	cfg = cfg.Normalize()
	if !configuracionPresentacionSinteticaValida(cfg) {
		return nil, ErrComposicionPresentacionRRHHInvalida
	}
	apiPublica, err := NewAPIPublicaBolsaWithConfig(cfg)
	if err != nil {
		return nil, errors.Join(ErrComposicionPresentacionRRHHInvalida, err)
	}
	return server.NewHTTPServerPresentacion(cfg, apiPublica)
}

// NewHTTPServerPresentacionPersonalWithConfig concede, solo en el perfil
// presentacion_rrhh doblemente protegido, la lectura sintética de la colección
// de categorías profesionales. No representa una identidad corporativa ni
// compone sesiones, autorización general, auditoría durable o efectos.
func NewHTTPServerPresentacionPersonalWithConfig(cfg config.Config) (*http.Server, error) {
	cfg = cfg.Normalize()
	if !configuracionPresentacionSinteticaValida(cfg) {
		return nil, ErrComposicionPresentacionRRHHInvalida
	}
	apiPublica, err := NewAPIPublicaBolsaWithConfig(cfg)
	if err != nil {
		return nil, errors.Join(ErrComposicionPresentacionRRHHInvalida, err)
	}
	_, categorias, err := nuevasDependenciasCategoriasProfesionales(cfg)
	if err != nil {
		return nil, errors.Join(ErrComposicionPresentacionRRHHInvalida, err)
	}
	concesionCategorias, err := vechttp.NewHandlerCategoriasProfesionalesPresentacion(cfg, categorias)
	if err != nil {
		return nil, errors.Join(ErrComposicionPresentacionRRHHInvalida, err)
	}
	return server.NewHTTPServerPresentacionCategorias(
		cfg,
		apiPublica,
		concesionCategorias,
	)
}

// NewHTTPServerPresentacionPersonalRPTWithConfig compone la fuente publica RPT
// inmovilizada por huella y su proyeccion minima, junto a la vista Personal.
func NewHTTPServerPresentacionPersonalRPTWithConfig(cfg config.Config) (*http.Server, error) {
	cfg = cfg.Normalize()
	if !configuracionPresentacionSinteticaValida(cfg) {
		return nil, ErrComposicionPresentacionRRHHInvalida
	}
	apiPublica, err := NewAPIPublicaBolsaWithConfig(cfg)
	if err != nil {
		return nil, errors.Join(ErrComposicionPresentacionRRHHInvalida, err)
	}
	_, categorias, err := nuevasDependenciasCategoriasProfesionales(cfg)
	if err != nil {
		return nil, errors.Join(ErrComposicionPresentacionRRHHInvalida, err)
	}
	concesionCategorias, err := vechttp.NewHandlerCategoriasProfesionalesPresentacion(cfg, categorias)
	if err != nil {
		return nil, errors.Join(ErrComposicionPresentacionRRHHInvalida, err)
	}
	if strings.TrimSpace(cfg.RPTCatalogoPath) == "" {
		return nil, errors.Join(ErrComposicionPresentacionRRHHInvalida, errors.New("bootstrap: fuente RPT publica requerida"))
	}
	fuente, err := personalrpt.NuevaFuente(cfg.RPTCatalogoPath)
	if err != nil {
		return nil, errors.Join(ErrComposicionPresentacionRRHHInvalida, err)
	}
	consultaRPT, err := personalapp.NuevoServicioConsultaRPTPublica(fuente)
	if err != nil {
		return nil, errors.Join(ErrComposicionPresentacionRRHHInvalida, err)
	}
	concesionRPT, err := vechttp.NewHandlerRPTPublicaPresentacion(cfg, consultaRPT)
	if err != nil {
		return nil, errors.Join(ErrComposicionPresentacionRRHHInvalida, err)
	}
	return server.NewHTTPServerPresentacionPersonalRPT(cfg, apiPublica, concesionCategorias, concesionRPT)
}

func configuracionPresentacionSinteticaValida(cfg config.Config) bool {
	return cfg.RRHHPresentationEnabledByDoubleGuard() &&
		cfg.AuthMode == config.AuthModeDisabled && cfg.StorageMode == config.StorageModeMemory &&
		cfg.FakeCredentialsPath == "" && cfg.PersonalCatalogPath == "" && cfg.PersonalCatalogInMemory &&
		cfg.OSRMBaseURL == "" && cfg.OSRMScopeName == "" && cfg.OSRMScopeBounds == "" &&
		len(cfg.OSRMAllowedCIDRs) == 0 && cfg.OSRMGraphVersion == "" &&
		rutaSinteticaPresentacion(cfg.BolsaPublicSourcePath) &&
		rutaSinteticaPresentacion(cfg.BolsaCategoriesSourcePath)
}

func rechazarSelectoresPresentacionEnComposicionNormal(cfg config.Config) error {
	if cfg.HasRRHHPresentationSelectors() {
		return ErrPresentacionRRHHEnComposicionNormal
	}
	return nil
}

func rutaSinteticaPresentacion(ruta string) bool {
	base := strings.ToLower(filepath.Base(strings.TrimSpace(ruta)))
	return strings.HasSuffix(base, ".demo.json") && base != ".demo.json"
}
