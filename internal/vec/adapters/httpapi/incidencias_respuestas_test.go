package httpapi

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	adminmodule "vec-diputacion-granada/internal/modules/administracion"
	"vec-diputacion-granada/internal/vec/adapters/memory"
	"vec-diputacion-granada/internal/vec/application"
	"vec-diputacion-granada/internal/vec/domain"
	"vec-diputacion-granada/internal/vec/ports"
)

// errSensiblePrueba simula el texto de un error de biblioteca que nunca debe
// llegar al cliente.
var errSensiblePrueba = errors.New("postgres://vec:clave@10.0.0.5/vec manifiesto /srv/privado 12345678Z")

type emisorContextoPrueba struct {
	mu          sync.Mutex
	solicitudes []domain.SolicitudIncidenciaTecnica
}

func (e *emisorContextoPrueba) Emitir(s domain.SolicitudIncidenciaTecnica) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.solicitudes = append(e.solicitudes, s)
}

func (e *emisorContextoPrueba) unica(t *testing.T) domain.SolicitudIncidenciaTecnica {
	t.Helper()
	e.mu.Lock()
	defer e.mu.Unlock()
	if len(e.solicitudes) != 1 {
		t.Fatalf("incidencias = %v, se esperaba una", e.solicitudes)
	}
	if _, saneada := domain.ClasificarIncidenciaTecnica(e.solicitudes[0]); saneada {
		t.Fatalf("incidencia no catalogada: %v", e.solicitudes[0])
	}
	return e.solicitudes[0]
}

type registroModulosFallido struct{ *memory.Store }

func (registroModulosFallido) ListModules(context.Context) ([]domain.ModuleManifest, error) {
	return nil, errSensiblePrueba
}

type auditoriaFallida struct{ *memory.Store }

func (auditoriaFallida) AppendAudit(context.Context, domain.AuditEntry) (domain.AuditEntry, error) {
	return domain.AuditEntry{}, errSensiblePrueba
}

func nuevoHandlerConAlmacenes(t *testing.T, modulos ports.ModuleRegistryStore, auditoria ports.AuditStore, eventos ports.EventStore) *Handler {
	t.Helper()
	service, internal, err := application.NewServiceWithInternalOperations(modulos, auditoria, eventos)
	if err != nil {
		t.Fatal(err)
	}
	handler, err := NewHandlerWithOptions(service, HandlerOptions{
		InternalOperations: internal, AllowDemoIdentity: true, DemoIdentityResolver: resolvedorIdentidadPruebas{},
	})
	if err != nil {
		t.Fatal(err)
	}
	return handler
}

func peticionConEmisor(metodo, ruta string, emisor ports.EmisorIncidenciasTecnicas) *http.Request {
	peticion := httptest.NewRequest(metodo, ruta, nil)
	return peticion.WithContext(ports.ConEmisorIncidenciasTecnicas(peticion.Context(), emisor))
}

func comprobarSinTextoInterno(t *testing.T, cuerpo string) {
	t.Helper()
	for _, sensible := range []string{"postgres", "10.0.0.5", "/srv/privado", "12345678Z", "clave"} {
		if strings.Contains(cuerpo, sensible) {
			t.Fatalf("la respuesta contiene texto interno %q: %s", sensible, cuerpo)
		}
	}
}

func TestCatalogoModulosInvalidoRespondeCodigoFijoYDeclaraIncidencia(t *testing.T) {
	store := memory.NewStore()
	handler := nuevoHandlerConAlmacenes(t, registroModulosFallido{store}, store, store)
	for _, ruta := range []string{"/api/vec/modules", "/api/vec/menu"} {
		emisor := &emisorContextoPrueba{}
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, peticionConEmisor(http.MethodGet, ruta, emisor))
		if rec.Code != http.StatusInternalServerError || strings.TrimSpace(rec.Body.String()) != `{"error":"catalogo_modulos_no_disponible"}` {
			t.Fatalf("%s: %d %s", ruta, rec.Code, rec.Body.String())
		}
		comprobarSinTextoInterno(t, rec.Body.String())
		got := emisor.unica(t)
		if got.Codigo != domain.IncidenciaCatalogoModulosInvalido || got.Componente != domain.ComponenteIncidenciaCatalogoModulos || got.Etapa != domain.EtapaIncidenciaValidacion {
			t.Fatalf("%s: incidencia %v", ruta, got)
		}
	}
}

func TestAccionDeModuloConAuditoriaCaidaResponde503YDeclaraIncidencia(t *testing.T) {
	store := memory.NewStore()
	handler := nuevoHandlerConAlmacenes(t, store, auditoriaFallida{store}, store)
	if err := handler.internal.RegisterModule(context.Background(), adminmodule.Manifest()); err != nil {
		t.Fatal(err)
	}
	emisor := &emisorContextoPrueba{}
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, peticionConEmisor(http.MethodPost, "/api/vec/modules/administracion/action", emisor))
	if rec.Code != http.StatusServiceUnavailable || strings.TrimSpace(rec.Body.String()) != `{"error":"auditoria_no_disponible"}` {
		t.Fatalf("respuesta: %d %s", rec.Code, rec.Body.String())
	}
	comprobarSinTextoInterno(t, rec.Body.String())
	if got := emisor.unica(t); got.Codigo != domain.IncidenciaAuditoriaNoRegistrada {
		t.Fatalf("incidencia %v", got)
	}
}

func TestSinEmisorEnContextoLaRespuestaEsLaMisma(t *testing.T) {
	store := memory.NewStore()
	handler := nuevoHandlerConAlmacenes(t, registroModulosFallido{store}, store, store)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/vec/modules", nil))
	if rec.Code != http.StatusInternalServerError || strings.TrimSpace(rec.Body.String()) != `{"error":"catalogo_modulos_no_disponible"}` {
		t.Fatalf("respuesta: %d %s", rec.Code, rec.Body.String())
	}
}
