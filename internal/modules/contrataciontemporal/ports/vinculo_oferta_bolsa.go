package ports

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
)

const (
	// La concesión, el recurso y la audiencia son propios de la oferta.
	// CT201 singular conserva íntegro su contrato anterior.
	AccionVincularOfertaBolsaCT     = "contratacion_temporal.bolsa.oferta.vincular"
	TipoRecursoVinculoOfertaBolsaCT = "vinculo_oferta_bolsa_expediente"
	AudienciaVincularOfertaBolsaCT  = "vec_contratacion_temporal.vinculo_oferta_bolsa.v1"
	FinalidadVincularOfertaBolsaCT  = "tramitacion_expediente_contratacion_temporal"
	AccionDescartarOfertaBolsaCT    = "contratacion_temporal.bolsa.oferta.descartar"
	TipoRecursoDescarteOfertaCT     = "seleccion_oferta_expediente"
	AudienciaDescartarOfertaCT      = "vec_contratacion_temporal.seleccion_oferta.descartar.v1"
	FinalidadDescartarOfertaCT      = FinalidadVincularOfertaBolsaCT
	EsquemaCierreOfertaBolsaCTV1    = "vec.bolsa.oferta-ct.cierre.v1"
	EstadoCierreOfertaBolsaCTV1     = "cerrada_sin_publicar"
	EsquemaReciboDescarteOfertaCTV1 = "vec.contratacion-temporal.recibo-descarte-oferta.v1"
	EstadoReciboDescarteOfertaCTV1  = "descartada"
	MotivoDescarteOfertaCTV1        = "origen_caducado"
)

var ErrVinculoOfertaBolsaCTInvalido = errors.New("contratacion temporal: vinculo de oferta de bolsa invalido")
var ErrCierreOfertaBolsaCTInvalido = errors.New("contratacion temporal: cierre de intencion de oferta invalido")

// OfertaPublicadaSeleccionCT minimiza el recibo autenticado de Bolsa. La
// composición verifica su correspondencia con el comando congelado y Bolsa
// vuelve a acreditar el origen ante CT mediante su función nominal B120.
type OfertaPublicadaSeleccionCT struct {
	ClaveBolsa      string                 `json:"clave_bolsa"`
	BolsaRef        string                 `json:"bolsa_ref"`
	OfertaRef       string                 `json:"oferta_ref"`
	ReciboOfertaRef string                 `json:"recibo_oferta_ref"`
	NumeroPlazas    int                    `json:"numero_plazas"`
	Datos           DatosOfertaSeleccionCT `json:"datos"`
	PublicadaEn     time.Time              `json:"publicada_en"`
	VenceAntesDe    time.Time              `json:"vence_antes_de"`
}

func (o OfertaPublicadaSeleccionCT) ValidarPara(c ComandoOfertaBolsaCT) error {
	if o.ClaveBolsa != c.ClaveBolsa || o.BolsaRef != c.BolsaRef ||
		!domain.ReferenciaOpacaValida(o.OfertaRef) ||
		!domain.ReferenciaOpacaValida(o.ReciboOfertaRef) ||
		o.NumeroPlazas != c.NumeroPlazas || o.Datos != c.Datos ||
		!domain.InstanteUTCCanonico(o.PublicadaEn) ||
		!domain.InstanteUTCCanonico(o.VenceAntesDe) ||
		!o.VenceAntesDe.After(o.PublicadaEn) {
		return ErrVinculoOfertaBolsaCTInvalido
	}
	return nil
}

// El publicador recibe siempre el comando original y una identidad
// revalidada; emite material V3 nuevo de Bolsa en cada llamada.
type PublicadorOfertaBolsaCT interface {
	PublicarORecuperar(context.Context, IntencionSeleccionOfertaCT, ComandoOfertaBolsaCT) (OfertaPublicadaSeleccionCT, error)
}

// ReciboSeleccionOfertaCT prueba el vínculo propio de CT con la oferta real.
// El recibo permanece estable al consultar una operación terminal histórica.
type ReciboSeleccionOfertaCT struct {
	Esquema           string    `json:"esquema"`
	Estado            string    `json:"estado"`
	OrganizacionRef   string    `json:"organizacion_ref"`
	ExpedienteRef     string    `json:"expediente_ref"`
	VersionExpediente uint64    `json:"version_expediente"`
	BolsaRef          string    `json:"bolsa_ref"`
	OfertaRef         string    `json:"oferta_ref"`
	ReciboOfertaRef   string    `json:"recibo_oferta_ref"`
	ReciboVinculoRef  string    `json:"recibo_vinculo_ref"`
	NumeroPlazas      int       `json:"numero_plazas"`
	PublicadaEn       time.Time `json:"publicada_en"`
	VenceAntesDe      time.Time `json:"vence_antes_de"`
}

