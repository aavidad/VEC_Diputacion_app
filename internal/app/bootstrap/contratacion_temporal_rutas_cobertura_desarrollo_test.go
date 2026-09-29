package bootstrap

import (
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	vechttp "vec-diputacion-granada/internal/vec/adapters/httpapi"
)

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
	descriptores := descriptoresFronterasContratacionTemporalDesarrollo(perfil, []string{perfil})
	descriptores = append(descriptores, descriptoresFronterasReincorporacionTitularDesarrollo(perfil)...)
	descriptores = append(descriptores, descriptoresFronterasPlantillasCTDesarrollo(perfil)...)
	descriptores = append(descriptores, descriptoresFronterasPlantillasDocumentalCTDesarrollo(perfil)...)
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
