package bootstrap

import (
	"context"
	"errors"

	bp "vec-diputacion-granada/internal/modules/bolsa/ports"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/httpinterno"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/application/resultadobolsa"
	ct "vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

// CT201 consume y registra la consulta nominal de detalle, y lee los hechos
// B98 en esa misma transacción. El servicio sólo devuelve el resultado tras
// confirmar ambos efectos; este puente no abre una conexión de Bolsa.
type lectorResultadoEmisionCTDesdeDetalle struct {
	soporte   *soporteAltaContratacionTemporalDesarrollo
	fronteras catalogoFronterasComunDesarrollo
	detalle   httpinterno.ConsultorDetalleRRHH
}

func (l *lectorResultadoEmisionCTDesdeDetalle) LeerResultadoEmisionCT(
	ctx context.Context, bolsaRef, llamamientoRef, reciboRef string,
) (bp.ResultadoEmisionCT, error) {
	vacia := bp.ResultadoEmisionCT{}
	if l == nil || l.soporte == nil || l.fronteras.identidad == nil ||
		dependenciaEsNulaContratacionTemporalDesarrollo(l.detalle) || ctx == nil ||
		!bp.ReferenciaOpacaLlamamientoValida(bolsaRef) ||
		!bp.ReferenciaOpacaLlamamientoValida(llamamientoRef) ||
		!bp.ReferenciaOpacaLlamamientoValida(reciboRef) {
		return vacia, bp.ErrResultadoEmisionCTNoDisponible
	}
	if err := ctx.Err(); err != nil {
		return vacia, err
	}
	capacidad, autorizada := l.soporte.capacidadValida(ctx)
	if !autorizada || (capacidad.ruta != httpinterno.RutaResolucionComunicacionLlamamiento &&
		capacidad.ruta != httpinterno.RutaPropuestaFormalizacion) {
		return vacia, bp.ErrResultadoEmisionCTDenegado
	}
	preparacion, presente := ctx.Value(clavePreparacionLlamamientoDesarrollo{}).(preparacionLlamamientoDesarrollo)
	expediente := preparacion.expediente.Fiscalizado
	if !presente || expediente.Validar() != nil || expediente.OrganizacionRef != organizacionAltaContratacionTemporalDesarrollo {
		return vacia, bp.ErrResultadoEmisionCTDenegado
	}
	solicitud, err := ct.NuevaSolicitudDetalleRRHH(expediente.Referencia, 0)
	if err != nil {
		return vacia, bp.ErrResultadoEmisionCTNoDisponible
	}
	lectura, err := subconsultaDetalleContratacionTemporalDesarrollo(ctx, l.soporte, l.fronteras)
	if err != nil {
		if ctx.Err() != nil {
			return vacia, ctx.Err()
		}
		return vacia, bp.ErrResultadoEmisionCTDenegado
	}
	var cursor string
	for pagina := 0; pagina < 256; pagina++ {
		paginaCtx := resultadobolsa.ConResultadoBolsaRRHH(lectura)
		if cursor != "" {
			paginaCtx = resultadobolsa.ConResultadoBolsaRRHHPagina(lectura, cursor)
		}
		if paginaCtx == nil {
			return vacia, bp.ErrResultadoEmisionCTDivergente
		}
		detalle, e := l.detalle.Consultar(paginaCtx, solicitud)
		if ctx.Err() != nil {
			return vacia, ctx.Err()
		}
		if e != nil {
			if errors.Is(e, ct.ErrAutorizacionDenegada) {
				return vacia, bp.ErrResultadoEmisionCTDenegado
			}
			return vacia, bp.ErrResultadoEmisionCTNoDisponible
		}
		if detalle.Resumen.ExpedienteRef != expediente.Referencia ||
			detalle.Resumen.OrganizacionRef != expediente.OrganizacionRef || detalle.ResultadoBolsa == nil {
			return vacia, bp.ErrResultadoEmisionCTDivergente
		}
		for _, vinculo := range detalle.ResultadoBolsa.Vinculos {
			if vinculo.BolsaRef == bolsaRef && vinculo.LlamamientoRef == llamamientoRef &&
				vinculo.ReciboEmisionRef == reciboRef {
				return resultadoEmisionCTDesdeVinculoRRHH(vinculo), nil
			}
		}
		if detalle.ResultadoBolsa.SiguienteCursor == nil {
			return vacia, bp.ErrResultadoEmisionCTNoEncontrado
		}
		siguiente := *detalle.ResultadoBolsa.SiguienteCursor
		if siguiente == cursor || !resultadobolsa.CursorResultadoBolsaRRHHValido(siguiente) {
			return vacia, bp.ErrResultadoEmisionCTDivergente
		}
		cursor = siguiente
	}
	return vacia, bp.ErrResultadoEmisionCTNoDisponible
}

func resultadoEmisionCTDesdeVinculoRRHH(v ct.VinculoResultadoBolsaRRHH) bp.ResultadoEmisionCT {
	r := bp.ResultadoEmisionCT{BolsaRef: v.BolsaRef, LlamamientoRef: v.LlamamientoRef,
		ReciboEmisionRef: v.ReciboEmisionRef, EmitidoEn: v.EmitidoEn,
		Participaciones: make([]bp.ResultadoParticipacionEmisionCT, 0, len(v.Participaciones))}
	for _, p := range v.Participaciones {
		r.Participaciones = append(r.Participaciones, bp.ResultadoParticipacionEmisionCT{
			ParticipacionRef: p.ParticipacionRef, Respuesta: valorResultadoEmisionCT(p.Respuesta),
			Modo: valorResultadoEmisionCT(p.Modo), ReciboRespuestaRef: valorResultadoEmisionCT(p.ReciboRespuestaRef),
			RespondidaEn: p.RespondidaEn, JustificanteRef: valorResultadoEmisionCT(p.JustificanteRef),
			ContactoResultado: valorResultadoEmisionCT(p.ContactoResultado), ReciboContactoRef: valorResultadoEmisionCT(p.ReciboContactoRef),
			ContactoEn: p.ContactoEn, SituacionActual: valorResultadoEmisionCT(p.SituacionActual),
			ReciboSituacionRef: valorResultadoEmisionCT(p.ReciboSituacionRef), SituacionDesde: p.SituacionDesde,
		})
	}
	return r
}

func valorResultadoEmisionCT(v *string) string {
	if v == nil {
		return ""
	}
	return *v
}

var _ bp.LectorResultadoEmisionCT = (*lectorResultadoEmisionCTDesdeDetalle)(nil)
