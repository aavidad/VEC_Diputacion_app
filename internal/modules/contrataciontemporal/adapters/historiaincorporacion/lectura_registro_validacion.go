package historiaincorporacion

import (
	"bytes"
	"context"
	"encoding/json"
	"time"
	"unicode/utf8"
	dom "vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	ct "vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

// ValidarDocumentoRegistroOriginal valida transporte y ligaduras ordinarias.
// NO crea Orden/Decision ni acredita origen, MAC o permiso. CT77 es fuente TCB;
// Restaurador debe reconstruir autoridades y cotejar la orden original completa.
// ahora sólo limita fechas futuras: una autorización histórica caducada es válida.
func ValidarDocumentoRegistroOriginal(ctx context.Context, b []byte, s Selector, ahora time.Time) error {
	if ctx == nil {
		return ErrHistoria
	}
	if e := ctx.Err(); e != nil {
		return e
	}
	if !dom.InstanteUTCCanonico(ahora) || !dom.ReferenciaOpacaValida(s.ReciboRef) || !sha(s.MaterialSHA256) || !sha(s.IntencionSHA256) || len(b) > MaximoBytesDocumento || !utf8.Valid(b) {
		return ErrHistoria
	}
	d, e := decodificarDocumento(b)
	if e != nil {
		return ErrHistoria
	}
	ev, r := d.Evidencia, d.Recibo
	if d.Esquema != EsquemaDocumento || ev.Esquema != ct.EsquemaEvidenciaOrdenOriginalIncorporacionV2 || r.Esquema != ct.EsquemaReciboRegistroIncorporacionV2 ||
		r.Transicion.ReciboRef != s.ReciboRef || ev.MaterialSHA256 != s.MaterialSHA256 || ev.IntencionSHA256 != s.IntencionSHA256 || r.MaterialOriginalSHA256 != s.MaterialSHA256 || r.IntencionSHA256 != s.IntencionSHA256 || hash(ev.MaterialCanonico) != s.MaterialSHA256 || !bytes.Equal(ev.MaterialCanonico, r.MaterialOriginalCanonico) || ev.ContextoOriginalRef != r.ContextoOriginalRef ||
		!r.EjercicioSintetico || r.FirmaOficial || r.EficaciaAdministrativa {
		return ErrHistoria
	}
	for _, f := range []time.Time{ev.PreparadoEn, ev.EvaluadaEn, ev.ConcesionRegistradaEn, r.Transicion.RegistradaEn, r.ConsumosOriginales.ConsumidaCTEn, r.ConsumosOriginales.LeidaPersonalEn} {
		if !dom.InstanteUTCCanonico(f) || f.After(ahora) {
			return ErrHistoria
		}
	}
	c := r.ConsumosOriginales
	if ev.EvaluadaEn.Before(ev.PreparadoEn) || ev.ConcesionRegistradaEn.After(ev.EvaluadaEn) || c.ConsumidaCTEn.Before(ev.EvaluadaEn) || c.LeidaPersonalEn.Before(c.ConsumidaCTEn) || r.Transicion.RegistradaEn.Before(c.LeidaPersonalEn) || c.HuellaExportacionCT != ev.HuellaExportacionSHA256 {
		return ErrHistoria
	}
	if jsonValido(ev.MaterialCanonico, 8<<20) != nil || !utf8.Valid(ev.MaterialCanonico) {
		return ErrHistoria
	}
	var m materialJSON
	dec := json.NewDecoder(bytes.NewReader(ev.MaterialCanonico))
	dec.DisallowUnknownFields()
	if dec.Decode(&m) != nil {
		return ErrHistoria
	}
	canon, e := json.Marshal(m)
	if e != nil {
		return ErrHistoria
	}
	a, e := jsonOrdenado(canon)
	if e != nil {
		return ErrHistoria
	}
	z, e := jsonOrdenado(ev.MaterialCanonico)
	if e != nil || !bytes.Equal(a, z) || m.Esquema != ct.EsquemaMaterialConfirmacionIncorporacionV2 || m.Vinculo.Validar() != nil || !m.EjercicioSintetico || m.FirmaOficial || m.EficaciaAdministrativa ||
		m.Vinculo.RegistroContextoRef != ev.ContextoOriginalRef || !bytes.Equal(ev.ContextoActorCanonico, m.ContextoCanonico) || r.VersionSolicitudPersonal != m.Confirmacion.SolicitudPersonal.VersionExpediente || r.VersionActualExpediente != m.VersionActualExpediente || r.VersionSeguimientoAnterior != m.Confirmacion.VersionSeguimientoEsperada ||
		r.VersionSeguimientoAnterior >= ct.MaximoEnteroSeguroOperacionAnalisis || r.VersionSeguimientoResultante != r.VersionSeguimientoAnterior+1 {
		return ErrHistoria
	}
	t := r.Transicion
	if t.Calendario != nil || t.RectificaActuacionRef != "" || t.TransicionClave != ct.TransicionConfirmarIncorporacion || t.ActorRef != m.Preparacion.ActorRef || t.UnidadRef != m.Preparacion.UnidadRef || t.CorrelacionRef != m.Preparacion.CorrelacionRef || t.MotivoClave != m.Confirmacion.MotivoClave || t.Periodo == nil || *t.Periodo != m.Confirmacion.PeriodoIncorporacion || !t.EfectivoEn.Equal(m.Confirmacion.PeriodoIncorporacion.Desde) {
		return ErrHistoria
	}
	x, _ := json.Marshal(t.Documentos)
	y, _ := json.Marshal(m.Confirmacion.Documentos)
	if !bytes.Equal(x, y) {
		return ErrHistoria
	}
	for _, h := range []string{ev.SolicitudSHA256, c.HuellaExportacionCT, c.ConsumoCTSHA256, c.ConsumoLecturaSHA256, r.HuellaRaizSeguimiento, r.HuellaEstadoAnterior, r.HuellaEstadoResultante} {
		if !sha(h) {
			return ErrHistoria
		}
	}
	for _, ref := range []string{r.SeguimientoRef, r.AuditoriaCTRef, r.OutboxCTRef, t.ActuacionRef, c.DecisionLecturaRef, c.AuditoriaAD3CTRef, c.AuditoriaAD3LecturaRef, c.AuditoriaPersonalLecturaRef} {
		if !dom.ReferenciaOpacaValida(ref) {
			return ErrHistoria
		}
	}
	if c.DecisionCTRef == c.DecisionLecturaRef || c.DecisionCTRef == m.Personal.DecisionOriginalRef || c.DecisionLecturaRef == m.Personal.DecisionOriginalRef || c.ConsumoCTSHA256 == c.ConsumoLecturaSHA256 || c.AuditoriaAD3CTRef == c.AuditoriaAD3LecturaRef || c.AuditoriaPersonalLecturaRef == m.Personal.AuditoriaRef {
		return ErrHistoria
	}
	// Sólo reconstrucción estructural de exportación pública, nunca concesión.
	export, e := recuperarExportacion(ev)
	if e != nil || export.ResumenCapacidad().DecisionRef() != c.DecisionCTRef {
		return ErrHistoria
	}
	def, e := dom.RestaurarDefinicionSeguimiento(d.Publicacion)
	if e != nil || !def.Referencia().Coincide(r.DefinicionSeguimiento) {
		return ErrHistoria
	}
	ant, e := dom.RehidratarSeguimiento(def, d.Anterior)
	if e != nil {
		return ErrHistoria
	}
	pos, e := dom.RehidratarSeguimiento(def, d.Posterior)
	if e != nil {
		return ErrHistoria
	}
	if d.Anterior.Referencia != r.SeguimientoRef || d.Anterior.OrganizacionRef != m.Preparacion.OrganizacionRef || d.Anterior.ExpedienteRef != m.Confirmacion.SolicitudPersonal.ExpedienteRef || d.Anterior.RelacionRef != m.Confirmacion.ResultadoPersonal.RelacionRef || d.Anterior.Version != r.VersionSeguimientoAnterior || d.Posterior.Version != r.VersionSeguimientoResultante || d.Anterior.HuellaRaizSHA256 != r.HuellaRaizSeguimiento {
		return ErrHistoria
	}
	esperado, e := ant.Aplicar(def, d.Anterior.Version, t)
	if e != nil {
		return ErrHistoria
	}
	ca, ea := dom.SerializarEstadoSeguimientoCanonico(def, ant.Estado())
	cp, ep := dom.SerializarEstadoSeguimientoCanonico(def, pos.Estado())
	ce, ee := dom.SerializarEstadoSeguimientoCanonico(def, esperado.Estado())
	if ea != nil || ep != nil || ee != nil || !bytes.Equal(cp, ce) || hash(ca) != r.HuellaEstadoAnterior || hash(cp) != r.HuellaEstadoResultante {
		return ErrHistoria
	}
	return ctx.Err()
}
