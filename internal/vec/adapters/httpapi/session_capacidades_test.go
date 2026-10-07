package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"vec-diputacion-granada/internal/vec/application"
	"vec-diputacion-granada/internal/vec/domain"
	"vec-diputacion-granada/internal/vec/ports"
)

type fuenteSesionHTTPPrueba struct {
	instantanea domain.InstantaneaAutorizacion
	err         error
}

func (f fuenteSesionHTTPPrueba) ObtenerInstantaneaAutorizacion(context.Context, string, string) (domain.InstantaneaAutorizacion, error) {
	return f.instantanea, f.err
}

type selectorSesionHTTPPrueba struct{ seleccion ports.SeleccionSesion }

func (s selectorSesionHTTPPrueba) SeleccionarSesion(context.Context, domain.Principal) (ports.SeleccionSesion, error) {
	return s.seleccion, nil
}

type relojSesionHTTPPrueba struct{ ahora time.Time }

func (r relojSesionHTTPPrueba) Ahora() time.Time { return r.ahora }

func proyectorSesionHTTPPrueba(t *testing.T, errorFuente error) *application.ProyectorCapacidadesSesion {
	t.Helper()
	ahora := time.Date(2026, 7, 15, 8, 0, 0, 0, time.UTC)
	rol := domain.VersionRol{
		RolID: "rrhh", Version: 1, Nombre: "RRHH", Estado: domain.EstadoVersionRolPublicada,
		Concesiones: []domain.ConcesionRol{{Accion: "consultar", ModuloID: "bolsa", TipoRecurso: "expediente",
			Finalidades: []string{"gestion"}, GarantiaMinima: domain.AuthAssuranceSubstantial}},
		PublicadaPor: "seguridad", PublicadaEn: ahora.Add(-3 * time.Hour),
	}
	huella, err := domain.HuellaCatalogoPoliticasAutorizacion(nil)
	if err != nil {
		t.Fatal(err)
	}
	instantanea := domain.InstantaneaAutorizacion{
		AsignacionPerfil: domain.AsignacionPerfil{
			AsignacionID: "asig-rrhh", Version: 2, PerfilActivoRef: "perfil-rrhh", PrincipalID: "actor-prueba-explicito",
			VersionRolRef: rol.Referencia(), Estado: domain.EstadoAsignacionPerfilActiva,
			Ambitos:      []domain.AmbitoPerfil{{Clave: "unidad", Valores: []string{"seleccion"}}},
			VigenteDesde: ahora.Add(-time.Hour), VigenteHasta: ahora.Add(time.Hour),
			EmitidaPor: "identidad", EmitidaEn: ahora.Add(-2 * time.Hour),
		},
		VersionRol: rol,
		ControlVigenciaVersionRol: domain.ControlVigenciaVersionRol{
			VersionRolRef: rol.Referencia(), Revision: 3, Estado: domain.EstadoControlVigenciaVersionRolHabilitada,
			ActualizadoPor: "seguridad", ActualizadoEn: rol.PublicadaEn,
		},
		RevisionCatalogoPoliticas: 4, CatalogoPoliticasHuellaSHA256: huella,
	}
	if err := instantanea.Validar(); err != nil {
		t.Fatalf("fixture invalida: %v", err)
	}
	proyector, err := application.NuevoProyectorCapacidadesSesion(
		fuenteSesionHTTPPrueba{instantanea: instantanea, err: errorFuente},
		selectorSesionHTTPPrueba{seleccion: ports.SeleccionSesion{
			PerfilActivoRef: "perfil-rrhh", Superficie: "interna", VigenteHasta: ahora.Add(time.Hour),
			Exposiciones: []ports.ExposicionSesion{{Superficie: "interna", ModuloID: "bolsa", TipoRecurso: "expediente", Accion: "consultar"}},
		}}, relojSesionHTTPPrueba{ahora: ahora},
	)
	if err != nil {
		t.Fatal(err)
	}
	return proyector
}

func TestSesionHTTPConservaPrincipalYProyectaCapacidadesSinCache(t *testing.T) {
	handler := newTestHandlerWithOptions(t, HandlerOptions{ProyectorCapacidadesSesion: proyectorSesionHTTPPrueba(t, nil)})
	respuesta := httptest.NewRecorder()
	handler.ServeHTTP(respuesta, httptest.NewRequest(http.MethodGet, "/api/vec/session", nil))
	if respuesta.Code != http.StatusOK || respuesta.Header().Get("Cache-Control") != "no-store" {
		t.Fatalf("sesion: %d %s", respuesta.Code, respuesta.Body.String())
	}
	var envoltura struct {
		Data struct {
			Principal       domain.Principal                        `json:"principal"`
			Capacidades     []application.CapacidadSesion           `json:"capacidades"`
			PerfilActivoRef string                                  `json:"perfil_activo_ref"`
			Revisiones      application.RevisionesCapacidadesSesion `json:"revisiones"`
		} `json:"data"`
	}
	if err := json.Unmarshal(respuesta.Body.Bytes(), &envoltura); err != nil {
		t.Fatal(err)
	}
	cuerpo := envoltura.Data
	if cuerpo.Principal.ID != "actor-prueba-explicito" || cuerpo.PerfilActivoRef != "perfil-rrhh" ||
		cuerpo.Revisiones.Asignacion != 2 || len(cuerpo.Capacidades) != 1 || cuerpo.Capacidades[0].ModuloID != "bolsa" {
		t.Fatalf("respuesta incompleta: %+v", cuerpo)
	}
}

func TestSesionHTTPDistingueFuenteCaidaDePermiso(t *testing.T) {
	handler := newTestHandlerWithOptions(t, HandlerOptions{
		ProyectorCapacidadesSesion: proyectorSesionHTTPPrueba(t, errors.Join(domain.ErrAutorizacionDenegada, ports.ErrFuenteAutorizacionNoDisponible)),
	})
	respuesta := httptest.NewRecorder()
	handler.ServeHTTP(respuesta, httptest.NewRequest(http.MethodGet, "/api/vec/session", nil))
	if respuesta.Code != http.StatusServiceUnavailable || respuesta.Header().Get("Cache-Control") != "no-store" {
		t.Fatalf("indisponibilidad presentada como permiso: %d %s", respuesta.Code, respuesta.Body.String())
	}
}
