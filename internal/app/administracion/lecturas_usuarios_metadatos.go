package administracion

import (
	"context"
	"errors"

	api "vec-diputacion-granada/internal/vec/adapters/httpapi/administracionperfiles"
	"vec-diputacion-granada/internal/vec/domain"
	"vec-diputacion-granada/internal/vec/ports"
)

// FuenteLecturasUsuariosMetadatos adapta el puerto nominal al contrato HTTP
// existente. Nunca obtiene nombres de un certificado, cargo o perfil.
type FuenteLecturasUsuariosMetadatos struct {
	lecturasNoDisponibles
	fuente ports.FuenteUsuariosAdministrables
}

var _ api.FuenteLecturas = (*FuenteLecturasUsuariosMetadatos)(nil)

func NuevaFuenteLecturasUsuariosMetadatos(fuente ports.FuenteUsuariosAdministrables) (*FuenteLecturasUsuariosMetadatos, error) {
	if dependenciaConfianzaPerfilesNula(fuente) {
		return nil, ErrConfiguracion
	}
	return &FuenteLecturasUsuariosMetadatos{fuente: fuente}, nil
}

func errorLecturaUsuarios(err error) error {
	if errors.Is(err, domain.ErrAutorizacionDenegada) {
		return domain.ErrAutorizacionDenegada
	}
	return ports.ErrAutoridadAdministracionPerfilesNoDisponible
}

func proyectarUsuarioMetadatos(p ports.UsuarioAdministrable) api.PersonaMetadatos {
	m := api.PersonaMetadatos{PersonaRef: p.PersonaRef, UnidadRef: p.UnidadRef, NombreEstado: "no_registrado", Perfiles: make([]api.PerfilUsuarioMetadatos, len(p.Perfiles))}
	if p.DenominacionVersion != nil {
		v := *p.DenominacionVersion
		m.DenominacionVersion, m.NombreEstado = &v, "no_consultado"
	}
	for i, perfil := range p.Perfiles {
		m.Perfiles[i] = api.PerfilUsuarioMetadatos{PerfilRef: perfil.PerfilRef, RolVersionRef: perfil.RolVersionRef, Version: perfil.Version, Estado: perfil.Estado, VigenteDesde: perfil.VigenteDesde, VigenteHasta: perfil.VigenteHasta}
	}
	return m
}

func (f *FuenteLecturasUsuariosMetadatos) BuscarPersonas(ctx context.Context, actor domain.ContextoActor, evidencia domain.EvidenciaSesionAdministracionPerfiles, consulta api.ConsultaPersonas) (api.PaginaPersonas, error) {
	if f == nil || ctx == nil || consulta.Texto != "" {
		return api.PaginaPersonas{}, ports.ErrAutoridadAdministracionPerfilesNoDisponible
	}
	p, err := f.fuente.ListarUsuarios(ctx, actor, evidencia, ports.FiltrosUsuariosAdministrables{PerfilRef: consulta.PerfilRef, UnidadRef: consulta.UnidadRef, Estado: consulta.Estado, Cursor: consulta.Cursor})
	if err != nil {
		return api.PaginaPersonas{}, errorLecturaUsuarios(err)
	}
	m := &api.PaginaPersonasMetadatos{Proyeccion: "metadatos_v1", Personas: make([]api.PersonaMetadatos, len(p.Personas)), SiguienteCursor: p.SiguienteCursor}
	for i, persona := range p.Personas {
		m.Personas[i] = proyectarUsuarioMetadatos(persona)
	}
	return api.PaginaPersonas{Metadatos: m}, nil
}

func (f *FuenteLecturasUsuariosMetadatos) ConsultarPersona(ctx context.Context, actor domain.ContextoActor, evidencia domain.EvidenciaSesionAdministracionPerfiles, persona string) (api.FichaPersona, error) {
	if f == nil || ctx == nil {
		return api.FichaPersona{}, ports.ErrAutoridadAdministracionPerfilesNoDisponible
	}
	p, err := f.fuente.ConsultarUsuario(ctx, actor, evidencia, persona)
	if err != nil {
		return api.FichaPersona{}, errorLecturaUsuarios(err)
	}
	if p == nil {
		return api.FichaPersona{}, api.ErrRecursoNoEncontrado
	}
	m := &api.FichaPersonaMetadatos{Proyeccion: "metadatos_v1", PersonaMetadatos: proyectarUsuarioMetadatos(*p), HistoriaEstado: "no_consultada", ActosEstado: "no_consultados"}
	return api.FichaPersona{PersonaRef: p.PersonaRef, Metadatos: m}, nil
}
