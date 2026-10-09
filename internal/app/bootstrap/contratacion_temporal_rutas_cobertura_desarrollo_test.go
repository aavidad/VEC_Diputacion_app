package bootstrap

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/httpinterno"
	postgresct "vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/postgres"
	vechttp "vec-diputacion-granada/internal/vec/adapters/httpapi"
	vecmemory "vec-diputacion-granada/internal/vec/adapters/memory"
	vecapp "vec-diputacion-granada/internal/vec/application"
)

type autoridadRutaB2DispatcherPrueba struct{ rutas []string }

func (a *autoridadRutaB2DispatcherPrueba) AutorizarRutaExacta(_ context.Context, ruta string) error {
	a.rutas = append(a.rutas, ruta)
	return nil
}

func TestCoberturaRutasCTFirmaRequiereFronterasPDPActivadas(t *testing.T) {
	const perfil = "prf_cobertura_ct"
	rutas := []vechttp.RutaExacta{
		rutaCoberturaCTPrueba(httpinterno.RutaFirmaDocumento),
		rutaCoberturaCTPrueba(httpinterno.RutaConsultaFirmaDocumento),
	}
	for _, activa := range []bool{false, true} {
		t.Run(fmt.Sprint(activa), func(t *testing.T) {
			catalogo, err := nuevoCatalogoFronterasComunDesarrollo(
				descriptoresFronterasContratacionTemporalDesarrollo(perfil, []string{perfil}, activa))
			if err != nil {
				t.Fatal(err)
			}
			err = validarCoberturaRutasCTDesarrollo(rutas, catalogo)
			if activa && err != nil {
				t.Fatalf("firma activa con sus dos fronteras: %v", err)
			}
			if !activa && !errors.Is(err, ErrCoberturaRutasCTDesarrollo) {
				t.Fatalf("firma registrada sin sus fronteras: %v", err)
			}
		})
	}
}

func catalogoCoberturaRutasCTPrueba(t *testing.T, omitirMetodo, omitirRuta string) catalogoFronterasComunDesarrollo {
	t.Helper()
	var descriptores []descriptorFronteraComunDesarrollo
	for ruta, metodos := range inventarioRutasCTDesarrollo() {
		for _, metodo := range metodos {
			if metodo.guardia != "" || metodo.metodo == omitirMetodo && ruta == omitirRuta {
				continue
			}
			clave := fmt.Sprintf("ct-cobertura-%d", len(descriptores))
			descriptores = append(descriptores, descriptorFronteraComunDesarrollo{
				Clave: clave, Superficie: superficieInternaSeguridadComunDesarrollo,
				Metodo: metodo.metodo, Ruta: ruta, PerfilesActivosRef: []string{"prf_cobertura_ct"},
				ClavePolitica: "ct-cobertura-prueba", ClaveCapacidad: clave,
			})
		}
	}
	catalogo, err := nuevoCatalogoFronterasComunDesarrollo(descriptores)
	if err != nil {
		t.Fatalf("catálogo sintético: %v", err)
	}
	return catalogo
}

func rutaCoberturaCTPrueba(ruta string) vechttp.RutaExacta {
	return vechttp.RutaExacta{Ruta: ruta, Manejador: http.HandlerFunc(func(http.ResponseWriter, *http.Request) {})}
}

