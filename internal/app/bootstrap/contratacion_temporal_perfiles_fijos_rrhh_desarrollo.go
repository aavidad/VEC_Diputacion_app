package bootstrap

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"vec-diputacion-granada/config"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/httpinterno"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
)

// Perfiles fijos de RRHH (corte 2; el análisis, corte 3).
//
// Las rutas de RRHH cuyo permiso no depende del expediente (alta directa,
// entrega de petición, cobertura, cambios de organización, análisis, asignación e informe
// jurídico) ya no comparten el perfil dinámico ni
// publican su permiso en cada petición. Cada forma de ámbito tiene un perfil
// propio de la misma persona (misma cuenta y persona; perfil, vínculo y sesión
// distintos) que la composición elige por la ruta y su método sellado, nunca el cliente. Su
// asignación se publica una sola vez, al arrancar y solo si falta; después se
// consume tal cual. Si lo publicado no es exactamente la plantilla del perfil
// (revocado, restringido por otro acto, rol retirado o plantilla cambiada) la
// ruta se deniega hasta una provisión aprobada por el operador (huella de la
// asignación vigente, CAS bajo bloqueo).
const (
	clavePerfilFijoAltaCTDesarrollo          = "alta"
	clavePerfilFijoCoberturaCTDesarrollo     = "cobertura"
	clavePerfilFijoOrganizacionCTDesarrollo  = "organizacion"
	clavePerfilFijoAnalisisCTDesarrollo      = "analisis"
	clavePerfilFijoAsignacionCTDesarrollo    = "asignacion"
	clavePerfilFijoInformeCTDesarrollo       = "informe_juridico"
	clavePerfilFijoSubsanacionCTDesarrollo   = "subsanacion"
	clavePerfilFijoEntregaCTDesarrollo       = "entrega_peticion"
	clavePerfilFijoLectorEntregaCTDesarrollo = "lector_entrega_peticion"
	clavePerfilFijoFirmaCTDesarrollo         = "firma_documento"
	// Acto con el que este circuito publica las asignaciones de los perfiles
	// fijos. Distinto del del perfil dinámico: una provisión solo reconoce como
	// propia una asignación puesta por él.
	actoAsignacionPerfilFijoCTDesarrollo = "acto:ct:perfil-fijo:asignacion:v1"
	actoSesionPerfilFijoCTDesarrollo     = "acto:ct:perfil-fijo:sesion:v1"
)

var errPerfilFijoCTNoConsumible = errors.New("perfil fijo de RRHH sin asignación publicada consumible")

// perfilFijoCTDesarrollo es un perfil de la persona de RRHH reservado a unas
// rutas concretas. Sus campos mutables se protegen con el mutex del soporte.
type perfilFijoCTDesarrollo struct {
	clave string
	rutas map[string]struct{}
	// metodo se usa solo cuando GET y POST comparten ruta con autoridades
	// distintas. Vacío equivale a las rutas históricas de método único.
	metodo    string
	contexto  ports.ContextoAutorizacionAltaV3
	plantilla dominiovec.InstantaneaAutorizacion
	// actoSesion es el de las filas de sesión del perfil (propio en los
	// perfiles nuevos, el histórico en un perfil que ya existía).
	actoSesion                 string
	actoControlRol             string
	contextoEsperadoRegistrado dominiovec.ResultadoContextoActorRegistradoV2
	sesionOperativa            proveedorSesionOperativaCTDesarrollo
	avisoNoConsumibleEn        time.Time
	// propioDelSoporte: el perfil es el del propio soporte (un lector de
	// consulta); conserva su contexto y su sesión, y solo cambia a consumir.
	propioDelSoporte bool
}

func (p *perfilFijoCTDesarrollo) atiende(ruta string) bool {
	if p == nil {
		return false
	}
	_, existe := p.rutas[ruta]
	return existe
}

func (p *perfilFijoCTDesarrollo) atiendeMetodo(ruta, metodo string) bool {
	return p.atiende(ruta) && (p.metodo == "" || p.metodo == metodo)
}

func (p *perfilFijoCTDesarrollo) perfilRef() string {
	if p == nil {
		return ""
	}
	return p.plantilla.AsignacionPerfil.PerfilActivoRef
}

func discriminadorPerfilFijoCTDesarrollo(clave string) discriminadorContextoSinteticoDesarrollo {
	prefijo := "perfil-fijo-" + clave + "-"
	return discriminadorContextoSinteticoDesarrollo{
		perfil: prefijo + "perfil", vinculo: prefijo + "vinculo",
		// La cuenta, la persona y la procedencia son las del perfil dinámico.
		procedencia: "procedencia", registro: prefijo + "registro-contexto",
		autenticacion: prefijo + "autenticacion", asercion: prefijo + "asercion",
		sesion: prefijo + "sesion", controlSesion: prefijo + "control-sesion",
		politicaGarantia: prefijo + "politica-garantia",
	}
}

