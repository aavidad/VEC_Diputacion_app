package bootstrap

import (
	"context"
	"errors"
	"sync"
	"time"

	contrataciontemporal "vec-diputacion-granada/internal/modules/contrataciontemporal"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/httpinterno"
	postgrescontratacion "vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/postgres"
	seguridadcontratacion "vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/seguridad"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/application"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	seguridadvec "vec-diputacion-granada/internal/vec/adapters/seguridad"
	aplicacionvec "vec-diputacion-granada/internal/vec/application"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
	puertosvec "vec-diputacion-granada/internal/vec/ports"
)

const (
	tipoRecursoFiscalizacionContratacionTemporalDesarrollo = "fiscalizacion_contratacion_temporal"
	finalidadFiscalizacionContratacionTemporalDesarrollo   = "gestionar_contratacion_temporal"
	definicionFiscalizacionContratacionTemporalDesarrollo  = "configuracion:ct:desarrollo:fiscalizacion:v1"
	unidadFiscalizadoraContratacionTemporalDesarrollo      = "unidad:desarrollo:intervencion"
)

var errFiscalizacionContratacionTemporalDesarrolloNoDisponible = errors.New(
	"contratacion temporal: fiscalizacion de desarrollo no disponible",
)

type dependenciasFiscalizacionContratacionTemporalDesarrollo struct {
	soporte  *soporteFiscalizacionContratacionTemporalDesarrollo
	servicio *application.ServicioFiscalizaciones
}

func nuevasDependenciasFiscalizacionContratacionTemporalDesarrollo(
	identidad *resolvedorIdentidadDesarrollo,
	derivador *derivadorIdentidadOperacionDesarrollo,
	alta *dependenciasAltaContratacionTemporalDesarrollo,
	sello *selloConsultasContratacionTemporalDesarrollo,
	reloj relojContratacionTemporalDesarrollo,
) (dependenciasFiscalizacionContratacionTemporalDesarrollo, error) {
	vacias := dependenciasFiscalizacionContratacionTemporalDesarrollo{}
	if derivador == nil || !derivador.valido() || alta == nil ||
		alta.postgresql.ejecucion == nil || alta.postgresql.proveedorMaterial == nil {
		return vacias, errFiscalizacionContratacionTemporalDesarrolloNoDisponible
	}
	soporte, autorizador, err := nuevoSoporteFiscalizacionContratacionTemporalDesarrollo(
		identidad,
		alta,
		sello,
		reloj,
	)
	if err != nil {
		return vacias, errFiscalizacionContratacionTemporalDesarrolloNoDisponible
	}
	ambitoActivo, ambitosRetenidos, err := configuracionesHMACAltaContratacionTemporalDesarrollo(
		derivador,
		ports.DominioAmbitoIdempotenciaFiscalizacion,
		true,
	)
	if err != nil {
		return vacias, errFiscalizacionContratacionTemporalDesarrolloNoDisponible
	}
	huellaActiva, huellasRetenidas, err := configuracionesHMACAltaContratacionTemporalDesarrollo(
		derivador,
		ports.DominioHuellaPeticionFiscalizacion,
		false,
	)
	if err != nil {
		return vacias, errFiscalizacionContratacionTemporalDesarrolloNoDisponible
	}
	sellos, err := seguridadcontratacion.NuevaAutoridadSellosFiscalizacionHMAC(
		ambitoActivo,
		ambitosRetenidos,
		huellaActiva,
		huellasRetenidas,
	)
	if err != nil {
		return vacias, errFiscalizacionContratacionTemporalDesarrolloNoDisponible
	}
	referencias := seguridadcontratacion.NuevoGeneradorReferenciasAltaCriptografico()
	preparaciones, err := postgrescontratacion.NuevoPreparadorFiscalizacionPostgreSQL(
		alta.postgresql.ejecucion,
		referencias,
	)
	if err != nil {
		return vacias, errFiscalizacionContratacionTemporalDesarrolloNoDisponible
	}
	transaccion, err := postgrescontratacion.NuevaTransaccionFiscalizacionesPostgreSQL(
		alta.postgresql.ejecucion,
		alta.postgresql.proveedorMaterial,
	)
	if err != nil {
		return vacias, errFiscalizacionContratacionTemporalDesarrolloNoDisponible
	}
	servicio, err := application.NuevoServicioFiscalizaciones(
		soporte,
		sellos,
		sellos,
		preparaciones,
		seguridadvec.GeneradorReferenciasCriptograficas{},
		autorizador,
		reloj,
		transaccion,
	)
	if err != nil {
		return vacias, errFiscalizacionContratacionTemporalDesarrolloNoDisponible
	}
	return dependenciasFiscalizacionContratacionTemporalDesarrollo{
		soporte:  soporte,
		servicio: servicio,
	}, nil
}

