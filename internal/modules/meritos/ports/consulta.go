package ports

import (
	"context"
	"errors"
	"time"

	"vec-diputacion-granada/internal/modules/meritos/domain"
	vec "vec-diputacion-granada/internal/vec/domain"
)

var ErrConsultaNoDisponible = errors.New("meritos.error.consulta_no_disponible")

// FichaHechoPropio es la proyección autorizada de la versión actual. No expone
// Persona, declarante ni identidad del revisor; las evidencias son referencias.
type FichaHechoPropio struct {
	Referencia   string                    `json:"referencia"`
	Version      int                       `json:"version"`
	Tipo         string                    `json:"tipo"`
	ConceptoRef  string                    `json:"concepto_ref"`
	Denominacion string                    `json:"denominacion"`
	Horas        *int                      `json:"horas,omitempty"`
	Procedencia  domain.Procedencia        `json:"procedencia"`
	Vigencia     domain.Vigencia           `json:"vigencia"`
	Estado       domain.Estado             `json:"estado"`
	Evidencias   []vec.ReferenciaDocumento `json:"evidencias"`
	Revision     *RevisionConsultaPropia   `json:"revision,omitempty"`
}

type RevisionConsultaPropia struct {
	Referencia string `json:"referencia"`
	MotivoRef  string `json:"motivo_ref"`
	Fecha      string `json:"fecha"`
}

type ReciboConsultaPropia struct {
	Referencia          string    `json:"referencia"`
	HechoRef            string    `json:"hecho_ref"`
	VersionConsultada   int       `json:"version_consultada"`
	DecisionRef         string    `json:"decision_ref"`
	ConsumoHuellaSHA256 string    `json:"consumo_huella_sha256"`
	AuditoriaRef        string    `json:"auditoria_ref"`
	CorrelacionRef      string    `json:"correlacion_ref"`
	ConsultadaEn        time.Time `json:"consultada_en"`
}

type ResultadoConsultaPropia struct {
	Codigo         string                `json:"codigo"`
	HechoActual    *FichaHechoPropio     `json:"hecho_actual"`
	ReciboConsulta *ReciboConsultaPropia `json:"recibo_consulta"`
}

// OrdenConsultaPropia contiene un selector construido en aplicación y material
// del emisor común. PersonaRef nunca se recibe de un cliente de transporte.
type OrdenConsultaPropia struct {
	SelectorCanonico     []byte
	HechoRef             string
	PersonaRef           string
	HuellaConsultaSHA256 string
	Motivo               vec.ReferenciaEntradaCatalogo
	Autorizacion         AutorizacionOperacion
}

// RepositorioConsultaPropia consume/revalida V3, selecciona la versión actual
// de la persona atestada y persiste auditoría y recibo en una transacción. No
// modifica el hecho ni crea historia/outbox de negocio. Tanto obtenida como
// no_encontrada llevan recibo; una denegación no contiene ficha ni recibo.
// Coteja el resultado antes de COMMIT y solo lo devuelve tras confirmarlo.
// Un fallo o confirmación incierta devuelve resultado vacío y error.
type RepositorioConsultaPropia interface {
	ConsultarActual(context.Context, OrdenConsultaPropia) (ResultadoConsultaPropia, error)
}
