package administracionperfiles

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"vec-diputacion-granada/internal/vec/domain"
	"vec-diputacion-granada/internal/vec/ports"
)

type servicioVersionBolsaExternaPrueba struct {
	propuestas, cierres  int
	propuesta            domain.SolicitudPropuestaVersionarRolBolsaExterna
	cierre               domain.SolicitudCierreVersionarRolBolsaExterna
	respuestaIncoherente bool
}

func (s *servicioVersionBolsaExternaPrueba) ProponerVersionarRolBolsaExterna(_ context.Context,
	solicitud domain.SolicitudPropuestaVersionarRolBolsaExterna) (ports.ResultadoPropuestaVersionarRolBolsaExterna, error) {
	s.propuestas++
	s.propuesta = solicitud
	if s.respuestaIncoherente {
		return ports.ResultadoPropuestaVersionarRolBolsaExterna{}, nil
	}
	return ports.ResultadoPropuestaVersionarRolBolsaExterna{}, domain.ErrAutorizacionDenegada
}

func (s *servicioVersionBolsaExternaPrueba) CerrarVersionarRolBolsaExternaPorReferencia(_ context.Context,
	solicitud domain.SolicitudCierreVersionarRolBolsaExterna) (domain.CierreVersionarRolBolsaExterna, error) {
	s.cierres++
	s.cierre = solicitud
	return domain.CierreVersionarRolBolsaExterna{}, domain.ErrAutorizacionDenegada
}

func handlerVersionBolsaExternaPrueba(t *testing.T) (*Handler, SesionConfiable, *servicioVersionBolsaExternaPrueba, *auditorPrueba) {
	t.Helper()
	sesion := sesionAplicacionNominalPrueba(t)
	auditor := &auditorPrueba{}
	h, err := NuevoHandlerUsuariosMetadatos("https://admin.example.test",
		&sesionPrueba{resultado: sesion}, &lecturasPrueba{}, auditor)
	if err != nil {
		t.Fatal(err)
	}
	servicio := &servicioVersionBolsaExternaPrueba{}
	if err := h.ConVersionarRolBolsaExterna(servicio, relojFocal{}); err != nil {
		t.Fatal(err)
	}
	return h, sesion, servicio, auditor
}

func cuerpoPropuestaVersionBolsaExterna(t *testing.T) map[string]any {
	t.Helper()
	return map[string]any{
		"operacion_ref":          "propuesta_admin:" + strings.Repeat("a", 32),
		"catalogo_ref":           "catalogo:externo",
		"catalogo_version":       1,
		"catalogo_huella_sha256": strings.Repeat("b", 64),
		"base_ref":               domain.RolIDVersionarBolsaExterna + ":v1",
		"base_huella_sha256":     strings.Repeat("c", 64),
		"control_revision":       1,
		"control_huella_sha256":  strings.Repeat("d", 64),
		"selecciones": []any{
			map[string]any{"entrada_ref": "entrada:consultar", "entrada_version": 1, "entrada_huella_sha256": strings.Repeat("e", 64)},
			map[string]any{"entrada_ref": "entrada:ordenar", "entrada_version": 1, "entrada_huella_sha256": strings.Repeat("f", 64)},
		},
		"asignaciones": []any{map[string]any{"asignacion_ref": "asignacion:cas"}},
		"motivo": map[string]any{
			"catalogo_id": "motivos_administracion", "catalogo_version": 1,
			"catalogo_huella_sha256": strings.Repeat("1", 64),
			"entrada_clave":          "motivo_" + strings.Repeat("2", 32),
		},
		"huella_plan_esperada": strings.Repeat("3", 64),
	}
}

func peticionPropuestaVersionBolsaExterna(t *testing.T, h *Handler, cuerpo map[string]any) *httptest.ResponseRecorder {
	t.Helper()
	b, err := json.Marshal(cuerpo)
	if err != nil {
		t.Fatal(err)
	}
	w := httptest.NewRecorder()
	h.ServeHTTP(w, peticionADMIN(http.MethodPost, RutaVersionarRolBolsaExternaProponer, string(b)))
	return w
}

func TestVersionBolsaExternaFronteraSoloAceptaPreimagenesYDosSelecciones(t *testing.T) {
	h, sesion, servicio, auditor := handlerVersionBolsaExternaPrueba(t)
	cuerpo := cuerpoPropuestaVersionBolsaExterna(t)
	for _, campo := range []string{"actor_persona_ref", "candidato_ref", "plan"} {
		cuerpo[campo] = "cliente"
		w := peticionPropuestaVersionBolsaExterna(t, h, cuerpo)
		if w.Code != http.StatusBadRequest || servicio.propuestas != 0 || auditor.ultima.Codigo != "solicitud_invalida" {
			t.Fatalf("campo %q cruzó frontera HTTP: estado %d, llamadas %d", campo, w.Code, servicio.propuestas)
		}
		delete(cuerpo, campo)
	}
	for _, n := range []int{1, 3} {
		original := cuerpo["selecciones"]
		cuerpo["selecciones"] = make([]any, n)
		w := peticionPropuestaVersionBolsaExterna(t, h, cuerpo)
		if w.Code != http.StatusBadRequest || servicio.propuestas != 0 {
			t.Fatalf("%d selecciones llegaron al servicio: %d", n, w.Code)
		}
		cuerpo["selecciones"] = original
	}
	w := peticionPropuestaVersionBolsaExterna(t, h, cuerpo)
	if w.Code != http.StatusForbidden || servicio.propuestas != 1 ||
		servicio.propuesta.Actor.PersonaRef != sesion.Actor.PersonaRef ||
		servicio.propuesta.Actor.PerfilActivoRef != sesion.Actor.PerfilActivoRef ||
		servicio.propuesta.HuellaPlanEsperada != strings.Repeat("3", 64) ||
		len(servicio.propuesta.Intencion.Selecciones) != 2 ||
		servicio.propuesta.Intencion.BaseRef != cuerpo["base_ref"] ||
		servicio.propuesta.Intencion.CatalogoRef != cuerpo["catalogo_ref"] {
		t.Fatalf("propuesta no conserva sesión y preimágenes: estado %d, llamadas %d", w.Code, servicio.propuestas)
	}
}

