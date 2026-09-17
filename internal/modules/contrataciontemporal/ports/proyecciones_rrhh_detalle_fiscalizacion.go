package ports

import (
	"time"
	"unicode/utf8"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
)

const claveReparoObservacionesFiscalizacion domain.ClaveCatalogo = "observaciones_fiscalizacion"

// ReparoFiscalizacionOperativaRRHH comunica el único reparo que el agregado
// conserva hoy: las observaciones registradas por fiscalización. La clave no
// pretende descomponer un texto libre en motivos que el dominio no conoce.
type ReparoFiscalizacionOperativaRRHH struct {
	Clave domain.ClaveCatalogo `json:"clave"`
	Texto string               `json:"texto"`
}

func (r ReparoFiscalizacionOperativaRRHH) validar() error {
	if r.Clave != claveReparoObservacionesFiscalizacion ||
		utf8.RuneCountInString(r.Texto) == 0 ||
		utf8.RuneCountInString(r.Texto) > 2000 {
		return ErrResultadoConsultaRRHHNoConfiable
	}
	return nil
}

// SubsanacionFiscalizacionOperativaRRHH conserva la actuación posterior al
// reparo vigente. No altera ni reinterpreta el resultado de fiscalización.
type SubsanacionFiscalizacionOperativaRRHH struct {
	RegistradaEn time.Time `json:"registrada_en"`
	Texto        string    `json:"texto"`
	vinculo      vinculoHitoOperativoRRHH
}

// FiscalizacionOperativaRRHH es el bloque opcional y atestado del canon V3.
// Excluye actores, retornos, recibos, unidades y documentos.
type FiscalizacionOperativaRRHH struct {
	ResultadoClave domain.ResultadoFiscalizacion          `json:"resultado_clave"`
	Reparos        []ReparoFiscalizacionOperativaRRHH     `json:"reparos"`
	RegistradaEn   time.Time                              `json:"registrada_en"`
	Subsanacion    *SubsanacionFiscalizacionOperativaRRHH `json:"subsanacion,omitempty"`
	vinculo        vinculoHitoOperativoRRHH
}

func (f FiscalizacionOperativaRRHH) validarDatos() error {
	if !f.ResultadoClave.Valido() || !domain.InstanteUTCCanonico(f.RegistradaEn) || len(f.Reparos) > 1 {
		return ErrResultadoConsultaRRHHNoConfiable
	}
	if f.ResultadoClave == domain.FiscalizacionFavorable {
		if len(f.Reparos) != 0 || f.Subsanacion != nil {
			return ErrResultadoConsultaRRHHNoConfiable
		}
	} else if len(f.Reparos) != 1 || f.Reparos[0].validar() != nil {
		return ErrResultadoConsultaRRHHNoConfiable
	}
	if f.Subsanacion == nil {
		return nil
	}
	if f.ResultadoClave != domain.FiscalizacionDesfavorable ||
		!domain.InstanteUTCCanonico(f.Subsanacion.RegistradaEn) ||
		utf8.RuneCountInString(f.Subsanacion.Texto) == 0 ||
		utf8.RuneCountInString(f.Subsanacion.Texto) > 2000 ||
		!f.Subsanacion.RegistradaEn.After(f.RegistradaEn) {
		return ErrResultadoConsultaRRHHNoConfiable
	}
	return nil
}

func (f FiscalizacionOperativaRRHH) validar() error {
	if f.validarDatos() != nil || f.vinculo.validar() != nil ||
		f.vinculo.accionClave != domain.AccionRegistrarFiscalizacion ||
		!f.RegistradaEn.Equal(f.vinculo.realizadaEn) {
		return ErrResultadoConsultaRRHHNoConfiable
	}
	if f.Subsanacion != nil && (f.Subsanacion.vinculo.validar() != nil ||
		f.Subsanacion.vinculo.accionClave != domain.AccionRegistrarSubsanacionReparo ||
		!f.Subsanacion.RegistradaEn.Equal(f.Subsanacion.vinculo.realizadaEn)) {
		return ErrResultadoConsultaRRHHNoConfiable
	}
	return nil
}

func fiscalizacionDesdeExpedienteRRHH(e domain.Expediente) *FiscalizacionOperativaRRHH {
	if e.Fiscalizacion == nil || e.Fiscalizacion.ActuacionRegistro == nil {
		return nil
	}
	f := e.Fiscalizacion
	proyeccion := &FiscalizacionOperativaRRHH{
		ResultadoClave: f.Resultado,
		RegistradaEn:   f.FiscalizadaEn,
		vinculo:        vinculoDesdeFiscalizacionRRHH(*f),
	}
	if f.Observaciones != "" {
		proyeccion.Reparos = []ReparoFiscalizacionOperativaRRHH{{
			Clave: claveReparoObservacionesFiscalizacion, Texto: f.Observaciones,
		}}
	}
	if f.Retorno == nil {
		return proyeccion
	}
	for _, actuacion := range e.Actuaciones {
		if actuacion.AccionClave != domain.AccionRegistrarSubsanacionReparo ||
			actuacion.RetornoRef != f.Retorno.RetornoRef {
			continue
		}
		proyeccion.Subsanacion = &SubsanacionFiscalizacionOperativaRRHH{
			RegistradaEn: actuacion.RealizadaEn,
			Texto:        actuacion.Observaciones, vinculo: vinculoDesdeActuacionRRHH(actuacion),
		}
		break
	}
	return proyeccion
}
