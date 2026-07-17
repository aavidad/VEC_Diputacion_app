// Package ports define las fronteras hexagonales del modulo de bolsa.
package ports

import (
	"context"
	"errors"
	"regexp"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	dominiobolsa "vec-diputacion-granada/internal/modules/bolsa/domain"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
	puertosvec "vec-diputacion-granada/internal/vec/ports"
)

const (
	AccionProponerLlamamiento    = "bolsa.llamamiento.proponer"
	FinalidadProponerLlamamiento = "gestion_propuestas_llamamiento"
	ModuloLlamamientos           = "bolsa"
	TipoRecursoNecesidad         = "necesidad_cobertura"
)

var (
	ErrSolicitudPropuestaLlamamientoInvalida      = errors.New("bolsa: solicitud de propuesta de llamamiento invalida")
	ErrRecursoNecesidadNoEncontrado               = errors.New("bolsa: recurso de necesidad no encontrado")
	ErrRecursoNecesidadAmbiguo                    = errors.New("bolsa: recurso de necesidad ambiguo")
	ErrRecursoNecesidadNoConfiable                = errors.New("bolsa: recurso de necesidad no confiable")
	ErrDatosLlamamientoNoEncontrados              = errors.New("bolsa: datos de llamamiento no encontrados")
	ErrDatosLlamamientoAmbiguos                   = errors.New("bolsa: datos de llamamiento ambiguos")
	ErrDatosLlamamientoNoConfiables               = errors.New("bolsa: datos de llamamiento no confiables")
	ErrMotorElegibilidadNoDisponible              = errors.New("bolsa: motor de elegibilidad no disponible")
	ErrEvaluacionMotorNoConfiable                 = errors.New("bolsa: evaluacion del motor no confiable")
	ErrGeneracionReferenciaLlamamiento            = errors.New("bolsa: no se pudo generar una referencia de llamamiento")
	ErrPersistenciaPropuestaNoDisponible          = errors.New("bolsa: persistencia de propuesta no disponible")
	ErrPropuestaLlamamientoYaExiste               = errors.New("bolsa: propuesta de llamamiento ya existe")
	ErrNecesidadLlamamientoYaPropuesta            = errors.New("bolsa: la version de la necesidad ya tiene propuesta de llamamiento")
	ErrReferenciaLlamamientoYaUtilizada           = errors.New("bolsa: referencia de llamamiento ya utilizada")
	ErrDecisionAutorizacionLlamamientoUsada       = errors.New("bolsa: decision de autorizacion ya consumida")
	ErrCapacidadMemoriaLlamamientosAgotada        = errors.New("bolsa: capacidad del adaptador de memoria agotada")
	ErrSerializacionSolicitudLlamamientoProhibida = errors.New("bolsa: serializacion de solicitud interna de llamamiento prohibida")
	ErrComandoGuardarPropuestaLlamamientoInvalido = errors.New("bolsa: comando para guardar propuesta de llamamiento invalido")
)

var (
	patronDocumentoPersonalLlamamiento = regexp.MustCompile(`(?i)(([0-9][._:/#-]?){8}|[XYZ][._:/#-]?([0-9][._:/#-]?){7})[A-Z]`)
	patronEtiquetaPersonalLlamamiento  = regexp.MustCompile(`(?i)(^|[._:/#-])(dni|nie|nif|pasaporte|passport)([._:/#-]|$)`)
)

// SolicitudProponerLlamamiento es el unico dato de entrada del caso de uso.
// Actor procede del resolutor canonico del nucleo: no se acepta identidad,
// perfiles, roles ni atributos reconstruidos por un adaptador HTTP. El perfil
// se repite deliberadamente para exigir una seleccion expresa y coincidente.
// No hay campos para listados, participaciones, estados ni evaluaciones.
type SolicitudProponerLlamamiento struct {
	Actor            dominiovec.ContextoActor
	PerfilActivoRef  string
	AutenticacionRef string
	SesionRef        string
	NecesidadRef     string
	CorrelacionRef   string
}

