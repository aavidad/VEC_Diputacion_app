package ports

import (
	"context"
	"time"
)

// AvisosOferta es el resultado agregado del despacho; enviado solo acredita
// aceptación por el relay, nunca entrega ni notificación administrativa.
type AvisosOferta struct {
	Revision        *int64 `json:"revision,omitempty"`
	Total           int    `json:"total"`
	Pendientes      int    `json:"pendientes"`
	Enviados        int    `json:"enviados"`
	SinConfirmacion int    `json:"sin_confirmacion"`
	Fallidos        int    `json:"fallidos"`
}

func (a AvisosOferta) Validar() bool {
	return (a.Revision == nil || *a.Revision >= 0 && *a.Revision <= 9007199254740991) &&
		a.Total >= 0 && a.Pendientes >= 0 && a.Enviados >= 0 && a.SinConfirmacion >= 0 && a.Fallidos >= 0 &&
		a.Total <= 1000000 && a.Pendientes <= a.Total && a.Enviados <= a.Total && a.SinConfirmacion <= a.Total && a.Fallidos <= a.Total && a.Total == a.Pendientes+a.Enviados+a.SinConfirmacion+a.Fallidos
}

// ReintentoAvisoOferta es el recibo del acto expreso. La oferta conserva su
// recibo y su plazo originales.
type ReintentoAvisoOferta struct {
	ReciboRef    string    `json:"recibo_ref"`
	RegistradoEn time.Time `json:"registrado_en"`
	Rearmados    int       `json:"rearmados"`
}

type IntentoAvisoOferta struct {
	ParticipacionRef string
	MessageID        string
	Intento          int
}

type RepositorioAvisosOferta interface {
	ReclamarAvisoOferta(context.Context, OfertaPublicada) (IntentoAvisoOferta, bool, error)
	FinalizarAvisoOferta(context.Context, OfertaPublicada, IntentoAvisoOferta, string) (OfertaPublicada, error)
}

// El token nunca sale del proceso ni forma parte del DTO HTTP. La versión
// persistida permite recuperar una publicación después de rotar el material.
type DerivadorTokenAvisoOferta interface {
	DerivarTokenAvisoOferta(context.Context, string, string) ([]byte, string, error)
}

type MensajeAvisoOferta struct {
	Destino, Asunto, Cuerpo, MessageID string
	Fecha                              time.Time
}

const (
	AvisoOfertaAceptado      = "aceptado"
	AvisoOfertaTransitorio   = "transitorio"
	AvisoOfertaPermanente    = "permanente"
	AvisoOfertaIndeterminado = "indeterminado"
)

type EmisorAvisoOferta interface {
	EnviarAvisoOferta(context.Context, MensajeAvisoOferta) string
}
