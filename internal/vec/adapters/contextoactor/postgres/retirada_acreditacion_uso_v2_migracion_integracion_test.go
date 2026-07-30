package postgres

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
)

const (
	gucRetiradaAcreditacionUsoV2    = "vec.confirmar_retirada_acreditacion_contexto_actor_v2"
	optInRetiradaAcreditacionUsoV2  = "RETIRAR_ACREDITACION_CONTEXTO_ACTOR_V2"
	consultaResetOptInRetiradaUsoV2 = "RESET vec.confirmar_retirada_acreditacion_contexto_actor_v2"
)

// TestRetiradaAcreditacionUsoV2EjecutaDocumentoSQLIntegro acredita que una
// aplicación puede ejecutar el mismo artefacto que psql sin interpretarlo.
// Emplea una conexión dedicada porque un error deliberado deja abortada la
// transacción abierta por el documento hasta el ROLLBACK explícito.
func TestRetiradaAcreditacionUsoV2EjecutaDocumentoSQLIntegro(t *testing.T) {
	dsn := os.Getenv("VEC_CONTEXTO_ACTOR_MIGRACION_POSTGRES_DSN")
	if dsn == "" {
		t.Skip("requiere PostgreSQL 18 efímero de contexto actor")
	}
	documento := leerDocumentoRetiradaAcreditacionUsoV2(t)

	ctx, cancelar := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancelar()
	conexion, err := pgx.Connect(ctx, dsn)
	if err != nil {
		t.Fatalf("abrir conexión dedicada de migración: %v", err)
	}
	defer cerrarConexionRetiradaAcreditacionUsoV2(t, conexion)

	var versionMayor int
	if err = conexion.QueryRow(ctx, `
		SELECT pg_catalog.current_setting('server_version_num')::integer / 10000
	`).Scan(&versionMayor); err != nil {
		t.Fatalf("consultar versión PostgreSQL: %v", err)
	}
	if versionMayor != 18 {
		t.Fatalf("requiere PostgreSQL 18; servidor conectado: %d", versionMayor)
	}

	probarRechazoDocumentoRetiradaUsoV2(t, ctx, conexion, documento, nil)
	incorrecta := "RETIRAR_SIN_CONTRATO"
	probarRechazoDocumentoRetiradaUsoV2(t, ctx, conexion, documento, &incorrecta)

	configurarOptInRetiradaUsoV2(t, ctx, conexion, optInRetiradaAcreditacionUsoV2)
	if _, err = conexion.Exec(ctx, string(documento)); err != nil {
		t.Fatalf("ejecutar documento íntegro con opt-in exacto: %v", err)
	}
	if estado := conexion.PgConn().TxStatus(); estado != 'I' {
		t.Fatalf("el COMMIT del documento no dejó la conexión libre: %q", estado)
	}
	resetearOptInRetiradaUsoV2(t, ctx, conexion)

	var retirada, baseIntacta bool
	if err = conexion.QueryRow(ctx, `
		SELECT
		  pg_catalog.to_regclass(
		    'vec_contexto_actor_v1.control_generacion_punteros_actuales_v2'
		  ) IS NULL
		  AND pg_catalog.to_regprocedure(
		    'vec_contexto_actor_v1.acreditar_uso_registro_contexto_actor_v2(text,text,text,text,text,text,numeric,text,numeric,text,numeric,text,numeric,text,text,timestamptz,timestamptz)'
		  ) IS NULL,
		  pg_catalog.to_regclass(
		    'vec_contexto_actor_v1.registros_contexto'
		  ) IS NOT NULL
	`).Scan(&retirada, &baseIntacta); err != nil {
		t.Fatalf("comprobar postcondición del documento: %v", err)
	}
	if !retirada || !baseIntacta {
		t.Fatalf("postcondición incoherente: retirada=%t base=%t", retirada, baseIntacta)
	}
}

