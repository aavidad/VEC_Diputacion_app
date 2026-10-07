package application

import (
	"context"
	"errors"
	"reflect"

	"vec-diputacion-granada/internal/modules/meritos/ports"
	vec "vec-diputacion-granada/internal/vec/domain"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

type Servicio struct {
	autorizador ports.Autorizador
	registro    ports.Registro
	auditoria   ports.AuditoriaIntentos
	reloj       vecports.Reloj
}

func NuevoServicio(a ports.Autorizador, r ports.Registro, audit ports.AuditoriaIntentos, reloj vecports.Reloj) (*Servicio, error) {
	if nulo(a) || nulo(r) || nulo(audit) || nulo(reloj) {
		return nil, ports.ErrRegistroNoDisponible
	}
	return &Servicio{a, r, audit, reloj}, nil
}

func (s *Servicio) Declarar(ctx context.Context, solicitud Solicitud) (ports.Recibo, error) {
	return s.ejecutar(ctx, solicitud, accionDeclarar)
}
func (s *Servicio) Verificar(ctx context.Context, solicitud Solicitud) (ports.Recibo, error) {
	return s.ejecutar(ctx, solicitud, accionVerificar)
}
func (s *Servicio) Rechazar(ctx context.Context, solicitud Solicitud) (ports.Recibo, error) {
	return s.ejecutar(ctx, solicitud, accionRechazar)
}
func (s *Servicio) Rectificar(ctx context.Context, solicitud Solicitud) (ports.Recibo, error) {
	return s.ejecutar(ctx, solicitud, accionRectificar)
}

func (s *Servicio) ejecutar(ctx context.Context, solicitud Solicitud, accion string) (recibo ports.Recibo, err error) {
	if s == nil || ctx == nil || nulo(s.autorizador) || nulo(s.registro) || nulo(s.auditoria) || nulo(s.reloj) {
		return ports.Recibo{}, ports.ErrRegistroNoDisponible
	}
	if err = ctx.Err(); err != nil {
		return ports.Recibo{}, err
	}
	// La frontera común audita autenticación/contexto inválidos. Aquí no se
	// escriben actores o correlaciones libres que no superan su validación nominal.
	if solicitud.validarContexto() != nil {
		return ports.Recibo{}, errors.Join(vec.ErrAutorizacionDenegada, ErrSolicitud)
	}
	ahora := s.reloj.Ahora().UTC()
	finalidad, _ := finalidadAudiencia(accion)
	correlacion, _ := solicitud.Correlacion.ValorCanonico()
	audit := vec.AuditEntry{ActorID: solicitud.Contexto.Contexto.PersonaRef, ActorProfile: solicitud.Contexto.Contexto.PerfilActivoRef,
		Action: accion, ModuleID: "meritos", Purpose: finalidad,
		CorrelationRef: correlacion, OccurredAt: ahora}
	auditoriaConfirmada := false
	defer func() {
		if err != nil && !auditoriaConfirmada {
			audit.Result = "no_confirmado"
			if _, auditErr := s.auditoria.AppendAudit(ctx, audit); auditErr != nil {
				recibo, err = ports.Recibo{}, ports.ErrRegistroNoDisponible
			}
		}
	}()
	// Un comando rechazado solo audita contexto nominal y metadatos fijos.
	// Su cuerpo no se añade a la auditoría hasta superar la validación de negocio.
	if solicitud.validarComando() != nil {
		return ports.Recibo{}, ErrSolicitud
	}
	audit.SubjectRef, audit.ObjectVersion, audit.RuleRef = solicitud.Hecho.Referencia, solicitud.Hecho.Version, solicitud.Motivo.EntradaClave
	orden, err := ordenSolicitud(solicitud, accion)
	if err != nil {
		return ports.Recibo{}, err
	}
	orden.Autorizacion, err = s.autorizar(ctx, solicitud, orden)
	if err != nil {
		return ports.Recibo{}, err
	}
	audit.AuthorizationRef = orden.Autorizacion.Material.ResumenCapacidad().DecisionRef()
	if accion == accionVerificar {
		return ports.Recibo{}, ErrAcreditacionPendiente
	}
	resultado, err := s.registro.EjecutarOperacion(ctx, orden)
	if err != nil {
		return ports.Recibo{}, err
	}
	if err := ValidarResultadoOperacion(orden, resultado); err != nil {
		return ports.Recibo{}, err
	}
	auditoriaConfirmada = true
	if resultado.Codigo != "confirmada" {
		return ports.Recibo{}, errorResultadoOperacion(resultado.Codigo)
	}
	return copiarRecibo(*resultado.Recibo), nil
}

func copiarRecibo(recibo ports.Recibo) ports.Recibo {
	recibo.Registro.Hecho = copiarHecho(recibo.Registro.Hecho)
	return recibo
}

func nulo(x any) bool {
	if x == nil {
		return true
	}
	v := reflect.ValueOf(x)
	switch v.Kind() {
	case reflect.Pointer, reflect.Interface, reflect.Map, reflect.Slice, reflect.Func, reflect.Chan:
		return v.IsNil()
	}
	return false
}
