package postgres

import (
	"context"
	"errors"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"vec-diputacion-granada/internal/vec/domain"
	"vec-diputacion-granada/internal/vec/ports"
)

func TestIntegracionPostgreSQLContextoActorV2(t *testing.T) {
	dsn := os.Getenv("VEC_CONTEXTO_ACTOR_V2_POSTGRES_DSN")
	if dsn == "" {
		t.Skip("requiere runner PostgreSQL 18 de contexto actor V1")
	}
	ctx, cancelar := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancelar()
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	adaptador, err := NuevoResolutorRegistroContextoActorPostgreSQLV2(ctx, pool)
	if err != nil {
		t.Fatalf("pool runtime acreditado rechazado: %v", err)
	}
	ref := func(prefijo, relleno string) string { return prefijo + strings.Repeat(relleno, 24) }
	solicitud := ports.SolicitudResolucionRegistroContextoActorV2{
		OperacionRef: ref("oca_integracion_", "i"),
		Contexto: domain.SolicitudContextoActor{
			Cuenta: domain.CuentaAutenticadaContextoActor{
				CuentaRef: "cta_sintetica_aaaaaaaaaaaaaaaaaaaaaaaa",
				Metodo:    domain.AuthMethodCertificate, Garantia: domain.AuthAssuranceHigh,
			},
			PerfilActivoRef: "prf_sintetico_cccccccccccccccccccccccc",
		},
		SolicitadoEn: time.Now().UTC().Truncate(time.Microsecond),
	}
	confirmacion, err := adaptador.ResolverYRegistrarContextoActorV2(ctx, solicitud)
	if err != nil || confirmacion.ValidarParaProductiva(solicitud) != nil {
		t.Fatalf("snapshot durable sintetico rechazado: %v", err)
	}
	manifiesto, err := domain.RehidratarManifiestoProcedenciaContextoActorV1(
		confirmacion.ManifiestoProcedenciaCanonico,
	)
	if err != nil || manifiesto.AutoridadEfectiva != domain.AutoridadProcedenciaContextoActorMaestraAcreditadaV1 ||
		manifiesto.Cuenta.Version != 2 || manifiesto.Persona.Version != 2 ||
		manifiesto.Perfil.Version != 2 || manifiesto.Contexto.Version != 2 {
		t.Fatalf("manifiesto maestro sintetico no fue exacto: %#v err=%v", manifiesto, err)
	}
	if confirmacion.Contexto.PersonaRef != "per_sintetica_bbbbbbbbbbbbbbbbbbbbbbbb" ||
		confirmacion.Contexto.Instantanea.CuentaVersion != 2 ||
		len(confirmacion.Contexto.Instantanea.Vinculos) != 2 ||
		confirmacion.Contexto.Principal.DisplayName != "" || confirmacion.Contexto.Principal.Email != "" ||
		len(confirmacion.Contexto.Principal.Roles) != 0 || len(confirmacion.Contexto.Principal.Permissions) != 0 {
		t.Fatal("el snapshot no fue completo o incorporo claims prohibidos")
	}

	// La misma operacion y solicitud recuperan el recibo ya confirmado aunque
	// la invocacion nueva haya generado material rca_ provisional distinto.
	repetida, err := adaptador.ResolverYRegistrarContextoActorV2(ctx, solicitud)
	if err != nil || repetida.RegistroContextoRef != confirmacion.RegistroContextoRef ||
		!repetida.ResueltoEnAutoritativo.Equal(confirmacion.ResueltoEnAutoritativo) {
		t.Fatal("la operacion idempotente exacta no recupero el recibo original")
	}
	// La misma operacion con cualquier contenido diferente colisiona cerrada.
	colision := solicitud
	colision.SolicitadoEn = colision.SolicitadoEn.Add(time.Microsecond)
	_, err = adaptador.ResolverYRegistrarContextoActorV2(ctx, colision)
	if !errors.Is(err, ports.ErrResolutorRegistroContextoActorNoDisponible) {
		t.Fatal("una colision de operacion con solicitud distinta no fallo cerrada")
	}

	// Dos altas ausentes y concurrentes se serializan por oca_. Si la segunda
	// conservaba un snapshot anterior, el adaptador repite con los mismos
	// oca_/rca_ y ambas recuperan un unico recibo durable.
	concurrente := solicitud
	concurrente.OperacionRef = ref("oca_concurrente_", "j")
	concurrente.SolicitadoEn = time.Now().UTC().Truncate(time.Microsecond)
	type resultado struct {
		confirmacion ports.ConfirmacionRegistroContextoActorV2
		err          error
	}
	canal := make(chan resultado, 2)
	inicio := make(chan struct{})
	for i := 0; i < 2; i++ {
		go func() {
			<-inicio
			c, e := adaptador.ResolverYRegistrarContextoActorV2(ctx, concurrente)
			canal <- resultado{confirmacion: c, err: e}
		}()
	}
	close(inicio)
	primero, segundo := <-canal, <-canal
	if primero.err != nil || segundo.err != nil ||
		primero.confirmacion.RegistroContextoRef == "" ||
		primero.confirmacion.RegistroContextoRef != segundo.confirmacion.RegistroContextoRef ||
		!primero.confirmacion.ResueltoEnAutoritativo.Equal(segundo.confirmacion.ResueltoEnAutoritativo) {
		t.Fatalf("carrera idempotente no convergio: primero=%v segundo=%v", primero.err, segundo.err)
	}
}

