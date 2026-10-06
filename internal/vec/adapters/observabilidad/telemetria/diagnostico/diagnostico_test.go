package diagnostico

import (
	"context"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// tokenPrueba es sintético y de baja entropía, solo para estas pruebas.
var tokenPrueba = strings.Repeat("prueba-", 6)

func peticion(metodo, ruta, token, remota string) *http.Request {
	r := httptest.NewRequest(metodo, ruta, nil)
	r.RemoteAddr = remota
	if token != "" {
		r.Header.Set("Authorization", "Bearer "+token)
	}
	return r
}

func TestManejadorDeniegaPorDefecto(t *testing.T) {
	h := Manejador([]byte(tokenPrueba), []Fuente{func(w io.Writer) { _, _ = io.WriteString(w, "vec_prueba 1\n") }})
	casos := []struct {
		nombre string
		r      *http.Request
		estado int
	}{
		{"remota no local", peticion(http.MethodGet, "/metrics", tokenPrueba, "192.0.2.10:5000"), http.StatusForbidden},
		{"sin token", peticion(http.MethodGet, "/metrics", "", "127.0.0.1:5000"), http.StatusUnauthorized},
		{"token ajeno", peticion(http.MethodGet, "/metrics", tokenPrueba+"x", "127.0.0.1:5000"), http.StatusUnauthorized},
		{"metodo", peticion(http.MethodPost, "/metrics", tokenPrueba, "127.0.0.1:5000"), http.StatusMethodNotAllowed},
		{"ruta ajena", peticion(http.MethodGet, "/api/vec/x", tokenPrueba, "127.0.0.1:5000"), http.StatusNotFound},
		{"cpu fuera de rango", peticion(http.MethodGet, "/debug/pprof/profile?seconds=600", tokenPrueba, "127.0.0.1:5000"), http.StatusBadRequest},
		{"metricas", peticion(http.MethodGet, "/metrics", tokenPrueba, "[::1]:5000"), http.StatusOK},
		{"memoria", peticion(http.MethodGet, "/debug/pprof/heap", tokenPrueba, "127.0.0.1:5000"), http.StatusOK},
		{"gorrutinas", peticion(http.MethodGet, "/debug/pprof/goroutine?debug=1", tokenPrueba, "127.0.0.1:5000"), http.StatusOK},
	}
	for _, c := range casos {
		w := httptest.NewRecorder()
		h.ServeHTTP(w, c.r)
		if w.Code != c.estado {
			t.Errorf("%s: estado %d, se esperaba %d", c.nombre, w.Code, c.estado)
		}
		if w.Header().Get("Set-Cookie") != "" || w.Header().Get("Cache-Control") != "no-store" {
			t.Errorf("%s: cabeceras %v", c.nombre, w.Header())
		}
		if c.nombre == "metricas" && w.Body.String() != "vec_prueba 1\n" {
			t.Errorf("metricas = %q", w.Body.String())
		}
	}
}

func TestLeerTokenExigePermisosYLongitud(t *testing.T) {
	dir := t.TempDir()
	escribir := func(nombre, contenido string, modo os.FileMode) string {
		ruta := filepath.Join(dir, nombre)
		if err := os.WriteFile(ruta, []byte(contenido), modo); err != nil {
			t.Fatal(err)
		}
		if err := os.Chmod(ruta, modo); err != nil {
			t.Fatal(err)
		}
		return ruta
	}
	if _, err := LeerToken(escribir("bueno", tokenPrueba+"\n", 0o600)); err != nil {
		t.Errorf("token valido rechazado: %v", err)
	}
	if _, err := LeerToken(escribir("legible", tokenPrueba, 0o644)); err == nil {
		t.Error("token legible por otros admitido")
	}
	if _, err := LeerToken(escribir("corto", "corto", 0o600)); err == nil {
		t.Error("token corto admitido")
	}
	enlace := filepath.Join(dir, "enlace")
	if err := os.Symlink(filepath.Join(dir, "bueno"), enlace); err != nil {
		t.Fatal(err)
	}
	if _, err := LeerToken(enlace); err == nil {
		t.Error("enlace simbolico admitido")
	}
	if _, err := LeerToken(""); err == nil {
		t.Error("ruta vacia admitida")
	}
}

func TestArrancarSoloEnBucleLocal(t *testing.T) {
	for _, escucha := range []string{"0.0.0.0:9464", ":9464", "192.0.2.1:9464", "localhost:9464", "127.0.0.1:0"} {
		if _, err := Arrancar(Configuracion{Escucha: escucha, Token: []byte(tokenPrueba)}); err != ErrEscuchaNoLocal {
			t.Errorf("%q: err = %v", escucha, err)
		}
	}
	libre, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	direccion := libre.Addr().String()
	_ = libre.Close()
	s, err := Arrancar(Configuracion{Escucha: direccion, Token: []byte(tokenPrueba),
		Fuentes: []Fuente{func(w io.Writer) { _, _ = io.WriteString(w, "vec_prueba 2\n") }}})
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		ctx, c := context.WithTimeout(context.Background(), 2*time.Second)
		defer c()
		_ = s.Cerrar(ctx)
	}()
	r, _ := http.NewRequest(http.MethodGet, "http://"+s.Direccion()+"/metrics", nil)
	r.Header.Set("Authorization", "Bearer "+tokenPrueba)
	respuesta, err := http.DefaultClient.Do(r)
	if err != nil {
		t.Fatal(err)
	}
	defer respuesta.Body.Close()
	cuerpo, _ := io.ReadAll(respuesta.Body)
	if respuesta.StatusCode != http.StatusOK || !strings.Contains(string(cuerpo), "vec_prueba 2") {
		t.Errorf("respuesta %d %q", respuesta.StatusCode, cuerpo)
	}
}

func TestMontarDesdeEntornoApagadoYMalConfigurado(t *testing.T) {
	var avisos strings.Builder
	cerrar := MontarDesdeEntorno(func(string) string { return "" }, "vec-server", nil, &avisos)
	cerrar()
	if avisos.Len() != 0 {
		t.Errorf("apagado dejo avisos: %q", avisos.String())
	}
	entorno := map[string]string{EnvEscucha: "0.0.0.0:9464", EnvTokenFile: "/no/existe"}
	MontarDesdeEntorno(func(k string) string { return entorno[k] }, "vec-server", nil, &avisos)()
	if !strings.Contains(avisos.String(), "fichero de token no valido") || strings.Contains(avisos.String(), "/no/existe") {
		t.Errorf("aviso = %q", avisos.String())
	}
}