func (s SolicitudProponerLlamamiento) Validar() error {
	revalidacion := dominiovec.SolicitudRevalidacionAutenticacionActorV1{
		AutenticacionRef: s.AutenticacionRef, SesionRef: s.SesionRef,
	}
	if s.Actor.Validar() != nil || s.PerfilActivoRef != s.Actor.PerfilActivoRef ||
		!ReferenciaOpacaLlamamientoValida(s.PerfilActivoRef) ||
		revalidacion.Validar() != nil ||
		!ReferenciaOpacaLlamamientoValida(s.NecesidadRef) ||
		!ReferenciaOpacaLlamamientoValida(s.CorrelacionRef) {
		return ErrSolicitudPropuestaLlamamientoInvalida
	}
	return nil
}

// CreadorVinculoAutenticacionActor revalida sesion, cuenta, superficie y
// garantia mediante el servicio del nucleo. La solicitud externa solo aporta
// dos referencias opacas; nunca puede rellenar el bloque de hechos resultante.
type CreadorVinculoAutenticacionActor interface {
	Crear(
		context.Context,
		dominiovec.SolicitudRevalidacionAutenticacionActorV1,
		dominiovec.ContextoActor,
	) (dominiovec.VinculoAutenticacionActorV1, error)
}

// VinculoAptoParaGestionLlamamientos comprueba la frontera de autenticacion
// reforzada de una operacion interna de RRHH. No concede autorizacion: exige
// que el vinculo opaco proceda de la superficie corporativa con cuenta
// ordinaria o de administracion con cuenta privilegiada, y que conserve
// exactamente el contexto y el perfil resueltos con garantia alta. La
// superficie personal externa y el metodo de demostracion nunca habilitan el
// acceso aunque un PDP defectuoso tratase de concederlo.
func VinculoAptoParaGestionLlamamientos(
	vinculo dominiovec.VinculoAutenticacionActorV1,
	actor dominiovec.ContextoActor,
	perfilActivoRef string,
) bool {
	datos, err := vinculo.Datos()
	superficieCorporativa := datos.Superficie == dominiovec.SuperficieAutenticacionInternaCorporativaV1 &&
		!datos.CuentaPrivilegiada
	superficiePrivilegiada := datos.Superficie == dominiovec.SuperficieAutenticacionAdministracionPrivilegiadaV1 &&
		datos.CuentaPrivilegiada
	return err == nil && actor.Validar() == nil && vinculo.ValidarPara(actor) == nil &&
		ReferenciaOpacaLlamamientoValida(perfilActivoRef) &&
		perfilActivoRef == actor.PerfilActivoRef && datos.PerfilActivoRef == perfilActivoRef &&
		datos.GarantiaObservada == dominiovec.AuthAssuranceHigh &&
		datos.MetodoObservado != dominiovec.AuthMethodDemo &&
		(superficieCorporativa || superficiePrivilegiada)
}

func (s SolicitudProponerLlamamiento) Clonar() (SolicitudProponerLlamamiento, error) {
	if err := s.Validar(); err != nil {
		return SolicitudProponerLlamamiento{}, err
	}
	actor, err := s.Actor.Clonar()
	if err != nil {
		return SolicitudProponerLlamamiento{}, ErrSolicitudPropuestaLlamamientoInvalida
	}
	s.Actor = actor
	return s, nil
}

// El comando es interno y no puede usarse como DTO HTTP. La frontera de
// entrada debe resolver ContextoActor en middleware confiable y construir el
// comando de forma deliberada a partir de referencias ya validadas.
func (SolicitudProponerLlamamiento) MarshalJSON() ([]byte, error) {
	return nil, ErrSerializacionSolicitudLlamamientoProhibida
}

func (*SolicitudProponerLlamamiento) UnmarshalJSON([]byte) error {
	return ErrSerializacionSolicitudLlamamientoProhibida
}

// ResolutorRecursoNecesidad devuelve todas las coincidencias autoritativas. El
// caso de uso exige exactamente una; el puerto no puede ocultar ambiguedades
// mediante LIMIT 1 ni escoger la primera fila.
type ResolutorRecursoNecesidad interface {
	ResolverRecursosNecesidad(context.Context, string) ([]dominiovec.RecursoAutorizable, error)
}

