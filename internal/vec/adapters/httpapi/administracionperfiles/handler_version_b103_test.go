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

type servicioVersionBolsaB103Prueba struct {
	propuestas, cierres int
	cierre              domain.SolicitudCierreVersionarRolBolsaB103
}

func (s *servicioVersionBolsaB103Prueba) ProponerVersionarRolBolsaB103(context.Context,
	domain.SolicitudPropuestaVersionarRolBolsaB103) (ports.ResultadoPropuestaVersionarRolBolsaB103, error) {
	s.propuestas++
	return ports.ResultadoPropuestaVersionarRolBolsaB103{}, domain.ErrVersionarRolBolsaB103Invalido
}

func (s *servicioVersionBolsaB103Prueba) CerrarVersionarRolBolsaB103PorReferencia(_ context.Context,
	solicitud domain.SolicitudCierreVersionarRolBolsaB103) (domain.CierreVersionarRolBolsaB103, error) {
	s.cierres++
	s.cierre = solicitud
	return domain.CierreVersionarRolBolsaB103{}, domain.ErrAutorizacionDenegada
}

func TestVersionBolsaB103RutaIndependienteYCuerpoSinIdentidad(t *testing.T) {
	sesion := sesionAplicacionNominalPrueba(t)
	auditor := &auditorPrueba{}
	h, err := NuevoHandlerUsuariosMetadatos("https://admin.example.test",
		&sesionPrueba{resultado: sesion}, &lecturasPrueba{}, auditor)
	if err != nil {
		t.Fatal(err)
	}
	servicio := &servicioVersionBolsaB103Prueba{}
	fuente := &fuenteVersionBolsaPrueba{}
	if err := h.ConVersionarRolBolsaB103(servicio, fuente, relojFocal{}); err != nil {
		t.Fatal(err)
	}
	w := httptest.NewRecorder()
	h.ServeHTTP(w, peticionADMIN(http.MethodPost, RutaVersionarRolBolsaProponer, `{}`))
	if w.Code != http.StatusNotFound || auditor.llamadas != 1 {
		t.Fatalf("B1 quedó habilitado por B103: %d", w.Code)
	}
	for i, ruta := range []string{RutaVersionarRolBolsaB103Proponer, RutaVersionarRolBolsaRRHHProponer} {
		w = httptest.NewRecorder()
		h.ServeHTTP(w, peticionADMIN(http.MethodPost, ruta,
			`{"actor_persona_ref":"per_cliente","operacion_ref":"propuesta_admin:`+strings.Repeat("a", 32)+`"}`))
		if w.Code != http.StatusBadRequest || auditor.llamadas != i+2 ||
			auditor.ultima.Codigo != "solicitud_invalida" || fuente.llamadas != 0 || servicio.propuestas != 0 {
			t.Fatalf("identidad del cuerpo llegó al servicio por %s: %d, auditorías %d", ruta, w.Code, auditor.llamadas)
		}
	}
}

func TestVersionBolsaB103ExigeSeleccionesAcotadasAntesDeFuente(t *testing.T) {
	sesion := sesionAplicacionNominalPrueba(t)
	auditor := &auditorPrueba{}
	h, err := NuevoHandlerUsuariosMetadatos("https://admin.example.test",
		&sesionPrueba{resultado: sesion}, &lecturasPrueba{}, auditor)
	if err != nil {
		t.Fatal(err)
	}
	servicio := &servicioVersionBolsaB103Prueba{}
	fuente := &fuenteVersionBolsaPrueba{}
	if err := h.ConVersionarRolBolsaB103(servicio, fuente, relojFocal{}); err != nil {
		t.Fatal(err)
	}
	base := map[string]any{
		"operacion_ref":          "propuesta_admin:" + strings.Repeat("a", 32),
		"catalogo_ref":           "catalogo:b103",
		"catalogo_version":       1,
		"catalogo_huella_sha256": strings.Repeat("b", 64),
		"base_ref":               "rol:bolsa:v6",
		"base_huella_sha256":     strings.Repeat("c", 64),
		"control_revision":       1,
		"control_huella_sha256":  strings.Repeat("d", 64),
		"asignaciones":           []any{map[string]any{}},
	}
	consultas := 0
	for _, n := range []int{0, 1, 2, 3, 4, 5, 512, 513} {
		base["selecciones"] = make([]any, n)
		cuerpo, err := json.Marshal(base)
		if err != nil {
			t.Fatal(err)
		}
		w := httptest.NewRecorder()
		h.ServeHTTP(w, peticionADMIN(http.MethodPost, RutaVersionarRolBolsaRRHHProponer, string(cuerpo)))
		if n == 0 || n > 512 {
			if w.Code != http.StatusBadRequest || fuente.llamadas != consultas {
				t.Fatalf("%d selecciones cruzaron frontera: estado %d, fuente %d", n, w.Code, fuente.llamadas)
			}
			continue
		}
		consultas++
		if w.Code != http.StatusServiceUnavailable || fuente.llamadas != consultas || servicio.propuestas != 0 {
			t.Fatalf("selecciones nominales no llegaron a fuente: estado %d, fuente %d", w.Code, fuente.llamadas)
		}
	}
	if auditor.llamadas != 8 {
		t.Fatalf("faltan auditorías nominales: %d", auditor.llamadas)
	}
}

func TestVersionBolsaB103CierreUsaSoloSesionConfiable(t *testing.T) {
	sesion := sesionAplicacionNominalPrueba(t)
	auditor := &auditorPrueba{}
	h, err := NuevoHandlerUsuariosMetadatos("https://admin.example.test",
		&sesionPrueba{resultado: sesion}, &lecturasPrueba{}, auditor)
	if err != nil {
		t.Fatal(err)
	}
	servicio := &servicioVersionBolsaB103Prueba{}
	if err := h.ConVersionarRolBolsaB103(servicio, &fuenteVersionBolsaPrueba{}, relojFocal{}); err != nil {
		t.Fatal(err)
	}
	cuerpo := `{"operacion_ref":"cierre_admin:` + strings.Repeat("a", 32) +
		`","propuesta_ref":"propuesta_admin:` + strings.Repeat("b", 32) +
		`","propuesta_huella_sha256":"` + strings.Repeat("c", 64) +
		`","motivo":{"catalogo_id":"motivos_administracion","catalogo_version":1,` +
		`"catalogo_huella_sha256":"` + strings.Repeat("d", 64) + `","entrada_clave":"motivo_` + strings.Repeat("e", 32) + `"}}`
	for i, ruta := range []string{RutaVersionarRolBolsaB103Cerrar, RutaVersionarRolBolsaRRHHCerrar} {
		w := httptest.NewRecorder()
		h.ServeHTTP(w, peticionADMIN(http.MethodPost, ruta, cuerpo))
		if w.Code != http.StatusForbidden || servicio.cierres != i+1 ||
			servicio.cierre.Aprobador.PersonaRef != sesion.Actor.PersonaRef ||
			servicio.cierre.Aprobador.PerfilActivoRef != sesion.Actor.PerfilActivoRef ||
			servicio.cierre.CorrelacionRef != sesion.CorrelacionRef ||
			servicio.cierre.InstantaneaAutorizacion.AsignacionPerfil.PrincipalID != sesion.Actor.PersonaRef ||
			auditor.llamadas != i+1 {
			t.Fatalf("cierre no respetó sesión por %s: %d, cierres %d, auditorías %d", ruta, w.Code, servicio.cierres, auditor.llamadas)
		}
	}
}
