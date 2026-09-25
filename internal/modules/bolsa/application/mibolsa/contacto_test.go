package mibolsa

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	bolsa "vec-diputacion-granada/internal/modules/bolsa/ports"
	"vec-diputacion-granada/internal/vec/domain"
)

type registroContactoPrueba struct {
	confirmaciones []bolsa.ConfirmacionContactoPortal
}

func (r *registroContactoPrueba) ConfirmarContacto(_ context.Context, c bolsa.ConfirmacionContactoPortal) (bolsa.ReciboConfirmacionContacto, error) {
	r.confirmaciones = append(r.confirmaciones, c)
	return bolsa.ReciboConfirmacionContacto{ReciboRef: c.ReciboRef, Version: c.Version, ConfirmadaEn: c.ConfirmadaEn}, nil
}

func nuevoEntornoContacto(t *testing.T, audiencia string) (*entornoPortal, *Portal, *registroContactoPrueba) {
	t.Helper()
	ahora := time.Date(2026, 7, 15, 8, 0, 0, 0, time.UTC)
	p := nuevoEntornoPortal(t, bolsa.AccionConfirmarContactoPropio, audiencia, reglasPortalPrueba{modo: "firme", maxima: ahora.AddDate(1, 0, 0)})
	registro := new(registroContactoPrueba)
	portal, err := p.portal.ConRegistroContacto(registro)
	exigir(t, err)
	return p, portal, registro
}

func TestConfirmarContactoActuaSobreLaBolsaPropia(t *testing.T) {
	p, portal, registro := nuevoEntornoContacto(t, bolsa.AudienciaConfirmarContactoPropio)
	recibo, err := portal.ConfirmarContacto(context.Background(), p.orden, "bolsa:auxiliar", 3, "clave-contacto-01")
	exigir(t, err)
	if len(registro.confirmaciones) != 1 || p.firmas != 1 || p.proveedor.acciones[0] != bolsa.AccionConfirmarContactoPropio {
		t.Fatal("no recorrió una sola cadena completa")
	}
	c := registro.confirmaciones[0]
	if c.Version != 3 || c.Bolsa != "bolsa:auxiliar" || c.Material.ValidarEstructura() != nil || !strings.HasPrefix(c.ReciboRef, "recibo:confirmacion-contacto:") || recibo.ReciboRef != c.ReciboRef {
		t.Fatalf("confirmación inexacta: %+v", c)
	}
	datos, err := p.concesiones.orden.Datos()
	exigir(t, err)
	nominal, err := datos.Solicitud.Datos()
	exigir(t, err)
	if nominal.Accion != bolsa.AccionConfirmarContactoPropio || nominal.Recurso.Referencia != "mi-bolsa:"+c.CandidatoRef || nominal.Recurso.Atributos["bolsa_ref"] != "bolsa:auxiliar" {
		t.Fatal("decisión sobre otro recurso o acción")
	}
}

func TestConfirmarContactoRechazaAntesDeAutorizar(t *testing.T) {
	p, portal, registro := nuevoEntornoContacto(t, bolsa.AudienciaConfirmarContactoPropio)
	for _, caso := range []struct {
		bolsa   string
		version int64
		clave   string
	}{{"", 1, "clave-contacto-01"}, {"bolsa:auxiliar", 0, "clave-contacto-01"}, {"bolsa:auxiliar", 1, "corta"}} {
		if _, err := portal.ConfirmarContacto(context.Background(), p.orden, caso.bolsa, caso.version, caso.clave); !errors.Is(err, bolsa.ErrPortalCandidatoInvalido) {
			t.Fatalf("%+v aceptado: %v", caso, err)
		}
	}
	if _, err := p.portal.ConfirmarContacto(context.Background(), p.orden, "bolsa:auxiliar", 1, "clave-contacto-01"); !errors.Is(err, bolsa.ErrPortalCandidatoNoDisponible) {
		t.Fatalf("portal sin registro de contacto: %v", err)
	}
	if p.firmas != 0 || len(registro.confirmaciones) != 0 {
		t.Fatal("se autorizó o escribió una petición rechazada")
	}
}

func TestConfirmarContactoNoAceptaMaterialDeOtraAudiencia(t *testing.T) {
	p, portal, registro := nuevoEntornoContacto(t, bolsa.AudienciaResponderLlamamientoPropio)
	if _, err := portal.ConfirmarContacto(context.Background(), p.orden, "bolsa:auxiliar", 1, "clave-contacto-01"); !errors.Is(err, domain.ErrAutorizacionDenegada) {
		t.Fatalf("material de otra audiencia: %v", err)
	}
	if len(registro.confirmaciones) != 0 {
		t.Fatal("escritura con material cruzado")
	}
}
