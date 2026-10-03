package bootstrap

import (
	"testing"

	meritos "vec-diputacion-granada/internal/modules/meritos/application"
)

func TestMaterialMeritosInternosTieneAudienciasNominalesSeparadas(t *testing.T) {
	descriptores := descriptoresMaterialMeritosInternosDesarrollo()
	_, declaracion := meritos.EspecificacionAutorizacion("meritos.hecho.declarar")
	if descriptores[0].Audiencia != declaracion || descriptores[1].Audiencia != meritos.AudienciaConsultaPropia ||
		descriptores[0].Audiencia == descriptores[1].Audiencia || descriptores[0].Dominio == descriptores[1].Dominio ||
		descriptores[0].Prefijo == descriptores[1].Prefijo {
		t.Fatal("audiencias o espacios de claves de Méritos cruzados")
	}
	for _, descriptor := range descriptores {
		if descriptor.Audiencia == "" || descriptor.Dominio == "" || descriptor.Prefijo == "" ||
			descriptor.ProveedorNominal != proveedorMaterialContratacionTemporal ||
			!audienciaConsumoGobiernoPostgreSQLContratacionTemporalDesarrolloEsPropia(descriptor.Audiencia) {
			t.Fatal("descriptor nominal de Méritos no publicable")
		}
		veces := 0
		for _, audiencia := range audienciasConsumoGobiernoCTDesarrollo() {
			if audiencia == descriptor.Audiencia {
				veces++
			}
		}
		if veces != 1 {
			t.Fatal("audiencia de Méritos ausente o repetida")
		}
	}
	for _, accion := range []string{"meritos.hecho.rectificar", "meritos.hecho.rechazar", "meritos.hecho.verificar"} {
		_, audiencia := meritos.EspecificacionAutorizacion(accion)
		if audienciaConsumoGobiernoPostgreSQLContratacionTemporalDesarrolloEsPropia(audiencia) {
			t.Fatal("publicación de otra operación de Méritos habilitada")
		}
	}
}