func (r ReciboSeleccionOfertaCT) ValidarPara(i IntencionSeleccionOfertaCT, c ComandoOfertaBolsaCT) error {
	if i.Validar() != nil || c.ValidarPara(i) != nil ||
		r.Esquema != EsquemaReciboSeleccionOfertaCTV1 ||
		r.Estado != EstadoReciboSeleccionOfertaCTV1 ||
		r.OrganizacionRef != i.OrganizacionRef || r.ExpedienteRef != i.ExpedienteRef ||
		r.VersionExpediente != i.VersionExpediente || r.BolsaRef != c.BolsaRef ||
		!domain.ReferenciaOpacaValida(r.OfertaRef) ||
		!domain.ReferenciaOpacaValida(r.ReciboOfertaRef) ||
		!domain.ReferenciaOpacaValida(r.ReciboVinculoRef) ||
		r.NumeroPlazas != c.NumeroPlazas ||
		!domain.InstanteUTCCanonico(r.PublicadaEn) ||
		!domain.InstanteUTCCanonico(r.VenceAntesDe) ||
		!r.VenceAntesDe.After(r.PublicadaEn) {
		return ErrVinculoOfertaBolsaCTInvalido
	}
	return nil
}

// ReciboCierreOfertaCT acredita que Bolsa cerró la intención original antes
// de crear oferta. Las huellas de JSONB las verifica la fachada nominal B121;
// Go comprueba identidad, formato y correspondencia con el comando congelado.
type ReciboCierreOfertaCT struct {
	Esquema           string    `json:"esquema"`
	Estado            string    `json:"estado"`
	OrganizacionRef   string    `json:"organizacion_ref"`
	ExpedienteRef     string    `json:"expediente_ref"`
	VersionExpediente uint64    `json:"version_expediente"`
	ClaveCT           string    `json:"clave_ct"`
	ActorRef          string    `json:"actor_ref"`
	BolsaRef          string    `json:"bolsa_ref"`
	ClaveBolsa        string    `json:"clave_bolsa"`
	NumeroPlazas      int       `json:"numero_plazas"`
	DatosSHA256       string    `json:"datos_sha256"`
	OrigenSHA256      string    `json:"origen_sha256"`
	ComandoSHA256     string    `json:"comando_sha256"`
	ReciboCierreRef   string    `json:"recibo_cierre_ref"`
	DecisionRef       string    `json:"decision_ref"`
	AuditoriaRef      string    `json:"auditoria_ref"`
	EventoRef         string    `json:"evento_ref"`
	CerradaEn         time.Time `json:"cerrada_en"`
}

// Bolsa guarda el recibo JSONB con seis decimales exactos. El cotejo nominal
// de CT usa ese mismo JSONB, por lo que no puede depender del formato variable
// RFC3339Nano que usa encoding/json para time.Time.
func (r ReciboCierreOfertaCT) MarshalJSON() ([]byte, error) {
	if !domain.InstanteUTCCanonico(r.CerradaEn) {
		return nil, ErrCierreOfertaBolsaCTInvalido
	}
	type reciboSinMetodo ReciboCierreOfertaCT
	return json.Marshal(struct {
		reciboSinMetodo
		CerradaEn string `json:"cerrada_en"`
	}{reciboSinMetodo(r), r.CerradaEn.Format("2006-01-02T15:04:05.000000Z")})
}

func (r ReciboCierreOfertaCT) ValidarPara(i IntencionSeleccionOfertaCT, c ComandoOfertaBolsaCT) error {
	if i.Validar() != nil || c.ValidarPara(i) != nil ||
		r.Esquema != EsquemaCierreOfertaBolsaCTV1 || r.Estado != EstadoCierreOfertaBolsaCTV1 ||
		r.OrganizacionRef != i.OrganizacionRef || r.ExpedienteRef != i.ExpedienteRef ||
		r.VersionExpediente != i.VersionExpediente || r.ClaveCT != i.ClaveIdempotencia ||
		r.ActorRef != i.ActorRef || r.BolsaRef != c.BolsaRef || r.ClaveBolsa != c.ClaveBolsa ||
		r.NumeroPlazas != c.NumeroPlazas || !huellaSeleccionOfertaCT.MatchString(r.DatosSHA256) ||
		!huellaSeleccionOfertaCT.MatchString(r.OrigenSHA256) ||
		!huellaSeleccionOfertaCT.MatchString(r.ComandoSHA256) ||
		!domain.ReferenciaOpacaValida(r.ReciboCierreRef) ||
		!domain.ReferenciaOpacaValida(r.DecisionRef) ||
		!domain.ReferenciaOpacaValida(r.AuditoriaRef) ||
		!domain.ReferenciaOpacaValida(r.EventoRef) ||
		!domain.InstanteUTCCanonico(r.CerradaEn) {
		return ErrCierreOfertaBolsaCTInvalido
	}
	return nil
}

