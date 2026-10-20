package bootstrap

import (
	"slices"
	"testing"

	"vec-diputacion-granada/internal/modules/bolsa/application/inscripcion"
)

// F1-101: la autorización de lectura exige que la concesión publicada tenga
// exactamente los campos del descriptor. La fuente v2 debe coincidir con las
// cuatro lecturas del aspirante; la v1 no lo hace para la lista y la ficha.
func TestFuenteInscripcionV2CoincideConDescriptoresExternos(t *testing.T) {
	acciones := []string{inscripcion.AccionListarAbiertas, inscripcion.AccionDetalleAbierta,
		inscripcion.AccionListarPropias, inscripcion.AccionDetallePropia}
	v1 := concesionesFuenteBolsaPrueba(t, "inscripcion_recorrido_20261013/inscripcion_bolsa_actos_v1.json", acciones...)
	v2 := concesionesFuenteBolsaPrueba(t, "fuentes/inscripcion_bolsa_actos_v2.json", acciones...)
	descriptores := descriptoresLecturaActualInscripcionBolsa(superficieExternaInscripcionBolsa)
	for _, c := range v2 {
		d, ok := descriptores[ClaveOperacionInscripcionBolsa{c.Accion, superficieExternaInscripcionBolsa}]
		if !ok {
			t.Fatalf("sin descriptor: %s", c.Accion)
		}
		campos := slices.Clone(c.CamposPermitidos)
		slices.Sort(campos)
		if c.ModuloID != d.ModuloID || c.TipoRecurso != d.TipoRecurso || !slices.Equal(c.Finalidades, []string{d.Finalidad}) ||
			len(c.Obligaciones) != 0 || !slices.Equal(campos, d.Campos) {
			t.Fatalf("la concesión v2 de %s no coincide con su descriptor", c.Accion)
		}
	}
	for _, c := range v1 {
		d := descriptores[ClaveOperacionInscripcionBolsa{c.Accion, superficieExternaInscripcionBolsa}]
		igual := slices.Equal(c.CamposPermitidos, d.Campos)
		convocatoria := c.Accion == inscripcion.AccionListarAbiertas || c.Accion == inscripcion.AccionDetalleAbierta
		if igual == convocatoria {
			t.Fatalf("v1 de %s: coincidencia inesperada %v", c.Accion, igual)
		}
	}
}
