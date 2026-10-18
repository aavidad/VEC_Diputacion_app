package bootstrap

import (
	"context"
	"errors"
	"net/http"
	"time"

	"vec-diputacion-granada/config"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/httpinterno"
	postgresct "vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/postgres"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/application"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	seguridadvec "vec-diputacion-granada/internal/vec/adapters/seguridad"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
	puertosvec "vec-diputacion-granada/internal/vec/ports"
)

// La lectura histórica conserva la preimagen de la clave original durante un
// replay, aunque la cabeza actual del expediente padre haya avanzado.
type preparadorDerivacionNombramientoDesarrollo struct {
	soporte   *soporteAltaContratacionTemporalDesarrollo
	fronteras catalogoFronterasComunDesarrollo
	detalle   httpinterno.ConsultorDetalleRRHH
}

func (p preparadorDerivacionNombramientoDesarrollo) PrepararDerivacionNombramiento(ctx context.Context,
	e ports.EntradaDerivacionNombramiento,
) (ports.SolicitudDerivacionNombramiento, error) {
	vacio := ports.SolicitudDerivacionNombramiento{}
	if ctx == nil || p.soporte == nil || p.fronteras.identidad == nil ||
		dependenciaEsNulaContratacionTemporalDesarrollo(p.detalle) || e.Validar() != nil {
		return vacio, ports.ErrDerivacionNombramientoNoDisponible
	}
	c, valida := p.soporte.capacidadValida(ctx)
	frontera, existe := fronteraSeguridadComunDesdeContexto(ctx)
	ahora := p.soporte.reloj.Ahora()
	if !valida || !existe || !p.fronteras.mismaInstancia(frontera.catalogo) ||
		frontera.ruta != httpinterno.RutaDerivacionNombramiento || frontera.metodo != http.MethodPost ||
		frontera.descriptor.Clave != "ct-nombramiento-derivar" ||
		frontera.descriptor.ClaveCapacidad != string(domain.AccionDerivarNombramiento) ||
		!frontera.descriptor.admitePerfil(p.soporte.contexto.Resultado.Contexto.PerfilActivoRef) ||
		c.ruta != frontera.ruta || c.metodo != frontera.metodo || c.consultaRRHH == nil ||
		!domain.InstanteUTCCanonico(c.certificadoVerificadoEn) ||
		!domain.InstanteUTCCanonico(c.certificadoValidoHasta) ||
		ahora.Before(c.certificadoVerificadoEn) || !ahora.Before(c.certificadoValidoHasta) {
		return vacio, ports.ErrDerivacionNombramientoDenegada
	}
	lectura, err := subconsultaDetalleContratacionTemporalDesarrollo(ctx, p.soporte, p.fronteras)
	if err != nil {
		return vacio, ports.ErrDerivacionNombramientoDenegada
	}
	padre, err := p.leerDetalle(lectura, e.PadreRef)
	if ctx.Err() != nil {
		return vacio, ctx.Err()
	}
	if err != nil && !errors.Is(err, ports.ErrDerivacionNombramientoDenegada) &&
		!errors.Is(err, ports.ErrDerivacionNombramientoNoDisponible) {
		return vacio, err
	}
	if err == nil && padre.Resumen.Version < e.VersionPadre {
		return vacio, ports.ErrDerivacionNombramientoNoConfiable
	}
	unidad := ""
	if err == nil && padre.Resumen.Version == e.VersionPadre {
		if padre.Resumen.FaseClave != domain.FaseFiscalizacion ||
			padre.Resumen.EstadoClave != domain.EstadoEnCurso || padre.Fiscalizacion == nil ||
			(padre.Fiscalizacion.ResultadoClave != domain.FiscalizacionFavorable &&
				padre.Fiscalizacion.ResultadoClave != domain.FiscalizacionFavorableConObservaciones) {
			return vacio, ports.ErrDerivacionNombramientoNoConfiable
		}
		unidad = padre.Asignacion.UnidadRef
	} else {
		// Si el padre avanzó o no es observable, otra lectura nominal del hijo
		// recupera su unidad. El SQL posterior coteja la preimagen original.
		hijoRef := ports.HijoRefDerivacionNombramiento(
			organizacionAltaContratacionTemporalDesarrollo, e.PadreRef, e.NumeroPlaza)
		hijo, fallo := p.leerDetalle(lectura, hijoRef)
		if fallo != nil || hijo.Resumen.Version < 6 {
			return vacio, ports.ErrDerivacionNombramientoNoDisponible
		}
		unidad = hijo.Asignacion.UnidadRef
	}
	s := ports.SolicitudDerivacionNombramiento{Esquema: ports.EsquemaSolicitudDerivacionNombramiento,
		OrganizacionRef: organizacionAltaContratacionTemporalDesarrollo, PadreRef: e.PadreRef,
		VersionPadre: e.VersionPadre, HijoRef: ports.HijoRefDerivacionNombramiento(
			organizacionAltaContratacionTemporalDesarrollo, e.PadreRef, e.NumeroPlaza),
		OfertaRef: e.OfertaRef, NumeroPlaza: e.NumeroPlaza, SecuenciaPlaza: e.SecuenciaPlaza,
		ParticipacionRef: e.ParticipacionRef, ReciboAdjudicacionRef: e.ReciboAdjudicacionRef,
		ClaveIdempotencia: e.ClaveIdempotencia, UnidadRef: unidad}
	if _, err := s.Canon(); err != nil {
		return vacio, err
	}
	return s, nil
}

