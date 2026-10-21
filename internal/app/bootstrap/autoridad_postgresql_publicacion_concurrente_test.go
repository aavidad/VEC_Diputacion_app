package bootstrap

import (
	"context"
	"errors"
	"reflect"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	dominiovec "vec-diputacion-granada/internal/vec/domain"
)

// publicarAlaVezPrueba lanza las publicaciones a la vez por la misma vía que
// usan las rutas dinámicas (PublicarInstantanea del perfil CT). Un bloqueo
// consultivo retenido por la prueba las hace esperar juntas: las dos toman su
// snapshot serializable antes de que la primera pueda confirmar.
func publicarAlaVezPrueba(
	t *testing.T, ctx context.Context, admin *pgxpool.Pool, s *soporteAltaContratacionTemporalDesarrollo,
	instantaneas ...dominiovec.InstantaneaAutorizacion,
) []error {
	t.Helper()
	autoridad := (&autoridadPostgreSQLContratacionTemporalDesarrollo{pool: admin, soporte: s}).autoridadComun()
	bloqueador, err := admin.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		t.Fatal(err)
	}
	defer bloqueador.Rollback(context.Background())
	if _, err := bloqueador.Exec(ctx, `SELECT pg_catalog.pg_advisory_xact_lock(pg_catalog.hashtextextended($1,0))`,
		autoridad.prefijoBloqueo+s.instantanea.AsignacionPerfil.PerfilActivoRef); err != nil {
		t.Fatal(err)
	}
	resultados := make([]error, len(instantaneas))
	var grupo sync.WaitGroup
	for i, instantanea := range instantaneas {
		grupo.Go(func() { resultados[i] = s.autoridadAsignaciones.PublicarInstantanea(ctx, instantanea) })
	}
	limite := time.NewTimer(10 * time.Second)
	defer limite.Stop()
	for {
		var esperando int
		if err := admin.QueryRow(ctx, `SELECT count(*) FROM pg_catalog.pg_locks
			WHERE locktype='advisory' AND NOT granted`).Scan(&esperando); err != nil {
			t.Fatal(err)
		}
		if esperando >= len(instantaneas) {
			break
		}
		select {
		case <-time.After(10 * time.Millisecond):
		case <-limite.C:
			t.Fatal("las publicaciones no llegaron juntas al bloqueo del perfil")
		}
	}
	if err := bloqueador.Commit(ctx); err != nil {
		t.Fatal(err)
	}
	grupo.Wait()
	return resultados
}

func huellaAsignacionVigentePrueba(t *testing.T, ctx context.Context, admin *pgxpool.Pool, perfil string) string {
	t.Helper()
	var huella string
	if err := admin.QueryRow(ctx, `SELECT asignacion.huella_sha256
		  FROM vec_autorizacion.asignacion_perfil_actual AS vigente
		  JOIN vec_autorizacion.asignacion_perfil AS asignacion
		    ON asignacion.perfil_activo_ref=vigente.perfil_activo_ref AND asignacion.asignacion_ref=vigente.asignacion_ref
		 WHERE vigente.perfil_activo_ref=$1`, perfil).Scan(&huella); err != nil {
		t.Fatal(err)
	}
	return huella
}

// F1-150: dos lecturas simultáneas que preparan la MISMA instantánea en la
// versión actual+1 terminan bien las dos y solo queda una versión nueva.
func TestAutoridadPublicacionConcurrenteMismaInstantaneaPostgreSQL18(t *testing.T) {
	ctx, gobierno, admin := poolesPerfilDinamicoRRHHPostgreSQLPrueba(t)
	s := soporteRRHHPostgreSQLPrueba(t, ctx, gobierno)
	if _, err := asegurarPerfilDinamicoCTDesarrollo(ctx, gobierno, s); err != nil {
		t.Fatal(err)
	}
	perfil := s.instantanea.AsignacionPerfil.PerfilActivoRef
	antes := historiaPerfilPostgreSQLPrueba(t, ctx, admin, perfil)
	solicitada := rolDeExpedientePrueba(s, "expediente:rrhh:f1-150:misma")
	primera, err := s.autoridadAsignaciones.PrepararInstantanea(ctx, solicitada)
	if err != nil {
		t.Fatal(err)
	}
	segunda, err := s.autoridadAsignaciones.PrepararInstantanea(ctx, solicitada)
	if err != nil || !reflect.DeepEqual(primera, segunda) {
		t.Fatalf("las dos lecturas no prepararon la misma instantánea: %v", err)
	}
	for _, err := range publicarAlaVezPrueba(t, ctx, admin, s, primera, segunda) {
		if err != nil {
			t.Fatalf("una lectura con la misma instantánea falló: %v", err)
		}
	}
	despues := historiaPerfilPostgreSQLPrueba(t, ctx, admin, perfil)
	if despues.versiones != antes.versiones+1 || despues.actual != primera.AsignacionPerfil.Referencia() {
		t.Fatalf("publicación duplicada o ausente: antes=%+v después=%+v", antes, despues)
	}
}

