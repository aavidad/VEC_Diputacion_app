package bootstrap

import (
	"context"
	"errors"
	"log/slog"
	"reflect"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/application"
	ctdomain "vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
	puertosvec "vec-diputacion-granada/internal/vec/ports"
)

const (
	rutaPreparacionVigenteCoberturaDesarrollo            = "/api/vec/contratacion-temporal/cobertura/preparacion-vigente"
	accionPreparacionVigenteCoberturaDesarrollo          = "contratacion_temporal.cobertura.catalogo_vigente.consultar"
	finalidadPreparacionVigenteCoberturaDesarrollo       = "consultar_preparacion_cobertura_rrhh"
	recursoPreparacionVigenteCoberturaDesarrollo         = "catalogo_vias_cobertura_vigente"
	tipoPreparacionVigenteCoberturaDesarrollo            = "catalogo_vias_cobertura"
	catalogoMotivosPreparacionVigenteCoberturaDesarrollo = "motivos_autorizacion_preparacion_vigente_cobertura"
)

var errAutorizacionPreparacionVigenteCoberturaDesarrolloNoDisponible = errors.New(
	"contratacion temporal: autorizacion de preparacion vigente no disponible",
)

// La consulta usa una accion y un recurso propios. Esta lista es cerrada: una
// concesion futura con campos adicionales tampoco autoriza esta proyeccion.
var camposPreparacionVigenteCoberturaDesarrollo = []string{
	"catalogo.referencia", "catalogo.version", "catalogo.huella_sha256",
	"catalogo.es_ejemplo", "catalogo.vias.clave", "catalogo.vias.orden",
	"catalogo.vias.documentos.clave", "catalogo.vias.documentos.orden",
	"catalogo.vias.documentos.clave_i18n", "catalogo.vias.datos.clave",
	"catalogo.vias.datos.orden", "catalogo.vias.datos.clave_i18n",
}

func motivoAutorizacionPreparacionVigenteCoberturaDesarrollo() dominiovec.ReferenciaEntradaCatalogo {
	return dominiovec.ReferenciaEntradaCatalogo{
		CatalogoID:           catalogoMotivosPreparacionVigenteCoberturaDesarrollo,
		CatalogoVersion:      1,
		CatalogoHuellaSHA256: huellaAltaContratacionTemporalDesarrollo("catalogo-motivos-preparacion-vigente-cobertura-v1"),
		EntradaClave:         referenciaAltaContratacionTemporalDesarrollo("motivo_", "consulta-preparacion-vigente-cobertura"),
	}
}

func concesionPreparacionVigenteCoberturaDesarrollo() dominiovec.ConcesionRol {
	return dominiovec.ConcesionRol{
		Accion:           accionPreparacionVigenteCoberturaDesarrollo,
		ModuloID:         ports.ModuloContratacion,
		TipoRecurso:      tipoPreparacionVigenteCoberturaDesarrollo,
		Finalidades:      []string{finalidadPreparacionVigenteCoberturaDesarrollo},
		GarantiaMinima:   dominiovec.AuthAssuranceHigh,
		CamposPermitidos: append([]string(nil), camposPreparacionVigenteCoberturaDesarrollo...),
		Obligaciones:     []string{"registrar_acceso"},
	}
}

