package bootstrap

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
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

func TestModalidadesSinPreflightNoAbrenRuta(t *testing.T) {
	if comprobarSQLPreferenciasModalidades(context.Background(), nil) == nil {
		t.Fatal("preflight aceptó un pool ausente")
	}
	// La consulta exige a la vez B138 y la restricción ampliada por AD300.
	for _, fragmento := range []string{"to_regprocedure($1)", "auditoria_lectura_preferencias_bolsas_formato_v2",
		"'''lectura_preferencias_modalidades_correcta'''", "c.convalidated", "a.grantee=0"} {
		if !strings.Contains(consultaSQLPreferenciasModalidades, fragmento) {
			t.Fatalf("la comprobación SQL no exige %s", fragmento)
		}
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
