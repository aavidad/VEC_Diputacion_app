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

func TestGobiernoRolAuditaFalloAntesDelConsumoYRespetaIntentoConfirmado(t *testing.T) {
	sesion := sesionAplicacionNominalPrueba(t)
	auditor := &auditorPrueba{}
	h := &Handler{auditor: auditor}
	ref := "propuesta_admin:" + strings.Repeat("a", 32)
	peticion := httptest.NewRequest(http.MethodPost, "https://admin.example.test"+RutaGobiernoRolProponer, nil)
	w := httptest.NewRecorder()
	h.responderErrorGobiernoRol(w, peticion, sesion,
		ports.ErrAutoridadAdministracionPerfilesNoDisponible, ref)
	if w.Code != http.StatusServiceUnavailable || auditor.llamadas != 1 ||
		auditor.ultima.RecursoRef != ref || auditor.ultima.ActorPersonaRef != sesion.Actor.PersonaRef ||
		auditor.ultima.CorrelacionRef != sesion.CorrelacionRef {
		t.Fatal("fallo previo al PDP sin auditoría nominal actual")
	}
	w = httptest.NewRecorder()
	h.responderErrorGobiernoRol(w, peticion, sesion,
		errors.Join(ports.ErrGobiernoRolIntentoAuditado, domain.ErrAutorizacionDenegada), ref)
	if w.Code != http.StatusForbidden || auditor.llamadas != 1 {
		t.Fatal("intento SQL ya confirmado se auditó de nuevo o perdió 403")
	}
	auditor.err = errors.New("auditoria_caida")
	w = httptest.NewRecorder()
	h.responderErrorGobiernoRol(w, peticion, sesion,
		ports.ErrAutoridadAdministracionPerfilesNoDisponible, ref)
	if w.Code != http.StatusServiceUnavailable || auditor.llamadas != 2 {
		t.Fatal("caída de auditoría previa al PDP aparentó acceso registrado")
	}
}

func TestGobiernoRolSinOverlayNoMontaRuta(t *testing.T) {
	sesion := &sesionPrueba{resultado: sesionAplicacionNominalPrueba(t)}
	auditor := &auditorPrueba{}
	h, err := NuevoHandlerUsuariosMetadatos("https://admin.example.test", sesion, &lecturasPrueba{}, auditor)
	if err != nil {
		t.Fatal(err)
	}
	w := httptest.NewRecorder()
	h.ServeHTTP(w, peticionADMIN(http.MethodPost, RutaGobiernoRolProponer, `{}`))
	if w.Code != http.StatusNotFound || auditor.llamadas != 1 ||
		auditor.ultima.Codigo != "recurso_no_encontrado" {
		t.Fatal("ruta Gov ausente respondió como servicio montado")
	}
}
