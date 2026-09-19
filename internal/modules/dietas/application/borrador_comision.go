package application

import (
	"context"
	"errors"
	"reflect"
	"regexp"
	"time"
	"vec-diputacion-granada/internal/modules/dietas/domain"
	dietasports "vec-diputacion-granada/internal/modules/dietas/ports"
	vecdomain "vec-diputacion-granada/internal/vec/domain"
)

var ErrComposicionBorradorInvalida = errors.New("dietas: composicion de borrador invalida")
var claveOperacion = regexp.MustCompile(`^[A-Za-z0-9_-]{16,128}$`)
var referenciaBorrador = regexp.MustCompile(`^[a-z][A-Za-z0-9_:-]{2,180}$`)

type ServicioBorradorComision struct {
	unidad   dietasports.UnidadTrabajoBorradorComision
	politica dietasports.ProveedorPoliticaKilometrajeBorrador
	zona     *time.Location
}

func NuevoServicioBorradorComision(u dietasports.UnidadTrabajoBorradorComision, p dietasports.ProveedorPoliticaKilometrajeBorrador, zona *time.Location) (*ServicioBorradorComision, error) {
	if unidadNula(u) {
		return nil, ErrComposicionBorradorInvalida
	}
	return &ServicioBorradorComision{unidad: u, politica: p, zona: zona}, nil
}
func (s *ServicioBorradorComision) CrearPropio(c context.Context, x dietasports.SolicitudCrearBorradorPropio) (dietasports.ReciboBorradorComision, error) {
	if s == nil || unidadNula(s.unidad) || c == nil || x.ContextoActor.Validar() != nil || !claveOperacion.MatchString(x.ClaveOperacion) || x.VersionEsperada != 0 || x.Borrador.PersonaRef != x.ContextoActor.PersonaRef {
		return dietasports.ReciboBorradorComision{}, domain.ErrBorradorComisionInvalido
	}
	if _, err := EmpleadoBorradorPropio(x.ContextoActor); err != nil {
		return dietasports.ReciboBorradorComision{}, err
	}
	if dependenciaNula(s.politica) || s.zona == nil {
		return dietasports.ReciboBorradorComision{}, dietasports.ErrPoliticaBorradorNoDisponible
	}
	inicio, e := domain.ResolverInstanteCivil(x.Borrador.FechaInicio, x.Borrador.HoraInicio, s.zona)
	if e != nil {
		return dietasports.ReciboBorradorComision{}, e
	}
	p, e := s.politica.ResolverPoliticaKilometraje(c, dietasports.SolicitudPoliticaKilometraje{ContextoActor: x.ContextoActor, Referencia: x.PoliticaReferencia, Version: x.PoliticaVersion, Inicio: inicio, VehiculoPropio: x.Borrador.VehiculoPropio})
	if e != nil || p.Validar() != nil || (x.PoliticaReferencia != "" && p.Referencia != x.PoliticaReferencia) || (x.PoliticaVersion != "" && p.Version != x.PoliticaVersion) {
		return dietasports.ReciboBorradorComision{}, dietasports.ErrPoliticaBorradorNoDisponible
	}
	x.Borrador, e = domain.PrepararBorrador(x.Borrador, p, s.zona)
	if e != nil {
		return dietasports.ReciboBorradorComision{}, e
	}
	r, e := s.unidad.CrearBorradorPropio(c, x)
	if e != nil {
		return dietasports.ReciboBorradorComision{}, e
	}
	if !reciboValido(r) || r.Version != 1 {
		return dietasports.ReciboBorradorComision{}, ErrComposicionBorradorInvalida
	}
	return r, nil
}

