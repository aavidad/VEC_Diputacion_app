package ports

import "context"

// EventoAvisoExterno contiene únicamente referencias gobernadas. La dirección
// y el material criptográfico pertenecen al receptor, nunca al productor.
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

type AvisoExternoPendiente struct {
	Evento          EventoAvisoExterno `json:"evento"`
	Huella          string             `json:"huella"`
	ReciboOutboxRef string             `json:"recibo_outbox_ref,omitempty"`
	EstadoDespacho  string             `json:"estado_despacho,omitempty"`
}

type RepositorioAvisosExternos interface {
	Extraer(context.Context, int) ([]AvisoExternoPendiente, error)
	ConfirmarAceptacion(context.Context, string, string, string, string) error
	RegistrarResultadoDespacho(context.Context, string, string, string, string, string) error
}

// FuenteDestinatarioExternoParticipacion consulta la autoridad del vínculo
// externo. Un vínculo ausente, vencido o revocado devuelve referencia vacía.
type FuenteDestinatarioExternoParticipacion interface {
	DestinatarioExterno(context.Context, string, string) (string, error)
}
