package main

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"vec-diputacion-granada/internal/modules/bolsa/adapters/simuladorlocal"
)

// El transporte selecciona entradas embebidas. No recibe personas, notas ni
// ficheros; un nuevo campo no puede eludir ese límite a través del nuevo POST.
func TestSeleccionLocalEntradaCerradaYRepetible(t *testing.T) {
	h := nuevoHandler(hostPrueba, nil)
	w := request(h, http.MethodGet, "/api/seleccion/v1/ensayos", "", nil)
	var catalogo struct {
		Ejemplos []struct {
			Referencia    string          `json:"referencia"`
			Configuracion json.RawMessage `json:"configuracion"`
		} `json:"ejemplos"`
	}
	if w.Code != http.StatusOK || json.Unmarshal(w.Body.Bytes(), &catalogo) != nil || len(catalogo.Ejemplos) != 3 {
		t.Fatalf("catálogo de modalidades no disponible: %d", w.Code)
	}
	for _, ejemplo := range catalogo.Ejemplos {
		t.Run(ejemplo.Referencia, func(t *testing.T) {
			body, err := json.Marshal(map[string]any{"ejemplo_ref": ejemplo.Referencia, "configuracion": ejemplo.Configuracion})
			if err != nil {
				t.Fatal(err)
			}
			primera := request(h, http.MethodPost, "/api/seleccion/v1/simulaciones", string(body), nil)
			segunda := request(h, http.MethodPost, "/api/seleccion/v1/simulaciones", string(body), nil)
			if primera.Code != http.StatusOK || segunda.Code != http.StatusOK || primera.Body.String() != segunda.Body.String() {
				t.Fatalf("ensayo no reproducible: %d/%d", primera.Code, segunda.Code)
			}
			if primera.Header().Get("Cache-Control") != "no-store" || primera.Header().Get("Set-Cookie") != "" || primera.Header().Get("Access-Control-Allow-Origin") != "" {
				t.Fatal("almacenamiento HTTP indebido")
			}
			for _, campo := range []string{`,"solicitudes":[]`, `,"persona_ref":"ajena"`, `,"notas":[]`, `,"archivo":"otro.json"`} {
				alterada := strings.TrimSuffix(string(body), "}") + campo + "}"
				if got := request(h, http.MethodPost, "/api/seleccion/v1/simulaciones", alterada, nil); got.Code != http.StatusBadRequest {
					t.Fatalf("hechos externos admitidos: %d", got.Code)
				}
			}
		})
	}
}

func TestSeleccionLocalConservaFrontera(t *testing.T) {
	h := nuevoHandler(hostPrueba, nil)
	for _, caso := range []struct {
		nombre, metodo, ruta string
		estado               int
		cambiar              func(*http.Request)
	}{
		{"origen", http.MethodPost, "/api/seleccion/v1/simulaciones", http.StatusForbidden, func(r *http.Request) { r.Header.Del("Origin") }},
		{"origen_ajeno", http.MethodPost, "/api/seleccion/v1/simulaciones", http.StatusForbidden, func(r *http.Request) { r.Header.Set("Origin", "http://ajeno.example") }},
		{"origen_multiple", http.MethodPost, "/api/seleccion/v1/simulaciones", http.StatusForbidden, func(r *http.Request) { r.Header.Add("Origin", "http://"+hostPrueba) }},
		{"cross_site", http.MethodGet, "/api/seleccion/v1/ensayos", http.StatusForbidden, func(r *http.Request) { r.Header.Set("Sec-Fetch-Site", "cross-site") }},
		{"cookie", http.MethodGet, "/api/seleccion/v1/ensayos", http.StatusForbidden, func(r *http.Request) { r.Header.Set("Cookie", "sesion=sintetica") }},
		{"credencial", http.MethodGet, "/api/seleccion/v1/ensayos", http.StatusForbidden, func(r *http.Request) { r.Header.Set("Authorization", "Bearer sintetico") }},
		{"host", http.MethodGet, "/api/seleccion/v1/ensayos", http.StatusForbidden, func(r *http.Request) { r.Host = "ajeno.example" }},
		{"metodo", http.MethodGet, "/api/seleccion/v1/simulaciones", http.StatusMethodNotAllowed, nil},
		{"metodo_catalogo", http.MethodPost, "/api/seleccion/v1/ensayos", http.StatusMethodNotAllowed, nil},
		{"selector_url", http.MethodGet, "/api/seleccion/v1/ensayos?persona=ajena", http.StatusBadRequest, nil},
		{"selector_post", http.MethodPost, "/api/seleccion/v1/simulaciones?ejemplo_ref=otro", http.StatusBadRequest, nil},
		{"tipo", http.MethodPost, "/api/seleccion/v1/simulaciones", http.StatusUnsupportedMediaType, func(r *http.Request) { r.Header.Set("Content-Type", "text/plain") }},
		{"compresion", http.MethodPost, "/api/seleccion/v1/simulaciones", http.StatusUnsupportedMediaType, func(r *http.Request) { r.Header.Set("Content-Encoding", "gzip") }},
		{"activacion", http.MethodPost, "/api/seleccion/v1/activar", http.StatusNotFound, nil},
		{"solicitudes", http.MethodPost, "/api/seleccion/v1/solicitudes", http.StatusNotFound, nil},
		{"fichero", http.MethodGet, "/seleccion_ensayos.json", http.StatusNotFound, nil},
	} {
		t.Run(caso.nombre, func(t *testing.T) {
			if got := request(h, caso.metodo, caso.ruta, "{}", caso.cambiar); got.Code != caso.estado {
				t.Fatalf("frontera: %d, esperado %d", got.Code, caso.estado)
			}
		})
	}
}

func TestSeleccionLocalLimitaBytesAntesDeDecodificar(t *testing.T) {
	h := nuevoHandler(hostPrueba, nil)
	w := request(h, http.MethodPost, "/api/seleccion/v1/simulaciones", strings.Repeat("x", simuladorlocal.MaximoBytes+1), nil)
	if w.Code != http.StatusRequestEntityTooLarge || strings.TrimSpace(w.Body.String()) != `{"error":"solicitud_invalida"}` {
		t.Fatalf("carga excesiva: %d %s", w.Code, w.Body.String())
	}
}
