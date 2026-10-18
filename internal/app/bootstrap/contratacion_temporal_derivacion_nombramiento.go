package bootstrap

import (
	"context"
	"encoding/json"
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
	soporte *soporteAltaContratacionTemporalDesarrollo
	lector  ports.LectorExpedienteSeleccionLlamamiento
}

func (p preparadorDerivacionNombramientoDesarrollo) PrepararDerivacionNombramiento(ctx context.Context,
	e ports.EntradaDerivacionNombramiento,
) (ports.SolicitudDerivacionNombramiento, error) {
	vacio := ports.SolicitudDerivacionNombramiento{}
	if ctx == nil || p.soporte == nil || dependenciaEsNulaContratacionTemporalDesarrollo(p.lector) || e.Validar() != nil {
		return vacio, ports.ErrDerivacionNombramientoNoDisponible
	}
	c, valida := p.soporte.capacidadValida(ctx)
	if !valida || c.ruta != httpinterno.RutaDerivacionNombramiento || c.metodo != http.MethodPost {
		return vacio, ports.ErrDerivacionNombramientoDenegada
	}
	padre, err := p.lector.LeerExpedienteParaSeleccion(ctx,
		organizacionAltaContratacionTemporalDesarrollo, e.PadreRef, e.VersionPadre)
	if err != nil {
		return vacio, ports.ErrDerivacionNombramientoNoDisponible
	}
	if padre.Fiscalizado.Asignacion == nil || padre.Fiscalizado.OrganizacionRef != organizacionAltaContratacionTemporalDesarrollo ||
		padre.Fiscalizado.Referencia != e.PadreRef || padre.Fiscalizado.Version != e.VersionPadre {
		return vacio, ports.ErrDerivacionNombramientoNoConfiable
	}
	canon, err := json.Marshal(padre.Fiscalizado)
	if err != nil {
		return vacio, ports.ErrDerivacionNombramientoNoDisponible
	}
	s := ports.SolicitudDerivacionNombramiento{Esquema: ports.EsquemaSolicitudDerivacionNombramiento,
		OrganizacionRef: organizacionAltaContratacionTemporalDesarrollo, PadreRef: e.PadreRef,
		VersionPadre: e.VersionPadre, HijoRef: ports.HijoRefDerivacionNombramiento(
			organizacionAltaContratacionTemporalDesarrollo, e.PadreRef, e.NumeroPlaza),
		OfertaRef: e.OfertaRef, NumeroPlaza: e.NumeroPlaza, SecuenciaPlaza: e.SecuenciaPlaza,
		ParticipacionRef: e.ParticipacionRef, ReciboAdjudicacionRef: e.ReciboAdjudicacionRef,
		ClaveIdempotencia: e.ClaveIdempotencia, UnidadRef: padre.Fiscalizado.Asignacion.UnidadRef,
		PadreCanonico: string(canon)}
	if _, err := s.Canon(); err != nil {
		return vacio, err
	}
	return s, nil
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
	fronteras catalogoFronterasComunDesarrollo,
	derivador *derivadorIdentidadOperacionDesarrollo,
	reloj relojContratacionTemporalDesarrollo,
) (http.Handler, func(), error) {
	cerrar := func() {}
	if alta == nil || alta.soporte == nil || alta.postgresql.ejecucion == nil {
		h, err := httpinterno.NuevoManejadorDerivacionNombramiento(derivacionNombramientoNoCompuesta{})
		return h, cerrar, err
	}
	lector, err := postgresct.NuevoLectorExpedienteSeleccionLlamamientoPostgreSQL(alta.postgresql.ejecucion)
	if err != nil {
		return nil, cerrar, err
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
		preparadorDerivacionNombramientoDesarrollo{soporte: alta.soporte, lector: lector},
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
