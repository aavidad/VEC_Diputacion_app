package httpinterno

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

type consultorEstadisticasPrueba struct {
	alcance  ports.AlcanceEstadisticasRRHH
	consulta ports.ConsultaEstadisticasRRHH
	err      error
}

func (c *consultorEstadisticasPrueba) ConsultarEstadisticasRRHH(_ context.Context, alcance ports.AlcanceEstadisticasRRHH, consulta ports.ConsultaEstadisticasRRHH) (ports.EstadisticasRRHH, error) {
	c.alcance, c.consulta = alcance, consulta
	if c.err != nil {
		return ports.EstadisticasRRHH{}, c.err
	}
	return ports.EstadisticasRRHH{CorteGlobal: 169, Series: []ports.SerieEstadisticasRRHH{
		{Inicio: time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC)},
		{Inicio: time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC), Altas: 71, Formalizaciones: 3, Incidencias: 2},
	}}, nil
}

type resolutorAlcancePrueba struct{ err error }

func (r resolutorAlcancePrueba) AlcanceEstadisticasRRHH(context.Context) (ports.AlcanceEstadisticasRRHH, error) {
	if r.err != nil {
		return ports.AlcanceEstadisticasRRHH{}, r.err
	}
	return ports.AlcanceEstadisticasRRHH{OrganizacionRef: "organizacion:desarrollo:dipgra", ClaseAmbito: ports.AmbitoOrganizacionRRHH, AmbitoRef: "organizacion:desarrollo:dipgra"}, nil
}

func relojEstadisticasPrueba() time.Time {
	return time.Date(2026, 9, 18, 22, 30, 0, 0, time.UTC)
}

func manejadorEstadisticasPrueba(t *testing.T, consultor *consultorEstadisticasPrueba, alcance ResolutorAlcanceEstadisticasRRHH) http.Handler {
	t.Helper()
	h, err := NuevoManejadorEstadisticasRRHH(consultor, alcance, relojEstadisticasPrueba)
	if err != nil {
		t.Fatalf("manejador: %v", err)
	}
	return h
}

