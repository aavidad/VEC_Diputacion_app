package httpinterno

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"vec-diputacion-granada/internal/modules/bolsa/ports"
)

type preparadorReintentoPrueba struct {
	preparadorOfertasPrueba
	entrada *EntradaReintentarAvisosOferta
}

func (p *preparadorReintentoPrueba) PrepararSolicitudReintentarAvisosOferta(
	_ context.Context, e EntradaReintentarAvisosOferta,
) (ports.SolicitudReintentarAvisosOferta, error) {
	p.entrada = &e
	return ports.SolicitudReintentarAvisosOferta{BolsaRef: e.BolsaRef,
		OfertaRef: e.OfertaRef, RevisionAvisos: e.RevisionAvisos,
		ConfirmarDuplicado: e.ConfirmarPosibleDuplicado, ClaveIdempotencia: e.ClaveIdempotencia}, nil
}

type operadorReintentoPrueba struct {
	operadorOfertasPrueba
	llamadas    int
	reutilizada bool
	err         error
}

func (o *operadorReintentoPrueba) ReintentarAvisosSinConfirmacion(
	_ context.Context, _ ports.SolicitudReintentarAvisosOferta,
) (ports.OfertaPublicada, error) {
	o.llamadas++
	return ports.OfertaPublicada{OfertaRef: "oferta:1", Reutilizada: o.reutilizada}, o.err
}

const cuerpoReintentoAviso = `{"tipo":"reintentar_avisos_sin_confirmacion","bolsa_ref":"bolsa:1","oferta_ref":"oferta:1","revision_avisos":3,"confirmar_posible_duplicado":true}`

func TestReintentoAvisoUsaLaRutaYClavePropias(t *testing.T) {
	for _, caso := range []struct {
		replay   bool
		esperado int
	}{{false, http.StatusCreated}, {true, http.StatusOK}} {
		p, o := &preparadorReintentoPrueba{}, &operadorReintentoPrueba{reutilizada: caso.replay}
		h, _ := NuevoHandlerOfertasPublicadas(p, o)
		w := httptest.NewRecorder()
		h.ServeHTTP(w, peticionOferta(http.MethodPost, RutaOfertasPublicadas, cuerpoReintentoAviso, "clave-reintento-001"))
		if w.Code != caso.esperado || p.entrada == nil || p.entrada.ClaveIdempotencia != "clave-reintento-001" ||
			p.entrada.RevisionAvisos != 3 || !p.entrada.ConfirmarPosibleDuplicado || o.llamadas != 1 || p.publicar != nil {
			t.Fatalf("estado=%d cuerpo=%s", w.Code, w.Body.String())
		}
	}
}

func TestReintentoAvisoRechazaCamposMezcladosYConflictosNominales(t *testing.T) {
	for _, cuerpo := range []string{
		`{"tipo":"otro","bolsa_ref":"bolsa:1","oferta_ref":"oferta:1","revision_avisos":3,"confirmar_posible_duplicado":true}`,
		`{"tipo":"reintentar_avisos_sin_confirmacion","bolsa_ref":"bolsa:1","oferta_ref":"oferta:1","revision_avisos":3}`,
		`{"tipo":"reintentar_avisos_sin_confirmacion","bolsa_ref":"bolsa:1","oferta_ref":"oferta:1","revision_avisos":3,"confirmar_posible_duplicado":false}`,
		`{"tipo":"reintentar_avisos_sin_confirmacion","bolsa_ref":"bolsa:1","oferta_ref":"oferta:1","revision_avisos":3,"confirmar_posible_duplicado":true,"numero_plazas":1}`,
		`{"tipo":"reintentar_avisos_sin_confirmacion","bolsa_ref":"bolsa:1","oferta_ref":"oferta:1","revision_avisos":3,"confirmar_posible_duplicado":true,"datos":{}}`,
		`{"tipo":"reintentar_avisos_sin_confirmacion","bolsa_ref":"bolsa:1","oferta_ref":"oferta:1","revision_avisos":3,"confirmar_posible_duplicado":true,"datos":null}`,
	} {
		p, o := &preparadorReintentoPrueba{}, &operadorReintentoPrueba{}
		h, _ := NuevoHandlerOfertasPublicadas(p, o)
		w := httptest.NewRecorder()
		h.ServeHTTP(w, peticionOferta(http.MethodPost, RutaOfertasPublicadas, cuerpo, "clave-reintento-001"))
		if w.Code != http.StatusBadRequest || o.llamadas != 0 || p.entrada != nil {
			t.Fatalf("estado=%d cuerpo=%s", w.Code, w.Body.String())
		}
	}
	for _, caso := range []struct {
		err    error
		codigo string
	}{
		{ports.ErrOfertaRevisionAvisos, "aviso_revision_obsoleta"},
		{ports.ErrOfertaAvisoEnCurso, "aviso_envio_en_curso"},
		{ports.ErrOfertaConflicto, "clave_divergente"},
	} {
		p, o := &preparadorReintentoPrueba{}, &operadorReintentoPrueba{err: caso.err}
		h, _ := NuevoHandlerOfertasPublicadas(p, o)
		w := httptest.NewRecorder()
		h.ServeHTTP(w, peticionOferta(http.MethodPost, RutaOfertasPublicadas, cuerpoReintentoAviso, "clave-reintento-001"))
		if w.Code != http.StatusConflict || codigoOferta(t, w) != caso.codigo {
			t.Fatalf("estado=%d cuerpo=%s", w.Code, w.Body.String())
		}
	}
}
