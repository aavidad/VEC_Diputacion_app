package bootstrap

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"

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
