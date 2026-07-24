package postgresql

import (
	"context"
	"errors"
	"testing"
	"time"
)

type transaccionReversionPrueba struct {
	contexto context.Context
}

func (t *transaccionReversionPrueba) Rollback(ctx context.Context) error {
	t.contexto = ctx
	return nil
}

func TestRevertirAcotadoUsaContextoIndependienteConFechaLimite(t *testing.T) {
	transaccion := &transaccionReversionPrueba{}
	RevertirAcotado(transaccion)
	ctx := transaccion.contexto
	if ctx == nil {
		t.Fatal("la reversión no recibió contexto")
	}
	limite, existe := ctx.Deadline()
	if !existe {
		t.Fatal("la reversión no recibió fecha límite")
	}
	restante := time.Until(limite)
	if restante <= 0 || restante > DuracionMaximaReversion {
		t.Fatalf("presupuesto de reversión inesperado: %s", restante)
	}
	if !errors.Is(ctx.Err(), context.Canceled) {
		t.Fatalf("el contexto no se liberó al terminar: %v", ctx.Err())
	}
}

func TestRevertirAcotadoAdmiteTransaccionNula(t *testing.T) {
	RevertirAcotado(nil)
}
