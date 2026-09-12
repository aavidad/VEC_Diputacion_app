package bootstrap

import (
	"testing"
	"time"

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
	for indice := 0; indice < 2; indice++ {
		vias := publicaciones[indice].Catalogo.Vias
		if publicaciones[indice].Catalogo.Version != 1 ||
			publicaciones[indice].Actuacion.Version != 1 || len(vias) != 1 ||
			vias[0].Clave != "bolsa_vigente" {
			t.Fatalf("v1 alterada en publicacion[%d]: %#v", indice, publicaciones[indice])
		}
	}
	for indice := 2; indice < len(publicaciones); indice++ {
		vias := publicaciones[indice].Catalogo.Vias
		if publicaciones[indice].Catalogo.Version != 2 ||
			publicaciones[indice].Actuacion.Version != 2 || len(vias) != 3 ||
			vias[0].Clave != "bolsa_vigente" || vias[1].Clave != "oferta_sae" ||
			vias[2].Clave != "nueva_convocatoria_bolsa" {
			t.Fatalf("v2 no publicó las tres vías: %#v", publicaciones[indice])
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

func TestMotivoEleccionProcedimientoRRHHDesarrolloEsCatalogadoYOpaco(t *testing.T) {
	motivo := motivoEleccionProcedimientoRRHHDesarrollo()
	if motivo.CatalogoID != catalogoMotivosDecisionCoberturaDesarrollo ||
		motivo.CatalogoVersion != 1 ||
		motivo.EntradaClave != "motivo_e17578f7625f0843107c4ec9c77053bc" ||
		motivo.Validar() != nil {
		t.Fatalf("motivo funcional inesperado: %#v", motivo)
	}
}

func TestMotivoEleccionProcedimientoRRHHPublicaInstanteNominalRepetible(t *testing.T) {
	primero := instantePublicacionMotivoEleccionProcedimientoRRHHDesarrollo()
	segundo := instantePublicacionMotivoEleccionProcedimientoRRHHDesarrollo()
	if !primero.Equal(segundo) || !primero.Equal(time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)) {
		t.Fatalf("instante de publicación no determinista: %s / %s", primero, segundo)
	}
}
