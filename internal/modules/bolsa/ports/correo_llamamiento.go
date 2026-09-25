package ports

import (
	"context"
	"errors"

	dominiovec "vec-diputacion-granada/internal/vec/domain"
)

// ErrPersonalizacionLlamamientoNoDisponible indica que la fuente de datos de
// las personas no respondió; nunca se sustituye por un texto sin personalizar.
var ErrPersonalizacionLlamamientoNoDisponible = errors.New("bolsa: datos de personalizacion del llamamiento no disponibles")

// DatosPersonalizacionLlamamiento son los datos de una persona de la bolsa que
// el correo puede incluir. No salen del servidor salvo en la vista previa a
// quien ya puede consultar la bolsa.
type DatosPersonalizacionLlamamiento struct {
	Nombre    string
	Apellidos string
	Posicion  int
	Bolsa     string
}

// FuentePersonalizacionLlamamiento resuelve los datos de varias participaciones
// de una misma bolsa en una sola consulta. Una participación ajena a la bolsa
// no aparece en el resultado.
type FuentePersonalizacionLlamamiento interface {
	DatosPersonalizacionLlamamiento(ctx context.Context, bolsaRef string, participaciones []string) (map[string]DatosPersonalizacionLlamamiento, error)
}

// HuellaCuerpoContacto fija, por destinatario, la huella del asunto y del
// cuerpo exactos enviados, para que cada correo personalizado sea verificable
// junto al recibo del llamamiento (migración Bolsa 000025).
type HuellaCuerpoContacto struct {
	ParticipacionRef   string `json:"participacion_ref"`
	HuellaAsuntoSHA256 string `json:"huella_asunto_sha256"`
	HuellaCuerpoSHA256 string `json:"huella_cuerpo_sha256"`
	Caracteres         int    `json:"caracteres"`
}

// RegistroHuellasCuerpoLlamamiento persiste las huellas antes del envío,
// ligado al token de finalización de la reserva.
type RegistroHuellasCuerpoLlamamiento interface {
	RegistrarHuellasCuerpo(ctx context.Context, bolsaRef, clave, actorRef string, tokenFinalizacion []byte, huellas []HuellaCuerpoContacto) error
}

type SolicitudVistaPreviaLlamamiento struct {
	ContextoActor    dominiovec.ContextoActor
	BolsaRef         string
	ParticipacionRef string
	Configuracion    ConfiguracionLlamamiento
}

type VistaPreviaCorreoLlamamiento struct {
	Asunto     string `json:"asunto"`
	Cuerpo     string `json:"cuerpo"`
	Caracteres int    `json:"caracteres"`
	Limite     int    `json:"limite"`
}

type MarcadorCorreoLlamamientoPublico struct {
	Clave string `json:"clave"`
}

// PlantillaCorreoLlamamientoPublica es lo que el asistente necesita del
// catálogo: nunca incluye datos de personas.
type PlantillaCorreoLlamamientoPublica struct {
	PlantillaVersion string                             `json:"plantilla_version"`
	Personalizada    bool                               `json:"personalizada"`
	Limite           int                                `json:"limite"`
	LimiteAsunto     int                                `json:"limite_asunto"`
	Asunto           string                             `json:"asunto"`
	Cuerpo           string                             `json:"cuerpo"`
	Marcadores       []MarcadorCorreoLlamamientoPublico `json:"marcadores"`
}
