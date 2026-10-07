package ports

import (
	"context"
	"errors"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

// ErrExpedienteConsultaFirmasNoEncontrado: expediente o versión inexistente.
// Contrato de los lectores autorizados V2 (consulta y recuperación R5): sólo
// lo devuelven después de confirmar la transacción que consumió la decisión V3
// con su auditoría de consumo. Por eso la auditoría de intentos no añade otro
// registro (adapters/auditoriafirma); un lector que lo devuelva antes de
// consumir dejaría el acceso sin auditar.
var ErrExpedienteConsultaFirmasNoEncontrado = errors.New("contratacion temporal: expediente de firmas no encontrado")

const (
	AccionConsultarFirmasDocumento     = "contratacion_temporal.documento.firmas.consultar"
	AudienciaConsultaFirmasDocumentoV3 = "vec_contratacion_temporal.firmas_documento.consultar.v1"
	TipoRecursoConsultaFirmasDocumento = "expediente_contratacion_temporal"
)

// MaterialConsultaFirmasDocumento identifica la lectura exacta. La organización
// y la identidad se resuelven en la frontera confiable; el cliente solo elige
// el expediente que intenta consultar.
type MaterialConsultaFirmasDocumento struct {
	OrganizacionRef string
	ExpedienteRef   string
}

// CapacidadConsultaFirmasDocumento transporta la autorización V3 opaca.
type CapacidadConsultaFirmasDocumento struct {
	material vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3
}

func TransportarMaterialConsultaFirmasDocumento(m vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3) CapacidadConsultaFirmasDocumento {
	return CapacidadConsultaFirmasDocumento{material: m}
}

func (c CapacidadConsultaFirmasDocumento) ExportarMaterialParaConsumidor() vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3 {
	return c.material
}

type AutorizadorConsultaFirmasDocumento interface {
	AutorizarConsultaFirmasDocumento(context.Context, MaterialConsultaFirmasDocumento) (CapacidadConsultaFirmasDocumento, error)
}

// LectorFirmasDocumentoAutorizadas exige material y autorización ligados a la
// lectura. El adaptador confirma consumo, resultado y auditoría juntos.
type LectorFirmasDocumentoAutorizadas interface {
	ConsultarFirmasAutorizadas(context.Context, MaterialConsultaFirmasDocumento, CapacidadConsultaFirmasDocumento) ([]FirmaRegistrada, error)
}