// Versiona una sola vez el rol DEMO de cobertura para todas sus rutas. La
// concesion GET es quinta y separada; v1/v2 publicadas no se reescriben.
func instantaneaCoberturaConPreparacionVigenteDesarrollo(
	base dominiovec.InstantaneaAutorizacion,
) (dominiovec.InstantaneaAutorizacion, error) {
	fallo := errAutorizacionPreparacionVigenteCoberturaDesarrolloNoDisponible
	if base.Validar() != nil || base.VersionRol.RolID != "tecnico_rrhh_cobertura_desarrollo" ||
		len(base.AsignacionPerfil.Ambitos) != 2 ||
		base.AsignacionPerfil.Ambitos[0].Clave != "organizacion_ref" ||
		base.AsignacionPerfil.Ambitos[1].Clave != "unidad_ejecutora_ref" {
		return dominiovec.InstantaneaAutorizacion{}, fallo
	}
	instantanea := clonarInstantaneaAutorizacionAltaContratacionTemporalDesarrollo(base)
	accionesPrevias := []string{
		accionPropuestaCoberturaDesarrollo,
		string(ctdomain.AccionDecidirCoberturaGobernada),
		string(ctdomain.AccionRectificarCoberturaGobernada),
		string(ports.AccionConsultarResultadoCobertura),
	}
	if len(instantanea.VersionRol.Concesiones) < len(accionesPrevias) {
		return dominiovec.InstantaneaAutorizacion{}, fallo
	}
	for indice, accion := range accionesPrevias {
		if instantanea.VersionRol.Concesiones[indice].Accion != accion {
			return dominiovec.InstantaneaAutorizacion{}, fallo
		}
	}
	esperada := concesionPreparacionVigenteCoberturaDesarrollo()
	if instantanea.VersionRol.Version >= 3 && len(instantanea.VersionRol.Concesiones) == 5 {
		concesion := instantanea.VersionRol.Concesiones[4]
		if reflect.DeepEqual(concesion, esperada) {
			return instantanea, nil
		}
		return dominiovec.InstantaneaAutorizacion{}, fallo
	}
	if instantanea.VersionRol.Version != 2 || len(instantanea.VersionRol.Concesiones) != 4 {
		return dominiovec.InstantaneaAutorizacion{}, fallo
	}
	for _, concesion := range instantanea.VersionRol.Concesiones {
		if concesion.Accion == accionPreparacionVigenteCoberturaDesarrollo {
			return dominiovec.InstantaneaAutorizacion{}, fallo
		}
	}
	instantanea.VersionRol.Concesiones = append(instantanea.VersionRol.Concesiones, esperada)
	instantanea.VersionRol.Version = 3
	instantanea.AsignacionPerfil.VersionRolRef = instantanea.VersionRol.Referencia()
	instantanea.ControlVigenciaVersionRol.VersionRolRef = instantanea.VersionRol.Referencia()
	if instantanea.Validar() != nil {
		return dominiovec.InstantaneaAutorizacion{}, fallo
	}
	return instantanea, nil
}

type politicaPreparacionVigenteCoberturaDesarrollo struct {
	soporte   *soporteAltaContratacionTemporalDesarrollo
	publicada dominiovec.InstantaneaAutorizacion
	motivo    dominiovec.ReferenciaEntradaCatalogo
	validador puertosvec.ValidadorReferenciaMotivoAutorizacionV2
}

// publicada debe proceder de la composición gobernada tras su COMMIT, nunca
// de una semilla calculada durante el GET. El registro V3 revalida la
// asignación, el rol y sus controles actuales antes de confirmar cada lectura.
func nuevaPoliticaPreparacionVigenteCoberturaDesarrollo(
	soporte *soporteAltaContratacionTemporalDesarrollo,
	validador puertosvec.ValidadorReferenciaMotivoAutorizacionV2,
	publicada dominiovec.InstantaneaAutorizacion,
) (politicaAutorizacionSolicitudLigadaV3Desarrollo, error) {
	if soporte == nil || dependenciaEsNulaContratacionTemporalDesarrollo(validador) ||
		soporte.registroDecisionesAnalisis == nil || publicada.Validar() != nil {
		return politicaAutorizacionSolicitudLigadaV3Desarrollo{}, errAutorizacionPreparacionVigenteCoberturaDesarrolloNoDisponible
	}
	vinculo, err := soporte.contexto.Vinculo.Datos()
	if err != nil {
		return politicaAutorizacionSolicitudLigadaV3Desarrollo{}, errAutorizacionPreparacionVigenteCoberturaDesarrolloNoDisponible
	}
	comprobada, err := instantaneaCoberturaConPreparacionVigenteDesarrollo(publicada)
	if err != nil || comprobada.VersionRol.Referencia() != publicada.VersionRol.Referencia() ||
		comprobada.AsignacionPerfil.Referencia() != publicada.AsignacionPerfil.Referencia() {
		return politicaAutorizacionSolicitudLigadaV3Desarrollo{}, errAutorizacionPreparacionVigenteCoberturaDesarrolloNoDisponible
	}
	if publicada.AsignacionPerfil.PrincipalID != vinculo.PrincipalID ||
		publicada.AsignacionPerfil.PerfilActivoRef != vinculo.PerfilActivoRef {
		return politicaAutorizacionSolicitudLigadaV3Desarrollo{}, errAutorizacionPreparacionVigenteCoberturaDesarrolloNoDisponible
	}
	p := &politicaPreparacionVigenteCoberturaDesarrollo{
		soporte: soporte, publicada: comprobada,
		motivo:    motivoAutorizacionPreparacionVigenteCoberturaDesarrollo(),
		validador: validador,
	}
	return nuevaPoliticaAutorizacionSolicitudLigadaV3Desarrollo(p, p, p, p)
}