func TestRetiradaAcreditacionUsoV2CancelaEsperaSinDeriva(t *testing.T) {
	dsn := os.Getenv("VEC_CONTEXTO_ACTOR_MIGRACION_POSTGRES_DSN")
	if dsn == "" {
		t.Skip("requiere PostgreSQL 18 efímero de contexto actor")
	}
	documento := leerDocumentoRetiradaAcreditacionUsoV2(t)
	ctx, cancelar := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancelar()

	bloqueador, err := pgx.Connect(ctx, dsn)
	if err != nil {
		t.Fatalf("abrir conexión bloqueadora: %v", err)
	}
	defer cerrarConexionRetiradaAcreditacionUsoV2(t, bloqueador)
	if _, err = bloqueador.Exec(ctx, `
		BEGIN;
		LOCK TABLE vec_contexto_actor_v1.control_generacion_punteros_actuales_v2
		IN ACCESS EXCLUSIVE MODE
	`); err != nil {
		t.Fatalf("bloquear tabla de control: %v", err)
	}

	conexion, err := pgx.Connect(ctx, dsn)
	if err != nil {
		t.Fatalf("abrir conexión dedicada de retirada: %v", err)
	}
	defer cerrarConexionRetiradaAcreditacionUsoV2(t, conexion)
	configurarOptInRetiradaUsoV2(t, ctx, conexion, optInRetiradaAcreditacionUsoV2)

	ctxCancelado, cancelarEspera := context.WithTimeout(ctx, 200*time.Millisecond)
	defer cancelarEspera()
	if _, err = conexion.Exec(ctxCancelado, string(documento)); err == nil {
		t.Fatal("la retirada bloqueada ignoró la cancelación")
	}
	if err = sanearConexionRetiradaAcreditacionUsoV2(conexion); err != nil {
		t.Fatalf("sanear conexión tras cancelación: %v", err)
	}
	if _, err = bloqueador.Exec(ctx, "ROLLBACK"); err != nil {
		t.Fatalf("liberar tabla de control: %v", err)
	}

	var intacta bool
	if err = bloqueador.QueryRow(ctx, `
		SELECT pg_catalog.to_regclass(
		  'vec_contexto_actor_v1.control_generacion_punteros_actuales_v2'
		) IS NOT NULL
		AND pg_catalog.to_regprocedure(
		  'vec_contexto_actor_v1.acreditar_uso_registro_contexto_actor_v2(text,text,text,text,text,text,numeric,text,numeric,text,numeric,text,numeric,text,text,timestamptz,timestamptz)'
		) IS NOT NULL
	`).Scan(&intacta); err != nil {
		t.Fatalf("comprobar rollback tras cancelación: %v", err)
	}
	if !intacta {
		t.Fatal("la cancelación dejó una retirada parcial")
	}
}

func TestRetiradaAcreditacionUsoV2DestruyeConexionQueNoPuedeSanear(t *testing.T) {
	dsn := os.Getenv("VEC_CONTEXTO_ACTOR_MIGRACION_POSTGRES_DSN")
	if dsn == "" {
		t.Skip("requiere PostgreSQL 18 efímero de contexto actor")
	}
	ctx, cancelar := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancelar()
	conexion, err := pgx.Connect(ctx, dsn)
	if err != nil {
		t.Fatalf("abrir conexión destinada a descarte: %v", err)
	}
	supervisor, err := pgx.Connect(ctx, dsn)
	if err != nil {
		_ = conexion.Close(ctx)
		t.Fatalf("abrir conexión supervisora: %v", err)
	}
	defer cerrarConexionRetiradaAcreditacionUsoV2(t, supervisor)

	if _, err = conexion.Exec(ctx, "BEGIN"); err != nil {
		t.Fatalf("abrir transacción destinada a descarte: %v", err)
	}
	var pid int32
	if err = conexion.QueryRow(ctx, "SELECT pg_catalog.pg_backend_pid()").Scan(&pid); err != nil {
		t.Fatalf("obtener pid destinado a descarte: %v", err)
	}
	var terminada bool
	if err = supervisor.QueryRow(
		ctx,
		"SELECT pg_catalog.pg_terminate_backend($1)",
		pid,
	).Scan(&terminada); err != nil || !terminada {
		t.Fatalf("terminar backend sintético: terminada=%t error=%v", terminada, err)
	}
	if err = sanearConexionRetiradaAcreditacionUsoV2(conexion); err == nil {
		t.Fatal("el saneado aparentó éxito sobre un backend terminado")
	}
	if !conexion.IsClosed() {
		t.Fatal("la conexión no saneable no fue destruida")
	}
}

