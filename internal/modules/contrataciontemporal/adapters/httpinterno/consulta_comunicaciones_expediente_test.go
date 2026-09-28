package httpinterno

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

type autoridadConsultaComunicacionesPrueba struct {
	err      error
	llamadas int
}

func (a *autoridadConsultaComunicacionesPrueba) ResolverContextoConsultaComunicacionesExpediente(context.Context) error {
	a.llamadas++
	return a.err
}

type lectorConsultaComunicacionesPrueba struct {
	err      error
	llamadas int
	ultima   ports.ConsultaComunicacionesExpediente
}

func (l *lectorConsultaComunicacionesPrueba) ConsultarComunicacionesExpediente(_ context.Context, c ports.ConsultaComunicacionesExpediente) (ports.PaginaComunicacionesExpediente, error) {
	l.llamadas++
	l.ultima = c
	if l.err != nil {
		return ports.PaginaComunicacionesExpediente{}, l.err
	}
	return ports.PaginaComunicacionesExpediente{ExpedienteRef: c.ExpedienteRef, Comunicaciones: []ports.ComunicacionExpediente{
		{OrganizacionRef: "organizacion:ct140", ExpedienteRef: c.ExpedienteRef, LlamamientoRef: "llamamiento:ct140",
			ComunicacionRef: "comunicacion:ct140", Version: 2, Estado: "registrada_localmente",
			RegistradaEn:          time.Date(2026, 9, 28, 12, 0, 0, 123456000, time.UTC),
			ReciboComunicacionRef: "recibo:ct140", AntecedenteTipo: "seleccion_confirmada", ReciboAntecedenteRef: "recibo:seleccion"},
	}}, nil
}

func TestConsultaComunicacionesExpedienteGET(t *testing.T) {
	a, l := &autoridadConsultaComunicacionesPrueba{}, &lectorConsultaComunicacionesPrueba{}
	h, err := NuevoManejadorConsultaComunicacionesExpediente(a, l)
	if err != nil {
		t.Fatal(err)
	}
	r := httptest.NewRequest(http.MethodGet, RutaConsultaComunicacionesExpediente+"?expediente_ref=expediente:ct140&limite=2", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != 200 || a.llamadas != 1 || l.llamadas != 1 || l.ultima.Limite != 2 {
		t.Fatalf("GET: %d, %+v, %+v", w.Code, a, l)
	}
	var salida struct {
		Data ports.PaginaComunicacionesExpediente `json:"data"`
	}
	if err = json.Unmarshal(w.Body.Bytes(), &salida); err != nil || len(salida.Data.Comunicaciones) != 1 ||
		salida.Data.Comunicaciones[0].ReciboAntecedenteRef != "recibo:seleccion" {
		t.Fatalf("salida: %v %+v", err, salida)
	}
	if w.Header().Get("Cache-Control") != "no-store, no-transform" {
		t.Fatalf("cache: %v", w.Header())
	}
}

func TestConsultaComunicacionesExpedienteErroresSinLectura(t *testing.T) {
	casos := []struct {
		nombre, url             string
		autoridad, errorLectura error
		estado                  int
	}{
		{"duplicado", RutaConsultaComunicacionesExpediente + "?expediente_ref=expediente:ct140&expediente_ref=expediente:otro", nil, nil, 400},
		{"ambito_cliente", RutaConsultaComunicacionesExpediente + "?expediente_ref=expediente:ct140&organizacion_ref=organizacion:ajena", nil, nil, 400},
		{"sin_identidad", RutaConsultaComunicacionesExpediente + "?expediente_ref=expediente:ct140", ErrContextoCanalAusente, nil, 401},
		{"denegado", RutaConsultaComunicacionesExpediente + "?expediente_ref=expediente:ct140", nil, ports.ErrConsultaComunicacionesExpedienteDenegada, 403},
		{"otro_expediente", RutaConsultaComunicacionesExpediente + "?expediente_ref=expediente:ct140", nil, ports.ErrConsultaComunicacionesExpedienteNoEncontrado, 404},
		{"transitorio", RutaConsultaComunicacionesExpediente + "?expediente_ref=expediente:ct140", nil, ports.ErrConsultaComunicacionesExpedienteNoDisponible, 503},
	}
	for _, tc := range casos {
		t.Run(tc.nombre, func(t *testing.T) {
			a, l := &autoridadConsultaComunicacionesPrueba{err: tc.autoridad}, &lectorConsultaComunicacionesPrueba{err: tc.errorLectura}
			h, err := NuevoManejadorConsultaComunicacionesExpediente(a, l)
			if err != nil {
				t.Fatal(err)
			}
			w := httptest.NewRecorder()
			h.ServeHTTP(w, httptest.NewRequest(http.MethodGet, tc.url, nil))
			if w.Code != tc.estado {
				t.Fatalf("estado %d, esperaba %d", w.Code, tc.estado)
			}
			if tc.estado == 400 || tc.estado == 401 {
				if l.llamadas != 0 {
					t.Fatal("consultó sin entrada/identidad válida")
				}
			}
			var salida map[string]any
			if err = json.Unmarshal(w.Body.Bytes(), &salida); err != nil || salida["data"] != nil {
				t.Fatalf("filtración: %v %s", err, w.Body.String())
			}
		})
	}
}
