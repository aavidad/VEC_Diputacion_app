package postgres

import (
	"context"
	"encoding/json"
	"math"
	"time"

	"vec-diputacion-granada/internal/vec/documentos/ports"
)

// operarIngesta usa una única fachada nominal. SQL revalida la autorización
// V3 y conserva CAS, estado y auditoría en la misma transacción SERIALIZABLE.
func (r *Repositorio) operarIngesta(ctx context.Context, accion, recursoRef, expedienteRef, planEsperado string, preimagen []byte, a ports.AutorizacionV3) (ports.PlanIngestaMayan, bool, error) {
	if r == nil || r.db == nil || ctx == nil || ctx.Err() != nil || len(preimagen) == 0 || len(preimagen) > 16384 ||
		a.ValidarParaIngestaMayan(accion, recursoRef, expedienteRef, time.Now().UTC()) != nil ||
		a.Material.ResumenCapacidad().EfectoHuellaSHA256() != ports.HuellaEfectoV3(preimagen) ||
		a.Material.PersonaVersion() > math.MaxInt64 || a.Material.PerfilVersion() > math.MaxInt64 {
		return ports.PlanIngestaMayan{}, false, ports.ErrSolicitudInvalida
	}
	auth, err := autorizacionJSON(a)
	if err != nil {
		return ports.PlanIngestaMayan{}, false, ports.ErrSolicitudInvalida
	}
	m := a.Material
	raw, err := r.transaccion(ctx,
		"SELECT vec_documentos.operar_ingesta_mayan_v1($1,$2::jsonb,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12)",
		preimagen, string(auth), m.CapacidadCanonica(), m.DecisionCanonica(), m.MotivoCanonico(), m.ContextoActorCanonico(),
		int64(m.PersonaVersion()), int64(m.PerfilVersion()), m.PayloadVECAD3(), m.SobreCOSESign1(), m.EvidenciaVerificacion(), m.RaizPublicaSPKI())
	if err != nil {
		return ports.PlanIngestaMayan{}, false, err
	}
	var salida struct {
		Plan          ports.PlanIngestaMayan `json:"plan"`
		NuevoDespacho bool                   `json:"nuevo_despacho"`
	}
	if json.Unmarshal(raw, &salida) != nil || salida.Plan.ValidarPersistido() != nil ||
		(planEsperado != "" && salida.Plan.PlanRef != planEsperado) ||
		salida.Plan.ExpedienteRef != expedienteRef {
		return ports.PlanIngestaMayan{}, false, ErrRepositorioNoDisponible
	}
	return salida.Plan, salida.NuevoDespacho, nil
}

func (r *Repositorio) operarReserva(ctx context.Context, accion, clave string, preimagen []byte, a ports.AutorizacionV3) (ports.ReservaRecursoDocumental, error) {
	if r == nil || r.db == nil || ctx == nil || ctx.Err() != nil || len(preimagen) == 0 || len(preimagen) > 16384 ||
		a.ValidarParaIngestaMayan(accion, clave, clave, time.Now().UTC()) != nil ||
		a.Material.ResumenCapacidad().EfectoHuellaSHA256() != ports.HuellaEfectoV3(preimagen) ||
		a.Material.PersonaVersion() > math.MaxInt64 || a.Material.PerfilVersion() > math.MaxInt64 {
		return ports.ReservaRecursoDocumental{}, ports.ErrSolicitudInvalida
	}
	auth, err := autorizacionJSON(a)
	if err != nil {
		return ports.ReservaRecursoDocumental{}, ports.ErrSolicitudInvalida
	}
	m := a.Material
	raw, err := r.transaccion(ctx,
		"SELECT vec_documentos.operar_reserva_recurso_mayan_v1($1,$2::jsonb,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12)",
		preimagen, string(auth), m.CapacidadCanonica(), m.DecisionCanonica(), m.MotivoCanonico(), m.ContextoActorCanonico(),
		int64(m.PersonaVersion()), int64(m.PerfilVersion()), m.PayloadVECAD3(), m.SobreCOSESign1(), m.EvidenciaVerificacion(), m.RaizPublicaSPKI())
	if err != nil {
		return ports.ReservaRecursoDocumental{}, err
	}
	var salida struct {
		Reserva ports.ReservaRecursoDocumental `json:"reserva"`
	}
	if json.Unmarshal(raw, &salida) != nil || salida.Reserva.ValidarPersistida() != nil ||
		salida.Reserva.ClaveIdempotencia != clave || salida.Reserva.PersonaRef != a.PrincipalID {
		return ports.ReservaRecursoDocumental{}, ErrRepositorioNoDisponible
	}
	return salida.Reserva, nil
}

