package ports

import (
	"context"
	"time"

	reglas "vec-diputacion-granada/internal/modules/bolsa/domain/reglasbaremo"
	puertosvec "vec-diputacion-granada/internal/vec/ports"
)

// OrdenConfirmacionReglasBaremo contiene exclusivamente material ya derivado
// y validado por application. Intencion es una referencia exacta al material
// idempotente canonico creado fuera de ports.
//
// EstadoEsperado hace explicito el CAS: nil solo es admisible para el alta; en
// el resto de operaciones contiene identificador, version, revision y huellas
// exactas. El adaptador no calcula ninguna transicion.
type OrdenConfirmacionReglasBaremo struct {
	Operacion        OperacionGobiernoReglasBaremo
	Intencion        reglas.ReferenciaVersionada
	EstadoEsperado   *reglas.VinculoEstadoReglasBaremo
	VersionResultado reglas.VersionGobernadaReglasBaremo
	Autorizacion     puertosvec.EvidenciaUsoDecisionAutorizacionSolicitudLigadaV2
	PruebaTransicion *reglas.ReferenciaVersionada
	EfectuarEn       time.Time
}

// ReciboConfirmacionReglasBaremo referencia los efectos confirmados por una
// unica transaccion. No constituye por si mismo la validacion del resultado;
// application coteja estos vinculos exactos con la orden original.
type ReciboConfirmacionReglasBaremo struct {
	Operacion               OperacionGobiernoReglasBaremo
	Intencion               reglas.ReferenciaVersionada
	EstadoResultado         reglas.VinculoEstadoReglasBaremo
	Transaccion             reglas.ReferenciaVersionada
	Auditoria               reglas.ReferenciaVersionada
	EventoOutbox            reglas.ReferenciaVersionada
	ConsumoAutorizacion     reglas.ReferenciaVersionada
	ConsumoPruebaTransicion *reglas.ReferenciaVersionada
	ConfirmadaEn            time.Time
}

// RepositorioGobiernoReglasBaremo confirma en una sola transaccion durable la
// idempotencia ya derivada, el CAS, la version resultante, el consumo de la
// autorizacion V2, la prueba de transicion cuando exista, auditoria y outbox.
// Ante cualquier fallo no confirma ningun efecto parcial.
type RepositorioGobiernoReglasBaremo interface {
	Confirmar(
		context.Context,
		OrdenConfirmacionReglasBaremo,
	) (ReciboConfirmacionReglasBaremo, error)
}
