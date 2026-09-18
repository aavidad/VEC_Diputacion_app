package httpapi

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

type fuentePublicaPrueba struct {
	bolsas     []BolsaPublica
	posiciones map[string][]PosicionPublica
	err        error
}

func (f fuentePublicaPrueba) BolsasPublicas(context.Context) ([]BolsaPublica, time.Time, error) {
	return f.bolsas, time.Date(2026, 9, 18, 20, 0, 0, 0, time.UTC), f.err
}

func (f fuentePublicaPrueba) ListaPublica(_ context.Context, ref string) (BolsaPublica, []PosicionPublica, time.Time, error) {
	if f.err != nil {
		return BolsaPublica{}, nil, time.Time{}, f.err
	}
	for _, bolsa := range f.bolsas {
		if bolsa.BolsaRef == ref {
			return bolsa, f.posiciones[ref], time.Date(2026, 9, 18, 20, 0, 0, 0, time.UTC), nil
		}
	}
	return BolsaPublica{}, nil, time.Time{}, ErrBolsaPublicaNoEncontrada
}

func fuentePruebaConPosiciones(n int) fuentePublicaPrueba {
	posiciones := make([]PosicionPublica, 0, n)
	for i := 1; i <= n; i++ {
		estado := "disponible"
		if i%7 == 0 {
			estado = "ocupado"
		}
		posiciones = append(posiciones, PosicionPublica{Orden: i, DocumentoEnmascarado: fmt.Sprintf("***%04d**", i), EstadoClave: estado})
	}
	return fuentePublicaPrueba{
		bolsas: []BolsaPublica{{BolsaRef: "bolsa:administrativo:2026-09-17", Categoria: "ADMINISTRATIVO", Grupos: []string{"C1"}, TipoLista: "definitiva",
			VigenteDesde: time.Date(2026, 9, 18, 0, 43, 15, 0, time.UTC), Total: n}},
		posiciones: map[string][]PosicionPublica{"bolsa:administrativo:2026-09-17": posiciones},
	}
}

func servirPublico(t *testing.T, fuente FuenteBolsasPublicas, metodo, url string) *httptest.ResponseRecorder {
	t.Helper()
	h, err := NuevoManejadorBolsasPublicas(fuente)
	if err != nil {
		t.Fatalf("manejador: %v", err)
	}
	w := httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest(metodo, url, nil))
	return w
}

func TestBolsasPublicasListaLasBolsas(t *testing.T) {
	w := servirPublico(t, fuentePruebaConPosiciones(41), http.MethodGet, RutaBolsasPublicas)
	if w.Code != http.StatusOK {
		t.Fatalf("estado %d: %s", w.Code, w.Body.String())
	}
	var respuesta struct {
		Data struct {
			Esquema    string `json:"esquema"`
			GeneradoEn string `json:"generado_en"`
			Bolsas     []struct {
				BolsaRef     string   `json:"bolsa_ref"`
				Categoria    string   `json:"categoria"`
				Grupos       []string `json:"grupos"`
				TipoLista    string   `json:"tipo_lista"`
				VigenteDesde string   `json:"vigente_desde"`
				VigenteHasta *string  `json:"vigente_hasta"`
				Total        int      `json:"total"`
			} `json:"bolsas"`
		} `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &respuesta); err != nil {
		t.Fatalf("json: %v", err)
	}
	d := respuesta.Data
	if d.Esquema != EsquemaBolsasPublico || d.GeneradoEn != "2026-09-18T20:00:00Z" || len(d.Bolsas) != 1 ||
		d.Bolsas[0].BolsaRef != "bolsa:administrativo:2026-09-17" || d.Bolsas[0].Total != 41 || d.Bolsas[0].VigenteHasta != nil ||
		len(d.Bolsas[0].Grupos) != 1 || d.Bolsas[0].VigenteDesde != "2026-09-18T00:43:15Z" {
		t.Fatalf("respuesta inesperada: %s", w.Body.String())
	}
	if w.Header().Get("Cache-Control") != "no-store" || w.Header().Get("Content-Security-Policy") == "" {
		t.Fatalf("cabeceras públicas: %v", w.Header())
	}
}

func TestBolsasPublicasListaPaginadaYFiltrada(t *testing.T) {
	fuente := fuentePruebaConPosiciones(41)
	w := servirPublico(t, fuente, http.MethodGet, RutaBolsasPublicas+"/bolsa:administrativo:2026-09-17/lista?limite=10")
	if w.Code != http.StatusOK {
		t.Fatalf("estado %d: %s", w.Code, w.Body.String())
	}
	var pagina struct {
		Data struct {
			Esquema    string `json:"esquema"`
			Bolsa      struct{ Total int }
			Posiciones []struct {
				Orden                int    `json:"orden"`
				DocumentoEnmascarado string `json:"documento_enmascarado"`
				EstadoClave          string `json:"estado_clave"`
			} `json:"posiciones"`
			HayMas          bool    `json:"hay_mas"`
			CursorSiguiente *string `json:"cursor_siguiente"`
		} `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &pagina); err != nil {
		t.Fatalf("json: %v", err)
	}
	if pagina.Data.Esquema != EsquemaListaPublico || len(pagina.Data.Posiciones) != 10 || !pagina.Data.HayMas || pagina.Data.CursorSiguiente == nil || *pagina.Data.CursorSiguiente != "11" ||
		pagina.Data.Posiciones[0].Orden != 1 || pagina.Data.Posiciones[9].Orden != 10 || pagina.Data.Posiciones[6].EstadoClave != "ocupado" {
		t.Fatalf("página inesperada: %s", w.Body.String())
	}
	if strings.Contains(w.Body.String(), "nombre") || strings.Contains(w.Body.String(), "participacion") {
		t.Fatal("la respuesta pública no debe contener nombres ni referencias de participación")
	}
	w = servirPublico(t, fuente, http.MethodGet, RutaBolsasPublicas+"/bolsa:administrativo:2026-09-17/lista?limite=10&cursor=41")
	if err := json.Unmarshal(w.Body.Bytes(), &pagina); err != nil || w.Code != http.StatusOK {
		t.Fatalf("última página: %d %v", w.Code, err)
	}
	if len(pagina.Data.Posiciones) != 1 || pagina.Data.Posiciones[0].Orden != 41 || pagina.Data.HayMas || pagina.Data.CursorSiguiente != nil {
		t.Fatalf("última página inesperada: %s", w.Body.String())
	}
	w = servirPublico(t, fuente, http.MethodGet, RutaBolsasPublicas+"/bolsa:administrativo:2026-09-17/lista?documento=***0014**")
	if err := json.Unmarshal(w.Body.Bytes(), &pagina); err != nil || w.Code != http.StatusOK {
		t.Fatalf("filtro por documento: %d %v", w.Code, err)
	}
	if len(pagina.Data.Posiciones) != 1 || pagina.Data.Posiciones[0].Orden != 14 || pagina.Data.Posiciones[0].EstadoClave != "ocupado" || pagina.Data.HayMas {
		t.Fatalf("filtro inesperado: %s", w.Body.String())
	}
	w = servirPublico(t, fuente, http.MethodGet, RutaBolsasPublicas+"/bolsa:administrativo:2026-09-17/lista?documento=***9999**")
	if err := json.Unmarshal(w.Body.Bytes(), &pagina); err != nil || w.Code != http.StatusOK || len(pagina.Data.Posiciones) != 0 {
		t.Fatalf("documento ausente: %d %v %s", w.Code, err, w.Body.String())
	}
}

