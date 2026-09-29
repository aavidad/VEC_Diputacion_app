package postgres

import (
	"context"
	"errors"
	"fmt"
	"os"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestReintentoSerializableRepiteSoloCarreras(t *testing.T) {
	ctx := context.Background()
	for _, codigo := range []string{"40001", "40P01"} {
		n := 0
		err := ejecutarConReintentoSerializable(ctx, func() error {
			n++
			if n < 3 {
				return &pgconn.PgError{Code: codigo}
			}
			return nil
		})
		if err != nil || n != 3 {
			t.Fatalf("%s: err=%v intentos=%d", codigo, err, n)
		}
	}
	n := 0
	otro := &pgconn.PgError{Code: "42501"}
	if err := ejecutarConReintentoSerializable(ctx, func() error { n++; return otro }); !errors.Is(err, otro) || n != 1 {
		t.Fatalf("un error no reintentable se repitió: err=%v intentos=%d", err, n)
	}
	n = 0
	if err := ejecutarConReintentoSerializable(ctx, func() error { n++; return &pgconn.PgError{Code: "40001"} }); err == nil || n != intentosTransaccionSerializable {
		t.Fatalf("el límite de intentos no se respeta: err=%v intentos=%d", err, n)
	}
}

func TestReintentoSerializableRespetaElContexto(t *testing.T) {
	ctx, cancelar := context.WithCancel(context.Background())
	n := 0
	err := ejecutarConReintentoSerializable(ctx, func() error {
		n++
		cancelar()
		return &pgconn.PgError{Code: "40001"}
	})
	if err == nil || n != 1 {
		t.Fatalf("con el contexto cancelado no debe repetirse: err=%v intentos=%d", err, n)
	}
}

// Las peticiones simultáneas de una página (bandeja, incorporaciones y
// cancelaciones) se pisan en la misma fila de control y terminan todas
// bien: como en PostgreSQL, cada intento fija su instantánea, espera el
// bloqueo de la fila y aborta con 40001 si otra la cambió entretanto.
func TestReintentoSerializableConcurrenteSinFallos(t *testing.T) {
	var fila sync.Mutex
	var version atomic.Int64
	ctx, cancelar := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancelar()
	const simultaneas = 3
	var wg sync.WaitGroup
	var fallos, carreras atomic.Int32
	salida := make(chan struct{})
	for range simultaneas {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-salida
			err := ejecutarConReintentoSerializable(ctx, func() error {
				instantanea := version.Load()
				time.Sleep(2 * time.Millisecond)
				fila.Lock()
				defer fila.Unlock()
				if version.Load() != instantanea {
					carreras.Add(1)
					return &pgconn.PgError{Code: "40001"}
				}
				time.Sleep(time.Millisecond)
				version.Add(1)
				return nil
			})
			if err != nil {
				fallos.Add(1)
			}
		}()
	}
	close(salida)
	wg.Wait()
	if fallos.Load() != 0 || version.Load() != simultaneas || carreras.Load() == 0 {
		t.Fatalf("fallos=%d versiones=%d carreras=%d", fallos.Load(), version.Load(), carreras.Load())
	}
}

// Reproduce en PostgreSQL real la carrera de las funciones de consumo V3:
// dos transacciones SERIALIZABLE bloquean y actualizan la misma fila de
// control. Sin reintento una de las dos recibe 40001; con él, ambas terminan.
// Solo corre con VEC_PG_REINTENTO_SERIALIZABLE_DSN (PostgreSQL desechable).
func TestReintentoSerializablePostgreSQLReal(t *testing.T) {
	dsn := os.Getenv("VEC_PG_REINTENTO_SERIALIZABLE_DSN")
	if dsn == "" {
		t.Skip("PostgreSQL desechable no solicitado")
	}
	ctx, cancelar := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancelar()
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	esquema := fmt.Sprintf("prueba_reintento_%d", time.Now().UnixNano())
	if _, err := pool.Exec(ctx, "CREATE SCHEMA "+esquema+"; CREATE TABLE "+esquema+".control(id boolean PRIMARY KEY, secuencia bigint NOT NULL); INSERT INTO "+esquema+".control VALUES (true, 0)"); err != nil {
		t.Fatal(err)
	}
	defer func() { _, _ = pool.Exec(context.Background(), "DROP SCHEMA "+esquema+" CASCADE") }()
	consumir := func(listo *sync.WaitGroup, soltar <-chan struct{}) func() error {
		primera := true
		return func() error {
			tx, err := iniciarTransaccionAltaCandidata(ctx, pool)
			if err != nil {
				return err
			}
			defer revertirTransaccion(tx)
			// Fija la instantánea antes del bloqueo, como las funciones reales.
			if _, err := tx.Exec(ctx, "SELECT count(*) FROM "+esquema+".control"); err != nil {
				return err
			}
			if primera {
				primera = false
				listo.Done()
				<-soltar
			}
			if _, err := tx.Exec(ctx, "SELECT secuencia FROM "+esquema+".control WHERE id FOR UPDATE"); err != nil {
				return err
			}
			if _, err := tx.Exec(ctx, "UPDATE "+esquema+".control SET secuencia = secuencia + 1 WHERE id"); err != nil {
				return err
			}
			return tx.Commit(ctx)
		}
	}
	for _, conReintento := range []bool{false, true} {
		var listo, fin sync.WaitGroup
		soltar := make(chan struct{})
		errores := make([]error, 2)
		for i := range 2 {
			listo.Add(1)
			fin.Add(1)
			intento := consumir(&listo, soltar)
			go func() {
				defer fin.Done()
				if conReintento {
					errores[i] = ejecutarConReintentoSerializable(ctx, intento)
				} else {
					errores[i] = intento()
				}
			}()
		}
		listo.Wait()
		close(soltar)
		fin.Wait()
		carreras := 0
		for _, err := range errores {
			if err != nil && errorPostgreSQLReintentable(err) {
				carreras++
			} else if err != nil {
				t.Fatalf("error inesperado: %v", err)
			}
		}
		if !conReintento && carreras != 1 {
			t.Fatalf("sin reintento se esperaba una carrera 40001 y hubo %d", carreras)
		}
		if conReintento && carreras != 0 {
			t.Fatalf("con reintento quedaron %d carreras sin resolver", carreras)
		}
	}
	var secuencia int64
	if err := pool.QueryRow(ctx, "SELECT secuencia FROM "+esquema+".control").Scan(&secuencia); err != nil || secuencia != 3 {
		t.Fatalf("secuencia=%d err=%v; se esperaban 3 consumos confirmados", secuencia, err)
	}
}
