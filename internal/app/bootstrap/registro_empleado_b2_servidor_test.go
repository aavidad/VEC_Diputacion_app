package bootstrap

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	apppersonal "vec-diputacion-granada/internal/modules/personal/application"
	personal "vec-diputacion-granada/internal/modules/personal/domain"
	pp "vec-diputacion-granada/internal/modules/personal/ports"
	"vec-diputacion-granada/internal/vec/adapters/httpapi"
	vecmemory "vec-diputacion-granada/internal/vec/adapters/memory"
	vecapp "vec-diputacion-granada/internal/vec/application"
	vp "vec-diputacion-granada/internal/vec/ports"
)

type repositorioLecturasPersonalSinUso struct{ llamadas int }

func (r *repositorioLecturasPersonalSinUso) ConsultarFichaRRHH(context.Context, pp.OrdenFichaEmpleadoB2) (pp.ResultadoFichaEmpleadoB2, error) {
	r.llamadas++
	return pp.ResultadoFichaEmpleadoB2{}, personal.ErrRegistroEmpleadoB2NoDisponible
}
func (r *repositorioLecturasPersonalSinUso) ListarVacantesRRHH(context.Context, pp.OrdenVacantesB2) (pp.ResultadoVacantesB2, error) {
	r.llamadas++
	return pp.ResultadoVacantesB2{}, personal.ErrRegistroEmpleadoB2NoDisponible
}
func (r *repositorioLecturasPersonalSinUso) ListarEmpleadosRRHH(context.Context, pp.OrdenEmpleadosB2) (pp.ResultadoEmpleadosB2, error) {
	r.llamadas++
	return pp.ResultadoEmpleadosB2{}, personal.ErrRegistroEmpleadoB2NoDisponible
}

func TestRegistroPersonalB2MontajeNativoDeniegaSinCanalYNoLeeRepositorio(t *testing.T) {
	c := configuracionB2PuraPrueba().PersonalB2
	a := &autoridadIncorporacionPersonalB2{perfiles: perfilesB2VinculoPrueba(t, c), organismoRef: c.OrganismoRef}
	repo := &repositorioLecturasPersonalSinUso{}
	ficha, err := apppersonal.NuevoServicioRegistroEmpleadoB2(a, repo)
	if err != nil {
		t.Fatal(err)
	}
	lista, err := apppersonal.NuevoServicioEmpleadosRegistroB2(a, repo)
	if err != nil {
		t.Fatal(err)
	}
	auditor := &registradorFronteraCatalogoPrueba{}
	rutas, colecciones, err := rutasRegistroEmpleadoPersonalB2(ficha, lista, a, auditor, nil, catalogoFronterasComunDesarrollo{})
	if err != nil {
		t.Fatal(err)
	}
	store := vecmemory.NewStore()
	service, _, err := vecapp.NewServiceWithInternalOperations(store, store, store)
	if err != nil {
		t.Fatal(err)
	}
	h, err := httpapi.NewHandlerWithOptions(service, httpapi.HandlerOptions{RutasExactas: rutas, RutasColeccion: colecciones, AutoridadRutasExactas: &autoridadRutaB2DispatcherPrueba{}})
	if err != nil {
		t.Fatalf("montaje raíz de Personal: %v", err)
	}
	ref := "emp_" + strings.Repeat("a", 32)
	query := "?vigente_en=2026-10-12&conocido_en=2026-10-12T10:00:00.000000Z"
	for _, ruta := range []string{httpapi.PrefijoFichaEmpleadoB2 + ref, httpapi.RutaVacantesEmpleadoB2, httpapi.RutaEmpleadosOrganismoB2} {
		w := httptest.NewRecorder()
		h.ServeHTTP(w, httptest.NewRequest(http.MethodGet, ruta+query, nil))
		if w.Code != http.StatusForbidden {
			t.Fatalf("%s sin canal: HTTP %d", ruta, w.Code)
		}
		orden := auditor.ordenes[len(auditor.ordenes)-1]
		if orden.Superficie != vp.SuperficieAuditoriaFronteraRutaExactaPersonal || orden.Ruta != httpapi.RutaAuditoriaRegistroEmpleadoB2(ruta) || strings.Contains(orden.Ruta, ref) {
			t.Fatalf("negativa sin apunte canónico Personal: %+v", orden)
		}
		ctx := context.WithValue(t.Context(), claveRutaPeticionIncorporacionB2{}, rutaPeticionIncorporacionB2{http.MethodGet, ruta})
		accion, ok := accionConsultaRegistroPersonalB2(ruta, http.MethodGet)
		if !ok || !operacionPermitidaEnRutaIncorporacionB2(ctx, accion) || operacionPermitidaEnRutaIncorporacionB2(ctx, personal.AccionAltaEmpleadoB2) {
			t.Fatal("la lectura nativa concede un acto o cambia su acción")
		}
	}
	if repo.llamadas != 0 || len(auditor.ordenes) != 3 {
		t.Fatal("lectura sin autoridad o negativa sin apunte")
	}
	if _, _, err := rutasRegistroEmpleadoPersonalB2(ficha, lista, a, nil, nil, catalogoFronterasComunDesarrollo{}); err == nil {
		t.Fatal("montaje sin auditoría admitido")
	}
}

func TestRegistroPersonalB2ListaOpcionalNoAmpliaElPerfilDelPlan(t *testing.T) {
	c := configuracionB2PuraPrueba().PersonalB2
	p := perfilesB2VinculoPrueba(t, c)
	lista, ficha := p.b2[personal.AccionEmpleadosB2], p.b2[personal.AccionFichaEmpleadoB2]
	if lista == nil || lista == ficha || len(lista.plantilla.VersionRol.Concesiones) != 1 || lista.plantilla.VersionRol.Concesiones[0].Accion != personal.AccionEmpleadosB2 {
		t.Fatal("la lista no conserva perfil lector propio")
	}
	delete(c.Operaciones, claveEmpleadosPersonalB2)
	if validarConfiguracionIncorporacionB2(c) != nil {
		t.Fatal("el material anterior dejó de ser válido")
	}
	p = perfilesB2VinculoPrueba(t, c)
	if p.b2[personal.AccionEmpleadosB2] != nil || p.b2[personal.AccionFichaEmpleadoB2] == nil {
		t.Fatal("la lista se provisionó sin configuración o se retiró la ficha")
	}
}
