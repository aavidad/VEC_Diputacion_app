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
	gucRetiradaVinculoCorporativoRRHHV1           = "vec.confirmar_retirada_vinculo_corporativo_rrhh_v1"
	optInRetiradaVinculoCorporativoRRHHV1         = "RETIRAR_VINCULO_CORPORATIVO_RRHH_V1"
	consultaResetRetiradaVinculoCorporativoRRHHV1 = "RESET vec.confirmar_retirada_vinculo_corporativo_rrhh_v1"
)

// TestRetiradaVinculoCorporativoRRHHV1EjecutaDocumentoSQLIntegro acredita
// que pgx ejecuta todos los bytes publicados del down sin preprocesarlos.
// La conexion dedicada solo puede reutilizarse despues de rollback y reset.
func TestRetiradaVinculoCorporativoRRHHV1EjecutaDocumentoSQLIntegro(t *testing.T) {
	dsn := os.Getenv("VEC_CONTEXTO_ACTOR_MIGRACION_POSTGRES_DSN")
	if dsn == "" {
		t.Skip("requiere PostgreSQL 18.4 efimero de contexto actor")
	}
	down := leerDocumentoVinculoCorporativoRRHHV1(t, "down")
	up := leerDocumentoVinculoCorporativoRRHHV1(t, "up")
	asegurarVinculoCorporativoRRHHV1InstaladoAlFinal(t, dsn, up)

	ctx, cancelar := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancelar()
	conexion := abrirConexionRetiradaAcreditacionUsoV2(t, ctx, dsn, "de vinculo corporativo")
	defer cerrarConexionRetiradaVinculoCorporativoRRHHV1(t, conexion)

	var version int
	if err := conexion.QueryRow(ctx, `
		SELECT pg_catalog.current_setting('server_version_num')::integer
	`).Scan(&version); err != nil {
		t.Fatalf("consultar version PostgreSQL: %v", err)
	}
	if version != 180004 {
		t.Fatalf("requiere PostgreSQL 18.4 exacto; servidor conectado: %d", version)
	}

	probarRechazoRetiradaVinculoCorporativoRRHHV1(t, ctx, conexion, down, nil)
	incorrecta := "RETIRAR_SIN_CONTRATO"
	probarRechazoRetiradaVinculoCorporativoRRHHV1(
		t, ctx, conexion, down, &incorrecta,
	)
	comprobarVinculoCorporativoRRHHV1Vacio(t, ctx, conexion)

	configurarOptInRetiradaVinculoCorporativoRRHHV1(
		t, ctx, conexion, optInRetiradaVinculoCorporativoRRHHV1,
	)
	if _, err := conexion.Exec(ctx, string(down)); err != nil {
		t.Fatalf("ejecutar documento integro con opt-in exacto: %v", err)
	}
	if estado := conexion.PgConn().TxStatus(); estado != 'I' {
		t.Fatalf("el COMMIT del documento no dejo la conexion libre: %q", estado)
	}
	if err := sanearConexionRetiradaVinculoCorporativoRRHHV1(conexion); err != nil {
		t.Fatalf("sanear conexion tras retirada valida: %v", err)
	}
	comprobarVinculoCorporativoRRHHV1Retirado(t, ctx, conexion)
	if _, err := conexion.Exec(ctx, string(up)); err != nil {
		t.Fatalf("restaurar 000004 tras retirada valida: %v", err)
	}
	comprobarVinculoCorporativoRRHHV1Instalado(t, ctx, conexion)

	probarCancelacionRetiradaVinculoCorporativoRRHHV1(t, ctx, dsn, down)
}

