package bootstrap

import (
	"context"
	"errors"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	dominiovec "vec-diputacion-granada/internal/vec/domain"
)

// Los dos intentos toman un snapshot serializable antes de soltar el bloqueo.
// El segundo pierde la carrera 40001 y debe repetir la transacción completa.
func TestAutoridadPublicacion40001YReplayPublicadoPostgreSQL18(t *testing.T) {
	ctx, gobierno, admin := poolesPerfilDinamicoRRHHPostgreSQLPrueba(t)
	soporte := soporteRRHHPostgreSQLPrueba(t, ctx, gobierno)
	if _, err := asegurarPerfilDinamicoCTDesarrollo(ctx, gobierno, soporte); err != nil {
		t.Fatal(err)
	}
	autoridad := (&autoridadPostgreSQLContratacionTemporalDesarrollo{
		pool: gobierno, soporte: soporte,
	}).autoridadComun()
	perfil := soporte.instantanea.AsignacionPerfil.PerfilActivoRef
	publicada, encontrada, err := leerInstantaneaPublicadaPostgreSQLDesarrollo(ctx, admin, perfil)
	if err != nil || !encontrada {
		t.Fatalf("instantánea de origen: %v", err)
	}
	preimagen := publicada.instantanea
	candidata, err := autoridad.prepararInstantanea(ctx,
		rolDeExpedientePrueba(soporte, "expediente:rrhh:serializacion-minima"), false)
	if err != nil || candidata.AsignacionPerfil.Version != preimagen.AsignacionPerfil.Version+1 {
		t.Fatalf("preparación: %v", err)
	}

	bloqueador, err := gobierno.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		t.Fatal(err)
	}
	defer bloqueador.Rollback(context.Background())
	if _, err := bloqueador.Exec(ctx,
		`SELECT pg_catalog.pg_advisory_xact_lock(pg_catalog.hashtextextended($1,0))`,
		autoridad.prefijoBloqueo+perfil); err != nil {
		t.Fatal(err)
	}
	resultados := make(chan error, 2)
	var grupo sync.WaitGroup
	for range 2 {
		grupo.Go(func() {
			resultados <- autoridad.publicarInstantaneaDesdePreimagen(ctx, candidata, preimagen)
		})
	}
	limite := time.NewTimer(10 * time.Second)
	defer limite.Stop()
	ticker := time.NewTicker(10 * time.Millisecond)
	defer ticker.Stop()
	for {
		var esperando int
		if err := admin.QueryRow(ctx, `SELECT count(*) FROM pg_catalog.pg_locks
			WHERE locktype='advisory' AND NOT granted`).Scan(&esperando); err != nil {
			t.Fatal(err)
		}
		if esperando >= 2 {
			break
		}
		select {
		case <-ticker.C:
		case <-limite.C:
			t.Fatal("los dos publicadores no alcanzaron el bloqueo consultivo")
		case <-ctx.Done():
			t.Fatal(ctx.Err())
		}
	}
	if err := bloqueador.Commit(ctx); err != nil {
		t.Fatal(err)
	}
	grupo.Wait()
	close(resultados)
	for err := range resultados {
		if err != nil {
			t.Fatalf("la publicación no se recuperó: %v", err)
		}
	}
	historia := historiaPerfilPostgreSQLPrueba(t, ctx, admin, perfil)
	if historia.versiones != 2 || historia.actual != candidata.AsignacionPerfil.Referencia() {
		t.Fatalf("publicación duplicada o ausente: %+v", historia)
	}

	// El arranque puede entregar otra preimagen válida del mismo perfil. El
	// replay ya publicado sigue autorizado por la candidata vigente exacta.
	preimagenArranque := clonarInstantaneaAutorizacionPostgreSQLDesarrollo(preimagen)
	preimagenArranque.AsignacionPerfil.EmitidaPor = "origen_arranque_prueba"
	if err := preimagenArranque.Validar(); err != nil {
		t.Fatal(err)
	}
	if err := autoridad.publicarInstantaneaDesdePreimagen(ctx, candidata, preimagenArranque); err != nil {
		t.Fatalf("el replay publicado exigió una preimagen que no le corresponde: %v", err)
	}
	candidataAjena := clonarInstantaneaAutorizacionPostgreSQLDesarrollo(candidata)
	candidataAjena.AsignacionPerfil.Ambitos = append(candidataAjena.AsignacionPerfil.Ambitos,
		dominiovec.AmbitoPerfil{Clave: "expediente_ajeno_ref", Valores: []string{"expediente:rrhh:ajeno"}})
	if err := candidataAjena.Validar(); err != nil {
		t.Fatal(err)
	}
	if err := autoridad.publicarInstantaneaDesdePreimagen(ctx, candidataAjena, preimagen); !errors.Is(err, errPostgreSQLContratacionTemporalDesarrolloNoDisponible) {
		t.Fatalf("la candidata de otro ámbito no se rechazó: %v", err)
	}
	if final := historiaPerfilPostgreSQLPrueba(t, ctx, admin, perfil); final != historia {
		t.Fatalf("un replay o rechazo cambió historia: antes=%+v después=%+v", historia, final)
	}
}

