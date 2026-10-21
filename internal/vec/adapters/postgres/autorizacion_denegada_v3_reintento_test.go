package postgres

import (
	"errors"
	"reflect"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgconn"

	"vec-diputacion-granada/internal/shared/postgresql"
	"vec-diputacion-granada/internal/vec/domain"
	"vec-diputacion-granada/internal/vec/ports"
)

func TestDenegacionContextoActorV3Repite40001ConLaMismaOrden(t *testing.T) {
	for _, persistente := range []bool{false, true} {
		escenario := nuevoEscenarioRegistroContextoActorV3PostgreSQLPrueba(t, false)
		orden, err := ports.NuevaOrdenRegistroDenegacionAutorizacionLigadaV3(
			escenario.solicitud, escenario.decision, escenario.motivo, escenario.resultado)
		if err != nil {
			t.Fatal(err)
		}
		_, codigo, _ := escenario.decision.Resultado()
		huella, _ := domain.HuellaSHA256DecisionAutorizacionV3(escenario.decision)
		registrada := escenario.ahora.Add(time.Microsecond)
		tx := nuevaTransaccionRegistroContextoActorV3PostgreSQLPrueba(false, codigo, huella, registrada)
		var argumentos []any
		consultas := 0
		tx.alConsultar = func() {
			consultas++
			if consultas == 1 {
				argumentos = copiarArgumentosRegistroContextoActorV3Prueba(tx.argumentos)
			} else if !reflect.DeepEqual(argumentos, tx.argumentos) || !tx.rollbackInvocado {
				t.Fatal("el reintento cambió la decisión o no cerró el intento anterior")
			}
			tx.errorConsulta = nil
			if persistente || consultas == 1 {
				tx.errorConsulta = &pgconn.PgError{Code: "40001", Message: "detalle_privado"}
			}
			tx.filas = nuevaTransaccionRegistroContextoActorV3PostgreSQLPrueba(false, codigo, huella, registrada).filas
		}
		iniciador := &iniciadorRegistroContextoActorV3PostgreSQLPrueba{tx: tx}
		almacen, err := nuevoAlmacenAutorizacion(iniciador)
		if err != nil {
			t.Fatal(err)
		}
		err = almacen.RegistrarDenegacionAutorizacionLigadaV3(t.Context(), orden)
		if !persistente && (err != nil || iniciador.invocaciones != 2 || !tx.commitConsiderado) {
			t.Fatalf("denegación no recuperada: %v intentos=%d", err, iniciador.invocaciones)
		}
		if persistente && (!errors.Is(err, ports.ErrInstantaneaAutorizacionObsoleta) ||
			iniciador.invocaciones != postgresql.IntentosMaximosCarreraSerializable || tx.commitInvocado) {
			t.Fatalf("agotamiento incorrecto: %v intentos=%d", err, iniciador.invocaciones)
		}
		var pg *pgconn.PgError
		if errors.As(err, &pg) {
			t.Fatal("el detalle SQL salió del adaptador")
		}
	}
}
