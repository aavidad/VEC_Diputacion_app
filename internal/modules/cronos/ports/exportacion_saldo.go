package ports

import (
	"context"
	"errors"
	"time"

	vecdomain "vec-diputacion-granada/internal/vec/domain"
)

var (
	ErrExportacionSaldoNoDisponible = errors.New("cronos_exportacion_saldo_no_disponible")
	ErrExportacionSaldoInvalida     = errors.New("cronos_exportacion_saldo_invalida")
)

// OrdenExportacionSaldo contiene exclusivamente la identidad resuelta por el
// servidor. Tener esta orden no concede permiso de lectura ni de exportación.
// El permiso de exportación es independiente del de consulta del saldo.
type OrdenExportacionSaldo struct{ actor vecdomain.ContextoActor }

func NuevaOrdenExportacionSaldo(actor vecdomain.ContextoActor) (OrdenExportacionSaldo, error) {
	copia, err := actor.Clonar()
	if err != nil {
		return OrdenExportacionSaldo{}, ErrExportacionSaldoInvalida
	}
	return OrdenExportacionSaldo{actor: copia}, nil
}

func (o OrdenExportacionSaldo) ContextoActor() (vecdomain.ContextoActor, error) {
	return o.actor.Clonar()
}

// SaldoExportable es una proyección mínima ya autorizada y auditada. No admite
// fichajes, motivos de permiso, documentos, unidades ni datos de otras personas.
// Los punteros nulos conservan un dato desconocido; no equivalen a cero.
type SaldoExportable struct {
	EmpleadoRef   string
	Periodo       PeriodoConsultaSaldo
	Resumen       ResumenConsultaSaldo
	FuenteRef     string
	FuenteVersion int64
}

// FuenteExportacionSaldo debe resolver el empleado propio del contexto servidor
// y consumir una autorización nominal de exportación para ese periodo y campos.
// Lectura, autorización y auditoría se confirman en la misma frontera durable.
// No se proporciona un adaptador productivo hasta disponer de esa autoridad.
type FuenteExportacionSaldo interface {
	LeerSaldoPropioParaExportar(context.Context, OrdenExportacionSaldo, PeriodoConsultaSaldo) (SaldoExportable, error)
}

// DocumentoSaldoPreparado todavía no es una descarga autorizada.
type DocumentoSaldoPreparado struct {
	Contenido       []byte
	CatalogoRef     string
	CatalogoVersion int64
	CatalogoSHA256  string
}

// El preparador debe validar la salida con RenderizadorDocumento.ValidarSalida
// y rechazar idioma discordante y errores antes de devolver cualquier byte.
type PreparadorInformeSaldo interface {
	PrepararInformeSaldo(context.Context, SaldoExportable) (DocumentoSaldoPreparado, error)
}

// EvidenciaExportacionSaldo liga la confirmación al PDF exacto y a su fuente y
// catálogo versionados. No contiene el saldo ni otros datos del documento.
type EvidenciaExportacionSaldo struct {
	SolicitadaUTC   time.Time
	EmpleadoRef     string
	Periodo         PeriodoConsultaSaldo
	FuenteRef       string
	FuenteVersion   int64
	CatalogoRef     string
	CatalogoVersion int64
	CatalogoSHA256  string
	DocumentoSHA256 string
	Tamano          int64
}

type ConfirmacionExportacionSaldo struct {
	Evidencia     EvidenciaExportacionSaldo
	ReciboRef     string
	AuditoriaRef  string
	ConfirmadaUTC time.Time
}

// RegistroExportacionSaldo revalida el permiso nominal propio, el contexto y la
// fuente inmediatamente antes de confirmar. Devuelve el recibo sólo tras COMMIT
// de autorización consumida y auditoría de exportación. Un bool o un registro
// en memoria no sustituyen esta confirmación. Su implementación durable falta.
type RegistroExportacionSaldo interface {
	ConfirmarExportacionSaldo(context.Context, OrdenExportacionSaldo, EvidenciaExportacionSaldo) (ConfirmacionExportacionSaldo, error)
}

type ResultadoExportacionSaldo struct {
	Contenido    []byte
	Confirmacion ConfirmacionExportacionSaldo
}
