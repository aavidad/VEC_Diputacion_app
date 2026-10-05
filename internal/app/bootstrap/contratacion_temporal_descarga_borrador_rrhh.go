package bootstrap

import (
	"context"
	"errors"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/httpinterno"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/application"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
	puertosvec "vec-diputacion-granada/internal/vec/ports"
)

// registradorDescargaBorradorRRHHDesarrollo autoriza cada descarga de
// borrador de la consulta de detalle con su propia acción, la consume en
// CT177 (AD199) y deja en la auditoría común (AD169) los intentos fallidos.
// Usa el contexto de la misma petición que la consulta del detalle: no emite
// otra identidad ni cambia de perfil.
type registradorDescargaBorradorRRHHDesarrollo struct {
	autoridad *autoridadConsultasRRHHDesarrollo
	material  *proveedorMaterialAltaContratacionTemporalDesarrollo
	repo      ports.RepositorioDescargaBorradorRRHH
	motivo    dominiovec.ReferenciaEntradaCatalogo
	// auditoria es opcional: sin registrador AD169 (Bolsa sin activar) las
	// descargas siguen consumiendo su decisión, pero sus fallos no se anotan.
	auditoria *auditoriaLecturasCT
}

var _ ports.RegistradorDescargaBorradorRRHH = (*registradorDescargaBorradorRRHHDesarrollo)(nil)

func nuevoRegistradorDescargaBorradorRRHHDesarrollo(autoridad *autoridadConsultasRRHHDesarrollo,
	material *proveedorMaterialAltaContratacionTemporalDesarrollo, repo ports.RepositorioDescargaBorradorRRHH,
	motivo dominiovec.ReferenciaEntradaCatalogo, registrador puertosvec.RegistradorIntentosAuditoria, proceso string,
) (*registradorDescargaBorradorRRHHDesarrollo, error) {
	if autoridad == nil || autoridad.soporte == nil || material == nil ||
		dependenciaEsNulaContratacionTemporalDesarrollo(repo) || !dominiovec.ReferenciaMotivoAutorizacionV2Valida(motivo) {
		return nil, ports.ErrDescargaBorradorRRHHNoDisponible
	}
	r := &registradorDescargaBorradorRRHHDesarrollo{autoridad: autoridad, material: material, repo: repo, motivo: motivo}
	if !dependenciaEsNulaContratacionTemporalDesarrollo(registrador) {
		c := configuracionAuditoriaLecturasCT{Proceso: proceso, Canal: string(dominiovec.SuperficieAutenticacionInternaCorporativaV1)}
		a, err := nuevaAuditoriaLecturasCT(autoridad.soporte, registrador, c, httpinterno.RutaConsultaDetalleRRHH,
			ports.AccionDescargarBorradorRRHH, motivo)
		if err != nil {
			return nil, ports.ErrDescargaBorradorRRHHNoDisponible
		}
		// El actor del intento es el de la consulta de esta petición.
		a.resolver = autoridad.contextoConsultaRRHHDesarrollo
		r.auditoria = &a
	}
	return r, nil
}

func (r *registradorDescargaBorradorRRHHDesarrollo) alcance() ports.AlcanceDescargaBorradorRRHH {
	return ports.AlcanceDescargaBorradorRRHH{OrganizacionRef: organizacionAltaContratacionTemporalDesarrollo,
		ClaseAmbito: r.autoridad.clase, AmbitoRef: r.autoridad.ambitoRef}
}