func (s *ServicioBorradorComision) ListarPropios(c context.Context, a vecdomain.ContextoActor, q dietasports.ConsultaBorradoresPropios) (dietasports.PaginaBorradoresPropios, error) {
	if s == nil || unidadNula(s.unidad) || c == nil || a.Validar() != nil || q.Limite < 1 || q.Limite > 20 || (q.Despues != "" && !regexp.MustCompile(`^dietas:borrador:[A-Za-z0-9_-]{16,128}$`).MatchString(q.Despues)) {
		return dietasports.PaginaBorradoresPropios{}, domain.ErrBorradorComisionInvalido
	}
	if _, e := EmpleadoBorradorPropio(a); e != nil {
		return dietasports.PaginaBorradoresPropios{}, e
	}
	p, e := s.unidad.ListarBorradoresPropios(c, a, q)
	if e != nil {
		return dietasports.PaginaBorradoresPropios{}, e
	}
	if len(p.Borradores) > q.Limite {
		return dietasports.PaginaBorradoresPropios{}, ErrComposicionBorradorInvalida
	}
	previo := q.Despues
	for i, item := range p.Borradores {
		if item.Borrador.ValidarResumen() != nil || item.Borrador.PersonaRef != a.PersonaRef || !reciboValido(item.Recibo) || item.Recibo.Version != 1 || item.Recibo.ComisionRef <= previo {
			return dietasports.PaginaBorradoresPropios{}, ErrComposicionBorradorInvalida
		}
		previo = item.Recibo.ComisionRef
		p.Borradores[i].Borrador = item.Borrador.Clonar()
	}
	if p.Siguiente != "" && (len(p.Borradores) != q.Limite || p.Siguiente != previo) {
		return dietasports.PaginaBorradoresPropios{}, ErrComposicionBorradorInvalida
	}
	return p, nil
}

// PoliticaActual es una proyección informativa para el formulario. Su ausencia
// no impide recuperar borradores guardados; cada creación la resuelve de nuevo
// para la fecha civil de la comisión, nunca utiliza este resultado como permiso.
func (s *ServicioBorradorComision) PoliticaActual(c context.Context, a vecdomain.ContextoActor) (domain.PoliticaKilometraje, error) {
	if s == nil || c == nil || a.Validar() != nil || dependenciaNula(s.politica) || s.zona == nil {
		return domain.PoliticaKilometraje{}, dietasports.ErrPoliticaBorradorNoDisponible
	}
	if _, e := EmpleadoBorradorPropio(a); e != nil {
		return domain.PoliticaKilometraje{}, e
	}
	p, e := s.politica.ResolverPoliticaKilometraje(c, dietasports.SolicitudPoliticaKilometraje{ContextoActor: a, Inicio: time.Now().UTC(), VehiculoPropio: true})
	if e != nil || p.Validar() != nil {
		return domain.PoliticaKilometraje{}, dietasports.ErrPoliticaBorradorNoDisponible
	}
	return p, nil
}
func (s *ServicioBorradorComision) RecuperarPropio(c context.Context, a vecdomain.ContextoActor, ref string) (domain.BorradorComision, dietasports.ReciboBorradorComision, error) {
	if s == nil || unidadNula(s.unidad) || c == nil || a.Validar() != nil || !referenciaBorrador.MatchString(ref) {
		return domain.BorradorComision{}, dietasports.ReciboBorradorComision{}, domain.ErrBorradorComisionInvalido
	}
	if _, err := EmpleadoBorradorPropio(a); err != nil {
		return domain.BorradorComision{}, dietasports.ReciboBorradorComision{}, err
	}
	b, r, e := s.unidad.RecuperarBorradorPropio(c, a, ref)
	if e != nil {
		return domain.BorradorComision{}, dietasports.ReciboBorradorComision{}, e
	}
	if b.Validar() != nil || b.PersonaRef != a.PersonaRef || !reciboValido(r) || r.ComisionRef != ref || r.Version != 1 {
		return domain.BorradorComision{}, dietasports.ReciboBorradorComision{}, ErrComposicionBorradorInvalida
	}
	return b, r, nil
}

// EmpleadoBorradorPropio exige el vínculo canónico del servidor. La titularidad
// restringe la operación; nunca sustituye su concesión nominal V3.
func EmpleadoBorradorPropio(a vecdomain.ContextoActor) (string, error) {
	refs, err := a.Referencias(vecdomain.TipoReferenciaContextoActorEmpleado)
	if err != nil || len(refs) != 1 {
		return "", dietasports.ErrAccesoBorradorDenegado
	}
	return refs[0], nil
}
func reciboValido(r dietasports.ReciboBorradorComision) bool {
	return referenciaBorrador.MatchString(r.ComisionRef) && referenciaBorrador.MatchString(r.ReciboRef) && referenciaBorrador.MatchString(r.CorrelacionRef) && r.Version > 0 && !r.RegistradoEn.IsZero() && r.RegistradoEn.Location() == time.UTC
}

func unidadNula(u dietasports.UnidadTrabajoBorradorComision) bool {
	return dependenciaNula(u)
}

func dependenciaNula(u any) bool {
	if u == nil {
		return true
	}
	v := reflect.ValueOf(u)
	switch v.Kind() {
	case reflect.Pointer, reflect.Interface, reflect.Map, reflect.Slice, reflect.Func, reflect.Chan:
		return v.IsNil()
	}
	return false
}