func TestVersionBolsaExternaFronteraAceptaCincoSeleccionesInscripcion(t *testing.T) {
	h, _, servicio, _ := handlerVersionBolsaExternaPrueba(t)
	cuerpo := cuerpoPropuestaVersionBolsaExterna(t)
	selecciones := make([]any, 5)
	for j := range selecciones {
		selecciones[j] = map[string]any{"entrada_ref": "entrada:inscripcion", "entrada_version": 1,
			"entrada_huella_sha256": strings.Repeat("e", 64)}
	}
	cuerpo["selecciones"] = selecciones
	w := peticionPropuestaVersionBolsaExterna(t, h, cuerpo)
	if w.Code != http.StatusForbidden || servicio.propuestas != 1 ||
		len(servicio.propuesta.Intencion.Selecciones) != 5 {
		t.Fatalf("delta aspirante no llegó al servicio: estado %d, llamadas %d", w.Code, servicio.propuestas)
	}
	cuerpo["selecciones"] = make([]any, 4)
	w = peticionPropuestaVersionBolsaExterna(t, h, cuerpo)
	if w.Code != http.StatusBadRequest || servicio.propuestas != 1 {
		t.Fatal("aceptó mezcla de selecciones")
	}
}

func TestVersionBolsaExternaCierreSoloSesionYReferencia(t *testing.T) {
	h, sesion, servicio, auditor := handlerVersionBolsaExternaPrueba(t)
	cuerpo := map[string]any{
		"operacion_ref":           "cierre_admin:" + strings.Repeat("a", 32),
		"propuesta_ref":           "propuesta_admin:" + strings.Repeat("b", 32),
		"propuesta_huella_sha256": strings.Repeat("c", 64),
		"motivo": map[string]any{
			"catalogo_id": "motivos_administracion", "catalogo_version": 1,
			"catalogo_huella_sha256": strings.Repeat("d", 64),
			"entrada_clave":          "motivo_" + strings.Repeat("e", 32),
		},
	}
	for _, campo := range []string{"aprobador", "decision", "material", "actor_persona_ref"} {
		cuerpo[campo] = "cliente"
		b, err := json.Marshal(cuerpo)
		if err != nil {
			t.Fatal(err)
		}
		w := httptest.NewRecorder()
		h.ServeHTTP(w, peticionADMIN(http.MethodPost, RutaVersionarRolBolsaExternaCerrar, string(b)))
		if w.Code != http.StatusBadRequest || servicio.cierres != 0 || auditor.ultima.Codigo != "solicitud_invalida" {
			t.Fatalf("campo %q cruzó el cierre: estado %d, llamadas %d", campo, w.Code, servicio.cierres)
		}
		delete(cuerpo, campo)
	}
	b, err := json.Marshal(cuerpo)
	if err != nil {
		t.Fatal(err)
	}
	w := httptest.NewRecorder()
	h.ServeHTTP(w, peticionADMIN(http.MethodPost, RutaVersionarRolBolsaExternaCerrar, string(b)))
	if w.Code != http.StatusForbidden || servicio.cierres != 1 ||
		servicio.cierre.Aprobador.PersonaRef != sesion.Actor.PersonaRef ||
		servicio.cierre.Aprobador.PerfilActivoRef != sesion.Actor.PerfilActivoRef ||
		servicio.cierre.Decision != domain.DecisionAprobarPropuestaPerfil ||
		servicio.cierre.PropuestaRef != cuerpo["propuesta_ref"] ||
		strings.Contains(w.Body.String(), "per_") {
		t.Fatalf("cierre no conserva frontera nominal: estado %d, llamadas %d", w.Code, servicio.cierres)
	}
}

func TestVersionBolsaExternaNoConfirmaRespuestaIncoherente(t *testing.T) {
	h, _, servicio, _ := handlerVersionBolsaExternaPrueba(t)
	servicio.respuestaIncoherente = true
	w := peticionPropuestaVersionBolsaExterna(t, h, cuerpoPropuestaVersionBolsaExterna(t))
	if w.Code != http.StatusServiceUnavailable || servicio.propuestas != 1 ||
		strings.Contains(w.Body.String(), "propuesta_ref") ||
		strings.Contains(w.Body.String(), "recibo_ref") {
		t.Fatalf("respuesta incoherente pareció confirmada: %d %s", w.Code, w.Body.String())
	}
}
