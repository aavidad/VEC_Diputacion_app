package httpinterno

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/application"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

type ejecutorPlazoPrueba struct {
	llamadas int
	estado   string
	err      error
}

func (e *ejecutorPlazoPrueba) Registrar(_ context.Context, s ports.SolicitudRegistrarEventoPlazoLlamamiento) (ports.EventoPlazoLlamamientoRegistrado, error) {
	e.llamadas++
	if e.err != nil {
		return ports.EventoPlazoLlamamientoRegistrado{}, e.err
	}
	r := ports.EventoPlazoLlamamientoRegistrado{Solicitud: s, EventoRef: "evento:sintetico", ReciboRef: "recibo:sintetico",
		AuditoriaRef: "auditoria:sintetica", RegistradoEn: s.InstanteEn.Add(time.Minute), Estado: e.estado}
	if s.Tipo == ports.EventoPlazoContactoEfectivo {
		ref := func(entrada string) ports.ReferenciaGobernadaComunicacionLlamamiento {
			return ports.ReferenciaGobernadaComunicacionLlamamiento{Referencia: "vec.bolsa.reglas:1:" + entrada, Version: 1, HuellaSHA256: strings.Repeat("f", 64)}
		}
		r.Plazo = &ports.PlazoRespuestaGobernado{RespuestaHasta: time.Date(2026, 9, 29, 22, 0, 0, 0, time.UTC), UltimoDia: "2026-09-29",
			Politica: ref("b05.plazo_respuesta"), TratamientoFueraDePlazo: domain.TratamientoFueraDePlazoExigeCausaJustificada,
			ConfirmacionExpiracion: domain.ConfirmacionExpiracionRRHH, CriterioRespuesta: ref("b07.fuera_de_plazo"),
			CriterioExpiracion: ref("b08.sin_respuesta_baja"), ReglaEjemplo: true}
	}
	return r, nil
}

func entradaPlazoPrueba(tipo string) eventoPlazoJSON {
	return eventoPlazoJSON{ClaveIdempotencia: "11111111-1111-4111-8111-111111111111", OrganizacionRef: "org:sintetica",
		ExpedienteRef: "exp:sintetico", LlamamientoRef: "llamamiento:sintetico", ComunicacionRef: "comunicacion:sintetica",
		VersionComunicacionEsperada: 2, Tipo: tipo, InstanteEn: "2026-09-28T09:30:00Z", PruebaRef: "prueba:llamada"}
}

func peticionPlazoPrueba(entrada any) *http.Request {
	b, _ := json.Marshal(entrada)
	r := httptest.NewRequest("POST", RutaEventoPlazoLlamamiento, strings.NewReader(string(b)))
	r.Header.Set("Content-Type", "application/json")
	r.Header.Set("Accept", "application/json")
	return r
}

func TestEventoPlazoHTTPPublicaVencimientoYPropuesta(t *testing.T) {
	for _, caso := range []struct {
		nombre, tipo, estado string
		err                  error
		http                 int
	}{
		{"contacto", "contacto_efectivo", "registrado", nil, 201},
		{"replay", "contacto_efectivo", "replay_registrado", nil, 200},
		{"causa", "causa_justificada", "registrado", nil, 201},
		{"sin reglas", "contacto_efectivo", "", application.ErrReglasPlazoLlamamientoNoDisponibles, 503},
		{"conflicto", "contacto_efectivo", "", application.ErrEventoPlazoEnConflicto, 409},
		{"denegado", "contacto_efectivo", "", application.ErrEventoPlazoDenegado, 403},
		{"clave", "contacto_efectivo", "", application.ErrClaveEventoPlazoEnColision, 409},
	} {
		t.Run(caso.nombre, func(t *testing.T) {
			e := &ejecutorPlazoPrueba{estado: caso.estado, err: caso.err}
			h, err := NuevoManejadorEventoPlazoLlamamiento(e)
			if err != nil {
				t.Fatal(err)
			}
			w := httptest.NewRecorder()
			h.ServeHTTP(w, peticionPlazoPrueba(entradaPlazoPrueba(caso.tipo)))
			if w.Code != caso.http || e.llamadas != 1 || w.Header().Get("Set-Cookie") != "" {
				t.Fatalf("HTTP %d: %s", w.Code, w.Body.String())
			}
			if caso.err != nil {
				if !strings.Contains(w.Body.String(), "api.contratacion_temporal.plazo_llamamiento.error.") {
					t.Fatalf("error sin clave i18n: %s", w.Body.String())
				}
				return
			}
			var salida struct {
				Data eventoPlazoSalidaJSON `json:"data"`
			}
			if json.Unmarshal(w.Body.Bytes(), &salida) != nil || salida.Data.Esquema != EsquemaEventoPlazoLlamamiento ||
				salida.Data.Estado != caso.estado || salida.Data.InstanteEn != "2026-09-28T09:30:00Z" {
				t.Fatalf("recibo desligado: %s", w.Body.String())
			}
			p := salida.Data.Plazo
			if (caso.tipo == "causa_justificada") != (p == nil) {
				t.Fatalf("plazo solo con el contacto efectivo: %s", w.Body.String())
			}
			if p != nil && (p.RespuestaHasta != "2026-09-29T22:00:00Z" || p.UltimoDia != "2026-09-29" ||
				p.TratamientoFueraDePlazo != "exige_causa_justificada" || p.ConfirmacionExpiracion != "rrhh" ||
				p.CriterioExpiracionRef != "vec.bolsa.reglas:1:b08.sin_respuesta_baja" || !p.ReglaEjemplo) {
				t.Fatalf("plazo inesperado: %+v", p)
			}
		})
	}
}

func TestEventoPlazoHTTPNoAceptaPlazoNiAutoridadDelCliente(t *testing.T) {
	for nombre, entrada := range map[string]any{
		"tipo correo": func() eventoPlazoJSON { e := entradaPlazoPrueba("correo"); return e }(),
		"version": func() eventoPlazoJSON {
			e := entradaPlazoPrueba("contacto_efectivo")
			e.VersionComunicacionEsperada = 1
			return e
		}(),
		"instante no canónico": func() eventoPlazoJSON {
			e := entradaPlazoPrueba("contacto_efectivo")
			e.InstanteEn = "2026-09-28T09:30:00.0000001Z"
			return e
		}(),
		"vencimiento del cliente": map[string]any{"clave_idempotencia": "11111111-1111-4111-8111-111111111111",
			"respuesta_hasta": "2026-12-31T00:00:00Z"},
		"actor": map[string]any{"actor_ref": "actor:ajeno"},
	} {
		t.Run(nombre, func(t *testing.T) {
			e := &ejecutorPlazoPrueba{estado: "registrado"}
			h, _ := NuevoManejadorEventoPlazoLlamamiento(e)
			w := httptest.NewRecorder()
			h.ServeHTTP(w, peticionPlazoPrueba(entrada))
			if w.Code < 400 || e.llamadas != 0 {
				t.Fatalf("entrada aceptada: %d %s", w.Code, w.Body.String())
			}
		})
	}
	h, _ := NuevoManejadorEventoPlazoLlamamiento(&ejecutorPlazoPrueba{})
	w := httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest("GET", RutaEventoPlazoLlamamiento, nil))
	if w.Code != http.StatusMethodNotAllowed {
		t.Fatalf("GET: %d", w.Code)
	}
}
