package httpinterno

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

type consultorPreflightPrueba struct {
	err     error
	alterar func(*ports.ResultadoPreflightFirmaR5)
	vistas  []ports.SolicitudPreflightFirmaR5
}

func (c *consultorPreflightPrueba) Consultar(_ context.Context, q ports.SolicitudPreflightFirmaR5) (ports.ResultadoPreflightFirmaR5, error) {
	c.vistas = append(c.vistas, q)
	r := ports.ResultadoPreflightFirmaR5{VersionExpediente: q.Canal.VersionObservada, Documento: q.Documento,
		CatalogoRef: "catalogo:ct:001", CatalogoHuella: strings.Repeat("a", 64), PasoPendiente: 1,
		OriginalRef: q.OriginalRef, OriginalVersion: q.OriginalVersion, ViasDisponibles: []string{}}
	if c.alterar != nil {
		c.alterar(&r)
	}
	return r, c.err
}

const cuerpoPreflightPrueba = `{"expediente_ref":"expediente:ct:001","version_observada":7,"documento":"resolucion","original_ref":"original:ct:001","original_version":1}`

func peticionPreflightPrueba(cuerpo, ruta string) *http.Request {
	r := httptest.NewRequest(http.MethodPost, ruta, strings.NewReader(cuerpo))
	r.Header.Set("Content-Type", "application/json")
	r.Header.Set("Accept", "application/json")
	return r
}