// soporteFiscalizacionContratacionTemporalDesarrollo mantiene separada la
// autoridad sintetica de Intervencion. No delega en soporteAlta porque este
// ultimo debe seguir rechazando cualquier principal que no sea tecnico_rrhh.
type soporteFiscalizacionContratacionTemporalDesarrollo struct {
	mu                    sync.Mutex
	sello                 *selloConsultasContratacionTemporalDesarrollo
	principalID           string
	certificadoSHA256     string
	contexto              ports.ContextoAutorizacionAltaV3
	instantanea           dominiovec.InstantaneaAutorizacion
	motivo                dominiovec.ReferenciaEntradaCatalogo
	reloj                 relojContratacionTemporalDesarrollo
	autoridadAsignaciones autoridadAsignacionesContratacionTemporalDesarrollo
	registroDecisiones    registroDecisionesAnalisisContratacionTemporalDesarrollo
	instantaneas          map[string]dominiovec.InstantaneaAutorizacion
}

var _ httpinterno.AutoridadContextoCanalFiscalizacion = (*soporteFiscalizacionContratacionTemporalDesarrollo)(nil)
var _ ports.ResolutorPoliticaFiscalizacion = (*soporteFiscalizacionContratacionTemporalDesarrollo)(nil)

func (s *soporteFiscalizacionContratacionTemporalDesarrollo) ResolverPoliticaFiscalizacion(
	ctx context.Context,
	solicitud ports.SolicitudResolverPoliticaFiscalizacion,
) (ports.PoliticaFiscalizacion, error) {
	if contextoInterfazNulo(ctx) || s == nil || solicitud.Validar() != nil {
		return ports.PoliticaFiscalizacion{},
			errFiscalizacionContratacionTemporalDesarrolloNoDisponible
	}
	s.mu.Lock()
	contexto := s.contexto
	motivo := s.motivo
	s.mu.Unlock()
	vinculo, err := contexto.Vinculo.Datos()
	if err != nil || solicitud.OrganizacionRef != organizacionAltaContratacionTemporalDesarrollo ||
		solicitud.VersionExpediente != 5 ||
		solicitud.FaseActual != domain.FaseInformeJuridico ||
		solicitud.EstadoActual != domain.EstadoEnCurso ||
		solicitud.UnidadAsignadaRef != unidadCoberturaContratacionTemporalDesarrollo ||
		solicitud.ResponsableAsignadoRef != responsableAsignacionContratacionTemporalDesarrollo ||
		solicitud.ActorRef != vinculo.PrincipalID ||
		solicitud.PerfilRef != vinculo.PerfilActivoRef {
		return ports.PoliticaFiscalizacion{},
			errFiscalizacionContratacionTemporalDesarrolloNoDisponible
	}
	if err := ctx.Err(); err != nil {
		return ports.PoliticaFiscalizacion{}, errors.Join(
			errFiscalizacionContratacionTemporalDesarrolloNoDisponible,
			err,
		)
	}
	politica := ports.PoliticaFiscalizacion{
		DefinicionRef:          definicionFiscalizacionContratacionTemporalDesarrollo,
		DefinicionVersion:      1,
		DefinicionHuellaSHA256: huellaAltaContratacionTemporalDesarrollo("politica-fiscalizacion"),
		Accion:                 domain.AccionRegistrarFiscalizacion,
		Finalidad:              domain.ClaveCatalogo(ports.FinalidadRegistrarFiscalizacion),
		UnidadFiscalizadoraRef: unidadFiscalizadoraContratacionTemporalDesarrollo,
		MotivoAutorizacion:     motivo,
		EvaluadaEn:             solicitud.Instante,
		ValidaHasta:            solicitud.Instante.Add(5 * time.Minute),
	}
	if politica.ValidarPara(solicitud, solicitud.Instante) != nil {
		return ports.PoliticaFiscalizacion{},
			errFiscalizacionContratacionTemporalDesarrolloNoDisponible
	}
	return politica, nil
}

