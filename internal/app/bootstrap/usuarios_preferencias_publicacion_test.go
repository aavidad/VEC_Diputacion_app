package bootstrap

import (
	"context"
	"errors"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestCuatroAudienciasCompartenTxYFalloTerceraNoConfirmaClaves(t *testing.T) {
	var materiales [4]materialAtestacionContratacionTemporalDesarrollo
	for i, d := range descriptoresMaterialPreferenciasUsuariosDesarrollo() {
		materiales[i].audienciaConsumo = d.Audiencia
	}
	var intentos, transacciones int
	var confirmadas, preparadas []string
	ejecutar := func(_ context.Context, _ *pgxpool.Pool, operar func(pgx.Tx) error) error {
		transacciones++
		preparadas = nil
		err := operar(nil)
		if err == nil {
			confirmadas = append(confirmadas, preparadas...)
		}
		return err
	}
	publicar := func(_ context.Context, _ pgx.Tx, m *materialAtestacionContratacionTemporalDesarrollo) error {
		intentos++
		if intentos == 3 {
			return errors.New("tercera audiencia denegada")
		}
		preparadas = append(preparadas, m.audienciaConsumo)
		return nil
	}
	if err := ejecutarPublicacionPreferenciasEnUnaTx(context.Background(), nil, &materiales, ejecutar, publicar); err == nil || intentos != 3 || transacciones != 1 || len(confirmadas) != 0 {
		t.Fatalf("lote parcial: intentos=%d transacciones=%d confirmadas=%d err=%v", intentos, transacciones, len(confirmadas), err)
	}
	intentos = 0
	publicar = func(_ context.Context, _ pgx.Tx, m *materialAtestacionContratacionTemporalDesarrollo) error {
		intentos++
		preparadas = append(preparadas, m.audienciaConsumo)
		return nil
	}
	if err := ejecutarPublicacionPreferenciasEnUnaTx(context.Background(), nil, &materiales, ejecutar, publicar); err != nil || intentos != 4 || transacciones != 2 || len(confirmadas) != 4 {
		t.Fatalf("lote completo: intentos=%d transacciones=%d confirmadas=%d err=%v", intentos, transacciones, len(confirmadas), err)
	}
}
