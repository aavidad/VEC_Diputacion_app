package postgresimportacionconvoca

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

func TestIntegracionPostgreSQLPoliticaGobernadaEncadenada(t *testing.T) {
	entorno := abrirEntornoPostgreSQLIntegracion(t)
	ctx, cancelar := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancelar()
	referencia := "politica:retencion:convoca:gobernada"
	_ = repositorioIntegracion(t, entorno, referencia, 48*time.Hour)
	var integra bool
	if err := entorno.admin.QueryRow(ctx, `
		SELECT vec_bolsa_importacion_convoca.politica_retencion_integra()`,
	).Scan(&integra); err != nil || !integra {
		t.Fatalf("politica inicial no integra: integra=%v error=%v", integra, err)
	}
	_, err := entorno.gobernanza.Exec(ctx, `
		SELECT vec_bolsa_importacion_convoca.publicar_politica_retencion_v1(
		    $1, 3, 172800, 'actor:gobernanza:integracion'
		)`, referencia)
	if err == nil {
		t.Fatal("salto de version de politica aceptado")
	}
	exigirSQLState(t, err, "B1703")
	_, err = entorno.gobernanza.Exec(ctx, `
		SELECT vec_bolsa_importacion_convoca.publicar_politica_retencion_v1(
		    $1, 2, 259200, 'actor:gobernanza:integracion'
		)`, referencia)
	if err != nil {
		t.Fatalf("sucesion contigua de politica rechazada: %v", err)
	}
	_, err = entorno.gobernanza.Exec(ctx, `
		SELECT vec_bolsa_importacion_convoca.publicar_politica_retencion_v1(
		    $1, 3, 259200, 'actor/gobernanza/invalido'
		)`, referencia)
	if err == nil {
		t.Fatal("politica acepto un actor fuera del contrato Go")
	}
	exigirSQLState(t, err, "B1701")
	_, err = entorno.gobernanza.Exec(ctx, `
		SELECT vec_bolsa_importacion_convoca.publicar_politica_retencion_v1(
		    $1, 1, 172800, 'actor:gobernanza:integracion'
		)`, referencia)
	if err == nil {
		t.Fatal("retroceso de politica aceptado")
	}
	exigirSQLState(t, err, "B1703")
	if _, err := entorno.admin.Exec(ctx, `
		UPDATE vec_bolsa_importacion_convoca.politica_retencion
		   SET duracion_segundos = duracion_segundos`); err == nil {
		t.Fatal("politica append-only permitio UPDATE")
	} else {
		exigirSQLState(t, err, "55000")
	}
	if _, err := entorno.admin.Exec(ctx, `
		UPDATE vec_bolsa_importacion_convoca.politica_retencion_actual
		   SET secuencia_publicacion = secuencia_publicacion - 1`); err == nil {
		t.Fatal("puntero de politica permitio retroceso")
	} else {
		exigirSQLState(t, err, "55000")
	}
	txManipulacion, err := entorno.admin.Begin(ctx)
	if err != nil {
		t.Fatalf("abrir manipulacion de cadena de politica: %v", err)
	}
	if _, err := txManipulacion.Exec(
		ctx, `SET LOCAL session_replication_role = replica`,
	); err != nil {
		_ = txManipulacion.Rollback(context.Background())
		t.Fatalf("aislar manipulacion de politica: %v", err)
	}
	if _, err := txManipulacion.Exec(ctx, `
		UPDATE vec_bolsa_importacion_convoca.politica_retencion
		   SET duracion_segundos = duracion_segundos + 1
		 WHERE politica_retencion_ref = $1
		   AND politica_retencion_version = 2`, referencia); err != nil {
		_ = txManipulacion.Rollback(context.Background())
		t.Fatalf("manipular politica sintetica: %v", err)
	}
	if err := txManipulacion.QueryRow(ctx, `
		SELECT vec_bolsa_importacion_convoca.politica_retencion_integra()`,
	).Scan(&integra); err != nil || integra {
		_ = txManipulacion.Rollback(context.Background())
		t.Fatalf("verificador acepto politica manipulada: integra=%v error=%v",
			integra, err)
	}
	if err := txManipulacion.Rollback(ctx); err != nil {
		t.Fatalf("revertir manipulacion de politica: %v", err)
	}
	tx, err := entorno.admin.Begin(ctx)
	if err != nil {
		t.Fatalf("abrir prueba de politica huerfana: %v", err)
	}
	defer func() { _ = tx.Rollback(context.Background()) }()
	_, err = tx.Exec(ctx, `
		INSERT INTO vec_bolsa_importacion_convoca.politica_retencion
		SELECT 'politica:retencion:convoca:huerfana', 1, 3600,
		       actual.secuencia_publicacion + 1,
		       politica.huella_publicacion_sha256,
		       date_trunc('microseconds', clock_timestamp()),
		       'actor:gobernanza:integracion', repeat('a',64)
		  FROM vec_bolsa_importacion_convoca.politica_retencion_actual AS actual
		  JOIN vec_bolsa_importacion_convoca.politica_retencion AS politica
		    ON politica.politica_retencion_ref =
		       actual.politica_retencion_ref
		   AND politica.politica_retencion_version =
		       actual.politica_retencion_version`)
	if err != nil {
		t.Fatalf("crear huerfana sintetica: %v", err)
	}
	_, err = tx.Exec(ctx, `
		SELECT vec_bolsa_importacion_convoca.publicar_politica_retencion_v1(
		    'politica:retencion:convoca:huerfana', 1, 3600,
		    'actor:gobernanza:integracion'
		)`)
	if err == nil {
		t.Fatal("politica huerfana se reutilizo")
	}
	exigirSQLState(t, err, "B1703")
}