// DatosAutoritativosLlamamiento agrupa la version juridica exacta de todos los
// insumos. Entradas procede del repositorio, nunca de la solicitud externa.
// La aplicacion es quien crea la instantanea de orden y liga su contenido
// mediante una huella. La huella no autentica ni constituye una firma: esa
// garantia exige una atestacion criptografica en el adaptador duradero.
type DatosAutoritativosLlamamiento struct {
	Bolsa     dominiobolsa.BolsaConstituida
	Necesidad dominiobolsa.NecesidadCobertura
	Politica  dominiobolsa.ReferenciaPoliticaLlamamiento
	Entradas  []dominiobolsa.EntradaOrdenBolsa
}

func (d DatosAutoritativosLlamamiento) Clonar() (DatosAutoritativosLlamamiento, error) {
	bolsa, err := d.Bolsa.ClonarCanonica()
	if err != nil {
		return DatosAutoritativosLlamamiento{}, ErrDatosLlamamientoNoConfiables
	}
	necesidad, err := d.Necesidad.ClonarCanonica()
	if err != nil {
		return DatosAutoritativosLlamamiento{}, ErrDatosLlamamientoNoConfiables
	}
	politica, err := d.Politica.ClonarCanonica()
	if err != nil || len(d.Entradas) == 0 {
		return DatosAutoritativosLlamamiento{}, ErrDatosLlamamientoNoConfiables
	}
	entradas := make([]dominiobolsa.EntradaOrdenBolsa, len(d.Entradas))
	for indice, entrada := range d.Entradas {
		participacion, err := entrada.Participacion.ClonarCanonica()
		if err != nil || entrada.Orden == 0 {
			return DatosAutoritativosLlamamiento{}, ErrDatosLlamamientoNoConfiables
		}
		entradas[indice] = dominiobolsa.EntradaOrdenBolsa{Orden: entrada.Orden, Participacion: participacion}
	}
	return DatosAutoritativosLlamamiento{Bolsa: bolsa, Necesidad: necesidad, Politica: politica, Entradas: entradas}, nil
}

// FuenteDatosLlamamiento se consulta solo despues de una concesion exacta. Al
// devolver todas las coincidencias permite que la aplicacion deniegue cero o
// multiples versiones sin revelar cual de ellas existe.
type FuenteDatosLlamamiento interface {
	CargarDatosAutoritativosLlamamiento(context.Context, string) ([]DatosAutoritativosLlamamiento, error)
}

// SolicitudEvaluarParticipacionLlamamiento contiene una sola posicion. El
// motor no recibe el resto del listado y por tanto no puede saltar posiciones.
// Estado y criterios siguen siendo referencias versionadas, no enumeraciones
// favorables codificadas en la aplicacion.
type SolicitudEvaluarParticipacionLlamamiento struct {
	Necesidad               dominiobolsa.NecesidadCobertura
	InstantaneaRef          string
	VersionInstantanea      uint64
	HuellaInstantaneaSHA256 string
	InstanteReferencia      time.Time
	InstantaneaGeneradaEn   time.Time
	Politica                dominiobolsa.ReferenciaPoliticaLlamamiento
	Entrada                 dominiobolsa.EntradaOrdenBolsa
	EvaluadaEn              time.Time
}

func (s SolicitudEvaluarParticipacionLlamamiento) Validar() error {
	if s.Necesidad.Validar() != nil || s.Politica.Validar() != nil ||
		!ReferenciaOpacaLlamamientoValida(s.InstantaneaRef) || s.VersionInstantanea == 0 ||
		!huellaSHA256LlamamientoValida(s.HuellaInstantaneaSHA256) ||
		!instanteCanonicoLlamamiento(s.InstanteReferencia) ||
		!instanteCanonicoLlamamiento(s.InstantaneaGeneradaEn) ||
		s.InstantaneaGeneradaEn.Before(s.InstanteReferencia) ||
		s.Entrada.Orden == 0 || s.Entrada.Participacion.Validar() != nil ||
		!instanteCanonicoLlamamiento(s.EvaluadaEn) || s.EvaluadaEn.Before(s.InstantaneaGeneradaEn) {
		return ErrEvaluacionMotorNoConfiable
	}
	return nil
}

