package ports

import (
	"strconv"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
)

const (
	DominioCanonContenidoDetalleRRHH  = "vec.contratacion_temporal.resultado_rrhh.contenido_detalle.v1"
	cabeceraCanonContenidoDetalleRRHH = "VEC-CT-CONTENIDO-DETALLE-RRHH-V1\n"
)

// ExportacionCanonicaContenidoDetalleRRHH conserva el detalle reducido antes
// de registrar la lectura. Se construye desde la misma entrada nominal opaca
// que posteriormente recibirá el ReciboLecturaRRHH; no introduce otro DTO ni
// una vía alternativa de autorización.
type ExportacionCanonicaContenidoDetalleRRHH struct {
	exportacionCanonicaRRHH
	generadaEn time.Time
}

// ExportarContenidoCanonicoParaSQL reconstruye y valida todas las invariantes
// no probatorias de la entrada. El instante pertenece al corte de lectura y
// debe ser igual o posterior a la última actualización del expediente.
//
// Formato V1, siempre en este orden:
//   - cabecera ASCII terminada en LF;
//   - los quince campos completos del resumen y los cuatro de la solicitud;
//   - máscara privada de bloques;
//   - por bloque: presencia, secuencia privada del hito y campos funcionales;
//   - cardinalidad y ocho campos de cada hito, en orden durable.
//
// Usa el encuadre «longitud UTF-8 decimal:valor LF» compartido con cuadro. Las
// ausencias son 0 y las presencias 1; enteros, importes y secuencias se
// representan en decimal canónico. No contiene recibos, actores, documentos,
// campos libres ni JSON.
func (e EntradaDetalleExpedienteRRHHMinimizada) ExportarContenidoCanonicoParaSQL(
	generadaEn time.Time,
) (ExportacionCanonicaContenidoDetalleRRHH, error) {
	detalle, err := reconstruirDetalleExpedienteRRHHMinimizado(e)
	if err != nil || !domain.InstanteUTCCanonico(generadaEn) ||
		generadaEn.Before(detalle.Resumen.ActualizadoEn) {
		return ExportacionCanonicaContenidoDetalleRRHH{},
			ErrResultadoConsultaRRHHNoConfiable
	}
	constructor := nuevoConstructorCanonResultadoRRHH(
		cabeceraCanonContenidoDetalleRRHH,
	)
	constructor.resumen(detalle.Resumen)
	constructor.solicitud(detalle.Solicitud)
	constructor.enteroSinSigno(uint64(detalle.bloques))
	constructor.bloqueAnalisis(
		detalle.Analisis,
		e.referenciaAnalisis.secuencia,
	)
	constructor.bloqueCobertura(
		detalle.Cobertura,
		e.referenciaCobertura.secuencia,
	)
	constructor.bloqueAsignacion(
		detalle.Asignacion,
		e.referenciaAsignacion.secuencia,
	)
	constructor.enteroSinSigno(uint64(len(detalle.Hitos)))
	for _, hito := range detalle.Hitos {
		constructor.hito(hito)
	}
	canon, err := constructor.finalizar()
	if err != nil {
		return ExportacionCanonicaContenidoDetalleRRHH{},
			ErrResultadoConsultaRRHHNoConfiable
	}
	exportacion, err := nuevaExportacionCanonicaResultadoRRHH(
		DominioCanonContenidoDetalleRRHH,
		VersionCanonContenidoResultadoRRHH,
		canon,
	)
	if err != nil {
		return ExportacionCanonicaContenidoDetalleRRHH{}, err
	}
	resultado := ExportacionCanonicaContenidoDetalleRRHH{
		exportacionCanonicaRRHH: exportacion,
		generadaEn:              generadaEn,
	}
	if !resultado.valida() {
		return ExportacionCanonicaContenidoDetalleRRHH{},
			ErrResultadoConsultaRRHHNoConfiable
	}
	return resultado, nil
}

