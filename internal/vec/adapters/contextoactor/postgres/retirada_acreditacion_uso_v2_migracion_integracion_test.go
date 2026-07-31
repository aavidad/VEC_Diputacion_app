package postgres

import (
	"context"
	"fmt"
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

func TestRetiradaAcreditacionUsoV2BloqueaDDLBaseHastaCommit(t *testing.T) {
	dsn := os.Getenv("VEC_CONTEXTO_ACTOR_MIGRACION_POSTGRES_DSN")
	if dsn == "" {
		t.Skip("requiere PostgreSQL 18 efímero de contexto actor")
	}
	down := leerDocumentoRetiradaAcreditacionUsoV2(t)
	up := leerDocumentoAltaAcreditacionUsoV2(t)
	asegurarAcreditacionUsoV2InstaladaAlFinal(t, dsn, up)

	ctx, cancelar := context.WithTimeout(context.Background(), 20*time.Second)
	bloqueador := abrirConexionRetiradaAcreditacionUsoV2(t, ctx, dsn, "catalogal")
	defer cerrarConexionRetiradaAcreditacionUsoV2(t, bloqueador)
	retirada := abrirConexionRetiradaAcreditacionUsoV2(t, ctx, dsn, "de retirada")
	defer cerrarConexionRetiradaAcreditacionUsoV2(t, retirada)
	mutador := abrirConexionRetiradaAcreditacionUsoV2(t, ctx, dsn, "de DDL")
	defer cerrarConexionRetiradaAcreditacionUsoV2(t, mutador)
	supervisor := abrirConexionRetiradaAcreditacionUsoV2(t, ctx, dsn, "supervisora")
	defer cerrarConexionRetiradaAcreditacionUsoV2(t, supervisor)
	resultadoRetirada := make(chan error, 1)
	resultadoDDL := make(chan error, 1)
	retiradaEnCurso, ddlEnCurso := false, false
	defer func() {
		cancelar()
		if ddlEnCurso {
			select {
			case <-resultadoDDL:
			case <-time.After(5 * time.Second):
				t.Error("el DDL concurrente no terminó tras cancelar la prueba")
			}
		}
		if retiradaEnCurso {
			select {
			case <-resultadoRetirada:
			case <-time.After(5 * time.Second):
				t.Error("la retirada no terminó tras cancelar la prueba")
			}
		}
	}()

	if _, err := bloqueador.Exec(ctx, `
		BEGIN;
		LOCK TABLE pg_catalog.pg_class IN ROW EXCLUSIVE MODE
	`); err != nil {
		t.Fatalf("retener catálogo: %v", err)
	}
	configurarOptInRetiradaUsoV2(t, ctx, retirada, optInRetiradaAcreditacionUsoV2)
	pidRetirada := consultarPIDRetiradaAcreditacionUsoV2(t, ctx, retirada)
	retiradaEnCurso = true
	go func() {
		_, err := retirada.Exec(ctx, string(down))
		resultadoRetirada <- err
	}()
	esperarCondicionRetiradaAcreditacionUsoV2(
		t, ctx, supervisor, "AEX base concedido y SHARE catalogal en espera", `
			SELECT EXISTS (
			  SELECT 1 FROM pg_catalog.pg_locks
			   WHERE pid=$1
			     AND relation='vec_contexto_actor_v1.procedencias'::regclass
			     AND mode='AccessExclusiveLock' AND granted
			) AND EXISTS (
			  SELECT 1 FROM pg_catalog.pg_locks
			   WHERE pid=$1 AND relation='pg_catalog.pg_class'::regclass
			     AND mode='ShareLock' AND NOT granted
			)
		`, pidRetirada,
	)

	pidMutador := consultarPIDRetiradaAcreditacionUsoV2(t, ctx, mutador)
	ctxDDL, cancelarDDL := context.WithCancel(ctx)
	ddlEnCurso = true
	go func() {
		_, err := mutador.Exec(ctxDDL, `
			ALTER TABLE vec_contexto_actor_v1.procedencias
			ALTER COLUMN procedencia_ref SET STATISTICS 777
		`)
		resultadoDDL <- err
	}()
	esperarCondicionRetiradaAcreditacionUsoV2(
		t, ctx, supervisor, "ALTER de procedencias en espera", `
			SELECT EXISTS (
			  SELECT 1 FROM pg_catalog.pg_locks
			   WHERE pid=$1
			     AND relation='vec_contexto_actor_v1.procedencias'::regclass
			     AND NOT granted
			)
		`, pidMutador,
	)
	cancelarDDL()
	errDDL := <-resultadoDDL
	ddlEnCurso = false
	if errDDL == nil {
		t.Fatal("el ALTER concurrente atravesó el AEX de la retirada")
	}
	if err := sanearConexionRetiradaAcreditacionUsoV2(mutador); err != nil {
		t.Fatalf("sanear conexión DDL cancelada: %v", err)
	}
	var estadistica int32
	if err := supervisor.QueryRow(ctx, `
		SELECT coalesce(attstattarget, -1)
		  FROM pg_catalog.pg_attribute
		 WHERE attrelid='vec_contexto_actor_v1.procedencias'::regclass
		   AND attname='procedencia_ref'
	`).Scan(&estadistica); err != nil {
		t.Fatalf("consultar estadística tras cancelar DDL: %v", err)
	}
	if estadistica != -1 {
		t.Fatalf("el DDL cancelado dejó attstattarget=%d", estadistica)
	}

	if _, err := bloqueador.Exec(ctx, "ROLLBACK"); err != nil {
		t.Fatalf("liberar catálogo: %v", err)
	}
	errRetirada := <-resultadoRetirada
	retiradaEnCurso = false
	if errRetirada != nil {
		t.Fatalf("finalizar retirada tras liberar catálogo: %v", errRetirada)
	}
	comprobarBaseTrasRetiradaAcreditacionUsoV2(t, ctx, supervisor)
	if _, err := supervisor.Exec(ctx, string(up)); err != nil {
		t.Fatalf("restaurar 000002 tras carrera: %v", err)
	}
}

func TestRetiradaAcreditacionUsoV2RechazaIndiceClusterDerivado(t *testing.T) {
	dsn := os.Getenv("VEC_CONTEXTO_ACTOR_MIGRACION_POSTGRES_DSN")
	if dsn == "" {
		t.Skip("requiere PostgreSQL 18 efímero de contexto actor")
	}
	down := leerDocumentoRetiradaAcreditacionUsoV2(t)
	up := leerDocumentoAltaAcreditacionUsoV2(t)
	asegurarAcreditacionUsoV2InstaladaAlFinal(t, dsn, up)
	ctx, cancelar := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancelar()
	conexion := abrirConexionRetiradaAcreditacionUsoV2(t, ctx, dsn, "de CLUSTER")
	defer cerrarConexionRetiradaAcreditacionUsoV2(t, conexion)

	if _, err := conexion.Exec(ctx, `
		CLUSTER vec_contexto_actor_v1.procedencias
		USING procedencias_pkey
	`); err != nil {
		t.Fatalf("derivar índice CLUSTER sintético: %v", err)
	}
	configurarOptInRetiradaUsoV2(t, ctx, conexion, optInRetiradaAcreditacionUsoV2)
	if _, err := conexion.Exec(ctx, string(down)); err == nil {
		t.Fatal("la retirada aceptó indisclustered derivado")
	}
	if err := sanearConexionRetiradaAcreditacionUsoV2(conexion); err != nil {
		t.Fatalf("sanear rechazo de CLUSTER: %v", err)
	}
	if _, err := conexion.Exec(ctx, `
		ALTER TABLE vec_contexto_actor_v1.procedencias SET WITHOUT CLUSTER
	`); err != nil {
		t.Fatalf("restaurar preferencia CLUSTER: %v", err)
	}
	configurarOptInRetiradaUsoV2(t, ctx, conexion, optInRetiradaAcreditacionUsoV2)
	if _, err := conexion.Exec(ctx, string(down)); err != nil {
		t.Fatalf("retirar tras restaurar CLUSTER: %v", err)
	}
	comprobarBaseTrasRetiradaAcreditacionUsoV2(t, ctx, conexion)
	if _, err := conexion.Exec(ctx, string(up)); err != nil {
		t.Fatalf("restaurar 000002 tras CLUSTER: %v", err)
	}
}

func TestRetiradaAcreditacionUsoV2IgnoraDependenciaDeBaseClonada(t *testing.T) {
	dsn := os.Getenv("VEC_CONTEXTO_ACTOR_MIGRACION_POSTGRES_DSN")
	if dsn == "" {
		t.Skip("requiere PostgreSQL 18 efímero de contexto actor")
	}
	down := leerDocumentoRetiradaAcreditacionUsoV2(t)
	up := leerDocumentoAltaAcreditacionUsoV2(t)
	asegurarAcreditacionUsoV2InstaladaAlFinal(t, dsn, up)
	ctx, cancelar := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancelar()
	configuracion, err := pgx.ParseConfig(dsn)
	if err != nil {
		t.Fatalf("interpretar DSN de migración: %v", err)
	}
	baseOrigen := configuracion.Database
	sufijo := fmt.Sprintf("%d", time.Now().UnixNano())
	baseClon := "ct126_oid_" + sufijo
	rolClon := "ct126_rol_" + sufijo
	configuracion.Database = "postgres"
	administrador, err := pgx.ConnectConfig(ctx, configuracion)
	if err != nil {
		t.Fatalf("abrir conexión administrativa: %v", err)
	}
	defer cerrarConexionRetiradaAcreditacionUsoV2(t, administrador)
	t.Cleanup(func() {
		ctxLimpieza, cancelarLimpieza := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancelarLimpieza()
		configuracionLimpieza := configuracion.Copy()
		configuracionLimpieza.Database = "postgres"
		conexionLimpieza, err := pgx.ConnectConfig(ctxLimpieza, configuracionLimpieza)
		if err != nil {
			t.Errorf("abrir conexión para limpiar clon: %v", err)
			return
		}
		defer func() { _ = conexionLimpieza.Close(ctxLimpieza) }()
		if _, err = conexionLimpieza.Exec(
			ctxLimpieza,
			"DROP DATABASE IF EXISTS "+pgx.Identifier{baseClon}.Sanitize()+" WITH (FORCE)",
		); err != nil {
			t.Errorf("eliminar base clonada: %v", err)
			return
		}
		if _, err = conexionLimpieza.Exec(
			ctxLimpieza,
			"DROP ROLE IF EXISTS "+pgx.Identifier{rolClon}.Sanitize(),
		); err != nil {
			t.Errorf("eliminar rol de base clonada: %v", err)
		}
	})
	if _, err = administrador.Exec(
		ctx,
		"CREATE ROLE "+pgx.Identifier{rolClon}.Sanitize()+" NOLOGIN",
	); err != nil {
		t.Fatalf("crear rol de dependencia clonada: %v", err)
	}
	if _, err = administrador.Exec(
		ctx,
		"CREATE DATABASE "+pgx.Identifier{baseClon}.Sanitize()+
			" WITH TEMPLATE "+pgx.Identifier{baseOrigen}.Sanitize(),
	); err != nil {
		t.Fatalf("clonar base con OID coincidentes: %v", err)
	}

	configuracion.Database = baseClon
	clon, err := pgx.ConnectConfig(ctx, configuracion)
	if err != nil {
		t.Fatalf("abrir base clonada: %v", err)
	}
	funcion := `vec_contexto_actor_v1.acreditar_uso_registro_contexto_actor_v2(
		text,text,text,text,text,text,numeric,text,numeric,text,numeric,text,numeric,
		text,text,timestamptz,timestamptz)`
	if _, err = clon.Exec(
		ctx,
		"GRANT EXECUTE ON FUNCTION "+funcion+" TO "+pgx.Identifier{rolClon}.Sanitize(),
	); err != nil {
		_ = clon.Close(ctx)
		t.Fatalf("crear dependencia en base clonada: %v", err)
	}
	var oidClon uint32
	if err = clon.QueryRow(ctx, "SELECT $1::regprocedure::oid", funcion).Scan(&oidClon); err != nil {
		_ = clon.Close(ctx)
		t.Fatalf("consultar OID clonado: %v", err)
	}
	if err = clon.Close(ctx); err != nil {
		t.Fatalf("cerrar base clonada: %v", err)
	}

	configuracion.Database = baseOrigen
	origen, err := pgx.ConnectConfig(ctx, configuracion)
	if err != nil {
		t.Fatalf("abrir base origen: %v", err)
	}
	defer cerrarConexionRetiradaAcreditacionUsoV2(t, origen)
	var oidOrigen uint32
	if err = origen.QueryRow(ctx, "SELECT $1::regprocedure::oid", funcion).Scan(&oidOrigen); err != nil {
		t.Fatalf("consultar OID origen: %v", err)
	}
	if oidOrigen != oidClon {
		t.Fatalf("el clon no conservó OID: origen=%d clon=%d", oidOrigen, oidClon)
	}
	configurarOptInRetiradaUsoV2(t, ctx, origen, optInRetiradaAcreditacionUsoV2)
	if _, err = origen.Exec(ctx, string(down)); err != nil {
		t.Fatalf("la dependencia de otra base contaminó pg_shdepend: %v", err)
	}
	comprobarBaseTrasRetiradaAcreditacionUsoV2(t, ctx, origen)
	if _, err = origen.Exec(ctx, string(up)); err != nil {
		t.Fatalf("restaurar 000002 tras prueba multibase: %v", err)
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

func leerDocumentoAltaAcreditacionUsoV2(t *testing.T) []byte {
	t.Helper()
	_, archivoPrueba, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("no se pudo localizar el archivo de prueba")
	}
	ruta := filepath.Join(
		filepath.Clean(filepath.Join(filepath.Dir(archivoPrueba), "../../../../..")),
		"deploy/postgresql/contexto_actor_v1/migraciones",
		"000002_acreditacion_uso_registro_contexto_actor_v2.up.sql",
	)
	documento, err := os.ReadFile(ruta)
	if err != nil {
		t.Fatalf("leer documento de alta 000002: %v", err)
	}
	return documento
}

func abrirConexionRetiradaAcreditacionUsoV2(
	t *testing.T,
	ctx context.Context,
	dsn string,
	finalidad string,
) *pgx.Conn {
	t.Helper()
	conexion, err := pgx.Connect(ctx, dsn)
	if err != nil {
		t.Fatalf("abrir conexión %s: %v", finalidad, err)
	}
	return conexion
}

func consultarPIDRetiradaAcreditacionUsoV2(
	t *testing.T,
	ctx context.Context,
	conexion *pgx.Conn,
) int32 {
	t.Helper()
	var pid int32
	if err := conexion.QueryRow(ctx, "SELECT pg_catalog.pg_backend_pid()").Scan(&pid); err != nil {
		t.Fatalf("consultar PID PostgreSQL: %v", err)
	}
	return pid
}

func esperarCondicionRetiradaAcreditacionUsoV2(
	t *testing.T,
	ctx context.Context,
	conexion *pgx.Conn,
	descripcion string,
	consulta string,
	argumentos ...any,
) {
	t.Helper()
	limite := time.NewTimer(5 * time.Second)
	defer limite.Stop()
	pulso := time.NewTicker(10 * time.Millisecond)
	defer pulso.Stop()
	for {
		var cumplida bool
		if err := conexion.QueryRow(ctx, consulta, argumentos...).Scan(&cumplida); err != nil {
			t.Fatalf("observar %s: %v", descripcion, err)
		}
		if cumplida {
			return
		}
		select {
		case <-ctx.Done():
			t.Fatalf("esperar %s: %v", descripcion, ctx.Err())
		case <-limite.C:
			t.Fatalf("no se observó %s", descripcion)
		case <-pulso.C:
		}
	}
}

func comprobarBaseTrasRetiradaAcreditacionUsoV2(
	t *testing.T,
	ctx context.Context,
	conexion *pgx.Conn,
) {
	t.Helper()
	var exacta bool
	if err := conexion.QueryRow(ctx, `
		SELECT
		  pg_catalog.to_regclass(
		    'vec_contexto_actor_v1.control_generacion_punteros_actuales_v2'
		  ) IS NULL
		  AND pg_catalog.to_regprocedure(
		    'vec_contexto_actor_v1.acreditar_uso_registro_contexto_actor_v2(text,text,text,text,text,text,numeric,text,numeric,text,numeric,text,numeric,text,text,timestamptz,timestamptz)'
		  ) IS NULL
		  AND pg_catalog.to_regclass(
		    'vec_contexto_actor_v1.registros_contexto'
		  ) IS NOT NULL
		  AND (
		    SELECT coalesce(attstattarget, -1) = -1
		      FROM pg_catalog.pg_attribute
		     WHERE attrelid='vec_contexto_actor_v1.procedencias'::regclass
		       AND attname='procedencia_ref'
		  )
		  AND NOT EXISTS (
		    SELECT 1 FROM pg_catalog.pg_index
		     WHERE indrelid='vec_contexto_actor_v1.procedencias'::regclass
		       AND indisclustered
		  )
	`).Scan(&exacta); err != nil {
		t.Fatalf("comprobar estado base tras retirada: %v", err)
	}
	if !exacta {
		t.Fatal("000001 no quedó exacta tras retirar 000002")
	}
}

func asegurarAcreditacionUsoV2InstaladaAlFinal(t *testing.T, dsn string, up []byte) {
	t.Helper()
	t.Cleanup(func() {
		ctx, cancelar := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancelar()
		conexion, err := pgx.Connect(ctx, dsn)
		if err != nil {
			t.Errorf("abrir conexión de restauración 000002: %v", err)
			return
		}
		defer func() { _ = conexion.Close(ctx) }()
		var instalada bool
		if err = conexion.QueryRow(ctx, `
			SELECT pg_catalog.to_regclass(
			  'vec_contexto_actor_v1.control_generacion_punteros_actuales_v2'
			) IS NOT NULL
		`).Scan(&instalada); err != nil {
			t.Errorf("comprobar restauración 000002: %v", err)
			return
		}
		if instalada {
			return
		}
		if _, err = conexion.Exec(ctx, string(up)); err != nil {
			t.Errorf("restaurar 000002 al finalizar prueba: %v", err)
		}
	})
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
