package ports

import (
	"bytes"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
)

// HistoriaRegistroIncorporacionV2 es una copia opaca validada, NO una prueba de
// commit. El adaptador durable debe leer orden/evidencia/recibo del mismo commit
// verificado. Rehidratar datos aportados por browser no acredita ese origen.
// La evidencia contiene la raíz REAL recibida del almacén, nunca una raíz creada
// por el servicio. El dominio comprueba el historial y su transición exacta.
type HistoriaRegistroIncorporacionV2 struct {
	original            OrdenConfirmacionIncorporacionV2
	recibo              ReciboRegistroIncorporacionV2
	definicion          domain.DefinicionSeguimiento
	anterior, posterior domain.Seguimiento
}

func NuevaHistoriaRegistroIncorporacionV2(o OrdenConfirmacionIncorporacionV2, r ReciboRegistroIncorporacionV2,
	definicion domain.PublicacionDefinicionSeguimiento, anterior, posterior domain.EstadoPersistidoSeguimiento,
) (HistoriaRegistroIncorporacionV2, error) {
	cero := HistoriaRegistroIncorporacionV2{}
	if r.ValidarPara(o) != nil {
		return cero, ErrRegistroIncorporacionV2
	}
	def, e := domain.RestaurarDefinicionSeguimiento(definicion)
	if e != nil {
		return cero, ErrRegistroIncorporacionV2
	}
	a, ea := domain.RehidratarSeguimiento(def, anterior)
	p, ep := domain.RehidratarSeguimiento(def, posterior)
	if ea != nil || ep != nil {
		return cero, ErrRegistroIncorporacionV2
	}
	d, _ := o.Material().Datos()
	if anterior.Referencia != r.SeguimientoRef || anterior.OrganizacionRef != d.Preparacion.OrganizacionRef ||
		anterior.ExpedienteRef != d.Confirmacion.SolicitudPersonal.ExpedienteRef || anterior.RelacionRef != d.Confirmacion.ResultadoPersonal.RelacionRef ||
		anterior.Version != r.VersionSeguimientoAnterior || posterior.Version != r.VersionSeguimientoResultante ||
		!def.Referencia().Coincide(r.DefinicionSeguimiento) || anterior.HuellaRaizSHA256 != r.HuellaRaizSeguimiento {
		return cero, ErrRegistroIncorporacionV2
	}
	esperado, e := a.Aplicar(def, anterior.Version, r.Transicion)
	ca, eac := domain.SerializarEstadoSeguimientoCanonico(def, a.Estado())
	cp, epc := domain.SerializarEstadoSeguimientoCanonico(def, p.Estado())
	ce, eec := domain.SerializarEstadoSeguimientoCanonico(def, esperado.Estado())
	if e != nil || eac != nil || epc != nil || eec != nil || !bytes.Equal(ce, cp) ||
		hashRegistroV2(ca) != r.HuellaEstadoAnterior || hashRegistroV2(cp) != r.HuellaEstadoResultante {
		return cero, ErrRegistroIncorporacionV2
	}
	return HistoriaRegistroIncorporacionV2{o, r.Copia(), def, a, p}, nil
}

func (h HistoriaRegistroIncorporacionV2) ReciboOriginal() ReciboRegistroIncorporacionV2 {
	return h.recibo.Copia()
}

// EvidenciaSeguimiento devuelve copias; no permite cambiar historia ya validada.
func (h HistoriaRegistroIncorporacionV2) EvidenciaSeguimiento() (domain.PublicacionDefinicionSeguimiento, domain.EstadoPersistidoSeguimiento, domain.EstadoPersistidoSeguimiento) {
	return h.definicion.Publicacion(), h.anterior.Estado(), h.posterior.Estado()
}

type ResultadoRegistroIncorporacionV2 struct {
	Recibo           ReciboRegistroIncorporacionV2
	Historia         HistoriaRegistroIncorporacionV2
	Recuperado       bool
	ConsumosActuales ConsumosRegistroIncorporacionV2
}

// ValidarPara distingue autoridad histórica y vigente. No elimina las ligaduras
// de la decisión/fecha originales ni revalida esa decisión caducada como actual.
func (r ResultadoRegistroIncorporacionV2) ValidarPara(o OrdenConfirmacionIncorporacionV2, ahora time.Time) error {
	h := r.Historia
	if o.ValidarEn(ahora) != nil || r.ConsumosActuales.validar(o, ahora) != nil ||
		!jsonIgualRegistroV2(r.Recibo, h.recibo) || h.recibo.Transicion.RegistradaEn.After(ahora) {
		return ErrRegistroIncorporacionV2
	}
	pub, a, p := h.EvidenciaSeguimiento()
	if _, e := NuevaHistoriaRegistroIncorporacionV2(h.original, h.recibo, pub, a, p); e != nil {
		return ErrRegistroIncorporacionV2
	}
	original, e := IntencionRegistroIncorporacionV2(h.original.Material())
	actual, ea := IntencionRegistroIncorporacionV2(o.Material())
	if e != nil || ea != nil || !bytes.Equal(original, actual) {
		return ErrRegistroIncorporacionV2
	}
	orig := h.original.Exportacion().ResumenCapacidad()
	fresh := o.Exportacion().ResumenCapacidad()
	if r.Recuperado {
		// Reutilizar la autorización original no es recuperación nominal fresca.
		if fresh.DecisionRef() == orig.DecisionRef() || fresh.EmitidaEn().Before(h.recibo.Transicion.RegistradaEn) ||
			o.evaluadaEn.Before(h.recibo.Transicion.RegistradaEn) ||
			r.ConsumosActuales.DecisionLecturaRef == h.recibo.ConsumosOriginales.DecisionLecturaRef ||
			r.ConsumosActuales.ConsumoCTSHA256 == h.recibo.ConsumosOriginales.ConsumoCTSHA256 ||
			r.ConsumosActuales.ConsumoLecturaSHA256 == h.recibo.ConsumosOriginales.ConsumoLecturaSHA256 ||
			r.ConsumosActuales.AuditoriaAD3CTRef == h.recibo.ConsumosOriginales.AuditoriaAD3CTRef ||
			r.ConsumosActuales.AuditoriaAD3LecturaRef == h.recibo.ConsumosOriginales.AuditoriaAD3LecturaRef ||
			r.ConsumosActuales.AuditoriaPersonalLecturaRef == h.recibo.ConsumosOriginales.AuditoriaPersonalLecturaRef {
			return ErrRegistroIncorporacionV2
		}
	} else {
		// Alta original: orden exacta, no una intención equivalente con otro contexto.
		hc, ec := o.Material().MaterialCanonico()
		he, ee := o.Exportacion().HuellaConjuntoSHA256()
		ho, eo := h.original.Exportacion().HuellaConjuntoSHA256()
		if ec != nil || ee != nil || eo != nil || he != ho || !bytes.Equal(hc, h.recibo.MaterialOriginalCanonico) ||
			!jsonIgualRegistroV2(r.ConsumosActuales, h.recibo.ConsumosOriginales) {
			return ErrRegistroIncorporacionV2
		}
	}
	return nil
}

func (r ResultadoRegistroIncorporacionV2) Copia() ResultadoRegistroIncorporacionV2 {
	r.Recibo = r.Recibo.Copia()
	return r
}
