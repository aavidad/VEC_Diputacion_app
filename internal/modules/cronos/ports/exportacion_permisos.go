package ports

import (
	"context"
	"errors"
	"time"

	"vec-diputacion-granada/internal/modules/cronos/domain"
	vecdomain "vec-diputacion-granada/internal/vec/domain"
)

var (
	ErrExportacionPermisosNoDisponible = errors.New("cronos_exportacion_permisos_no_disponible")
	ErrExportacionPermisosInvalida     = errors.New("cronos_exportacion_permisos_invalida")
)

const AccionExportarPermisosPDF = "exportar_pdf"
const (
	ConciliacionPermisosConfirmada = "conciliado"
	ConciliacionPermisosPendiente  = "sin_conciliar"
)

// OrdenExportacionPermisos se construye con el contexto servidor; no acredita
// autorización. El ejercicio y la acción quedan fijados antes de la lectura.
type OrdenExportacionPermisos struct {
	actor     vecdomain.ContextoActor
	ejercicio int
}

func NuevaOrdenExportacionPermisos(actor vecdomain.ContextoActor, ejercicio int) (OrdenExportacionPermisos, error) {
	copia, err := actor.Clonar()
	refs, errRefs := actor.Referencias(vecdomain.TipoReferenciaContextoActorEmpleado)
	if err != nil || errRefs != nil || len(refs) != 1 || ejercicio < 1 || ejercicio > 9999 {
		return OrdenExportacionPermisos{}, ErrExportacionPermisosInvalida
	}
	return OrdenExportacionPermisos{actor: copia, ejercicio: ejercicio}, nil
}

func (o OrdenExportacionPermisos) ContextoActor() (vecdomain.ContextoActor, error) {
	return o.actor.Clonar()
}
func (o OrdenExportacionPermisos) Ejercicio() int { return o.ejercicio }
func (o OrdenExportacionPermisos) Accion() string { return AccionExportarPermisosPDF }

// Política nominal de la MISMA instantánea durable que la proyección. No se
// obtiene de catálogos de textos, argumentos CLI ni nombres de los permisos.
type PoliticaExportacionPermisos struct {
	Referencia       string
	Version          int64
	SHA256           string
	TiposPermitidos  []string
	CamposPermitidos []string
}

// TipoRef sólo permite comprobar la lista positiva; se elimina antes de renderizar.
type FilaPermisoExportable struct {
	TipoRef string
	Resumen FilaInformePermisos
}

type PermisosExportables struct {
	EmpleadoRef   string
	Ejercicio     int
	CorteUTC      time.Time
	FuenteRef     string
	FuenteVersion int64
	FuenteSHA256  string
	Politica      PoliticaExportacionPermisos
	Filas         []FilaPermisoExportable
}

// El adaptador futuro autoriza empleado propio+ejercicio+exportar_pdf y filtra
// tipos/campos en la lectura durable auditada. Una consulta completa de permisos
// no satisface este puerto. Los campos excluidos pueden llegar vacíos; si una
// fila trae restante conocido, incluye conciliación como metadato interno
// confirmado aunque ese campo no sea exportable. La aplicación lo borra antes
// del preparador cuando la política lo excluye. Filas excluidas no dejan
// subtotales ni huecos.
type FuenteExportacionPermisos interface {
	LeerResumenPropioParaExportar(context.Context, OrdenExportacionPermisos) (PermisosExportables, error)
}

// Modelo neutral mínimo: sin tipos opacos, solicitudes, fechas de disfrute,
// motivos, responsables, circuitos, referencias ni documentos. Cantidades de
// horas en minutos enteros; días en días enteros. nil conserva lo desconocido.
type FilaInformePermisos struct {
	Etiqueta          string                `json:"etiqueta"`
	Unidad            domain.LeaveUnit      `json:"unidad"`
	Computo           domain.ComputoPermiso `json:"computo"`
	PendienteResolver *int64                `json:"pendiente_resolver"`
	Concedido         *int64                `json:"concedido"`
	Restante          *int64                `json:"restante"`
	Conciliacion      string                `json:"conciliacion"`
}
type ResumenPermisosInforme struct {
	Ejercicio        int                   `json:"ejercicio"`
	CorteUTC         time.Time             `json:"corte_utc"`
	CamposPermitidos []string              `json:"campos_permitidos"`
	Filas            []FilaInformePermisos `json:"filas"`
}

type DocumentoPermisosPreparado struct {
	Contenido       []byte
	CatalogoRef     string
	CatalogoVersion int64
	CatalogoSHA256  string
}

// Usa el renderer común y ValidarSalida. Preparar no concede una descarga.
type PreparadorInformePermisos interface {
	PrepararInformePermisos(context.Context, ResumenPermisosInforme) (DocumentoPermisosPreparado, error)
}

type EvidenciaExportacionPermisos struct {
	SolicitadaUTC   time.Time
	EmpleadoRef     string
	Ejercicio       int
	Accion          string
	CorteUTC        time.Time
	FuenteRef       string
	FuenteVersion   int64
	FuenteSHA256    string
	PoliticaRef     string
	PoliticaVersion int64
	PoliticaSHA256  string
	CatalogoRef     string
	CatalogoVersion int64
	CatalogoSHA256  string
	DocumentoSHA256 string
	Tamano          int64
}
type ConfirmacionExportacionPermisos struct {
	Evidencia     EvidenciaExportacionPermisos
	ReciboRef     string
	AuditoriaRef  string
	ConfirmadaUTC time.Time
}

// Revalida autorización y la instantánea antes del COMMIT de consumo y auditoría
// de exportación. Falta su adaptador durable: ningún registro en memoria lo suple.
type RegistroExportacionPermisos interface {
	ConfirmarExportacionPermisos(context.Context, OrdenExportacionPermisos, EvidenciaExportacionPermisos) (ConfirmacionExportacionPermisos, error)
}
type ResultadoExportacionPermisos struct {
	Contenido    []byte
	Confirmacion ConfirmacionExportacionPermisos
}
