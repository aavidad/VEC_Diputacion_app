// Package rpt consume la relación elegida en la preparación existente.
package rpt

import (
	"context"
	"reflect"
	"time"

	"vec-diputacion-granada/internal/modules/personal/domain"
	"vec-diputacion-granada/internal/modules/personal/ports"
	"vec-diputacion-granada/internal/shared/plazoarranque"
	vecdomain "vec-diputacion-granada/internal/vec/domain"
)

type LectorRelacionSeleccionadaRPT struct {
	lector   ports.LectorRelacionParaRPTV1
	intentos ports.RegistroIntentosLectorRelacionRPT
}

func NuevoLectorRelacionSeleccionadaRPT(l ports.LectorRelacionParaRPTV1, i ports.RegistroIntentosLectorRelacionRPT) (*LectorRelacionSeleccionadaRPT, error) {
	if lectorRPTNulo(l) || lectorRPTNulo(i) {
		return nil, domain.ErrLectorRelacionRPTNoDisponible
	}
	return &LectorRelacionSeleccionadaRPT{l, i}, nil
}

// ConsultarSeleccionada usa la preparación únicamente como selector histórico
// del servidor. El lector obtiene una concesión nueva para la relación concreta;
// no recibe ni convierte la ficha, la concesión o la evidencia B2 originales.
func (l *LectorRelacionSeleccionadaRPT) ConsultarSeleccionada(ctx context.Context, actor vecdomain.ContextoActor, preparacion domain.PreparacionRelacionParaRPT, organismo, seleccion string) (ports.ResultadoRelacionParaRPTV1, error) {
	var cero ports.ResultadoRelacionParaRPTV1
	if l == nil || ctx == nil || lectorRPTNulo(l.lector) || lectorRPTNulo(l.intentos) {
		return cero, domain.ErrLectorRelacionRPTNoDisponible
	}
	fallar := func(err error) (ports.ResultadoRelacionParaRPTV1, error) {
		auditCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), plazoarranque.Ampliar(2*time.Second))
		defer cancel()
		motivo := "entrada_invalida"
		if err == domain.ErrLectorRelacionRPTNoDisponible {
			motivo = "no_disponible"
		}
		if e := l.intentos.RegistrarIntentoRelacionRPT(auditCtx, ports.IntentoLectorRelacionRPT{Actor: actor, RelacionRef: seleccion, Motivo: motivo}); e != nil {
			return cero, domain.ErrLectorRelacionRPTNoDisponible
		}
		return cero, err
	}
	if err := l.intentos.VerificarRegistroRelacionRPT(ctx); err != nil {
		return fallar(domain.ErrLectorRelacionRPTNoDisponible)
	}
	if preparacion.Esquema != "vec.personal.preparacion-relacion-rpt.v1" || preparacion.Uso != "preparacion" || preparacion.Cobertura != "no_acreditada" || preparacion.EstadoRPT != "pendiente_fuente_rpt" || preparacion.Corte.Validar() != nil || !domain.ReferenciaEmpleadoValida(preparacion.EmpleadoRef) || !domain.ReferenciaRelacionValida(seleccion) || len(preparacion.Relaciones) > 200 {
		return fallar(domain.ErrLectorRelacionRPTInvalido)
	}
	var elegida domain.RelacionPreparacionParaRPT
	n := 0
	for _, r := range preparacion.Relaciones {
		if r.RelacionRef == seleccion {
			elegida = r
			n++
		}
	}
	if n != 1 || elegida.Traza.ValidarEn(preparacion.Corte) != nil {
		return fallar(domain.ErrLectorRelacionRPTInvalido)
	}
	copia, err := actor.Clonar()
	if err != nil {
		return fallar(domain.ErrLectorRelacionRPTInvalido)
	}
	consulta := ports.ConsultaRelacionParaRPTV1{Actor: copia, EmpleadoRef: preparacion.EmpleadoRef, RelacionRef: seleccion, OrganismoRef: organismo, VersionEsperada: elegida.Traza.Version, Corte: preparacion.Corte}
	return l.lector.ConsultarRelacionParaRPT(ctx, consulta)
}
func lectorRPTNulo(v any) bool {
	if v == nil {
		return true
	}
	r := reflect.ValueOf(v)
	switch r.Kind() {
	case reflect.Ptr, reflect.Interface, reflect.Func, reflect.Map, reflect.Slice:
		return r.IsNil()
	}
	return false
}
