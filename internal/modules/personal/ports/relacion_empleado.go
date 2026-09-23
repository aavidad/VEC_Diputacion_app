package ports

import (
	"context"
	"errors"
	"time"
	personaldomain "vec-diputacion-granada/internal/modules/personal/domain"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

var (
	ErrConsultaRelacionEmpleadoInvalida = errors.New("personal: consulta propia de relacion invalida")
	ErrRelacionEmpleadoNoDisponible     = errors.New("personal: consulta propia de relacion no disponible")
	ErrRelacionEmpleadoAmbigua          = errors.New("personal: consulta propia de relacion supera el limite")
)

type OrdenConsultaRelacionPropia struct {
	Material     personaldomain.MaterialConsultaRelacionPropia
	Autorizacion vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3
}
type ProveedorAutorizacionConsultaRelacionPropia interface {
	AutorizarConsultaRelacionPropia(context.Context, personaldomain.MaterialConsultaRelacionPropia) (vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3, error)
}
type EvidenciaConsultaRelacionPropia struct {
	ReciboRef, DecisionRef, EfectoRef, ConsumoHuellaSHA256, AuditoriaRef string
	ConsultadaEn                                                         time.Time
}
type ResultadoConsultaRelacionPropia struct {
	Relaciones []personaldomain.RelacionEmpleado
	Evidencia  EvidenciaConsultaRelacionPropia
}
type RepositorioRelacionesEmpleado interface {
	ConsultarRelacionesPropiasDietas(context.Context, OrdenConsultaRelacionPropia) (ResultadoConsultaRelacionPropia, error)
}