type claveSolicitudPreparacionVigenteCoberturaDesarrollo struct{}

func (p *politicaPreparacionVigenteCoberturaDesarrollo) solicitudValida(ctx context.Context) (dominiovec.SolicitudAutorizacionLigadaV3, string, bool) {
	if p == nil || p.soporte == nil || contextoInterfazNulo(ctx) {
		return dominiovec.SolicitudAutorizacionLigadaV3{}, "", false
	}
	capacidad, valida := p.soporte.capacidadValida(ctx)
	solicitud, existe := ctx.Value(claveSolicitudPreparacionVigenteCoberturaDesarrollo{}).(dominiovec.SolicitudAutorizacionLigadaV3)
	datos, err := solicitud.Datos()
	if !valida || capacidad.ruta != rutaPreparacionVigenteCoberturaDesarrollo || !existe || err != nil ||
		datos.Accion != accionPreparacionVigenteCoberturaDesarrollo ||
		datos.Finalidad != finalidadPreparacionVigenteCoberturaDesarrollo ||
		datos.ReferenciaMotivo != p.motivo ||
		datos.Recurso.Referencia != recursoPreparacionVigenteCoberturaDesarrollo ||
		datos.Recurso.ModuloID != ports.ModuloContratacion ||
		datos.Recurso.Tipo != tipoPreparacionVigenteCoberturaDesarrollo ||
		len(datos.Recurso.Ambitos) != 2 || len(datos.Recurso.Atributos) != 0 ||
		datos.Recurso.Ambitos["organizacion_ref"] != organizacionAltaContratacionTemporalDesarrollo ||
		datos.Recurso.Ambitos["unidad_ejecutora_ref"] != unidadCoberturaContratacionTemporalDesarrollo {
		return dominiovec.SolicitudAutorizacionLigadaV3{}, "", false
	}
	clave, valida := claveInstantaneaContratacionTemporalDesarrollo(solicitud)
	return solicitud, clave, valida
}

func (p *politicaPreparacionVigenteCoberturaDesarrollo) ObtenerInstantaneaAutorizacion(
	ctx context.Context, principalID, perfilRef string,
) (dominiovec.InstantaneaAutorizacion, error) {
	_, _, valida := p.solicitudValida(ctx)
	if !valida || principalID != p.publicada.AsignacionPerfil.PrincipalID ||
		perfilRef != p.publicada.AsignacionPerfil.PerfilActivoRef {
		return dominiovec.InstantaneaAutorizacion{}, puertosvec.ErrFuenteAutorizacionNoDisponible
	}
	return clonarInstantaneaAutorizacionAltaContratacionTemporalDesarrollo(p.publicada), nil
}

func (p *politicaPreparacionVigenteCoberturaDesarrollo) registrarDecisionVigente(
	ctx context.Context, solicitud dominiovec.SolicitudAutorizacionLigadaV3,
	errorNoDisponible error,
	registrar func(registroDecisionesAnalisisContratacionTemporalDesarrollo) (time.Time, error),
) (time.Time, error) {
	_, clave, valida := p.solicitudValida(ctx)
	claveOrden, claveOrdenValida := claveInstantaneaContratacionTemporalDesarrollo(solicitud)
	if !valida || !claveOrdenValida || claveOrden != clave {
		return time.Time{}, puertosvec.ErrInstantaneaAutorizacionObsoleta
	}
	p.soporte.mu.Lock()
	registro := p.soporte.registroDecisionesAnalisis
	p.soporte.mu.Unlock()
	if registro == nil {
		return time.Time{}, errorNoDisponible
	}
	return registrar(registro)
}

func (p *politicaPreparacionVigenteCoberturaDesarrollo) RegistrarConcesionCandidataAutorizacionLigadaV3SiInstantaneaVigente(
	ctx context.Context, orden puertosvec.OrdenRegistroConcesionCandidataAutorizacionLigadaV3,
) (time.Time, error) {
	datos, err := orden.Datos()
	if err != nil || datos.Decision.ValidarPara(datos.Solicitud) != nil {
		return time.Time{}, puertosvec.ErrRegistroConcesionAutorizacionLigadaV3NoDisponible
	}
	return p.registrarDecisionVigente(ctx, datos.Solicitud,
		puertosvec.ErrRegistroConcesionAutorizacionLigadaV3NoDisponible,
		func(r registroDecisionesAnalisisContratacionTemporalDesarrollo) (time.Time, error) {
			return r.RegistrarConcesionCandidataAutorizacionLigadaV3SiInstantaneaVigente(ctx, orden)
		})
}

