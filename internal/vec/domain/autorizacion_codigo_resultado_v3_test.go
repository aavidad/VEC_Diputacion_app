package domain

import "testing"

func TestCodigoResultadoEvaluacionAutorizacionV3ValidoEsAutoritativo(t *testing.T) {
	denegaciones := []string{
		"perfil_no_vigente",
		"ambito_no_autorizado",
		"rol_no_publicado",
		"rol_retirado",
		"accion_no_concedida",
		"finalidad_no_autorizada",
		"denegada_por_politica",
		"restriccion_abac_incumplida",
		"garantia_insuficiente",
	}
	if !CodigoResultadoEvaluacionAutorizacionV3Valido("concedida", true) {
		t.Fatal("concesión autoritativa rechazada")
	}
	for _, codigo := range denegaciones {
		if !CodigoResultadoEvaluacionAutorizacionV3Valido(codigo, false) {
			t.Fatalf("denegación autoritativa rechazada: %s", codigo)
		}
	}
	for _, caso := range []struct {
		codigo    string
		concedida bool
	}{
		{"inventado", false},
		{"concedida", false},
		{"accion_no_concedida", true},
		{"", false},
	} {
		if CodigoResultadoEvaluacionAutorizacionV3Valido(
			caso.codigo,
			caso.concedida,
		) {
			t.Fatalf(
				"resultado no autoritativo aceptado: %q/%t",
				caso.codigo,
				caso.concedida,
			)
		}
	}
}