func TestBolsasPublicasRechazaConsultasNoValidas(t *testing.T) {
	fuente := fuentePruebaConPosiciones(5)
	lista := RutaBolsasPublicas + "/bolsa:administrativo:2026-09-17/lista"
	casos := []struct {
		metodo, url string
		estado      int
	}{
		{http.MethodPost, RutaBolsasPublicas, http.StatusMethodNotAllowed},
		{http.MethodGet, RutaBolsasPublicas + "?limite=1", http.StatusBadRequest},
		{http.MethodGet, RutaBolsasPublicas + "/", http.StatusNotFound},
		{http.MethodGet, RutaBolsasPublicas + "/bolsa:administrativo:2026-09-17", http.StatusNotFound},
		{http.MethodGet, RutaBolsasPublicas + "/bolsa:otra:2026/lista", http.StatusNotFound},
		{http.MethodGet, RutaBolsasPublicas + "/Bolsa:MAYUSCULAS/lista", http.StatusNotFound},
		{http.MethodGet, lista + "?limite=0", http.StatusBadRequest},
		{http.MethodGet, lista + "?limite=101", http.StatusBadRequest},
		{http.MethodGet, lista + "?cursor=1", http.StatusBadRequest},
		{http.MethodGet, lista + "?cursor=abc", http.StatusBadRequest},
		{http.MethodGet, lista + "?documento=12345678A", http.StatusBadRequest},
		{http.MethodGet, lista + "?documento=***12**", http.StatusBadRequest},
		{http.MethodGet, lista + "?nombre=x", http.StatusBadRequest},
		{http.MethodGet, lista + "?limite=5&limite=6", http.StatusBadRequest},
	}
	for _, caso := range casos {
		w := servirPublico(t, fuente, caso.metodo, caso.url)
		if w.Code != caso.estado {
			t.Fatalf("%s %s: estado %d (esperado %d): %s", caso.metodo, caso.url, w.Code, caso.estado, w.Body.String())
		}
	}
}

func TestBolsasPublicasFuenteNoDisponible(t *testing.T) {
	w := servirPublico(t, fuentePublicaPrueba{err: fmt.Errorf("caída")}, http.MethodGet, RutaBolsasPublicas)
	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("estado %d", w.Code)
	}
	if _, err := NuevoManejadorBolsasPublicas(nil); err == nil {
		t.Fatal("fuente nula")
	}
}

func TestBolsasPublicasNoSirveDatosMalFormados(t *testing.T) {
	fuente := fuentePruebaConPosiciones(3)
	fuente.posiciones["bolsa:administrativo:2026-09-17"][1].DocumentoEnmascarado = "12345678A"
	w := servirPublico(t, fuente, http.MethodGet, RutaBolsasPublicas+"/bolsa:administrativo:2026-09-17/lista")
	if w.Code != http.StatusInternalServerError || strings.Contains(w.Body.String(), "12345678A") {
		t.Fatalf("un documento sin enmascarar no debe salir: %d %s", w.Code, w.Body.String())
	}
	fuente = fuentePruebaConPosiciones(3)
	fuente.posiciones["bolsa:administrativo:2026-09-17"][2].EstadoClave = "otro"
	w = servirPublico(t, fuente, http.MethodGet, RutaBolsasPublicas+"/bolsa:administrativo:2026-09-17/lista")
	if w.Code != http.StatusInternalServerError {
		t.Fatalf("una situación desconocida no debe salir: %d", w.Code)
	}
}

func TestBolsasPublicasHead(t *testing.T) {
	w := servirPublico(t, fuentePruebaConPosiciones(3), http.MethodHead, RutaBolsasPublicas)
	if w.Code != http.StatusOK || w.Body.Len() != 0 {
		t.Fatalf("HEAD: %d %d", w.Code, w.Body.Len())
	}
}