// Esta prueba alcanza exactamente Exec(pg_advisory_xact_lock): el interbloqueo
// PostgreSQL real produce 40P01 antes de leer las tablas. Requiere una base
// vacía desechable con el rol NOLOGIN propietario; no instala SQL de VEC ni
// acredita provisión, permisos, publicación o la guarda de preimagen.
func TestAutoridadErrorBloqueoConsultivoPostgreSQL18(t *testing.T) {
	if os.Getenv("VEC_AUTORIDAD_BLOQUEO_PG_MINIMO") != "1" {
		t.Skip("requiere PostgreSQL 18 vacío desechable para interbloqueo consultivo")
	}
	for _, operacion := range []string{"preparar", "publicar"} {
		t.Run(operacion, func(t *testing.T) {
			ctx, cancelar := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancelar()
			cfg, err := pgxpool.ParseConfig(os.Getenv("VEC_AUTORIDAD_BLOQUEO_PG_DSN"))
			if err != nil {
				t.Fatal(err)
			}
			erroresSQL := make(chan string, 4)
			cfg.ConnConfig.OnPgError = func(_ *pgconn.PgConn, err *pgconn.PgError) bool {
				erroresSQL <- err.Code
				return err.Severity != "FATAL"
			}
			cfg.MaxConns = 1
			cfg.MinConns = 0
			cfg.ConnConfig.RuntimeParams["deadlock_timeout"] = "50ms"
			pool, err := pgxpool.NewWithConfig(ctx, cfg)
			if err != nil {
				t.Fatal(err)
			}
			defer pool.Close()
			soporte, _, _ := escenarioAutorizacionCoberturaDesarrolloPrueba(t)
			autoridad := autoridadPostgreSQLDesarrollo{pool: pool, vinculo: soporte.contexto.Vinculo,
				prefijoBloqueo: "prueba:propagacion:" + operacion + ":", actoControlRol: "acto:prueba:control",
				actoAsignacion: "acto:prueba:asignacion", actoSesion: "acto:prueba:sesion"}
			var pid int
			claveAuxiliar := autoridad.prefijoBloqueo + "auxiliar"
			if err := pool.QueryRow(ctx, `SELECT pg_backend_pid()`).Scan(&pid); err != nil {
				t.Fatal(err)
			}
			if _, err := pool.Exec(ctx, `SELECT pg_advisory_lock(hashtextextended($1,0))`, claveAuxiliar); err != nil {
				t.Fatal(err)
			}
			bloqueador, err := pgx.Connect(ctx, os.Getenv("VEC_AUTORIDAD_BLOQUEO_PG_DSN"))
			if err != nil {
				t.Fatal(err)
			}
			defer bloqueador.Close(context.Background())
			if _, err := bloqueador.Exec(ctx, `SET deadlock_timeout='5s'`); err != nil {
				t.Fatal(err)
			}
			tx, err := bloqueador.Begin(ctx)
			if err != nil {
				t.Fatal(err)
			}
			defer tx.Rollback(context.Background())
			if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtextextended($1,0))`,
				autoridad.prefijoBloqueo+soporte.instantanea.AsignacionPerfil.PerfilActivoRef); err != nil {
				t.Fatal(err)
			}
			resultado := make(chan error, 1)
			go func() {
				if operacion == "preparar" {
					_, err := autoridad.prepararInstantaneaUnaVez(ctx, soporte.instantanea, true)
					resultado <- err
				} else {
					resultado <- autoridad.publicarInstantaneaConPreimagenUnaVez(ctx, soporte.instantanea, nil)
				}
			}()
			// Barrera real: no se inicia el segundo lado del ciclo hasta que el método
			// esté esperando el bloqueo del perfil en la conexión identificada.
			for {
				var espera bool
				if err := bloqueador.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM pg_locks
      WHERE pid=$1 AND locktype='advisory' AND NOT granted)`, pid).Scan(&espera); err != nil {
					t.Fatal(err)
				}
				if espera {
					break
				}
				select {
				case err := <-resultado:
					t.Fatalf("el método no alcanzó el bloqueo: %v", err)
				case <-ctx.Done():
					t.Fatal(ctx.Err())
				case <-time.After(time.Millisecond):
				}
			}
			otroLado := make(chan error, 1)
			go func() {
				_, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtextextended($1,0))`, claveAuxiliar)
				otroLado <- err
			}()
			var fallo error
			select {
			case fallo = <-resultado:
			case <-ctx.Done():
				t.Fatal(ctx.Err())
			}
			select {
			case codigo := <-erroresSQL:
				if codigo != "40P01" {
					t.Fatalf("error SQL distinto del interbloqueo: %s", codigo)
				}
			case <-ctx.Done():
				t.Fatal(ctx.Err())
			}
			// Liberar la sesión auxiliar permite terminar el publicador bloqueador.
			if _, err := pool.Exec(ctx, `SELECT pg_advisory_unlock(hashtextextended($1,0))`, claveAuxiliar); err != nil {
				t.Fatal(err)
			}
			select {
			case err := <-otroLado:
				if err != nil {
					t.Fatalf("el bloqueador fue víctima: %v", err)
				}
			case <-ctx.Done():
				t.Fatal(ctx.Err())
			}
			if err := tx.Rollback(ctx); err != nil {
				t.Fatal(err)
			}
			if !errors.Is(fallo, errPostgreSQLContratacionTemporalDesarrolloNoDisponible) {
				t.Fatalf("se perdió el centinela: %v", fallo)
			}
			if !falloReintentableSerializacionCTDesarrollo(fallo) {
				t.Fatalf("el error real del bloqueo perdió su marca reintentable: %v", fallo)
			}
		})
	}
}
