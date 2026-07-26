package application

import (
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/cobertura"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
)

// DatosReciboDecisionCoberturaParaAdaptador es la proyección mínima que una
// frontera puede serializar una vez que aplicación ha comprobado el recibo
// contra la consulta terminal exacta de la operación.
type DatosReciboDecisionCoberturaParaAdaptador struct {
	ReciboRef            string
	Estado               string
	DecisionCoberturaRef string
	VersionResultante    uint64
	ConfirmadaEn         time.Time
}

// ResultadoDecisionCoberturaParaAdaptador no se puede fabricar fuera de
// application: solo se crea desde un recibo validado para su consulta terminal.
type ResultadoDecisionCoberturaParaAdaptador struct {
	datos  DatosReciboDecisionCoberturaParaAdaptador
	recibo cobertura.ReciboOperacionDecisionCobertura
	sello  string
}

func nuevoResultadoDecisionCoberturaParaAdaptador(
	recibo cobertura.ReciboOperacionDecisionCobertura,
) (ResultadoDecisionCoberturaParaAdaptador, error) {
	// Esta función es privada y solo la invoca ejecutarParaAdaptador después de
	// ejecutarValidada: esa ruta obtiene el recibo de la consulta terminal
	// exacta o de la orden transaccional ya cotejada. La frontera nunca recibe
	// el recibo bruto antes de esta minimización.
	if !domain.ReferenciaOpacaValida(recibo.ReciboRef) || !domain.InstanteUTCCanonico(recibo.ConfirmadaEn) {
		return ResultadoDecisionCoberturaParaAdaptador{}, ErrConfirmacionDecisionCoberturaNoConfiable
	}
	datos := DatosReciboDecisionCoberturaParaAdaptador{ReciboRef: recibo.ReciboRef, ConfirmadaEn: recibo.ConfirmadaEn}
	if aplicado, ok := recibo.ResultadoAplicado(); ok {
		if !domain.ReferenciaOpacaValida(aplicado.DecisionCoberturaRef) || aplicado.VersionResultante == 0 || aplicado.VersionResultante > cobertura.MaximoEnteroSeguroOperacionDecisionCobertura {
			return ResultadoDecisionCoberturaParaAdaptador{}, ErrConfirmacionDecisionCoberturaNoConfiable
		}
		datos.Estado = "aplicada"
		datos.DecisionCoberturaRef = aplicado.DecisionCoberturaRef
		datos.VersionResultante = aplicado.VersionResultante
	} else if _, ok := recibo.ResultadoDenegadoVEC(); ok {
		datos.Estado = "denegada"
	} else {
		return ResultadoDecisionCoberturaParaAdaptador{}, ErrConfirmacionDecisionCoberturaNoConfiable
	}
	return ResultadoDecisionCoberturaParaAdaptador{datos: datos, recibo: recibo, sello: recibo.ReciboRef}, nil
}

func nuevoResultadoDecisionCoberturaParaAdaptadorDesdeConsulta(
	consulta cobertura.SolicitudConsultarOperacionDecisionCoberturaConfirmada,
	recibo cobertura.ReciboOperacionDecisionCobertura,
) (ResultadoDecisionCoberturaParaAdaptador, error) {
	if recibo.ValidarPara(consulta) != nil {
		return ResultadoDecisionCoberturaParaAdaptador{}, ErrConfirmacionDecisionCoberturaNoConfiable
	}
	return nuevoResultadoDecisionCoberturaParaAdaptador(recibo)
}

func (r ResultadoDecisionCoberturaParaAdaptador) DatosParaAdaptador() (DatosReciboDecisionCoberturaParaAdaptador, bool) {
	if r.sello == "" || r.datos.Estado == "" {
		return DatosReciboDecisionCoberturaParaAdaptador{}, false
	}
	return r.datos, true
}

func (r ResultadoDecisionCoberturaParaAdaptador) reciboParaAplicacion() cobertura.ReciboOperacionDecisionCobertura {
	return r.recibo
}
