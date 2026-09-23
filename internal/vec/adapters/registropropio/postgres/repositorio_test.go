package postgres

import (
	"context"
	"errors"
	"testing"

	"github.com/jackc/pgx/v5"
	"vec-diputacion-granada/internal/vec/ports"
)

type iniciadorRegistroPropioObservado struct{ aperturas int }

func (i *iniciadorRegistroPropioObservado) BeginTx(context.Context, pgx.TxOptions) (pgx.Tx, error) {
	i.aperturas++
	return nil, errors.New("no debe abrirse una instantánea antes de la concesión V3")
}

func TestRegistroPropioNoAbreSerializableSinMaterialV3Confirmado(t *testing.T) {
	iniciador := &iniciadorRegistroPropioObservado{}
	r := &RepositorioRegistroPropioPostgreSQL{pool: iniciador}
	llamado := false
	_, err := r.RegistrarPropio(context.Background(), ports.OrdenRegistroPropioV1{}, func(context.Context) error {
		llamado = true
		return nil
	})
	if !errors.Is(err, ports.ErrRegistroPropioNoDisponible) || iniciador.aperturas != 0 || llamado {
		t.Fatal("la transacción o la revalidación comenzaron antes de disponer de material V3")
	}
}
