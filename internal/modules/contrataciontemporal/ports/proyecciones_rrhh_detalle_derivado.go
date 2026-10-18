package ports

import (
	"strconv"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
)

// InstruccionCompartidaRRHH muestra la instrucción conservada en el hijo sin
// atribuirle las actuaciones ni los recibos originales del padre.
type InstruccionCompartidaRRHH struct {
	DerivacionRef       string               `json:"derivacion_ref"`
	PadreRef            string               `json:"padre_ref"`
	NumeroVisiblePadre  string               `json:"numero_visible_padre"`
	OfertaRef           string               `json:"oferta_ref"`
	NumeroPlaza         uint32               `json:"numero_plaza"`
	ReciboDerivacionRef string               `json:"recibo_derivacion_ref"`
	HuellaPadreSHA256   string               `json:"huella_padre_sha256"`
	CreadoEnPadre       time.Time            `json:"creado_en_padre"`
	ActualizadoEnPadre  time.Time            `json:"actualizado_en_padre"`
	HitosPadre          []HitoExpedienteRRHH `json:"hitos_padre"`
}

func (i InstruccionCompartidaRRHH) validarPara(d DetalleExpedienteRRHH) error {
	if !domain.ReferenciaOpacaValida(i.DerivacionRef) ||
		!domain.ReferenciaOpacaValida(i.PadreRef) || i.PadreRef == d.Resumen.ExpedienteRef ||
		!domain.ReferenciaOpacaValida(i.OfertaRef) ||
		!domain.ReferenciaOpacaValida(i.ReciboDerivacionRef) ||
		!domain.HuellaSHA256FirmaValida(i.HuellaPadreSHA256) ||
		i.NumeroPlaza == 0 || i.NumeroPlaza > 100 ||
		d.Resumen.NumeroVisible != i.NumeroVisiblePadre+"-P"+strconv.FormatUint(uint64(i.NumeroPlaza), 10) ||
		!domain.InstanteUTCCanonico(i.CreadoEnPadre) ||
		!domain.InstanteUTCCanonico(i.ActualizadoEnPadre) ||
		i.ActualizadoEnPadre.Before(i.CreadoEnPadre) ||
		d.Resumen.CreadoEn.Before(i.ActualizadoEnPadre) ||
		len(i.HitosPadre) < 6 || len(i.HitosPadre) > limiteMaximoHitosPorPresupuestoRRHH {
		return ErrResultadoConsultaRRHHNoConfiable
	}
	for n, h := range i.HitosPadre {
		if h.validar() != nil || h.Secuencia != uint64(n+1) ||
			h.RealizadaEn.After(i.ActualizadoEnPadre) {
			return ErrResultadoConsultaRRHHNoConfiable
		}
		if n == 0 {
			if h.FaseOrigen != "" || h.EstadoOrigen != domain.EstadoPendiente ||
				!h.RealizadaEn.Equal(i.CreadoEnPadre) {
				return ErrResultadoConsultaRRHHNoConfiable
			}
		} else if h.FaseOrigen != i.HitosPadre[n-1].FaseDestino ||
			h.EstadoOrigen != i.HitosPadre[n-1].EstadoDestino ||
			h.RealizadaEn.Before(i.HitosPadre[n-1].RealizadaEn) {
			return ErrResultadoConsultaRRHHNoConfiable
		}
	}
	ultimo := i.HitosPadre[len(i.HitosPadre)-1]
	primeroHijo := d.Hitos[0]
	if !ultimo.RealizadaEn.Equal(i.ActualizadoEnPadre) ||
		ultimo.FaseDestino != domain.FaseFiscalizacion ||
		ultimo.EstadoDestino != domain.EstadoEnCurso ||
		primeroHijo.AccionClave != domain.AccionDerivarNombramiento ||
		primeroHijo.FaseOrigen != ultimo.FaseDestino ||
		primeroHijo.EstadoOrigen != ultimo.EstadoDestino ||
		primeroHijo.FaseDestino != domain.FaseFiscalizacion ||
		primeroHijo.EstadoDestino != domain.EstadoEnCurso ||
		!primeroHijo.RealizadaEn.Equal(d.Resumen.CreadoEn) {
		return ErrResultadoConsultaRRHHNoConfiable
	}
	return nil
}

func (d DetalleExpedienteRRHH) hitosDeInstruccion() []HitoExpedienteRRHH {
	if d.InstruccionCompartida != nil {
		return d.InstruccionCompartida.HitosPadre
	}
	return d.Hitos
}

func clonarInstruccionCompartidaRRHH(i *InstruccionCompartidaRRHH) *InstruccionCompartidaRRHH {
	if i == nil {
		return nil
	}
	clon := *i
	clon.HitosPadre = clonarHitosRRHH(i.HitosPadre)
	return &clon
}
