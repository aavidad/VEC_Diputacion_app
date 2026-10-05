package adminselector

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

// Con la administración publicada en un puerto propio el origen y la cabecera
// Host llevan ese puerto, mientras la observación de la frontera conserva el
// nombre sin puerto (host_admin de la política de certificado).
func TestSelectorConPuertoPublicoExigeHostYOriginExactos(t *testing.T) {
	const origen = "https://admin.example.invalid:8444"
	casos := map[string]struct {
		cambiar  func(*http.Request)
		estado   int
		observar bool
	}{
		"host y origin con puerto":    {func(*http.Request) {}, 200, true},
		"host sin puerto":             {func(r *http.Request) { r.Host = "admin.example.invalid" }, 401, false},
		"host puerto por defecto":     {func(r *http.Request) { r.Host = "admin.example.invalid:443" }, 401, false},
		"host otro puerto":            {func(r *http.Request) { r.Host = "admin.example.invalid:8443" }, 401, false},
		"origin sin puerto":           {func(r *http.Request) { r.Header.Set("Origin", "https://admin.example.invalid") }, 403, false},
		"origin otro puerto":          {func(r *http.Request) { r.Header.Set("Origin", "https://admin.example.invalid:8443") }, 403, false},
		"origin puerto por defecto":   {func(r *http.Request) { r.Header.Set("Origin", "https://admin.example.invalid:443") }, 403, false},
		"origin http mismo puerto":    {func(r *http.Request) { r.Header.Set("Origin", "http://admin.example.invalid:8444") }, 403, false},
		"origin con barra":            {func(r *http.Request) { r.Header.Set("Origin", origen+"/") }, 403, false},
		"origin otro host con puerto": {func(r *http.Request) { r.Header.Set("Origin", "https://otro.example.invalid:8444") }, 403, false},
	}
	for nombre, caso := range casos {
		t.Run(nombre, func(t *testing.T) {
			_, o, s, a := escenario(t)
			h, err := NuevoHandler(origen, "vec.admin.perfiles.v1", o, s, a, relojPrueba{})
			if err != nil {
				t.Fatal(err)
			}
			r := peticion(http.MethodPost, RutaSeleccion, cuerpoValido())
			r.Host = "admin.example.invalid:8444"
			r.Header.Set("Origin", origen)
			caso.cambiar(r)
			w := httptest.NewRecorder()
			h.ServeHTTP(w, r)
			if w.Code != caso.estado || (o.llamadas != 0) != caso.observar || (s.escrituras == 1) != (caso.estado == 200) {
				t.Fatalf("estado=%d observador=%d selector=%d", w.Code, o.llamadas, s.escrituras)
			}
		})
	}
}

func TestSelectorConPuertoPublicoRechazaObservacionConPuerto(t *testing.T) {
	_, o, s, a := escenario(t)
	o.observado.Host = "admin.example.invalid:8444"
	h, err := NuevoHandler("https://admin.example.invalid:8444", "vec.admin.perfiles.v1", o, s, a, relojPrueba{})
	if err != nil {
		t.Fatal(err)
	}
	r := peticion(http.MethodGet, RutaPropios, "")
	r.Host = "admin.example.invalid:8444"
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != 401 || s.lecturas != 0 {
		t.Fatalf("observación con puerto aceptada: %d", w.Code)
	}
}
