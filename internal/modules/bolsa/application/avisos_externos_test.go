package application

import (
	"context"
	"errors"
	"testing"
	"time"

	"vec-diputacion-granada/internal/modules/bolsa/ports"
	seguridad "vec-diputacion-granada/internal/vec/adapters/seguridad"
	core "vec-diputacion-granada/internal/vec/domain"
)

type fuenteDestinatarioPrueba struct {
	candidato string
	err       error
}

func (f fuenteDestinatarioPrueba) DestinatarioExterno(context.Context, string, string) (string, error) {
	return f.candidato, f.err
}

type reservaAvisoPrueba struct{ previa ports.EmisionLlamamiento }

func (r reservaAvisoPrueba) Reservar(context.Context, ports.ComandoEmitirLlamamiento) (ports.EmisionLlamamiento, error) {
	return r.previa, nil
}
func (r reservaAvisoPrueba) RegistrarContactos(context.Context, string, string, string, []byte, []ports.ResultadoContactoEmision) (ports.EmisionLlamamiento, error) {
	return r.previa, nil
}
func (r reservaAvisoPrueba) Recuperar(context.Context, string, string) (ports.EmisionLlamamiento, error) {
	return r.previa, nil
}

func solicitudAvisoExterno(t *testing.T) ports.SolicitudEmitirLlamamiento {
	t.Helper()
	correlacion, err := core.GenerarReferenciaCorrelacionAutorizacionV2(context.Background(), seguridad.GeneradorReferenciasCriptograficas{})
	if err != nil {
		t.Fatal(err)
	}
	return ports.SolicitudEmitirLlamamiento{BolsaRef: "bolsa:sintetica", ClaveIdempotencia: "clave:sintetica", Participaciones: []string{"participacion:sintetica"}, Correlacion: correlacion, Configuracion: ports.ConfiguracionLlamamiento{PlantillaVersion: "bolsa-llamamiento-v1"}}
}
func TestAvisosExternosNoConvierteAusenciaEnCorreoDelAlta(t *testing.T) {
	for _, f := range []fuenteDestinatarioPrueba{{}, {err: errors.New("caida")}} {
		q := solicitudAvisoExterno(t)
		s := &ServicioEmisionLlamamiento{repositorio: reservaAvisoPrueba{}, avisosExternos: &configuracionAvisosExternos{fuente: f, productor: "productor:sintetico"}}
		eventos, _, err := s.prepararAvisosExternos(context.Background(), q, "llamamiento:sintetico", time.Now())
		if err == nil || len(eventos) != 0 {
			t.Fatal("la ausencia o caída de la autoridad externa debe impedir todo aviso")
		}
	}
}
func TestAvisosExternosConservaEventoOriginalAlRecuperar(t *testing.T) {
	q := solicitudAvisoExterno(t)
	original := ports.EventoAvisoExterno{EventoRef: "evento_aviso:original", ProductorRef: "productor:sintetico", OcurridoEn: "2026-09-29T10:00:00.000000Z", CorrelacionRef: "correlacion:original", DestinatarioExternoRef: "can_abcdefghijklmnopqrstuvwxyz"}
	previa := ports.EmisionLlamamiento{BolsaRef: q.BolsaRef, Participaciones: q.Participaciones, Configuracion: q.Configuracion, AvisosExternos: []ports.AvisoExternoPendiente{{Evento: original}}}
	s := &ServicioEmisionLlamamiento{repositorio: reservaAvisoPrueba{previa}, avisosExternos: &configuracionAvisosExternos{fuente: fuenteDestinatarioPrueba{candidato: original.DestinatarioExternoRef}, productor: original.ProductorRef}}
	eventos, externos, err := s.prepararAvisosExternos(context.Background(), q, "llamamiento:sintetico", time.Now())
	if err != nil || len(eventos) != 1 || eventos[0] != original || !externos[q.Participaciones[0]] {
		t.Fatalf("replay divergente: %v", err)
	}
}
func TestAvisosExternosRechazaMezclaConLectorDeDirecciones(t *testing.T) {
	avisador := servicioAvisosPrueba(&emisorAnotadoPrueba{}, correoAltaPrueba{})
	if err := avisador.EstablecerCorreoAvisosPersona(candidatosAvisosPrueba{}, &fuenteAvisosPrueba{}); err != nil {
		t.Fatal(err)
	}
	s := &ServicioEmisionLlamamiento{avisador: avisador}
	if s.EstablecerAvisosExternos(fuenteDestinatarioPrueba{candidato: "can_abcdefghijklmnopqrstuvwxyz"}, "productor:sintetico") == nil {
		t.Fatal("no se pueden mezclar lectores de dirección y outbox externo")
	}
}
