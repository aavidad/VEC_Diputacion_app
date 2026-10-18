package bootstrap

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"

	"vec-diputacion-granada/config"
	bolsahttp "vec-diputacion-granada/internal/modules/bolsa/adapters/httppersonal"
	bolsapuertos "vec-diputacion-granada/internal/modules/bolsa/ports"
	vecapp "vec-diputacion-granada/internal/vec/application"
)

func TestCatalogoPreferenciasModalidadesDeclaradoYOpcional(t *testing.T) {
	if _, err := catalogoPreferenciasModalidadesDesarrollo(t.Context(), config.Config{}); err == nil {
		t.Fatal("sin catálogo CT se habilitaron modalidades")
	}
	cfg := configuracionDesarrolloReglasEjemplo("", rutaReglasCTEjemploPrueba)
	catalogo, err := catalogoPreferenciasModalidadesDesarrollo(t.Context(), cfg)
	if err != nil {
		t.Fatalf("catálogo CT declarado no disponible: %v", err)
	}
	opciones, err := catalogo.ConsultarModalidadesDisponibles(t.Context())
	if err != nil || len(opciones) == 0 {
		t.Fatalf("catálogo CT sin opciones: %v", err)
	}
}

func TestModalidadesSinPreflightNoAbrenRutaNiIdentidad(t *testing.T) {
	if comprobarFuncionPreferenciasModalidades(context.Background(), nil) {
		t.Fatal("preflight aceptó un pool ausente")
	}
	t.Setenv(envBolsaPreferenciasInternas, "true")
	ruta, cerrar := nuevaRutaPreferenciasBolsasInternas(config.Config{}, nil, nil, nil)
	if ruta == nil || ruta.Manejador == nil || ruta.Ruta != bolsahttp.RutaPreferenciasBolsas {
		t.Fatal("la ruta opcional incompleta alteró el montaje")
	}
	defer cerrar()
	w := httptest.NewRecorder()
	ruta.Manejador.ServeHTTP(w, httptest.NewRequest(http.MethodGet,
		bolsahttp.RutaPreferenciasBolsas+"?tipo=modalidades", nil))
	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("modalidades sin preflight respondieron %d", w.Code)
	}
}

func TestModalidadesExternasSinPreflightConservanBolsa(t *testing.T) {
	d := dependenciasMiBolsaPortalExterno{
		preparador:  preparadorExteriorSinEfectos{},
		autorizador: new(vecapp.ServicioAutorizacionSolicitudLigadaV3),
		bolsa:       &pgxpool.Pool{},
		proveedores: map[string]*proveedorMaterialAltaContratacionTemporalDesarrollo{
			bolsapuertos.AudienciaMiBolsa:          {},
			bolsapuertos.AudienciaHistorialMiBolsa: {},
		},
	}
	rutas, err := nuevasRutasMiBolsaPortalExterno(d)
	if err != nil || len(rutas) != 4 || rutas[2].Ruta != bolsahttp.RutaPreferenciasBolsas {
		t.Fatalf("montaje exterior sin modalidades: %v", err)
	}
	for _, ruta := range []string{bolsahttp.RutaPreferenciasBolsas,
		bolsahttp.RutaPreferenciasBolsas + "?tipo=modalidades"} {
		w := httptest.NewRecorder()
		rutas[2].Manejador.ServeHTTP(w, httptest.NewRequest(http.MethodGet, ruta, nil))
		if w.Code != http.StatusServiceUnavailable {
			t.Fatalf("ruta opcional %s respondió %d", ruta, w.Code)
		}
	}
}
