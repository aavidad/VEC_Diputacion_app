package ports

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	vd "vec-diputacion-granada/internal/vec/domain"
	vp "vec-diputacion-granada/internal/vec/ports"
)

const (
	AccionComprobacionOfrecimiento    = "bolsa.comprobacion_ofrecimiento.registrar"
	AudienciaComprobacionOfrecimiento = "vec.bolsa.comprobacion_ofrecimiento.v1"
	FinalidadComprobacionOfrecimiento = "gestionar_contratacion_temporal"
)

var (
	ErrComprobacionInvalida              = errors.New("bolsa: comprobacion invalida")
	ErrComprobacionConflicto             = errors.New("bolsa: comprobacion en conflicto")
	ErrComprobacionDependencia           = errors.New("bolsa: comprobacion no disponible")
	ErrComprobacionResultadoNoConfirmado = errors.New("bolsa: resultado de comprobacion no confirmado")
)

type ContextoComprobacion struct {
	AmbitoRef                   string `json:"ambito_ref"`
	BolsaRef                    string `json:"bolsa_ref"`
	CategoriaRef                string `json:"categoria_ref"`
	CausaOfrecida               string `json:"causa_ofrecida"`
	ExpedienteRef               string `json:"expediente_ref"`
	HuellaPoliticaOfertasSHA256 string `json:"huella_politica_ofertas_sha256"`
	ModalidadOfrecida           string `json:"modalidad_ofrecida"`
	NecesidadRef                string `json:"necesidad_ref"`
	OrdenActa                   int64  `json:"orden_acta"`
	OrganizacionRef             string `json:"organizacion_ref"`
	ParticipacionRef            string `json:"participacion_ref"`
	UnidadRef                   string `json:"unidad_ref"`
	VersionBolsa                int64  `json:"version_bolsa"`
	VersionExpediente           int64  `json:"version_expediente"`
	VersionInstantanea          int64  `json:"version_instantanea"`
	VersionPoliticaOfertas      int64  `json:"version_politica_ofertas"`
}

type HechoComprobacion struct {
	Valor        json.RawMessage `json:"valor"`
	DocumentoRef string          `json:"documento_ref"`
	Huella       string          `json:"huella"`
}

type SolicitudComprobacion struct {
	Vinculo            vd.VinculoAutenticacionActorV2         `json:"-"`
	ResultadoContexto  vd.ResultadoContextoActorRegistradoV2  `json:"-"`
	Correlacion        vd.ReferenciaCorrelacionAutorizacionV2 `json:"-"`
	MotivoAutorizacion vd.ReferenciaEntradaCatalogo           `json:"-"`
	Contexto           ContextoComprobacion                   `json:"contexto"`
	Hechos             map[string]HechoComprobacion           `json:"hechos"`
	VersionEsperada    int64                                  `json:"version_esperada"`
	ClaveIdempotencia  string                                 `json:"clave_idempotencia"`
}

type ReciboComprobacion struct {
	ComprobacionRef        string    `json:"comprobacion_ref"`
	Version                int64     `json:"version"`
	VersionEsperada        int64     `json:"version_esperada"`
	Resultado              string    `json:"resultado"`
	MotivoClave            string    `json:"motivo_clave"`
	VersionPoliticaOfertas int64     `json:"version_politica_ofertas"`
	ReciboRef              string    `json:"recibo_ref"`
	EventoRef              string    `json:"evento_ref"`
	AuditoriaRef           string    `json:"auditoria_ref"`
	RegistradaEn           time.Time `json:"registrada_en"`
	Reutilizada            bool      `json:"reutilizada"`
}

type IntencionComprobacion struct {
	Solicitud       SolicitudComprobacion
	CanonPostgreSQL string
	HuellaSHA256    string
	Material        vp.ExportacionMaterialConsumoAutorizacionAtestadaV3
}

type RepositorioComprobacion interface {
	NormalizarComprobacion(context.Context, SolicitudComprobacion) (string, string, error)
	RegistrarComprobacion(context.Context, IntencionComprobacion) (ReciboComprobacion, error)
}

type ServicioComprobacion interface {
	Registrar(context.Context, SolicitudComprobacion) (ReciboComprobacion, error)
}
