package modulos

import (
	"context"
	"reflect"

	domain "vec-diputacion-granada/internal/modules/administracion/domain/modulos"
	ports "vec-diputacion-granada/internal/modules/administracion/ports/modulos"
	vec "vec-diputacion-granada/internal/vec/domain"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

type Servicio struct {
	cfg       domain.Configuracion
	catalogos vecports.ConsultaCatalogosConfigurablesAcotada
	registro  ports.RegistroCompuesto
	reloj     vecports.Reloj
}

func Nuevo(cfg domain.Configuracion, catalogos vecports.ConsultaCatalogosConfigurablesAcotada, registro ports.RegistroCompuesto, reloj vecports.Reloj) (*Servicio, error) {
	if cfg.Validar() != nil || nulo(catalogos) || nulo(registro) || nulo(reloj) {
		return nil, domain.ErrConfiguracion
	}
	cfg.Gobernados = append([]string(nil), cfg.Gobernados...)
	return &Servicio{cfg: cfg, catalogos: catalogos, registro: registro, reloj: reloj}, nil
}

func (s *Servicio) vigente(ctx context.Context) (vec.CatalogoConfigurable, []vec.ModuleManifest, error) {
	if s == nil || ctx == nil {
		return vec.CatalogoConfigurable{}, nil, domain.ErrConfiguracion
	}
	if err := ctx.Err(); err != nil {
		return vec.CatalogoConfigurable{}, nil, err
	}
	limite := vecports.LimitesConsultaCatalogosAcotada{Versiones: 64, Entradas: 4096, Atributos: 8192, BytesAproximados: 4 << 20}
	resultado, err := s.catalogos.ListarVersionesCatalogoAcotado(ctx, s.cfg.CatalogoID, limite)
	if err != nil || resultado.Truncado {
		return vec.CatalogoConfigurable{}, nil, domain.ErrCatalogo
	}
	var vigente vec.CatalogoConfigurable
	versiones := map[int]bool{}
	for _, c := range resultado.Catalogos {
		if c.ID != s.cfg.CatalogoID || versiones[c.Version] || c.Validar() != nil {
			return vec.CatalogoConfigurable{}, nil, domain.ErrCatalogo
		}
		versiones[c.Version] = true
		if c.Estado != vec.EstadoCatalogoBorrador && c.Version > vigente.Version {
			vigente = c
		}
	}
	if vigente.Version == 0 {
		return vec.CatalogoConfigurable{}, nil, domain.ErrCatalogo
	}
	registrados, err := s.registro.ListModules(ctx)
	if err != nil {
		return vec.CatalogoConfigurable{}, nil, domain.ErrConfiguracion
	}
	if err := ctx.Err(); err != nil {
		return vec.CatalogoConfigurable{}, nil, err
	}
	return vigente, registrados, nil
}

func (s *Servicio) Consultar(ctx context.Context) (domain.Estado, error) {
	c, r, err := s.vigente(ctx)
	if err != nil {
		return domain.Estado{}, err
	}
	return domain.Proyectar(s.cfg, c, r, s.reloj.Ahora())
}

func (s *Servicio) PrepararCambio(ctx context.Context, cambio domain.Cambio) (domain.Preparacion, error) {
	c, r, err := s.vigente(ctx)
	if err != nil {
		return domain.Preparacion{}, err
	}
	return domain.Preparar(s.cfg, c, r, s.reloj.Ahora(), cambio)
}

func (s *Servicio) ExigirHabilitado(ctx context.Context, id string) error {
	// No se consulta el catálogo para el núcleo ni para ADMIN. La composición
	// sólo monta este guard sobre los IDs declarados como gobernados.
	if s == nil || ctx == nil {
		return domain.ErrConfiguracion
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	gobernado := false
	for _, m := range s.cfg.Gobernados {
		if m == id {
			gobernado = true
		}
	}
	if !gobernado {
		return domain.ErrConfiguracion
	}
	estado, err := s.Consultar(ctx)
	if err != nil {
		return err
	}
	return estado.ExigirHabilitado(id)
}

func nulo(v any) bool {
	if v == nil {
		return true
	}
	r := reflect.ValueOf(v)
	switch r.Kind() {
	case reflect.Pointer, reflect.Interface, reflect.Func, reflect.Map, reflect.Slice, reflect.Chan:
		return r.IsNil()
	}
	return false
}