type MotorElegibilidadLlamamiento interface {
	EvaluarParticipacion(context.Context, SolicitudEvaluarParticipacionLlamamiento) (dominiobolsa.EvaluacionParticipacionLlamamiento, error)
}

type RelojLlamamientos interface {
	Ahora() time.Time
}

// GeneradorReferenciasLlamamiento es la unica autoridad para identificadores
// nuevos. Version 1 pertenece a cada instantanea nueva de referencia unica; un
// futuro repositorio que versione la misma referencia debe aportar otro puerto
// y CAS, no inferir una version en este caso de uso.
type GeneradorReferenciasLlamamiento interface {
	NuevaReferenciaInstantaneaOrdenBolsa() (string, error)
	NuevaReferenciaPropuestaLlamamiento() (string, error)
}

// ComandoGuardarPropuestaLlamamiento conserva, como una unica capacidad opaca,
// todos los datos que una persistencia duradera debe confirmar de forma
// indivisible. La propuesta solo contiene el prefijo evaluado; por eso el
// comando retiene ademas la instantanea completa que genero dicho prefijo.
//
// Sus campos son privados y Datos devuelve copias profundas. Un adaptador no
// puede completar la instantanea consultando de nuevo una fuente mutable ni
// aceptar propuesta y evidencia por parametros independientes.
type ComandoGuardarPropuestaLlamamiento struct {
	datos *datosComandoGuardarPropuestaLlamamiento
}

type datosComandoGuardarPropuestaLlamamiento struct {
	instantanea dominiobolsa.InstantaneaOrdenBolsa
	propuesta   dominiobolsa.PropuestaLlamamiento
	evidencia   puertosvec.EvidenciaUsoDecisionAutorizacion
}

func NuevoComandoGuardarPropuestaLlamamiento(
	instantanea dominiobolsa.InstantaneaOrdenBolsa,
	propuesta dominiobolsa.PropuestaLlamamiento,
	evidencia puertosvec.EvidenciaUsoDecisionAutorizacion,
) (ComandoGuardarPropuestaLlamamiento, error) {
	instantaneaCanonica, err := instantanea.ClonarCanonica()
	if err != nil {
		return ComandoGuardarPropuestaLlamamiento{}, ErrComandoGuardarPropuestaLlamamientoInvalido
	}
	propuestaCanonica, err := propuesta.ClonarCanonica()
	if err != nil || !instantaneaYPropuestaLlamamientoCoinciden(instantaneaCanonica, propuestaCanonica) {
		return ComandoGuardarPropuestaLlamamiento{}, ErrComandoGuardarPropuestaLlamamientoInvalido
	}
	if !evidenciaLlamamientoLigadaYVigente(evidencia, propuestaCanonica) {
		return ComandoGuardarPropuestaLlamamiento{}, errors.Join(
			ErrComandoGuardarPropuestaLlamamientoInvalido,
			puertosvec.ErrEvidenciaUsoDecisionAutorizacionInvalida,
		)
	}
	return ComandoGuardarPropuestaLlamamiento{datos: &datosComandoGuardarPropuestaLlamamiento{
		instantanea: instantaneaCanonica,
		propuesta:   propuestaCanonica,
		evidencia:   evidencia,
	}}, nil
}