func TestCoberturaRutasCTInventarioCompletoYOpcionales(t *testing.T) {
	const perfil = "prf_cobertura_ct"
	descriptores := descriptoresFronterasContratacionTemporalDesarrollo(perfil, []string{perfil}, true)
	descriptores = append(descriptores, descriptoresFronterasReincorporacionTitularDesarrollo(perfil)...)
	descriptores = append(descriptores, descriptoresFronterasPlantillasCTDesarrollo(perfil)...)
	descriptores = append(descriptores, descriptoresFronterasPlantillasDocumentalCTDesarrollo(perfil)...)
	descriptores = append(descriptores, descriptoresFronteraAjustesReglasCT(perfil)...)
	descriptores = append(descriptores, fronteraContratacionTemporalDesarrollo(
		"ct-circuito-rrhh-consultar", postgresct.AccionConsultaCircuitoRRHH,
		httpinterno.RutaConsultaCircuitoRRHH, []string{perfil}))
	for _, descriptor := range descriptoresFronterasIncorporacionB2Desarrollo() {
		descriptor.PerfilesActivosRef = []string{perfil}
		descriptores = append(descriptores, descriptor)
	}
	catalogo, err := nuevoCatalogoFronterasComunDesarrollo(descriptores)
	if err != nil {
		t.Fatalf("catálogo real de descriptores CT: %v", err)
	}
	for ruta, metodos := range inventarioRutasCTDesarrollo() {
		if !esRutaContratacionTemporalDesarrollo(httptest.NewRequest(http.MethodGet, ruta, nil)) {
			t.Errorf("ruta CT inventariada sin revalidador mTLS: %s", ruta)
		}
		for _, metodo := range metodos {
			if metodo.guardia != "" {
				continue
			}
			if _, ok := catalogo.resolver(metodo.metodo, ruta); !ok {
				t.Errorf("falta frontera real %s %s", metodo.metodo, ruta)
			}
		}
	}
	rutas := make([]vechttp.RutaExacta, 0, len(inventarioRutasCTDesarrollo())+1)
	for ruta := range inventarioRutasCTDesarrollo() {
		rutas = append(rutas, rutaCoberturaCTPrueba(ruta))
	}
	// El slice de composición mezcla módulos; Bolsa no está bajo prefijo CT.
	rutas = append(rutas, rutaCoberturaCTPrueba("/api/vec/bolsa/avisos"))
	if err := validarCoberturaRutasCTDesarrollo(rutas, catalogo); err != nil {
		t.Fatalf("rutas conocidas con fronteras presentes: %v", err)
	}
}

func TestCoberturaRutasIncorporacionB2ExigeCadaFrontera(t *testing.T) {
	descriptores := descriptoresFronterasIncorporacionB2Desarrollo()
	for i := range descriptores {
		descriptores[i].PerfilesActivosRef = []string{"prf_cobertura_b2"}
	}
	rutas := []vechttp.RutaExacta{rutaCoberturaCTPrueba(httpinterno.RutaPlanB2), rutaCoberturaCTPrueba(httpinterno.RutaConfirmacionB2)}
	for _, ausente := range []struct{ metodo, ruta string }{{http.MethodGet, httpinterno.RutaPlanB2}, {http.MethodPost, httpinterno.RutaPlanB2}, {http.MethodPost, httpinterno.RutaConfirmacionB2}} {
		filtrados := make([]descriptorFronteraComunDesarrollo, 0, len(descriptores)-1)
		for _, d := range descriptores {
			if d.Metodo != ausente.metodo || d.Ruta != ausente.ruta {
				filtrados = append(filtrados, d)
			}
		}
		catalogo, err := nuevoCatalogoFronterasComunDesarrollo(filtrados)
		if err != nil {
			t.Fatal(err)
		}
		if err := validarCoberturaRutasCTDesarrollo(rutas, catalogo); !errors.Is(err, ErrCoberturaRutasCTDesarrollo) {
			t.Fatalf("falta %s %s: %v", ausente.metodo, ausente.ruta, err)
		}
	}
	catalogo, err := nuevoCatalogoFronterasComunDesarrollo(descriptores)
	if err != nil {
		t.Fatal(err)
	}
	if err := validarCoberturaRutasCTDesarrollo(rutas, catalogo); err != nil {
		t.Fatal(err)
	}
}

