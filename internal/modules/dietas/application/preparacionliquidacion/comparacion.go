package preparacionliquidacion

import (
	"errors"

	"vec-diputacion-granada/internal/modules/dietas/domain"
)

const EsquemaComparacionLiquidacion = "vec_dietas_comparacion_liquidacion_v1"

var ErrComparacionLiquidacion = errors.New("comparacion_liquidacion_invalida")

type EntradaComparacion struct {
	Esquema   string                                 `json:"esquema"`
	Anterior  domain.InstantaneaLiquidacionPropuesta `json:"anterior"`
	Propuesta domain.InstantaneaLiquidacionPropuesta `json:"propuesta"`
}

type FuentesComparacion struct {
	CatalogoRef     string `json:"catalogo_ref"`
	CatalogoVersion string `json:"catalogo_version"`
	CatalogoSHA256  string `json:"catalogo_sha256"`
	SnapshotSHA256  string `json:"snapshot_sha256"`
}

type ImportesComparacion struct {
	OriginalCentimos             int64 `json:"original_centimos"`
	ReconocidoAnteriorCentimos   int64 `json:"reconocido_anterior_centimos"`
	ReconocidoPropuestoCentimos  int64 `json:"reconocido_propuesto_centimos"`
	DiferenciaReconocidoCentimos int64 `json:"diferencia_reconocido_centimos"`
	RechazadoAnteriorCentimos    int64 `json:"rechazado_anterior_centimos"`
	RechazadoPropuestoCentimos   int64 `json:"rechazado_propuesto_centimos"`
	DiferenciaRechazadoCentimos  int64 `json:"diferencia_rechazado_centimos"`
}

type LineaComparacion struct {
	Indice int `json:"indice"`
	ImportesComparacion
	ReglaAnteriorRef      string `json:"regla_anterior_ref"`
	ReglaPropuestaRef     string `json:"regla_propuesta_ref"`
	MotivoAnteriorCodigo  string `json:"motivo_anterior_codigo"`
	MotivoPropuestoCodigo string `json:"motivo_propuesto_codigo"`
	CambioRegla           bool   `json:"cambio_regla"`
	CambioMotivo          bool   `json:"cambio_motivo"`
}

// ComparacionLiquidacion describe diferencias locales, sin registrar decisiones.
// Las huellas identifican datos coherentes; no prueban aprobación ni autenticidad.
type ComparacionLiquidacion struct {
	Esquema            string              `json:"esquema"`
	Procedencia        string              `json:"procedencia"`
	Liquidable         bool                `json:"liquidable"`
	ComisionRef        string              `json:"comision_ref"`
	ComisionVersion    int64               `json:"comision_version"`
	DocumentoSHA256    string              `json:"documento_sha256"`
	Anterior           FuentesComparacion  `json:"anterior"`
	Propuesta          FuentesComparacion  `json:"propuesta"`
	CatalogosDistintos bool                `json:"catalogos_distintos"`
	Lineas             []LineaComparacion  `json:"lineas"`
	Totales            ImportesComparacion `json:"totales"`
}

// Comparar recupera ambas propuestas antes de cotejarlas por índice documental.
// No recalcula tarifas ni usa la propuesta como una liquidación aprobada.
func Comparar(e EntradaComparacion) (*ComparacionLiquidacion, error) {
	if e.Esquema != EsquemaComparacionLiquidacion {
		return nil, ErrComparacionLiquidacion
	}
	a, err := Recuperar(e.Anterior)
	if err != nil {
		return nil, err
	}
	p, err := Recuperar(e.Propuesta)
	if err != nil {
		return nil, err
	}
	anterior, propuesta := a.Instantanea(), p.Instantanea()
	if anterior.ComisionRef != propuesta.ComisionRef || anterior.ComisionVersion != propuesta.ComisionVersion || anterior.DocumentoSHA256 != propuesta.DocumentoSHA256 {
		return nil, ErrComparacionLiquidacion
	}
	c := &ComparacionLiquidacion{
		Esquema: EsquemaComparacionLiquidacion, Procedencia: "comparacion_local_sin_registrar",
		ComisionRef: anterior.ComisionRef, ComisionVersion: anterior.ComisionVersion, DocumentoSHA256: anterior.DocumentoSHA256,
		Anterior: fuentesComparacion(anterior), Propuesta: fuentesComparacion(propuesta),
		CatalogosDistintos: anterior.CatalogoSHA256 != propuesta.CatalogoSHA256,
		Lineas:             make([]LineaComparacion, len(anterior.Lineas)),
		Totales:            compararImportes(anterior.Totales.OriginalCentimos, anterior.Totales.ReconocidoPropuestoCentimos, propuesta.Totales.ReconocidoPropuestoCentimos, anterior.Totales.RechazadoCentimos, propuesta.Totales.RechazadoCentimos),
	}
	for i, antes := range anterior.Lineas {
		despues := propuesta.Lineas[i]
		c.Lineas[i] = LineaComparacion{
			Indice:              antes.Indice,
			ImportesComparacion: compararImportes(antes.OriginalCentimos, antes.ReconocidoPropuestoCentimos, despues.ReconocidoPropuestoCentimos, antes.RechazadoCentimos, despues.RechazadoCentimos),
			ReglaAnteriorRef:    antes.ReglaRef, ReglaPropuestaRef: despues.ReglaRef,
			MotivoAnteriorCodigo: antes.MotivoCodigo, MotivoPropuestoCodigo: despues.MotivoCodigo,
			CambioRegla: antes.ReglaRef != despues.ReglaRef, CambioMotivo: antes.MotivoCodigo != despues.MotivoCodigo,
		}
	}
	return c, nil
}

func fuentesComparacion(s domain.InstantaneaLiquidacionPropuesta) FuentesComparacion {
	return FuentesComparacion{s.CatalogoRef, s.CatalogoVersion, s.CatalogoSHA256, s.SnapshotSHA256}
}

func compararImportes(original, reconocidoAnterior, reconocidoPropuesto, rechazadoAnterior, rechazadoPropuesto int64) ImportesComparacion {
	// Recuperar limita cada importe al intervalo [0, MaxInt64]; sus diferencias caben en int64.
	return ImportesComparacion{original, reconocidoAnterior, reconocidoPropuesto, reconocidoPropuesto - reconocidoAnterior, rechazadoAnterior, rechazadoPropuesto, rechazadoPropuesto - rechazadoAnterior}
}