// Datos devuelve una fotografia defensiva del conjunto indivisible. La
// evidencia ya es una capacidad opaca e inmutable del nucleo; instantanea y
// propuesta se clonan para no compartir slices con el adaptador.
func (c ComandoGuardarPropuestaLlamamiento) Datos() (
	dominiobolsa.InstantaneaOrdenBolsa,
	dominiobolsa.PropuestaLlamamiento,
	puertosvec.EvidenciaUsoDecisionAutorizacion,
	error,
) {
	if c.datos == nil {
		return dominiobolsa.InstantaneaOrdenBolsa{}, dominiobolsa.PropuestaLlamamiento{},
			puertosvec.EvidenciaUsoDecisionAutorizacion{}, ErrComandoGuardarPropuestaLlamamientoInvalido
	}
	instantanea, err := c.datos.instantanea.ClonarCanonica()
	if err != nil {
		return dominiobolsa.InstantaneaOrdenBolsa{}, dominiobolsa.PropuestaLlamamiento{},
			puertosvec.EvidenciaUsoDecisionAutorizacion{}, ErrComandoGuardarPropuestaLlamamientoInvalido
	}
	propuesta, err := c.datos.propuesta.ClonarCanonica()
	if err != nil || !instantaneaYPropuestaLlamamientoCoinciden(instantanea, propuesta) ||
		!evidenciaLlamamientoLigadaYVigente(c.datos.evidencia, propuesta) {
		return dominiobolsa.InstantaneaOrdenBolsa{}, dominiobolsa.PropuestaLlamamiento{},
			puertosvec.EvidenciaUsoDecisionAutorizacion{}, ErrComandoGuardarPropuestaLlamamientoInvalido
	}
	return instantanea, propuesta, c.datos.evidencia, nil
}

// ValidarEn revalida la capacidad en el reloj efectivo del adaptador. No
// sustituye el bloqueo y la relectura autoritativa dentro de la transaccion.
func (c ComandoGuardarPropuestaLlamamiento) ValidarEn(instante time.Time) error {
	_, propuesta, evidencia, err := c.Datos()
	if err != nil || !instanteCanonicoLlamamiento(instante) || propuesta.GeneradaEn.After(instante) ||
		evidencia.ValidarEn(instante) != nil {
		return errors.Join(
			ErrComandoGuardarPropuestaLlamamientoInvalido,
			puertosvec.ErrEvidenciaUsoDecisionAutorizacionInvalida,
		)
	}
	return nil
}

func instantaneaYPropuestaLlamamientoCoinciden(
	instantanea dominiobolsa.InstantaneaOrdenBolsa,
	propuesta dominiobolsa.PropuestaLlamamiento,
) bool {
	if instantanea.Validar() != nil || propuesta.Validar() != nil ||
		propuesta.InstantaneaRef != instantanea.InstantaneaRef ||
		propuesta.VersionInstantanea != instantanea.Version ||
		propuesta.HuellaInstantaneaSHA256 != instantanea.HuellaContenidoSHA256 ||
		propuesta.BolsaRef != instantanea.BolsaRef ||
		propuesta.VersionBolsa != instantanea.VersionBolsa ||
		propuesta.HuellaBolsaSHA256 != instantanea.HuellaBolsaSHA256 ||
		!propuesta.InstanteReferencia.Equal(instantanea.ReferidaEn) ||
		!propuesta.InstantaneaGeneradaEn.Equal(instantanea.GeneradaEn) ||
		propuesta.TotalParticipacionesInstantanea != uint64(len(instantanea.Entradas)) ||
		len(propuesta.Evaluaciones) > len(instantanea.Entradas) {
		return false
	}
	for indice, evaluacion := range propuesta.Evaluaciones {
		entrada := instantanea.Entradas[indice]
		situacion, vigente := entrada.Participacion.SituacionVigenteEn(instantanea.ReferidaEn)
		if !vigente || evaluacion.Orden != entrada.Orden ||
			evaluacion.ParticipacionRef != entrada.Participacion.ParticipacionRef ||
			evaluacion.SujetoRef != entrada.Participacion.SujetoRef ||
			evaluacion.SituacionSecuencia != situacion.Secuencia ||
			evaluacion.EstadoClave != situacion.EstadoClave ||
			evaluacion.EstadoVersion != situacion.EstadoVersion ||
			evaluacion.HuellaEstadoSHA256 != situacion.HuellaEstadoSHA256 {
			return false
		}
	}
	return true
}

