package postgres

import (
	"context"
	"errors"
	"testing"

	"github.com/jackc/pgx/v5"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

type iniciadorPertenenciaPrueba struct{ llamadas int }

func (i *iniciadorPertenenciaPrueba) BeginTx(context.Context, pgx.TxOptions) (pgx.Tx, error) {
	i.llamadas++
	return nil, errors.New("sin base")
}

// La pertenencia valida la entrada antes de abrir la transacción y no pide
// ninguna autorización al proveedor.
func TestExpedienteDelCentroValidaAntesDeConsultar(t *testing.T) {
	pool := &iniciadorPertenenciaPrueba{}
	r := &RepositorioIncorporacionCentroPostgreSQL{pool: pool}
	actor := domain.ActorPeticionCentro{ActorRef: "per_ana", PerfilRef: "prf_centro", CentroRef: "centro-520", PuestoRef: "puesto-1"}
	for _, caso := range []struct {
		actor      domain.ActorPeticionCentro
		org, exped string
	}{{domain.ActorPeticionCentro{}, "organizacion:1", "expediente:ct:1"}, {actor, "", "expediente:ct:1"}, {actor, "organizacion:1", ""}} {
		if suyo, err := r.ExpedienteDelCentro(context.Background(), caso.org, caso.actor, caso.exped); suyo || !errors.Is(err, ports.ErrIncorporacionCentroInvalida) {
			t.Fatalf("%+v: %v %v", caso, suyo, err)
		}
	}
	if pool.llamadas != 0 {
		t.Fatal("abre transacción con entrada inválida")
	}
	if suyo, err := r.ExpedienteDelCentro(context.Background(), "organizacion:1", actor, "expediente:ct:1"); suyo || err == nil || pool.llamadas != 1 {
		t.Fatalf("sin base: %v %v", suyo, err)
	}
}