func TestRutasIncorporacionB2MontanEnDispatcherVEC(t *testing.T) {
	store := vecmemory.NewStore()
	service, _, err := vecapp.NewServiceWithInternalOperations(store, store, store)
	if err != nil {
		t.Fatal(err)
	}
	autoridad := &autoridadRutaB2DispatcherPrueba{}
	rutas := []vechttp.RutaExacta{
		{Ruta: httpinterno.RutaPlanB2, Manejador: http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusNoContent) })},
		{Ruta: httpinterno.RutaConfirmacionB2, Manejador: http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusNoContent) })},
	}
	handler, err := vechttp.NewHandlerWithOptions(service, vechttp.HandlerOptions{RutasExactas: rutas, AutoridadRutasExactas: autoridad})
	if err != nil {
		t.Fatalf("las rutas B2 no montan en el dispatcher de VEC: %v", err)
	}
	for _, caso := range []struct{ metodo, ruta string }{{http.MethodGet, httpinterno.RutaPlanB2}, {http.MethodPost, httpinterno.RutaConfirmacionB2}} {
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, httptest.NewRequest(caso.metodo, caso.ruta, nil))
		if w.Code != http.StatusNoContent {
			t.Fatalf("%s %s: HTTP %d", caso.metodo, caso.ruta, w.Code)
		}
	}
	if len(autoridad.rutas) != 2 || autoridad.rutas[0] != httpinterno.RutaPlanB2 || autoridad.rutas[1] != httpinterno.RutaConfirmacionB2 {
		t.Fatalf("las rutas B2 eludieron la autoridad exacta: %v", autoridad.rutas)
	}
}

func TestFronterasIncorporacionB2SoloPerfilesNominalesYAccionesExactas(t *testing.T) {
	descriptores := descriptoresFronterasIncorporacionB2Desarrollo()
	esperadas := []struct{ metodo, ruta, accion string }{
		{http.MethodGet, httpinterno.RutaPlanB2, "contratacion_temporal.incorporacion_personal.plan.consultar"},
		{http.MethodPost, httpinterno.RutaPlanB2, "contratacion_temporal.incorporacion_personal.plan.registrar"},
		{http.MethodPost, httpinterno.RutaConfirmacionB2, "contratacion_temporal.incorporacion_personal.origen.confirmar"},
	}
	if len(descriptores) != len(esperadas) {
		t.Fatalf("fronteras B2: %d", len(descriptores))
	}
	for i, d := range descriptores {
		if d.Metodo != esperadas[i].metodo || d.Ruta != esperadas[i].ruta || d.ClaveCapacidad != esperadas[i].accion || len(d.PerfilesActivosRef) != 0 {
			t.Fatalf("frontera B2 abierta o incorrecta: %+v", d)
		}
	}
}

func TestFronterasIncorporacionB2AsignanPerfilesAntesDelCatalogo(t *testing.T) {
	autoridad, _, _, _ := escenarioPermisoIncorporacionPrueba(t)
	soporte := autoridad.soporte
	base := soporte.contexto.Resultado.Contexto.PerfilActivoRef
	declaraciones := descriptoresFronterasContratacionTemporalDesarrollo(base, []string{base})
	declaraciones = append(declaraciones, descriptoresFronterasIncorporacionB2Desarrollo()...)
	asignadas, err := asignarPerfilesNominalesB2EnFronteras(soporte, declaraciones)
	if err != nil {
		t.Fatal(err)
	}
	catalogo, err := nuevoCatalogoFronterasComunDesarrollo(asignadas)
	if err != nil {
		t.Fatal(err)
	}
	for _, par := range []struct{ metodo, ruta string }{{http.MethodGet, httpinterno.RutaPlanB2}, {http.MethodPost, httpinterno.RutaPlanB2}, {http.MethodPost, httpinterno.RutaConfirmacionB2}} {
		d, ok := catalogo.resolver(par.metodo, par.ruta)
		if !ok || d.admitePerfil(base) || len(d.PerfilesActivosRef) != len(gruposPerfilesIncorporacionB2()) {
			t.Fatalf("frontera B2 sin perfiles nominales cerrados: %s %s", par.metodo, par.ruta)
		}
	}
}

func TestCoberturaRutasCTEntregaExigePOSTPropio(t *testing.T) {
	catalogo := catalogoCoberturaRutasCTPrueba(t, http.MethodPost, rutaEntregaPeticionCentro)
	if _, ok := catalogo.resolver(http.MethodGet, rutaEntregaPeticionCentro); !ok {
		t.Fatal("fixture sin lectura GET")
	}
	if err := validarCoberturaRutasCTDesarrollo([]vechttp.RutaExacta{rutaCoberturaCTPrueba(rutaEntregaPeticionCentro)}, catalogo); !errors.Is(err, ErrCoberturaRutasCTDesarrollo) {
		t.Fatalf("GET no debe cubrir POST de entrega: %v", err)
	}
}