func (r *Repositorio) ReservarRecurso(ctx context.Context, o ports.OrdenReservarRecursoMayan) (ports.ReservaRecursoDocumental, error) {
	if o.Reserva.ValidarCandidata() != nil {
		return ports.ReservaRecursoDocumental{}, ports.ErrSolicitudInvalida
	}
	preimagen, err := ports.PreimagenIngestaMayan(ports.AccionReservarRecursoMayan, o.Reserva)
	if err != nil {
		return ports.ReservaRecursoDocumental{}, err
	}
	return r.operarReserva(ctx, ports.AccionReservarRecursoMayan, o.Reserva.ClaveIdempotencia, preimagen, o.Autorizacion)
}

func (r *Repositorio) ConsultarReserva(ctx context.Context, o ports.OrdenConsultaReservaMayan) (ports.ReservaRecursoDocumental, error) {
	dato := struct {
		ClaveIdempotencia string `json:"clave_idempotencia"`
	}{o.ClaveIdempotencia}
	preimagen, err := ports.PreimagenIngestaMayan(ports.AccionConsultarReservaMayan, dato)
	if err != nil {
		return ports.ReservaRecursoDocumental{}, err
	}
	return r.operarReserva(ctx, ports.AccionConsultarReservaMayan, o.ClaveIdempotencia, preimagen, o.Autorizacion)
}

func (r *Repositorio) AbandonarRecurso(ctx context.Context, o ports.OrdenAbandonarRecursoMayan) (ports.ReservaRecursoDocumental, error) {
	dato := struct {
		ClaveIdempotencia string `json:"clave_idempotencia"`
	}{o.ClaveIdempotencia}
	preimagen, err := ports.PreimagenIngestaMayan(ports.AccionAbandonarRecursoMayan, dato)
	if err != nil {
		return ports.ReservaRecursoDocumental{}, err
	}
	return r.operarReserva(ctx, ports.AccionAbandonarRecursoMayan, o.ClaveIdempotencia, preimagen, o.Autorizacion)
}

func (r *Repositorio) PrepararIngesta(ctx context.Context, o ports.OrdenPrepararIngestaMayan) (ports.PlanIngestaMayan, error) {
	if o.Plan.ValidarBase() != nil || o.Plan.PlanRef != "" || o.Plan.Version != 0 || o.Plan.Estado != "" {
		return ports.PlanIngestaMayan{}, ports.ErrSolicitudInvalida
	}
	preimagen, err := ports.PreimagenIngestaMayan(ports.AccionPrepararIngestaMayan, o.Plan)
	if err != nil {
		return ports.PlanIngestaMayan{}, err
	}
	p, _, err := r.operarIngesta(ctx, ports.AccionPrepararIngestaMayan, o.Plan.CargaID, o.Plan.ExpedienteRef, "", preimagen, o.Autorizacion)
	return p, err
}

func (r *Repositorio) DespacharIngesta(ctx context.Context, o ports.OrdenCambiarIngestaMayan) (ports.PlanIngestaMayan, bool, error) {
	if o.Estado != ports.EstadoIngestaDespachada || o.VersionEsperada < 1 {
		return ports.PlanIngestaMayan{}, false, ports.ErrSolicitudInvalida
	}
	preimagen, err := o.Preimagen(ports.AccionDespacharIngestaMayan)
	if err != nil {
		return ports.PlanIngestaMayan{}, false, err
	}
	return r.operarIngesta(ctx, ports.AccionDespacharIngestaMayan, o.PlanRef, o.ExpedienteRef, o.PlanRef, preimagen, o.Autorizacion)
}

func (r *Repositorio) ResolverIngesta(ctx context.Context, o ports.OrdenCambiarIngestaMayan) (ports.PlanIngestaMayan, error) {
	accion := o.Autorizacion.Accion
	if accion != ports.AccionResolverIngestaMayan && accion != ports.AccionConciliarIngestaMayan {
		return ports.PlanIngestaMayan{}, ports.ErrSolicitudInvalida
	}
	if o.VersionEsperada < 2 || (o.Estado != ports.EstadoIngestaConfirmada && o.Estado != ports.EstadoIngestaIndeterminada && o.Estado != ports.EstadoIngestaRechazada) {
		return ports.PlanIngestaMayan{}, ports.ErrSolicitudInvalida
	}
	preimagen, err := o.Preimagen(accion)
	if err != nil {
		return ports.PlanIngestaMayan{}, err
	}
	p, _, err := r.operarIngesta(ctx, accion, o.PlanRef, o.ExpedienteRef, o.PlanRef, preimagen, o.Autorizacion)
	return p, err
}

func (r *Repositorio) ConsultarIngesta(ctx context.Context, o ports.OrdenConsultarIngestaMayan) (ports.PlanIngestaMayan, error) {
	preimagen, err := o.Preimagen()
	if err != nil {
		return ports.PlanIngestaMayan{}, err
	}
	p, _, err := r.operarIngesta(ctx, ports.AccionConsultarIngestaMayan, o.PlanRef, o.ExpedienteRef, o.PlanRef, preimagen, o.Autorizacion)
	return p, err
}

var _ ports.RepositorioIngestaMayan = (*Repositorio)(nil)
