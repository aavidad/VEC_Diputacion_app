package bootstrap

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/httpinterno"
	postgresct "vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/postgres"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
)

func escenarioRespuestaRecibidaPrueba(t *testing.T) (context.Context, *proveedorRespuestaRecibidaDesarrollo, *autorizacionComunicacionDesarrolloPrueba, ports.SolicitudRegistrarRespuestaRecibida) {
	t.Helper()
	soporte, _, principal := escenarioAutorizacionCoberturaDesarrolloPrueba(t)
	e := expedientePuenteBolsaPrueba(t)
	s := ports.SolicitudRegistrarRespuestaRecibida{ClaveIdempotencia: "11111111-1111-4111-8111-111111111111",
		OrganizacionRef: e.Fiscalizado.OrganizacionRef, ExpedienteRef: e.Fiscalizado.Referencia,
		LlamamientoRef: "llamamiento:sintetico", ComunicacionRef: "comunicacion:sintetica", VersionComunicacionEsperada: 2,
		Respuesta: ports.RespuestaLlamamientoAceptada, CorreoRef: "correo:sintetico", CorreoSHA256: strings.Repeat("a", 64),
		RecibidaEn: time.Date(2026, 9, 5, 10, 0, 0, 0, time.UTC)}
	ctx := contextoRutaCoberturaDesarrolloPrueba(soporte, principal, httpinterno.RutaRegistroRespuestaRecibida)
	ctx = context.WithValue(ctx, clavePreparacionLlamamientoDesarrollo{}, preparacionLlamamientoDesarrollo{expediente: e})
	ctx = context.WithValue(ctx, claveSolicitudRespuestaRecibidaDesarrollo{}, s)
	a := &autorizacionComunicacionDesarrolloPrueba{}
	p := &proveedorRespuestaRecibidaDesarrollo{soporte: soporte, autorizador: a, reloj: relojFijoAltaContratacionTemporalDesarrollo{ahora: s.RecibidaEn.Add(time.Hour)}}
	return ctx, p, a, s
}

func TestRespuestaRecibidaDesarrolloLigaPermisoPropioYMaterialCompleto(t *testing.T) {
	ctx, _, _, s := escenarioRespuestaRecibidaPrueba(t)
	r, err := postgresct.RecursoRegistroRespuestaRecibida(s)
	if err != nil {
		t.Fatal(err)
	}
	d := dominiovec.DatosSolicitudAutorizacionLigadaV3{Finalidad: "gestionar_contratacion_temporal", ReferenciaMotivo: motivoRespuestaRecibidaDesarrollo(), Accion: postgresct.AccionRegistroRespuestaRecibida, Recurso: r}
	if !solicitudAutorizacionLlamamientoDesarrolloValida(ctx, httpinterno.RutaRegistroRespuestaRecibida, d) {
		t.Fatal("ligadura válida rechazada")
	}
	for nombre, mutar := range map[string]func(*dominiovec.DatosSolicitudAutorizacionLigadaV3){
		"aviso": func(d *dominiovec.DatosSolicitudAutorizacionLigadaV3) {
			d.Accion = postgresct.AccionRegistroComunicacionLlamamiento
		},
		"motivo aviso": func(d *dominiovec.DatosSolicitudAutorizacionLigadaV3) {
			d.ReferenciaMotivo = motivoLlamamientoDesarrollo(true)
		},
		"otro expediente": func(d *dominiovec.DatosSolicitudAutorizacionLigadaV3) { d.Recurso.Referencia += "b" },
		"otro material": func(d *dominiovec.DatosSolicitudAutorizacionLigadaV3) {
			d.Recurso.Atributos = map[string]string{"material_sha256": strings.Repeat("b", 64)}
		},
	} {
		t.Run(nombre, func(t *testing.T) {
			otro := d
			mutar(&otro)
			if solicitudAutorizacionLlamamientoDesarrolloValida(ctx, httpinterno.RutaRegistroRespuestaRecibida, otro) {
				t.Fatal("ampliación de autoridad")
			}
		})
	}
	if solicitudAutorizacionLlamamientoDesarrolloValida(ctx, httpinterno.RutaRegistroComunicacionLlamamiento, d) {
		t.Fatal("respuesta admitida por permiso de aviso")
	}
}

func TestRespuestaRecibidaDesarrolloReacreditaReplayYRechazaPeticionAjena(t *testing.T) {
	ctx, p, a, s := escenarioRespuestaRecibidaPrueba(t)
	for intento := 1; intento <= 2; intento++ {
		if _, err := p.AutorizarRegistroRespuestaRecibida(ctx, s); !errors.Is(err, ports.ErrOperacionRespuestaRecibidaDenegada) || a.llamadas != intento || a.accion != postgresct.AccionRegistroRespuestaRecibida {
			t.Fatal("identidad o replay sustituyeron AD3")
		}
	}
	s.CorreoRef += "b"
	if _, err := p.AutorizarRegistroRespuestaRecibida(ctx, s); !errors.Is(err, ports.ErrOperacionRespuestaRecibidaDenegada) || a.llamadas != 2 {
		t.Fatal("declaración ajena alcanzó autoridad")
	}
}