func (r *registradorDescargaBorradorRRHHDesarrollo) RegistrarDescarga(ctx context.Context, s ports.SolicitudDescargaBorradorRRHH) (ports.ReciboDescargaBorradorRRHH, error) {
	vacio := ports.ReciboDescargaBorradorRRHH{}
	if r == nil || contextoInterfazNulo(ctx) {
		return vacio, ports.ErrDescargaBorradorRRHHNoDisponible
	}
	// El actor y la correlación se fijan antes de autorizar y sin depender de
	// la cancelación de la petición: una revocación, una desconexión o un plazo
	// agotado después no borran el intento ni cambian su identidad.
	z, ok := r.actorDePeticion(ctx)
	if !ok {
		return vacio, ports.ErrDescargaBorradorRRHHNoDisponible
	}
	correlacion, err := puertosvec.ReferenciaCorrelacionAutorizacionV2DePeticion(ctx)
	if err != nil {
		return vacio, r.anotarFallo(ctx, z, s.ExpedienteRef, ports.ErrDescargaBorradorRRHHNoDisponible)
	}
	if s.Validar() != nil {
		return vacio, r.anotarFallo(ctx, z, s.ExpedienteRef, ports.ErrDescargaBorradorRRHHInvalida)
	}
	recibo, causa := r.autorizarYRegistrar(ctx, z, correlacion, s)
	if causa == nil {
		return recibo, nil
	}
	return vacio, r.anotarFallo(ctx, z, s.ExpedienteRef, causa)
}

func (r *registradorDescargaBorradorRRHHDesarrollo) autorizarYRegistrar(ctx context.Context, z ports.ContextoAutorizacionAltaV3,
	correlacion dominiovec.ReferenciaCorrelacionAutorizacionV2, s ports.SolicitudDescargaBorradorRRHH,
) (ports.ReciboDescargaBorradorRRHH, error) {
	vacio := ports.ReciboDescargaBorradorRRHH{}
	alcance := r.alcance()
	recurso, err := ports.RecursoDescargaBorradorRRHH(s, alcance)
	if err != nil {
		return vacio, ports.ErrDescargaBorradorRRHHNoDisponible
	}
	solicitud, err := dominiovec.NuevaSolicitudAutorizacionLigadaV3(dominiovec.DatosSolicitudAutorizacionLigadaV3{
		VinculoAutenticacionActor: z.Vinculo, ReferenciaMotivo: r.motivo, Accion: ports.AccionDescargarBorradorRRHH,
		Recurso: recurso, Finalidad: ports.FinalidadDescargarBorradorRRHH, Correlacion: correlacion,
	})
	if err != nil {
		return vacio, ports.ErrDescargaBorradorRRHHNoDisponible
	}
	decision, confirmacion, err := r.autoridad.ExigirSolicitudLigadaV3(ctx, solicitud, z.Resultado)
	if err != nil {
		if errorDenegacionDescargaBorrador(err) {
			return vacio, ports.ErrAutorizacionDenegada
		}
		return vacio, ports.ErrDescargaBorradorRRHHNoDisponible
	}
	material, err := r.material.proveerMaterialConfirmacion(ctx, solicitud, decision, confirmacion, r.motivo, z.Resultado)
	if err != nil || material.ValidarEstructura() != nil {
		return vacio, ports.ErrDescargaBorradorRRHHNoDisponible
	}
	resumen := material.ResumenCapacidad()
	huella, err := recurso.HuellaContextoAutorizacionSHA256()
	if err != nil || resumen.Operacion() != ports.AccionDescargarBorradorRRHH ||
		resumen.AudienciaConsumo() != ports.AudienciaConsumoDescargaBorradorRRHHV3 ||
		resumen.EfectoRef() != s.ExpedienteRef || resumen.EfectoHuellaSHA256() != huella {
		return vacio, ports.ErrDescargaBorradorRRHHNoDisponible
	}
	return r.repo.RegistrarDescargaBorrador(ctx, alcance, s, material)
}

// RegistrarFalloDescarga anota un intento fallido anterior al registro.
// Devuelve nil si no hay nada que anotar: sin registrador AD169 configurado o
// sin actor resuelto (la consulta ni siquiera identificó a la persona).
func (r *registradorDescargaBorradorRRHHDesarrollo) RegistrarFalloDescarga(ctx context.Context, expedienteRef string, causa error) error {
	if r == nil || contextoInterfazNulo(ctx) || causa == nil || r.auditoria == nil {
		return nil
	}
	z, ok := r.actorDePeticion(ctx)
	if !ok {
		return nil
	}
	return r.anotarFallo(ctx, z, expedienteRef, causa)
}

