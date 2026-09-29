package postgres

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgconn"

	"vec-diputacion-granada/internal/vec/domain"
	"vec-diputacion-granada/internal/vec/ports"
)

// Reproduce el 503 visto en la página del centro: la bandeja y las
// incorporaciones registran a la vez su decisión V3 y PostgreSQL aborta una
// con 40001 («pivot, during write») al insertar en
// decision_concedida_contexto_actor_v3. El registro se repite entero y la
// petición termina bien; solo un código de carrera se repite.
func TestRegistroContextoActorV3PostgreSQLRepiteCarreraSerializable(t *testing.T) {
	escenario := nuevoEscenarioRegistroContextoActorV3PostgreSQLPrueba(t, true)
	orden, err := ports.NuevaOrdenRegistroConcesionCandidataAutorizacionLigadaV3(
		escenario.solicitud, escenario.decision, escenario.motivo, escenario.resultado,
	)
	if err != nil {
		t.Fatal(err)
	}
	huella, _ := domain.HuellaSHA256DecisionAutorizacionV3(escenario.decision)
	registrada := escenario.ahora.Add(time.Microsecond)
	carrera := &pgconn.PgError{Code: "40001", Message: "dato_privado"}
	casos := []struct {
		nombre         string
		fallosConsulta int
		errorConsulta  error
		fallosCommit   int
		errorCommit    error
		esperado       error
		inicios        int
	}{
		{"40001 al insertar se recupera", 2, carrera, 0, nil, nil, 3},
		{"40P01 al insertar se recupera", 1, &pgconn.PgError{Code: "40P01"}, 0, nil, nil, 2},
		{"40001 al confirmar se recupera", 0, nil, 1, carrera, nil, 2},
		{"carrera persistente agota los intentos", -1, carrera, 0, nil, ports.ErrInstantaneaAutorizacionObsoleta, intentosRegistroContextoActorV3},
		{"55P03 no se repite", -1, &pgconn.PgError{Code: "55P03"}, 0, nil, ports.ErrInstantaneaAutorizacionObsoleta, 1},
		{"otro error no se repite", -1, &pgconn.PgError{Code: "XX000"}, 0, nil, ports.ErrRegistroConcesionAutorizacionLigadaV3NoDisponible, 1},
	}
	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			tx := nuevaTransaccionRegistroContextoActorV3PostgreSQLPrueba(true, "concedida", huella, registrada)
			consultas, commits := 0, 0
			tx.errorConsulta = caso.errorConsulta
			tx.alConsultar = func() {
				// Cada intento es una transacción nueva con su propia fila.
				tx.filas = nuevaTransaccionRegistroContextoActorV3PostgreSQLPrueba(true, "concedida", huella, registrada).filas
				consultas++
				if caso.fallosConsulta >= 0 && consultas > caso.fallosConsulta {
					tx.errorConsulta = nil
				}
			}
			tx.errorCommit = caso.errorCommit
			tx.alCommit = func() {
				commits++
				if commits > caso.fallosCommit {
					tx.errorCommit = nil
				}
			}
			iniciador := &iniciadorRegistroContextoActorV3PostgreSQLPrueba{tx: tx}
			almacen, _ := nuevoAlmacenAutorizacion(iniciador)
			obtenida, err := almacen.RegistrarConcesionCandidataAutorizacionLigadaV3SiInstantaneaVigente(context.Background(), orden)
			if !errors.Is(err, caso.esperado) || (caso.esperado == nil && !obtenida.Equal(registrada)) {
				t.Fatalf("resultado=%v error=%v", obtenida, err)
			}
			if err != nil && err != caso.esperado {
				t.Fatalf("la marca interna de carrera salió del adaptador: %T", err)
			}
			if iniciador.invocaciones != caso.inicios {
				t.Fatalf("transacciones abiertas %d; se esperaban %d", iniciador.invocaciones, caso.inicios)
			}
		})
	}
	t.Run("contexto cancelado deja de repetir", func(t *testing.T) {
		tx := nuevaTransaccionRegistroContextoActorV3PostgreSQLPrueba(true, "concedida", huella, registrada)
		ctx, cancelar := context.WithCancel(context.Background())
		tx.errorConsulta = carrera
		tx.alConsultar = cancelar
		iniciador := &iniciadorRegistroContextoActorV3PostgreSQLPrueba{tx: tx}
		almacen, _ := nuevoAlmacenAutorizacion(iniciador)
		if _, err := almacen.RegistrarConcesionCandidataAutorizacionLigadaV3SiInstantaneaVigente(ctx, orden); !errors.Is(err, context.Canceled) || iniciador.invocaciones != 1 {
			t.Fatalf("error=%v transacciones=%d", err, iniciador.invocaciones)
		}
	})
}
