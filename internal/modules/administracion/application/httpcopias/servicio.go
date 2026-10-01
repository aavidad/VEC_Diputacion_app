package httpcopias

import (
	"context"
	"errors"
	"reflect"
	p "vec-diputacion-granada/internal/modules/administracion/ports/httpcopias"
)

type Servicio struct {
	Autoridad      p.Autorizador
	Lecturas       p.Consultas
	Cambios        p.Cambios
	Control        p.Control
	Opciones       p.FuenteOpciones
	FuenteRevision p.FuenteRevision
}

func Ausente(v any) bool {
	if v == nil {
		return true
	}
	r := reflect.ValueOf(v)
	switch r.Kind() {
	case reflect.Pointer, reflect.Func, reflect.Interface, reflect.Map, reflect.Slice, reflect.Chan:
		return r.IsNil()
	}
	return false
}
func (s *Servicio) Autorizar(ctx context.Context, ses p.Sesion, op p.Operacion, ref string) error {
	if s == nil || Ausente(s.Autoridad) {
		return p.ErrNoDisponible
	}
	if ctx.Err() != nil {
		return p.ErrNoDisponible
	}
	return s.Autoridad.AutorizarCopias(ctx, ses, op, ref)
}

// Capacidades is only presentation. Every actual operation is authorized again.
func (s *Servicio) Capacidades(ctx context.Context, ses p.Sesion) (p.Capacidades, error) {
	var c p.Capacidades
	for _, v := range []struct {
		op  p.Operacion
		dst *bool
		dep any
	}{
		{p.Consultar, &c.Consultar, s.Lecturas}, {p.Lanzar, &c.Lanzar, s.Cambios},
		{p.ConfigurarCalendario, &c.ConfigurarCalendario, s.Cambios}, {p.ConfigurarRetencion, &c.ConfigurarRetencion, s.Cambios},
		{p.Proponer, &c.Proponer, s.Control}, {p.Ejecutar, &c.Ejecutar, s.Control},
	} {
		if Ausente(v.dep) {
			continue
		}
		err := s.Autorizar(ctx, ses, v.op, "copias")
		if err == nil {
			*v.dst = true
		} else if !errors.Is(err, p.ErrDenegado) {
			return p.Capacidades{}, err
		}
	}
	if c.Consultar {
		c.Revisar = s.revisarDisponible(ctx, ses)
	}
	return c, nil
}
