package main

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"vec-diputacion-granada/internal/modules/bolsa/adapters/simuladorlocal"
	"vec-diputacion-granada/internal/modules/seleccion/adapters/simulacion"
	"vec-diputacion-granada/internal/modules/seleccion/domain"
)

// El transporte selecciona entradas embebidas y admite solo notas sintéticas
// por referencia. No recibe personas ni ficheros.
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

func TestSeleccionLocalNotasEditablesRecalculanSinPersistir(t *testing.T) {
	h := nuevoHandler(hostPrueba, nil)
	w := request(h, http.MethodGet, "/api/seleccion/v1/ensayos", "", nil)
	var catalogo struct {
		Ejemplos []simulacion.Ejemplo `json:"ejemplos"`
	}
	if w.Code != http.StatusOK || json.Unmarshal(w.Body.Bytes(), &catalogo) != nil {
		t.Fatal("catálogo no disponible")
	}
	ejemplo := catalogo.Ejemplos[0]
	if len(ejemplo.NotasPrueba) != 8 || ejemplo.NotasPrueba[0].Nombre == "" || ejemplo.NotasPrueba[0].SolicitudRef != "solicitud_1" || ejemplo.NotasPrueba[0].FaseRef != "ejercicio_1" || *ejemplo.NotasPrueba[0].PuntosMicropuntos != 8_000_000 || len(catalogo.Ejemplos[1].NotasPrueba) != 0 {
		t.Fatal("catálogo incompleto o con méritos editables")
	}
	base := map[string]any{"ejemplo_ref": ejemplo.Referencia, "configuracion": ejemplo.Configuracion}
	post := func() string {
		t.Helper()
		body, err := json.Marshal(base)
		if err != nil {
			t.Fatal(err)
		}
		return string(body)
	}
	original := request(h, http.MethodPost, "/api/seleccion/v1/simulaciones", post(), nil)
	nota := int64(10_000_000)
	base["notas_prueba"] = []simulacion.NotaPrueba{{SolicitudRef: "solicitud_2", FaseRef: "ejercicio_1", PuntosMicropuntos: &nota}}
	editada := request(h, http.MethodPost, "/api/seleccion/v1/simulaciones", post(), nil)
	repetida := request(h, http.MethodPost, "/api/seleccion/v1/simulaciones", post(), nil)
	var resultado domain.Resultado
	if original.Code != http.StatusOK || editada.Code != http.StatusOK || repetida.Code != http.StatusOK || repetida.Body.String() != editada.Body.String() || json.Unmarshal(editada.Body.Bytes(), &resultado) != nil {
		t.Fatal("edición no repetible")
	}
	segunda := resultado.Solicitudes[1]
	if resultado.Alcance != "ensayo_sintetico" || segunda.Nombre != ejemplo.NotasPrueba[2].Nombre || segunda.Orden == nil || *segunda.Orden != 1 || segunda.TotalMicropuntos == nil || *segunda.TotalMicropuntos != 8_000_000 || segunda.Fases[0].Origen != "prueba_editada" || segunda.Fases[1].Origen != "prueba_embebida" {
		t.Fatalf("edición no recalculada o nombre sustituido: %+v", segunda)
	}
	base["notas_prueba"] = []simulacion.NotaPrueba{{SolicitudRef: "solicitud_2", FaseRef: "ejercicio_1", PuntosMicropuntos: nil}}
	pendiente := request(h, http.MethodPost, "/api/seleccion/v1/simulaciones", post(), nil)
	if pendiente.Code != http.StatusOK || json.Unmarshal(pendiente.Body.Bytes(), &resultado) != nil || resultado.Estado != "indeterminado" || resultado.Solicitudes[1].TotalMicropuntos != nil || resultado.Solicitudes[1].Orden != nil || resultado.Solicitudes[1].Fases[0].Origen != "prueba_editada" || resultado.Solicitudes[1].Fases[0].Estado != "pendiente" {
		t.Fatal("la nota pendiente produjo un aprobado o perdió procedencia")
	}
	delete(base, "notas_prueba")
	restaurada := request(h, http.MethodPost, "/api/seleccion/v1/simulaciones", post(), nil)
	if restaurada.Body.String() != original.Body.String() {
		t.Fatal("una edición contaminó el siguiente ensayo")
	}
	if editada.Header().Get("Cache-Control") != "no-store" || editada.Header().Get("Set-Cookie") != "" || editada.Header().Get("Access-Control-Allow-Origin") != "" {
		t.Fatal("se alteró la frontera de almacenamiento u origen")
	}
}

func TestSeleccionLocalNotasRechazaHechosAjenosYFormaAmbigua(t *testing.T) {
	h := nuevoHandler(hostPrueba, nil)
	ejemplos, err := simulacion.Ejemplos()
	if err != nil {
		t.Fatal(err)
	}
	e := ejemplos[2]
	base, err := json.Marshal(map[string]any{"ejemplo_ref": e.Referencia, "configuracion": e.Configuracion})
	if err != nil {
		t.Fatal(err)
	}
	for _, lista := range []string{
		`null`,
		`[{"solicitud_ref":"solicitud_1","fase_ref":"ejercicio_1"}]`,
		`[{"solicitud_ref":"solicitud_1","fase_ref":"ejercicio_1","puntos_micropuntos":1.5}]`,
		`[{"solicitud_ref":"solicitud_1","fase_ref":"ejercicio_1","puntos_micropuntos":-1}]`,
		`[{"solicitud_ref":"solicitud_1","fase_ref":"ejercicio_1","puntos_micropuntos":10000001}]`,
		`[{"solicitud_ref":"solicitud_1","fase_ref":"meritos","puntos_micropuntos":0}]`,
		`[{"solicitud_ref":"solicitud_1","fase_ref":"ejercicio_2","puntos_micropuntos":0}]`,
		`[{"solicitud_ref":"desconocida","fase_ref":"ejercicio_1","puntos_micropuntos":0}]`,
		`[{"solicitud_ref":"ſolicitud_1","fase_ref":"ejercicio_1","puntos_micropuntos":0}]`,
		`[{"solicitud_ref":"solicitud_1","fase_ref":"ejercicio_1","puntos_micropuntos":0,"nombre":"Otra"}]`,
		`[{"solicitud_ref":"solicitud_1","fase_ref":"ejercicio_1","puntos_micropuntos":0,"persona_ref":"otra"}]`,
		`[{"solicitud_ref":"solicitud_1","fase_ref":"ejercicio_1","puntoſ_micropuntos":0}]`,
		`[{"solicitud_ref":"solicitud_1","fase_ref":"ejercicio_1","puntos_micropuntos":0,"puntos_micropuntos":1}]`,
		`[{"solicitud_ref":"solicitud_1","fase_ref":"ejercicio_1","puntos_micropuntos":0},{"solicitud_ref":"solicitud_1","fase_ref":"ejercicio_1","puntos_micropuntos":1}]`,
	} {
		body := strings.TrimSuffix(string(base), "}") + `,"notas_prueba":` + lista + "}"
		w := request(h, http.MethodPost, "/api/seleccion/v1/simulaciones", body, nil)
		if w.Code != http.StatusBadRequest || strings.TrimSpace(w.Body.String()) != `{"error":"solicitud_invalida"}` {
			t.Fatalf("nota inválida no rechazada: %s => %d %s", lista, w.Code, w.Body.String())
		}
	}
}