func forzarVencimientoIntegracion(
	t *testing.T,
	ctx context.Context,
	entorno *entornoPostgreSQLIntegracion,
	importacionRef string,
) {
	t.Helper()
	tx, err := entorno.admin.Begin(ctx)
	if err != nil {
		t.Fatalf("abrir viaje temporal sintetico: %v", err)
	}
	defer func() { _ = tx.Rollback(context.Background()) }()
	if _, err := tx.Exec(
		ctx, `SET LOCAL session_replication_role = replica`,
	); err != nil {
		t.Fatalf("aislar viaje temporal sintetico: %v", err)
	}
	_, err = tx.Exec(ctx, `
		WITH instante AS (
		    SELECT date_trunc(
		        'microseconds', clock_timestamp() - interval '3 hours'
		    ) AS valor
		), acta AS (
		    SELECT jsonb_set(
		        lote.acta_canonica, '{registrada_en}',
		        to_jsonb(to_char(
		            instante.valor AT TIME ZONE 'UTC',
		            'YYYY-MM-DD"T"HH24:MI:SS.US"Z"'
		        )), false
		    ) AS valor, instante.valor AS registrada
		      FROM vec_bolsa_importacion_convoca.lote AS lote, instante
		     WHERE lote.importacion_ref = $1
		), evento AS (
		    SELECT historia.evento_ref,
		           vec_bolsa_importacion_convoca.huella_evento_estado(
		               historia.evento_ref, historia.importacion_ref,
		               historia.secuencia, historia.huella_anterior_sha256,
		               historia.tipo, historia.actor_ref,
		               historia.estado_conciliacion,
		               historia.estado_staging,
		               historia.bloqueo_retencion, acta.registrada
		           ) AS huella, acta.registrada
		      FROM vec_bolsa_importacion_convoca.historia_estado AS historia,
		           acta
		     WHERE historia.importacion_ref = $1
		       AND historia.secuencia = 1
		), actualizar_historia AS (
		    UPDATE vec_bolsa_importacion_convoca.historia_estado AS historia
		       SET registrada_en = evento.registrada,
		           huella_evento_sha256 = evento.huella
		      FROM evento WHERE historia.evento_ref = evento.evento_ref
		      RETURNING evento.huella
		)
		UPDATE vec_bolsa_importacion_convoca.lote AS lote
		   SET acta_canonica = acta.valor,
		       huella_acta_sha256 = encode(
		           sha256(convert_to(acta.valor::text, 'UTF8')), 'hex'
		       ),
		       registrada_en = acta.registrada,
		       conservar_staging_hasta = acta.registrada + interval '1 hour',
		       cabeza_historia_sha256 = actualizar_historia.huella
		  FROM acta, actualizar_historia
		 WHERE lote.importacion_ref = $1`, importacionRef)
	if err != nil {
		t.Fatalf("preparar vencimiento sintetico: %v", err)
	}
	var integra bool
	if err := tx.QueryRow(ctx, `
		SELECT vec_bolsa_importacion_convoca.lote_integro($1)`,
		importacionRef,
	).Scan(&integra); err != nil || !integra {
		t.Fatalf("viaje temporal rompio integridad: integra=%v error=%v",
			integra, err)
	}
	if err := tx.Commit(ctx); err != nil {
		t.Fatalf("confirmar vencimiento sintetico: %v", err)
	}
}

