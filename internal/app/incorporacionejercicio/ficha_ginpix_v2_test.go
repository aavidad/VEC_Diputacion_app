package incorporacionejercicio

import (
	"context"
	"errors"
	"testing"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

type consultaFichaPrueba struct {
	p   ports.ProyeccionIncorporacionAplicacionV2
	err error
}

func (x consultaFichaPrueba) RecuperarFichaGINPIXV2(context.Context, string) (RecuperacionFichaGINPIXV2, error) {
	if x.err != nil {
		return RecuperacionFichaGINPIXV2{}, x.err
	}
	if x.p.Recibo == nil {
		return RecuperacionFichaGINPIXV2{}, nil
	}
	return RecuperacionFichaGINPIXV2{Recibo: *x.p.Recibo, ProcedenciaModeloRef: "procedencia:descarga", CorrelacionRef: "correlacion:original", IdempotenciaRef: "idempotencia:original"}, nil
}

type mapeoFichaPrueba struct {
	m   domain.MapeoVersionadoGINPIX
	err error
}

func (x mapeoFichaPrueba) ResolverMapeoFichaGINPIXV2(context.Context, ports.ReciboIncorporacionAplicacionV2) (domain.MapeoVersionadoGINPIX, error) {
	return x.m, x.err
}

func reciboFichaPrueba() ports.ReciboIncorporacionAplicacionV2 {
	return ports.ReciboIncorporacionAplicacionV2{Esquema: "vec.contratacion-temporal.incorporacion-ejercicio.recibo.v2", ExpedienteRef: "expediente:uno", SolicitudPersonalRef: "solicitud:uno", RelacionRef: "relacion:uno", ReciboRef: "recibo:uno", ActuacionRef: "actuacion:uno", SeguimientoRef: "seguimiento:uno", AuditoriaRef: "auditoria:uno", OutboxRef: "outbox:uno", VersionSolicitudPersonal: 1, VersionActualExpediente: 2, VersionSeguimientoResultante: 1, RegistradaEn: time.Date(2026, 9, 10, 10, 0, 0, 0, time.UTC), Periodo: domain.IntervaloSeguimiento{Desde: time.Date(2026, 9, 11, 0, 0, 0, 0, time.UTC), Hasta: time.Date(2026, 9, 12, 0, 0, 0, 0, time.UTC)}, EjercicioSintetico: true}
}
func mapeoFichaPruebaValido(t *testing.T) domain.MapeoVersionadoGINPIX {
	t.Helper()
	claves := []domain.ClaveCatalogo{"actuacion_ref", "expediente_ref", "recibo_ref", "relacion_ref", "seguimiento_ref", "solicitud_personal_ref"}
	reglas := make([]domain.ReglaMapeoGINPIX, 0, len(claves))
	for _, c := range claves {
		reglas = append(reglas, domain.ReglaMapeoGINPIX{CampoCanonico: c, CampoDestino: "ginpix_" + c, Obligatorio: true})
	}
	m, e := domain.PublicarMapeoVersionadoGINPIX(domain.BorradorMapeoVersionadoGINPIX{Esquema: domain.EsquemaMapeoGINPIXV1, Referencia: "mapeo:ginpix-ejercicio", Version: 1, ProcedenciaRef: "configuracion:ginpix-ejercicio", Reglas: reglas})
	if e != nil {
		t.Fatal(e)
	}
	return m
}
func TestFichaGINPIXV2NominalAusenciaCruceYDenegacion(t *testing.T) {
	r := reciboFichaPrueba()
	m := mapeoFichaPruebaValido(t)
	f, e := NuevaFichaGINPIXV2(consultaFichaPrueba{p: ports.ProyeccionIncorporacionAplicacionV2{Esquema: "vec.contratacion-temporal.incorporacion-ejercicio.preparacion.v2", ExpedienteRef: r.ExpedienteRef, VersionActualExpediente: 2, Recibo: &r}}, mapeoFichaPrueba{m: m})
	if e != nil {
		t.Fatal(e)
	}
	o, e := f.Preparar(context.Background(), r.ExpedienteRef)
	if e != nil {
		t.Fatal(e)
	}
	b, e := o.Contenido()
	if e != nil || len(b) == 0 {
		t.Fatalf("ficha no codificada: %v", e)
	}
	for nombre, p := range map[string]ports.ProyeccionIncorporacionAplicacionV2{"ausencia": {Esquema: "vec.contratacion-temporal.incorporacion-ejercicio.preparacion.v2", ExpedienteRef: r.ExpedienteRef, VersionActualExpediente: 2}} {
		f, _ = NuevaFichaGINPIXV2(consultaFichaPrueba{p: p}, mapeoFichaPrueba{m: m})
		if _, e = f.Preparar(context.Background(), r.ExpedienteRef); !errors.Is(e, ErrFichaGINPIXV2Conflicto) {
			t.Fatalf("%s: %v", nombre, e)
		}
	}
	f, _ = NuevaFichaGINPIXV2(consultaFichaPrueba{err: ports.ErrDenegadaIncorporacionAplicacion}, mapeoFichaPrueba{m: m})
	if _, e = f.Preparar(context.Background(), r.ExpedienteRef); !errors.Is(e, ErrFichaGINPIXV2Denegada) {
		t.Fatalf("denegacion: %v", e)
	}
}
