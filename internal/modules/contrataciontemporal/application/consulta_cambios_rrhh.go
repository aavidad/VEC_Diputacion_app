package application

import (
	"context"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

// ConsultarCambios devuelve el histórico de cambios del expediente con valor
// anterior y nuevo (petición RRHH p.4). Exige la misma capacidad que el
// detalle completo: la proyección limitada de seguimiento no lo alcanza.
func (s *ServicioConsultaDetalleRRHH) ConsultarCambios(
	ctx context.Context, solicitud ports.SolicitudDetalleRRHH,
) (ports.ResultadoConsultaCambiosRRHH, error) {
	var cero ports.ResultadoConsultaCambiosRRHH
	if err := errorContextoConsultaRRHH(ctx); err != nil {
		return cero, err
	}
	if s == nil || dependenciaNula(s.autoridad) || dependenciaNula(s.emisor) ||
		dependenciaNula(s.sesion) || dependenciaNula(s.reloj) || solicitud.ExpedienteRef() == "" {
		return cero, ErrSolicitudConsultaRRHHInvalida
	}
	lector, disponible := s.sesion.(ports.SesionConsultaCambiosRRHH)
	if !disponible || dependenciaNula(lector) {
		return cero, ErrConsultaRRHHNoDisponible
	}
	contexto, err := s.autoridad.ResolverContextoConsultaRRHH(ctx)
	if errContexto := errorContextoConsultaRRHH(ctx); errContexto != nil {
		return cero, errContexto
	}
	if err != nil {
		return cero, normalizarFalloConsultaRRHH(err)
	}
	material, err := s.emisor.EmitirMaterialDetalleRRHH(ctx, contexto, solicitud)
	if errContexto := errorContextoConsultaRRHH(ctx); errContexto != nil {
		return cero, errContexto
	}
	if err != nil {
		return cero, normalizarFalloConsultaRRHH(err)
	}
	instanteCapacidad := s.reloj.Ahora()
	if !domain.InstanteUTCCanonico(instanteCapacidad) {
		return cero, ErrResultadoConsultaRRHHNoConfiable
	}
	capacidad, err := ports.NuevaCapacidadConsultaDetalleRRHH(contexto, material, solicitud, instanteCapacidad)
	if errContexto := errorContextoConsultaRRHH(ctx); errContexto != nil {
		return cero, errContexto
	}
	if err != nil {
		return cero, ErrResultadoConsultaRRHHNoConfiable
	}
	if capacidad.AutorizaCamposSeguimiento() {
		return cero, ErrConsultaRRHHNoObservable
	}
	instanteOrden := s.reloj.Ahora()
	if !domain.InstanteUTCCanonico(instanteOrden) || instanteOrden.Before(instanteCapacidad) ||
		instanteOrden.Before(capacidad.ValidaDesde()) || !instanteOrden.Before(capacidad.ValidaHasta()) {
		return cero, ErrResultadoConsultaRRHHNoConfiable
	}
	orden, err := ports.NuevaOrdenConsultaDetalleRRHH(contexto, capacidad, solicitud, instanteOrden)
	if err != nil {
		return cero, ErrResultadoConsultaRRHHNoConfiable
	}
	resultado, err := lector.ConsultarCambiosYRegistrar(ctx, orden)
	if errContexto := errorContextoConsultaRRHH(ctx); errContexto != nil {
		return cero, errContexto
	}
	if err != nil {
		return cero, normalizarFalloConsultaRRHH(err)
	}
	if resultado.ValidarPara(orden) != nil {
		return cero, ErrResultadoConsultaRRHHNoConfiable
	}
	resultado.Cambios = append([]ports.CambioExpedienteRRHH(nil), resultado.Cambios...)
	return resultado, nil
}