func TestPreflightFirmaR5HTTPContextoRespuestaYMinimizacion(t *testing.T) {
	a := &autoridadCircuitoRRHHPrueba{}
	c := &consultorPreflightPrueba{}
	h, err := NuevoManejadorPreflightFirmaR5(a, c)
	if err != nil {
		t.Fatal(err)
	}
	w := httptest.NewRecorder()
	h.ServeHTTP(w, peticionPreflightPrueba(cuerpoPreflightPrueba, RutaPreflightFirmaR5))
	if w.Code != http.StatusOK || len(c.vistas) != 1 {
		t.Fatalf("preflight: %d %s", w.Code, w.Body.String())
	}
	q := c.vistas[0]
	if q.Canal.PerfilRef != "prf_cccccccccccccccccccccccc" || q.Canal.OrganizacionRef != "organizacion:dipgra:circuito:001" || q.Canal.ExpedienteRef != "expediente:ct:001" || q.OriginalVersion != 1 {
		t.Fatal("canal/material no conservados")
	}
	var salida struct {
		Data map[string]json.RawMessage `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &salida); err != nil {
		t.Fatal(err)
	}
	campos := []string{"version_expediente", "documento", "catalogo_ref", "catalogo_huella", "paso_pendiente", "original_ref", "original_version", "vias_disponibles"}
	if len(salida.Data) != len(campos) {
		t.Fatalf("campos ajenos: %s", w.Body.String())
	}
	for _, campo := range campos {
		if _, ok := salida.Data[campo]; !ok {
			t.Fatalf("falta %s", campo)
		}
	}
	if string(salida.Data["vias_disponibles"]) != "[]" {
		t.Fatal("vías no cerradas")
	}
	if !strings.Contains(w.Header().Get("Cache-Control"), "no-store") {
		t.Fatal("falta no-store")
	}
}

func TestPreflightFirmaR5HTTPRechazaIdentidadURLYOtrasEntradas(t *testing.T) {
	casos := []struct {
		nombre, cuerpo, ruta string
		estado               int
	}{
		{"perfil_body", strings.TrimSuffix(cuerpoPreflightPrueba, "}") + `,"perfil_ref":"prf_ajeno"}`, RutaPreflightFirmaR5, 400},
		{"actor_body", strings.TrimSuffix(cuerpoPreflightPrueba, "}") + `,"actor_ref":"per_ajeno"}`, RutaPreflightFirmaR5, 400},
		{"config_body", strings.TrimSuffix(cuerpoPreflightPrueba, "}") + `,"disponible":true}`, RutaPreflightFirmaR5, 400},
		{"token_url", cuerpoPreflightPrueba, RutaPreflightFirmaR5 + "?token=secreto", 404},
		{"original_ausente", `{"expediente_ref":"expediente:ct:001","version_observada":7,"documento":"resolucion"}`, RutaPreflightFirmaR5, 422},
		{"version_insegura", strings.Replace(cuerpoPreflightPrueba, `"original_version":1`, `"original_version":9007199254740992`, 1), RutaPreflightFirmaR5, 422},
		{"duplicado", strings.TrimSuffix(cuerpoPreflightPrueba, "}") + `,"original_version":2}`, RutaPreflightFirmaR5, 400},
	}
	for _, tc := range casos {
		t.Run(tc.nombre, func(t *testing.T) {
			a := &autoridadCircuitoRRHHPrueba{}
			c := &consultorPreflightPrueba{}
			h, err := NuevoManejadorPreflightFirmaR5(a, c)
			if err != nil {
				t.Fatal(err)
			}
			w := httptest.NewRecorder()
			h.ServeHTTP(w, peticionPreflightPrueba(tc.cuerpo, tc.ruta))
			if w.Code != tc.estado || a.llamadas != 0 || len(c.vistas) != 0 {
				t.Fatalf("entrada aceptada: %d %s", w.Code, w.Body.String())
			}
		})
	}
}

func TestPreflightFirmaR5HTTPErroresNominalesSinDetalles(t *testing.T) {
	casos := []struct {
		err    error
		estado int
	}{
		{ports.ErrFirmaDocumentoDenegada, 404}, {ports.ErrOriginalFirmaNoAutorizado, 404},
		{ports.ErrPreflightFirmaR5NoDisponible, 503}, {ports.ErrPreflightFirmaR5NoConfiable, 502},
		{errors.New("certificado y persona ajena privados"), 503},
	}
	for _, tc := range casos {
		a := &autoridadCircuitoRRHHPrueba{}
		c := &consultorPreflightPrueba{err: tc.err}
		h, err := NuevoManejadorPreflightFirmaR5(a, c)
		if err != nil {
			t.Fatal(err)
		}
		w := httptest.NewRecorder()
		h.ServeHTTP(w, peticionPreflightPrueba(cuerpoPreflightPrueba, RutaPreflightFirmaR5))
		if w.Code != tc.estado || strings.Contains(w.Body.String(), "privados") || strings.Contains(w.Body.String(), "persona") {
			t.Fatalf("error filtra detalle: %d %s", w.Code, w.Body.String())
		}
	}
}

func TestPreflightFirmaR5HTTPNoAceptaDisponibilidadMalformada(t *testing.T) {
	casos := []func(*ports.ResultadoPreflightFirmaR5){
		func(r *ports.ResultadoPreflightFirmaR5) {
			r.ViasDisponibles = []string{ports.ViaFirmaCertificadoVEC, ports.ViaFirmaCertificadoVEC}
		},
		func(r *ports.ResultadoPreflightFirmaR5) { r.OriginalVersion = 2 },
		func(r *ports.ResultadoPreflightFirmaR5) {
			r.PasoPendiente = 0
			r.ViasDisponibles = []string{ports.ViaFirmaCertificadoVEC}
		},
		func(r *ports.ResultadoPreflightFirmaR5) { r.ViasDisponibles = nil },
	}
	for _, alterar := range casos {
		h, err := NuevoManejadorPreflightFirmaR5(&autoridadCircuitoRRHHPrueba{}, &consultorPreflightPrueba{alterar: alterar})
		if err != nil {
			t.Fatal(err)
		}
		w := httptest.NewRecorder()
		h.ServeHTTP(w, peticionPreflightPrueba(cuerpoPreflightPrueba, RutaPreflightFirmaR5))
		if w.Code != 502 {
			t.Fatalf("salida insegura: %d %s", w.Code, w.Body.String())
		}
	}
}
