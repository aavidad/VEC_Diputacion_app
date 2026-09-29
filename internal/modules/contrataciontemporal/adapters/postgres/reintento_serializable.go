package postgres

import (
	"context"

	postgresqlcomun "vec-diputacion-granada/internal/shared/postgresql"
)

// Las funciones de consumo V3 serializan toda la cadena de auditoría con
// SELECT ... FOR UPDATE sobre su única fila de control. Dos transacciones
// SERIALIZABLE simultáneas (por ejemplo, la bandeja de peticiones y la de
// incorporaciones que la página del centro pide a la vez) se ordenan allí:
// la segunda espera y, al confirmar la primera, PostgreSQL la aborta con
// 40001 porque la fila cambió después de su instantánea. Esa pérdida de
// carrera no es un fallo del servicio: se repite la transacción entera con
// el mismo material, que no llegó a consumirse porque el aborto lo revierte.
// ejecutarConReintentoSerializable repite intento mientras pierda una carrera
// de serialización o un interbloqueo (40001, 40P01), con la política única de
// internal/shared/postgresql. Cada intento abre y cierra su propia
// transacción; el contexto de la petición manda y nunca se repite otro error.
func ejecutarConReintentoSerializable(ctx context.Context, intento func() error) error {
	return postgresqlcomun.RepetirTrasCarreraSerializable(ctx, intento)
}

// decodificarJSONLimpio vacía el destino antes de decodificar: con el
// reintento serializable la misma validación puede correr varias veces y no
// debe arrastrar campos de un intento anterior.
func decodificarJSONLimpio[T any](b []byte, destino *T) error {
	var cero T
	*destino = cero
	return decodificarJSONEstricto(b, destino)
}
