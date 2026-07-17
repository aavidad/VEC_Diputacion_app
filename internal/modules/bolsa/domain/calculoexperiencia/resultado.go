package calculoexperiencia

import (
	"vec-diputacion-granada/internal/modules/bolsa/domain/reglasbaremo"
	"vec-diputacion-granada/internal/shared/baremacion"
)

// ResultadoExperienciaV1 es el recibo semantico inmutable del motor. No
// contiene identidad directa, modo oficial/simulacion, actor ni instante de
// ejecucion. Esos datos pertenecen al recibo administrativo que lo envuelva.
// Un error tecnico nunca produce este agregado: devuelve su valor cero y error.
type ResultadoExperienciaV1 struct {
	estado       EstadoResultadoExperienciaV1
	fase         FaseResultadoExperienciaV1
	vinculos     VinculosResultadoExperienciaV1
	seleccion    SeleccionResultadoExperienciaV1
	intervalos   []IntervaloAplicacionResultadoExperienciaV1
	aplicaciones []AplicacionCalculadaResultadoExperienciaV1
	reglas       []ResultadoReglaExperienciaV1
	secciones    []SubtotalSeccionResultadoExperienciaV1
	total        baremacion.Puntos
	tieneTotal   bool
	bloqueos     []BloqueoResultadoExperienciaV1
}

func (r ResultadoExperienciaV1) Estado() EstadoResultadoExperienciaV1 { return r.estado }
func (r ResultadoExperienciaV1) Fase() FaseResultadoExperienciaV1     { return r.fase }
func (r ResultadoExperienciaV1) Vinculos() VinculosResultadoExperienciaV1 {
	return r.vinculos
}
func (r ResultadoExperienciaV1) Seleccion() SeleccionResultadoExperienciaV1 {
	return clonarSeleccionResultadoV1(r.seleccion)
}
func (r ResultadoExperienciaV1) Intervalos() []IntervaloAplicacionResultadoExperienciaV1 {
	return append([]IntervaloAplicacionResultadoExperienciaV1(nil), r.intervalos...)
}
func (r ResultadoExperienciaV1) Aplicaciones() []AplicacionCalculadaResultadoExperienciaV1 {
	return append([]AplicacionCalculadaResultadoExperienciaV1(nil), r.aplicaciones...)
}
func (r ResultadoExperienciaV1) Reglas() []ResultadoReglaExperienciaV1 {
	return append([]ResultadoReglaExperienciaV1(nil), r.reglas...)
}
func (r ResultadoExperienciaV1) Secciones() []SubtotalSeccionResultadoExperienciaV1 {
	return append([]SubtotalSeccionResultadoExperienciaV1(nil), r.secciones...)
}
func (r ResultadoExperienciaV1) Total() (baremacion.Puntos, bool) {
	return r.total, r.tieneTotal
}
func (r ResultadoExperienciaV1) Bloqueos() []BloqueoResultadoExperienciaV1 {
	return clonarBloqueosResultadoV1(r.bloqueos)
}

// Validar comprueba estructura, presencia por fase y aritmetica registrada.
// No reinterpreta criterios, divisor temporal ni umbral de jornada: esa
// verificacion exige el PlanExperiencia y la EntradaExperiencia exactos ligados
// y se realizara reejecutando el unico motor o comparando su prueba confiable.
func (r ResultadoExperienciaV1) Validar() error { return validarResultadoExperienciaV1(r) }

func (r ResultadoExperienciaV1) clonar() ResultadoExperienciaV1 {
	r.seleccion = clonarSeleccionResultadoV1(r.seleccion)
	r.intervalos = append([]IntervaloAplicacionResultadoExperienciaV1(nil), r.intervalos...)
	r.aplicaciones = append([]AplicacionCalculadaResultadoExperienciaV1(nil), r.aplicaciones...)
	r.reglas = append([]ResultadoReglaExperienciaV1(nil), r.reglas...)
	r.secciones = append([]SubtotalSeccionResultadoExperienciaV1(nil), r.secciones...)
	r.bloqueos = clonarBloqueosResultadoV1(r.bloqueos)
	return r
}

func clonarSeleccionResultadoV1(s SeleccionResultadoExperienciaV1) SeleccionResultadoExperienciaV1 {
	s.aplicaciones = append([]AplicacionSeleccionResultadoExperienciaV1(nil), s.aplicaciones...)
	s.descartes = append([]DescarteSeleccionResultadoExperienciaV1(nil), s.descartes...)
	s.sinCoincidencia = append([]SinCoincidenciaResultadoExperienciaV1(nil), s.sinCoincidencia...)
	return s
}

func clonarBloqueosResultadoV1(origen []BloqueoResultadoExperienciaV1) []BloqueoResultadoExperienciaV1 {
	resultado := make([]BloqueoResultadoExperienciaV1, len(origen))
	for indice := range origen {
		resultado[indice] = origen[indice]
		resultado[indice].tramos = append(
			[]reglasbaremo.ReferenciaVersionada(nil),
			origen[indice].tramos...,
		)
		resultado[indice].reglas = append([]string(nil), origen[indice].reglas...)
	}
	return resultado
}