type autorizadorFiscalizacionContratacionTemporalDesarrollo struct {
	delegado autorizadorLigadoContratacionTemporalDesarrollo
	soporte  *soporteFiscalizacionContratacionTemporalDesarrollo
}

func (a *autorizadorFiscalizacionContratacionTemporalDesarrollo) ExigirSolicitudLigadaV3(
	ctx context.Context,
	solicitud dominiovec.SolicitudAutorizacionLigadaV3,
	resultado dominiovec.ResultadoContextoActorRegistradoV2,
) (
	dominiovec.DecisionAutorizacionLigadaV3,
	puertosvec.ConfirmacionRegistroConcesionAutorizacionLigadaV3,
	error,
) {
	if a == nil || a.soporte == nil || dependenciaEsNulaContratacionTemporalDesarrollo(a.delegado) ||
		ctx == nil {
		return dominiovec.DecisionAutorizacionLigadaV3{},
			puertosvec.ConfirmacionRegistroConcesionAutorizacionLigadaV3{},
			errFiscalizacionContratacionTemporalDesarrolloNoDisponible
	}
	datos, err := solicitud.Datos()
	if err != nil || !solicitudAutorizacionFiscalizacionContratacionTemporalDesarrolloValida(datos) {
		return dominiovec.DecisionAutorizacionLigadaV3{},
			puertosvec.ConfirmacionRegistroConcesionAutorizacionLigadaV3{},
			errFiscalizacionContratacionTemporalDesarrolloNoDisponible
	}
	ctx = context.WithValue(
		ctx,
		claveSolicitudAutorizacionContratacionTemporalDesarrollo{},
		datos,
	)
	return a.delegado.ExigirSolicitudLigadaV3(ctx, solicitud, resultado)
}

func (a *autorizadorFiscalizacionContratacionTemporalDesarrollo) PrepararRegistroCompuestoSolicitudLigadaV3(
	ctx context.Context,
	solicitud dominiovec.SolicitudAutorizacionLigadaV3,
	resultado dominiovec.ResultadoContextoActorRegistradoV2,
	generador puertosvec.GeneradorReferenciaDecisionAutorizacion,
) (
	dominiovec.DecisionAutorizacionLigadaV3,
	puertosvec.CandidataRegistroDecisionAutorizacionLigadaV3,
	error,
) {
	if a == nil || a.soporte == nil || dependenciaEsNulaContratacionTemporalDesarrollo(a.delegado) ||
		ctx == nil {
		return dominiovec.DecisionAutorizacionLigadaV3{},
			puertosvec.CandidataRegistroDecisionAutorizacionLigadaV3{},
			errFiscalizacionContratacionTemporalDesarrolloNoDisponible
	}
	datos, err := solicitud.Datos()
	if err != nil || !solicitudAutorizacionFiscalizacionContratacionTemporalDesarrolloValida(datos) {
		return dominiovec.DecisionAutorizacionLigadaV3{},
			puertosvec.CandidataRegistroDecisionAutorizacionLigadaV3{},
			errFiscalizacionContratacionTemporalDesarrolloNoDisponible
	}
	ctx = context.WithValue(
		ctx,
		claveSolicitudAutorizacionContratacionTemporalDesarrollo{},
		datos,
	)
	return a.delegado.PrepararRegistroCompuestoSolicitudLigadaV3(
		ctx,
		solicitud,
		resultado,
		generador,
	)
}

