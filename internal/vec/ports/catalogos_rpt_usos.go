package ports

import (
	"context"
	"errors"

	"vec-diputacion-granada/internal/vec/domain"
)

var (
	ErrUsoCategoriaRPTInvalido     = errors.New("vec: operación de uso de categoría RPT inválida")
	ErrUsoCategoriaRPTDenegado     = errors.New("vec: operación de uso de categoría RPT denegada")
	ErrUsoCategoriaRPTConflicto    = errors.New("vec: operación de uso de categoría RPT en conflicto")
	ErrUsoCategoriaRPTNoDisponible = errors.New("vec: operación de uso de categoría RPT no disponible")
	ErrUsoCategoriaRPTNoConfiable  = errors.New("vec: recibo de uso de categoría RPT no confiable")
)

// MaterialReservaUsoCategoriaRPT identifica la publicación exacta y el uso
// que el módulo propietario va a confirmar. El descriptor de catálogo y el
// consumidor técnico se fijan aparte en la composición confiable.
type MaterialReservaUsoCategoriaRPT struct {
	Consumidor       string
	UsoRef           string
	CategoriaID      string
	Publicacion      ReferenciaPublicacionRPT
	ReservaReciboRef string
}

// MaterialTerminalUsoCategoriaRPT añade el recibo estable y el testigo opaco
// de la operación del módulo propietario. El caso de uso propietario debe
// verificar ese efecto por su puerto antes de solicitar V3. RPT no realiza el
// efecto de CT, Bolsa o Personal ni convierte este testigo en prueba legal.
type MaterialTerminalUsoCategoriaRPT struct {
	Reserva           MaterialReservaUsoCategoriaRPT
	TerminalReciboRef string
	EvidenciaRef      string
	EvidenciaSHA256   string
}

type OrdenReservaUsoCategoriaRPT struct {
	Material     MaterialReservaUsoCategoriaRPT
	Solicitud    domain.SolicitudAutorizacionLigadaV3
	Autorizacion ExportacionMaterialConsumoAutorizacionAtestadaV3
}

type OrdenConfirmacionUsoCategoriaRPT struct {
	Material     MaterialTerminalUsoCategoriaRPT
	Solicitud    domain.SolicitudAutorizacionLigadaV3
	Autorizacion ExportacionMaterialConsumoAutorizacionAtestadaV3
}

type OrdenCancelacionUsoCategoriaRPT struct {
	Material     MaterialTerminalUsoCategoriaRPT
	Solicitud    domain.SolicitudAutorizacionLigadaV3
	Autorizacion ExportacionMaterialConsumoAutorizacionAtestadaV3
}

// PreparacionAutorizacionUsoCategoriaRPT contiene el recurso exacto para
// solicitar V3. No es una concesión ni prueba de existencia o efecto: la
// preparación sólo calcula la huella del material suministrado. El actor,
// motivo y correlación proceden de las autoridades del caso de uso.
type PreparacionAutorizacionUsoCategoriaRPT struct {
	Accion           string
	Finalidad        string
	AudienciaConsumo string
	Recurso          domain.RecursoAutorizable
}

// PreparadorUsosCategoriaRPT mantiene la canonicalización del material en
// la autoridad PostgreSQL que lo consume. El llamador prepara, obtiene V3
// y envía el mismo material al gestor; el gestor vuelve a cotejar la huella
// dentro de la transacción de efecto. Nunca imita jsonb::text desde Go.
type PreparadorUsosCategoriaRPT interface {
	PrepararReservaUsoCategoriaRPT(context.Context, MaterialReservaUsoCategoriaRPT) (PreparacionAutorizacionUsoCategoriaRPT, error)
	PrepararConfirmacionUsoCategoriaRPT(context.Context, MaterialTerminalUsoCategoriaRPT) (PreparacionAutorizacionUsoCategoriaRPT, error)
	PrepararCancelacionUsoCategoriaRPT(context.Context, MaterialTerminalUsoCategoriaRPT) (PreparacionAutorizacionUsoCategoriaRPT, error)
}

// GestorUsosCategoriaRPT es el puerto de la autoridad común. Cada operación
// consume una decisión V3 nueva en la misma transacción que el recibo de uso.
// Una reserva repetida puede devolver un uso ya confirmado o cancelado; el
// estado y los recibos de ResultadoUsoCategoriaRPT son la fuente del resultado,
// sin un booleano de replay inventado por el cliente.
type GestorUsosCategoriaRPT interface {
	ReservarUsoCategoriaRPT(context.Context, OrdenReservaUsoCategoriaRPT) (ResultadoUsoCategoriaRPT, error)
	ConfirmarUsoCategoriaRPT(context.Context, OrdenConfirmacionUsoCategoriaRPT) (ResultadoUsoCategoriaRPT, error)
	CancelarUsoCategoriaRPT(context.Context, OrdenCancelacionUsoCategoriaRPT) (ResultadoUsoCategoriaRPT, error)
}
