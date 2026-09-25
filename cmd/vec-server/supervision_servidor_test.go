package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
)

type bufferSincronizado struct {
	mu sync.Mutex
	b  bytes.Buffer
}

func (s *bufferSincronizado) Write(p []byte) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.b.Write(p)
}

func (s *bufferSincronizado) String() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.b.String()
}

func TestComponerSupervisionServidorRegistraRespuesta503SinDatosDeLaPeticion(t *testing.T) {
	t.Setenv(envEntornoSupervision, "desarrollo")
	destino := &bufferSincronizado{}
	registro := &bufferSincronizado{}
	srv := &http.Server{Handler: http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "servicio_no_disponible", http.StatusServiceUnavailable)
	})}
	cerrar := componerSupervisionServidor(srv, destino, registro)

	respuesta := httptest.NewRecorder()
	peticion := httptest.NewRequest(http.MethodGet, "/api/vec/personas/12345678Z?correo=ana@example.org", nil)
	peticion.RemoteAddr = "203.0.113.9:4444"
	peticion.Header.Set("X-Marcador", "marcador-sensible")
	srv.Handler.ServeHTTP(respuesta, peticion)
	cerrar()
	cerrar()

	if respuesta.Code != http.StatusServiceUnavailable || strings.TrimSpace(respuesta.Body.String()) != "servicio_no_disponible" {
		t.Fatalf("respuesta alterada: %d %q", respuesta.Code, respuesta.Body.String())
	}
	texto := destino.String()
	for _, sensible := range []string{"12345678Z", "ana@example.org", "203.0.113.9", "marcador-sensible", "/api/vec"} {
		if strings.Contains(texto, sensible) {
			t.Fatalf("la incidencia contiene %q: %s", sensible, texto)
		}
	}
	lineas := strings.Split(strings.TrimSuffix(texto, "\n"), "\n")
	if len(lineas) != 1 {
		t.Fatalf("se esperaba una incidencia: %q", texto)
	}
	var campos map[string]any
	if err := json.Unmarshal([]byte(lineas[0]), &campos); err != nil {
		t.Fatal(err)
	}
	if campos["codigo"] != "HTTP_INTERNO_FALLIDO" || campos["componente"] != "http" || campos["etapa"] != "peticion" || campos["entorno"] != "desarrollo" {
		t.Fatalf("incidencia inesperada: %v", campos)
	}
	if correlacion, _ := campos["correlacion"].(string); len(correlacion) != 32 {
		t.Fatalf("correlación técnica ausente: %v", campos)
	}
	if srv.ErrorLog == nil {
		t.Fatal("ErrorLog sin sanear")
	}
	if registro.String() != "" {
		t.Fatalf("registro inesperado: %q", registro.String())
	}
}
