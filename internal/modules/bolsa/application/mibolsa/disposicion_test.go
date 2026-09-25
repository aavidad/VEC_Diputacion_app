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

type registroOfertasPrueba struct {
	disposiciones []bolsa.DisposicionPortalCandidato
}

func (r *registroOfertasPrueba) ManifestarDisposicion(_ context.Context, d bolsa.DisposicionPortalCandidato) (bolsa.ReciboDisposicionPortal, error) {
	r.disposiciones = append(r.disposiciones, d)
	return bolsa.ReciboDisposicionPortal{OfertaRef: d.OfertaRef, ReciboRef: d.ReciboRef, ManifestadaEn: d.ManifestadaEn}, nil
}

func nuevoEntornoDisposicion(t *testing.T, audiencia string) (*entornoPortal, *Portal, *registroOfertasPrueba) {
	t.Helper()
	ahora := time.Date(2026, 7, 15, 8, 0, 0, 0, time.UTC)
	p := nuevoEntornoPortal(t, bolsa.AccionManifestarDisposicionPropia, audiencia, reglasPortalPrueba{modo: "firme", maxima: ahora.AddDate(1, 0, 0)})
	p.fuente.instantanea.VersionRol.Concesiones[0].TipoRecurso = bolsa.TipoRecursoOfertaBolsa
	registro := new(registroOfertasPrueba)
	portal, err := p.portal.ConRegistroOfertas(registro)
	exigir(t, err)
	return p, portal, registro
}

func TestDisposicionActuaSobreLaOfertaConLaAccionPropia(t *testing.T) {
	p, portal, registro := nuevoEntornoDisposicion(t, bolsa.AudienciaManifestarDisposicionPropia)
	oferta := "oferta:" + strings.Repeat("a", 64)
	recibo, err := portal.ManifestarDisposicion(context.Background(), p.orden, oferta, "clave-disposicion-1")
	exigir(t, err)
	if len(registro.disposiciones) != 1 || p.firmas != 1 || len(p.proveedor.acciones) != 1 || p.proveedor.acciones[0] != bolsa.AccionManifestarDisposicionPropia {
		t.Fatal("no recorrió una sola cadena completa")
	}
	d := registro.disposiciones[0]
	if d.OfertaRef != oferta || d.CandidatoRef != referenciaServicioContextoActorPrueba("can_", "c") || d.Material.ValidarEstructura() != nil ||
		!strings.HasPrefix(d.ReciboRef, "recibo:disposicion:") || len(d.ReciboRef) != len("recibo:disposicion:")+64 || recibo.ReciboRef != d.ReciboRef {
		t.Fatalf("disposición inexacta: %+v", d)
	}
	datos, err := p.concesiones.orden.Datos()
	exigir(t, err)
	nominal, err := datos.Solicitud.Datos()
	exigir(t, err)
	if nominal.Accion != bolsa.AccionManifestarDisposicionPropia || nominal.Recurso.Referencia != oferta ||
		nominal.Recurso.Tipo != bolsa.TipoRecursoOfertaBolsa || nominal.Recurso.Ambitos["candidato_ref"] != d.CandidatoRef {
		t.Fatal("decisión sobre otro recurso o acción")
	}
	// Misma clave, mismas referencias: el replay lo resuelve PostgreSQL.
	_, err = portal.ManifestarDisposicion(context.Background(), p.orden, oferta, "clave-disposicion-1")
	exigir(t, err)
	if registro.disposiciones[1].ReciboRef != d.ReciboRef {
		t.Fatal("la repetición cambia el recibo")
	}
}

func TestDisposicionRechazaAntesDeAutorizarYSinRegistro(t *testing.T) {
	p, portal, registro := nuevoEntornoDisposicion(t, bolsa.AudienciaManifestarDisposicionPropia)
	for _, caso := range [][2]string{{"oferta:corta", "clave-disposicion-1"}, {"mi-bolsa:x", "clave-disposicion-1"}, {"oferta:" + strings.Repeat("a", 64), "corta"}} {
		if _, err := portal.ManifestarDisposicion(context.Background(), p.orden, caso[0], caso[1]); !errors.Is(err, bolsa.ErrPortalCandidatoInvalido) {
			t.Fatalf("%v aceptado: %v", caso, err)
		}
	}
	// Sin registro de ofertas compuesto no se atiende.
	if _, err := p.portal.ManifestarDisposicion(context.Background(), p.orden, "oferta:"+strings.Repeat("a", 64), "clave-disposicion-1"); !errors.Is(err, bolsa.ErrPortalCandidatoNoDisponible) {
		t.Fatalf("portal sin ofertas: %v", err)
	}
	if p.firmas != 0 || len(registro.disposiciones) != 0 {
		t.Fatal("se autorizó o escribió una petición rechazada")
	}
}

func TestDisposicionNoAceptaMaterialDeOtraAudiencia(t *testing.T) {
	p, portal, registro := nuevoEntornoDisposicion(t, bolsa.AudienciaResponderLlamamientoPropio)
	if _, err := portal.ManifestarDisposicion(context.Background(), p.orden, "oferta:"+strings.Repeat("a", 64), "clave-disposicion-1"); !errors.Is(err, domain.ErrAutorizacionDenegada) {
		t.Fatalf("material de otra audiencia: %v", err)
	}
	if len(registro.disposiciones) != 0 {
		t.Fatal("escritura con material cruzado")
	}
}
