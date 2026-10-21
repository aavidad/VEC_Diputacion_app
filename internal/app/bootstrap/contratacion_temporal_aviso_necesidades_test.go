package bootstrap

import (
	"bytes"
	"log/slog"
	"strings"
	"testing"

	postgresct "vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/postgres"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/application"
)

// Un arranque sin fuente de necesidades avisa una sola vez por requisito: la
// composición emite el aviso donde decide la fuente y el montaje posterior de
// las rutas de peticiones del centro, con el alta sin fuente, no lo repite.
func TestAvisoFuenteNecesidadesSaleUnaVezPorArranque(t *testing.T) {
	for _, caso := range []struct {
		nombre, rutaRPT  string
		fuente, catalogo int
	}{
		{"con_rpt", "rpt/catalogo.json", 1, 0},
		{"sin_rpt", "", 1, 1},
	} {
		t.Run(caso.nombre, func(t *testing.T) {
			var registro bytes.Buffer
			previo := slog.Default()
			slog.SetDefault(slog.New(slog.NewJSONHandler(&registro, nil)))
			t.Cleanup(func() { slog.SetDefault(previo) })

			// Mismo orden que la composición: decisión de la fuente y, después,
			// montaje de las rutas sobre el mismo servicio de alta sin fuente.
			avisarFuenteNecesidadesAltaNoConfigurada(caso.rutaRPT)
			p := &proveedorPeticionCentroDesarrollo{reloj: relojContratacionTemporalDesarrollo{},
				alta: &dependenciasAltaContratacionTemporalDesarrollo{servicio: &application.ServicioRegistroSolicitud{}}}
			if p.alta.servicio.FuenteNecesidadesAltaConfigurada() {
				t.Fatal("el alta de prueba no debía tener fuente")
			}
			if _, err := rutasHTTPPeticionCentroDesarrollo(p, &postgresct.RepositorioPeticionesCentroPostgreSQL{}); err != nil {
				t.Fatal(err)
			}
			salida := registro.String()
			fuente := strings.Count(salida, `"requisito":"VEC_CT_NECESIDADES_ALTA_SOURCE_PATH"`)
			catalogo := strings.Count(salida, `"requisito":"VEC_RPT_CATALOGO_PATH"`)
			if fuente != caso.fuente || catalogo != caso.catalogo ||
				strings.Count(salida, "fuente de necesidades de alta no configurada al arrancar") != caso.fuente+caso.catalogo {
				t.Fatalf("avisos de fuente=%d catálogo=%d, esperados %d y %d:\n%s", fuente, catalogo, caso.fuente, caso.catalogo, salida)
			}
		})
	}
}
