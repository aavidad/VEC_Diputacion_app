package ports

import (
	"context"
	"time"

	calculo "vec-diputacion-granada/internal/modules/bolsa/domain/calculoexperiencia"
	reglas "vec-diputacion-granada/internal/modules/bolsa/domain/reglasbaremo"
	puertosvec "vec-diputacion-granada/internal/vec/ports"
)

// SolicitudConsultaExactaReglasBaremo solo admite un estado identificado por
// contenido, version, revision y huellas exactas.
type SolicitudConsultaExactaReglasBaremo struct {
	Selector     reglas.VinculoEstadoReglasBaremo
	Autorizacion puertosvec.EvidenciaUsoDecisionAutorizacionSolicitudLigadaV2
	SolicitadaEn time.Time
}

type ResultadoConsultaExactaReglasBaremo struct {
	Version             reglas.VersionGobernadaReglasBaremo
	Auditoria           reglas.ReferenciaVersionada
	ConsumoAutorizacion reglas.ReferenciaVersionada
	ConsultadaEn        time.Time
}

// ConsultaAutorizadaReglasBaremo no resuelve alias temporales ni selecciona
// versiones por orden de insercion.
type ConsultaAutorizadaReglasBaremo interface {
	ObtenerVersionExacta(
		context.Context,
		SolicitudConsultaExactaReglasBaremo,
	) (ResultadoConsultaExactaReglasBaremo, error)
}

// SelectorFuenteExactaCalculoReglasBaremo liga el calculo a reglas, entrada,
// sujeto pseudonimizado y convocatoria exactos. La referencia del sujeto la
// crea la capa confiable; este contrato no acepta DNI ni otros datos directos.
type SelectorFuenteExactaCalculoReglasBaremo struct {
	EstadoReglas       reglas.VinculoEstadoReglasBaremo
	InstantaneaEntrada reglas.ReferenciaVersionada
	SujetoPseudonimo   reglas.ReferenciaVersionada
	Convocatoria       reglas.ReferenciaVersionada
}

type SolicitudFuenteExactaCalculoReglasBaremo struct {
	Selector     SelectorFuenteExactaCalculoReglasBaremo
	Autorizacion puertosvec.EvidenciaUsoDecisionAutorizacionSolicitudLigadaV2
	SolicitadaEn time.Time
}

// PruebaFuenteExactaCalculoReglasBaremo es compacta: liga la evidencia
// verificable a la instantanea y su contenido, sin repetir tramos ni catalogos.
type PruebaFuenteExactaCalculoReglasBaremo struct {
	Evidencia           reglas.ReferenciaVersionada
	Verificador         reglas.ReferenciaVersionada
	EstadoReglas        reglas.VinculoEstadoReglasBaremo
	InstantaneaEntrada  reglas.ReferenciaVersionada
	HuellaEntradaSHA256 string
	SujetoPseudonimo    reglas.ReferenciaVersionada
	Convocatoria        reglas.ReferenciaVersionada
	EmitidaEn           time.Time
	ValidaHasta         time.Time
}

type FuenteExactaCalculoReglasBaremo struct {
	Version             reglas.VersionGobernadaReglasBaremo
	Entrada             calculo.EntradaExperiencia
	Prueba              PruebaFuenteExactaCalculoReglasBaremo
	Auditoria           reglas.ReferenciaVersionada
	ConsumoAutorizacion reglas.ReferenciaVersionada
	ConsumoPrueba       reglas.ReferenciaVersionada
	ObtenidaEn          time.Time
}

// FuenteReglasBaremoParaCalculo obtiene y verifica la procedencia de una
// instantanea exacta. El adaptador devuelve el consumo durable de la prueba;
// una entrada restaurada localmente no satisface este contrato.
type FuenteReglasBaremoParaCalculo interface {
	ObtenerFuenteExacta(
		context.Context,
		SolicitudFuenteExactaCalculoReglasBaremo,
	) (FuenteExactaCalculoReglasBaremo, error)
}
