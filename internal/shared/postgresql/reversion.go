// Package postgresql contiene salvaguardas técnicas comunes a adaptadores
// PostgreSQL. No conoce dominios, credenciales ni esquemas de negocio.
package postgresql

import (
	"context"
	"time"
)

const DuracionMaximaReversion = 2 * time.Second

// TransaccionReversible es el contrato mínimo necesario para liberar una
// transacción abandonada sin permitir que la limpieza bloquee indefinidamente
// el cierre de una petición o del proceso.
type TransaccionReversible interface {
	Rollback(context.Context) error
}

// RevertirAcotado intenta revertir con un contexto independiente y acotado.
// Es segura como defer tras Commit: PostgreSQL devolverá la transacción ya
// cerrada y el error no reemplaza al resultado principal de la operación.
func RevertirAcotado(transaccion TransaccionReversible) {
	if transaccion == nil {
		return
	}
	ctx, cancelar := context.WithTimeout(context.Background(), DuracionMaximaReversion)
	defer cancelar()
	_ = transaccion.Rollback(ctx)
}
