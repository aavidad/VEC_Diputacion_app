package postgres

import (
	"context"
	"errors"
	"math"
	"os"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"vec-diputacion-granada/internal/vec/domain"
	"vec-diputacion-granada/internal/vec/ports"
)

func TestAcreditadorUsoRegistroContextoActorPostgreSQLV2IntegracionPostgreSQL18(t *testing.T) {
	dsnRuntime := os.Getenv("VEC_CONTEXTO_ACTOR_V2_POSTGRES_DSN")
	dsnAcreditador := os.Getenv("VEC_CONTEXTO_ACTOR_ACREDITADOR_V2_POSTGRES_DSN")
	if dsnRuntime == "" || dsnAcreditador == "" {
		t.Skip("requiere PostgreSQL 18 efimero de deploy/postgresql/contexto_actor_v1/probar_integracion.sh")
	}
	ctx, cancelar := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancelar()
	poolRuntime, err := pgxpool.New(ctx, dsnRuntime)
	if err != nil {
		t.Fatal(err)
	}
	defer poolRuntime.Close()
	poolAcreditador, err := pgxpool.New(ctx, dsnAcreditador)
	if err != nil {
		t.Fatal(err)
	}
	defer poolAcreditador.Close()
	var versionMayor int
	if err = poolRuntime.QueryRow(ctx, `
		SELECT current_setting('server_version_num')::integer / 10000`).Scan(&versionMayor); err != nil {
		t.Fatalf("consultar version PostgreSQL: %v", err)
	}
	if versionMayor != 18 {
		t.Fatalf("requiere PostgreSQL 18; servidor conectado: PostgreSQL %d", versionMayor)
	}

	orden := registrarOrdenAcreditacionIntegracion(t, ctx, poolRuntime)

	t.Run("misma_transaccion_serializable_dos_acreditaciones_y_rollback", func(t *testing.T) {
		tx := comenzarTransaccionAcreditacionIntegracion(t, ctx, poolAcreditador)
		defer func() { _ = tx.Rollback(context.Background()) }()
		acreditador, err := NuevoAcreditadorUsoRegistroContextoActorPostgreSQLV2EnTransaccion(tx)
		if err != nil {
			t.Fatalf("crear acreditador: %v", err)
		}
		primera, err := acreditador.AcreditarUsoRegistroContextoActorV2(ctx, orden)
		if err != nil {
			t.Fatalf("primera acreditacion: %v", err)
		}
		segunda, err := acreditador.AcreditarUsoRegistroContextoActorV2(ctx, orden)
		if err != nil {
			t.Fatalf("segunda acreditacion: %v", err)
		}
		if segunda.Before(primera) {
			t.Fatalf("reloj PostgreSQL retrocedio: primera=%s segunda=%s", primera, segunda)
		}
		if err = tx.Rollback(ctx); err != nil {
			t.Fatalf("rollback: %v", err)
		}
		if _, err = acreditador.AcreditarUsoRegistroContextoActorV2(ctx, orden); err == nil ||
			!errors.Is(err, ports.ErrAcreditacionUsoRegistroContextoActorV2Denegada) {
			t.Fatalf("transaccion revertida reutilizable: %v", err)
		}
	})

	t.Run("null_real_ante_dato_alterado", func(t *testing.T) {
		tx := comenzarTransaccionAcreditacionIntegracion(t, ctx, poolAcreditador)
		defer func() { _ = tx.Rollback(context.Background()) }()
		acreditador, err := nuevoAcreditadorUsoRegistroContextoActorPostgreSQLV2EnTransaccion(
			&consultorAcreditacionAlteradaIntegracion{tx: tx},
		)
		if err != nil {
			t.Fatal(err)
		}
		_, err = acreditador.AcreditarUsoRegistroContextoActorV2(ctx, orden)
		if !errors.Is(err, ports.ErrAcreditacionUsoRegistroContextoActorV2Denegada) ||
			errors.Is(err, ports.ErrAcreditadorUsoRegistroContextoActorV2NoDisponible) {
			t.Fatalf("huella alterada no produjo NULL limpio: %v", err)
		}
	})

	t.Run("cuatro_uint64_max_texto_a_numeric", func(t *testing.T) {
		ordenMaxima := ordenAcreditacionTodasVersionesMaximasIntegracion(t)
		tx := comenzarTransaccionAcreditacionIntegracion(t, ctx, poolAcreditador)
		defer func() { _ = tx.Rollback(context.Background()) }()
		acreditador, err := nuevoAcreditadorUsoRegistroContextoActorPostgreSQLV2EnTransaccion(
			&consultorCodecNumericAcreditacionIntegracion{t: t, tx: tx},
		)
		if err != nil {
			t.Fatal(err)
		}
		if _, err = acreditador.AcreditarUsoRegistroContextoActorV2(ctx, ordenMaxima); err != nil {
			t.Fatalf("codec text -> numeric para MaxUint64: %v", err)
		}
	})

	t.Run("cancelacion_durante_scan_bloqueado", func(t *testing.T) {
		probarCancelacionAcreditacionBloqueadaIntegracion(t, ctx, poolAcreditador, orden)
	})
}

