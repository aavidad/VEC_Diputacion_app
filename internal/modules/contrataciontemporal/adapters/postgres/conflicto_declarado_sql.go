package postgres

import (
	"errors"

	"github.com/jackc/pgx/v5/pgconn"
)

// rutinaRaisePLpgSQL es la rutina del servidor que emite un RAISE de PL/pgSQL.
const rutinaRaisePLpgSQL = "exec_stmt_raise"

// conflictoDeclaradoPorFuncionSQL distingue un 40001 que la propia función
// SQL lanza con RAISE para señalar un conflicto de versión (el expediente ya
// no está donde la operación lo esperaba: otra operación lo avanzó, o una
// repetición llega tras un despliegue con otro perfil) de una carrera de
// serialización del servidor. El primero no se arregla repitiendo: la
// transacción ya se revirtió sin efectos y se responde conflicto.
func conflictoDeclaradoPorFuncionSQL(err error) bool {
	var postgres *pgconn.PgError
	return errors.As(err, &postgres) && postgres.Code == "40001" &&
		postgres.Routine == rutinaRaisePLpgSQL
}
