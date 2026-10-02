package application

import (
	"context"
	"errors"
	"time"

	"vec-diputacion-granada/internal/modules/meritos/ports"
	vec "vec-diputacion-granada/internal/vec/domain"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

type ServicioConsultaPropia struct {
	autorizador ports.Autorizador
	repositorio ports.RepositorioConsultaPropia
	auditoria   ports.AuditoriaIntentos
	reloj       vecports.Reloj
}

func NuevoServicioConsultaPropia(a ports.Autorizador, r ports.RepositorioConsultaPropia, audit ports.AuditoriaIntentos, reloj vecports.Reloj) (*ServicioConsultaPropia, error) {
	if nulo(a) || nulo(r) || nulo(audit) || nulo(reloj) {
		return nil, ports.ErrConsultaNoDisponible
	}
	return &ServicioConsultaPropia{a, r, audit, reloj}, nil
}

func (s *ServicioConsultaPropia) ConsultarActual(ctx context.Context, solicitud SolicitudConsultaPropia) (out ports.ResultadoConsultaPropia, err error) {
	if s == nil || ctx == nil || nulo(s.autorizador) || nulo(s.repositorio) || nulo(s.auditoria) || nulo(s.reloj) {
		return ports.ResultadoConsultaPropia{}, ports.ErrConsultaNoDisponible
	}
	if err = ctx.Err(); err != nil {
		return ports.ResultadoConsultaPropia{}, err
	}
	// La frontera común audita contextos inválidos. No registrar aquí identidad
	// o correlación libres que no han superado su validación nominal.
	if solicitud.validarContexto() != nil {
		return ports.ResultadoConsultaPropia{}, errors.Join(vec.ErrAutorizacionDenegada, ErrSolicitud)
	}
	// Las denegaciones de emisión se registran por el puerto V3 central. La
	// auditoría de una consulta obtenida o ausente pertenece a su transacción.
	// Un error, cancelación o COMMIT incierto nunca afirma un acceso confirmado.
	o, err := ordenConsultaPropia(solicitud)
	if err != nil {
		return ports.ResultadoConsultaPropia{}, err
	}
	o.Autorizacion, err = s.autorizarConsulta(ctx, solicitud, o)
	if err != nil {
		return ports.ResultadoConsultaPropia{}, err
	}
	if ValidarOrdenConsultaPropia(o) != nil {
		return ports.ResultadoConsultaPropia{}, ports.ErrConsultaNoDisponible
	}
	// El repositorio termina su rollback antes de devolver un error. Ese
	// resultado necesita constancia por la autoridad común independiente; la
	// concesión emitida no acredita una denegación de su consumo posterior.
	defer func() {
		if err == nil {
			return
		}
		correlacion, _ := solicitud.Correlacion.ValorCanonico()
		entrada := vec.AuditEntry{ActorID: solicitud.Contexto.Contexto.PersonaRef,
			ActorProfile: solicitud.Contexto.Contexto.PerfilActivoRef,
			Action:       AccionConsultaPropia, ModuleID: "meritos", Purpose: FinalidadConsultaPropia,
			SubjectRef: o.HechoRef, CorrelationRef: correlacion, RuleRef: solicitud.Motivo.EntradaClave,
			AuthorizationRef: o.Autorizacion.Material.ResumenCapacidad().DecisionRef(),
			OccurredAt:       s.reloj.Ahora().UTC(), Result: "no_confirmado"}
		if errors.Is(err, vec.ErrAutorizacionDenegada) {
			entrada.Result = "denegada"
		}
		// Un cierre de la petición no borra el intento. El plazo del registro
		// es independiente y corto; no conserva credenciales o material V3.
		auditCtx, cancelar := context.WithTimeout(context.WithoutCancel(ctx), 2*time.Second)
		defer cancelar()
		if _, auditErr := s.auditoria.AppendAudit(auditCtx, entrada); auditErr != nil {
			out, err = ports.ResultadoConsultaPropia{}, ports.ErrConsultaNoDisponible
		}
	}()
	resultado, err := s.repositorio.ConsultarActual(ctx, o)
	if err != nil {
		return ports.ResultadoConsultaPropia{}, err
	}
	if ValidarResultadoConsultaPropia(o, resultado) != nil {
		return ports.ResultadoConsultaPropia{}, ports.ErrConsultaNoDisponible
	}
	if resultado.Codigo == "denegada" {
		return ports.ResultadoConsultaPropia{}, vec.ErrAutorizacionDenegada
	}
	return copiarResultadoConsultaPropia(resultado), nil
}

func (s *ServicioConsultaPropia) autorizarConsulta(ctx context.Context, solicitud SolicitudConsultaPropia, o ports.OrdenConsultaPropia) (ports.AutorizacionOperacion, error) {
	auth, err := vec.NuevaSolicitudAutorizacionLigadaV3(vec.DatosSolicitudAutorizacionLigadaV3{
		VinculoAutenticacionActor: solicitud.Vinculo, ReferenciaMotivo: solicitud.Motivo,
		Accion: AccionConsultaPropia, Finalidad: FinalidadConsultaPropia, Correlacion: solicitud.Correlacion,
		Recurso: vec.RecursoAutorizable{Referencia: o.HechoRef, ModuloID: "meritos", Tipo: "hecho",
			Ambitos: map[string]string{"persona_ref": o.PersonaRef, "huella_consulta_sha256": o.HuellaConsultaSHA256}},
	})
	if err != nil {
		return ports.AutorizacionOperacion{}, vec.ErrAutorizacionDenegada
	}
	d, c, exportador, err := s.autorizador.EmitirMaterialAutorizacionAtestadaV3(ctx, auth, solicitud.Contexto)
	if err != nil || nulo(exportador) {
		if errors.Is(err, vec.ErrAutorizacionDenegada) {
			return ports.AutorizacionOperacion{}, vec.ErrAutorizacionDenegada
		}
		return ports.AutorizacionOperacion{}, ports.ErrConsultaNoDisponible
	}
	a := ports.AutorizacionOperacion{Solicitud: auth, Decision: d, Confirmacion: c}
	if exigirProyeccionConsulta(a) != nil {
		return ports.AutorizacionOperacion{}, vec.ErrAutorizacionDenegada
	}
	a.Material, err = exportador.ExportarMaterialParaConsumidor()
	if err != nil || !vecports.MaterialAtestadoLigadoV3(auth, d, c, solicitud.Contexto, solicitud.Motivo, a.Material, AudienciaConsultaPropia) {
		return ports.AutorizacionOperacion{}, ports.ErrConsultaNoDisponible
	}
	ahora, resumen := s.reloj.Ahora().UTC(), a.Material.ResumenCapacidad()
	if ahora.IsZero() || ahora.Before(resumen.EmitidaEn()) || !ahora.Before(resumen.ExpiraEn()) {
		return ports.AutorizacionOperacion{}, vec.ErrAutorizacionDenegada
	}
	a.Contexto, err = solicitud.Contexto.Clonar()
	if err != nil {
		return ports.AutorizacionOperacion{}, ports.ErrConsultaNoDisponible
	}
	return a, nil
}

func copiarResultadoConsultaPropia(r ports.ResultadoConsultaPropia) ports.ResultadoConsultaPropia {
	if r.ReciboConsulta != nil {
		v := *r.ReciboConsulta
		r.ReciboConsulta = &v
	}
	if r.HechoActual != nil {
		h := *r.HechoActual
		h.Evidencias = append([]vec.ReferenciaDocumento{}, h.Evidencias...)
		if h.Horas != nil {
			v := *h.Horas
			h.Horas = &v
		}
		if h.Revision != nil {
			v := *h.Revision
			h.Revision = &v
		}
		r.HechoActual = &h
	}
	return r
}
