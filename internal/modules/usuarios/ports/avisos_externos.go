package ports

import (
	"context"
	"errors"
)

var (
	ErrAvisoExternoInvalido     = errors.New("usuarios: aviso externo invalido")
	ErrAvisoExternoConflicto    = errors.New("usuarios: aviso externo conflicto")
	ErrAvisoExternoNoDisponible = errors.New("usuarios: aviso externo no disponible")
)

const TipoAvisoLlamamientoExternoV1 = "vec.bolsa.aviso-llamamiento.v1"

// EventoAvisoExterno es el contrato cerrado del inbox. El orden de los campos
// forma parte de la preimagen SHA256; el campo opcional se serializa vacío.
type EventoAvisoExterno struct {
	EventoRef              string `json:"evento_ref"`
	ProductorRef           string `json:"productor_ref"`
	TipoVersionado         string `json:"tipo_versionado"`
	OcurridoEn             string `json:"ocurrido_en"`
	CorrelacionRef         string `json:"correlacion_ref"`
	DestinatarioExternoRef string `json:"destinatario_externo_ref"`
	ComunicacionRef        string `json:"comunicacion_ref"`
	PlantillaRef           string `json:"plantilla_ref"`
	PlantillaVersion       string `json:"plantilla_version"`
	RecursoPublicoRef      string `json:"recurso_publico_ref"`
}

type ReciboAvisoExterno struct {
	ReciboRef  string `json:"recibo_ref"`
	Huella     string `json:"huella"`
	AceptadoEn string `json:"aceptado_en"`
	Replay     bool   `json:"replay"`
}

// ReservaAvisoExterno sólo circula dentro de Usuarios. Un replay no contiene
// token, persona ni sobre. Estado indica el resultado durable conocido.
type ReservaAvisoExterno struct {
	Recibo     ReciboAvisoExterno
	Evento     EventoAvisoExterno
	ReservaRef string
	PersonaRef string
	Sobre      SobreDireccionCorreo
	Estado     string
	Replay     bool
}

func (ReservaAvisoExterno) String() string   { return "usuarios.ReservaAvisoExterno{redactado}" }
func (ReservaAvisoExterno) GoString() string { return "usuarios.ReservaAvisoExterno{redactado}" }

type ResultadoDespachoAvisoExterno struct {
	ReciboRef string
	Estado    string
	Replay    bool
}

type RegistroAvisosExternos interface {
	AceptarAvisoExterno(context.Context, []byte) (ReciboAvisoExterno, error)
	ReservarAvisoExterno(context.Context, string) (ReservaAvisoExterno, error)
	ConfirmarAvisoExterno(context.Context, string, string, string) error
}

// La composición adapta el catálogo gobernado existente; esta interfaz no
// crea una segunda autoridad de plantillas ni admite texto libre del evento.
type CatalogoPlantillasAvisoExterno interface {
	AdmitePlantillaAvisoExterno(context.Context, string, string, string) bool
}

type MensajeAvisoExterno struct {
	EnvioRef string
	Destino  string
	Evento   EventoAvisoExterno
}

func (MensajeAvisoExterno) String() string   { return "usuarios.MensajeAvisoExterno{redactado}" }
func (MensajeAvisoExterno) GoString() string { return "usuarios.MensajeAvisoExterno{redactado}" }

// true acredita únicamente aceptación del relay SMTP; una respuesta incierta
// se conserva como no_aceptado, sin reintento automático de la reserva.
type TransportadorAvisoExterno interface {
	EnviarAvisoExterno(context.Context, MensajeAvisoExterno) bool
}
