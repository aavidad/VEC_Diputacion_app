package bootstrap

import (
	"testing"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
)

func TestPublicacionesGobiernoCoberturaDesarrolloSonExactasYRepetibles(t *testing.T) {
	soporte, _, _ := escenarioAutorizacionCoberturaDesarrolloPrueba(t)
	publicaciones, err := nuevasPublicacionesGobiernoCoberturaDesarrollo(soporte)
	if err != nil {
		t.Fatal(err)
	}
	acciones := []domain.ClaveCatalogo{
		domain.AccionDecidirCoberturaGobernada,
		domain.AccionRectificarCoberturaGobernada,
	}
	if len(publicaciones) != len(acciones) {
		t.Fatalf("publicaciones=%d", len(publicaciones))
	}
	for indice, publicacion := range publicaciones {
		if publicacion.Secuencia != uint64(indice+1) ||
			publicacion.Actuacion.Accion != acciones[indice] ||
			publicacion.Actuacion.OrganizacionRef != organizacionAltaContratacionTemporalDesarrollo ||
			publicacion.Actuacion.UnidadEjecutoraRef != unidadCoberturaContratacionTemporalDesarrollo ||
			publicacion.Actuacion.Validar() != nil {
			t.Fatalf("publicacion[%d] invalida: %#v", indice, publicacion)
		}
	}
	repetidas, err := nuevasPublicacionesGobiernoCoberturaDesarrollo(soporte)
	if err != nil {
		t.Fatal(err)
	}
	for indice := range publicaciones {
		if publicaciones[indice].EventoRef != repetidas[indice].EventoRef ||
			publicaciones[indice].Actuacion.HuellaSHA256 !=
				repetidas[indice].Actuacion.HuellaSHA256 {
			t.Fatalf("publicacion[%d] no determinista", indice)
		}
	}
}