// nuevoPerfilFijoCTDesarrollo compone el perfil de la clave dada para la
// persona del soporte. La plantilla se construye con su propio perfil.
func nuevoPerfilFijoCTDesarrollo(
	principal dominiovec.Principal, base ports.ContextoAutorizacionAltaV3, ahora time.Time, clave string, rutas []string,
	plantilla func(principalID, perfilRef string) (dominiovec.InstantaneaAutorizacion, error),
) (*perfilFijoCTDesarrollo, error) {
	if clave == "" || len(rutas) == 0 || plantilla == nil || base.Resultado.Validar() != nil {
		return nil, errAltaContratacionTemporalDesarrolloNoDisponible
	}
	contexto, err := nuevoContextoSinteticoContratacionTemporalDesarrolloConDiscriminador(
		principal, ahora, discriminadorPerfilFijoCTDesarrollo(clave))
	if err != nil {
		return nil, err
	}
	v, err := contexto.Vinculo.Datos()
	b := base.Resultado.Contexto
	if err != nil || v.PerfilActivoRef == b.PerfilActivoRef ||
		contexto.Resultado.Contexto.Instantanea.CuentaRef != b.Instantanea.CuentaRef ||
		contexto.Resultado.Contexto.PersonaRef != b.PersonaRef {
		return nil, errAltaContratacionTemporalDesarrolloNoDisponible
	}
	instantanea, err := plantilla(v.PrincipalID, v.PerfilActivoRef)
	if err != nil || instantanea.Validar() != nil || instantanea.AsignacionPerfil.PerfilActivoRef != v.PerfilActivoRef {
		return nil, errAltaContratacionTemporalDesarrolloNoDisponible
	}
	p := &perfilFijoCTDesarrollo{clave: clave, rutas: make(map[string]struct{}, len(rutas)), contexto: contexto,
		plantilla: instantanea, actoSesion: actoSesionPerfilFijoCTDesarrollo}
	for _, ruta := range rutas {
		p.rutas[ruta] = struct{}{}
	}
	return p, nil
}

