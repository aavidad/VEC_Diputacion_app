package bootstrap

import (
	"net/http"
	"testing"

	seleccioninterno "vec-diputacion-granada/internal/modules/seleccion/adapters/httpinterno"
	seleccionpersonal "vec-diputacion-granada/internal/modules/seleccion/adapters/httppersonal"
)

// Las operaciones de Selección quedan declaradas en la frontera común junto a
// las de CT y Bolsa, sin colisiones: persona en la superficie personal con el
// perfil de su certificado y RRHH en la interna con el perfil de Selección.
// Lo no declarado (otro método u otra ruta) no resuelve ninguna frontera.
func TestFronterasSeleccionDeclaradasSinColisiones(t *testing.T) {
	perfilCT := referenciaAltaContratacionTemporalDesarrollo("prf_", "ct")
	perfilBolsa := referenciaAltaContratacionTemporalDesarrollo("prf_", "bolsa")
	perfilPersona := referenciaAltaContratacionTemporalDesarrollo("prf_", "persona")
	perfilSeleccion := referenciaAltaContratacionTemporalDesarrollo("prf_", "seleccion")
	declaraciones := descriptoresFronterasContratacionTemporalDesarrollo(perfilCT, []string{perfilCT})
	bback, err := descriptoresFronterasBorradorLlamamientoBolsaDesarrollo(perfilBolsa)
	if err != nil {
		t.Fatal(err)
	}
	declaraciones = append(declaraciones, bback...)
	declaraciones = append(declaraciones, descriptoresFronterasSeleccionPersonalDesarrollo(perfilPersona)...)
	declaraciones = append(declaraciones, descriptoresFronterasSeleccionRRHHDesarrollo(perfilSeleccion)...)
	catalogo, err := nuevoCatalogoFronterasComunDesarrollo(declaraciones)
	if err != nil {
		t.Fatalf("las fronteras de Selección no son válidas o colisionan: %v", err)
	}
	for _, o := range seleccionpersonal.Operaciones() {
		d, ok := catalogo.resolver(o.Metodo, o.Ruta)
		if !ok || d.Superficie != superficieExternaPersonalSeguridadComunDesarrollo || !d.admitePerfil(perfilPersona) || d.admitePerfil(perfilCT) {
			t.Fatalf("operación personal %s %s mal declarada", o.Metodo, o.Ruta)
		}
		if !rutaSeleccionPersonalDesarrollo(o.Ruta) || !rutaPersonalCandidatoDesarrollo(o.Ruta) {
			t.Fatalf("ruta personal %s sin la protección de la superficie personal", o.Ruta)
		}
	}
	for _, o := range seleccioninterno.Operaciones() {
		d, ok := catalogo.resolver(o.Metodo, o.Ruta)
		if !ok || d.Superficie != superficieInternaSeguridadComunDesarrollo || !d.admitePerfil(perfilSeleccion) || d.admitePerfil(perfilBolsa) {
			t.Fatalf("operación de RRHH %s %s mal declarada", o.Metodo, o.Ruta)
		}
	}
	for _, sin := range [][2]string{{http.MethodDelete, seleccionpersonal.RutaBorrador}, {http.MethodPost, seleccionpersonal.RutaMisSolicitudes},
		{http.MethodGet, seleccioninterno.RutaConsultas}, {http.MethodGet, "/api/vec/seleccion/otra"}} {
		if _, ok := catalogo.resolver(sin[0], sin[1]); ok {
			t.Fatalf("%s %s no declarada y sin embargo resuelta", sin[0], sin[1])
		}
	}
	if len(descriptoresFronterasSeleccionPersonalDesarrollo("")) == 0 {
		t.Fatal("sin perfil no se declara nada útil, pero la lista debe existir para rechazarse al construir")
	}
	if _, err := nuevoCatalogoFronterasComunDesarrollo(descriptoresFronterasSeleccionPersonalDesarrollo("")); err == nil {
		t.Fatal("una frontera sin perfil debe rechazarse")
	}
}
