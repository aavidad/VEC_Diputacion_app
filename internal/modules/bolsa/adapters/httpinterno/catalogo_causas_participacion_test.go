package httpinterno

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	puertos "vec-diputacion-granada/internal/modules/bolsa/ports"
)

type preparadorCatalogoPrueba struct {
	entrada EntradaPublicarCatalogoCausaParticipacion
}

func (p *preparadorCatalogoPrueba) PrepararPublicacion(_ context.Context, _ string, e EntradaPublicarCatalogoCausaParticipacion) (puertos.SolicitudPublicarCausaParticipacion, error) {
	p.entrada = e
	return puertos.SolicitudPublicarCausaParticipacion{}, nil
}
func (p *preparadorCatalogoPrueba) PrepararPropuesta(_ context.Context, e EntradaPublicarCatalogoCausaParticipacion) (puertos.SolicitudProponerCausaParticipacion, error) {
	p.entrada = e
	return puertos.SolicitudProponerCausaParticipacion{}, nil
}
func (*preparadorCatalogoPrueba) PrepararConsulta(context.Context) (puertos.SolicitudConsultarCausasParticipacion, error) {
	return puertos.SolicitudConsultarCausasParticipacion{}, nil
}
func (*preparadorCatalogoPrueba) PrepararConsultaPropuesta(context.Context, string) (puertos.SolicitudConsultarPropuestaCausaParticipacion, error) {
	return puertos.SolicitudConsultarPropuestaCausaParticipacion{}, nil
}

type operadorCatalogoPrueba struct {
	causas []puertos.CausaParticipacionCatalogada
	causa  puertos.CausaParticipacionCatalogada
	recibo string
	err    error
}

func (o operadorCatalogoPrueba) Publicar(context.Context, puertos.SolicitudPublicarCausaParticipacion) (puertos.CausaParticipacionCatalogada, string, error) {
	return o.causa, o.recibo, o.err
}
func (o operadorCatalogoPrueba) Proponer(context.Context, puertos.SolicitudProponerCausaParticipacion) (puertos.PropuestaCausaParticipacion, string, error) {
	return puertos.PropuestaCausaParticipacion{PropuestaRef: "propuesta:causa:" + strings.Repeat("a", 64), CausaParticipacionCatalogada: o.causa}, o.recibo, o.err
}
func (o operadorCatalogoPrueba) Consultar(context.Context, puertos.SolicitudConsultarCausasParticipacion) ([]puertos.CausaParticipacionCatalogada, error) {
	return o.causas, o.err
}
func (o operadorCatalogoPrueba) ConsultarPropuesta(context.Context, puertos.SolicitudConsultarPropuestaCausaParticipacion) (puertos.PropuestaCausaParticipacionLeida, error) {
	return puertos.PropuestaCausaParticipacionLeida{}, o.err
}

