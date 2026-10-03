package application

import (
	"context"
	"errors"

	"vec-diputacion-granada/internal/modules/meritos/domain"
	"vec-diputacion-granada/internal/modules/meritos/ports"
	vec "vec-diputacion-granada/internal/vec/domain"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

// DenegacionConsultaReal concede prioridad al fallo técnico y la cancelación.
// Un error compuesto nunca transforma indisponibilidad en una decisión 403.
func DenegacionConsultaReal(err error) bool {
	return errors.Is(err, vec.ErrAutorizacionDenegada) &&
		!errors.Is(err, ports.ErrConsultaNoDisponible) &&
		!errors.Is(err, vecports.ErrIntentoAuditoriaNoDisponible) &&
		!errors.Is(err, vecports.ErrRegistroDenegacionAutorizacionLigadaV3NoDisponible) &&
		!errors.Is(err, vecports.ErrRegistroConcesionAutorizacionLigadaV3NoDisponible) &&
		!errors.Is(err, context.Canceled) && !errors.Is(err, context.DeadlineExceeded)
}

// RegistrarFalloConsulta registra un fallo de entrada HTTP con el contexto ya
// acreditado. El recurso es la referencia configurada del circuito; no copia
// cuerpo, selector inválido, cabeceras o diagnósticos del cliente.
func (s *ServicioConsultaPropia) RegistrarFalloConsulta(ctx context.Context, solicitud SolicitudConsultaPropia, causa error) error {
	if s == nil {
		return ports.ErrConsultaNoDisponible
	}
	recurso := s.auditoria.RecursoConsultaRef
	if !errors.Is(causa, ErrSolicitud) && domain.ReferenciaValida(solicitud.HechoRef) {
		recurso = solicitud.HechoRef
	}
	return s.registrarIntentoConsulta(ctx, solicitud, recurso, causa)
}

func (s *ServicioConsultaPropia) registrarIntentoConsulta(ctx context.Context, solicitud SolicitudConsultaPropia, recurso string, causa error) error {
	if s == nil || ctx == nil || causa == nil || nulo(s.auditoria.Registrador) ||
		nulo(s.auditoria.ValidadorMotivos) || nulo(s.reloj) || solicitud.validarContexto() != nil {
		return ports.ErrConsultaNoDisponible
	}
	correlacion, _ := solicitud.Correlacion.ValorCanonico()
	vinculo, vinculoErr := solicitud.Vinculo.Datos()
	ref, refErr := vecports.NuevaReferenciaIntentoAuditoria()
	resultado, motivo := vec.ResultadoIntentoAuditoriaError, s.auditoria.MotivoError
	if DenegacionConsultaReal(causa) {
		resultado, motivo = vec.ResultadoIntentoAuditoriaDenegado, s.auditoria.MotivoDenegacion
	}
	datos := vec.DatosIntentoAuditoria{Accion: AccionConsultaPropia, ModuloID: "meritos", RecursoRef: recurso,
		FinalidadRef: FinalidadConsultaPropia, Resultado: resultado, Motivo: motivo,
		Proceso: s.auditoria.Proceso, Canal: string(vinculo.Superficie), CorrelacionRef: correlacion}
	orden, ordenErr := vecports.NuevaOrdenIntentoAuditoria(ref, solicitud.Contexto, solicitud.Vinculo, datos)
	if vinculoErr != nil || refErr != nil || ordenErr != nil {
		return ports.ErrConsultaNoDisponible
	}
	// El intento original ya terminó. La auditoría tiene su propio plazo corto,
	// independiente de la cancelación del transporte.
	auditCtx, cancelar := context.WithTimeout(context.WithoutCancel(ctx), s.auditoria.Plazo)
	defer cancelar()
	if s.auditoria.ValidadorMotivos.ValidarReferenciaMotivoAutorizacionV2(auditCtx, motivo, s.reloj.Ahora().UTC()) != nil {
		return ports.ErrConsultaNoDisponible
	}
	acuse, auditErr := s.auditoria.Registrador.AppendIntentoAuditoria(auditCtx, orden)
	if auditErr != nil || acuse.ValidarPara(orden) != nil {
		return ports.ErrConsultaNoDisponible
	}
	return causa
}