func (p *politicaPreparacionVigenteCoberturaDesarrollo) RegistrarDenegacionAutorizacionLigadaV3(
	ctx context.Context, orden puertosvec.OrdenRegistroDenegacionAutorizacionLigadaV3,
) error {
	datos, err := orden.Datos()
	if err != nil || datos.Decision.ValidarPara(datos.Solicitud) != nil {
		return puertosvec.ErrRegistroDenegacionAutorizacionLigadaV3NoDisponible
	}
	_, err = p.registrarDecisionVigente(ctx, datos.Solicitud,
		puertosvec.ErrRegistroDenegacionAutorizacionLigadaV3NoDisponible,
		func(r registroDecisionesAnalisisContratacionTemporalDesarrollo) (time.Time, error) {
			return time.Time{}, r.RegistrarDenegacionAutorizacionLigadaV3(ctx, orden)
		})
	return err
}

func (p *politicaPreparacionVigenteCoberturaDesarrollo) ValidarReferenciaMotivoAutorizacionV2(
	ctx context.Context, motivo dominiovec.ReferenciaEntradaCatalogo, instante time.Time,
) error {
	if _, _, valida := p.solicitudValida(ctx); !valida || motivo != p.motivo {
		return dominiovec.ErrSolicitudAutorizacionInvalida
	}
	return p.validador.ValidarReferenciaMotivoAutorizacionV2(ctx, motivo, instante)
}

// autorizadorConsultaPreparacionVigenteDesarrollo no publica concesiones ni
// motivos. La composicion debe proporcionarle una politica V3 propia cuyo rol
// y motivo esten publicados y cuyo registrador durable haya pasado preflight.
// Sin esa politica, el servicio central deniega la consulta.
type autorizadorConsultaPreparacionVigenteDesarrollo struct {
	soporte     *soporteAltaContratacionTemporalDesarrollo
	autorizador puertosvec.AutorizadorSolicitudLigadaV3
	generador   generadorCorrelacionCoberturaDesarrollo
	motivo      dominiovec.ReferenciaEntradaCatalogo
}

func nuevoAutorizadorConsultaPreparacionVigenteDesarrollo(
	soporte *soporteAltaContratacionTemporalDesarrollo,
	autorizador puertosvec.AutorizadorSolicitudLigadaV3,
	generador generadorCorrelacionCoberturaDesarrollo,
	motivo dominiovec.ReferenciaEntradaCatalogo,
) (*autorizadorConsultaPreparacionVigenteDesarrollo, error) {
	if soporte == nil || dependenciaEsNulaContratacionTemporalDesarrollo(autorizador) ||
		dependenciaEsNulaContratacionTemporalDesarrollo(generador) ||
		!dominiovec.ReferenciaMotivoAutorizacionV2Valida(motivo) ||
		motivo != motivoAutorizacionPreparacionVigenteCoberturaDesarrollo() ||
		motivo == soporte.motivoPropuestaCobertura ||
		motivo == soporte.motivoResultadoCobertura ||
		motivo == soporte.motivoDecisionCobertura ||
		motivo == soporte.motivoRectificacionCobertura {
		return nil, errAutorizacionPreparacionVigenteCoberturaDesarrolloNoDisponible
	}
	return &autorizadorConsultaPreparacionVigenteDesarrollo{
		soporte: soporte, autorizador: autorizador, generador: generador, motivo: motivo,
	}, nil
}