func registrarOrdenAcreditacionIntegracion(
	t *testing.T,
	ctx context.Context,
	pool *pgxpool.Pool,
) ports.OrdenAcreditacionUsoRegistroContextoActorV2 {
	t.Helper()
	resolutor, err := NuevoResolutorRegistroContextoActorPostgreSQLV2(ctx, pool)
	if err != nil {
		t.Fatalf("crear resolutor: %v", err)
	}
	solicitada := time.Now().UTC().Truncate(time.Microsecond)
	solicitud := ports.SolicitudResolucionRegistroContextoActorV2{
		OperacionRef: "oca_acreditador_go_integracion_000000000001",
		Contexto: domain.SolicitudContextoActor{
			Cuenta: domain.CuentaAutenticadaContextoActor{
				CuentaRef: "cta_sintetica_aaaaaaaaaaaaaaaaaaaaaaaa",
				Metodo:    domain.AuthMethodCertificate,
				Garantia:  domain.AuthAssuranceHigh,
			},
			PerfilActivoRef: "prf_sintetico_cccccccccccccccccccccccc",
		},
		SolicitadoEn: solicitada,
	}
	confirmacion, err := resolutor.ResolverYRegistrarContextoActorV2(ctx, solicitud)
	if err != nil {
		t.Fatalf("registrar contexto para acreditacion: %v", err)
	}
	resultado := domain.ResultadoContextoActorRegistradoV2{
		RegistroContextoRef:               confirmacion.RegistroContextoRef,
		Contexto:                          confirmacion.Contexto,
		RepresentacionCanonica:            confirmacion.RepresentacionCanonica,
		HuellaSHA256:                      confirmacion.HuellaSHA256,
		ManifiestoProcedenciaCanonico:     confirmacion.ManifiestoProcedenciaCanonico,
		ManifiestoProcedenciaHuellaSHA256: confirmacion.ManifiestoProcedenciaHuellaSHA256,
		AutoridadEfectiva:                 confirmacion.AutoridadEfectiva,
		ResueltoEnAutoritativo:            confirmacion.ResueltoEnAutoritativo,
	}
	emitida := time.Now().UTC().Truncate(time.Microsecond)
	if emitida.Before(confirmacion.ResueltoEnAutoritativo) {
		emitida = confirmacion.ResueltoEnAutoritativo
	}
	validaHasta := emitida.Add(time.Minute)
	orden, err := ports.NuevaOrdenAcreditacionUsoRegistroContextoActorV2(
		resultado, emitida, validaHasta,
	)
	if err != nil {
		t.Fatalf("crear orden acreditable: %v", err)
	}
	return orden
}

func comenzarTransaccionAcreditacionIntegracion(
	t *testing.T,
	ctx context.Context,
	pool *pgxpool.Pool,
) pgx.Tx {
	t.Helper()
	tx, err := pool.BeginTx(ctx, pgx.TxOptions{
		IsoLevel: pgx.Serializable, AccessMode: pgx.ReadWrite,
	})
	if err != nil {
		t.Fatalf("begin serializable: %v", err)
	}
	return tx
}

type consultorAcreditacionAlteradaIntegracion struct{ tx pgx.Tx }

func (c *consultorAcreditacionAlteradaIntegracion) QueryRow(
	ctx context.Context,
	consulta string,
	argumentos ...any,
) pgx.Row {
	alterados := append([]any(nil), argumentos...)
	alterados[2] = strings.Repeat("0", 64)
	return c.tx.QueryRow(ctx, consulta, alterados...)
}

// Las tablas aceptan numeric hasta MaxUint64, pero sembrar las cuatro
// versiones maximas requeriria otra historia completa y otro recibo canonico.
// Este wrapper temporal conserva el camino del adaptador y hace que PostgreSQL
// decodifique exactamente sus cuatro strings como numeric, sin tocar runtime.
type consultorCodecNumericAcreditacionIntegracion struct {
	t  *testing.T
	tx pgx.Tx
}

func (c *consultorCodecNumericAcreditacionIntegracion) QueryRow(
	ctx context.Context,
	_ string,
	argumentos ...any,
) pgx.Row {
	c.t.Helper()
	for _, indice := range []int{6, 8, 10, 12} {
		if valor, ok := argumentos[indice].(string); !ok || valor != "18446744073709551615" {
			c.t.Fatalf("parametro %d no fue texto decimal MaxUint64: %#v", indice+1, argumentos[indice])
		}
	}
	return c.tx.QueryRow(ctx, `
		SELECT CASE WHEN $1::numeric = 18446744073709551615::numeric
		                  AND $2::numeric = 18446744073709551615::numeric
		                  AND $3::numeric = 18446744073709551615::numeric
		                  AND $4::numeric = 18446744073709551615::numeric
		            THEN $5::timestamptz + interval '1 microsecond'
		       END`, argumentos[6], argumentos[8], argumentos[10], argumentos[12], argumentos[15])
}

