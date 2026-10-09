package administracionperfiles

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"vec-diputacion-granada/internal/vec/domain"
	"vec-diputacion-granada/internal/vec/ports"
)

func TestGobiernoInscripcionSinOverlayNoMontaRuta(t *testing.T) {
	sesion := &sesionPrueba{resultado: sesionAplicacionNominalPrueba(t)}
	auditor := &auditorPrueba{}
	h, err := NuevoHandlerUsuariosMetadatos("https://admin.example.test", sesion, &lecturasPrueba{}, auditor)
	if err != nil {
		t.Fatal(err)
	}
	for _, ruta := range []string{RutaGobiernoInscripcionProponer, RutaGobiernoInscripcionCerrar} {
		w := httptest.NewRecorder()
		h.ServeHTTP(w, peticionADMIN(http.MethodPost, ruta, `{}`))
		if w.Code != http.StatusNotFound || auditor.ultima.Codigo != "recurso_no_encontrado" {
			t.Fatalf("ruta sin overlay pareció activa: %s %d", ruta, w.Code)
		}
	}
	if auditor.llamadas != 2 {
		t.Fatal("denegaciones de ruta no auditadas")
	}
}

func TestGobiernoInscripcionDistingueIntentoSQLAuditado(t *testing.T) {
	sesion := sesionAplicacionNominalPrueba(t)
	auditor := &auditorPrueba{}
	h := &Handler{auditor: auditor}
	ref := "propuesta_admin:" + strings.Repeat("a", 32)
	peticion := httptest.NewRequest(http.MethodPost, "https://admin.example.test"+RutaGobiernoInscripcionProponer, nil)
	w := httptest.NewRecorder()
	h.responderErrorGobiernoInscripcion(w, peticion, sesion,
		ports.ErrAutoridadAdministracionPerfilesNoDisponible, ref)
	if w.Code != http.StatusServiceUnavailable || auditor.llamadas != 1 ||
		auditor.ultima.RecursoRef != ref || auditor.ultima.ActorPersonaRef != sesion.Actor.PersonaRef {
		t.Fatal("fallo anterior al consumo perdió auditoría de frontera")
	}
	w = httptest.NewRecorder()
	h.responderErrorGobiernoInscripcion(w, peticion, sesion,
		errors.Join(ports.ErrGobiernoInscripcionIntentoAuditado, domain.ErrAutorizacionDenegada), ref)
	if w.Code != http.StatusForbidden || auditor.llamadas != 1 {
		t.Fatal("intento SQL confirmado se auditó dos veces o perdió denegación")
	}
}