// TestRetiradaVinculoCorporativoRRHHV1DestruyeConexionQueNoPuedeSanear
// demuestra que un backend terminado nunca vuelve contaminado al consumidor.
func TestRetiradaVinculoCorporativoRRHHV1DestruyeConexionQueNoPuedeSanear(
	t *testing.T,
) {
	dsn := os.Getenv("VEC_CONTEXTO_ACTOR_MIGRACION_POSTGRES_DSN")
	if dsn == "" {
		t.Skip("requiere PostgreSQL 18.4 efimero de contexto actor")
	}
	ctx, cancelar := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancelar()
	conexion := abrirConexionRetiradaAcreditacionUsoV2(t, ctx, dsn, "destinada a descarte")
	supervisor := abrirConexionRetiradaAcreditacionUsoV2(t, ctx, dsn, "supervisora")
	defer cerrarConexionRetiradaAcreditacionUsoV2(t, supervisor)

	configurarOptInRetiradaVinculoCorporativoRRHHV1(
		t, ctx, conexion, optInRetiradaVinculoCorporativoRRHHV1,
	)
	if _, err := conexion.Exec(ctx, "BEGIN"); err != nil {
		t.Fatalf("abrir transaccion destinada a descarte: %v", err)
	}
	pid := consultarPIDRetiradaAcreditacionUsoV2(t, ctx, conexion)
	var terminada bool
	if err := supervisor.QueryRow(
		ctx, "SELECT pg_catalog.pg_terminate_backend($1)", pid,
	).Scan(&terminada); err != nil || !terminada {
		t.Fatalf("terminar backend sintetico: terminada=%t error=%v", terminada, err)
	}
	if err := sanearConexionRetiradaVinculoCorporativoRRHHV1(conexion); err == nil {
		t.Fatal("el saneado aparento exito sobre un backend terminado")
	}
	if !conexion.IsClosed() {
		t.Fatal("la conexion no saneable no fue destruida")
	}
}

func probarCancelacionRetiradaVinculoCorporativoRRHHV1(
	t *testing.T,
	ctx context.Context,
	dsn string,
	down []byte,
) {
	t.Helper()
	bloqueador := abrirConexionRetiradaAcreditacionUsoV2(t, ctx, dsn, "bloqueadora")
	defer cerrarConexionRetiradaAcreditacionUsoV2(t, bloqueador)
	retirada := abrirConexionRetiradaAcreditacionUsoV2(t, ctx, dsn, "de retirada cancelable")
	defer cerrarConexionRetiradaVinculoCorporativoRRHHV1(t, retirada)
	supervisor := abrirConexionRetiradaAcreditacionUsoV2(t, ctx, dsn, "de supervision")
	defer cerrarConexionRetiradaAcreditacionUsoV2(t, supervisor)

	if _, err := bloqueador.Exec(ctx, `
		BEGIN;
		LOCK TABLE vec_contexto_actor_v1.vinculo_corporativo_actual
		IN ACCESS EXCLUSIVE MODE
	`); err != nil {
		t.Fatalf("bloquear puntero corporativo: %v", err)
	}
	pidBloqueador := consultarPIDRetiradaAcreditacionUsoV2(t, ctx, bloqueador)
	configurarOptInRetiradaVinculoCorporativoRRHHV1(
		t, ctx, retirada, optInRetiradaVinculoCorporativoRRHHV1,
	)
	pidRetirada := consultarPIDRetiradaAcreditacionUsoV2(t, ctx, retirada)
	ctxRetirada, cancelarRetirada := context.WithCancel(ctx)
	resultado := make(chan error, 1)
	go func() {
		_, err := retirada.Exec(ctxRetirada, string(down))
		resultado <- err
	}()
	esperarCondicionRetiradaAcreditacionUsoV2(
		t, ctx, supervisor, "down 000004 esperando un bloqueo real", `
			SELECT EXISTS (
			  SELECT 1 FROM pg_catalog.pg_locks
			   WHERE pid=$1
			     AND relation=
			       'vec_contexto_actor_v1.vinculo_corporativo_actual'::regclass
			     AND mode='AccessExclusiveLock' AND NOT granted
			)
			AND pg_catalog.pg_blocking_pids($1)=ARRAY[$2]::integer[]
		`, pidRetirada, pidBloqueador,
	)
	cancelarRetirada()
	select {
	case err := <-resultado:
		if err == nil {
			t.Fatal("la retirada bloqueada ignoro la cancelacion")
		}
	case <-time.After(5 * time.Second):
		t.Fatal("la retirada no termino tras cancelar su contexto")
	}
	if err := sanearConexionRetiradaVinculoCorporativoRRHHV1(retirada); err != nil &&
		!retirada.IsClosed() {
		t.Fatalf("la conexion cancelada no quedo saneada ni destruida: %v", err)
	}
	if _, err := bloqueador.Exec(ctx, "ROLLBACK"); err != nil {
		t.Fatalf("liberar puntero corporativo: %v", err)
	}
	comprobarVinculoCorporativoRRHHV1Instalado(t, ctx, supervisor)
	if !retirada.IsClosed() {
		comprobarOptInRetiradaVinculoCorporativoRRHHV1Limpio(t, ctx, retirada)
	}
}