func (p preparadorDerivacionNombramientoDesarrollo) leerDetalle(ctx context.Context, ref string) (ports.DetalleExpedienteRRHH, error) {
	vacio := ports.DetalleExpedienteRRHH{}
	q, err := ports.NuevaSolicitudDetalleRRHH(ref, 0)
	if err != nil {
		return vacio, ports.ErrDerivacionNombramientoInvalida
	}
	d, err := p.detalle.Consultar(ctx, q)
	if ctx.Err() != nil {
		return vacio, ctx.Err()
	}
	if err != nil {
		if errors.Is(err, ports.ErrAutorizacionDenegada) || errors.Is(err, ports.ErrConsultaRRHHNoObservable) {
			return vacio, ports.ErrDerivacionNombramientoDenegada
		}
		return vacio, ports.ErrDerivacionNombramientoNoDisponible
	}
	if d.Resumen.Validar() != nil || d.Resumen.ExpedienteRef != ref ||
		d.Resumen.OrganizacionRef != organizacionAltaContratacionTemporalDesarrollo ||
		d.Asignacion == nil || !domain.ReferenciaOpacaValida(d.Asignacion.UnidadRef) ||
		(d.Resumen.UnidadRef != "" && d.Resumen.UnidadRef != d.Asignacion.UnidadRef) {
		return vacio, ports.ErrDerivacionNombramientoNoConfiable
	}
	return d, nil
}

type autorizadorDerivacionNombramientoDesarrollo struct {
	perfil *autoridadPerfilOfertaCTDesarrollo
	base   autoridadSeleccionOfertaCTDesarrollo
}

