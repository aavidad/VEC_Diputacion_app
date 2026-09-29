package postgres

import (
	"context"
	"time"
)

// Las funciones de consumo V3 serializan toda la cadena de auditoría con
// SELECT ... FOR UPDATE sobre su única fila de control. Dos transacciones
// SERIALIZABLE simultáneas (por ejemplo, la bandeja de peticiones y la de
// incorporaciones que la página del centro pide a la vez) se ordenan allí:
// la segunda espera y, al confirmar la primera, PostgreSQL la aborta con
// 40001 porque la fila cambió después de su instantánea. Esa pérdida de
// carrera no es un fallo del servicio: se repite la transacción entera con
// el mismo material, que no llegó a consumirse porque el aborto lo revierte.
const (
	intentosTransaccionSerializable = 6
	esperaBaseReintentoSerializable = 25 * time.Millisecond
)

// ejecutarConReintentoSerializable repite intento mientras pierda una carrera
// de serialización o un interbloqueo (40001, 40P01). Cada intento abre y
// cierra su propia transacción. La espera crece con el número de intento y
// lleva una parte aleatoria para que las peticiones no vuelvan a coincidir.
// El contexto de la petición manda: si vence o se cancela se deja de esperar
// y se devuelve el último error. Nunca repite otro tipo de error.
func ejecutarConReintentoSerializable(ctx context.Context, intento func() error) error {
	var err error
	for n := 1; ; n++ {
		err = intento()
		if err == nil || !errorPostgreSQLReintentable(err) || n == intentosTransaccionSerializable || ctx.Err() != nil {
			return err
		}
		espera := time.Duration(n)*esperaBaseReintentoSerializable +
			time.Duration(time.Now().UnixNano())%esperaBaseReintentoSerializable
		t := time.NewTimer(espera)
		select {
		case <-ctx.Done():
			t.Stop()
			return err
		case <-t.C:
		}
	}
}

// decodificarJSONLimpio vacía el destino antes de decodificar: con el
// reintento serializable la misma validación puede correr varias veces y no
// debe arrastrar campos de un intento anterior.
func decodificarJSONLimpio[T any](b []byte, destino *T) error {
	var cero T
	*destino = cero
	return decodificarJSONEstricto(b, destino)
}