func ordenAcreditacionTodasVersionesMaximasIntegracion(
	t *testing.T,
) ports.OrdenAcreditacionUsoRegistroContextoActorV2 {
	t.Helper()
	_, resultado, emitida, validaHasta := ordenAcreditacionPostgreSQLV2Prueba(t, math.MaxUint64)
	resultado.Contexto.Instantanea.PersonaVersion = math.MaxUint64
	resultado.Contexto.Instantanea.PerfilVersion = math.MaxUint64
	resultado.Contexto.Instantanea.VinculoVersion = math.MaxUint64

	manifiesto, err := domain.RehidratarManifiestoProcedenciaContextoActorV1(
		resultado.ManifiestoProcedenciaCanonico,
	)
	if err != nil {
		t.Fatal(err)
	}
	manifiesto.Cuenta.Version = math.MaxUint64
	manifiesto.Persona.Version = math.MaxUint64
	manifiesto.Perfil.Version = math.MaxUint64
	manifiesto.Contexto.Version = math.MaxUint64
	resultado.RepresentacionCanonica, err = resultado.Contexto.RepresentacionCanonicaVinculadaV2()
	if err != nil {
		t.Fatal(err)
	}
	resultado.HuellaSHA256, err = resultado.Contexto.HuellaSHA256VinculadaV2()
	if err != nil {
		t.Fatal(err)
	}
	resultado.ManifiestoProcedenciaCanonico, err = manifiesto.RepresentacionCanonicaV1()
	if err != nil {
		t.Fatal(err)
	}
	resultado.ManifiestoProcedenciaHuellaSHA256, err =
		domain.HuellaSHA256ManifiestoProcedenciaContextoActorV1(resultado.ManifiestoProcedenciaCanonico)
	if err != nil {
		t.Fatal(err)
	}
	orden, err := ports.NuevaOrdenAcreditacionUsoRegistroContextoActorV2(
		resultado, emitida, validaHasta,
	)
	if err != nil {
		t.Fatalf("orden con cuatro MaxUint64: %v", err)
	}
	return orden
}

func probarCancelacionAcreditacionBloqueadaIntegracion(
	t *testing.T,
	ctx context.Context,
	pool *pgxpool.Pool,
	orden ports.OrdenAcreditacionUsoRegistroContextoActorV2,
) {
	t.Helper()
	bloqueador := comenzarTransaccionAcreditacionIntegracion(t, ctx, pool)
	defer func() { _ = bloqueador.Rollback(context.Background()) }()
	if _, err := bloqueador.Exec(ctx, `
		SELECT pg_advisory_xact_lock(hashtextextended(
			'vec_contexto_actor_v1:mutacion_punteros_actuales:v2', 0))`); err != nil {
		t.Fatalf("tomar advisory exclusivo: %v", err)
	}

	esperador := comenzarTransaccionAcreditacionIntegracion(t, ctx, pool)
	defer func() { _ = esperador.Rollback(context.Background()) }()
	var pid int32
	if err := esperador.QueryRow(ctx, `SELECT pg_backend_pid()`).Scan(&pid); err != nil {
		t.Fatalf("pid acreditador: %v", err)
	}
	acreditador, err := NuevoAcreditadorUsoRegistroContextoActorPostgreSQLV2EnTransaccion(esperador)
	if err != nil {
		t.Fatal(err)
	}
	ctxConsulta, cancelar := context.WithCancel(ctx)
	defer cancelar()
	resultado := make(chan error, 1)
	go func() {
		_, errConsulta := acreditador.AcreditarUsoRegistroContextoActorV2(ctxConsulta, orden)
		resultado <- errConsulta
	}()

	for {
		var esperando bool
		if err = pool.QueryRow(ctx, `
			SELECT EXISTS (
				SELECT 1 FROM pg_locks
				 WHERE pid=$1 AND locktype='advisory' AND granted IS FALSE
			)`, pid).Scan(&esperando); err != nil {
			t.Fatalf("observar espera advisory: %v", err)
		}
		if esperando {
			break
		}
		select {
		case err = <-resultado:
			t.Fatalf("Scan termino antes de bloquearse: %v", err)
		case <-ctx.Done():
			t.Fatalf("no se observo espera en pg_locks: %v", ctx.Err())
		default:
			runtime.Gosched()
		}
	}
	cancelar()
	select {
	case err = <-resultado:
		if !errors.Is(err, context.Canceled) ||
			!errors.Is(err, ports.ErrAcreditacionUsoRegistroContextoActorV2Denegada) {
			t.Fatalf("cancelacion durante Scan no preservada: %v", err)
		}
	case <-ctx.Done():
		t.Fatalf("Scan no atendio cancelacion: %v", ctx.Err())
	}
}