func TestIntegracionPostgreSQLRollbackLimitesRLSYPrivilegios(t *testing.T) {
	entorno := abrirEntornoPostgreSQLIntegracion(t)
	ctx, cancelar := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancelar()
	assertPostgreSQL18TLSYRLS(t, ctx, entorno)
	assertSegregacionRoles(t, ctx, entorno)
	assertRollbackAtomico(t, ctx, entorno)
	assertLimitesSQL(t, ctx, entorno)
}

func TestIntegracionPostgreSQLInsercionSetBasedEnMaximoDeclarado(t *testing.T) {
	entorno := abrirEntornoPostgreSQLIntegracion(t)
	ctx, cancelar := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancelar()
	publicarPoliticaIntegracion(
		t, entorno, "politica:retencion:convoca:maximo", 1, 365*24*time.Hour,
	)
	lote := loteIntegracion(
		huellaIntegracion("4"), time.Date(2026, 7, 26, 10, 0, 0, 0, time.UTC), 0,
	)
	lote.Acta.FilasLeidas = maximoFilasStaging
	lote.Acta.FilasAceptadas = maximoFilasStaging
	acta, err := serializarActa(lote.Acta)
	if err != nil {
		t.Fatalf("serializar acta maxima: %v", err)
	}
	defer borrarBytes(acta)
	tx, err := entorno.admin.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		t.Fatalf("abrir insercion maxima: %v", err)
	}
	defer func() { _ = tx.Rollback(context.Background()) }()
	if _, err := tx.Exec(
		ctx, `SET LOCAL ROLE vec_convoca_ejecutor_prueba`,
	); err != nil {
		t.Fatalf("adoptar identidad ejecutora para maximo: %v", err)
	}
	inicio := time.Now()
	var reutilizada bool
	err = tx.QueryRow(ctx, `
		WITH filas AS (
		    SELECT jsonb_agg(jsonb_build_object(
		        'numero', numero,
		        'esquema_proteccion',
		          'vec.bolsa.importacion-convoca.proteccion-staging.v1',
		        'clave_ref', 'kms:c:v1',
		        'clave_derivacion_ref', 'kms:d:v1',
		        'clave_atestacion_ref', 'kms:a:v1',
		        'nonce_hex', repeat('00', 12),
		        'contenido_cifrado_hex', repeat('00', 16),
		        'huella_contenido_cifrado_sha256',
		          encode(sha256(decode(repeat('00', 16), 'hex')), 'hex'),
		        'derivacion_documento_hmac_sha256', repeat('01', 32),
		        'atestacion_fila_hmac_sha256', repeat('02', 32)
		    ) ORDER BY numero) AS valor
		      FROM generate_series(2, 100002) AS numero
		)
		SELECT reutilizada
		  FROM filas,
		       vec_bolsa_importacion_convoca.guardar_lote_v1(
		           $1::jsonb, filas.valor
		       )`,
		json.RawMessage(acta),
	).Scan(&reutilizada)
	if err != nil || reutilizada {
		t.Fatalf("insercion set-based maxima: reutilizada=%v duracion=%s error=%v",
			reutilizada, time.Since(inicio), err)
	}
	if _, err := tx.Exec(ctx, `RESET ROLE`); err != nil {
		t.Fatalf("restaurar identidad administrativa de prueba: %v", err)
	}
	var filas int
	if err := tx.QueryRow(ctx, `
		SELECT count(*) FROM vec_bolsa_importacion_convoca.fila_staging
		 WHERE importacion_ref = $1`, lote.Acta.ImportacionRef,
	).Scan(&filas); err != nil || filas != maximoFilasStaging {
		t.Fatalf("cardinalidad maxima: filas=%d error=%v", filas, err)
	}
	t.Logf("insercion set-based de %d filas: %s", filas, time.Since(inicio))
}

