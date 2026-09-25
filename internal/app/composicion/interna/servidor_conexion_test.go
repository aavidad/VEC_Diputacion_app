package interna

import (
	"net/http"
	"testing"
)

func TestCabecerasSinConexionPersistente(t *testing.T) {
	for _, valor := range []string{"keep-alive", "close", "Keep-Alive", "keep-alive, close"} {
		h := http.Header{"Connection": {valor}, "Accept": {"application/json"}}
		limpia, ok := cabecerasSinConexionPersistente(h)
		if !ok || limpia.Get("Connection") != "" || limpia.Get("Accept") != "application/json" {
			t.Fatalf("%q no normalizada", valor)
		}
		if h.Get("Connection") == "" {
			t.Fatal("se mutó la cabecera original")
		}
	}
	for _, h := range []http.Header{
		{"Connection": {"X-Actor"}},
		{"Connection": {"keep-alive, X-Vec-Perfil"}},
		{"Connection": {"upgrade"}},
		{"Connection": {"keep-alive"}, "Keep-Alive": {"timeout=5"}},
	} {
		if _, ok := cabecerasSinConexionPersistente(h); ok {
			t.Fatalf("Connection con salto admitida: %v", h)
		}
	}
	h := http.Header{"Accept": {"application/json"}}
	if limpia, ok := cabecerasSinConexionPersistente(h); !ok || limpia.Get("Accept") == "" {
		t.Fatal("petición sin Connection rechazada")
	}
}
