package main

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"
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
			if primera.Header().Get("Cache-Control") != "no-store" || primera.Header().Get("Set-Cookie") != "" {
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
		{"host", http.MethodGet, "/api/seleccion/v1/ensayos", http.StatusForbidden, func(r *http.Request) { r.Host = "ajeno.example" }},
		{"metodo", http.MethodGet, "/api/seleccion/v1/simulaciones", http.StatusMethodNotAllowed, nil},
		{"selector_url", http.MethodGet, "/api/seleccion/v1/ensayos?persona=ajena", http.StatusBadRequest, nil},
		{"tipo", http.MethodPost, "/api/seleccion/v1/simulaciones", http.StatusUnsupportedMediaType, func(r *http.Request) { r.Header.Set("Content-Type", "text/plain") }},
	} {
		t.Run(caso.nombre, func(t *testing.T) {
			if got := request(h, caso.metodo, caso.ruta, "{}", caso.cambiar); got.Code != caso.estado {
				t.Fatalf("frontera: %d, esperado %d", got.Code, caso.estado)
			}
		})
	}
}