func nuevoSoporteFiscalizacionContratacionTemporalDesarrollo(
	identidad *resolvedorIdentidadDesarrollo,
	alta *dependenciasAltaContratacionTemporalDesarrollo,
	sello *selloConsultasContratacionTemporalDesarrollo,
	reloj relojContratacionTemporalDesarrollo,
) (
	*soporteFiscalizacionContratacionTemporalDesarrollo,
	autorizadorLigadoContratacionTemporalDesarrollo,
	error,
) {
	if identidad == nil || alta == nil || alta.soporte == nil ||
		alta.postgresql.gobierno == nil || sello == nil {
		return nil, nil, errFiscalizacionContratacionTemporalDesarrolloNoDisponible
	}
	principal, valida := identidad.principalConRolUnico(
		rolIntervencionContratacionTemporalDesarrollo,
	)
	if !valida || !principalIntervencionContratacionTemporalDesarrolloValido(principal) {
		return nil, nil, errFiscalizacionContratacionTemporalDesarrolloNoDisponible
	}
	ahora := reloj.Ahora()
	contexto, err := nuevoContextoSinteticoContratacionTemporalDesarrollo(principal, ahora)
	if err != nil {
		return nil, nil, errFiscalizacionContratacionTemporalDesarrolloNoDisponible
	}
	datosVinculo, err := contexto.Vinculo.Datos()
	if err != nil {
		return nil, nil, errFiscalizacionContratacionTemporalDesarrolloNoDisponible
	}
	instantanea, err := nuevaInstantaneaAutorizacionFiscalizacionContratacionTemporalDesarrollo(
		datosVinculo.PrincipalID,
		datosVinculo.PerfilActivoRef,
		ahora,
	)
	motivo := referenciaMotivoAutorizacionFiscalizacionDesarrollo()
	if err != nil || !dominiovec.ReferenciaMotivoAutorizacionV2Valida(motivo) {
		return nil, nil, errFiscalizacionContratacionTemporalDesarrolloNoDisponible
	}

	puente := &soporteAltaContratacionTemporalDesarrollo{
		principalID:       principal.ID,
		certificadoSHA256: principal.Attributes["certificate_sha256"],
		contexto:          contexto,
		instantanea:       instantanea,
		reloj:             reloj,
	}
	ctx, cancelar := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancelar()
	if publicarContextoPostgreSQLContratacionTemporalDesarrollo(
		ctx,
		alta.postgresql.gobierno,
		puente,
	) != nil || publicarAutorizacionPostgreSQLContratacionTemporalDesarrollo(
		ctx,
		alta.postgresql.gobierno,
		puente,
	) != nil {
		return nil, nil, errFiscalizacionContratacionTemporalDesarrolloNoDisponible
	}
	desde, _, vigente := ventanaAutoridadSinteticaContratacionTemporalDesarrollo(ahora)
	if !vigente || publicarCatalogoMotivosPostgreSQLContratacionTemporalDesarrollo(
		ctx,
		alta.postgresql.gobierno,
		[]dominiovec.ReferenciaEntradaCatalogo{motivo},
		desde,
	) != nil {
		return nil, nil, errFiscalizacionContratacionTemporalDesarrolloNoDisponible
	}

	alta.soporte.mu.Lock()
	registro := alta.soporte.registroDecisionesAnalisis
	alta.soporte.mu.Unlock()
	if registro == nil {
		return nil, nil, errFiscalizacionContratacionTemporalDesarrolloNoDisponible
	}
	soporte := &soporteFiscalizacionContratacionTemporalDesarrollo{
		sello:                 sello,
		principalID:           principal.ID,
		certificadoSHA256:     principal.Attributes["certificate_sha256"],
		contexto:              contexto,
		instantanea:           puente.instantanea,
		motivo:                motivo,
		reloj:                 reloj,
		autoridadAsignaciones: &autoridadPostgreSQLContratacionTemporalDesarrollo{pool: alta.postgresql.gobierno, soporte: puente},
		registroDecisiones:    registro,
		instantaneas:          make(map[string]dominiovec.InstantaneaAutorizacion),
	}
	autorizadorBase, err := aplicacionvec.NuevoServicioAutorizacionSolicitudLigadaV3(
		soporte,
		soporte,
		soporte,
		soporte,
		reloj,
		seguridadvec.GeneradorReferenciasCriptograficas{},
		aplicacionvec.ConfiguracionServicioAutorizacion{VigenciaDecision: 90 * time.Second},
	)
	if err != nil {
		return nil, nil, errFiscalizacionContratacionTemporalDesarrolloNoDisponible
	}
	autorizador := &autorizadorFiscalizacionContratacionTemporalDesarrollo{
		delegado: autorizadorBase,
		soporte:  soporte,
	}
	return soporte, autorizador, nil
}

