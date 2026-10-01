package application

import (
	"context"
	"reflect"
	"strings"
	"time"

	cal "vec-diputacion-granada/internal/modules/calendarios/domain"
	"vec-diputacion-granada/internal/modules/cronos/domain"
	"vec-diputacion-granada/internal/modules/cronos/ports"
)

// EnsayarIncidencias reads one explicitly synthetic, versioned snapshot. It
// does not authenticate a person or connect this query to the runtime.
func EnsayarIncidencias(ctx context.Context, l ports.LectorEnsayoIncidencias) (ports.ResultadoEnsayoIncidencias, error) {
	var vacio ports.ResultadoEnsayoIncidencias
	if ctx == nil || l == nil {
		return vacio, domain.ErrIncidenciasPeriodoInvalido
	}
	v := reflect.ValueOf(l)
	if (v.Kind() == reflect.Pointer || v.Kind() == reflect.Map || v.Kind() == reflect.Slice || v.Kind() == reflect.Func || v.Kind() == reflect.Chan || v.Kind() == reflect.Interface) && v.IsNil() {
		return vacio, domain.ErrIncidenciasPeriodoInvalido
	}
	if err := ctx.Err(); err != nil {
		return vacio, err
	}
	s, err := l.LeerSnapshotEnsayoIncidencias(ctx)
	if err != nil {
		return vacio, err
	}
	if err = ctx.Err(); err != nil {
		return vacio, err
	}
	if !strings.HasPrefix(s.Ref, "demo:") || !strings.HasPrefix(s.PersonaRef, "demo:") || !strings.HasPrefix(s.Fuente.Referencia, "demo:") || s.VersionEsquema != 1 || !s.Demostracion || !cal.ReferenciaValida(s.Ref) || s.Version < 1 || !cal.ReferenciaValida(s.PersonaRef) || !cal.ReferenciaValida(s.Fuente.Referencia) || s.Fuente.Version < 1 || !domain.HuellaEfectosValida(s.Fuente.SHA256) || !domain.HuellaEfectosValida(s.SHA256) {
		return vacio, domain.ErrIncidenciasPeriodoInvalido
	}
	for _, h := range s.Consulta.Hechos {
		if !strings.HasPrefix(h.Ref, "demo:") {
			return vacio, domain.ErrIncidenciasPeriodoInvalido
		}
	}
	dias, err := domain.ConsultarIncidenciasPeriodo(s.Consulta)
	if err != nil {
		return vacio, err
	}
	if err = ctx.Err(); err != nil {
		return vacio, err
	}
	return ports.ResultadoEnsayoIncidencias{Demostracion: true, SnapshotRef: s.Ref, SnapshotVersion: s.Version, SnapshotSHA256: s.SHA256, PersonaRef: s.PersonaRef, Fuente: s.Fuente, Desde: s.Consulta.Desde, Hasta: s.Consulta.Hasta, Zona: s.Consulta.Zona, CorteUTC: s.Consulta.CorteUTC.Format(time.RFC3339Nano), AntecedentesCompletos: s.Consulta.AntecedentesCompletos, Dias: dias}, nil
}
