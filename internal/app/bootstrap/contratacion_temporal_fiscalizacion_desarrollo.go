package bootstrap

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	contrataciontemporal "vec-diputacion-granada/internal/modules/contrataciontemporal"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/httpinterno"
	postgrescontratacion "vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/postgres"
	seguridadcontratacion "vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/seguridad"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/application"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	"vec-diputacion-granada/internal/shared/plazoarranque"
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
	aprobacion aprobacionProvisionPerfilesRRHHDesarrollo,
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
		aprobacion,
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
	contador, err := postgrescontratacion.NuevoContadorNumeroVisiblePostgreSQL(alta.postgresql.ejecucion)
	if err != nil {
		return vacias, errFiscalizacionContratacionTemporalDesarrolloNoDisponible
	}
	referencias := seguridadcontratacion.NuevoGeneradorReferenciasAltaCriptograficoConContador(contador)
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
	mu          sync.Mutex
	sello       *selloConsultasContratacionTemporalDesarrollo
	principalID string
	// principalOriginal conserva la identidad validada del canal, separada
	// del principal canónico V3 que no transporta roles ni atributos.
	principalOriginal  dominiovec.Principal
	certificadoSHA256  string
	contexto           ports.ContextoAutorizacionAltaV3
	instantanea        dominiovec.InstantaneaAutorizacion
	motivo             dominiovec.ReferenciaEntradaCatalogo
	reloj              relojContratacionTemporalDesarrollo
	registroDecisiones registroDecisionesAnalisisContratacionTemporalDesarrollo
	// fijo es el perfil fijo de Intervención (su propio perfil): se publica
	// una vez y después solo se consume, a través de puente, que lleva la
	// autoridad PostgreSQL de lectura y el reloj.
	fijo   *perfilFijoCTDesarrollo
	puente *soporteAltaContratacionTemporalDesarrollo
	// fase son los pares fase/estado del catálogo (c23) en que Intervención
	// fiscaliza; los mismos que cubre su perfil fijo.
	fase faseOperacionCT
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
		!origenFiscalizacionContratacionTemporalDesarrolloValido(
			s.fase, solicitud.VersionExpediente, solicitud.FaseActual, solicitud.EstadoActual,
		) ||
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