type SituacionCierreOfertaBolsaCT string

const (
	CierreOfertaPublicada SituacionCierreOfertaBolsaCT = "publicada"
	CierreOfertaCerrada   SituacionCierreOfertaBolsaCT = "cerrada"
)

type ResultadoCierreOfertaBolsaCT struct {
	Situacion SituacionCierreOfertaBolsaCT
	Oferta    *OfertaPublicadaSeleccionCT
	Cierre    *ReciboCierreOfertaCT
}

func (r ResultadoCierreOfertaBolsaCT) ValidarPara(i IntencionSeleccionOfertaCT, c ComandoOfertaBolsaCT) error {
	switch r.Situacion {
	case CierreOfertaPublicada:
		if r.Oferta == nil || r.Cierre != nil || r.Oferta.ValidarPara(c) != nil {
			return ErrCierreOfertaBolsaCTInvalido
		}
	case CierreOfertaCerrada:
		if r.Cierre == nil || r.Oferta != nil || r.Cierre.ValidarPara(i, c) != nil {
			return ErrCierreOfertaBolsaCTInvalido
		}
	default:
		return ErrCierreOfertaBolsaCTInvalido
	}
	return nil
}

// Resolver debe usar el perfil de cierre de Bolsa y V3 fresca. Devuelve
// exactamente una publicación acreditada o el recibo durable de cierre B121.
type ResolutorCierreOfertaBolsaCT interface {
	Resolver(context.Context, IntencionSeleccionOfertaCT, ComandoOfertaBolsaCT) (ResultadoCierreOfertaBolsaCT, error)
}

// ReciboDescarteOfertaCT confirma el terminal CT después de verificar B121
// en la misma transacción. El comando original queda en la historia.
type ReciboDescarteOfertaCT struct {
	Esquema           string    `json:"esquema"`
	Estado            string    `json:"estado"`
	OrganizacionRef   string    `json:"organizacion_ref"`
	ExpedienteRef     string    `json:"expediente_ref"`
	VersionExpediente uint64    `json:"version_expediente"`
	BolsaRef          string    `json:"bolsa_ref"`
	ClaveBolsa        string    `json:"clave_bolsa"`
	ReciboCierreRef   string    `json:"recibo_cierre_ref"`
	ReciboDescarteRef string    `json:"recibo_descarte_ref"`
	Motivo            string    `json:"motivo"`
	DescartadaEn      time.Time `json:"descartada_en"`
	AuditoriaRef      string    `json:"auditoria_ref"`
	EventoRef         string    `json:"evento_ref"`
}

func (r ReciboDescarteOfertaCT) ValidarTerminalPara(i IntencionSeleccionOfertaCT, c ComandoOfertaBolsaCT) error {
	if i.Validar() != nil || c.ValidarPara(i) != nil ||
		r.Esquema != EsquemaReciboDescarteOfertaCTV1 || r.Estado != EstadoReciboDescarteOfertaCTV1 ||
		r.OrganizacionRef != i.OrganizacionRef || r.ExpedienteRef != i.ExpedienteRef ||
		r.VersionExpediente != i.VersionExpediente || r.BolsaRef != c.BolsaRef ||
		r.ClaveBolsa != c.ClaveBolsa || !domain.ReferenciaOpacaValida(r.ReciboCierreRef) ||
		!domain.ReferenciaOpacaValida(r.ReciboDescarteRef) || r.Motivo != MotivoDescarteOfertaCTV1 ||
		!domain.InstanteUTCCanonico(r.DescartadaEn) ||
		!domain.ReferenciaOpacaValida(r.AuditoriaRef) || !domain.ReferenciaOpacaValida(r.EventoRef) {
		return ErrCierreOfertaBolsaCTInvalido
	}
	return nil
}

func (r ReciboDescarteOfertaCT) ValidarPara(i IntencionSeleccionOfertaCT, c ComandoOfertaBolsaCT, cierre ReciboCierreOfertaCT) error {
	if r.ValidarTerminalPara(i, c) != nil || cierre.ValidarPara(i, c) != nil ||
		r.ReciboCierreRef != cierre.ReciboCierreRef || r.DescartadaEn.Before(cierre.CerradaEn) {
		return ErrCierreOfertaBolsaCTInvalido
	}
	return nil
}