func (a *autorizadorConsultaPreparacionVigenteDesarrollo) AutorizarConsultaPreparacionCoberturaVigente(
	ctx context.Context,
	solicitudContexto ports.SolicitudResolverContextoAutorizacionAltaV3,
	contexto ports.ContextoAutorizacionAltaV3,
	organizacionRef string,
	instante time.Time,
) error {
	if a == nil || a.soporte == nil || contextoInterfazNulo(ctx) ||
		contexto.ValidarPara(solicitudContexto, instante) != nil {
		return application.ErrPresentacionPropuestaCoberturaDenegada
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	capacidad, valida := a.soporte.capacidadValida(ctx)
	vinculo, err := contexto.Vinculo.Datos()
	if !valida || capacidad.ruta != rutaPreparacionVigenteCoberturaDesarrollo ||
		organizacionRef != organizacionAltaContratacionTemporalDesarrollo ||
		err != nil || vinculo.PerfilActivoRef != a.soporte.instantaneaCobertura.AsignacionPerfil.PerfilActivoRef {
		return application.ErrPresentacionPropuestaCoberturaDenegada
	}
	solicitud, err := a.nuevaSolicitud(ctx, contexto, organizacionRef)
	if err != nil {
		return application.ErrPresentacionPropuestaCoberturaNoDisponible
	}
	ctx = context.WithValue(ctx, claveSolicitudPreparacionVigenteCoberturaDesarrollo{}, solicitud)
	decision, confirmacion, err := a.autorizador.ExigirSolicitudLigadaV3(ctx, solicitud, contexto.Resultado)
	if errContexto := ctx.Err(); errContexto != nil {
		return errContexto
	}
	if errors.Is(err, puertosvec.ErrInstantaneaAutorizacionObsoleta) {
		return application.ErrPresentacionPropuestaCoberturaDenegada
	}
	if falloInfraestructuraAutorizacionCoberturaDesarrollo(err) {
		return application.ErrPresentacionPropuestaCoberturaNoDisponible
	}
	if err != nil {
		return application.ErrPresentacionPropuestaCoberturaDenegada
	}
	concedida, _, err := decision.Resultado()
	if err != nil || !concedida || decision.ValidarPara(solicitud) != nil {
		return application.ErrPresentacionPropuestaCoberturaDenegada
	}
	confirmada, err := confirmacion.Datos()
	huella, errHuella := dominiovec.HuellaSHA256DecisionAutorizacionV3(decision)
	if err != nil || errHuella != nil || confirmada.DecisionHuellaSHA256 != huella {
		return application.ErrPresentacionPropuestaCoberturaNoDisponible
	}
	if decision.ExigirProyeccionPara(solicitud, camposPreparacionVigenteCoberturaDesarrollo,
		[]string{"registrar_acceso"}) != nil {
		registrarIncidenciaPreparacionVigenteCoberturaDesarrollo()
		return application.ErrPreparacionCatalogoCoberturaNoDisponiblePerfil
	}
	restricciones, err := decision.RestriccionesProyeccionPara(solicitud)
	if err != nil || len(restricciones.Obligaciones) != 1 || restricciones.Obligaciones[0] != "registrar_acceso" {
		registrarIncidenciaPreparacionVigenteCoberturaDesarrollo()
		return application.ErrPreparacionCatalogoCoberturaNoDisponiblePerfil
	}
	return nil
}

func registrarIncidenciaPreparacionVigenteCoberturaDesarrollo() {
	slog.Warn("contratacion temporal: preparacion vigente no disponible para el perfil",
		"evento", "preparacion_vigente_cobertura_restringida",
		"accion", accionPreparacionVigenteCoberturaDesarrollo,
		"resultado", "sin_datos")
}

func (a *autorizadorConsultaPreparacionVigenteDesarrollo) nuevaSolicitud(
	ctx context.Context,
	contexto ports.ContextoAutorizacionAltaV3,
	organizacionRef string,
) (dominiovec.SolicitudAutorizacionLigadaV3, error) {
	correlacion, err := dominiovec.GenerarReferenciaCorrelacionAutorizacionV2(ctx, a.generador)
	if err != nil {
		return dominiovec.SolicitudAutorizacionLigadaV3{}, err
	}
	return dominiovec.NuevaSolicitudAutorizacionLigadaV3(dominiovec.DatosSolicitudAutorizacionLigadaV3{
		VinculoAutenticacionActor: contexto.Vinculo,
		ReferenciaMotivo:          a.motivo,
		Accion:                    accionPreparacionVigenteCoberturaDesarrollo,
		Recurso: dominiovec.RecursoAutorizable{
			Referencia: recursoPreparacionVigenteCoberturaDesarrollo,
			ModuloID:   ports.ModuloContratacion,
			Tipo:       tipoPreparacionVigenteCoberturaDesarrollo,
			Ambitos: map[string]string{
				"organizacion_ref":     organizacionRef,
				"unidad_ejecutora_ref": unidadCoberturaContratacionTemporalDesarrollo,
			},
		},
		Finalidad:   finalidadPreparacionVigenteCoberturaDesarrollo,
		Correlacion: correlacion,
	})
}