func TestPrepararRecuperacionPostgreSQLTrasReinicio(t *testing.T) {
	entorno := abrirEntornoPostgreSQLIntegracion(t)
	ctx, cancelar := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancelar()
	repositorio := repositorioIntegracion(
		t, entorno, "politica:retencion:convoca:reinicio", 365*24*time.Hour,
	)
	lote := loteIntegracion(
		huellaIntegracion("f"), time.Date(2026, 7, 26, 10, 0, 0, 0, time.UTC), 1,
	)
	if _, _, err := repositorio.GuardarSiAusente(ctx, lote); err != nil {
		t.Fatalf("preparar reinicio: %v", err)
	}
}

func TestRecuperacionPostgreSQLTrasReinicio(t *testing.T) {
	if strings.TrimSpace(os.Getenv("VEC_PRUEBA_BOLSA_CONVOCA_TRAS_REINICIO")) != "1" {
		t.Skip("fase posterior al reinicio no solicitada")
	}
	entorno := abrirEntornoPostgreSQLIntegracion(t)
	ctx, cancelar := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancelar()
	recuperador := recuperadorIntegracion(t, entorno)
	huella := huellaIntegracion("f")
	lote, estado, existe, err := recuperador.RecuperarLote(ctx, huella)
	if err != nil || !existe || estado.Acta.HuellaFicheroSHA256 != huella ||
		len(lote.Aceptadas) != 1 {
		t.Fatalf("recuperacion tras reinicio: existe=%v estado=%+v lote=%+v error=%v",
			existe, estado, lote, err)
	}
}

func assertPostgreSQL18TLSYRLS(
	t *testing.T,
	ctx context.Context,
	entorno *entornoPostgreSQLIntegracion,
) {
	t.Helper()
	var version int
	var tls bool
	if err := entorno.ejecutor.QueryRow(ctx, `
		SELECT current_setting('server_version_num')::integer,
		       (SELECT ssl FROM pg_catalog.pg_stat_ssl
		         WHERE pid = pg_backend_pid())`).Scan(&version, &tls); err != nil {
		t.Fatalf("consultar version/TLS: %v", err)
	}
	if version < 180000 || version >= 190000 || !tls {
		t.Fatalf("entorno no es PostgreSQL 18 con TLS: version=%d tls=%v", version, tls)
	}
	var protegidas int
	if err := entorno.admin.QueryRow(ctx, `
		SELECT count(*)
		  FROM pg_catalog.pg_class AS tabla
		  JOIN pg_catalog.pg_namespace AS esquema ON esquema.oid = tabla.relnamespace
		 WHERE esquema.nspname = 'vec_bolsa_importacion_convoca'
		   AND tabla.relkind = 'r'
		   AND tabla.relrowsecurity AND tabla.relforcerowsecurity`).Scan(&protegidas); err != nil ||
		protegidas != 9 {
		t.Fatalf("RLS no forzada en nueve tablas: tablas=%d error=%v", protegidas, err)
	}
	var funcionesCerradas, funcionesDefiner int
	if err := entorno.admin.QueryRow(ctx, `
		SELECT count(*) FILTER (
		           WHERE proconfig @> ARRAY['search_path=pg_catalog']
		             AND NOT EXISTS (
		                 SELECT 1 FROM unnest(proconfig) AS opcion
		                  WHERE opcion LIKE 'search_path=%'
		                    AND opcion <> 'search_path=pg_catalog'
		             )
		       ),
		       count(*)
		  FROM pg_catalog.pg_proc AS funcion
		  JOIN pg_catalog.pg_namespace AS esquema
		    ON esquema.oid = funcion.pronamespace
		 WHERE esquema.nspname = 'vec_bolsa_importacion_convoca'
		   AND funcion.prosecdef`,
	).Scan(&funcionesCerradas, &funcionesDefiner); err != nil ||
		funcionesCerradas != funcionesDefiner {
		t.Fatalf("search_path SECURITY DEFINER abierto: cerradas=%d total=%d error=%v",
			funcionesCerradas, funcionesDefiner, err)
	}
	var propietarioPuedeCrear bool
	if err := entorno.admin.QueryRow(ctx, `
		SELECT has_database_privilege(
		    'vec_bolsa_importacion_convoca_propietario',
		    current_database(), 'CREATE'
		)`).Scan(&propietarioPuedeCrear); err != nil || propietarioPuedeCrear {
		t.Fatalf("propietario conserva CREATE en base: permitido=%v error=%v",
			propietarioPuedeCrear, err)
	}
	var truncadosProtegidos int
	if err := entorno.admin.QueryRow(ctx, `
		SELECT count(DISTINCT tabla.oid)
		  FROM pg_catalog.pg_class AS tabla
		  JOIN pg_catalog.pg_namespace AS esquema
		    ON esquema.oid = tabla.relnamespace
		  JOIN pg_catalog.pg_trigger AS disparador
		    ON disparador.tgrelid = tabla.oid
		 WHERE esquema.nspname = 'vec_bolsa_importacion_convoca'
		   AND tabla.relkind = 'r' AND NOT disparador.tgisinternal
		   AND disparador.tgtype & 32 = 32`,
	).Scan(&truncadosProtegidos); err != nil || truncadosProtegidos != 9 {
		t.Fatalf("proteccion TRUNCATE incompleta: tablas=%d error=%v",
			truncadosProtegidos, err)
	}
	if _, err := entorno.admin.Exec(
		ctx, `TRUNCATE vec_bolsa_importacion_convoca.outbox`,
	); err == nil {
		t.Fatal("TRUNCATE privilegiado no fue bloqueado")
	} else {
		exigirSQLState(t, err, "55000")
	}
}