func (s *soporteFiscalizacionContratacionTemporalDesarrollo) capacidadValida(
	ctx context.Context,
) bool {
	if s == nil || contextoInterfazNulo(ctx) || ctx.Err() != nil || s.sello == nil ||
		s.principalID == "" || s.certificadoSHA256 == "" {
		return false
	}
	capacidad, existe := ctx.Value(
		claveCapacidadConsultasContratacionTemporalDesarrollo{},
	).(capacidadConsultaContratacionTemporalDesarrollo)
	return existe && capacidad.sello == s.sello &&
		capacidad.ruta == httpinterno.RutaResultadosFiscalizacion &&
		principalIntervencionContratacionTemporalDesarrolloValido(capacidad.principal) &&
		capacidad.principal.ID == s.principalID &&
		capacidad.principal.Attributes["certificate_sha256"] == s.certificadoSHA256
}

func (s *soporteFiscalizacionContratacionTemporalDesarrollo) ResolverContextoCanalFiscalizacion(
	ctx context.Context,
) (httpinterno.ContextoCanalFiscalizacion, error) {
	if !s.capacidadValida(ctx) {
		return httpinterno.ContextoCanalFiscalizacion{}, ports.ErrAutorizacionDenegada
	}
	vinculo, err := s.contexto.Vinculo.Datos()
	if err != nil {
		return httpinterno.ContextoCanalFiscalizacion{}, ports.ErrAutorizacionDenegada
	}
	return httpinterno.ContextoCanalFiscalizacion{
		AutenticacionRef: vinculo.AutenticacionRef,
		SesionRef:        vinculo.SesionRef,
		PerfilRef:        vinculo.PerfilActivoRef,
		OrganizacionRef:  organizacionAltaContratacionTemporalDesarrollo,
	}, nil
}

func (s *soporteFiscalizacionContratacionTemporalDesarrollo) ResolverContextoAutorizacionAltaV3(
	ctx context.Context,
	solicitud ports.SolicitudResolverContextoAutorizacionAltaV3,
) (ports.ContextoAutorizacionAltaV3, error) {
	if !s.capacidadValida(ctx) || solicitud.Validar() != nil {
		return ports.ContextoAutorizacionAltaV3{}, ports.ErrAutorizacionDenegada
	}
	vinculo, err := s.contexto.Vinculo.Datos()
	if err != nil || solicitud.AutenticacionRef != vinculo.AutenticacionRef ||
		solicitud.SesionRef != vinculo.SesionRef || solicitud.PerfilRef != vinculo.PerfilActivoRef {
		return ports.ContextoAutorizacionAltaV3{}, ports.ErrAutorizacionDenegada
	}
	resultado, err := s.contexto.Resultado.Clonar()
	if err != nil {
		return ports.ContextoAutorizacionAltaV3{}, ports.ErrAutorizacionDenegada
	}
	return ports.ContextoAutorizacionAltaV3{
		Vinculo:   s.contexto.Vinculo,
		Resultado: resultado,
	}, nil
}

func (s *soporteFiscalizacionContratacionTemporalDesarrollo) ObtenerInstantaneaAutorizacion(
	ctx context.Context,
	principalID string,
	perfilRef string,
) (dominiovec.InstantaneaAutorizacion, error) {
	if !s.capacidadValida(ctx) {
		return dominiovec.InstantaneaAutorizacion{}, puertosvec.ErrFuenteAutorizacionNoDisponible
	}
	instantanea, valida := s.instantaneaParaContexto(ctx)
	if !valida || principalID != instantanea.AsignacionPerfil.PrincipalID ||
		perfilRef != instantanea.AsignacionPerfil.PerfilActivoRef {
		return dominiovec.InstantaneaAutorizacion{}, puertosvec.ErrFuenteAutorizacionNoDisponible
	}
	return clonarInstantaneaAutorizacionAltaContratacionTemporalDesarrollo(instantanea), nil
}

func (s *soporteFiscalizacionContratacionTemporalDesarrollo) ValidarReferenciaMotivoAutorizacionV2(
	ctx context.Context,
	referencia dominiovec.ReferenciaEntradaCatalogo,
	instante time.Time,
) error {
	if !s.capacidadValida(ctx) || referencia != s.motivo ||
		!domain.InstanteUTCCanonico(instante) {
		return dominiovec.ErrSolicitudAutorizacionInvalida
	}
	return nil
}

