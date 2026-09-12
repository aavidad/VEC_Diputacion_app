package httpinterno

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

type renderizadorDOCXRRHHPrueba struct {
	llamadas int
	tipo     ports.TipoBorradorRRHH
}

func (r *renderizadorDOCXRRHHPrueba) RenderizarBorradorDOCX(
	_ context.Context, tipo ports.TipoBorradorRRHH, _ ports.DetalleExpedienteRRHH,
) ([]byte, error) {
	r.llamadas++
	r.tipo = tipo
	return []byte("PK\x03\x04docx-sintetico"), nil
}

func TestConsultaRRHHDOCXSeisDescargasTrasLaMismaLecturaAutorizada(t *testing.T) {
	for _, caso := range []struct {
		tipo   ports.TipoBorradorRRHH
		accept string
		nombre string
	}{
		{ports.BorradorInformeDefinitivo, AcceptInformeDefinitivoDOCXRRHH, "informe-definitivo-borrador.docx"},
		{ports.BorradorResolucion, AcceptResolucionDOCXRRHH, "resolucion-borrador.docx"},
		{ports.BorradorDiligencia, AcceptDiligenciaDOCXRRHH, "diligencia-borrador.docx"},
		{ports.BorradorTomaPosesion, AcceptTomaPosesionDOCXRRHH, "toma-posesion-borrador.docx"},
		{ports.BorradorNotificacion, AcceptNotificacionDOCXRRHH, "notificacion-borrador.docx"},
		{ports.BorradorComunicacionCentro, AcceptComunicacionCentroDOCXRRHH, "comunicacion-centro-borrador.docx"},
	} {
		t.Run(string(caso.tipo), func(t *testing.T) {
			consultor := &consultorDetalleRRHHPrueba{detalle: detalleInformeRRHHPrueba()}
			docx := &renderizadorDOCXRRHHPrueba{}
			h, err := NuevoManejadorConsultaDetalleRRHHConDOCX(consultor,
				&renderizadorBorradorRRHHPrueba{contenido: []byte("%PDF-1.4")}, docx)
			if err != nil {
				t.Fatal(err)
			}
			peticion := peticionInformeRRHHPrueba()
			peticion.Header.Set("Accept", caso.accept)
			respuesta := httptest.NewRecorder()
			h.ServeHTTP(respuesta, peticion)
			if respuesta.Code != http.StatusOK || consultor.llamadas != 1 || docx.llamadas != 1 || docx.tipo != caso.tipo {
				t.Fatalf("lectura/render DOCX: estado=%d lectura=%d render=%d tipo=%s", respuesta.Code, consultor.llamadas, docx.llamadas, docx.tipo)
			}
			if !bytes.HasPrefix(respuesta.Body.Bytes(), []byte("PK\x03\x04")) ||
				respuesta.Header().Get("Content-Type") != MIMEDOCXBorradorRRHH ||
				respuesta.Header().Get("Content-Disposition") != `attachment; filename="`+caso.nombre+`"` {
				t.Fatal("representación DOCX inesperada")
			}
		})
	}
}

func TestConsultaRRHHDOCXDeniegaSinRenderizar(t *testing.T) {
	detalle := detalleInformeRRHHPrueba()
	detalle.Hitos = nil
	consultor := &consultorDetalleRRHHPrueba{detalle: detalle}
	docx := &renderizadorDOCXRRHHPrueba{}
	h, err := NuevoManejadorConsultaDetalleRRHHConDOCX(consultor,
		&renderizadorBorradorRRHHPrueba{contenido: []byte("%PDF-1.4")}, docx)
	if err != nil {
		t.Fatal(err)
	}
	peticion := peticionInformeRRHHPrueba()
	peticion.Header.Set("Accept", AcceptResolucionDOCXRRHH)
	respuesta := httptest.NewRecorder()
	h.ServeHTTP(respuesta, peticion)
	if respuesta.Code != http.StatusBadGateway || consultor.llamadas != 1 || docx.llamadas != 0 {
		t.Fatalf("detalle inválido produjo DOCX: estado=%d lectura=%d render=%d", respuesta.Code, consultor.llamadas, docx.llamadas)
	}
}