func assertSegregacionRoles(
	t *testing.T,
	ctx context.Context,
	entorno *entornoPostgreSQLIntegracion,
) {
	t.Helper()
	for nombre, prueba := range map[string]func() error{
		"ejecutor lee lote": func() error {
			_, err := entorno.ejecutor.Exec(ctx,
				`SELECT importacion_ref FROM vec_bolsa_importacion_convoca.lote`)
			return err
		},
		"ejecutor concilia": func() error {
			_, err := entorno.ejecutor.Exec(ctx,
				`SELECT vec_bolsa_importacion_convoca.conciliar_v1(
				 'importacion:convoca:' || repeat('a',64), 'conciliacion:prueba:acl',
				 'registro:corporativo:acl', 'confirmada', 'actor:rrhh:acl', 'motivo_acl')`)
			return err
		},
		"ejecutor recupera staging": func() error {
			_, err := entorno.ejecutor.Exec(ctx,
				`SELECT vec_bolsa_importacion_convoca.recuperar_lote_pagina_v1(
				    repeat('a',64), 2, 1
				)`)
			return err
		},
		"conciliador recupera staging": func() error {
			_, err := entorno.conciliador.Exec(ctx,
				`SELECT vec_bolsa_importacion_convoca.recuperar_lote_pagina_v1(
				    repeat('a',64), 2, 1
				)`)
			return err
		},
		"recuperador importa": func() error {
			_, err := entorno.recuperador.Exec(ctx,
				`SELECT * FROM vec_bolsa_importacion_convoca.guardar_lote_v1(
				    '{}'::jsonb, '[]'::jsonb
				)`)
			return err
		},
		"gobernanza importa": func() error {
			_, err := entorno.gobernanza.Exec(ctx,
				`SELECT * FROM vec_bolsa_importacion_convoca.guardar_lote_v1(
				    '{}'::jsonb, '[]'::jsonb
				)`)
			return err
		},
		"retencion lee staging": func() error {
			_, err := entorno.retencion.Exec(ctx,
				`SELECT contenido_cifrado FROM vec_bolsa_importacion_convoca.fila_staging`)
			return err
		},
	} {
		if err := prueba(); err == nil {
			t.Fatalf("segregacion ausente: %s", nombre)
		} else {
			exigirSQLState(t, err, "42501")
		}
	}
}