func leerDocumentoRetiradaAcreditacionUsoV2(t *testing.T) []byte {
	t.Helper()
	_, archivoPrueba, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("no se pudo localizar el archivo de prueba")
	}
	raiz := filepath.Clean(filepath.Join(filepath.Dir(archivoPrueba), "../../../../.."))
	ruta := filepath.Join(
		raiz,
		"deploy/postgresql/contexto_actor_v1/migraciones",
		"000002_acreditacion_uso_registro_contexto_actor_v2.down.sql",
	)
	directorioAnterior, err := os.Getwd()
	if err != nil {
		t.Fatalf("consultar directorio actual: %v", err)
	}
	if err = os.Chdir(t.TempDir()); err != nil {
		t.Fatalf("cambiar a directorio ajeno: %v", err)
	}
	t.Cleanup(func() {
		if err := os.Chdir(directorioAnterior); err != nil {
			t.Errorf("restaurar directorio de trabajo: %v", err)
		}
	})
	documento, err := os.ReadFile(ruta)
	if err != nil {
		t.Fatalf("leer bytes reales del documento desde cwd ajeno: %v", err)
	}
	return documento
}

func probarRechazoDocumentoRetiradaUsoV2(
	t *testing.T,
	ctx context.Context,
	conexion *pgx.Conn,
	documento []byte,
	optIn *string,
) {
	t.Helper()
	if optIn == nil {
		resetearOptInRetiradaUsoV2(t, ctx, conexion)
	} else {
		configurarOptInRetiradaUsoV2(t, ctx, conexion, *optIn)
	}
	if _, err := conexion.Exec(ctx, string(documento)); err == nil {
		t.Fatal("el documento aceptó un opt-in ausente o incorrecto")
	}
	if estado := conexion.PgConn().TxStatus(); estado != 'E' {
		t.Fatalf("el error no dejó observable la transacción abortada: %q", estado)
	}
	if _, err := conexion.Exec(ctx, "ROLLBACK"); err != nil {
		t.Fatalf("revertir conexión abortada: %v", err)
	}
	resetearOptInRetiradaUsoV2(t, ctx, conexion)

	var presente bool
	if err := conexion.QueryRow(ctx, `
		SELECT pg_catalog.to_regprocedure(
		  'vec_contexto_actor_v1.acreditar_uso_registro_contexto_actor_v2(text,text,text,text,text,text,numeric,text,numeric,text,numeric,text,numeric,text,text,timestamptz,timestamptz)'
		) IS NOT NULL
	`).Scan(&presente); err != nil {
		t.Fatalf("comprobar rollback del rechazo: %v", err)
	}
	if !presente {
		t.Fatal("un rechazo retiró parcialmente la acreditación")
	}
}

func configurarOptInRetiradaUsoV2(
	t *testing.T,
	ctx context.Context,
	conexion *pgx.Conn,
	valor string,
) {
	t.Helper()
	var observado string
	if err := conexion.QueryRow(
		ctx,
		"SELECT pg_catalog.set_config($1,$2,false)",
		gucRetiradaAcreditacionUsoV2,
		valor,
	).Scan(&observado); err != nil {
		t.Fatalf("configurar opt-in de sesión: %v", err)
	}
	if observado != valor {
		t.Fatalf("opt-in observado distinto: %q", observado)
	}
}

func resetearOptInRetiradaUsoV2(
	t *testing.T,
	ctx context.Context,
	conexion *pgx.Conn,
) {
	t.Helper()
	if _, err := conexion.Exec(ctx, consultaResetOptInRetiradaUsoV2); err != nil {
		t.Fatalf("limpiar opt-in de sesión: %v", err)
	}
}

func cerrarConexionRetiradaAcreditacionUsoV2(t *testing.T, conexion *pgx.Conn) {
	t.Helper()
	if err := sanearConexionRetiradaAcreditacionUsoV2(conexion); err != nil {
		t.Errorf("sanear conexión dedicada: %v", err)
	}
	if conexion.IsClosed() {
		return
	}
	ctx, cancelar := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancelar()
	if err := conexion.Close(ctx); err != nil {
		t.Errorf("cerrar conexión dedicada: %v", err)
	}
}

func sanearConexionRetiradaAcreditacionUsoV2(conexion *pgx.Conn) error {
	if conexion.IsClosed() {
		return nil
	}
	ctx, cancelar := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancelar()
	if conexion.PgConn().TxStatus() != 'I' {
		if _, err := conexion.Exec(ctx, "ROLLBACK"); err != nil {
			_ = conexion.Close(ctx)
			return err
		}
	}
	if _, err := conexion.Exec(ctx, consultaResetOptInRetiradaUsoV2); err != nil {
		_ = conexion.Close(ctx)
		return err
	}
	return nil
}
