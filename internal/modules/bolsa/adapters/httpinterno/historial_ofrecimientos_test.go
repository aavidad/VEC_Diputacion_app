package httpinterno

import (
	"context"
	"strings"
	"testing"

	dominiobolsa "vec-diputacion-granada/internal/modules/bolsa/domain"
	puertosbolsa "vec-diputacion-granada/internal/modules/bolsa/ports"
)

type preparadorHistorialOfertaPrueba struct {
	preparadorIntentosPrueba
	entrada EntradaRegistrarContactoParticipacion
}

func (p *preparadorHistorialOfertaPrueba) PrepararSolicitudRegistrarContacto(_ context.Context, e EntradaRegistrarContactoParticipacion) (puertosbolsa.SolicitudRegistrarContactoParticipacion, error) {
	p.entrada = e
	return puertosbolsa.SolicitudRegistrarContactoParticipacion{BolsaRef: e.BolsaRef, ParticipacionRef: e.ParticipacionRef, OfertaRef: e.OfertaRef, EvidenciaRef: e.EvidenciaRef, EvidenciaHuellaSHA256: e.EvidenciaHuellaSHA256, Canal: e.Canal, Resultado: e.Resultado, Anotacion: e.Anotacion, Instante: e.Instante}, nil
}
func (*preparadorHistorialOfertaPrueba) PrepararConsultaContactosBolsa(_ context.Context, bolsa, cursor string, limite int) (puertosbolsa.ConsultaContactosBolsa, error) {
	return puertosbolsa.ConsultaContactosBolsa{BolsaRef: bolsa, Cursor: cursor, Limite: limite}, nil
}

type operadorHistorialOfertaPrueba struct {
	operadorIntentosPrueba
	consulta puertosbolsa.ConsultaContactosBolsa
	registro puertosbolsa.SolicitudRegistrarContactoParticipacion
}

func (o *operadorHistorialOfertaPrueba) ListarContactosBolsa(_ context.Context, q puertosbolsa.ConsultaContactosBolsa) (puertosbolsa.PaginaContactosParticipacion, error) {
	o.consulta = q
	return puertosbolsa.PaginaContactosParticipacion{Contactos: []dominiobolsa.ContactoParticipacion{{ContactoRef: "contacto:1", BolsaRef: q.BolsaRef, ParticipacionRef: "participacion:1", OfertaRef: q.OfertaRef, Canal: "correo", Resultado: "enviado"}}}, nil
}
func (o *operadorHistorialOfertaPrueba) RegistrarContactoParticipacion(_ context.Context, s puertosbolsa.SolicitudRegistrarContactoParticipacion) (puertosbolsa.RegistroContactoParticipacion, error) {
	o.registro = s
	return puertosbolsa.RegistroContactoParticipacion{Contacto: dominiobolsa.ContactoParticipacion{ContactoRef: "contacto:1", OfertaRef: s.OfertaRef, Canal: s.Canal, Resultado: s.Resultado, EvidenciaRef: s.EvidenciaRef, EvidenciaHuellaSHA256: s.EvidenciaHuellaSHA256, Instante: s.Instante}, ReciboRef: "recibo:1"}, nil
}

func TestHistorialOfertaFiltraYConservaDeclaracionExpresa(t *testing.T) {
	ref := "oferta:" + strings.Repeat("a", 64)
	p := &preparadorHistorialOfertaPrueba{}
	o := &operadorHistorialOfertaPrueba{}
	h, err := NuevoHandlerContactoParticipacion(p, o)
	if err != nil {
		t.Fatal(err)
	}
	estado, cuerpo := pedirContactos(t, h, "GET", RutaContactosOferta+"?bolsa_ref=bolsa:1&oferta_ref="+ref, "")
	if estado != 200 || o.consulta.BolsaRef != "bolsa:1" || o.consulta.OfertaRef != ref {
		t.Fatalf("consulta: %d %#v", estado, o.consulta)
	}
	items := cuerpo["data"].(map[string]any)["contactos"].([]any)
	if items[0].(map[string]any)["oferta_ref"] != ref || items[0].(map[string]any)["resultado"] != "enviado" {
		t.Fatalf("historia: %#v", items[0])
	}
	estado, _ = pedirContactos(t, h, "GET", RutaContactosOferta+"?bolsa_ref=bolsa:2&oferta_ref="+ref+"&desconocido=1", "")
	if estado != 400 {
		t.Fatalf("filtro desconocido: %d", estado)
	}
	cuerpoPOST := `{"canal":"correo","resultado":"entrega_declarada","instante":"2026-10-02T11:00:00Z","anotacion":"RRHH registró el acuse","oferta_ref":"` + ref + `","evidencia_ref":"acuse:prueba_sintetica","evidencia_huella_sha256":"` + strings.Repeat("b", 64) + `"}`
	estado, cuerpo = pedirContactos(t, h, "POST", rutaContactosPrueba, cuerpoPOST)
	if estado != 201 || p.entrada.OfertaRef != ref || o.registro.EvidenciaRef != "acuse:prueba_sintetica" || cuerpo["data"].(map[string]any)["recibo_ref"] != "recibo:1" {
		t.Fatalf("registro: %d %#v", estado, cuerpo)
	}
}
