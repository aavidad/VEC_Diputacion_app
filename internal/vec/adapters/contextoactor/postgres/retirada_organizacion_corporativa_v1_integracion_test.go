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
	gucRetiradaOrganizacionCorporativaV1           = "vec.confirmar_retirada_organizacion_corporativa_v1"
	optInRetiradaOrganizacionCorporativaV1         = "RETIRAR_ORGANIZACION_CORPORATIVA_V1"
	consultaResetRetiradaOrganizacionCorporativaV1 = "RESET vec.confirmar_retirada_organizacion_corporativa_v1"
)

// TestRetiradaOrganizacionCorporativaV1EjecutaDocumentoSQLIntegro acredita
// que una aplicacion puede ejecutar los bytes publicados del down sin
// preprocesarlos. La conexion es dedicada: un rechazo del propio documento
// deja abierta una transaccion abortada que debe sanearse antes de reutilizarla.
func TestRetiradaOrganizacionCorporativaV1EjecutaDocumentoSQLIntegro(t *testing.T) {
	dsn := os.Getenv("VEC_CONTEXTO_ACTOR_MIGRACION_POSTGRES_DSN")
	if dsn == "" {
		t.Skip("requiere PostgreSQL 18.4 efimero de contexto actor")
	}
	down := leerDocumentoOrganizacionCorporativaV1(t, "down")
	up := leerDocumentoOrganizacionCorporativaV1(t, "up")
	asegurarOrganizacionCorporativaV1InstaladaAlFinal(t, dsn, up)

	ctx, cancelar := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancelar()
	conexion := abrirConexionRetiradaAcreditacionUsoV2(t, ctx, dsn, "de organizacion")
	defer cerrarConexionRetiradaOrganizacionCorporativaV1(t, conexion)

	var version int
	if err := conexion.QueryRow(ctx, `
		SELECT pg_catalog.current_setting('server_version_num')::integer
	`).Scan(&version); err != nil {
		t.Fatalf("consultar version PostgreSQL: %v", err)
	}
	if version != 180004 {
		t.Fatalf("requiere PostgreSQL 18.4 exacto; servidor conectado: %d", version)
	}

	probarRechazoRetiradaOrganizacionCorporativaV1(t, ctx, conexion, down, nil)
	incorrecta := "RETIRAR_SIN_CONTRATO"
	probarRechazoRetiradaOrganizacionCorporativaV1(
		t, ctx, conexion, down, &incorrecta,
	)
	comprobarOrganizacionCorporativaV1Vacia(t, ctx, conexion)

	configurarOptInRetiradaOrganizacionCorporativaV1(
		t, ctx, conexion, optInRetiradaOrganizacionCorporativaV1,
	)
	if _, err := conexion.Exec(ctx, string(down)); err != nil {
		t.Fatalf("ejecutar documento integro con opt-in exacto: %v", err)
	}
	if estado := conexion.PgConn().TxStatus(); estado != 'I' {
		t.Fatalf("el COMMIT del documento no dejo la conexion libre: %q", estado)
	}
	if err := sanearConexionRetiradaOrganizacionCorporativaV1(conexion); err != nil {
		t.Fatalf("sanear conexion tras retirada valida: %v", err)
	}
	comprobarOrganizacionCorporativaV1Retirada(t, ctx, conexion)
	if _, err := conexion.Exec(ctx, string(up)); err != nil {
		t.Fatalf("restaurar 000003 tras retirada valida: %v", err)
	}
	comprobarOrganizacionCorporativaV1Instalada(t, ctx, conexion)

	probarCancelacionRetiradaOrganizacionCorporativaV1(t, ctx, dsn, down)
}