// ExportarResultadoCanonicoParaSQL encuadra la evidencia tipada de detalle.
// La cardinalidad siempre es uno y un detalle nunca transporta cursor.
func (e ExportacionCanonicaContenidoDetalleRRHH) ExportarResultadoCanonicoParaSQL() (
	ExportacionCanonicaResultadoConsultaRRHH,
	error,
) {
	if !e.valida() {
		return ExportacionCanonicaResultadoConsultaRRHH{},
			ErrResultadoConsultaRRHHNoConfiable
	}
	return nuevaExportacionCanonicaResultadoConsultaRRHH(
		tipoResultadoDetalleRRHH,
		e.generadaEn,
		1,
		e.HuellaSHA256(),
		"",
	)
}

func (e ExportacionCanonicaContenidoDetalleRRHH) valida() bool {
	return domain.InstanteUTCCanonico(e.generadaEn) &&
		exportacionCanonicaResultadoRRHHValida(
			e.exportacionCanonicaRRHH,
			DominioCanonContenidoDetalleRRHH,
			VersionCanonContenidoResultadoRRHH,
		)
}

func (c *constructorCanonResultadoRRHH) solicitud(
	s SolicitudOperativaRRHH,
) {
	c.texto(s.GrupoSubgrupo)
	c.texto(string(s.MotivoClave))
	c.instante(s.PeriodoInicio)
	c.instante(s.PeriodoFin)
}

func (c *constructorCanonResultadoRRHH) bloqueAnalisis(
	a *AnalisisOperativoRRHH,
	secuencia uint64,
) {
	c.booleano(a != nil)
	c.enteroSinSigno(secuencia)
	if a == nil {
		return
	}
	c.texto(string(a.ModalidadClave))
	c.texto(a.CategoriaRef)
	c.texto(string(a.CausaClave))
	c.instante(a.PeriodoInicio)
	c.instante(a.PeriodoFin)
	c.enteroSinSigno(uint64(a.PorcentajeJornada))
	c.texto(string(a.ResultadoRC))
	c.booleano(a.CostePrevisto != nil)
	if a.CostePrevisto != nil {
		c.enteroConSigno(a.CostePrevisto.Centimos)
		c.texto(a.CostePrevisto.Moneda)
	}
	c.texto(a.FuenteCosteRef)
}

func (c *constructorCanonResultadoRRHH) bloqueCobertura(
	cobertura *CoberturaOperativaRRHH,
	secuencia uint64,
) {
	c.booleano(cobertura != nil)
	c.enteroSinSigno(secuencia)
	if cobertura == nil {
		return
	}
	c.texto(string(cobertura.ViaClave))
	c.booleano(cobertura.DecisionGobernada)
	c.texto(cobertura.ProcedimientoRef)
	c.texto(cobertura.BolsaRef)
	c.enteroSinSigno(uint64(len(cobertura.Comprobaciones)))
	for _, comprobacion := range cobertura.Comprobaciones {
		c.texto(string(comprobacion.Clave))
		c.texto(string(comprobacion.Resultado))
	}
}

func (c *constructorCanonResultadoRRHH) bloqueAsignacion(
	a *AsignacionOperativaRRHH,
	secuencia uint64,
) {
	c.booleano(a != nil)
	c.enteroSinSigno(secuencia)
	if a == nil {
		return
	}
	c.texto(a.UnidadRef)
	c.instante(a.AsignadaEn)
	c.texto(string(a.MotivoClave))
}

func (c *constructorCanonResultadoRRHH) hito(h HitoExpedienteRRHH) {
	c.enteroSinSigno(h.Secuencia)
	c.enteroSinSigno(h.VersionExpediente)
	c.texto(string(h.AccionClave))
	c.instante(h.RealizadaEn)
	c.texto(string(h.FaseOrigen))
	c.texto(string(h.FaseDestino))
	c.texto(string(h.EstadoOrigen))
	c.texto(string(h.EstadoDestino))
}

func (c *constructorCanonResultadoRRHH) enteroConSigno(valor int64) {
	c.texto(strconv.FormatInt(valor, 10))
}
