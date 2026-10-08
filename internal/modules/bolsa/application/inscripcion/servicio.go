package inscripcion

import (
	"context"
	"reflect"
)

type Servicio struct{ repositorio Repositorio }

func lecturaAutorizada(actor Actor, accion string, filtro Filtro, ref string) bool {
	idioma := actor.Idioma
	if idioma == "" {
		idioma = "es"
	}
	recurso, err := RecursoLectura(accion, actor.PersonaRef, idioma, filtro, ref)
	return err == nil && actor.LecturaValida(accion, recurso, filtro)
}

func NuevoServicio(r Repositorio) (*Servicio, error) {
	if r == nil || reflect.ValueOf(r).Kind() == reflect.Pointer && reflect.ValueOf(r).IsNil() {
		return nil, ErrNoDisponible
	}
	return &Servicio{repositorio: r}, nil
}

func (s *Servicio) Abiertas(ctx context.Context, actor Actor, limite int, cursor string) (PaginaAbiertas, error) {
	if s == nil || s.repositorio == nil {
		return PaginaAbiertas{}, ErrNoDisponible
	}
	if (cursor != "" && !convocatoriaRefValida(cursor)) ||
		!lecturaAutorizada(actor, AccionListarAbiertas, Filtro{Limite: limite, Cursor: cursor}, "") || limite < 1 || limite > 100 {
		return PaginaAbiertas{}, ErrSolicitudInvalida
	}
	p, err := s.repositorio.Abiertas(ctx, actor, limite, cursor)
	if err != nil {
		return PaginaAbiertas{}, err
	}
	if len(p.Bolsas) > limite || uint64(len(p.Bolsas)) > p.Total {
		return PaginaAbiertas{}, ErrNoDisponible
	}
	for _, b := range p.Bolsas {
		if !bolsaAbiertaValida(b) {
			return PaginaAbiertas{}, ErrNoDisponible
		}
	}
	return p, nil
}

func (s *Servicio) DetalleAbierta(ctx context.Context, actor Actor, ref string) (BolsaAbierta, error) {
	if s == nil || s.repositorio == nil {
		return BolsaAbierta{}, ErrNoDisponible
	}
	if !lecturaAutorizada(actor, AccionDetalleAbierta, Filtro{}, ref) {
		return BolsaAbierta{}, ErrSolicitudInvalida
	}
	b, err := s.repositorio.DetalleAbierta(ctx, actor, ref)
	if err != nil {
		return BolsaAbierta{}, err
	}
	if !bolsaAbiertaValida(b) || b.ConvocatoriaRef != ref {
		return BolsaAbierta{}, ErrNoDisponible
	}
	return b, nil
}

func (s *Servicio) MotivosRRHH(ctx context.Context, actor Actor, decision string) (CatalogoMotivos, error) {
	if s == nil || s.repositorio == nil {
		return CatalogoMotivos{}, ErrNoDisponible
	}
	if (decision != "admitir" && decision != "rechazar") || !lecturaAutorizada(actor, AccionMotivosRRHH, Filtro{}, decision) {
		return CatalogoMotivos{}, ErrSolicitudInvalida
	}
	c, err := s.repositorio.MotivosRRHH(ctx, actor, decision)
	if err != nil {
		return CatalogoMotivos{}, err
	}
	if c.Version == 0 {
		return CatalogoMotivos{}, ErrNoDisponible
	}
	for _, m := range c.Motivos {
		if !referenciaOpaca.MatchString(m.Codigo) || m.Etiqueta == "" {
			return CatalogoMotivos{}, ErrNoDisponible
		}
	}
	return c, nil
}

func bolsaAbiertaValida(b BolsaAbierta) bool {
	if !convocatoriaRefValida(b.ConvocatoriaRef) || b.Titulo == "" ||
		b.CategoriasResumen == "" || len(b.Categorias) == 0 || len(b.Categorias) > 32 ||
		b.CatalogoVersion == 0 ||
		b.PlazoInicio.IsZero() || !b.PlazoFin.After(b.PlazoInicio) ||
		b.RequisitosResumen == "" ||
		(b.EstadoPropio == nil && b.SolicitudRef != nil) ||
		(b.EstadoPropio != nil && b.SolicitudRef == nil) {
		return false
	}
	if b.EstadoPropio != nil && b.SolicitudRef != nil &&
		(*b.EstadoPropio != EstadoPendiente && *b.EstadoPropio != EstadoAdmitidaAConvocatoria && *b.EstadoPropio != EstadoIncorporada && *b.EstadoPropio != EstadoRechazada ||
			!referenciaOpaca.MatchString(*b.SolicitudRef)) {
		return false
	}
	for _, r := range b.Requisitos {
		if !referenciaOpaca.MatchString(r.Codigo) || r.Descripcion == "" ||
			(r.Estado != "cumple" && r.Estado != "no_cumple" && r.Estado != "pendiente") ||
			r.MotivoEtiqueta == "" ||
			(r.HitoCumplimiento == nil) != (r.HitoEtiqueta == nil) {
			return false
		}
	}
	for _, c := range b.Categorias {
		if !referenciaOpaca.MatchString(c.CategoriaRef) || len(c.CategoriaRef) > 200 || c.Categoria == "" || len(c.Categoria) > 200 {
			return false
		}
	}
	if !b.PuedeIniciar && b.ImpedimentoEtiqueta == "" {
		return false
	}
	return true
}

