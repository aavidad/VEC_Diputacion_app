// Package personalincorporacion adapta el puerto CT de acreditación a la
// lectura nominal propietaria de Personal. No concede permisos ni implementa
// una transacción compuesta CT.
package personalincorporacion

import (
	"bytes"
	"context"
	"reflect"
	"time"

	ct "vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	lector "vec-diputacion-granada/internal/modules/personal/adapters/lecturaincorporacion"
)

// LectorNominal es la única superficie de Personal que necesita este puente.
// El lector obtiene su propia autorización V3; la exportación CT no se emplea
// como capacidad de lectura.
type LectorNominal interface {
	Leer(context.Context, lector.Selector, ct.ContextoAutorizacionAltaV3) (lector.Resultado, error)
}

type Reloj interface{ Ahora() time.Time }

type Puente struct {
	lector LectorNominal
	reloj  Reloj
}

func Nuevo(lectorPersonal LectorNominal, reloj Reloj) (*Puente, error) {
	if nilTipado(lectorPersonal) || nilTipado(reloj) {
		return nil, ct.ErrAcreditacionPersonalIncorporacion
	}
	return &Puente{lector: lectorPersonal, reloj: reloj}, nil
}

// AcreditarRegistroPersonal satisface ports.AcreditadorPersonalIncorporacion.
// La fuente de verdad es siempre el resultado del lector propietario; el
// registro incorporado en el material CT solo fija el cotejo esperado.
func (p *Puente) AcreditarRegistroPersonal(ctx context.Context, orden ct.OrdenConfirmacionIncorporacionV2) (ct.RegistroPersonalEjercicio, error) {
	var cero ct.RegistroPersonalEjercicio
	if p == nil || ctx == nil || nilTipado(p.lector) || nilTipado(p.reloj) {
		return cero, ct.ErrAcreditacionPersonalIncorporacion
	}
	if err := ctx.Err(); err != nil {
		return cero, err
	}
	antes := p.reloj.Ahora()
	if orden.ValidarEn(antes) != nil {
		return cero, ct.ErrAcreditacionPersonalIncorporacion
	}
	datos, err := orden.Material().Datos()
	if err != nil {
		return cero, ct.ErrAcreditacionPersonalIncorporacion
	}
	expectativa := copiarRegistro(datos.Personal)
	selector := lector.Selector{
		OrganizacionRef:   datos.Preparacion.OrganizacionRef,
		SolicitudRef:      expectativa.Solicitud.SolicitudRef,
		ExpedienteRef:     expectativa.Solicitud.ExpedienteRef,
		VersionExpediente: expectativa.Solicitud.VersionExpediente,
		ResultadoRef:      expectativa.Resultado.ResultadoRef,
		ReciboRef:         expectativa.Resultado.ReciboRef,
		RelacionRef:       expectativa.Resultado.RelacionRef,
		OcupacionRef:      expectativa.Resultado.OcupacionRef,
		MaterialSHA256:    expectativa.MaterialSHA256,
	}
	resultado, err := p.lector.Leer(ctx, selector, datos.Contexto)
	if ctxErr := ctx.Err(); ctxErr != nil {
		return cero, ctxErr
	}
	if err != nil {
		return cero, ct.ErrAcreditacionPersonalIncorporacion
	}
	despues := p.reloj.Ahora()
	if despues.Before(antes) || orden.ValidarEn(despues) != nil ||
		resultado.Registro.ValidarEstructuraPara(expectativa.Solicitud, despues) != nil ||
		!mismoRegistro(resultado.Registro, expectativa) {
		return cero, ct.ErrAcreditacionPersonalIncorporacion
	}
	return copiarRegistro(resultado.Registro), nil
}

func mismoRegistro(a, b ct.RegistroPersonalEjercicio) bool {
	return a.Solicitud == b.Solicitud && a.Resultado == b.Resultado &&
		bytes.Equal(a.MaterialCanonico, b.MaterialCanonico) &&
		a.MaterialSHA256 == b.MaterialSHA256 && a.RegistradoEn.Equal(b.RegistradoEn) &&
		a.DecisionOriginalRef == b.DecisionOriginalRef && a.AuditoriaRef == b.AuditoriaRef &&
		a.OutboxRef == b.OutboxRef && a.EjercicioSintetico == b.EjercicioSintetico &&
		a.FirmaOficial == b.FirmaOficial && a.EficaciaAdministrativa == b.EficaciaAdministrativa
}

func copiarRegistro(r ct.RegistroPersonalEjercicio) ct.RegistroPersonalEjercicio {
	r.MaterialCanonico = bytes.Clone(r.MaterialCanonico)
	return r
}

func nilTipado(v any) bool {
	if v == nil {
		return true
	}
	x := reflect.ValueOf(v)
	return (x.Kind() == reflect.Pointer || x.Kind() == reflect.Interface) && x.IsNil()
}

var _ ct.AcreditadorPersonalIncorporacion = (*Puente)(nil)