// F1-150: si la otra lectura publica una instantánea distinta (otra ruta con
// otro rol o ámbito en el mismo perfil dinámico) en esa misma versión, la
// guarda sigue cerrada: una gana, la otra se rechaza y no altera la historia.
func TestAutoridadPublicacionConcurrenteInstantaneaDistintaPostgreSQL18(t *testing.T) {
	ctx, gobierno, admin := poolesPerfilDinamicoRRHHPostgreSQLPrueba(t)
	s := soporteRRHHPostgreSQLPrueba(t, ctx, gobierno)
	if _, err := asegurarPerfilDinamicoCTDesarrollo(ctx, gobierno, s); err != nil {
		t.Fatal(err)
	}
	perfil := s.instantanea.AsignacionPerfil.PerfilActivoRef
	antes := historiaPerfilPostgreSQLPrueba(t, ctx, admin, perfil)
	a, err := s.autoridadAsignaciones.PrepararInstantanea(ctx, rolDeExpedientePrueba(s, "expediente:rrhh:f1-150:a"))
	if err != nil {
		t.Fatal(err)
	}
	b, err := s.autoridadAsignaciones.PrepararInstantanea(ctx, rolDeExpedientePrueba(s, "expediente:rrhh:f1-150:b"))
	if err != nil {
		t.Fatal(err)
	}
	huellaA, errA := a.AsignacionPerfil.HuellaSHA256()
	huellaB, errB := b.AsignacionPerfil.HuellaSHA256()
	if errA != nil || errB != nil || a.AsignacionPerfil.Referencia() != b.AsignacionPerfil.Referencia() || huellaA == huellaB {
		t.Fatal("el escenario exige dos instantáneas distintas en la misma versión")
	}
	resultados := publicarAlaVezPrueba(t, ctx, admin, s, a, b)
	var ganadora string
	var rechazos int
	for i, err := range resultados {
		switch {
		case err == nil && ganadora == "":
			ganadora = []string{huellaA, huellaB}[i]
		case errors.Is(err, errPostgreSQLContratacionTemporalDesarrolloNoDisponible):
			rechazos++
		default:
			t.Fatalf("resultado inesperado %d: %v", i, err)
		}
	}
	if ganadora == "" || rechazos != 1 {
		t.Fatalf("debía ganar una y rechazarse la otra: %v", resultados)
	}
	despues := historiaPerfilPostgreSQLPrueba(t, ctx, admin, perfil)
	if despues.versiones != antes.versiones+1 || despues.actual != a.AsignacionPerfil.Referencia() ||
		huellaAsignacionVigentePrueba(t, ctx, admin, perfil) != ganadora {
		t.Fatalf("la rechazada alteró la historia: antes=%+v después=%+v", antes, despues)
	}
	// Repetir la rechazada después tampoco la convierte en publicada.
	perdedora := a
	if ganadora == huellaA {
		perdedora = b
	}
	if err := s.autoridadAsignaciones.PublicarInstantanea(ctx, perdedora); !errors.Is(err, errPostgreSQLContratacionTemporalDesarrolloNoDisponible) {
		t.Fatalf("la instantánea distinta se aceptó al repetirla: %v", err)
	}
	if final := historiaPerfilPostgreSQLPrueba(t, ctx, admin, perfil); final != despues {
		t.Fatalf("el rechazo repetido cambió la historia: %+v", final)
	}
}