func (s *soporteFiscalizacionContratacionTemporalDesarrollo) RegistrarConcesionCandidataAutorizacionLigadaV3SiInstantaneaVigente(
	ctx context.Context,
	orden puertosvec.OrdenRegistroConcesionCandidataAutorizacionLigadaV3,
) (time.Time, error) {
	if !s.capacidadValida(ctx) {
		return time.Time{}, puertosvec.ErrInstantaneaAutorizacionObsoleta
	}
	datos, err := orden.Datos()
	clave, claveValida := claveInstantaneaContratacionTemporalDesarrollo(datos.Solicitud)
	s.mu.Lock()
	instantanea, existe := s.instantaneas[clave]
	autoridad := s.autoridadAsignaciones
	registro := s.registroDecisiones
	s.mu.Unlock()
	if err != nil || !claveValida || !existe || autoridad == nil || registro == nil ||
		datos.ReferenciaMotivo != s.motivo || datos.ResultadoContexto.Validar() != nil ||
		datos.Decision.ValidarPara(datos.Solicitud) != nil || instantanea.Validar() != nil ||
		autoridad.PublicarInstantanea(ctx, instantanea) != nil {
		return time.Time{}, puertosvec.ErrInstantaneaAutorizacionObsoleta
	}
	return registro.RegistrarConcesionCandidataAutorizacionLigadaV3SiInstantaneaVigente(ctx, orden)
}

func (s *soporteFiscalizacionContratacionTemporalDesarrollo) RegistrarDenegacionAutorizacionLigadaV3(
	ctx context.Context,
	orden puertosvec.OrdenRegistroDenegacionAutorizacionLigadaV3,
) error {
	if !s.capacidadValida(ctx) {
		return puertosvec.ErrRegistroDenegacionAutorizacionLigadaV3NoDisponible
	}
	datos, err := orden.Datos()
	clave, claveValida := claveInstantaneaContratacionTemporalDesarrollo(datos.Solicitud)
	s.mu.Lock()
	instantanea, existe := s.instantaneas[clave]
	autoridad := s.autoridadAsignaciones
	registro := s.registroDecisiones
	s.mu.Unlock()
	if err != nil || !claveValida || !existe || autoridad == nil || registro == nil ||
		datos.ReferenciaMotivo != s.motivo || instantanea.Validar() != nil ||
		autoridad.PublicarInstantanea(ctx, instantanea) != nil {
		return puertosvec.ErrRegistroDenegacionAutorizacionLigadaV3NoDisponible
	}
	return registro.RegistrarDenegacionAutorizacionLigadaV3(ctx, orden)
}

func (s *soporteFiscalizacionContratacionTemporalDesarrollo) instantaneaParaContexto(
	ctx context.Context,
) (dominiovec.InstantaneaAutorizacion, bool) {
	if s == nil || ctx == nil {
		return dominiovec.InstantaneaAutorizacion{}, false
	}
	datos, existe := ctx.Value(
		claveSolicitudAutorizacionContratacionTemporalDesarrollo{},
	).(dominiovec.DatosSolicitudAutorizacionLigadaV3)
	if !existe || !solicitudAutorizacionFiscalizacionContratacionTemporalDesarrolloValida(datos) {
		return dominiovec.InstantaneaAutorizacion{}, false
	}
	clave, valida := claveInstantaneaContratacionTemporalDesarrolloDesdeDatos(datos)
	if !valida {
		return dominiovec.InstantaneaAutorizacion{}, false
	}
	s.mu.Lock()
	preparada, yaExiste := s.instantaneas[clave]
	autoridad := s.autoridadAsignaciones
	s.mu.Unlock()
	if yaExiste {
		return clonarInstantaneaAutorizacionAltaContratacionTemporalDesarrollo(preparada),
			preparada.Validar() == nil
	}
	if autoridad == nil {
		return dominiovec.InstantaneaAutorizacion{}, false
	}
	preparada = clonarInstantaneaAutorizacionAltaContratacionTemporalDesarrollo(s.instantanea)
	preparada.AsignacionPerfil.Ambitos = []dominiovec.AmbitoPerfil{
		{Clave: "organizacion_ref", Valores: []string{datos.Recurso.Ambitos["organizacion_ref"]}},
		{Clave: "expediente_ref", Valores: []string{datos.Recurso.Ambitos["expediente_ref"]}},
		{Clave: "fase_previa", Valores: []string{datos.Recurso.Ambitos["fase_previa"]}},
		{Clave: "estado_previo", Valores: []string{datos.Recurso.Ambitos["estado_previo"]}},
	}
	preparada, err := autoridad.PrepararInstantanea(ctx, preparada)
	if err != nil || preparada.Validar() != nil {
		return dominiovec.InstantaneaAutorizacion{}, false
	}
	s.mu.Lock()
	if existente, encontrada := s.instantaneas[clave]; encontrada {
		preparada = existente
	} else {
		s.instantaneas[clave] = preparada
	}
	s.mu.Unlock()
	return clonarInstantaneaAutorizacionAltaContratacionTemporalDesarrollo(preparada),
		preparada.Validar() == nil
}