func TestReconciliacionPostgreSQLContextoActorV2EsperaFinalizacionConcurrente(t *testing.T) {
	dsn := os.Getenv("VEC_CONTEXTO_ACTOR_V2_POSTGRES_DSN")
	if dsn == "" {
		t.Skip("requiere runner PostgreSQL 18 de contexto actor V2")
	}
	ctx, cancelar := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancelar()
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	ref := func(prefijo, relleno string) string { return prefijo + strings.Repeat(relleno, 24) }
	solicitud := ports.SolicitudResolucionRegistroContextoActorV2{
		OperacionRef: ref("oca_reconciliacion_", "k"),
		Contexto: domain.SolicitudContextoActor{
			Cuenta: domain.CuentaAutenticadaContextoActor{
				CuentaRef: "cta_sintetica_aaaaaaaaaaaaaaaaaaaaaaaa",
				Metodo:    domain.AuthMethodCertificate, Garantia: domain.AuthAssuranceHigh,
			},
			PerfilActivoRef: "prf_sintetico_cccccccccccccccccccccccc",
		},
		SolicitadoEn: time.Now().UTC().Truncate(time.Microsecond),
	}
	reciboRef := ref("rca_reconciliacion_", "l")
	argumentos := argumentosContextoActorPostgreSQL(solicitud, reciboRef)

	escritura, err := pool.BeginTx(ctx, pgx.TxOptions{
		IsoLevel: pgx.Serializable, AccessMode: pgx.ReadWrite,
	})
	if err != nil {
		t.Fatal(err)
	}
	defer revertirContextoActorPostgreSQL(escritura)
	if err = prepararTransaccionContextoActorPostgreSQL(ctx, escritura); err != nil {
		t.Fatal(err)
	}
	antesCommit, err := consultarRespuestaContextoActor(
		ctx, escritura, consultaResolverContextoActorV2, argumentos,
	)
	if err != nil {
		t.Fatalf("preparar escritura concurrente: %v", err)
	}

	type resultadoReconciliacion struct {
		respuesta respuestaContextoActorPostgreSQL
		err       error
	}
	pidReconciliacion := make(chan int32, 1)
	resultado := make(chan resultadoReconciliacion, 1)
	go func() {
		tx, e := pool.BeginTx(ctx, pgx.TxOptions{
			IsoLevel: pgx.ReadCommitted, AccessMode: pgx.ReadWrite,
		})
		if e != nil {
			resultado <- resultadoReconciliacion{err: e}
			return
		}
		defer revertirContextoActorPostgreSQL(tx)
		if e = prepararTransaccionContextoActorPostgreSQL(ctx, tx); e != nil {
			resultado <- resultadoReconciliacion{err: e}
			return
		}
		var pid int32
		if e = tx.QueryRow(ctx, `SELECT pg_catalog.pg_backend_pid()`).Scan(&pid); e != nil {
			resultado <- resultadoReconciliacion{err: e}
			return
		}
		pidReconciliacion <- pid
		respuesta, e := consultarRespuestaContextoActor(
			ctx, tx, consultaReconciliarContextoActorV2, argumentos,
		)
		if e == nil {
			e = tx.Commit(ctx)
		}
		resultado <- resultadoReconciliacion{respuesta: respuesta, err: e}
	}()
	select {
	case pid := <-pidReconciliacion:
		esperarAdvisoryContextoActorV2(t, ctx, pool, pid)
	case temprano := <-resultado:
		t.Fatalf("reconciliacion fallo antes de consultar el lock: %v", temprano.err)
	case <-ctx.Done():
		t.Fatalf("reconciliacion no alcanzo el lock: %v", ctx.Err())
	}
	select {
	case temprano := <-resultado:
		t.Fatalf("reconciliacion concluyo mientras pg_locks la mostraba esperando: %v", temprano.err)
	default:
	}
	if err = escritura.Commit(ctx); err != nil {
		t.Fatalf("confirmar escritura concurrente: %v", err)
	}
	select {
	case reconciliada := <-resultado:
		if reconciliada.err != nil ||
			!respuestasContextoActorIguales(antesCommit, reconciliada.respuesta) {
			t.Fatalf("reconciliacion no observo el COMMIT final: %#v, %v", reconciliada.respuesta, reconciliada.err)
		}
		if _, err = confirmarRespuestaContextoActor(solicitud, reconciliada.respuesta); err != nil {
			t.Fatalf("recibo reconciliado no fue V2 exacto: %v", err)
		}
	case <-ctx.Done():
		t.Fatalf("reconciliacion no desperto tras COMMIT: %v", ctx.Err())
	}
}

func esperarAdvisoryContextoActorV2(
	t *testing.T,
	ctx context.Context,
	pool *pgxpool.Pool,
	pid int32,
) {
	t.Helper()
	ticker := time.NewTicker(5 * time.Millisecond)
	defer ticker.Stop()
	for {
		var esperando bool
		err := pool.QueryRow(ctx, `
			SELECT EXISTS (
			  SELECT 1 FROM pg_catalog.pg_locks
			   WHERE pid=$1 AND locktype='advisory' AND granted IS FALSE
			)`, pid).Scan(&esperando)
		if err != nil {
			t.Fatalf("observar lock de reconciliacion: %v", err)
		}
		if esperando {
			return
		}
		select {
		case <-ticker.C:
		case <-ctx.Done():
			t.Fatalf("reconciliacion no quedo esperando el advisory: %v", ctx.Err())
		}
	}
}
