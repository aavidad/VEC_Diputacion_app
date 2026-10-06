package ports

import (
	"context"
	"errors"
	"time"

	dominiobolsa "vec-diputacion-granada/internal/modules/bolsa/domain"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
	puertosvec "vec-diputacion-granada/internal/vec/ports"
)

// B4: datos de contacto (correo y dos teléfonos) de una participación en bolsa.
const (
	AccionRegistrarDatosContactoParticipacion    = "bolsa.datos_contacto_participacion.registrar"
	FinalidadRegistrarDatosContactoParticipacion = "gestion_datos_contacto_participacion"
	AudienciaRegistrarDatosContactoParticipacion = "vec_bolsa_llamamientos.datos_contacto_participacion.registrar.v1"
	ClaveRefSobreDatosContactoDesarrollo         = "clave:kms:desarrollo:datos-contacto-participacion:v1"
	// Consulta RRHH del correo y los teléfonos completos: acción, finalidad y
	// audiencia propias, distintas de las de registro (AD197/B78).
	AccionConsultarDatosContactoParticipacion    = "bolsa.datos_contacto_participacion.consultar"
	FinalidadConsultarDatosContactoParticipacion = "consulta_datos_contacto_participacion"
	AudienciaConsultarDatosContactoParticipacion = "vec_bolsa_llamamientos.datos_contacto_participacion.consultar.v1"
)

var (
	ErrDatosContactoParticipacionNoDisponibles = errors.New("bolsa: datos de contacto de participacion no disponibles")
	ErrDatosContactoParticipacionNoEncontrados = errors.New("bolsa: datos de contacto de participacion no encontrados")
)

// SobreDatosContacto es lo único que cruza la frontera durable: el canónico
// cifrado con AEAD, ligado a la participación y a la versión por los datos
// asociados. No contiene ni permite reconstruir el claro sin el KMS.
type SobreDatosContacto struct {
	Version  uint64
	ClaveRef string
	Nonce    []byte
	Cifrado  []byte
}

func (s SobreDatosContacto) Validar() error {
	if s.Version == 0 || s.ClaveRef == "" || len(s.Nonce) == 0 || len(s.Cifrado) == 0 {
		return ErrDatosContactoParticipacionNoDisponibles
	}
	return nil
}

// CifradorDatosContactoParticipacion protege el canónico. La lectura entrega el
// claro sólo dentro del callback, que no debe conservarlo.
type CifradorDatosContactoParticipacion interface {
	CifrarDatosContactoParticipacion(context.Context, string, uint64, []byte) (SobreDatosContacto, error)
	ConDatosContactoParticipacionDescifrados(context.Context, string, SobreDatosContacto, func([]byte) error) error
}

// RegistroDatosContactoParticipacion es el recibo de una versión registrada.
type RegistroDatosContactoParticipacion struct {
	Reutilizada      bool
	ReciboRef        string
	ParticipacionRef string
	Version          uint64
	Motivo           string
	RegistradaEn     time.Time
	Sobre            SobreDatosContacto
	// Origen solo existe en una versión de origen CONVOCA (duda 45).
	Origen *dominiobolsa.MarcaOrigenDatosContacto
}

// DatosContactoParticipacionLeidos es la lectura autorizada para RRHH: claro
// (para contactar) y enmascarado (para listar).
type DatosContactoParticipacionLeidos struct {
	ParticipacionRef string
	Version          uint64
	RegistradaEn     time.Time
	Datos            dominiobolsa.DatosContactoParticipacion
	Enmascarados     dominiobolsa.DatosContactoEnmascarados
	// Origen y EstadoOrigen (vigente o vencido a la hora de la consulta) solo
	// existen si la versión vigente es de origen CONVOCA.
	Origen       *dominiobolsa.MarcaOrigenDatosContacto
	EstadoOrigen string
	// AuditoriaRef y DecisionRef solo existen en la consulta completa: son el
	// asiento de la auditoría común y la decisión V3 que se consumieron en la
	// misma transacción que la lectura. La consulta enmascarada no los tiene.
	AuditoriaRef string
	DecisionRef  string
}

type SolicitudRegistrarDatosContactoParticipacion struct {
	Vinculo            dominiovec.VinculoAutenticacionActorV2
	ResultadoContexto  dominiovec.ResultadoContextoActorRegistradoV2
	BolsaRef           string
	ParticipacionRef   string
	Datos              dominiobolsa.DatosContactoParticipacion
	Motivo             string
	ClaveIdempotencia  string
	Correlacion        dominiovec.ReferenciaCorrelacionAutorizacionV2
	MotivoAutorizacion dominiovec.ReferenciaEntradaCatalogo
	// Origen vacío registra un contacto propio; «convoca» lo marca como
	// contacto de origen CONVOCA con la vigencia de la regla b29.
	Origen string
}

