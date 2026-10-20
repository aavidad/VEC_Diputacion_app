package bootstrap

import (
	"context"
	"errors"
	"net/http"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/httpinterno"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	seguridadvec "vec-diputacion-granada/internal/vec/adapters/seguridad"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
	puertosvec "vec-diputacion-granada/internal/vec/ports"
)

// La lectura nominal del padre aporta el ámbito contextual para crear el hijo.
// Un replay usa la proyección V5 nominal del hijo; CT247 coteja su ámbito con
// el material inmutable de la derivación antes de publicar esa lectura.
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
			padre.Asignacion == nil || !domain.ReferenciaOpacaValida(padre.Asignacion.UnidadRef) ||
			(padre.Resumen.UnidadRef != "" && padre.Resumen.UnidadRef != padre.Asignacion.UnidadRef) ||
			(padre.Fiscalizacion.ResultadoClave != domain.FiscalizacionFavorable &&
				padre.Fiscalizacion.ResultadoClave != domain.FiscalizacionFavorableConObservaciones) {
			return vacio, ports.ErrDerivacionNombramientoNoConfiable
		}
		unidad = padre.Asignacion.UnidadRef
	} else {
		// Solo la lectura V5 nominal del hijo aporta la unidad para el replay.
		// Su asignación de instrucción no se usa como autoridad.
		hijoRef := ports.HijoRefDerivacionNombramiento(
			organizacionAltaContratacionTemporalDesarrollo, e.PadreRef, e.NumeroPlaza)
		hijo, fallo := p.leerDetalle(lectura, hijoRef)
		if fallo != nil || !replayHijoDerivacionNombramientoConfiable(hijo, hijoRef, e) {
			return vacio, ports.ErrDerivacionNombramientoNoDisponible
		}
		unidad = hijo.Resumen.UnidadRef
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
		d.Resumen.OrganizacionRef != organizacionAltaContratacionTemporalDesarrollo {
		return vacio, ports.ErrDerivacionNombramientoNoConfiable
	}
	return d, nil
}

func replayHijoDerivacionNombramientoConfiable(d ports.DetalleExpedienteRRHH,
	hijoRef string, e ports.EntradaDerivacionNombramiento,
) bool {
	q, err := ports.NuevaSolicitudDetalleRRHH(hijoRef, 0)
	if err != nil || d.Resumen.ExpedienteRef != hijoRef || d.Resumen.Version != 6 ||
		d.Resumen.OrganizacionRef != organizacionAltaContratacionTemporalDesarrollo ||
		!domain.ReferenciaOpacaValida(d.Resumen.UnidadRef) ||
		d.InstruccionCompartida == nil ||
		d.Lectura.ExpedienteRef() != hijoRef || d.Lectura.Version() != d.Resumen.Version ||
		d.Lectura.TotalPublicado() != 1 ||
		!domain.InstanteUTCCanonico(d.Lectura.RegistradaEn()) ||
		d.Lectura.RegistradaEn().Before(d.Resumen.ActualizadoEn) ||
		d.ValidarContenidoPublicablePara(q) != nil {
		return false
	}
	i := d.InstruccionCompartida
	return hijoRef == ports.HijoRefDerivacionNombramiento(organizacionAltaContratacionTemporalDesarrollo, e.PadreRef, e.NumeroPlaza) &&
		i.PadreRef == e.PadreRef && i.OfertaRef == e.OfertaRef && i.NumeroPlaza == e.NumeroPlaza &&
		domain.ReferenciaOpacaValida(i.ReciboDerivacionRef)
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