func nuevaInstantaneaAutorizacionFiscalizacionContratacionTemporalDesarrollo(
	principalID string,
	perfilRef string,
	ahora time.Time,
) (dominiovec.InstantaneaAutorizacion, error) {
	return nuevaInstantaneaAutorizacionContratacionTemporalDesarrollo(
		principalID,
		perfilRef,
		ahora,
		"intervencion_fiscalizacion_desarrollo",
		"Intervencion de fiscalizacion de desarrollo",
		"fiscalizacion-desarrollo-no-autoritativa",
		[]dominiovec.ConcesionRol{{
			Accion:         contrataciontemporal.PermisoRegistrarFiscalizacion,
			ModuloID:       ports.ModuloContratacion,
			TipoRecurso:    tipoRecursoFiscalizacionContratacionTemporalDesarrollo,
			Finalidades:    []string{finalidadFiscalizacionContratacionTemporalDesarrollo},
			GarantiaMinima: dominiovec.AuthAssuranceHigh,
		}},
		[]dominiovec.AmbitoPerfil{
			{Clave: "organizacion_ref", Valores: []string{organizacionAltaContratacionTemporalDesarrollo}},
			{Clave: "expediente_ref", Valores: []string{expedienteContratacionTemporalDesarrolloRef}},
			{Clave: "fase_previa", Valores: []string{string(domain.FaseInformeJuridico)}},
			{Clave: "estado_previo", Valores: []string{string(domain.EstadoEnCurso)}},
		},
	)
}

func referenciaMotivoAutorizacionFiscalizacionDesarrollo() dominiovec.ReferenciaEntradaCatalogo {
	return dominiovec.ReferenciaEntradaCatalogo{
		CatalogoID:           "motivos_autorizacion_fiscalizacion",
		CatalogoVersion:      1,
		CatalogoHuellaSHA256: huellaAltaContratacionTemporalDesarrollo("catalogo-motivos-fiscalizacion"),
		EntradaClave: referenciaAltaContratacionTemporalDesarrollo(
			"motivo_",
			"registrar-fiscalizacion",
		),
	}
}

func solicitudAutorizacionFiscalizacionContratacionTemporalDesarrolloValida(
	datos dominiovec.DatosSolicitudAutorizacionLigadaV3,
) bool {
	ambitos := datos.Recurso.Ambitos
	return datos.Accion == contrataciontemporal.PermisoRegistrarFiscalizacion &&
		datos.ReferenciaMotivo == referenciaMotivoAutorizacionFiscalizacionDesarrollo() &&
		datos.Recurso.ModuloID == ports.ModuloContratacion &&
		datos.Recurso.Tipo == tipoRecursoFiscalizacionContratacionTemporalDesarrollo &&
		datos.Recurso.Referencia == ambitos["expediente_ref"] &&
		datos.Finalidad == finalidadFiscalizacionContratacionTemporalDesarrollo &&
		len(ambitos) == 4 &&
		ambitos["organizacion_ref"] == organizacionAltaContratacionTemporalDesarrollo &&
		ambitos["fase_previa"] == string(domain.FaseInformeJuridico) &&
		ambitos["estado_previo"] == string(domain.EstadoEnCurso)
}