func TestCoberturaRutasCTRechazaRutaNuevaYMetodoNoInventariado(t *testing.T) {
	catalogo := catalogoCoberturaRutasCTPrueba(t, "", "")
	if err := validarCoberturaRutasCTDesarrollo([]vechttp.RutaExacta{rutaCoberturaCTPrueba("/api/vec/contratacion-temporal/nueva-escritura")}, catalogo); !errors.Is(err, ErrCoberturaRutasCTDesarrollo) {
		t.Fatalf("ruta CT nueva sin inventario: %v", err)
	}
	if err := validarMetodoRutaCTDesarrollo(rutaOrganizacionContratacionTemporalDesarrollo,
		nominalCT(http.MethodPost, "contratacion_temporal_organizacion_desarrollo.go:manejadorOrganizacionContratacionTemporalDesarrollo"), catalogo); !errors.Is(err, ErrCoberturaRutasCTDesarrollo) {
		t.Fatalf("POST no inventariado sobre ruta GET/HEAD: %v", err)
	}
}

func TestCoberturaRutasCTNoAceptaPDPEnRutaNominal(t *testing.T) {
	catalogo, err := nuevoCatalogoFronterasComunDesarrollo([]descriptorFronteraComunDesarrollo{{
		Clave: "ct-organizacion-pdp-ajeno", Superficie: superficieInternaSeguridadComunDesarrollo,
		Metodo: http.MethodGet, Ruta: rutaOrganizacionContratacionTemporalDesarrollo,
		PerfilesActivosRef: []string{"prf_cobertura_ct"}, ClavePolitica: "ct-prueba", ClaveCapacidad: "ct-prueba",
	}})
	if err != nil {
		t.Fatal(err)
	}
	if err := validarCoberturaRutasCTDesarrollo([]vechttp.RutaExacta{rutaCoberturaCTPrueba(rutaOrganizacionContratacionTemporalDesarrollo)}, catalogo); !errors.Is(err, ErrCoberturaRutasCTDesarrollo) {
		t.Fatalf("PDP inesperado en autoridad nominal: %v", err)
	}
}

func TestCoberturaRutasCTRestringeElManejadorFinalAntesDeEjecutarlo(t *testing.T) {
	catalogo := catalogoCoberturaRutasCTPrueba(t, "", "")
	llamadas := 0
	rutas := []vechttp.RutaExacta{{Ruta: rutaOrganizacionContratacionTemporalDesarrollo,
		Manejador: http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			llamadas++
			w.WriteHeader(http.StatusNoContent)
		})}}
	if err := validarCoberturaRutasCTDesarrollo(rutas, catalogo); err != nil {
		t.Fatal(err)
	}
	for _, metodo := range []string{http.MethodPost, http.MethodPut, http.MethodDelete} {
		w := httptest.NewRecorder()
		rutas[0].Manejador.ServeHTTP(w, httptest.NewRequest(metodo, rutas[0].Ruta, nil))
		if w.Code != http.StatusMethodNotAllowed || llamadas != 0 || w.Header().Get("Allow") != "GET, HEAD" {
			t.Fatalf("verbo extra %s ejecutó el manejador: HTTP=%d llamadas=%d", metodo, w.Code, llamadas)
		}
	}
	for _, metodo := range []string{http.MethodGet, http.MethodHead} {
		w := httptest.NewRecorder()
		rutas[0].Manejador.ServeHTTP(w, httptest.NewRequest(metodo, rutas[0].Ruta, nil))
		if w.Code != http.StatusNoContent {
			t.Fatalf("verbo nominal %s bloqueado: %d", metodo, w.Code)
		}
	}
	if llamadas != 2 {
		t.Fatalf("ejecuciones nominales: %d", llamadas)
	}
}