// perfilFijoParaRuta devuelve el perfil fijo que la composición asigna a la
// ruta, o nil si la ruta sigue en el perfil dinámico.
func (s *soporteAltaContratacionTemporalDesarrollo) perfilFijoParaRuta(ruta string) *perfilFijoCTDesarrollo {
	if s == nil {
		return nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.perfilFijoParaRutaBloqueado(ruta)
}

// La ruta de entrega sirve también GET. Solo el método POST, sellado por la
// frontera mTLS, usa el perfil de entrega y del alta anidada. Una llamada sin
// capacidad válida nunca puede elegir ese perfil a partir de un valor libre.
func (s *soporteAltaContratacionTemporalDesarrollo) perfilFijoParaContexto(
	ctx context.Context, ruta string,
) *perfilFijoCTDesarrollo {
	if ruta == rutaEntregaPeticionCentro {
		capacidad, valida := s.capacidadValida(ctx)
		if !valida || capacidad.ruta != ruta ||
			(capacidad.metodo != http.MethodGet && capacidad.metodo != http.MethodPost) {
			return nil
		}
		return s.perfilFijoParaRutaYMetodo(ruta, capacidad.metodo)
	}
	return s.perfilFijoParaRuta(ruta)
}

func (s *soporteAltaContratacionTemporalDesarrollo) perfilFijoParaRutaYMetodo(ruta, metodo string) *perfilFijoCTDesarrollo {
	if s == nil {
		return nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.perfilFijoParaRutaYMetodoBloqueado(ruta, metodo)
}

func (s *soporteAltaContratacionTemporalDesarrollo) perfilFijoParaRutaBloqueado(ruta string) *perfilFijoCTDesarrollo {
	for _, p := range s.perfilesFijos {
		if p.atiende(ruta) {
			return p
		}
	}
	return nil
}

func (s *soporteAltaContratacionTemporalDesarrollo) perfilFijoParaRutaYMetodoBloqueado(ruta, metodo string) *perfilFijoCTDesarrollo {
	for _, p := range s.perfilesFijos {
		if p.atiendeMetodo(ruta, metodo) {
			return p
		}
	}
	return nil
}

// registrarPerfilFijoCTDesarrollo añade un perfil durante la composición,
// antes de servir peticiones. Rechaza rutas o perfiles repetidos.
func (s *soporteAltaContratacionTemporalDesarrollo) registrarPerfilFijoCTDesarrollo(p *perfilFijoCTDesarrollo) error {
	if s == nil || p == nil {
		return errAltaContratacionTemporalDesarrolloNoDisponible
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, otro := range s.perfilesFijos {
		if otro.clave == p.clave || otro.perfilRef() == p.perfilRef() {
			return errAltaContratacionTemporalDesarrolloNoDisponible
		}
		for ruta := range p.rutas {
			if otro.atiende(ruta) && (otro.metodo == "" || p.metodo == "" || otro.metodo == p.metodo) {
				return errAltaContratacionTemporalDesarrolloNoDisponible
			}
		}
	}
	s.perfilesFijos = append(s.perfilesFijos, p)
	return nil
}

func (s *soporteAltaContratacionTemporalDesarrollo) perfilesFijosRegistrados() []*perfilFijoCTDesarrollo {
	if s == nil {
		return nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]*perfilFijoCTDesarrollo(nil), s.perfilesFijos...)
}

// autoridadPerfilFijoCTDesarrollo es la autoridad común con los actos del
// perfil fijo. La versión de rol es compartida y conserva el acto de control
// con el que la composición publica siempre los roles de Contratación.
func autoridadPerfilFijoCTDesarrollo(pool *pgxpool.Pool, p *perfilFijoCTDesarrollo) autoridadPostgreSQLDesarrollo {
	actoControl := actoControlRolCTDesarrollo
	if p.actoControlRol != "" {
		actoControl = p.actoControlRol
	}
	return autoridadPostgreSQLDesarrollo{
		pool: pool, vinculo: p.contexto.Vinculo, prefijoBloqueo: "vec:ct:desarrollo:autorizacion:",
		actoControlRol: actoControl, actoAsignacion: actoAsignacionPerfilFijoCTDesarrollo,
		actoSesion: p.actoSesion, exigirOrigenOperativo: true,
	}
}

// publicarContextoPerfilFijoCTDesarrollo registra el contexto del perfil con
// una operación propia (incluye el perfil para no colisionar con el general).
func publicarContextoPerfilFijoCTDesarrollo(ctx context.Context, pool *pgxpool.Pool, p *perfilFijoCTDesarrollo) error {
	if p == nil {
		return falloPostgreSQLCTDesarrollo(nil)
	}
	v, err := p.contexto.Vinculo.Datos()
	if err != nil {
		return falloPostgreSQLCTDesarrollo(err)
	}
	operacion := referenciaAltaContratacionTemporalDesarrollo("oca_",
		v.PrincipalID+"\x00"+v.PerfilActivoRef+"\x00registro-contexto-perfil-fijo")
	return publicarResultadoContextoPostgreSQLDesarrollo(ctx, pool, p.contexto.Resultado, operacion)
}

// aprobacionProvisionPerfilesRRHHDesarrollo liga la aprobación del operador a
// las asignaciones vigentes exactas (sus huellas) que autoriza sustituir.
type aprobacionProvisionPerfilesRRHHDesarrollo struct {
	referencia  string
	preimagenes map[string]bool
}

func (a aprobacionProvisionPerfilesRRHHDesarrollo) valida() bool {
	return a.referencia != "" && len(a.preimagenes) != 0
}

type estadoPerfilFijoCTDesarrollo string

const (
	perfilFijoPublicadoInicial   estadoPerfilFijoCTDesarrollo = "publicado_inicial"
	perfilFijoVigente            estadoPerfilFijoCTDesarrollo = "vigente"
	perfilFijoProvisionado       estadoPerfilFijoCTDesarrollo = "provisionado"
	perfilFijoPendienteProvision estadoPerfilFijoCTDesarrollo = "pendiente_provision"
)

// asegurarPerfilFijoCTDesarrollo es el paso de arranque de un perfil fijo.
// Sin asignación publica la plantilla (inicial, transaccional). Con la
// plantilla exacta y operativa no escribe. En cualquier otro caso no escribe,
// salvo que el operador haya aprobado esa asignación exacta (su huella) y
// admitida diga que es una asignación de este mismo circuito, activa, en
// vigor y con su rol habilitado: entonces publica la plantilla contra esa
// preimagen (CAS bajo bloqueo). Una revocación, una restricción de otro acto
// o un rol retirado nunca se tocan: la ruta queda denegada.
func asegurarPerfilFijoCTDesarrollo(
	ctx context.Context, pool *pgxpool.Pool, s *soporteAltaContratacionTemporalDesarrollo, p *perfilFijoCTDesarrollo,
	aprobacion aprobacionProvisionPerfilesRRHHDesarrollo,
	admitida func(instantaneaPublicadaDesarrollo, time.Time) bool,
) (estadoPerfilFijoCTDesarrollo, error) {
	if ctx == nil || pool == nil || s == nil || p == nil || admitida == nil ||
		dependenciaEsNulaContratacionTemporalDesarrollo(s.reloj) || p.plantilla.Validar() != nil {
		return "", falloPostgreSQLCTDesarrollo(nil)
	}
	plantilla := clonarInstantaneaAutorizacionAltaContratacionTemporalDesarrollo(p.plantilla)
	comun := autoridadPerfilFijoCTDesarrollo(pool, p)
	publicada, encontrada, err := leerInstantaneaPublicadaPostgreSQLDesarrollo(ctx, pool, p.perfilRef())
	if !encontrada {
		if err != nil {
			return "", err
		}
		comun.soloInicial = true
		preparada, err := comun.prepararInstantanea(ctx, plantilla, true)
		if err != nil || preparada.AsignacionPerfil.Version != 1 {
			return "", falloPostgreSQLCTDesarrollo(err)
		}
		if err := comun.publicarInstantanea(ctx, preparada); err != nil {
			return "", err
		}
		return perfilFijoPublicadoInicial, nil
	}
	ahora := s.reloj.Ahora()
	if err != nil {
		registrarFalloPostgreSQLContratacionTemporalDesarrollo("asegurar_perfil_fijo", causaFalloPostgreSQLCTDesarrollo(err))
		return perfilFijoPendienteProvision, nil
	}
	if _, exacta := instantaneaConsumible(publicada, plantilla, ahora); exacta &&
		publicada.actoAsignacion == actoAsignacionPerfilFijoCTDesarrollo {
		return perfilFijoVigente, nil
	}
	huella, errHuella := publicada.instantanea.AsignacionPerfil.HuellaSHA256()
	if errHuella != nil || !aprobacion.valida() || !aprobacion.preimagenes[huella] || !admitida(publicada, ahora) {
		slog.Warn("perfil fijo de RRHH sin asignación consumible: sus peticiones se deniegan hasta la provisión",
			"perfil", p.clave, "perfil_ref", p.perfilRef(), "asignacion_vigente_huella_sha256", huella,
			"estado", string(perfilFijoPendienteProvision))
		return perfilFijoPendienteProvision, nil
	}
	// La preimagen ya se ha comprobado operativa y propia; el CAS la exige
	// exacta bajo bloqueo, así que la guarda de origen no se repite aquí (la
	// preimagen puede venir del perfil dinámico, con otro acto).
	comun.exigirOrigenOperativo = false
	objetivo, err := comun.prepararInstantanea(ctx, plantilla, false)
	if err != nil || objetivo.AsignacionPerfil.Version != publicada.instantanea.AsignacionPerfil.Version+1 ||
		objetivo.AsignacionPerfil.AsignacionID != publicada.instantanea.AsignacionPerfil.AsignacionID {
		return "", falloPostgreSQLCTDesarrollo(err)
	}
	if err := comun.publicarInstantaneaDesdePreimagen(ctx, objetivo, publicada.instantanea); err != nil {
		return "", err
	}
	huellaNueva, _ := objetivo.AsignacionPerfil.HuellaSHA256()
	slog.Info("provisión de perfil fijo de RRHH aplicada",
		"perfil", p.clave, "perfil_ref", p.perfilRef(), "aprobacion_ref", aprobacion.referencia,
		"preimagen_huella_sha256", huella, "version_previa", publicada.instantanea.AsignacionPerfil.Version,
		"version", objetivo.AsignacionPerfil.Version, "asignacion_huella_sha256", huellaNueva,
		"estado", string(perfilFijoProvisionado))
	return perfilFijoProvisionado, nil
}

// preimagenPropiaPerfilFijoCTDesarrollo admite como preimagen de una
// provisión solo una asignación del propio perfil, puesta por uno de los
// actos dados, activa, en vigor y con su rol habilitado.
func preimagenPropiaPerfilFijoCTDesarrollo(p *perfilFijoCTDesarrollo, actos ...string) func(instantaneaPublicadaDesarrollo, time.Time) bool {
	return func(publicada instantaneaPublicadaDesarrollo, ahora time.Time) bool {
		i := publicada.instantanea
		if p == nil || i.AsignacionPerfil.PerfilActivoRef != p.perfilRef() ||
			i.AsignacionPerfil.PrincipalID != p.plantilla.AsignacionPerfil.PrincipalID {
			return false
		}
		for _, acto := range actos {
			if origenOperativoPublicadoCTDesarrollo(publicada, acto, ahora) {
				return true
			}
		}
		return false
	}
}

// lectorAsignacionPublicadaCTDesarrollo lee la asignación vigente de un
// perfil, sin escribir. En la composición real solo lo implementa la
// autoridad PostgreSQL; sin él, las rutas de perfil fijo se deniegan.
type lectorAsignacionPublicadaCTDesarrollo interface {
	leerAsignacionPublicada(ctx context.Context, perfilRef string) (instantaneaPublicadaDesarrollo, bool, error)
}

type estadoConsumoPerfilFijoCTDesarrollo uint8

const (
	perfilFijoConsumoVigente estadoConsumoPerfilFijoCTDesarrollo = iota + 1
	perfilFijoConsumoDenegado
	perfilFijoConsumoFuenteNoDisponible
)

func (a *autoridadPostgreSQLContratacionTemporalDesarrollo) leerAsignacionPublicada(
	ctx context.Context, perfilRef string,
) (instantaneaPublicadaDesarrollo, bool, error) {
	if a == nil || a.pool == nil {
		return instantaneaPublicadaDesarrollo{}, false, falloPostgreSQLCTDesarrollo(nil)
	}
	return leerInstantaneaPublicadaPostgreSQLDesarrollo(ctx, a.pool, perfilRef)
}

// consumirPerfilFijoCTDesarrollo entrega al PDP la plantilla contrastada con
// la asignación publicada. Nunca prepara ni publica.
func (s *soporteAltaContratacionTemporalDesarrollo) consumirPerfilFijoCTDesarrollo(
	ctx context.Context, p *perfilFijoCTDesarrollo,
) (dominiovec.InstantaneaAutorizacion, bool) {
	instantanea, estado := s.consumirPerfilFijoCTDesarrolloConEstado(ctx, p)
	return instantanea, estado == perfilFijoConsumoVigente
}

// El estado distingue una asignación legible pero no consumible de un fallo
// del lector. Entrega GET/POST lo usa para responder 403 o 503 sin revelar
// referencias ni preparar otra asignación.
func (s *soporteAltaContratacionTemporalDesarrollo) consumirPerfilFijoCTDesarrolloConEstado(
	ctx context.Context, p *perfilFijoCTDesarrollo,
) (dominiovec.InstantaneaAutorizacion, estadoConsumoPerfilFijoCTDesarrollo) {
	if s == nil {
		return dominiovec.InstantaneaAutorizacion{}, perfilFijoConsumoFuenteNoDisponible
	}
	s.mu.Lock()
	lector, ok := s.autoridadAsignaciones.(lectorAsignacionPublicadaCTDesarrollo)
	s.mu.Unlock()
	if !ok || dependenciaEsNulaContratacionTemporalDesarrollo(lector) || p == nil {
		return dominiovec.InstantaneaAutorizacion{}, perfilFijoConsumoFuenteNoDisponible
	}
	publicada, encontrada, err := lector.leerAsignacionPublicada(ctx, p.perfilRef())
	estado := perfilFijoConsumoDenegado
	if err != nil || publicada.instantanea.Validar() != nil && encontrada {
		estado = perfilFijoConsumoFuenteNoDisponible
	} else if encontrada && publicada.actoAsignacion == actoAsignacionPerfilFijoCTDesarrollo {
		consumida, valida := instantaneaConsumible(publicada, p.plantilla, s.reloj.Ahora())
		if valida && consumida.Validar() == nil {
			return consumida, perfilFijoConsumoVigente
		}
	}
	causa := "asignacion_no_consumible"
	if err != nil {
		causa = causaFalloPostgreSQLCTDesarrollo(err)
	}
	ahora := time.Now()
	s.mu.Lock()
	avisar := p.avisoNoConsumibleEn.IsZero() || ahora.Sub(p.avisoNoConsumibleEn) >= time.Minute
	if avisar {
		p.avisoNoConsumibleEn = ahora
	}
	s.mu.Unlock()
	if avisar {
		slog.Warn("petición de RRHH denegada: asignación publicada del perfil fijo no consumible",
			"perfil", p.clave, "perfil_ref", p.perfilRef(), "causa", causa)
	}
	return dominiovec.InstantaneaAutorizacion{}, estado
}

// instantaneaPerfilFijoParaContexto valida la solicitud de la ruta como
// siempre y, en lugar de preparar su rol, consume la asignación del perfil.
func (s *soporteAltaContratacionTemporalDesarrollo) instantaneaPerfilFijoParaContexto(
	ctx context.Context, ruta string, p *perfilFijoCTDesarrollo,
) (dominiovec.InstantaneaAutorizacion, bool) {
	if ctx == nil || p == nil {
		return dominiovec.InstantaneaAutorizacion{}, false
	}
	datos, existe := ctx.Value(claveSolicitudAutorizacionContratacionTemporalDesarrollo{}).(dominiovec.DatosSolicitudAutorizacionLigadaV3)
	if existe {
		var valida bool
		switch {
		case ruta == httpinterno.RutaAltaSolicitudes:
			valida = s.solicitudAutorizacionAltaContratacionTemporalDesarrolloValida(ruta, datos)
		case ruta == rutaEntregaPeticionCentro:
			if p.metodo == http.MethodGet {
				valida = datos.Accion == ports.AccionConsultarPeticionesRRHH &&
					solicitudAutorizacionEntregaPeticionValida(ctx, datos)
			} else if p.metodo == http.MethodPost && datos.Accion == ports.AccionCrearSolicitud {
				valida = solicitudAutorizacionAltaDePeticionValida(ctx, datos) &&
					s.centroDeOrganizacionPeticion(datos.Recurso.Ambitos["centro_ref"]) &&
					s.categoriaDeCatalogo(datos.Recurso.Ambitos["categoria_ref"])
			} else {
				valida = p.metodo == http.MethodPost && datos.Accion == ports.AccionEntregarPeticionRRHH &&
					solicitudAutorizacionEntregaPeticionValida(ctx, datos)
			}
		case ruta == httpinterno.RutaPropuestaCobertura:
			valida = solicitudAutorizacionPropuestaCoberturaDesarrolloValida(datos)
		case ruta == httpinterno.RutaDecisionCobertura || ruta == httpinterno.RutaRectificacionCobertura:
			valida = solicitudAutorizacionDecisionCoberturaDesarrolloValida(ruta, datos)
		case ruta == rutaCambiosOrganizacionContratacionTemporalDesarrollo:
			valida = solicitudAutorizacionOrganizacionDesarrolloValida(ctx, datos)
		case ruta == httpinterno.RutaResultadoCobertura:
			valida = true
		case rutaAnalisisContratacionTemporalDesarrollo(ruta):
			fase, ok := s.opcionesCatalogo.faseOperacionVigente(operacionFaseAnalisisCT)
			valida = ok && solicitudAutorizacionAnalisisContratacionTemporalDesarrolloValida(ruta, datos, fase)
		case rutaFirmaDocumentoCTDesarrollo(ruta):
			valida = solicitudAutorizacionConsultaFirmasDocumentoCTDesarrolloValida(ctx, datos) ||
				ruta == httpinterno.RutaFirmaDocumento && (solicitudAutorizacionFirmaDocumentoCTDesarrolloValida(ctx, datos) || solicitudAutorizacionCustodiaFirmadoCTDesarrolloValida(ctx, datos))
		case rutaReincorporacionTitularDesarrollo(ruta):
			valida = solicitudAutorizacionReincorporacionTitularValida(ruta, datos)
		case rutaAsignacionContratacionTemporalDesarrollo(ruta):
			fase, ok := s.opcionesCatalogo.faseOperacionVigente(operacionFaseAsignacionCT)
			valida = ok && solicitudAutorizacionAsignacionContratacionTemporalDesarrolloValida(ruta, datos, fase)
		case ruta == httpinterno.RutaSubsanacionReparos:
			valida = s.solicitudAutorizacionSubsanacionReparosValida(datos)
		case rutaInformeJuridicoContratacionTemporalDesarrollo(ruta):
			fase, ok := s.opcionesCatalogo.faseOperacionVigente(operacionFaseInformeJuridicoCT)
			valida = ok && solicitudAutorizacionInformeJuridicoContratacionTemporalDesarrolloValida(ruta, datos, fase)
		case rutaConsultaRRHHContratacionTemporalDesarrollo(ruta):
			valida = s.lectorConsultasRRHH && s.solicitudAutorizacionConsultaRRHHDesarrolloValida(ruta, datos)
		case ruta == httpinterno.RutaConsultaCircuitoRRHH:
			valida = solicitudAutorizacionConsultaCircuitoRRHHValida(ctx, datos)
		case rutaFirmasR5V2CTDesarrollo(ruta):
			valida = solicitudAutorizacionFirmasR5V2CTDesarrolloValida(ruta, datos)
		case rutaOriginalFirmableCTDesarrollo(ruta):
			valida = solicitudAutorizacionOriginalFirmableCTDesarrolloValida(ctx, datos)
		}
		if !valida {
			return dominiovec.InstantaneaAutorizacion{}, false
		}
	} else if ruta != httpinterno.RutaAltaSolicitudes && ruta != httpinterno.RutaPropuestaCobertura &&
		ruta != httpinterno.RutaResultadoCobertura {
		return dominiovec.InstantaneaAutorizacion{}, false
	}
	return s.consumirPerfilFijoCTDesarrollo(ctx, p)
}

// asegurarPerfilesFijosCTDesarrollo registra el contexto y asegura la
// asignación de cada perfil fijo ya compuesto. Un perfil pendiente de
// provisión no detiene el arranque: sus rutas se deniegan.
func asegurarPerfilesFijosCTDesarrollo(
	ctx context.Context, pool *pgxpool.Pool, s *soporteAltaContratacionTemporalDesarrollo,
	aprobacion aprobacionProvisionPerfilesRRHHDesarrollo, perfiles ...*perfilFijoCTDesarrollo,
) error {
	for _, p := range perfiles {
		if p == nil || p.propioDelSoporte {
			return falloPostgreSQLCTDesarrollo(nil)
		}
		if err := publicarContextoPerfilFijoCTDesarrollo(ctx, pool, p); err != nil {
			return err
		}
		if _, err := asegurarPerfilFijoCTDesarrollo(ctx, pool, s, p, aprobacion,
			preimagenPropiaPerfilFijoCTDesarrollo(p, actoAsignacionPerfilFijoCTDesarrollo)); err != nil {
			return err
		}
	}
	return nil
}

// configurarSesionesPerfilesFijosCTDesarrollo liga a cada perfil fijo su
// contexto registrado y su sesión operativa, derivada de la del perfil
// dinámico (misma cuenta y persona; perfil propio).
func configurarSesionesPerfilesFijosCTDesarrollo(
	ctx context.Context, s *soporteAltaContratacionTemporalDesarrollo, base *proveedorSesionConsultaRRHHDesarrollo,
) error {
	if s == nil || base == nil {
		return ports.ErrConsultaRRHHNoDisponible
	}
	for _, p := range s.perfilesFijosRegistrados() {
		if p.propioDelSoporte {
			continue
		}
		esperado, err := contextoEsperadoRegistradoParaSemillaDesarrollo(ctx, base.resolutor, s, p.contexto.Resultado)
		if err != nil {
			return err
		}
		sesion, err := nuevaSesionReincorporacionTitularDesarrollo(base, esperado)
		if err != nil {
			return err
		}
		s.mu.Lock()
		if p.sesionOperativa != nil {
			s.mu.Unlock()
			return ports.ErrConsultaRRHHNoDisponible
		}
		p.contextoEsperadoRegistrado, p.sesionOperativa = esperado, sesion
		s.mu.Unlock()
	}
	return nil
}

// componerPerfilesFijosAltaCoberturaCTDesarrollo compone los perfiles fijos
// del alta directa (organización, centro y categoría), de la cobertura
// (organización y unidad ejecutora) y del análisis (organización y fase y
// estado previos del catálogo). La entrega usa dos perfiles: GET lector
// con organización y POST más alta anidada con organización, centro y
// categoría. La reserva y el recibo sellan el perfil de POST.
func componerPerfilesFijosAltaCoberturaCTDesarrollo(
	s *soporteAltaContratacionTemporalDesarrollo, principal dominiovec.Principal, ahora time.Time,
	origen *origenConsultasContratacionTemporalDesarrollo,
) error {
	if s == nil {
		return errAltaContratacionTemporalDesarrolloNoDisponible
	}
	alta, err := nuevoPerfilFijoCTDesarrollo(principal, s.contexto, ahora, clavePerfilFijoAltaCTDesarrollo,
		[]string{httpinterno.RutaAltaSolicitudes},
		func(principalID, perfilRef string) (dominiovec.InstantaneaAutorizacion, error) {
			return nuevaInstantaneaAutorizacionAltaContratacionTemporalDesarrollo(principalID, perfilRef, ahora, origen)
		})
	if err != nil {
		return err
	}
	var entrega, lectorEntrega *perfilFijoCTDesarrollo
	if origen != nil {
		catalogos, errCatalogo := origen.catalogosAlta()
		if errCatalogo != nil {
			return errCatalogo
		}
		if len(catalogos.centrosOrganizacion) == 0 {
			origen = nil
		}
	}
	if origen != nil {
		entrega, err = nuevoPerfilFijoCTDesarrollo(principal, s.contexto, ahora, clavePerfilFijoEntregaCTDesarrollo,
			[]string{rutaEntregaPeticionCentro},
			func(principalID, perfilRef string) (dominiovec.InstantaneaAutorizacion, error) {
				return nuevaInstantaneaAutorizacionEntregaPeticionDesarrollo(principalID, perfilRef, ahora, origen)
			})
		if err != nil {
			return err
		}
		entrega.metodo = http.MethodPost
		lectorEntrega, err = nuevoPerfilFijoCTDesarrollo(principal, s.contexto, ahora, clavePerfilFijoLectorEntregaCTDesarrollo,
			[]string{rutaEntregaPeticionCentro},
			func(principalID, perfilRef string) (dominiovec.InstantaneaAutorizacion, error) {
				return nuevaInstantaneaAutorizacionLectorEntregaPeticionDesarrollo(principalID, perfilRef, ahora)
			})
		if err != nil {
			return err
		}
		lectorEntrega.metodo = http.MethodGet
	}
	cobertura, err := nuevoPerfilFijoCTDesarrollo(principal, s.contexto, ahora, clavePerfilFijoCoberturaCTDesarrollo,
		[]string{httpinterno.RutaPropuestaCobertura, httpinterno.RutaDecisionCobertura,
			httpinterno.RutaRectificacionCobertura, httpinterno.RutaResultadoCobertura},
		func(principalID, perfilRef string) (dominiovec.InstantaneaAutorizacion, error) {
			return nuevaInstantaneaAutorizacionCoberturaContratacionTemporalDesarrollo(principalID, perfilRef, ahora)
		})
	if err != nil {
		return err
	}
	// Análisis (corte 3): la organización y las fases y el estado previos del
	// catálogo (c23). El expediente va en la referencia del recurso.
	faseAnalisis, ok := s.opcionesCatalogo.faseOperacionVigente(operacionFaseAnalisisCT)
	if !ok {
		return errAltaContratacionTemporalDesarrolloNoDisponible
	}
	analisis, err := nuevoPerfilFijoCTDesarrollo(principal, s.contexto, ahora, clavePerfilFijoAnalisisCTDesarrollo,
		[]string{httpinterno.RutaRegistroAnalisisRRHH, httpinterno.RutaRectificacionAnalisisRRHH},
		func(principalID, perfilRef string) (dominiovec.InstantaneaAutorizacion, error) {
			return nuevaInstantaneaAutorizacionAnalisisContratacionTemporalDesarrollo(principalID, perfilRef, ahora, faseAnalisis)
		})
	if err != nil {
		return err
	}
	// Asignación a unidad e informe jurídico (corte 3): cada uno con su perfil,
	// también sin expediente en el permiso.
	asignacion, err := nuevoPerfilFijoCTDesarrollo(principal, s.contexto, ahora, clavePerfilFijoAsignacionCTDesarrollo,
		[]string{httpinterno.RutaAsignaciones},
		func(principalID, perfilRef string) (dominiovec.InstantaneaAutorizacion, error) {
			return nuevaInstantaneaAutorizacionAsignacionContratacionTemporalDesarrollo(principalID, perfilRef, ahora,
				faseDeOperacionCTDesarrollo(s.opcionesCatalogo, operacionFaseAsignacionCT))
		})
	if err != nil {
		return err
	}
	informe, err := nuevoPerfilFijoCTDesarrollo(principal, s.contexto, ahora, clavePerfilFijoInformeCTDesarrollo,
		[]string{httpinterno.RutaPreparacionesInformeJuridico},
		func(principalID, perfilRef string) (dominiovec.InstantaneaAutorizacion, error) {
			return nuevaInstantaneaAutorizacionInformeJuridicoContratacionTemporalDesarrollo(principalID, perfilRef, ahora,
				faseDeOperacionCTDesarrollo(s.opcionesCatalogo, operacionFaseInformeJuridicoCT))
		})
	if err != nil {
		return err
	}
	for _, p := range []*perfilFijoCTDesarrollo{alta, entrega, lectorEntrega, cobertura, analisis, asignacion, informe} {
		if p == nil {
			continue
		}
		if err := s.registrarPerfilFijoCTDesarrollo(p); err != nil {
			return err
		}
	}
	return nil
}

// asignarPerfilesFijosEnFronterasCTDesarrollo hace que la frontera común de
// cada ruta con perfil fijo admita solo ese perfil (antes, el dinámico).
func asignarPerfilesFijosEnFronterasCTDesarrollo(
	s *soporteAltaContratacionTemporalDesarrollo, perfilDinamico string, declaraciones []descriptorFronteraComunDesarrollo,
) ([]descriptorFronteraComunDesarrollo, error) {
	resultado := append([]descriptorFronteraComunDesarrollo(nil), declaraciones...)
	for i := range resultado {
		d := &resultado[i]
		fijo := s.perfilFijoParaRutaYMetodo(d.Ruta, d.Metodo)
		if fijo == nil {
			continue
		}
		if len(d.PerfilesActivosRef) != 1 || d.PerfilesActivosRef[0] != perfilDinamico ||
			!perfilActivoSeguridadComunValido(fijo.perfilRef()) {
			return nil, ErrActivacionDesarrolloInvalida
		}
		d.PerfilesActivosRef = []string{fijo.perfilRef()}
	}
	return resultado, nil
}

func aprobacionProvisionPerfilesRRHHDesdeConfig(cfg config.Config) aprobacionProvisionPerfilesRRHHDesarrollo {
	return aprobacionProvisionPerfilesRRHHDesarrollo{
		referencia:  cfg.CTProvisionPerfilesRRHHAprobacionRef(),
		preimagenes: cfg.CTProvisionPerfilesRRHHPreimagenes(),
	}
}

const rolLectorConsultaRRHHDesarrollo = "consulta_rrhh_lector_desarrollo"

// componerPerfilFijoLectorRRHHDesarrollo convierte el perfil de un lector de
// consulta (no el técnico, que comparte perfil con incorporación y
// continuidad) en perfil fijo: un solo rol con la consulta de la bandeja y la
// del expediente, con los ámbitos de siempre, que solo se consume. Antes el
// perfil alternaba en cada petición entre el rol de la bandeja y el del
// expediente. La provisión admite como preimagen esa asignación alternante
// (acto del perfil dinámico) si sigue operativa y con los mismos ámbitos.
func componerPerfilFijoLectorRRHHDesarrollo(
	ctx context.Context, pool *pgxpool.Pool, s *soporteAltaContratacionTemporalDesarrollo,
	aprobacion aprobacionProvisionPerfilesRRHHDesarrollo,
) error {
	if s == nil || pool == nil || !s.lectorConsultasRRHH || s.tecnicoConsultaRRHH {
		return ports.ErrConsultaRRHHNoDisponible
	}
	s.mu.Lock()
	cuadro := clonarInstantaneaAutorizacionAltaContratacionTemporalDesarrollo(s.instantaneaCuadroRRHH)
	detalle := clonarInstantaneaAutorizacionAltaContratacionTemporalDesarrollo(s.instantaneaDetalleRRHH)
	s.mu.Unlock()
	// El perfil fijo del lector conserva su plantilla: solo la consulta del
	// expediente, sin la descarga de borradores. Concederla a un lector exige
	// una provisión aprobada aparte; hasta entonces su descarga se deniega.
	var consultaDetalle []dominiovec.ConcesionRol
	for _, c := range detalle.VersionRol.Concesiones {
		if c.Accion == ports.AccionConsultarDetalleRRHH {
			consultaDetalle = append(consultaDetalle, c)
		}
	}
	if cuadro.Validar() != nil || detalle.Validar() != nil || len(cuadro.VersionRol.Concesiones) != 1 ||
		len(consultaDetalle) != 1 {
		return ports.ErrConsultaRRHHNoDisponible
	}
	v, err := s.contexto.Vinculo.Datos()
	if err != nil {
		return ports.ErrConsultaRRHHNoDisponible
	}
	plantilla, err := nuevaInstantaneaAutorizacionContratacionTemporalDesarrollo(v.PrincipalID, v.PerfilActivoRef, s.reloj.Ahora(),
		rolLectorConsultaRRHHDesarrollo, "Consulta de bandeja y expediente de desarrollo", rolLectorConsultaRRHHDesarrollo,
		[]dominiovec.ConcesionRol{cuadro.VersionRol.Concesiones[0], consultaDetalle[0]},
		cuadro.AsignacionPerfil.Ambitos)
	if err != nil {
		return ports.ErrConsultaRRHHNoDisponible
	}
	p := &perfilFijoCTDesarrollo{clave: "lector", contexto: s.contexto, plantilla: plantilla,
		rutas:      map[string]struct{}{httpinterno.RutaConsultaCuadroRRHH: {}, httpinterno.RutaConsultaDetalleRRHH: {}},
		actoSesion: "acto:ct:desarrollo:sesion:v1", propioDelSoporte: true}
	if err := s.registrarPerfilFijoCTDesarrollo(p); err != nil {
		return err
	}
	admitida := func(publicada instantaneaPublicadaDesarrollo, ahora time.Time) bool {
		i := publicada.instantanea
		if !preimagenPropiaPerfilFijoCTDesarrollo(p, actoAsignacionPerfilFijoCTDesarrollo, actoAsignacionCTDesarrollo)(publicada, ahora) {
			return false
		}
		rol := i.VersionRol.RolID
		if rol != cuadro.VersionRol.RolID && rol != detalle.VersionRol.RolID && rol != rolLectorConsultaRRHHDesarrollo {
			return false
		}
		// Mismos ámbitos que la plantilla: la provisión solo cambia el rol.
		con := i
		con.AsignacionPerfil.Ambitos = plantilla.AsignacionPerfil.Ambitos
		a, errA := con.AsignacionPerfil.HuellaSHA256()
		b, errB := i.AsignacionPerfil.HuellaSHA256()
		return errA == nil && errB == nil && a == b
	}
	if _, err := asegurarPerfilFijoCTDesarrollo(ctx, pool, s, p, aprobacion, admitida); err != nil {
		return err
	}
	return nil
}
