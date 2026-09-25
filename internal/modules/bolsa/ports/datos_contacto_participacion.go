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
}

// DatosContactoParticipacionLeidos es la lectura autorizada para RRHH: claro
// (para contactar) y enmascarado (para listar).
type DatosContactoParticipacionLeidos struct {
	ParticipacionRef string
	Version          uint64
	RegistradaEn     time.Time
	Datos            dominiobolsa.DatosContactoParticipacion
	Enmascarados     dominiobolsa.DatosContactoEnmascarados
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
}

func (s SolicitudRegistrarDatosContactoParticipacion) Validar() error {
	if s.ResultadoContexto.Validar() != nil || s.Vinculo.ValidarPara(s.ResultadoContexto) != nil ||
		s.BolsaRef == "" || s.ParticipacionRef == "" || s.Datos.ParticipacionRef != s.ParticipacionRef ||
		s.Motivo == "" || s.ClaveIdempotencia == "" ||
		s.Correlacion.Validar() != nil || !dominiovec.ReferenciaMotivoAutorizacionV2Valida(s.MotivoAutorizacion) {
		return ErrDatosContactoParticipacionNoDisponibles
	}
	return nil
}

// SolicitudConsultarDatosContactoParticipacion es la lectura de RRHH desde la
// ficha B5: exige el mismo contexto de unidad y ámbito que la escritura.
type SolicitudConsultarDatosContactoParticipacion struct {
	ContextoActor    dominiovec.ContextoActor
	BolsaRef         string
	ParticipacionRef string
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
}
