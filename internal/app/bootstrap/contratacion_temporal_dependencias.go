package bootstrap

import (
	"io"
	"sync"

	"vec-diputacion-granada/config"
	vechttp "vec-diputacion-granada/internal/vec/adapters/httpapi"
)

// DependenciasCT reúne las dependencias ya validadas por la raíz. No abre
// pools ni lee configuración: los constructores de módulos reciben esta misma
// instancia en vez de reconstruir credenciales o material de desarrollo.
type DependenciasCT struct {
	cfg        config.Config
	resolvedor *resolvedorIdentidadDesarrollo
	derivador  *derivadorIdentidadOperacionDesarrollo
	kms        *emisorKMSDesarrollo
	registro   io.Writer
	reloj      relojContratacionTemporalDesarrollo
	sello      *selloConsultasContratacionTemporalDesarrollo
	// reglasEjemplo solo existe en desarrollo con paquete de ejemplo
	// declarado; vacío significa «sin catálogo».
	reglasEjemplo reglasEjemploDesarrollo
	cerrar        func()
	unaVez        sync.Once
}

func nuevasDependenciasCT(cfg config.Config, resolvedor vechttp.DemoIdentityResolver, derivador *derivadorIdentidadOperacionDesarrollo, kms *emisorKMSDesarrollo, registro io.Writer) (*DependenciasCT, error) {
	cfg = cfg.Normalize()
	identidad, ok := resolvedor.(*resolvedorIdentidadDesarrollo)
	if !cfg.DevelopmentEnabledByDoubleKey() || validarRedLocalDesarrollo(cfg) != nil || !ok || identidad == nil || derivador == nil || !derivador.valido() {
		return nil, ErrActivacionDesarrolloInvalida
	}
	return &DependenciasCT{cfg: cfg, resolvedor: identidad, derivador: derivador, kms: kms, registro: registro, reloj: relojContratacionTemporalDesarrollo{}, sello: &selloConsultasContratacionTemporalDesarrollo{}, cerrar: func() {}}, nil
}

func (d *DependenciasCT) Cerrar() {
	if d != nil && d.cerrar != nil {
		d.unaVez.Do(d.cerrar)
	}
}