func probarRechazoRetiradaVinculoCorporativoRRHHV1(
	t *testing.T,
	ctx context.Context,
	conexion *pgx.Conn,
	documento []byte,
	optIn *string,
) {
	t.Helper()
	if optIn == nil {
		if err := sanearConexionRetiradaVinculoCorporativoRRHHV1(conexion); err != nil {
			t.Fatalf("preparar rechazo sin opt-in: %v", err)
		}
	} else {
		configurarOptInRetiradaVinculoCorporativoRRHHV1(t, ctx, conexion, *optIn)
	}
	if _, err := conexion.Exec(ctx, string(documento)); err == nil {
		t.Fatal("el documento acepto un opt-in ausente o incorrecto")
	}
	if estado := conexion.PgConn().TxStatus(); estado != 'E' {
		t.Fatalf("el rechazo no dejo observable la transaccion abortada: %q", estado)
	}
	if err := sanearConexionRetiradaVinculoCorporativoRRHHV1(conexion); err != nil {
		t.Fatalf("sanear conexion tras rechazo: %v", err)
	}
	comprobarOptInRetiradaVinculoCorporativoRRHHV1Limpio(t, ctx, conexion)
	comprobarVinculoCorporativoRRHHV1Instalado(t, ctx, conexion)
}

func configurarOptInRetiradaVinculoCorporativoRRHHV1(
	t *testing.T,
	ctx context.Context,
	conexion *pgx.Conn,
	valor string,
) {
	t.Helper()
	var observado string
	if err := conexion.QueryRow(
		ctx, "SELECT pg_catalog.set_config($1,$2,false)",
		gucRetiradaVinculoCorporativoRRHHV1, valor,
	).Scan(&observado); err != nil {
		t.Fatalf("configurar opt-in de sesion: %v", err)
	}
	if observado != valor {
		t.Fatalf("opt-in observado distinto: %q", observado)
	}
}

func sanearConexionRetiradaVinculoCorporativoRRHHV1(conexion *pgx.Conn) error {
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
	if _, err := conexion.Exec(
		ctx, consultaResetRetiradaVinculoCorporativoRRHHV1,
	); err != nil {
		_ = conexion.Close(ctx)
		return err
	}
	if estado := conexion.PgConn().TxStatus(); estado != 'I' {
		_ = conexion.Close(ctx)
		return fmt.Errorf("conexion no inactiva tras sanear: %q", estado)
	}
	var observado string
	if err := conexion.QueryRow(ctx, `
		SELECT COALESCE(pg_catalog.current_setting($1,true),'')
	`, gucRetiradaVinculoCorporativoRRHHV1).Scan(&observado); err != nil {
		_ = conexion.Close(ctx)
		return err
	}
	if observado != "" {
		_ = conexion.Close(ctx)
		return fmt.Errorf("el GUC conservo un valor residual tras el reset: %q", observado)
	}
	return nil
}

func cerrarConexionRetiradaVinculoCorporativoRRHHV1(
	t *testing.T,
	conexion *pgx.Conn,
) {
	t.Helper()
	if err := sanearConexionRetiradaVinculoCorporativoRRHHV1(conexion); err != nil {
		t.Errorf("sanear conexion dedicada de vinculo corporativo: %v", err)
	}
	if conexion.IsClosed() {
		return
	}
	ctx, cancelar := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancelar()
	if err := conexion.Close(ctx); err != nil {
		t.Errorf("cerrar conexion dedicada de vinculo corporativo: %v", err)
	}
}

func comprobarOptInRetiradaVinculoCorporativoRRHHV1Limpio(
	t *testing.T,
	ctx context.Context,
	conexion *pgx.Conn,
) {
	t.Helper()
	var observado string
	if err := conexion.QueryRow(ctx, `
		SELECT COALESCE(pg_catalog.current_setting($1,true),'')
	`, gucRetiradaVinculoCorporativoRRHHV1).Scan(&observado); err != nil {
		t.Fatalf("comprobar reset del opt-in: %v", err)
	}
	if observado != "" {
		t.Fatalf("el GUC conservo un valor residual tras el saneamiento: %q", observado)
	}
}

func comprobarVinculoCorporativoRRHHV1Instalado(
	t *testing.T,
	ctx context.Context,
	conexion *pgx.Conn,
) {
	t.Helper()
	var instalado bool
	if err := conexion.QueryRow(ctx, `
		SELECT pg_catalog.to_regclass(
		         'vec_contexto_actor_v1.vinculo_corporativo_versiones'
		       ) IS NOT NULL
		   AND pg_catalog.to_regclass(
		         'vec_contexto_actor_v1.vinculo_corporativo_actual'
		       ) IS NOT NULL
		   AND EXISTS (
		         SELECT 1 FROM pg_catalog.pg_constraint
		          WHERE connamespace='vec_contexto_actor_v1'::regnamespace
		            AND conname='perfil_versiones_persona_uq'
		       )
		   AND EXISTS (
		         SELECT 1 FROM pg_catalog.pg_constraint
		          WHERE connamespace='vec_contexto_actor_v1'::regnamespace
		            AND conname='vinculo_contexto_versiones_actor_uq'
		       )
	`).Scan(&instalado); err != nil {
		t.Fatalf("comprobar instalacion 000004: %v", err)
	}
	if !instalado {
		t.Fatal("un rechazo o una cancelacion retiro parcialmente 000004")
	}
}