func assertRollbackAtomico(
	t *testing.T,
	ctx context.Context,
	entorno *entornoPostgreSQLIntegracion,
) {
	t.Helper()
	_, err := entorno.admin.Exec(ctx, `
		CREATE FUNCTION vec_bolsa_importacion_convoca.fallar_segunda_fila_prueba()
		RETURNS trigger LANGUAGE plpgsql AS $$
		BEGIN
		    IF NEW.numero = 3 THEN RAISE EXCEPTION 'fallo sintetico de rollback'; END IF;
		    RETURN NEW;
		END $$;
		CREATE TRIGGER fallar_segunda_fila_prueba
		BEFORE INSERT ON vec_bolsa_importacion_convoca.fila_staging
		FOR EACH ROW EXECUTE FUNCTION
		    vec_bolsa_importacion_convoca.fallar_segunda_fila_prueba()`)
	if err != nil {
		t.Fatalf("instalar fallo sintetico: %v", err)
	}
	defer func() {
		_, _ = entorno.admin.Exec(context.Background(), `
			DROP TRIGGER IF EXISTS fallar_segunda_fila_prueba
			    ON vec_bolsa_importacion_convoca.fila_staging;
			DROP FUNCTION IF EXISTS
			    vec_bolsa_importacion_convoca.fallar_segunda_fila_prueba()`)
	}()
	repositorio := repositorioIntegracion(
		t, entorno, "politica:retencion:convoca:rollback", 365*24*time.Hour,
	)
	lote := loteIntegracion(
		huellaIntegracion("e"), time.Date(2026, 7, 26, 10, 0, 0, 0, time.UTC), 2,
	)
	if _, _, err := repositorio.GuardarSiAusente(ctx, lote); err == nil {
		t.Fatal("fallo en segunda fila no revirtio transaccion")
	}
	var lotes, filas, historia int
	if err := entorno.admin.QueryRow(ctx, `
		SELECT (SELECT count(*) FROM vec_bolsa_importacion_convoca.lote
		         WHERE importacion_ref = $1),
		       (SELECT count(*) FROM vec_bolsa_importacion_convoca.fila_staging
		         WHERE importacion_ref = $1),
		       (SELECT count(*) FROM vec_bolsa_importacion_convoca.historia_estado
		         WHERE importacion_ref = $1)`, lote.Acta.ImportacionRef,
	).Scan(&lotes, &filas, &historia); err != nil ||
		lotes != 0 || filas != 0 || historia != 0 {
		t.Fatalf("rollback parcial: lotes=%d filas=%d historia=%d error=%v",
			lotes, filas, historia, err)
	}
}

func assertLimitesSQL(
	t *testing.T,
	ctx context.Context,
	entorno *entornoPostgreSQLIntegracion,
) {
	t.Helper()
	lote := loteIntegracion(
		huellaIntegracion("9"), time.Date(2026, 7, 26, 10, 0, 0, 0, time.UTC), 1,
	)
	acta, err := serializarActa(lote.Acta)
	if err != nil {
		t.Fatalf("serializar acta de limite: %v", err)
	}
	cifrado := strings.Repeat("00", maximoBytesCifradosPorFila+1)
	filas := fmt.Sprintf(`[{
		"numero":2,
		"esquema_proteccion":"%s",
		"clave_ref":"kms:fixture:convoca:b1:v1",
		"nonce_hex":"%s",
		"contenido_cifrado_hex":"%s",
		"huella_contenido_cifrado_sha256":"%s",
		"derivacion_documento_hmac_sha256":"%s",
		"clave_derivacion_ref":"kms:fixture:convoca:derivacion:v1",
		"clave_atestacion_ref":"kms:fixture:convoca:atestacion:v1",
		"atestacion_fila_hmac_sha256":"%s"
	}]`, EsquemaProteccionStagingV1, strings.Repeat("00", 12), cifrado,
		strings.Repeat("a", 64), strings.Repeat("01", 32),
		strings.Repeat("02", 32))
	_, err = entorno.ejecutor.Exec(ctx, `
		SELECT * FROM vec_bolsa_importacion_convoca.guardar_lote_v1(
		    $1::jsonb, $2::jsonb
		)`, json.RawMessage(acta), json.RawMessage(filas))
	if err == nil {
		t.Fatal("PostgreSQL acepto contenido cifrado excesivo")
	}
	var errorPG *pgconn.PgError
	if !errors.As(err, &errorPG) || errorPG.Code != "B1701" {
		t.Fatalf("limite fallo con causa inesperada: %v", err)
	}
	_, err = entorno.retencion.Exec(ctx, `
		SELECT vec_bolsa_importacion_convoca.expurgar_staging_vencido_v1(
		    'expurgo:convoca:limite', 'actor:archivo:limite',
		    'politica:retencion:convoca:limites', 1, 1001
		)`)
	if err == nil {
		t.Fatal("PostgreSQL acepto cardinalidad de expurgo excesiva")
	}
	exigirSQLState(t, err, "B1701")
}