// actorDePeticion devuelve el contexto de la consulta ya resuelto para esta
// petición. Ignora la cancelación: conserva los valores de la petición.
func (r *registradorDescargaBorradorRRHHDesarrollo) actorDePeticion(ctx context.Context) (ports.ContextoAutorizacionAltaV3, bool) {
	z, err := r.autoridad.contextoConsultaRRHHDesarrollo(context.WithoutCancel(ctx))
	if err != nil || z.Resultado.Validar() != nil || z.Vinculo.ValidarPara(z.Resultado) != nil {
		return ports.ContextoAutorizacionAltaV3{}, false
	}
	return z, true
}

func (r *registradorDescargaBorradorRRHHDesarrollo) anotarFallo(ctx context.Context, z ports.ContextoAutorizacionAltaV3,
	expedienteRef string, causa error,
) error {
	if r.auditoria == nil {
		return causa
	}
	// Si no se puede anotar, el resultado es indisponibilidad (503), nunca la
	// causa original: un 403 sin acuse dejaría una denegación sin rastro.
	_, correlacion, err := r.auditoria.capturar(context.WithoutCancel(ctx))
	if err != nil {
		return ports.ErrConsultaRRHHNoDisponible
	}
	resultado := dominiovec.ResultadoIntentoAuditoriaError
	if errorDenegacionDescargaBorrador(causa) && !errors.Is(causa, context.Canceled) && !errors.Is(causa, context.DeadlineExceeded) {
		resultado = dominiovec.ResultadoIntentoAuditoriaDenegado
	}
	return r.auditoria.registrar(ctx, z, correlacion, expedienteRef, resultado, causa)
}

// Denegación es solo la del PDP o la base (42501) y la consulta no observable,
// que es como la consulta de detalle oculta una denegación. La consulta no
// distingue entre «no autorizado» y «no existe», así que un expediente
// inexistente pedido por el cliente también queda como «denegado».
func errorDenegacionDescargaBorrador(err error) bool {
	return errors.Is(err, ports.ErrAutorizacionDenegada) || errors.Is(err, dominiovec.ErrAutorizacionDenegada) ||
		errors.Is(err, dominiovec.ErrPermissionDenied) || errors.Is(err, puertosvec.ErrDenegacionExplicitaAutorizacionLigadaV3) ||
		errors.Is(err, application.ErrConsultaRRHHNoObservable)
}

// multiplexorDescargasLectoresRRHHDesarrollo elige el registrador del lector
// de la petición con la misma capacidad mTLS que el detalle.
type multiplexorDescargasLectoresRRHHDesarrollo struct {
	*multiplexoresLectoresRRHHDesarrollo
}

func (m multiplexorDescargasLectoresRRHHDesarrollo) registrador(ctx context.Context) (ports.RegistradorDescargaBorradorRRHH, bool) {
	if m.multiplexoresLectoresRRHHDesarrollo == nil || contextoInterfazNulo(ctx) {
		return nil, false
	}
	c, ok := capacidadLectorRRHHDesarrollo(ctx, httpinterno.RutaConsultaDetalleRRHH)
	if !ok {
		return nil, false
	}
	registrador, soporte := m.descargas[c.principal.ID], m.soportes[c.principal.ID]
	if c.sello != m.sello || registrador == nil || soporte == nil || soporte.sello != m.sello || soporte.principalID != c.principal.ID ||
		soporte.certificadoSHA256 != c.principal.Attributes["certificate_sha256"] {
		return nil, false
	}
	return registrador, true
}

func (m multiplexorDescargasLectoresRRHHDesarrollo) RegistrarDescarga(ctx context.Context, s ports.SolicitudDescargaBorradorRRHH) (ports.ReciboDescargaBorradorRRHH, error) {
	r, ok := m.registrador(ctx)
	if !ok {
		return ports.ReciboDescargaBorradorRRHH{}, ports.ErrDescargaBorradorRRHHNoDisponible
	}
	return r.RegistrarDescarga(ctx, s)
}

func (m multiplexorDescargasLectoresRRHHDesarrollo) RegistrarFalloDescarga(ctx context.Context, expedienteRef string, causa error) error {
	r, ok := m.registrador(ctx)
	if !ok {
		return nil
	}
	return r.RegistrarFalloDescarga(ctx, expedienteRef, causa)
}
