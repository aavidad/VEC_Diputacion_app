package httpinterno

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

type ejecutorPortalHTTPPrueba struct {
	ejecutorComunicacionLlamamientoHTTPPrueba
	solicitud ports.SolicitudResolucionPortal
	resultado ports.ResultadoResolucionPortal
	llamadas  int
}

func (e *ejecutorPortalHTTPPrueba) ResolverPortal(_ context.Context, s ports.SolicitudResolucionPortal) (ports.ResultadoResolucionPortal, error) {
	e.solicitud = s
	e.llamadas++
	return e.resultado, nil
}

func TestResolucionPortalHTTPConservaFuenteYSesion(t *testing.T) {
	entrada := resolucionPortalJSON{
		FuenteRespuesta: "portal", ClaveIdempotencia: claveResolverComunicacionHTTPPrueba,
		ExpedienteRef: "expediente:http-comunicacion", BolsaRef: "bolsa:http-comunicacion",
		LlamamientoRef: "llamamiento:http-comunicacion", ReciboEmisionRef: "recibo:emision:http-comunicacion",
		ParticipacionRef: "participacion:http-comunicacion", ReciboRespuestaRef: "recibo:respuesta:http-comunicacion",
		Respuesta: "aceptacion", RevisionRespuestaRRHH: true, RevisionPlazoRRHH: true,
		CriterioValidacionRef: "politica:ct:revision-manual-sintetica:20260906",
	}
	cuerpo, err := json.Marshal(entrada)
	if err != nil {
		t.Fatal(err)
	}
	s, err := entrada.solicitud()
	if err != nil {
		t.Fatal(err)
	}
	s.OrganizacionRef = "organizacion:http-comunicacion"
	resuelta := time.Date(2026, 10, 20, 12, 0, 0, 0, time.UTC)
	r := ports.ResultadoResolucionPortal{
		Solicitud: s, Politica: ports.ReferenciaGobernadaComunicacionLlamamiento{
			Referencia: s.CriterioValidacionRef, Version: 1, HuellaSHA256: strings.Repeat("a", 64),
		}, FuenteRespuesta: "portal", EstadoPlazo: "revision_manual_portal",
		RespuestaPortal: ports.RespuestaPortalResolucion{ParticipacionRef: s.ParticipacionRef,
			RespuestaRef: "respuesta:http-comunicacion", ReciboRespuestaRef: s.ReciboRespuestaRef,
			RespondidaEn: resuelta.Add(-time.Minute)},
		EvaluacionPlazoRef: "evaluacion:http-comunicacion", ResolucionRef: "resolucion:http-comunicacion",
		ReciboLocalRef: "recibo:resolucion:http-comunicacion", AuditoriaRef: "auditoria:http-comunicacion",
		VersionResultante: 1, ResueltaEn: resuelta, Estado: ports.ResultadoComunicacionLlamamientoConfirmado,
	}
	if r.ValidarPara(s) != nil {
		t.Fatal("recibo de prueba inválido")
	}
	ejecutor := &ejecutorPortalHTTPPrueba{resultado: r}
	h := nuevoManejadorComunicacionHTTPPrueba(t, ejecutor)
	for indice, caso := range []struct {
		cuerpo string
		estado int
		llama  bool
	}{
		{string(cuerpo), http.StatusCreated, true},
		{strings.Replace(string(cuerpo), `"participacion_ref":"participacion:http-comunicacion"`, `"participacion_ref":""`, 1), http.StatusUnprocessableEntity, false},
		{strings.Replace(string(cuerpo), `"fuente_respuesta":"portal"`, `"fuente_respuesta":"correo"`, 1), http.StatusUnprocessableEntity, false},
		{strings.Replace(string(cuerpo), `"expediente_ref":`, `"organizacion_ref":"organizacion:falsa","expediente_ref":`, 1), http.StatusBadRequest, false},
	} {
		if caso.llama {
			_, _, _, err := leerVarianteResolucion(httptest.NewRecorder(), peticionComunicacionHTTPPrueba(RutaResolucionComunicacionLlamamiento, caso.cuerpo))
			if err != nil {
				t.Fatalf("entrada portal: %v", err)
			}
		}
		antes := ejecutor.llamadas
		w := httptest.NewRecorder()
		h.ServeHTTP(w, peticionComunicacionHTTPPrueba(RutaResolucionComunicacionLlamamiento, caso.cuerpo))
		if w.Code != caso.estado || (ejecutor.llamadas > antes) != caso.llama {
			t.Fatalf("variante portal %d: estado %d, llamadas %d: %s", indice, w.Code, ejecutor.llamadas-antes, w.Body.String())
		}
		if caso.llama && (ejecutor.solicitud.OrganizacionRef != "" ||
			!strings.Contains(w.Body.String(), `"fuente_respuesta":"portal"`) ||
			!strings.Contains(w.Body.String(), `"estado_plazo":"revision_manual_portal"`)) {
			t.Fatal("respuesta portal o ámbito confiable alterado")
		}
	}
	// Sin dependencia portal, la rama antigua sigue montada y esta falla localmente.
	legacy := nuevoManejadorComunicacionHTTPPrueba(t, &ejecutorComunicacionLlamamientoHTTPPrueba{})
	w := httptest.NewRecorder()
	legacy.ServeHTTP(w, peticionComunicacionHTTPPrueba(RutaResolucionComunicacionLlamamiento, string(cuerpo)))
	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("portal opcional sin dependencia: %d", w.Code)
	}
}
