package ports

import (
	"context"

	"vec-diputacion-granada/internal/vec/domain"
)

// EmisorResultadosTecnicosConContexto registra el resultado observado de una
// operación usando el emisor técnico común. Nunca hace E/S en el llamante ni
// cambia el recibo de negocio: la pérdida de registro se cuenta en el emisor.
// El consumidor debe llamarlo después de conocer el resultado real; un éxito
// confirmado en SQL no se reintenta por fallo del destino técnico.
type EmisorResultadosTecnicosConContexto interface {
	EmitirResultadoConContexto(context.Context, domain.SolicitudResultadoTecnico)
}

type MetricasEmisionResultadosTecnicos struct {
	Aceptados       uint64
	Descartados     uint64
	Invalidos       uint64
	SinCorrelacion  uint64
	Escritos        uint64
	FallosEscritura uint64
}

type ConsultaMetricasEmisionResultadosTecnicos interface {
	MetricasResultadosTecnicos() MetricasEmisionResultadosTecnicos
}