func evidenciaLlamamientoLigadaYVigente(
	evidencia puertosvec.EvidenciaUsoDecisionAutorizacion,
	propuesta dominiobolsa.PropuestaLlamamiento,
) bool {
	datos, err := evidencia.Datos()
	if err != nil || evidencia.ValidarEn(propuesta.GeneradaEn) != nil ||
		!datos.VerificadaEn.Equal(propuesta.GeneradaEn) {
		return false
	}
	decision := datos.Decision
	vinculo, err := decision.VinculoAutenticacionActor.Datos()
	if err != nil {
		return false
	}
	superficieCorporativa := vinculo.Superficie == dominiovec.SuperficieAutenticacionInternaCorporativaV1 &&
		!vinculo.CuentaPrivilegiada
	superficiePrivilegiada := vinculo.Superficie == dominiovec.SuperficieAutenticacionAdministracionPrivilegiadaV1 &&
		vinculo.CuentaPrivilegiada
	return decision.ValidarEvidenciaInstantanea() == nil && decision.Concedida &&
		decision.Accion == AccionProponerLlamamiento && decision.RecursoRef == propuesta.NecesidadRef &&
		decision.ModuloID == ModuloLlamamientos && decision.TipoRecurso == TipoRecursoNecesidad &&
		decision.Finalidad == FinalidadProponerLlamamiento &&
		decision.GarantiaMinima == dominiovec.AuthAssuranceHigh &&
		len(decision.CamposPermitidos) == 0 && len(decision.Obligaciones) == 0 &&
		vinculo.GarantiaObservada == dominiovec.AuthAssuranceHigh &&
		vinculo.MetodoObservado != dominiovec.AuthMethodDemo &&
		(superficieCorporativa || superficiePrivilegiada)
}

// TransaccionPropuestasLlamamiento consume la concesion y confirma el efecto
// de negocio en una unica operacion. Un adaptador duradero debe bloquear/releer
// necesidad y vigencia, validar la atestacion del PDP y hacer COMMIT de
// propuesta, unicidad de la version inmutable de necesidad, uso unico de
// decision, auditoria y outbox de forma indivisible. La clave gobernada minima
// es (necesidad_ref, version_necesidad, huella_necesidad); una reapertura debe
// ser explicita y producir otra version, nunca reemplazar esta propuesta.
// La implementacion en memoria solo cubre semantica e idempotencia para tests.
type TransaccionPropuestasLlamamiento interface {
	GuardarPropuestaLlamamiento(
		context.Context,
		ComandoGuardarPropuestaLlamamiento,
	) error
}

// ReferenciaOpacaLlamamientoValida evita documentos personales evidentes,
// comodines, controles, espacios no canonicos y texto Unicode ambiguo. No
// pretende sustituir el emisor criptograficamente aleatorio de referencias.
func ReferenciaOpacaLlamamientoValida(valor string) bool {
	if valor == "" || len(valor) > 512 || valor != strings.TrimSpace(valor) || strings.ContainsRune(valor, '*') ||
		!utf8.ValidString(valor) || patronDocumentoPersonalLlamamiento.MatchString(valor) ||
		patronEtiquetaPersonalLlamamiento.MatchString(valor) {
		return false
	}
	for _, caracter := range valor {
		if unicode.IsControl(caracter) || unicode.Is(unicode.Pattern_White_Space, caracter) ||
			unicode.Is(unicode.Bidi_Control, caracter) || caracter == unicode.ReplacementChar {
			return false
		}
	}
	return true
}

func instanteCanonicoLlamamiento(instante time.Time) bool {
	return !instante.IsZero() && instante.Location() == time.UTC && instante.Year() >= 1 && instante.Year() <= 9999 &&
		instante.Nanosecond()%1_000 == 0
}

func huellaSHA256LlamamientoValida(valor string) bool {
	if len(valor) != 64 {
		return false
	}
	for _, caracter := range valor {
		if (caracter < '0' || caracter > '9') && (caracter < 'a' || caracter > 'f') {
			return false
		}
	}
	return true
}
