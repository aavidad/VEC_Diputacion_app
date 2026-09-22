package ports

import (
	"context"
	"errors"
	"time"

	dominiovec "vec-diputacion-granada/internal/vec/domain"
	puertosvec "vec-diputacion-granada/internal/vec/ports"
)

const (
	AccionEmitirLlamamiento    = "llamamiento.emitir.v1"
	FinalidadEmitirLlamamiento = "gestion_llamamientos_bolsa"
	AudienciaEmitirLlamamiento = "vec_bolsa_llamamientos.llamamiento.emitir.v1"
	TipoRecursoEmision         = "bolsa_constituida"
)

var (
	ErrEmisionLlamamientoNoDisponible = errors.New("bolsa: emision de llamamiento no disponible")
	ErrEmisionLlamamientoInvalida     = errors.New("bolsa: emision de llamamiento invalida")
	ErrEmisionLlamamientoConflicto    = errors.New("bolsa: clave de emision reutilizada con otro comando")
)

type ConfiguracionLlamamiento struct {
	Referencia       string `json:"referencia"`
	Descripcion      string `json:"descripcion"`
	Categoria        string `json:"categoria"`
	Centro           string `json:"centro"`
	Modalidad        string `json:"modalidad"`
	FechaInicio      string `json:"fecha_inicio"`
	Plazo            string `json:"plazo"`
	PlantillaVersion string `json:"plantilla_version"`
	Asunto           string `json:"asunto"`
	Cuerpo           string `json:"cuerpo"`
}

type SolicitudEmitirLlamamiento struct {
	Vinculo            dominiovec.VinculoAutenticacionActorV2
	ResultadoContexto  dominiovec.ResultadoContextoActorRegistradoV2
	BolsaRef           string
	Participaciones    []string
	Configuracion      ConfiguracionLlamamiento
	ClaveIdempotencia  string
	Correlacion        dominiovec.ReferenciaCorrelacionAutorizacionV2
	MotivoAutorizacion dominiovec.ReferenciaEntradaCatalogo
}

type ComandoEmitirLlamamiento struct {
	LlamamientoRef, ReciboRef, BolsaRef, ActorRef, ClaveIdempotencia string
	Participaciones                                                  []string
	Contactos                                                        []ResultadoContactoEmision
	Configuracion                                                    ConfiguracionLlamamiento
	EmitidoEn                                                        time.Time
	SolicitudAutorizacion                                            dominiovec.SolicitudAutorizacionLigadaV3
	Decision                                                         dominiovec.DecisionAutorizacionLigadaV3
	Confirmacion                                                     puertosvec.ConfirmacionRegistroConcesionAutorizacionLigadaV3
	Material                                                         puertosvec.ExportacionMaterialConsumoAutorizacionAtestadaV3
}

type ResultadoContactoEmision struct {
	ParticipacionRef string `json:"participacion_ref"`
	Resultado        string `json:"resultado"`
	ReciboRef        string `json:"recibo_ref"`
}

type EmisionLlamamiento struct {
	LlamamientoRef  string                     `json:"llamamiento_ref"`
	ReciboRef       string                     `json:"recibo_ref"`
	BolsaRef        string                     `json:"bolsa_ref"`
	Estado          string                     `json:"estado"`
	Participaciones []string                   `json:"participaciones"`
	Configuracion   ConfiguracionLlamamiento   `json:"configuracion"`
	EmitidoEn       time.Time                  `json:"emitido_en"`
	Reutilizada     bool                       `json:"reutilizada"`
	Contactos       []ResultadoContactoEmision `json:"contactos,omitempty"`
}

type RepositorioEmisionLlamamiento interface {
	Emitir(context.Context, ComandoEmitirLlamamiento) (EmisionLlamamiento, error)
	Recuperar(context.Context, string, string) (EmisionLlamamiento, error)
}

type FuenteCorreoParticipacion interface {
	CorreoParticipacion(context.Context, string) (string, error)
}

type EmisorCorreoBolsa interface {
	EnviarCorreo(context.Context, string, string, string, string, time.Time) bool
}