func (s SolicitudRegistrarDatosContactoParticipacion) Validar() error {
	if s.ResultadoContexto.Validar() != nil || s.Vinculo.ValidarPara(s.ResultadoContexto) != nil ||
		s.BolsaRef == "" || s.ParticipacionRef == "" || s.Datos.ParticipacionRef != s.ParticipacionRef ||
		s.Motivo == "" || s.ClaveIdempotencia == "" || !dominiobolsa.OrigenDatosContactoAdmitido(s.Origen) ||
		s.Correlacion.Validar() != nil || !dominiovec.ReferenciaMotivoAutorizacionV2Valida(s.MotivoAutorizacion) {
		return ErrDatosContactoParticipacionNoDisponibles
	}
	return nil
}

// SolicitudConsultarDatosContactoParticipacion es la lectura de RRHH desde la
// ficha B5: exige el mismo contexto de unidad y ámbito que la escritura. Sin
// Completo solo se entregan la forma enmascarada y el origen; con Completo el
// caso de uso emite y consume la decisión V3 propia de la consulta, ligada a
// la participación, antes de devolver el claro.
type SolicitudConsultarDatosContactoParticipacion struct {
	ContextoActor      dominiovec.ContextoActor
	Vinculo            dominiovec.VinculoAutenticacionActorV2
	ResultadoContexto  dominiovec.ResultadoContextoActorRegistradoV2
	BolsaRef           string
	ParticipacionRef   string
	Completo           bool
	Correlacion        dominiovec.ReferenciaCorrelacionAutorizacionV2
	MotivoAutorizacion dominiovec.ReferenciaEntradaCatalogo
}

// ValidarCompleta exige lo que necesita la decisión V3 de la consulta completa.
func (s SolicitudConsultarDatosContactoParticipacion) ValidarCompleta() error {
	if !s.Completo || s.ResultadoContexto.Validar() != nil || s.Vinculo.ValidarPara(s.ResultadoContexto) != nil ||
		s.BolsaRef == "" || s.ParticipacionRef == "" || s.ResultadoContexto.Contexto.PersonaRef == "" ||
		s.ContextoActor.PersonaRef != s.ResultadoContexto.Contexto.PersonaRef ||
		s.Correlacion.Validar() != nil || !dominiovec.ReferenciaMotivoAutorizacionV2Valida(s.MotivoAutorizacion) {
		return ErrDatosContactoParticipacionNoDisponibles
	}
	return nil
}

// LecturaDatosContactoAutorizada es la versión vigente leída en la misma
// transacción que consumió la decisión de consulta, con su acuse común.
type LecturaDatosContactoAutorizada struct {
	Registro     RegistroDatosContactoParticipacion
	DecisionRef  string
	AuditoriaRef string
	ConsumidaEn  time.Time
}

type ComandoRegistrarDatosContactoParticipacion struct {
	ParticipacionRef      string
	BolsaRef              string
	Sobre                 SobreDatosContacto
	Motivo                string
	Actor                 string
	RegistradaEn          time.Time
	ClaveIdempotencia     string
	ReciboRef             string
	SolicitudAutorizacion dominiovec.SolicitudAutorizacionLigadaV3
	Decision              dominiovec.DecisionAutorizacionLigadaV3
	Confirmacion          puertosvec.ConfirmacionRegistroConcesionAutorizacionLigadaV3
	Material              puertosvec.ExportacionMaterialConsumoAutorizacionAtestadaV3
	// CamposCambiados nombra, sin valores, los campos que difieren de la
	// versión anterior; alimenta la traza de valores (petición RRHH p.4).
	CamposCambiados []string
	// Origen se registra en la misma transacción que la versión.
	Origen *dominiobolsa.MarcaOrigenDatosContacto
}

// VerificadorPertenenciaParticipacion lo satisface el repositorio de B2.
type VerificadorPertenenciaParticipacion interface {
	ParticipacionPerteneceABolsa(context.Context, string, string) (bool, error)
}

type RepositorioDatosContactoParticipacion interface {
	// DatosContactoVigentes devuelve la última versión registrada.
	DatosContactoVigentes(context.Context, string) (RegistroDatosContactoParticipacion, error)
	// BuscarRegistroDatosContacto recupera por clave de idempotencia.
	BuscarRegistroDatosContacto(context.Context, string, string) (RegistroDatosContactoParticipacion, error)
	RegistrarDatosContacto(context.Context, ComandoRegistrarDatosContactoParticipacion) (RegistroDatosContactoParticipacion, error)
	// ConsultarDatosContactoAutorizados consume el material de la consulta
	// completa y lee la versión vigente en una sola transacción. Llama a
	// entregar antes del COMMIT: si no hay datos, la participación no es de la
	// bolsa, la decisión no vale o entregar falla (p. ej., no se puede
	// descifrar), se revierte y no queda consumo.
	ConsultarDatosContactoAutorizados(ctx context.Context, bolsaRef, participacionRef, actorRef string, material puertosvec.ExportacionMaterialConsumoAutorizacionAtestadaV3, entregar func(LecturaDatosContactoAutorizada) error) (LecturaDatosContactoAutorizada, error)
}
