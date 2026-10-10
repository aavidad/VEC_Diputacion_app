package administracionperfiles

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"vec-diputacion-granada/internal/vec/domain"
	"vec-diputacion-granada/internal/vec/ports"
)

type servicioVersionBolsaPrueba struct{ llamadas int }

func (s *servicioVersionBolsaPrueba) ProponerVersionarRolBolsa(context.Context,
	domain.SolicitudPropuestaVersionarRolBolsa) (ports.ResultadoPropuestaVersionarRolBolsa, error) {
	s.llamadas++
	return ports.ResultadoPropuestaVersionarRolBolsa{}, domain.ErrVersionarRolBolsaInvalido
}

func (s *servicioVersionBolsaPrueba) CerrarVersionarRolBolsaPorReferencia(context.Context,
	domain.SolicitudCierreVersionarRolBolsa) (domain.CierreVersionarRolBolsa, error) {
	s.llamadas++
	return domain.CierreVersionarRolBolsa{}, domain.ErrVersionarRolBolsaInvalido
}

type fuenteVersionBolsaPrueba struct{ llamadas int }

func (f *fuenteVersionBolsaPrueba) ObtenerCatalogoAccionesAdministracionV1(context.Context,
	string, int, string) (domain.CatalogoAccionesAdministracionV1, error) {
	f.llamadas++
	return domain.CatalogoAccionesAdministracionV1{}, domain.ErrCatalogoAccionesAdministracionInvalido
}

func TestVersionBolsaNoAceptaIdentidadEnCuerpoHTTP(t *testing.T) {
	sesion := &sesionPrueba{resultado: sesionAplicacionNominalPrueba(t)}
	auditor := &auditorPrueba{}
	h, err := NuevoHandlerUsuariosMetadatos("https://admin.example.test", sesion, &lecturasPrueba{}, auditor)
	if err != nil {
		t.Fatal(err)
	}
	servicio := &servicioVersionBolsaPrueba{}
	fuente := &fuenteVersionBolsaPrueba{}
	if err := h.ConVersionarRolBolsa(servicio, fuente, relojFocal{}); err != nil {
		t.Fatal(err)
	}
	w := httptest.NewRecorder()
	h.ServeHTTP(w, peticionADMIN(http.MethodPost, RutaVersionarRolBolsaProponer,
		`{"actor_persona_ref":"per_cliente","operacion_ref":"propuesta_admin:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"}`))
	if w.Code != http.StatusBadRequest || auditor.llamadas != 1 ||
		auditor.ultima.Codigo != "solicitud_invalida" || servicio.llamadas != 0 || fuente.llamadas != 0 {
		t.Fatal("el cuerpo con identidad llegó al servicio o quedó sin auditoría nominal")
	}
}