// TestRetiradaOrganizacionCorporativaV1DestruyeConexionQueNoPuedeSanear
// demuestra que una conexion dedicada no vuelve a ningun consumidor cuando
// ni siquiera puede ejecutar el rollback y el reset obligatorios.
func TestRetiradaOrganizacionCorporativaV1DestruyeConexionQueNoPuedeSanear(
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

	configurarOptInRetiradaOrganizacionCorporativaV1(
		t, ctx, conexion, optInRetiradaOrganizacionCorporativaV1,
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
	if err := sanearConexionRetiradaOrganizacionCorporativaV1(conexion); err == nil {
		t.Fatal("el saneado aparento exito sobre un backend terminado")
	}
	if !conexion.IsClosed() {
		t.Fatal("la conexion no saneable no fue destruida")
	}
}

func probarCancelacionRetiradaOrganizacionCorporativaV1(
	t *testing.T,
	ctx context.Context,
	dsn string,
	down []byte,
) {
	t.Helper()
	bloqueador := abrirConexionRetiradaAcreditacionUsoV2(t, ctx, dsn, "bloqueadora")
	defer cerrarConexionRetiradaAcreditacionUsoV2(t, bloqueador)
	retirada := abrirConexionRetiradaAcreditacionUsoV2(t, ctx, dsn, "de retirada cancelable")
	defer cerrarConexionRetiradaOrganizacionCorporativaV1(t, retirada)
	supervisor := abrirConexionRetiradaAcreditacionUsoV2(t, ctx, dsn, "de supervision")
	defer cerrarConexionRetiradaAcreditacionUsoV2(t, supervisor)

	if _, err := bloqueador.Exec(ctx, `
		BEGIN;
		LOCK TABLE vec_contexto_actor_v1.organizacion_actual
		IN ACCESS EXCLUSIVE MODE
	`); err != nil {
		t.Fatalf("bloquear puntero organizativo: %v", err)
	}
	configurarOptInRetiradaOrganizacionCorporativaV1(
		t, ctx, retirada, optInRetiradaOrganizacionCorporativaV1,
	)
	pidRetirada := consultarPIDRetiradaAcreditacionUsoV2(t, ctx, retirada)
	ctxRetirada, cancelarRetirada := context.WithCancel(ctx)
	resultado := make(chan error, 1)
	go func() {
		_, err := retirada.Exec(ctxRetirada, string(down))
		resultado <- err
	}()
	esperarCondicionRetiradaAcreditacionUsoV2(
		t, ctx, supervisor, "down 000003 esperando un bloqueo real", `
			SELECT EXISTS (
			  SELECT 1 FROM pg_catalog.pg_locks
			   WHERE pid=$1
			     AND relation=
			       'vec_contexto_actor_v1.organizacion_actual'::regclass
			     AND mode='AccessExclusiveLock' AND NOT granted
			)
		`, pidRetirada,
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
	if err := sanearConexionRetiradaOrganizacionCorporativaV1(retirada); err != nil &&
		!retirada.IsClosed() {
		t.Fatalf("la conexion cancelada no quedo saneada ni destruida: %v", err)
	}
	if _, err := bloqueador.Exec(ctx, "ROLLBACK"); err != nil {
		t.Fatalf("liberar puntero organizativo: %v", err)
	}
	comprobarOrganizacionCorporativaV1Instalada(t, ctx, supervisor)
	if !retirada.IsClosed() {
		comprobarOptInRetiradaOrganizacionCorporativaV1Limpio(t, ctx, retirada)
	}
}

func probarRechazoRetiradaOrganizacionCorporativaV1(
	t *testing.T,
	ctx context.Context,
	conexion *pgx.Conn,
	documento []byte,
	optIn *string,
) {
	t.Helper()
	if optIn == nil {
		if err := sanearConexionRetiradaOrganizacionCorporativaV1(conexion); err != nil {
			t.Fatalf("preparar rechazo sin opt-in: %v", err)
		}
	} else {
		configurarOptInRetiradaOrganizacionCorporativaV1(t, ctx, conexion, *optIn)
	}
	if _, err := conexion.Exec(ctx, string(documento)); err == nil {
		t.Fatal("el documento acepto un opt-in ausente o incorrecto")
	}
	if estado := conexion.PgConn().TxStatus(); estado != 'E' {
		t.Fatalf("el rechazo no dejo observable la transaccion abortada: %q", estado)
	}
	if err := sanearConexionRetiradaOrganizacionCorporativaV1(conexion); err != nil {
		t.Fatalf("sanear conexion tras rechazo: %v", err)
	}
	comprobarOptInRetiradaOrganizacionCorporativaV1Limpio(t, ctx, conexion)
	comprobarOrganizacionCorporativaV1Instalada(t, ctx, conexion)
}

func configurarOptInRetiradaOrganizacionCorporativaV1(
	t *testing.T,
	ctx context.Context,
	conexion *pgx.Conn,
	valor string,
) {
	t.Helper()
	var observado string
	if err := conexion.QueryRow(
		ctx, "SELECT pg_catalog.set_config($1,$2,false)",
		gucRetiradaOrganizacionCorporativaV1, valor,
	).Scan(&observado); err != nil {
		t.Fatalf("configurar opt-in de sesion: %v", err)
	}
	if observado != valor {
		t.Fatalf("opt-in observado distinto: %q", observado)
	}
}

func sanearConexionRetiradaOrganizacionCorporativaV1(conexion *pgx.Conn) error {
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
		ctx, consultaResetRetiradaOrganizacionCorporativaV1,
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
	`, gucRetiradaOrganizacionCorporativaV1).Scan(&observado); err != nil {
		_ = conexion.Close(ctx)
		return err
	}
	if observado == optInRetiradaOrganizacionCorporativaV1 {
		_ = conexion.Close(ctx)
		return fmt.Errorf("el opt-in sobrevivio al reset de sesion")
	}
	return nil
}

func cerrarConexionRetiradaOrganizacionCorporativaV1(
	t *testing.T,
	conexion *pgx.Conn,
) {
	t.Helper()
	if err := sanearConexionRetiradaOrganizacionCorporativaV1(conexion); err != nil {
		t.Errorf("sanear conexion dedicada de organizacion: %v", err)
	}
	if conexion.IsClosed() {
		return
	}
	ctx, cancelar := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancelar()
	if err := conexion.Close(ctx); err != nil {
		t.Errorf("cerrar conexion dedicada de organizacion: %v", err)
	}
}

func comprobarOptInRetiradaOrganizacionCorporativaV1Limpio(
	t *testing.T,
	ctx context.Context,
	conexion *pgx.Conn,
) {
	t.Helper()
	var observado string
	if err := conexion.QueryRow(ctx, `
		SELECT COALESCE(pg_catalog.current_setting($1,true),'')
	`, gucRetiradaOrganizacionCorporativaV1).Scan(&observado); err != nil {
		t.Fatalf("comprobar reset del opt-in: %v", err)
	}
	if observado == optInRetiradaOrganizacionCorporativaV1 {
		t.Fatal("el opt-in exacto sobrevivio al saneamiento")
	}
}

func comprobarOrganizacionCorporativaV1Instalada(
	t *testing.T,
	ctx context.Context,
	conexion *pgx.Conn,
) {
	t.Helper()
	var instalada bool
	if err := conexion.QueryRow(ctx, `
		SELECT pg_catalog.to_regprocedure(
		         'vec_contexto_actor_v1.organizacion_ref_valida(text)'
		       ) IS NOT NULL
		   AND pg_catalog.to_regclass(
		         'vec_contexto_actor_v1.organizacion_versiones'
		       ) IS NOT NULL
		   AND pg_catalog.to_regclass(
		         'vec_contexto_actor_v1.organizacion_actual'
		       ) IS NOT NULL
	`).Scan(&instalada); err != nil {
		t.Fatalf("comprobar instalacion 000003: %v", err)
	}
	if !instalada {
		t.Fatal("un rechazo o una cancelacion retiro parcialmente 000003")
	}
}

func comprobarOrganizacionCorporativaV1Retirada(
	t *testing.T,
	ctx context.Context,
	conexion *pgx.Conn,
) {
	t.Helper()
	var retirada, baseIntacta bool
	if err := conexion.QueryRow(ctx, `
		SELECT pg_catalog.to_regprocedure(
		         'vec_contexto_actor_v1.organizacion_ref_valida(text)'
		       ) IS NULL
		   AND pg_catalog.to_regclass(
		         'vec_contexto_actor_v1.organizacion_versiones'
		       ) IS NULL
		   AND pg_catalog.to_regclass(
		         'vec_contexto_actor_v1.organizacion_actual'
		       ) IS NULL,
		       pg_catalog.to_regclass(
		         'vec_contexto_actor_v1.control_generacion_punteros_actuales_v2'
		       ) IS NOT NULL
	`).Scan(&retirada, &baseIntacta); err != nil {
		t.Fatalf("comprobar postcondicion de retirada: %v", err)
	}
	if !retirada || !baseIntacta {
		t.Fatalf("postcondicion incoherente: retirada=%t base=%t", retirada, baseIntacta)
	}
}

func comprobarOrganizacionCorporativaV1Vacia(
	t *testing.T,
	ctx context.Context,
	conexion *pgx.Conn,
) {
	t.Helper()
	var filas int64
	if err := conexion.QueryRow(ctx, `
		SELECT (SELECT pg_catalog.count(*)
		          FROM vec_contexto_actor_v1.organizacion_versiones)
		     + (SELECT pg_catalog.count(*)
		          FROM vec_contexto_actor_v1.organizacion_actual)
	`).Scan(&filas); err != nil {
		t.Fatalf("comprobar instalacion organizativa vacia: %v", err)
	}
	if filas != 0 {
		t.Fatalf("la retirada valida requiere instalacion vacia; filas=%d", filas)
	}
}

func leerDocumentoOrganizacionCorporativaV1(t *testing.T, sentido string) []byte {
	t.Helper()
	_, archivoPrueba, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("no se pudo localizar el archivo de prueba")
	}
	ruta := filepath.Join(
		filepath.Clean(filepath.Join(filepath.Dir(archivoPrueba), "../../../../..")),
		"deploy/postgresql/contexto_actor_v1/migraciones",
		"000003_organizacion_corporativa_v1."+sentido+".sql",
	)
	documento, err := os.ReadFile(ruta)
	if err != nil {
		t.Fatalf("leer bytes reales del documento 000003 %s: %v", sentido, err)
	}
	return documento
}

func asegurarOrganizacionCorporativaV1InstaladaAlFinal(
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
			t.Errorf("abrir conexion de restauracion 000003: %v", err)
			return
		}
		defer func() { _ = conexion.Close(ctx) }()
		var instalada bool
		if err = conexion.QueryRow(ctx, `
			SELECT pg_catalog.to_regclass(
			  'vec_contexto_actor_v1.organizacion_actual'
			) IS NOT NULL
		`).Scan(&instalada); err != nil {
			t.Errorf("comprobar restauracion 000003: %v", err)
			return
		}
		if instalada {
			return
		}
		if _, err = conexion.Exec(ctx, string(up)); err != nil {
			t.Errorf("restaurar 000003 al finalizar prueba: %v", err)
		}
	})
}
