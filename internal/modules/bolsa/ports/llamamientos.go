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
		dominiobolsa.PropuestaLlamamiento,
		puertosvec.EvidenciaUsoDecisionAutorizacion,
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