func TestHandlerCatalogoGETSoloSeisCamposYVacioArray(t *testing.T) {
	p := &preparadorCatalogoPrueba{}
	h, e := NuevoHandlerCatalogoCausasParticipacion(p, operadorCatalogoPrueba{causas: []puertos.CausaParticipacionCatalogada{{Codigo: "gestion_situacion", Version: 1, HuellaSHA256: "abc", Etiqueta: "Cambio", AplicaSituacion: true, Publicable: true, Activa: true}}})
	if e != nil {
		t.Fatal(e)
	}
	r := httptest.NewRequest(http.MethodGet, RutaCatalogoCausasParticipacion, nil)
	r.Header.Set("Accept", "application/json")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != 200 {
		t.Fatalf("estado=%d cuerpo=%s", w.Code, w.Body.String())
	}
	for _, prohibido := range []string{"publicable", "activa"} {
		if strings.Contains(w.Body.String(), prohibido) {
			t.Fatalf("GET expone %s: %s", prohibido, w.Body.String())
		}
	}
	h, e = NuevoHandlerCatalogoCausasParticipacion(p, operadorCatalogoPrueba{})
	if e != nil {
		t.Fatal(e)
	}
	w = httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != 200 || !strings.Contains(w.Body.String(), `"data":[]`) {
		t.Fatalf("vacio=%d %s", w.Code, w.Body.String())
	}
}
func TestHandlerCatalogoPOSTRespuestaYFormato(t *testing.T) {
	p := &preparadorCatalogoPrueba{}
	c := puertos.CausaParticipacionCatalogada{Codigo: "gestion_situacion", Version: 2, HuellaSHA256: "abc", Etiqueta: "Cambio", AplicaSituacion: true, Publicable: true, Activa: true}
	h, _ := NuevoHandlerCatalogoCausasParticipacion(p, operadorCatalogoPrueba{causa: c, recibo: "recibo:causa:abc"})
	r := httptest.NewRequest(http.MethodPost, RutaPropuestasCausasParticipacion, strings.NewReader(`{"codigo":"gestion_situacion","version":2,"etiqueta":"Cambio","aplica_situacion":true,"aplica_contacto":false,"publicable":true,"activa":true}`))
	r.Header.Set("Content-Type", "application/json")
	r.Header.Set("Accept", "application/json")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != 201 || !strings.Contains(w.Body.String(), `"propuesta"`) || !strings.Contains(w.Body.String(), `"recibo":"recibo:causa:abc"`) {
		t.Fatalf("respuesta=%d %s", w.Code, w.Body.String())
	}
	if p.entrada.Codigo != "gestion_situacion" {
		t.Fatal("no paso entrada")
	}
	r = httptest.NewRequest(http.MethodPost, RutaPropuestasCausasParticipacion, strings.NewReader(`{"codigo":"x","extra":true}`))
	r.Header.Set("Content-Type", "application/json")
	r.Header.Set("Accept", "application/json")
	w = httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != 400 {
		t.Fatalf("formato=%d", w.Code)
	}
	for _, body := range []string{
		`{"codigo":"gestion_situacion","version":2,"etiqueta":"Cambio","aplica_situacion":true,"aplica_contacto":false,"publicable":true}`,
		`{"codigo":"gestion_situacion","version":2,"etiqueta":"Cambio\u0001","aplica_situacion":true,"aplica_contacto":false,"publicable":true,"activa":true}`,
		`{"codigo":"X","version":2,"etiqueta":"Cambio","aplica_situacion":true,"aplica_contacto":false,"publicable":true,"activa":true}`,
		`{"codigo":"gestion_situacion","version":2,"etiqueta":" Cambio","aplica_situacion":true,"aplica_contacto":false,"publicable":true,"activa":true}`,
		`{"codigo":"gestion_situacion","version":2,"etiqueta":"Cambio\u200b","aplica_situacion":true,"aplica_contacto":false,"publicable":true,"activa":true}`,
	} {
		r = httptest.NewRequest(http.MethodPost, RutaPropuestasCausasParticipacion, strings.NewReader(body))
		r.Header.Set("Content-Type", "application/json")
		r.Header.Set("Accept", "application/json")
		w = httptest.NewRecorder()
		h.ServeHTTP(w, r)
		if w.Code != 400 {
			t.Fatalf("POST incompleto o control aceptado: %d", w.Code)
		}
	}
}
func TestHandlerCatalogoConflicto(t *testing.T) {
	p := &preparadorCatalogoPrueba{}
	h, _ := NuevoHandlerCatalogoCausasParticipacion(p, operadorCatalogoPrueba{err: puertos.ErrCatalogoCausasParticipacionEnConflicto})
	r := httptest.NewRequest(http.MethodPost, RutaPropuestasCausasParticipacion, strings.NewReader(`{"codigo":"gestion_situacion","version":2,"etiqueta":"Cambio","aplica_situacion":true,"aplica_contacto":false,"publicable":true,"activa":true}`))
	r.Header.Set("Content-Type", "application/json")
	r.Header.Set("Accept", "application/json")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != 409 || !strings.Contains(w.Body.String(), "catalogo_en_conflicto") {
		t.Fatalf("conflicto=%d %s", w.Code, w.Body.String())
	}
}
