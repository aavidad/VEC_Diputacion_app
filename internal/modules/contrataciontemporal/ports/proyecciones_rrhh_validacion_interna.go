package ports

import (
	"crypto/subtle"
)

// ValidarParaEjecucionInterna es la frontera productiva del cuadro RRHH. A la
// validación histórica añade la exigencia de recibo V2 y vuelve a calcular los
// canones de contenido y resultado a partir de la página recibida.
//
// De este modo, un recibo válido no puede reutilizarse sobre otra página, otra
// cardinalidad, otro cursor o un instante de generación distinto.
func (p PaginaCuadroRRHH) ValidarParaEjecucionInterna(
	orden OrdenConsultaCuadroRRHH,
) error {
	if p.ValidarPara(orden) != nil || p.Lectura.versionRecibo != 2 ||
		p.Lectura.evidenciaV2.generadaEn.Before(orden.instante) {
		return ErrResultadoConsultaRRHHNoConfiable
	}
	sinRecibo := p
	sinRecibo.Lectura = ReciboLecturaRRHH{}
	contenido, err := sinRecibo.ExportarContenidoCanonicoParaSQL()
	if err != nil {
		return ErrResultadoConsultaRRHHNoConfiable
	}
	resultado, err := contenido.ExportarResultadoCanonicoParaSQL()
	if err != nil || !p.Lectura.coincideConResultadoCuadroV2(resultado) {
		return ErrResultadoConsultaRRHHNoConfiable
	}
	return nil
}

func (r ReciboLecturaRRHH) coincideConResultadoCuadroV2(
	resultado ExportacionCanonicaResultadoConsultaRRHH,
) bool {
	return r.coincideConResultadoV2(resultado, tipoResultadoCuadroRRHH)
}

func (r ReciboLecturaRRHH) coincideConResultadoV2(
	resultado ExportacionCanonicaResultadoConsultaRRHH,
	tipoEsperado string,
) bool {
	if r.validarV2() != nil || !resultado.valida() ||
		resultado.tipoConsulta != tipoEsperado ||
		!r.evidenciaV2.generadaEn.Equal(resultado.generadaEn) ||
		r.evidenciaV2.total != resultado.total {
		return false
	}
	return huellasIgualesReciboRRHHV2(
		r.evidenciaV2.contenidoHuellaSHA256,
		resultado.contenidoHuellaSHA256,
	) && huellasIgualesReciboRRHHV2(
		r.evidenciaV2.resultadoHuellaSHA256,
		resultado.HuellaSHA256(),
	) && huellasIgualesOpcionalesReciboRRHHV2(
		r.evidenciaV2.cursorHuellaSHA256,
		resultado.cursorHuellaSHA256,
	)
}

func huellasIgualesReciboRRHHV2(izquierda, derecha string) bool {
	return huellaSHA256CanonicaRRHH(izquierda) &&
		huellaSHA256CanonicaRRHH(derecha) &&
		subtle.ConstantTimeCompare([]byte(izquierda), []byte(derecha)) == 1
}

func huellasIgualesOpcionalesReciboRRHHV2(izquierda, derecha string) bool {
	if izquierda == "" || derecha == "" {
		return izquierda == derecha
	}
	return huellasIgualesReciboRRHHV2(izquierda, derecha)
}

// ValidarParaEjecucionInterna reconstruye desde el detalle la misma entrada
// nominal minimizada que usa el adaptador y cruza contenido y resultado. V1 se
// conserva sólo en ValidarPara para compatibilidad histórica.
func (d DetalleExpedienteRRHH) ValidarParaEjecucionInterna(
	orden OrdenConsultaDetalleRRHH,
) error {
	if d.ValidarPara(orden) != nil || d.Lectura.versionRecibo != 2 ||
		d.Lectura.evidenciaV2.generadaEn.Before(orden.instante) {
		return ErrResultadoConsultaRRHHNoConfiable
	}
	entrada, err := d.entradaCanonicaMinimizada()
	if err != nil {
		return ErrResultadoConsultaRRHHNoConfiable
	}
	contenido, err := entrada.ExportarContenidoCanonicoParaSQL(
		d.Lectura.evidenciaV2.generadaEn,
	)
	if err != nil {
		return ErrResultadoConsultaRRHHNoConfiable
	}
	resultado, err := contenido.ExportarResultadoCanonicoParaSQL()
	if err != nil ||
		!d.Lectura.coincideConResultadoV2(resultado, tipoResultadoDetalleRRHH) {
		return ErrResultadoConsultaRRHHNoConfiable
	}
	return nil
}

func (d DetalleExpedienteRRHH) entradaCanonicaMinimizada() (
	EntradaDetalleExpedienteRRHHMinimizada,
	error,
) {
	var referenciaAnalisis ReferenciaHitoAnalisisRRHH
	if d.Analisis != nil {
		referenciaAnalisis.secuencia = d.Analisis.vinculo.secuencia
	}
	var referenciaCobertura ReferenciaHitoCoberturaRRHH
	if d.Cobertura != nil {
		referenciaCobertura.secuencia = d.Cobertura.vinculo.secuencia
	}
	var referenciaAsignacion ReferenciaHitoAsignacionRRHH
	if d.Asignacion != nil {
		referenciaAsignacion.secuencia = d.Asignacion.vinculo.secuencia
	}
	return NuevaEntradaDetalleExpedienteRRHHMinimizada(
		d.Resumen, d.Solicitud,
		d.Analisis, referenciaAnalisis,
		d.Cobertura, referenciaCobertura,
		d.Asignacion, referenciaAsignacion,
		d.Hitos,
	)
}
