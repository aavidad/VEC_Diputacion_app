package bootstrap

import (
	"context"
	"errors"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/httpinterno"
	postgresct "vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/postgres"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	core "vec-diputacion-granada/internal/vec/domain"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

// Proceso y canal proceden de la configuración privada del registrador común.
// La acción, la finalidad y el motivo conservan la autoridad de cada consulta.
type configuracionAuditoriaLecturasCT struct {
	Proceso, Canal string
}

type falloLecturaAuditadoCT struct {
	causa error
	acuse vecports.AcuseIntentoAuditoria
}

func (e falloLecturaAuditadoCT) Error() string                                        { return "ct_lectura_fallida_auditada" }
func (e falloLecturaAuditadoCT) Unwrap() error                                        { return e.causa }
func (e falloLecturaAuditadoCT) AcuseLecturaAuditada() vecports.AcuseIntentoAuditoria { return e.acuse }

var _ ports.FalloLecturaAuditado = falloLecturaAuditadoCT{}

type auditoriaLecturasCT struct {
	resolver      func(context.Context) (ports.ContextoAutorizacionAltaV3, error)
	registrador   vecports.RegistradorIntentosAuditoria
	configuracion configuracionAuditoriaLecturasCT
	accion        string
	motivo        core.ReferenciaEntradaCatalogo
}

func nuevaAuditoriaLecturasCT(soporte *soporteAltaContratacionTemporalDesarrollo,
	registrador vecports.RegistradorIntentosAuditoria, c configuracionAuditoriaLecturasCT,
	ruta, accion string, motivo core.ReferenciaEntradaCatalogo,
) (auditoriaLecturasCT, error) {
	if soporte == nil || dependenciaEsNulaContratacionTemporalDesarrollo(registrador) ||
		!procesoAuditoriaIntentosConfigurado(c.Proceso) || c.Canal != string(core.SuperficieAutenticacionInternaCorporativaV1) ||
		!core.ReferenciaMotivoAutorizacionV2Valida(motivo) {
		return auditoriaLecturasCT{}, ports.ErrConsultaRRHHNoDisponible
	}
	return auditoriaLecturasCT{
		resolver: func(ctx context.Context) (ports.ContextoAutorizacionAltaV3, error) {
			capacidad, valida := soporte.capacidadValida(ctx)
			if !valida || capacidad.ruta != ruta {
				return ports.ContextoAutorizacionAltaV3{}, ports.ErrAutorizacionDenegada
			}
			// El holder de la frontera mTLS conserva el mismo contexto registrado
			// que consumirá el proveedor del lector, incluido su perfil activo.
			return soporte.contextoOperativoDesarrollo(ctx)
		},
		registrador: registrador, configuracion: c, accion: accion, motivo: motivo,
	}, nil
}

func (a auditoriaLecturasCT) capturar(ctx context.Context) (ports.ContextoAutorizacionAltaV3, string, error) {
	vacio := ports.ContextoAutorizacionAltaV3{}
	if ctx == nil || a.resolver == nil || dependenciaEsNulaContratacionTemporalDesarrollo(a.registrador) {
		return vacio, "", ports.ErrConsultaRRHHNoDisponible
	}
	correlacion, err := vecports.ReferenciaCorrelacionAutorizacionV2DePeticion(ctx)
	if err != nil {
		return vacio, "", ports.ErrConsultaRRHHNoDisponible
	}
	ref, err := correlacion.ValorCanonico()
	if err != nil {
		return vacio, "", ports.ErrConsultaRRHHNoDisponible
	}
	z, err := a.resolver(ctx)
	if err != nil || z.Resultado.Validar() != nil || z.Vinculo.ValidarPara(z.Resultado) != nil {
		return vacio, "", ports.ErrConsultaRRHHNoDisponible
	}
	v, err := z.Vinculo.Datos()
	if err != nil || string(v.Superficie) != a.configuracion.Canal {
		return vacio, "", ports.ErrConsultaRRHHNoDisponible
	}
	// No resolver otra vez después del retorno: una revocación o cancelación
	// posterior no cambia la identidad histórica del intento que acaba de ocurrir.
	z.Resultado, err = z.Resultado.Clonar()
	if err != nil {
		return vacio, "", ports.ErrConsultaRRHHNoDisponible
	}
	return z, ref, nil
}

func (a auditoriaLecturasCT) registrar(ctx context.Context, z ports.ContextoAutorizacionAltaV3,
	correlacion, recurso string, resultado core.ResultadoIntentoAuditoria, original error,
) error {
	ref, err := vecports.NuevaReferenciaIntentoAuditoria()
	if err != nil {
		return ports.ErrConsultaRRHHNoDisponible
	}
	orden, err := vecports.NuevaOrdenIntentoAuditoria(ref, z.Resultado, z.Vinculo, core.DatosIntentoAuditoria{
		Accion: a.accion, ModuloID: "contratacion_temporal", RecursoRef: recurso,
		FinalidadRef: "gestionar_contratacion_temporal", Resultado: resultado, Motivo: a.motivo,
		Proceso: a.configuracion.Proceso, Canal: a.configuracion.Canal, CorrelacionRef: correlacion,
	})
	if err != nil {
		return ports.ErrConsultaRRHHNoDisponible
	}
	// El lector ya retornó y ejecutó su cierre diferido. El registrador usa
	// su propia transacción y plazo; la desconexión HTTP no cancela este hecho.
	ctx = context.WithoutCancel(ctx)
	acuse, err := a.registrador.AppendIntentoAuditoria(ctx, orden)
	if errors.Is(err, vecports.ErrIntentoAuditoriaNoDisponible) {
		// Un COMMIT incierto puede haber escrito. Recuperar con la misma orden
		// y clave conserva un único registro; no afirmamos rollback del lector.
		acuse, err = a.registrador.AppendIntentoAuditoria(ctx, orden)
	}
	if err != nil || acuse.ValidarPara(orden) != nil {
		return ports.ErrConsultaRRHHNoDisponible
	}
	return falloLecturaAuditadoCT{causa: original, acuse: acuse}
}

type lectorReciboRespuestaAuditadoCT struct {
	lector    ports.LectorReciboRespuesta
	auditoria auditoriaLecturasCT
}

func nuevoLectorReciboRespuestaAuditadoCT(lector ports.LectorReciboRespuesta,
	soporte *soporteAltaContratacionTemporalDesarrollo, registrador vecports.RegistradorIntentosAuditoria,
	c configuracionAuditoriaLecturasCT,
) (ports.LectorReciboRespuesta, error) {
	if dependenciaEsNulaContratacionTemporalDesarrollo(lector) {
		return nil, ports.ErrConsultaReciboRespuestaFallo
	}
	a, err := nuevaAuditoriaLecturasCT(soporte, registrador, c, httpinterno.RutaConsultaReciboRespuesta,
		postgresct.AccionConsultaReciboRespuesta, motivoConsultaReciboRespuestaDesarrollo())
	if err != nil {
		return nil, ports.ErrConsultaReciboRespuestaFallo
	}
	return &lectorReciboRespuestaAuditadoCT{lector: lector, auditoria: a}, nil
}

func (l *lectorReciboRespuestaAuditadoCT) ConsultarReciboRespuesta(ctx context.Context, s ports.SolicitudConsultaReciboRespuesta) (ports.ReciboRespuestaConsultado, error) {
	vacio := ports.ReciboRespuestaConsultado{}
	if l == nil || dependenciaEsNulaContratacionTemporalDesarrollo(l.lector) || s.Validar() != nil {
		return vacio, ports.ErrConsultaReciboRespuestaFallo
	}
	z, correlacion, err := l.auditoria.capturar(ctx)
	if err != nil {
		return vacio, ports.ErrConsultaReciboRespuestaFallo
	}
	r, err := l.lector.ConsultarReciboRespuesta(ctx, s)
	if ctx.Err() != nil {
		err = ctx.Err()
	} else if (err != nil && r != vacio) || (err == nil && r.ValidarPara(s) != nil) {
		err = ports.ErrReciboRespuestaNoConfiable
	}
	if err == nil {
		return r, nil // La lectura permitida ya confirmó consumo y auditoría SQL.
	}
	if err == ports.ErrReciboRespuestaNoEncontrado {
		return vacio, err // Ausencia autorizada con auditoría SQL confirmada.
	}
	resultado := core.ResultadoIntentoAuditoriaError
	if errors.Is(err, ports.ErrConsultaReciboRespuestaDenegada) &&
		!errors.Is(err, context.Canceled) && !errors.Is(err, context.DeadlineExceeded) &&
		!errors.Is(err, ports.ErrConsultaReciboRespuestaFallo) && !errors.Is(err, ports.ErrReciboRespuestaNoConfiable) {
		resultado = core.ResultadoIntentoAuditoriaDenegado
	}
	err = l.auditoria.registrar(ctx, z, correlacion, s.ComunicacionRef, resultado, err)
	if errors.Is(err, ports.ErrConsultaRRHHNoDisponible) {
		err = ports.ErrConsultaReciboRespuestaFallo
	}
	return vacio, err
}

type lectorComunicacionesAuditadoCT struct {
	lector    ports.LectorComunicacionesExpediente
	auditoria auditoriaLecturasCT
}

func nuevoLectorComunicacionesAuditadoCT(lector ports.LectorComunicacionesExpediente,
	soporte *soporteAltaContratacionTemporalDesarrollo, registrador vecports.RegistradorIntentosAuditoria,
	c configuracionAuditoriaLecturasCT,
) (ports.LectorComunicacionesExpediente, error) {
	if dependenciaEsNulaContratacionTemporalDesarrollo(lector) {
		return nil, ports.ErrConsultaComunicacionesExpedienteNoDisponible
	}
	a, err := nuevaAuditoriaLecturasCT(soporte, registrador, c, httpinterno.RutaConsultaComunicacionesExpediente,
		postgresct.AccionConsultaComunicacionesExpediente, motivoConsultaComunicacionesExpedienteDesarrollo())
	if err != nil {
		return nil, ports.ErrConsultaComunicacionesExpedienteNoDisponible
	}
	return &lectorComunicacionesAuditadoCT{lector: lector, auditoria: a}, nil
}

func (l *lectorComunicacionesAuditadoCT) ConsultarComunicacionesExpediente(ctx context.Context, c ports.ConsultaComunicacionesExpediente) (ports.PaginaComunicacionesExpediente, error) {
	vacia := ports.PaginaComunicacionesExpediente{}
	if l == nil || dependenciaEsNulaContratacionTemporalDesarrollo(l.lector) || c.Validar() != nil {
		return vacia, ports.ErrConsultaComunicacionesExpedienteNoDisponible
	}
	z, correlacion, err := l.auditoria.capturar(ctx)
	if err != nil {
		return vacia, ports.ErrConsultaComunicacionesExpedienteNoDisponible
	}
	p, err := l.lector.ConsultarComunicacionesExpediente(ctx, c)
	if ctx.Err() != nil {
		err = ctx.Err()
	} else if (err != nil && (p.ExpedienteRef != "" || p.Comunicaciones != nil || p.SiguienteCursor != "")) ||
		(err == nil && p.ValidarPara(c) != nil) {
		err = ports.ErrResultadoComunicacionesExpedienteNoConfiable
	}
	if err == nil {
		return p, nil
	}
	if err == ports.ErrConsultaComunicacionesExpedienteNoEncontrado {
		return vacia, err
	}
	resultado := core.ResultadoIntentoAuditoriaError
	if errors.Is(err, ports.ErrConsultaComunicacionesExpedienteDenegada) &&
		!errors.Is(err, context.Canceled) && !errors.Is(err, context.DeadlineExceeded) &&
		!errors.Is(err, ports.ErrConsultaComunicacionesExpedienteNoDisponible) && !errors.Is(err, ports.ErrResultadoComunicacionesExpedienteNoConfiable) {
		resultado = core.ResultadoIntentoAuditoriaDenegado
	}
	err = l.auditoria.registrar(ctx, z, correlacion, c.ExpedienteRef, resultado, err)
	if errors.Is(err, ports.ErrConsultaRRHHNoDisponible) {
		err = ports.ErrConsultaComunicacionesExpedienteNoDisponible
	}
	return vacia, err
}

var _ ports.LectorReciboRespuesta = (*lectorReciboRespuestaAuditadoCT)(nil)
var _ ports.LectorComunicacionesExpediente = (*lectorComunicacionesAuditadoCT)(nil)