func (s *Servicio) Presentar(ctx context.Context, actor Actor, p Presentacion) (Recibo, error) {
	if s == nil || s.repositorio == nil {
		return Recibo{}, ErrNoDisponible
	}
	if !actor.EscrituraValida() || p.Validar() != nil {
		return Recibo{}, ErrSolicitudInvalida
	}
	r, err := s.repositorio.Presentar(ctx, actor, p)
	if err != nil {
		return Recibo{}, err
	}
	if r.Solicitud.Validar() != nil || r.ConvocatoriaRef != p.ConvocatoriaRef ||
		r.CategoriaRef != p.CategoriaRef || r.Estado != EstadoPendiente {
		return Recibo{}, ErrNoDisponible
	}
	return r, nil
}

func (s *Servicio) Propias(ctx context.Context, actor Actor, filtro Filtro) (Pagina, error) {
	if s == nil || s.repositorio == nil {
		return Pagina{}, ErrNoDisponible
	}
	if filtro.Validar() != nil || (filtro.Cursor != "" && !solicitudRefValida(filtro.Cursor)) ||
		!lecturaAutorizada(actor, AccionListarPropias, filtro, "") {
		return Pagina{}, ErrSolicitudInvalida
	}
	p, err := s.repositorio.Propias(ctx, actor, filtro)
	if err != nil {
		return Pagina{}, err
	}
	if !paginaValida(p, filtro.Limite) {
		return Pagina{}, ErrNoDisponible
	}
	return p, nil
}

func (s *Servicio) Propia(ctx context.Context, actor Actor, ref string) (Solicitud, error) {
	if s == nil || s.repositorio == nil {
		return Solicitud{}, ErrNoDisponible
	}
	if !lecturaAutorizada(actor, AccionDetallePropia, Filtro{}, ref) {
		return Solicitud{}, ErrSolicitudInvalida
	}
	r, err := s.repositorio.Propia(ctx, actor, ref)
	if err != nil {
		return Solicitud{}, err
	}
	if r.Validar() != nil || r.SolicitudRef != ref {
		return Solicitud{}, ErrNoDisponible
	}
	return r, nil
}

func (s *Servicio) PendientesRRHH(ctx context.Context, actor Actor, filtro Filtro) (Pagina, error) {
	if s == nil || s.repositorio == nil {
		return Pagina{}, ErrNoDisponible
	}
	if filtro.Validar() != nil || (filtro.Cursor != "" && !solicitudRefValida(filtro.Cursor)) ||
		!lecturaAutorizada(actor, AccionListarRRHH, filtro, "") {
		return Pagina{}, ErrSolicitudInvalida
	}
	p, err := s.repositorio.PendientesRRHH(ctx, actor, filtro)
	if err != nil {
		return Pagina{}, err
	}
	if !paginaValida(p, filtro.Limite) {
		return Pagina{}, ErrNoDisponible
	}
	return p, nil
}

func (s *Servicio) DetalleRRHH(ctx context.Context, actor Actor, ref string) (Solicitud, error) {
	if s == nil || s.repositorio == nil {
		return Solicitud{}, ErrNoDisponible
	}
	if !lecturaAutorizada(actor, AccionDetalleRRHH, Filtro{}, ref) {
		return Solicitud{}, ErrSolicitudInvalida
	}
	r, err := s.repositorio.DetalleRRHH(ctx, actor, ref)
	if err != nil {
		return Solicitud{}, err
	}
	if r.Validar() != nil || r.SolicitudRef != ref {
		return Solicitud{}, ErrNoDisponible
	}
	return r, nil
}

func (s *Servicio) Decidir(ctx context.Context, actor Actor, d Decision) (Recibo, error) {
	if s == nil || s.repositorio == nil {
		return Recibo{}, ErrNoDisponible
	}
	if !actor.EscrituraValida() || actor.Canal != "interna_corporativa" || d.Validar() != nil {
		return Recibo{}, ErrSolicitudInvalida
	}
	r, err := s.repositorio.Decidir(ctx, actor, d)
	if err != nil {
		return Recibo{}, err
	}
	if r.Solicitud.Validar() != nil || r.SolicitudRef != d.SolicitudRef ||
		(d.Tipo == "rechazar" && r.Estado != EstadoRechazada) ||
		(d.Tipo == "admitir" && r.Estado != EstadoAdmitidaAConvocatoria && r.Estado != EstadoIncorporada) {
		return Recibo{}, ErrNoDisponible
	}
	return r, nil
}

func (s *Servicio) Incorporar(ctx context.Context, actor Actor, i Incorporacion) (Recibo, error) {
	if s == nil || s.repositorio == nil {
		return Recibo{}, ErrNoDisponible
	}
	if !actor.EscrituraValida() || actor.Canal != "interna_corporativa" || i.Validar() != nil {
		return Recibo{}, ErrSolicitudInvalida
	}
	r, err := s.repositorio.Incorporar(ctx, actor, i)
	if err != nil {
		return Recibo{}, err
	}
	if r.Solicitud.Validar() != nil || r.SolicitudRef != i.SolicitudRef || r.Estado != EstadoIncorporada {
		return Recibo{}, ErrNoDisponible
	}
	return r, nil
}

func paginaValida(p Pagina, limite int) bool {
	if len(p.Solicitudes) > limite || uint64(len(p.Solicitudes)) > p.Total {
		return false
	}
	for _, s := range p.Solicitudes {
		if s.Validar() != nil {
			return false
		}
	}
	return true
}