func TestEstadisticasRRHHRespondeSeriesYTotales(t *testing.T) {
	consultor := &consultorEstadisticasPrueba{}
	h := manejadorEstadisticasPrueba(t, consultor, resolutorAlcancePrueba{})
	w := httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest(http.MethodGet, RutaEstadisticasRRHH+"?periodo=mensual&desde=2026-07-01&hasta=2026-09-30", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("estado %d: %s", w.Code, w.Body.String())
	}
	if w.Header().Get("Cache-Control") != "no-store, no-transform" || w.Header().Get("Content-Type") != "application/json; charset=utf-8" {
		t.Fatalf("cabeceras: %v", w.Header())
	}
	var respuesta struct {
		Data struct {
			Esquema     string `json:"esquema"`
			Periodo     string `json:"periodo"`
			Desde       string `json:"desde"`
			Hasta       string `json:"hasta"`
			ZonaHoraria string `json:"zona_horaria"`
			CorteGlobal uint64 `json:"corte_global"`
			Series      []struct {
				Inicio string `json:"inicio"`
				Altas  uint64 `json:"altas"`
			} `json:"series"`
			Totales struct {
				Altas           uint64 `json:"altas"`
				Formalizaciones uint64 `json:"formalizaciones"`
				Incidencias     uint64 `json:"incidencias"`
			} `json:"totales"`
		} `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &respuesta); err != nil {
		t.Fatalf("json: %v", err)
	}
	d := respuesta.Data
	if d.Esquema != EsquemaEstadisticasRRHH || d.Periodo != "mensual" || d.Desde != "2026-07-01" || d.Hasta != "2026-09-30" ||
		d.ZonaHoraria != "Europe/Madrid" || d.CorteGlobal != 169 || len(d.Series) != 2 || d.Series[1].Inicio != "2026-09-01" || d.Series[1].Altas != 71 ||
		d.Totales.Altas != 71 || d.Totales.Formalizaciones != 3 || d.Totales.Incidencias != 2 {
		t.Fatalf("respuesta inesperada: %s", w.Body.String())
	}
	if consultor.alcance.OrganizacionRef != "organizacion:desarrollo:dipgra" || consultor.consulta.Periodo != "mensual" {
		t.Fatalf("el consultor no recibió alcance y consulta: %+v %+v", consultor.alcance, consultor.consulta)
	}
}

func TestEstadisticasRRHHValoresPorDefecto(t *testing.T) {
	consultor := &consultorEstadisticasPrueba{}
	h := manejadorEstadisticasPrueba(t, consultor, resolutorAlcancePrueba{})
	w := httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest(http.MethodGet, RutaEstadisticasRRHH, nil))
	if w.Code != http.StatusOK {
		t.Fatalf("estado %d: %s", w.Code, w.Body.String())
	}
	// 18/09/2026 22:30 UTC son las 00:30 del 19/09 en Madrid.
	if consultor.consulta.Periodo != "mensual" || consultor.consulta.Hasta.Format("2006-01-02") != "2026-09-19" || consultor.consulta.Desde.Format("2006-01-02") != "2025-10-01" {
		t.Fatalf("valores por defecto inesperados: %+v", consultor.consulta)
	}
	w = httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest(http.MethodGet, RutaEstadisticasRRHH+"?periodo=semanal", nil))
	if w.Code != http.StatusOK || consultor.consulta.Desde.Format("2006-01-02") != "2026-07-04" {
		t.Fatalf("semanal por defecto: %d %+v", w.Code, consultor.consulta)
	}
	w = httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest(http.MethodGet, RutaEstadisticasRRHH+"?periodo=anual", nil))
	if w.Code != http.StatusOK || consultor.consulta.Desde.Format("2006-01-02") != "2022-01-01" {
		t.Fatalf("anual por defecto: %d %+v", w.Code, consultor.consulta)
	}
}

func TestEstadisticasRRHHRechazaPeticionesNoValidas(t *testing.T) {
	consultor := &consultorEstadisticasPrueba{}
	h := manejadorEstadisticasPrueba(t, consultor, resolutorAlcancePrueba{})
	casos := []struct {
		metodo, url string
		estado      int
	}{
		{http.MethodPost, RutaEstadisticasRRHH, http.StatusMethodNotAllowed},
		{http.MethodGet, RutaEstadisticasRRHH + "/", http.StatusNotFound},
		{http.MethodGet, RutaEstadisticasRRHH + "?periodo=diario", http.StatusBadRequest},
		{http.MethodGet, RutaEstadisticasRRHH + "?periodo=mensual&periodo=anual", http.StatusBadRequest},
		{http.MethodGet, RutaEstadisticasRRHH + "?desde=2026-09-19&hasta=2026-09-18", http.StatusBadRequest},
		{http.MethodGet, RutaEstadisticasRRHH + "?desde=2026-9-1", http.StatusBadRequest},
		{http.MethodGet, RutaEstadisticasRRHH + "?limite=10", http.StatusBadRequest},
		{http.MethodGet, RutaEstadisticasRRHH + "?periodo=", http.StatusBadRequest},
		{http.MethodGet, RutaEstadisticasRRHH + "?periodo=semanal&desde=2000-01-01&hasta=2026-01-01", http.StatusBadRequest},
	}
	for _, caso := range casos {
		w := httptest.NewRecorder()
		h.ServeHTTP(w, httptest.NewRequest(caso.metodo, caso.url, nil))
		if w.Code != caso.estado {
			t.Fatalf("%s %s: estado %d (esperado %d): %s", caso.metodo, caso.url, w.Code, caso.estado, w.Body.String())
		}
	}
	if consultor.consulta.Periodo != "" {
		t.Fatal("una petición no válida no debe llegar al consultor")
	}
}

func TestEstadisticasRRHHSinLectorAcreditadoDeniega(t *testing.T) {
	consultor := &consultorEstadisticasPrueba{}
	h := manejadorEstadisticasPrueba(t, consultor, resolutorAlcancePrueba{err: ports.ErrEstadisticasRRHHNoDisponible})
	w := httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest(http.MethodGet, RutaEstadisticasRRHH, nil))
	if w.Code != http.StatusForbidden || consultor.consulta.Periodo != "" {
		t.Fatalf("estado %d: %s", w.Code, w.Body.String())
	}
}

func TestEstadisticasRRHHErroresDelConsultor(t *testing.T) {
	consultor := &consultorEstadisticasPrueba{err: ports.ErrEstadisticasRRHHInvalida}
	h := manejadorEstadisticasPrueba(t, consultor, resolutorAlcancePrueba{})
	w := httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest(http.MethodGet, RutaEstadisticasRRHH, nil))
	if w.Code != http.StatusBadRequest {
		t.Fatalf("consulta inválida: estado %d", w.Code)
	}
	consultor.err = errors.New("caída")
	w = httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest(http.MethodGet, RutaEstadisticasRRHH, nil))
	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("consultor no disponible: estado %d", w.Code)
	}
}

func TestEstadisticasRRHHHead(t *testing.T) {
	h := manejadorEstadisticasPrueba(t, &consultorEstadisticasPrueba{}, resolutorAlcancePrueba{})
	w := httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest(http.MethodHead, RutaEstadisticasRRHH, nil))
	if w.Code != http.StatusOK || w.Body.Len() != 0 {
		t.Fatalf("HEAD: %d %d", w.Code, w.Body.Len())
	}
}

func TestNuevoManejadorEstadisticasRRHHExigeDependencias(t *testing.T) {
	if _, err := NuevoManejadorEstadisticasRRHH(nil, resolutorAlcancePrueba{}, relojEstadisticasPrueba); err == nil {
		t.Fatal("consultor nulo")
	}
	if _, err := NuevoManejadorEstadisticasRRHH(&consultorEstadisticasPrueba{}, nil, relojEstadisticasPrueba); err == nil {
		t.Fatal("resolutor nulo")
	}
	if _, err := NuevoManejadorEstadisticasRRHH(&consultorEstadisticasPrueba{}, resolutorAlcancePrueba{}, nil); err == nil {
		t.Fatal("reloj nulo")
	}
}
