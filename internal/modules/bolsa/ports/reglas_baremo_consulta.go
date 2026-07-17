package ports

import (
	"context"
	"time"

	calculo "vec-diputacion-granada/internal/modules/bolsa/domain/calculoexperiencia"
	oficial "vec-diputacion-granada/internal/modules/bolsa/domain/calculoexperienciaoficial"
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

// SelectorFuenteExactaCalculoReglasBaremo conserva en la frontera hexagonal el
// contrato canonico versionado del dominio. Sus metodos Validar,
// RepresentacionCanonicaV1 y HuellaSHA256V1 son la unica fuente de verdad para
// aplicacion y adaptadores; el algoritmo no debe copiarse fuera del dominio.
type SelectorFuenteExactaCalculoReglasBaremo = oficial.SelectorFuenteExactaCalculoReglasBaremo

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
	ConsumoAutorizacion oficial.ReciboConsumoAutorizacionFuenteV2
	ConsumoPrueba       reglas.ReferenciaVersionada
	ObtenidaEn          time.Time
}

// FuenteReglasBaremoParaCalculo obtiene y verifica la procedencia de una
// instantanea exacta. El adaptador devuelve el consumo durable de la prueba;
// una entrada restaurada localmente no satisface este contrato.
//
// NO-GO PRODUCCION: el contrato exige un recibo canonico que liga la decision
// V2, el selector, la fuente, la prueba y la auditoria exactas. Aun no existe
// un adaptador PostgreSQL productivo que lo cree atomicamente con el consumo;
// una construccion en memoria o posterior a la lectura no satisface el puerto.
type FuenteReglasBaremoParaCalculo interface {
	ObtenerFuenteExacta(
		context.Context,
		SolicitudFuenteExactaCalculoReglasBaremo,
	) (FuenteExactaCalculoReglasBaremo, error)
}
