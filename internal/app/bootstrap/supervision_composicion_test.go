package bootstrap

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"vec-diputacion-granada/config"
	vecdomain "vec-diputacion-granada/internal/vec/domain"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

type emisorComposicionPrueba struct {
	mu          sync.Mutex
	solicitudes []vecdomain.SolicitudIncidenciaTecnica
}

func (e *emisorComposicionPrueba) Emitir(s vecdomain.SolicitudIncidenciaTecnica) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.solicitudes = append(e.solicitudes, s)
}

// P2-6: el emisor de incidencias viaja por constructor; la composición que
// construye la API VEC y la raíz supervisada fallan si se les pasa nil, en
// vez de perder en silencio las incidencias específicas.
func TestComposicionRechazaEmisorIncidenciasNil(t *testing.T) {
	if api, err := newVECShellAPICompuestaConIdentidadYRutas(config.Config{PersonalCatalogPath: "memory"}, nil, nil, nil, nil, nil, nil, nil); api != nil || !errors.Is(err, ErrEmisorIncidenciasRequerido) {
		t.Fatalf("composición de la API VEC con emisor nil = (%v, %v)", api, err)
	}
	if srv, err := NuevoServidorHTTPSupervisado(config.Config{}, nil); srv != nil || !errors.Is(err, ErrEmisorIncidenciasRequerido) {
		t.Fatalf("raíz supervisada con emisor nil = (%v, %v)", srv, err)
	}
}

// La composición entrega al manejador cartográfico el emisor recibido: un
// OSRM caído declara OSRM_NO_DISPONIBLE en ese emisor y marca la petición.
func TestComposicionInyectaElEmisorEnLaCartografia(t *testing.T) {
	osrm := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer osrm.Close()
	emisor := &emisorComposicionPrueba{}
	manejador, err := nuevoManejadorProductivoCalculoRutas(config.Config{
		OSRMBaseURL: osrm.URL, OSRMScopeName: "Granada provincia + 15 km",
		OSRMScopeBounds: "36.45,-4.6,38.25,-2.15", OSRMAllowedCIDRs: []string{"127.0.0.1/32"},
		OSRMGraphVersion: "grafo-osm-granada-prueba-v1",
	}, emisor)
	if err != nil || manejador == nil {
		t.Fatalf("manejador cartográfico = (%v, %v)", manejador, err)
	}
	ctx, declarada := vecports.ConMarcaIncidenciasPeticion(t.Context())
	peticion := httptest.NewRequest(http.MethodPost, "/api/vec/dietas/road-route",
		strings.NewReader(`{"coordinates":[{"lat":37.1773,"lon":-3.5986},{"lat":37.2306,"lon":-3.6554}]}`)).WithContext(ctx)
	peticion.Header.Set("Content-Type", "application/json; charset=utf-8")
	respuesta := httptest.NewRecorder()
	manejador.ServeHTTP(respuesta, peticion)
	if respuesta.Code != http.StatusBadGateway {
		t.Fatalf("estado = %d: %s", respuesta.Code, respuesta.Body.String())
	}
	emisor.mu.Lock()
	defer emisor.mu.Unlock()
	if len(emisor.solicitudes) != 1 || emisor.solicitudes[0].Codigo != vecdomain.IncidenciaOSRMNoDisponible || !declarada() {
		t.Fatalf("incidencias = %v, marca = %v", emisor.solicitudes, declarada())
	}
}
