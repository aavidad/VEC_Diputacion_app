package postgres

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"vec-diputacion-granada/internal/vec/domain"
	"vec-diputacion-granada/internal/vec/ports"
)

func TestRegistroContextoActorV3PostgreSQLClasificaCommitDefinitivoYAmbiguo(t *testing.T) {
	escenario := nuevoEscenarioRegistroContextoActorV3PostgreSQLPrueba(t, true)
	orden, _ := ports.NuevaOrdenRegistroConcesionCandidataAutorizacionLigadaV3(
		escenario.solicitud, escenario.decision, escenario.motivo, escenario.resultado,
	)
	huella, _ := domain.HuellaSHA256DecisionAutorizacionV3(escenario.decision)
	registrada := escenario.ahora.Add(time.Microsecond)
	cancelacionTardia40001, cancelarTarde := context.WithCancel(context.Background())
	cancelacionTardia40P01, cancelarTardeDeadlock := context.WithCancel(context.Background())
	cancelacionTardia55P03, cancelarTardeLock := context.WithCancel(context.Background())
	cancelacionTardiaDesconocida, cancelarTardeDesconocida := context.WithCancel(context.Background())
	cancelacionTardiaRollback, cancelarTardeRollback := context.WithCancel(context.Background())
	cancelacionTardiaExitosa, cancelarTardeExitosa := context.WithCancel(context.Background())
	casos := []struct {
		nombre      string
		ctx         context.Context
		alCommit    func()
		errorCommit error
		errEsperado error
	}{
		{
			"40001 definitivo aunque cancelacion compita", cancelacionTardia40001, cancelarTarde,
			fmt.Errorf("envuelto: %w", &pgconn.PgError{Code: "40001", Message: "dato_privado"}),
			ports.ErrInstantaneaAutorizacionObsoleta,
		},
		{
			"40P01 definitivo aunque cancelacion compita", cancelacionTardia40P01, cancelarTardeDeadlock,
			&pgconn.PgError{Code: "40P01", Message: "dato_privado"},
			ports.ErrInstantaneaAutorizacionObsoleta,
		},
		{
			"55P03 no prueba obsolescencia", cancelacionTardia55P03, cancelarTardeLock,
			&pgconn.PgError{Code: "55P03", Message: "dato_privado"},
			ports.ErrRegistroConcesionAutorizacionLigadaV3NoDisponible,
		},
		{
			"codigo desconocido ambiguo", cancelacionTardiaDesconocida, cancelarTardeDesconocida,
			&pgconn.PgError{Code: "XX000", Message: "dato_privado"},
			ports.ErrRegistroConcesionAutorizacionLigadaV3NoDisponible,
		},
		{
			"rollback implicito ambiguo", cancelacionTardiaRollback, cancelarTardeRollback,
			pgx.ErrTxCommitRollback,
			ports.ErrRegistroConcesionAutorizacionLigadaV3NoDisponible,
		},
		{
			"commit exitoso definitivo", cancelacionTardiaExitosa, cancelarTardeExitosa,
			nil, nil,
		},
	}
	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			tx := nuevaTransaccionRegistroContextoActorV3PostgreSQLPrueba(
				true, "concedida", huella, registrada,
			)
			tx.alCommit = caso.alCommit
			tx.errorCommit = caso.errorCommit
			almacen, _ := nuevoAlmacenAutorizacion(
				&iniciadorRegistroContextoActorV3PostgreSQLPrueba{tx: tx},
			)
			obtenida, err := almacen.RegistrarConcesionCandidataAutorizacionLigadaV3SiInstantaneaVigente(
				caso.ctx, orden,
			)
			if caso.errEsperado == nil {
				if err != nil || !obtenida.Equal(registrada) || !tx.commitConsiderado {
					t.Fatalf("commit valido: instante=%v error=%v eventos=%v", obtenida, err, tx.eventos)
				}
				return
			}
			if !errors.Is(err, caso.errEsperado) || errors.Is(err, context.Canceled) ||
				strings.Contains(err.Error(), "dato_privado") || tx.commitConsiderado {
				t.Fatalf("clasificacion de commit: error=%v eventos=%v", err, tx.eventos)
			}
		})
	}
}