// origenFiscalizacionContratacionTemporalDesarrolloValido admite la
// fiscalización solo en los pares fase/estado del catálogo (c23) y, dentro de
// ellos, conserva la coherencia de cada origen con la versión: la primera
// parte del hito del informe (v5); una refiscalización no se autoriza por una
// comparación de versión (su fase y estado proceden de la preimagen
// persistida, que el preparador valida contra la fiscalización desfavorable y
// su subsanación); la modificación tras el nombramiento (CT120) exige una
// versión posterior y la comprueba el preparador contra la modificación.
func origenFiscalizacionContratacionTemporalDesarrolloValido(
	fase faseOperacionCT,
	version uint64,
	faseActual domain.ClaveFase,
	estado domain.EstadoOperativo,
) bool {
	if !fase.admite(faseActual, estado) {
		return false
	}
	return version == 5 && faseActual == domain.FaseInformeJuridico && estado == domain.EstadoEnCurso ||
		faseActual == domain.FaseSubsanacionUnidad && estado == domain.EstadoIncidencia ||
		version >= 7 && faseActual == domain.FaseFiscalizacion && estado == domain.EstadoEnCurso
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
	if err != nil || !solicitudAutorizacionFiscalizacionContratacionTemporalDesarrolloValida(datos, a.soporte.fase) {
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
	if err != nil || !solicitudAutorizacionFiscalizacionContratacionTemporalDesarrolloValida(datos, a.soporte.fase) {
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
	aprobacion aprobacionProvisionPerfilesRRHHDesarrollo,
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
	principal = clonarPrincipalDesarrollo(principal)
	ahora := reloj.Ahora()
	contexto, err := nuevoContextoSinteticoContratacionTemporalDesarrollo(principal, ahora)
	if err != nil {
		return nil, nil, errFiscalizacionContratacionTemporalDesarrolloNoDisponible
	}
	datosVinculo, err := contexto.Vinculo.Datos()
	if err != nil {
		return nil, nil, errFiscalizacionContratacionTemporalDesarrolloNoDisponible
	}
	fase, faseValida := alta.soporte.opcionesCatalogo.faseOperacionVigente(operacionFaseFiscalizacionCT)
	instantanea, err := nuevaInstantaneaAutorizacionFiscalizacionContratacionTemporalDesarrollo(
		datosVinculo.PrincipalID,
		datosVinculo.PerfilActivoRef,
		ahora,
		fase,
	)
	motivo := referenciaMotivoAutorizacionFiscalizacionDesarrollo()
	if err != nil || !faseValida || !dominiovec.ReferenciaMotivoAutorizacionV2Valida(motivo) {
		return nil, nil, errFiscalizacionContratacionTemporalDesarrolloNoDisponible
	}

	puente := &soporteAltaContratacionTemporalDesarrollo{
		principalID:       principal.ID,
		certificadoSHA256: principal.Attributes["certificate_sha256"],
		contexto:          contexto,
		instantanea:       instantanea,
		reloj:             reloj,
	}
	lector := &autoridadPostgreSQLContratacionTemporalDesarrollo{pool: alta.postgresql.gobierno, soporte: puente}
	puente.autoridadAsignaciones = lector
	ctx, cancelar := context.WithTimeout(context.Background(), plazoarranque.Ampliar(15*time.Second))
	defer cancelar()
	if publicarContextoPostgreSQLContratacionTemporalDesarrollo(
		ctx,
		alta.postgresql.gobierno,
		puente,
	) != nil {
		return nil, nil, errFiscalizacionContratacionTemporalDesarrolloNoDisponible
	}
	fijo, err := componerPerfilFijoIntervencionCTDesarrollo(ctx, alta.postgresql.gobierno, puente, instantanea, aprobacion)
	if err != nil {
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
		sello:              sello,
		principalID:        principal.ID,
		principalOriginal:  clonarPrincipalDesarrollo(principal),
		certificadoSHA256:  principal.Attributes["certificate_sha256"],
		contexto:           contexto,
		instantanea:        puente.instantanea,
		motivo:             motivo,
		reloj:              reloj,
		registroDecisiones: registro,
		fijo:               fijo,
		puente:             puente,
		fase:               fase,
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

// componerPerfilFijoIntervencionCTDesarrollo convierte el propio perfil de
// Intervención en perfil fijo: la plantilla (organización y pares fase/estado
// del catálogo, sin expediente) se publica una vez si falta. Si lo vigente es
// el permiso por expediente de antes, la ruta se deniega hasta que el
// operador apruebe esa huella exacta; entonces se sustituye por CAS. Una
// asignación revocada, restringida por otro acto o con otro rol nunca se
// toca.
func componerPerfilFijoIntervencionCTDesarrollo(
	ctx context.Context, pool *pgxpool.Pool, puente *soporteAltaContratacionTemporalDesarrollo,
	plantilla dominiovec.InstantaneaAutorizacion, aprobacion aprobacionProvisionPerfilesRRHHDesarrollo,
) (*perfilFijoCTDesarrollo, error) {
	if puente == nil || pool == nil || plantilla.Validar() != nil {
		return nil, errFiscalizacionContratacionTemporalDesarrolloNoDisponible
	}
	fijo := &perfilFijoCTDesarrollo{clave: operacionFaseFiscalizacionCT, contexto: puente.contexto, plantilla: plantilla,
		rutas: map[string]struct{}{httpinterno.RutaResultadosFiscalizacion: {}}, actoSesion: "acto:ct:desarrollo:sesion:v1",
		propioDelSoporte: true}
	if fijo.perfilRef() != puente.contexto.Resultado.Contexto.PerfilActivoRef {
		return nil, errFiscalizacionContratacionTemporalDesarrolloNoDisponible
	}
	rolIntervencion := plantilla.VersionRol.RolID
	admitida := func(publicada instantaneaPublicadaDesarrollo, instante time.Time) bool {
		return preimagenPropiaPerfilFijoCTDesarrollo(fijo, actoAsignacionPerfilFijoCTDesarrollo, actoAsignacionCTDesarrollo)(publicada, instante) &&
			publicada.instantanea.VersionRol.RolID == rolIntervencion
	}
	if _, err := asegurarPerfilFijoCTDesarrollo(ctx, pool, puente, fijo, aprobacion, admitida); err != nil {
		return nil, err
	}
	return fijo, nil
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

// RegistrarConcesionCandidataAutorizacionLigadaV3SiInstantaneaVigente no
// publica nada: el registro V3 comprueba bajo bloqueo que la asignación de la
// decisión sigue siendo la vigente del perfil fijo.
func (s *soporteFiscalizacionContratacionTemporalDesarrollo) RegistrarConcesionCandidataAutorizacionLigadaV3SiInstantaneaVigente(
	ctx context.Context,
	orden puertosvec.OrdenRegistroConcesionCandidataAutorizacionLigadaV3,
) (time.Time, error) {
	if !s.capacidadValida(ctx) {
		return time.Time{}, puertosvec.ErrInstantaneaAutorizacionObsoleta
	}
	datos, err := orden.Datos()
	s.mu.Lock()
	registro := s.registroDecisiones
	s.mu.Unlock()
	if err != nil || registro == nil || datos.ReferenciaMotivo != s.motivo || datos.ResultadoContexto.Validar() != nil ||
		datos.Decision.ValidarPara(datos.Solicitud) != nil {
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
	s.mu.Lock()
	registro := s.registroDecisiones
	s.mu.Unlock()
	if err != nil || registro == nil || datos.ReferenciaMotivo != s.motivo {
		return puertosvec.ErrRegistroDenegacionAutorizacionLigadaV3NoDisponible
	}
	return registro.RegistrarDenegacionAutorizacionLigadaV3(ctx, orden)
}

// instantaneaParaContexto valida la solicitud y consume la asignación
// publicada del perfil fijo de Intervención. Nunca prepara ni publica.
func (s *soporteFiscalizacionContratacionTemporalDesarrollo) instantaneaParaContexto(
	ctx context.Context,
) (dominiovec.InstantaneaAutorizacion, bool) {
	if s == nil || ctx == nil || s.fijo == nil || s.puente == nil {
		return dominiovec.InstantaneaAutorizacion{}, false
	}
	datos, existe := ctx.Value(
		claveSolicitudAutorizacionContratacionTemporalDesarrollo{},
	).(dominiovec.DatosSolicitudAutorizacionLigadaV3)
	if !existe || !solicitudAutorizacionFiscalizacionContratacionTemporalDesarrolloValida(datos, s.fase) {
		return dominiovec.InstantaneaAutorizacion{}, false
	}
	return s.puente.consumirPerfilFijoCTDesarrollo(ctx, s.fijo)
}

// nuevaInstantaneaAutorizacionFiscalizacionContratacionTemporalDesarrollo es
// la plantilla del perfil fijo de Intervención: la organización y los pares
// fase/estado del catálogo (c23), sin expediente.
func nuevaInstantaneaAutorizacionFiscalizacionContratacionTemporalDesarrollo(
	principalID string,
	perfilRef string,
	ahora time.Time,
	fase faseOperacionCT,
) (dominiovec.InstantaneaAutorizacion, error) {
	if !fase.valida() {
		return dominiovec.InstantaneaAutorizacion{}, errFiscalizacionContratacionTemporalDesarrolloNoDisponible
	}
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
		fase.ambitosPerfil(organizacionAltaContratacionTemporalDesarrollo),
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

// solicitudAutorizacionFiscalizacionContratacionTemporalDesarrolloValida: el
// expediente va en la referencia del recurso; los ámbitos son la organización
// y un par fase/estado que admite el catálogo, con los atributos coherentes
// con el origen (inicial, refiscalización o modificación).
func solicitudAutorizacionFiscalizacionContratacionTemporalDesarrolloValida(
	datos dominiovec.DatosSolicitudAutorizacionLigadaV3,
	fase faseOperacionCT,
) bool {
	ambitos := datos.Recurso.Ambitos
	if !fase.admite(domain.ClaveFase(ambitos["fase_previa"]), domain.EstadoOperativo(ambitos["estado_previo"])) {
		return false
	}
	origenInicial := ambitos["fase_previa"] == string(domain.FaseInformeJuridico) &&
		ambitos["estado_previo"] == string(domain.EstadoEnCurso)
	origenRefiscalizacion := ambitos["fase_previa"] == string(domain.FaseSubsanacionUnidad) &&
		ambitos["estado_previo"] == string(domain.EstadoIncidencia) &&
		domain.ReferenciaOpacaValida(datos.Recurso.Atributos["retorno_previo_ref"]) &&
		domain.ReferenciaOpacaValida(datos.Recurso.Atributos["subsanacion_recibo_ref"])
	origenModificacion := ambitos["fase_previa"] == string(domain.FaseFiscalizacion) &&
		ambitos["estado_previo"] == string(domain.EstadoEnCurso) &&
		domain.ReferenciaOpacaValida(datos.Recurso.Atributos["modificacion_recibo_ref"]) &&
		datos.Recurso.Atributos["retorno_previo_ref"] == "" && datos.Recurso.Atributos["subsanacion_recibo_ref"] == ""
	return datos.Accion == contrataciontemporal.PermisoRegistrarFiscalizacion &&
		datos.ReferenciaMotivo == referenciaMotivoAutorizacionFiscalizacionDesarrollo() &&
		datos.Recurso.ModuloID == ports.ModuloContratacion &&
		datos.Recurso.Tipo == tipoRecursoFiscalizacionContratacionTemporalDesarrollo &&
		datos.Recurso.Referencia != "" &&
		datos.Finalidad == finalidadFiscalizacionContratacionTemporalDesarrollo &&
		len(ambitos) == 3 &&
		ambitos["organizacion_ref"] == organizacionAltaContratacionTemporalDesarrollo &&
		((origenInicial && datos.Recurso.Atributos["retorno_previo_ref"] == "" &&
			datos.Recurso.Atributos["subsanacion_recibo_ref"] == "" && datos.Recurso.Atributos["modificacion_recibo_ref"] == "") ||
			(origenRefiscalizacion && datos.Recurso.Atributos["modificacion_recibo_ref"] == "") || origenModificacion)
}
