package adminperfiles

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
)

type filaRuntimeContextoADMIN struct {
	login      string
	acreditada bool
	err        error
}

func (f filaRuntimeContextoADMIN) Scan(destinos ...any) error {
	if f.err != nil {
		return f.err
	}
	*destinos[0].(*string) = f.login
	*destinos[1].(*bool) = f.acreditada
	return nil
}

type poolRuntimeContextoADMIN struct {
	poolContextoFalso
	fila     filaRuntimeContextoADMIN
	consulta string
}

func (p *poolRuntimeContextoADMIN) QueryRow(_ context.Context, q string, _ ...any) pgx.Row {
	p.consulta = q
	return p.fila
}

func TestContextoADMINExigeGrupoPropioAlConstruir(t *testing.T) {
	const acreditar = `SELECT identidad_login,acreditada FROM vec_contexto_actor_v1.acreditar_runtime_contexto_admin_v1()`
	for _, caso := range []struct {
		fila   filaRuntimeContextoADMIN
		valida bool
	}{
		{filaRuntimeContextoADMIN{login: "", acreditada: true}, false},
		{filaRuntimeContextoADMIN{login: "login_admin", acreditada: false}, false},
		{filaRuntimeContextoADMIN{err: errors.New("42501")}, false},
		{filaRuntimeContextoADMIN{login: "login_admin", acreditada: true}, true},
	} {
		pool := &poolRuntimeContextoADMIN{fila: caso.fila}
		a, err := nuevoContextoRegistradoPostgreSQL(context.Background(), pool, relojPrueba{ahora: time.Now().UTC()})
		if (a != nil && err == nil) != caso.valida || pool.consulta != acreditar || len(pool.opciones) != 0 {
			t.Fatal("el contexto aceptó un LOGIN no acreditado o inició trabajo antes de construir")
		}
	}
}
