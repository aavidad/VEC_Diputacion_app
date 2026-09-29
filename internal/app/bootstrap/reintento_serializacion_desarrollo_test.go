package bootstrap

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/jackc/pgx/v5/pgconn"
)

// El fallo por pérdida de carrera sigue siendo el centinela de siempre y solo
// él (40001, 40P01) queda marcado como repetible.
func TestFalloSerializacionCTDesarrolloConservaCentinela(t *testing.T) {
	for _, codigo := range []string{"40001", "40P01"} {
		err := falloPostgreSQLCTDesarrollo(&pgconn.PgError{Code: codigo})
		if !errors.Is(err, errPostgreSQLContratacionTemporalDesarrolloNoDisponible) || !falloReintentableSerializacionCTDesarrollo(err) {
			t.Fatalf("%s: %v", codigo, err)
		}
		if causa := causaFalloPostgreSQLCTDesarrollo(err); causa != "dependencia_no_disponible" {
			t.Fatalf("causa del envoltorio %q", causa)
		}
	}
	for _, causa := range []error{nil, &pgconn.PgError{Code: "42501"}, context.Canceled} {
		if err := falloPostgreSQLCTDesarrollo(causa); err != errPostgreSQLContratacionTemporalDesarrolloNoDisponible {
			t.Fatalf("%v: %v", causa, err)
		}
	}
}

func TestReintentarSerializacionCTDesarrolloPolitica(t *testing.T) {
	carrera := falloSerializacionPostgreSQLCTDesarrollo{}
	contar := func(fallos int, err error) func() error {
		n := 0
		return func() error {
			n++
			if fallos < 0 || n <= fallos {
				return err
			}
			return nil
		}
	}
	casos := []struct {
		nombre   string
		fallos   int
		err      error
		llamadas int
		conExito bool
	}{
		{"se recupera", 2, carrera, 3, true},
		{"otro error no se repite", -1, errPostgreSQLContratacionTemporalDesarrolloNoDisponible, 1, false},
		{"agota los intentos", -1, carrera, intentosPublicacionSerializableCTDesarrollo, false},
	}
	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			llamadas := 0
			intento := contar(caso.fallos, caso.err)
			err := reintentarSerializacionCTDesarrollo(context.Background(), func() error {
				llamadas++
				return intento()
			})
			if (err == nil) != caso.conExito || llamadas != caso.llamadas {
				t.Fatalf("err=%v llamadas=%d", err, llamadas)
			}
		})
	}
	ctx, cancelar := context.WithCancel(context.Background())
	llamadas := 0
	err := reintentarSerializacionCTDesarrollo(ctx, func() error {
		llamadas++
		cancelar()
		return carrera
	})
	if err == nil || llamadas != 1 {
		t.Fatalf("contexto cancelado: err=%v llamadas=%d", err, llamadas)
	}
}

// Simula en memoria la carrera de la página del centro: cada publicación toma
// su instantánea, espera el bloqueo del perfil y aborta con 40001 si otra
// confirmó entretanto. El primer intento de todas parte de la misma
// instantánea, así que sin reintento solo una confirma; con él, todas
// terminan y cada confirmación parte de la versión vigente. Cada ronda
// confirma al menos una, de modo que caben tantas peticiones como intentos.
func TestReintentarSerializacionCTDesarrolloCarreraConcurrente(t *testing.T) {
	const peticiones = intentosPublicacionSerializableCTDesarrollo
	ejecutar := func(reintentar bool) (fallidas int64) {
		var bloqueo sync.Mutex
		var version atomic.Int64
		var falladas atomic.Int64
		var salida sync.WaitGroup
		salida.Add(peticiones)
		publicar := func(primero *bool) error {
			instantanea := version.Load()
			if *primero {
				*primero = false
				salida.Done()
				salida.Wait()
			}
			bloqueo.Lock()
			defer bloqueo.Unlock()
			if version.Load() != instantanea {
				return falloPostgreSQLCTDesarrollo(&pgconn.PgError{Code: "40001"})
			}
			version.Add(1)
			return nil
		}
		var grupo sync.WaitGroup
		for range peticiones {
			grupo.Go(func() {
				primero := true
				var err error
				if reintentar {
					err = reintentarSerializacionCTDesarrollo(context.Background(), func() error { return publicar(&primero) })
				} else {
					err = publicar(&primero)
				}
				if err != nil {
					falladas.Add(1)
				}
			})
		}
		grupo.Wait()
		if int64(peticiones)-falladas.Load() != version.Load() {
			t.Fatalf("confirmaciones %d; fallidas %d", version.Load(), falladas.Load())
		}
		return falladas.Load()
	}
	if fallidas := ejecutar(false); fallidas != peticiones-1 {
		t.Fatalf("la simulación no reprodujo la carrera: %d fallidas", fallidas)
	}
	if fallidas := ejecutar(true); fallidas != 0 {
		t.Fatalf("con reintento fallaron %d publicaciones", fallidas)
	}
}
