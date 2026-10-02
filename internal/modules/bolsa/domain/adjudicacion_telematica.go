package domain

// ConfirmacionOfertaAceptacionPrevia identifica la política telemática:
// quienes aceptaron en plazo ya manifestaron su disposición antes de adjudicar.
const ConfirmacionOfertaAceptacionPrevia = "aceptacion_previa"

// ConfirmacionAdjudicacionValida conserva las políticas históricas sin este
// campo y limita las nuevas a la aceptación previa aprobada por RRHH.
func ConfirmacionAdjudicacionValida(confirmacion string) bool {
	return confirmacion == "" || confirmacion == ConfirmacionOfertaAceptacionPrevia
}
