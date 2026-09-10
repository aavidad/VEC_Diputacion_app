package personalincorporacion

import (
	"context"
	"reflect"

	ct "vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	lector "vec-diputacion-granada/internal/modules/personal/adapters/lecturaincorporacion"
)

// LectorNominalV2 obtiene autorización lectora independiente para organización
// y unidad. Los tipos Personal quedan confinados a esta frontera anticorrupción.
type LectorNominalV2 interface {
	Leer(context.Context, lector.Selector, string, ct.ContextoAutorizacionAltaV3) (lector.Resultado, error)
}

// PuenteV2 no consume permisos ni acredita por sí mismo commit propietario.
type PuenteV2 struct {
	lector LectorNominalV2
	reloj  Reloj
}

func dependenciaPuenteV2Nula(v any) bool {
	if nilTipado(v) {
		return true
	}
	switch reflect.ValueOf(v).Kind() {
	case reflect.Func, reflect.Map, reflect.Slice, reflect.Chan:
		return reflect.ValueOf(v).IsNil()
	}
	return false
}

func NuevoV2(l LectorNominalV2, reloj Reloj) (*PuenteV2, error) {
	if dependenciaPuenteV2Nula(l) || dependenciaPuenteV2Nula(reloj) {
		return nil, ct.ErrAcreditacionPersonalIncorporacion
	}
	return &PuenteV2{lector: l, reloj: reloj}, nil
}

func (p *PuenteV2) AcreditarRegistroPersonal(ctx context.Context, orden ct.OrdenConfirmacionIncorporacionV2) (ct.RegistroPersonalEjercicio, error) {
	var cero ct.RegistroPersonalEjercicio
	if p == nil || ctx == nil || dependenciaPuenteV2Nula(p.lector) || dependenciaPuenteV2Nula(p.reloj) {
		return cero, ct.ErrAcreditacionPersonalIncorporacion
	}
	if err := ctx.Err(); err != nil {
		return cero, err
	}
	antes := p.reloj.Ahora()
	if err := ctx.Err(); err != nil {
		return cero, err
	}
	if orden.ValidarEn(antes) != nil {
		return cero, ct.ErrAcreditacionPersonalIncorporacion
	}
	datos, err := orden.Material().Datos()
	if err != nil {
		return cero, ct.ErrAcreditacionPersonalIncorporacion
	}
	esperado := copiarRegistro(datos.Personal)
	selector := lector.Selector{
		OrganizacionRef: datos.Preparacion.OrganizacionRef,
		SolicitudRef:    esperado.Solicitud.SolicitudRef, ExpedienteRef: esperado.Solicitud.ExpedienteRef,
		VersionExpediente: esperado.Solicitud.VersionExpediente, ResultadoRef: esperado.Resultado.ResultadoRef,
		ReciboRef: esperado.Resultado.ReciboRef, RelacionRef: esperado.Resultado.RelacionRef,
		OcupacionRef: esperado.Resultado.OcupacionRef, MaterialSHA256: esperado.MaterialSHA256,
	}
	// Valida la unidad obligatoria con el contrato propietario, sin valor exterior
	// ni fallback V1. El constructor de material no concede autorización.
	if _, err = lector.NuevoMaterialV2(selector, datos.Preparacion.UnidadRef, datos.Contexto, antes); err != nil {
		return cero, ct.ErrAcreditacionPersonalIncorporacion
	}
	if err = ctx.Err(); err != nil {
		return cero, err
	}
	r, err := p.lector.Leer(ctx, selector, datos.Preparacion.UnidadRef, datos.Contexto)
	if e := ctx.Err(); e != nil {
		return cero, e
	}
	if err != nil {
		return cero, ct.ErrAcreditacionPersonalIncorporacion
	}
	despues := p.reloj.Ahora()
	if err = ctx.Err(); err != nil {
		return cero, err
	}
	if despues.Before(antes) || orden.ValidarEn(despues) != nil ||
		r.Registro.ValidarEstructuraPara(esperado.Solicitud, despues) != nil || !mismoRegistro(r.Registro, esperado) {
		return cero, ct.ErrAcreditacionPersonalIncorporacion
	}
	return copiarRegistro(r.Registro), nil
}

var _ ct.AcreditadorPersonalIncorporacion = (*PuenteV2)(nil)