func comprobarVinculoCorporativoRRHHV1Retirado(
	t *testing.T,
	ctx context.Context,
	conexion *pgx.Conn,
) {
	t.Helper()
	var retirado, baseIntacta bool
	if err := conexion.QueryRow(ctx, `
		SELECT pg_catalog.to_regclass(
		         'vec_contexto_actor_v1.vinculo_corporativo_versiones'
		       ) IS NULL
		   AND pg_catalog.to_regclass(
		         'vec_contexto_actor_v1.vinculo_corporativo_actual'
		       ) IS NULL
		   AND NOT EXISTS (
		         SELECT 1 FROM pg_catalog.pg_constraint
		          WHERE connamespace='vec_contexto_actor_v1'::regnamespace
		            AND conname IN (
		              'perfil_versiones_persona_uq',
		              'vinculo_contexto_versiones_actor_uq'
		            )
		       ),
		       pg_catalog.to_regclass(
		         'vec_contexto_actor_v1.organizacion_versiones'
		       ) IS NOT NULL
		   AND pg_catalog.to_regclass(
		         'vec_contexto_actor_v1.control_generacion_punteros_actuales_v2'
		       ) IS NOT NULL
	`).Scan(&retirado, &baseIntacta); err != nil {
		t.Fatalf("comprobar postcondicion de retirada: %v", err)
	}
	if !retirado || !baseIntacta {
		t.Fatalf("postcondicion incoherente: retirada=%t base=%t", retirado, baseIntacta)
	}
}

func comprobarVinculoCorporativoRRHHV1Vacio(
	t *testing.T,
	ctx context.Context,
	conexion *pgx.Conn,
) {
	t.Helper()
	var filas int64
	if err := conexion.QueryRow(ctx, `
		SELECT (SELECT pg_catalog.count(*)
		          FROM vec_contexto_actor_v1.vinculo_corporativo_versiones)
		     + (SELECT pg_catalog.count(*)
		          FROM vec_contexto_actor_v1.vinculo_corporativo_actual)
	`).Scan(&filas); err != nil {
		t.Fatalf("comprobar instalacion corporativa vacia: %v", err)
	}
	if filas != 0 {
		t.Fatalf("la retirada valida requiere instalacion vacia; filas=%d", filas)
	}
}

func leerDocumentoVinculoCorporativoRRHHV1(t *testing.T, sentido string) []byte {
	t.Helper()
	_, archivoPrueba, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("no se pudo localizar el archivo de prueba")
	}
	ruta := filepath.Join(
		filepath.Clean(filepath.Join(filepath.Dir(archivoPrueba), "../../../../..")),
		"deploy/postgresql/contexto_actor_v1/migraciones",
		"000004_vinculo_corporativo_rrhh_v1."+sentido+".sql",
	)
	documento, err := os.ReadFile(ruta)
	if err != nil {
		t.Fatalf("leer bytes reales del documento 000004 %s: %v", sentido, err)
	}
	return documento
}

func asegurarVinculoCorporativoRRHHV1InstaladoAlFinal(
	t *testing.T,
	dsn string,
	up []byte,
) {
	t.Helper()
	t.Cleanup(func() {
		ctx, cancelar := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancelar()
		conexion, err := pgx.Connect(ctx, dsn)
		if err != nil {
			t.Errorf("abrir conexion de restauracion 000004: %v", err)
			return
		}
		defer func() { _ = conexion.Close(ctx) }()
		var instalado bool
		if err = conexion.QueryRow(ctx, `
			SELECT pg_catalog.to_regclass(
			  'vec_contexto_actor_v1.vinculo_corporativo_actual'
			) IS NOT NULL
		`).Scan(&instalado); err != nil {
			t.Errorf("comprobar restauracion 000004: %v", err)
			return
		}
		if instalado {
			return
		}
		if _, err = conexion.Exec(ctx, string(up)); err != nil {
			t.Errorf("restaurar 000004 al finalizar prueba: %v", err)
		}
	})
}
