package bootstrap

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"vec-diputacion-granada/config"
	ajustesapp "vec-diputacion-granada/internal/modules/contrataciontemporal/application/ajustesreglas"
	vechttp "vec-diputacion-granada/internal/vec/adapters/httpapi"
	vecdomain "vec-diputacion-granada/internal/vec/domain"
)

type lectorAjustesCTCaidoPrueba struct {
	*autoridadAsignacionesContratacionTemporalDesarrolloPrueba
}

func (lectorAjustesCTCaidoPrueba) leerAsignacionPublicada(context.Context, string) (instantaneaPublicadaDesarrollo, bool, error) {
	return instantaneaPublicadaDesarrollo{}, false, errors.New("fuente de prueba caída")
}

func TestCatalogoMotivoAutorizacionAjustesSoloConDobleLlave(t *testing.T) {
	t.Setenv(envCTMotivoAutorizacionAjustes,
		"../../../data/catalogos/contratacion_temporal/motivo_autorizacion_ajustes_v1.json")
	if _, activo, err := cargarMotivoAutorizacionAjustesCT(config.Config{}); err == nil || activo {
		t.Fatal("el catálogo no activa una ruta fuera del perfil de desarrollo")
	}
	cfg := config.Config{ExecutionProfile: config.ExecutionProfileDevelopment,
		AuthMode: config.AuthModeDevelopment, DevelopmentGuard: config.DevelopmentGuardAcknowledgement}
	motivo, activo, err := cargarMotivoAutorizacionAjustesCT(cfg)
	if err != nil || !activo || motivo.Referencia.Validar() != nil || motivo.RolID == "" {
		t.Fatalf("motivo técnico no cargado: activo=%t err=%v", activo, err)
	}
	t.Setenv(envCTMotivoAutorizacionAjustes, "")
	if _, activo, err := cargarMotivoAutorizacionAjustesCT(cfg); err != nil || activo {
		t.Fatalf("catálogo ausente activó ruta: activo=%t err=%v", activo, err)
	}
}

func TestRutaAjustesCTSoloAdmiteGET(t *testing.T) {
	if !esRutaContratacionTemporalDesarrollo(httptest.NewRequest(http.MethodGet, rutaAjustesReglasCT, nil)) {
		t.Fatal("la consulta queda fuera del perímetro mTLS")
	}
	llamadas := 0
	rutas := []vechttp.RutaExacta{{Ruta: rutaAjustesReglasCT, Manejador: http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		llamadas++
		w.WriteHeader(http.StatusOK)
	})}}
	if err := validarCoberturaRutasCTDesarrollo(rutas, catalogoCoberturaRutasCTPrueba(t, "", "")); err != nil {
		t.Fatal(err)
	}
	post := httptest.NewRecorder()
	rutas[0].Manejador.ServeHTTP(post, httptest.NewRequest(http.MethodPost, rutaAjustesReglasCT, nil))
	if post.Code != http.StatusMethodNotAllowed || post.Header().Get("Allow") != "GET" || llamadas != 0 {
		t.Fatalf("POST llegó al manejador: estado=%d llamadas=%d", post.Code, llamadas)
	}
	get := httptest.NewRecorder()
	rutas[0].Manejador.ServeHTTP(get, httptest.NewRequest(http.MethodGet, rutaAjustesReglasCT, nil))
	if get.Code != http.StatusOK || llamadas != 1 {
		t.Fatalf("GET nominal bloqueado: estado=%d llamadas=%d", get.Code, llamadas)
	}
}

func TestPerfilAjustesCTSigueSoloLectura(t *testing.T) {
	plantilla, err := plantillaConsultaAjustesCT("principal_prueba", "prf_reglas_ct_prueba",
		time.Now().UTC().Truncate(time.Microsecond), "ct_reglas_ajustes_consulta_desarrollo")
	if err != nil || plantilla.Validar() != nil {
		t.Fatalf("perfil de consulta inválido: %v", err)
	}
	concesiones := plantilla.VersionRol.Concesiones
	if len(concesiones) != 1 || concesiones[0].Accion != accionConsultarAjustesCT ||
		len(concesiones[0].CamposPermitidos) != 2 ||
		len(plantilla.AsignacionPerfil.Ambitos) != 1 ||
		plantilla.AsignacionPerfil.Ambitos[0].Clave != "organizacion_ref" {
		t.Fatal("el perfil de reglas amplió acción, campos o ámbito")
	}
}

func TestConsultaAjustesDistingueFuenteCaidaDePerfilRevocado(t *testing.T) {
	s, lector := escenarioPerfilesFijosPrueba(t)
	perfil := s.perfilFijoParaRutaYMetodo(rutaEntregaPeticionCentro, http.MethodGet)
	if perfil == nil {
		t.Fatal("falta perfil fijo de prueba")
	}
	lector.asignaciones = map[string]instantaneaPublicadaDesarrollo{
		perfil.perfilRef(): {instantanea: clonarInstantaneaAutorizacionAltaContratacionTemporalDesarrollo(perfil.plantilla),
			actoAsignacion: actoAsignacionPerfilFijoCTDesarrollo},
	}
	if err := comprobarPerfilConsultaAjustesCT(context.Background(), s, perfil); err != nil {
		t.Fatalf("perfil publicado rechazado: %v", err)
	}
	delete(lector.asignaciones, perfil.perfilRef())
	if err := comprobarPerfilConsultaAjustesCT(context.Background(), s, perfil); !errors.Is(err, vecdomain.ErrAutorizacionDenegada) {
		t.Fatalf("perfil retirado no devolvió denegación: %v", err)
	}
	s.mu.Lock()
	s.autoridadAsignaciones = lectorAjustesCTCaidoPrueba{lector}
	s.mu.Unlock()
	if err := comprobarPerfilConsultaAjustesCT(context.Background(), s, perfil); !errors.Is(err, ajustesapp.ErrNoDisponible) {
		t.Fatalf("lector caído se confundió con revocación: %v", err)
	}
}