func (a autorizadorDerivacionNombramientoDesarrollo) AutorizarDerivacionNombramiento(ctx context.Context,
	s ports.SolicitudDerivacionNombramiento,
) (puertosvec.ExportacionMaterialConsumoAutorizacionAtestadaV3, error) {
	vacio := puertosvec.ExportacionMaterialConsumoAutorizacionAtestadaV3{}
	if a.perfil == nil || a.perfil.material == nil || a.perfil.pdp == nil {
		return vacio, ports.ErrDerivacionNombramientoNoDisponible
	}
	recurso, err := s.Recurso()
	if err != nil {
		return vacio, err
	}
	hijo, actor, err := a.base.contextoPerfilOferta(ctx, a.perfil, http.MethodPost, httpinterno.RutaDerivacionNombramiento)
	if err != nil {
		return vacio, clasificarAutoridadDerivacionNombramiento(err)
	}
	correlacion, err := dominiovec.GenerarReferenciaCorrelacionAutorizacionV2(hijo, seguridadvec.GeneradorReferenciasCriptograficas{})
	if err != nil {
		return vacio, ports.ErrDerivacionNombramientoNoDisponible
	}
	solicitud, err := dominiovec.NuevaSolicitudAutorizacionLigadaV3(dominiovec.DatosSolicitudAutorizacionLigadaV3{
		VinculoAutenticacionActor: actor.Vinculo, ReferenciaMotivo: a.perfil.motivo,
		Accion: string(domain.AccionDerivarNombramiento), Recurso: recurso,
		Finalidad: "gestionar_contratacion_temporal", Correlacion: correlacion,
	})
	if err != nil {
		return vacio, ports.ErrDerivacionNombramientoDenegada
	}
	decision, confirmacion, err := a.perfil.pdp.ExigirSolicitudLigadaV3(hijo, solicitud, actor.Resultado)
	if err != nil {
		return vacio, clasificarAutoridadDerivacionNombramiento(err)
	}
	material, err := a.perfil.material.proveerMaterialConfirmacion(hijo, solicitud, decision, confirmacion,
		a.perfil.motivo, actor.Resultado)
	if err != nil || material.ValidarEstructura() != nil ||
		material.ResumenCapacidad().AudienciaConsumo() != ports.AudienciaDerivacionNombramiento {
		return vacio, ports.ErrDerivacionNombramientoNoDisponible
	}
	return material, nil
}

func clasificarAutoridadDerivacionNombramiento(err error) error {
	if errors.Is(err, ports.ErrIntegracionBolsaNoDisponible) ||
		errors.Is(err, puertosvec.ErrFuenteAutorizacionNoDisponible) ||
		errors.Is(err, puertosvec.ErrRegistroConcesionAutorizacionLigadaV3NoDisponible) {
		return ports.ErrDerivacionNombramientoNoDisponible
	}
	return ports.ErrDerivacionNombramientoDenegada
}

func nuevaRutaDerivacionNombramientoDesarrollo(cfg config.Config,
	alta *dependenciasAltaContratacionTemporalDesarrollo,
	base *preparadorBorradorLlamamientoDesarrollo,
	detalle httpinterno.ConsultorDetalleRRHH,
	fronteras catalogoFronterasComunDesarrollo,
	derivador *derivadorIdentidadOperacionDesarrollo,
	reloj relojContratacionTemporalDesarrollo,
) (http.Handler, func(), error) {
	cerrar := func() {}
	if alta == nil || alta.soporte == nil || alta.postgresql.ejecucion == nil {
		h, err := httpinterno.NuevoManejadorDerivacionNombramiento(derivacionNombramientoNoCompuesta{})
		return h, cerrar, err
	}
	registro, err := postgresct.NuevoRegistroDerivacionNombramientoPostgreSQL(alta.postgresql.ejecucion)
	if err != nil {
		return nil, cerrar, err
	}
	autoridad := autorizadorDerivacionNombramientoDesarrollo{}
	if base != nil && derivador != nil {
		ctx, cancelar := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancelar()
		perfil, detener, disponible := nuevaAutoridadDerivacionNombramientoDesarrollo(ctx, cfg, alta,
			base, fronteras, derivador, reloj)
		if disponible {
			autoridad = autorizadorDerivacionNombramientoDesarrollo{perfil: perfil,
				base: autoridadSeleccionOfertaCTDesarrollo{alta: alta, publico: fronteras,
					vinculo: perfil, reloj: reloj}}
			cerrar = detener
		}
	}
	servicio, err := application.NuevoServicioDerivacionNombramiento(
		preparadorDerivacionNombramientoDesarrollo{soporte: alta.soporte, fronteras: fronteras, detalle: detalle},
		autoridad, registro)
	if err != nil {
		cerrar()
		return nil, func() {}, err
	}
	h, err := httpinterno.NuevoManejadorDerivacionNombramiento(servicio)
	if err != nil {
		cerrar()
		return nil, func() {}, err
	}
	return h, cerrar, nil
}

type derivacionNombramientoNoCompuesta struct{}

func (derivacionNombramientoNoCompuesta) Derivar(context.Context,
	ports.EntradaDerivacionNombramiento) (ports.ResultadoDerivacionNombramiento, error) {
	return ports.ResultadoDerivacionNombramiento{}, ports.ErrDerivacionNombramientoNoDisponible
}
