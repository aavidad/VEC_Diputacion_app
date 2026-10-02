package ports

import (
	"context"
	"errors"
	"time"

	bolsadomain "vec-diputacion-granada/internal/modules/bolsa/domain"
	bolsaports "vec-diputacion-granada/internal/modules/bolsa/ports"
	vecdomain "vec-diputacion-granada/internal/vec/domain"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

var (
	ErrConsultaConvocatoriaInvalida  = errors.New("seleccion_consulta_convocatoria_invalida")
	ErrConsultaConvocatoriaDenegada  = errors.New("seleccion_consulta_convocatoria_denegada")
	ErrConvocatoriaNoEncontrada      = errors.New("seleccion_convocatoria_no_encontrada")
	ErrConvocatoriaNoDisponible      = errors.New("seleccion_convocatoria_no_disponible")
	ErrRespuestaConvocatoriaInvalida = errors.New("seleccion_respuesta_convocatoria_invalida")
)

// SolicitudConsultaConvocatoria es interna: el servidor aporta Actor y
// Correlacion. Un ContextoActor válido no constituye una concesión.
type SolicitudConsultaConvocatoria struct {
	Selector    bolsaports.SelectorVersionConvocatoriaExacta
	Actor       vecdomain.ContextoActor                       `json:"-"`
	Correlacion vecdomain.ReferenciaCorrelacionAutorizacionV2 `json:"-"`
}

// FichaConvocatoria conserva referencias exactas del propietario de las bases.
// Requisitos contiene el texto original, sin evaluar acceso. El flujo fijado
// no permite deducir sus fases ni las referencias de OEP, plaza, RPT o plantilla:
// esos datos quedan pendientes hasta disponer de sus contratos y fuentes.
type FichaConvocatoria struct {
	ConvocatoriaID             string                                               `json:"convocatoria_id"`
	Secuencia                  int                                                  `json:"secuencia"`
	Revision                   int                                                  `json:"revision"`
	FuenteRef                  string                                               `json:"fuente_ref"`
	HuellaVersionSHA256        string                                               `json:"huella_version_sha256"`
	Bases                      []bolsadomain.ReferenciaDocumentoOficialConvocatoria `json:"bases"`
	Requisitos                 []bolsadomain.RequisitoConvocatoria                  `json:"requisitos"`
	FlujoProceso               bolsadomain.ReferenciaConfiguracionConvocatoria      `json:"flujo_proceso"`
	ReglasBaremacion           bolsadomain.ReferenciaConfiguracionConvocatoria      `json:"reglas_baremacion"`
	FasesEstado                string                                               `json:"fases_estado"`
	ReferenciasCoberturaEstado string                                               `json:"referencias_cobertura_estado"`
}

// EvidenciaLecturaConvocatoria contiene coordenadas mínimas de la lectura;
// comprobar su forma no acredita la concesión ni sustituye el consumo V3.
type EvidenciaLecturaConvocatoria struct {
	ReciboRef           string
	DecisionRef         string
	ConsumoHuellaSHA256 string
	AuditoriaRef        string
	CorrelacionRef      string
	ConsultadaEn        time.Time
}

type LecturaConvocatoria struct {
	Ficha     FichaConvocatoria
	Evidencia EvidenciaLecturaConvocatoria
}

// LectorConvocatoriaExacta pertenece al propietario Bolsa. Cada invocación
// exige una concesión central V3 positiva, exacta y vigente para la solicitud,
// y consume esa autorización junto con la lectura y auditoría en la transacción.
// También audita las denegaciones. No adapta capacidades V1 ni devuelve datos
// parciales ante denegación o indisponibilidad. La composición real solo puede
// inyectar un lector que cumpla este contrato, nunca una fuente de ensayo.
type LectorConvocatoriaExacta interface {
	ConsultarExacta(context.Context, SolicitudConsultaConvocatoria) (LecturaConvocatoria, error)
}

// Los dos puertos conservan la separación entre decisión central y lectura.
// El repositorio vuelve a validar y consumir V3 en su transacción nominal.
type ProveedorAutorizacionConsultaConvocatoria interface {
	AutorizarConsultaConvocatoria(context.Context, SolicitudConsultaConvocatoria) (vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3, error)
}

type OrdenConsultaConvocatoria struct {
	Solicitud    SolicitudConsultaConvocatoria
	Autorizacion vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3
}

type ResultadoVersionConvocatoria struct {
	Estado              string
	Version             bolsadomain.VersionConvocatoriaGobernada
	HuellaVersionSHA256 string
	Evidencia           EvidenciaLecturaConvocatoria
}

type RepositorioConsultaConvocatoria interface {
	ObtenerVersionExactaV3(context.Context, OrdenConsultaConvocatoria) (ResultadoVersionConvocatoria, error)
}
