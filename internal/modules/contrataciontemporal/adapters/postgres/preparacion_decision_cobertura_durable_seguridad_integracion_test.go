package postgres

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/cobertura"
)

type filaReservaTerminalPreparacionDecisionCobertura struct {
	reservaRef        string
	reciboRef         string
	actuacionRef      string
	auditoriaRef      string
	eventoRef         string
	correlacionRef    string
	decisionVECRef    string
	versionExpediente uint64
	secuencia         int64
	revision          int64
	tokenSHA256       string
	observadaEn       time.Time
	propiedadHasta    time.Time
}

func confirmarTerminalPreparacionDecisionCobertura(
	t *testing.T,
	ctx context.Context,
	admin *pgxpool.Pool,
	par parHMACPreparacionDecisionCoberturaPrueba,
	rama string,
) {
	t.Helper()
	tx, err := admin.BeginTx(ctx, pgx.TxOptions{
		IsoLevel: pgx.Serializable, AccessMode: pgx.ReadWrite,
	})
	if err != nil {
		t.Fatal(err)
	}
	defer revertirTransaccion(tx)
	if _, err := tx.Exec(
		ctx,
		"SET LOCAL ROLE vec_contratacion_temporal_propietario",
	); err != nil {
		t.Fatal(err)
	}
	var fila filaReservaTerminalPreparacionDecisionCobertura
	if err := tx.QueryRow(ctx, `
		SELECT b.reserva_ref,b.recibo_ref,b.actuacion_ref,b.auditoria_ref,
		       b.evento_ref,b.correlacion_vec_ref,b.decision_vec_ref,
		       b.version_expediente,
		       a.secuencia,v.revision_cercado,v.token_propietario_sha256,
		       v.observada_en,v.propiedad_hasta
		  FROM vec_contratacion_temporal
		         .reserva_operacion_decision_cobertura b
		  JOIN vec_contratacion_temporal
		         .reserva_operacion_decision_cobertura_actual a
		    USING (ambito_raiz_hmac)
		  JOIN vec_contratacion_temporal
		         .reserva_operacion_decision_cobertura_version v
		    USING (ambito_raiz_hmac,secuencia)
		 WHERE b.ambito_raiz_hmac=$1
		 FOR UPDATE OF b,a,v`,
		par.ambitoHMAC(),
	).Scan(
		&fila.reservaRef,
		&fila.reciboRef,
		&fila.actuacionRef,
		&fila.auditoriaRef,
		&fila.eventoRef,
		&fila.correlacionRef,
		&fila.decisionVECRef,
		&fila.versionExpediente,
		&fila.secuencia,
		&fila.revision,
		&fila.tokenSHA256,
		&fila.observadaEn,
		&fila.propiedadHasta,
	); err != nil {
		t.Fatal(err)
	}
	huellaOrden := huellaPreparacionDecisionCobertura(
		par.ambitoHMAC() + ":orden:" + rama,
	)
	huellaVEC := huellaPreparacionDecisionCobertura(
		par.ambitoHMAC() + ":vec:" + rama,
	)
	huellaRecibo := huellaPreparacionDecisionCobertura(
		par.ambitoHMAC() + ":recibo:" + rama,
	)
	confirmadaEn := fila.observadaEn.Add(time.Microsecond)
	estado := "aplicada"
	codigo := "concedida"
	if rama == "denegada" {
		estado = "denegada_vec"
		codigo = "accion_no_concedida"
	} else if rama != "concedida" {
		t.Fatalf("rama terminal de prueba inválida: %s", rama)
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO
		  vec_contratacion_temporal
		    .reserva_operacion_decision_cobertura_version (
		      ambito_raiz_hmac,secuencia,estado,revision_cercado,
		      token_propietario_sha256,observada_en,propiedad_hasta,
		      huella_orden_sha256,confirmada_en
		    ) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)`,
		par.ambitoHMAC(),
		fila.secuencia+1,
		estado,
		fila.revision,
		fila.tokenSHA256,
		fila.observadaEn,
		fila.propiedadHasta,
		huellaOrden,
		confirmadaEn,
	); err != nil {
		t.Fatal(err)
	}
	huellaDecision := huellaPreparacionDecisionCobertura(
		par.ambitoHMAC() + ":decision:cobertura",
	)
	var decisionRef any
	var decisionHuella any
	var versionResultante any
	var eventoRef any
	var actuacionRef any
	if rama == "concedida" {
		decisionRef = "decision-cobertura:sha256:" + huellaDecision
		decisionHuella = huellaDecision
		versionResultante = int64(fila.versionExpediente + 1)
		eventoRef = fila.eventoRef
		actuacionRef = fila.actuacionRef
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO
		  vec_contratacion_temporal
		    .confirmacion_operacion_decision_cobertura (
		      ambito_raiz_hmac,recibo_ref,reserva_ref,huella_orden_sha256,
		      rama,auditoria_ref,correlacion_vec_ref,decision_vec_ref,
		      decision_vec_huella_sha256,codigo_probatorio_vec,
		      revision_cercado,ambito_idempotencia_hmac,
		      huella_semantica_hmac,decision_cobertura_ref,
		      decision_cobertura_huella_sha256,version_resultante,
		      evento_ref,actuacion_ref,confirmada_en,recibo_huella_sha256
		    ) VALUES (
		      $1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,
		      $14,$15,$16,$17,$18,$19,$20
		    )`,
		par.ambitoHMAC(),
		fila.reciboRef,
		fila.reservaRef,
		huellaOrden,
		rama,
		fila.auditoriaRef,
		fila.correlacionRef,
		fila.decisionVECRef,
		huellaVEC,
		codigo,
		fila.revision,
		par.ambitoHMAC(),
		par.semanticaHMAC(),
		decisionRef,
		decisionHuella,
		versionResultante,
		eventoRef,
		actuacionRef,
		confirmadaEn,
		huellaRecibo,
	); err != nil {
		t.Fatal(err)
	}
	var gobiernoRef any
	var gobiernoHuella any
	var loteRef any
	var loteHuella any
	if rama == "concedida" {
		gobiernoRef = "gobierno_o404c_01"
		gobiernoHuella = huellaPreparacionDecisionCobertura(
			par.ambitoHMAC() + ":gobierno",
		)
		loteRef = "lote_c1_o404c_01"
		loteHuella = huellaPreparacionDecisionCobertura(
			par.ambitoHMAC() + ":lote",
		)
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO
		  vec_contratacion_temporal
		    .terminal_operacion_decision_cobertura (
		      ambito_raiz_hmac,secuencia_terminal,recibo_ref,
		      huella_orden_sha256,rama,decision_vec_ref,auditoria_ref,
		      outbox_ref,gobierno_ref,gobierno_huella_sha256,
		      consumo_c1_lote_ref,consumo_c1_lote_huella_sha256,
		      decision_cobertura_ref,actuacion_ref,version_resultante,
		      marcada_en
		    ) VALUES (
		      $1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16
		    )`,
		par.ambitoHMAC(),
		fila.secuencia+1,
		fila.reciboRef,
		huellaOrden,
		rama,
		fila.decisionVECRef,
		fila.auditoriaRef,
		fila.eventoRef,
		gobiernoRef,
		gobiernoHuella,
		loteRef,
		loteHuella,
		decisionRef,
		actuacionRef,
		versionResultante,
		confirmadaEn,
	); err != nil {
		t.Fatal(err)
	}
	resultado, err := tx.Exec(ctx, `
		UPDATE vec_contratacion_temporal
		         .reserva_operacion_decision_cobertura_actual
		   SET secuencia=$2
		 WHERE ambito_raiz_hmac=$1 AND secuencia=$3`,
		par.ambitoHMAC(),
		fila.secuencia+1,
		fila.secuencia,
	)
	if err != nil || resultado.RowsAffected() != 1 {
		t.Fatalf("puntero terminal no cercado: %d / %v", resultado.RowsAffected(), err)
	}
	if err := tx.Commit(ctx); err != nil {
		t.Fatal(err)
	}
}

func huellaPreparacionDecisionCobertura(valor string) string {
	suma := sha256.Sum256([]byte(valor))
	return hex.EncodeToString(suma[:])
}

func probarCorrupcionPreparacionDecisionCobertura(
	t *testing.T,
	ctx context.Context,
	admin *pgxpool.Pool,
	preparador *PreparadorOperacionDecisionCoberturaDurablePostgreSQL,
	fixture fixturePreparacionDecisionCoberturaDurable,
) {
	t.Helper()
	identidad := fixture.identidad(
		t,
		"018f3b2a-7c4d-4e5f-8a9b-0c1d2e3f4a56",
		fixture.expediente.OrganizacionRef,
	)
	par := parHMACDecisionCoberturaPrueba(1, "b", "c")
	consulta, solicitud := solicitudPreparacionDecisionCoberturaPrueba(
		t,
		identidad,
		par,
	)
	if _, err :=
		preparador.ReservarOReapropiarOperacionDecisionCobertura(
			ctx,
			solicitud,
		); err != nil {
		t.Fatal(err)
	}
	confirmarTerminalPreparacionDecisionCobertura(
		t,
		ctx,
		admin,
		par,
		"denegada",
	)
	corrupcion, err := admin.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		t.Fatal(err)
	}
	defer revertirTransaccion(corrupcion)
	if _, err := corrupcion.Exec(ctx, `
		ALTER TABLE vec_contratacion_temporal
		  .terminal_operacion_decision_cobertura
		  DISABLE TRIGGER bloquear_mutacion`); err != nil {
		t.Fatal(err)
	}
	if _, err := corrupcion.Exec(ctx, `
		UPDATE vec_contratacion_temporal
		         .terminal_operacion_decision_cobertura
		   SET outbox_ref='evento:o404c:corrupto'
		 WHERE ambito_raiz_hmac=$1`,
		par.ambitoHMAC(),
	); err != nil {
		t.Fatal(err)
	}
	if _, err := corrupcion.Exec(ctx, `
		ALTER TABLE vec_contratacion_temporal
		  .terminal_operacion_decision_cobertura
		  ENABLE TRIGGER bloquear_mutacion`,
	); err != nil {
		t.Fatal(err)
	}
	if err := corrupcion.Commit(ctx); err != nil {
		t.Fatal(err)
	}
	_, existe, err :=
		preparador.ConsultarOperacionDecisionCoberturaConfirmada(
			ctx,
			consulta,
		)
	if err != nil || existe {
		t.Fatalf(
			"corrupción se presentó como terminal: existe=%t err=%v",
			existe,
			err,
		)
	}
	_, err = preparador.ReservarOReapropiarOperacionDecisionCobertura(
		ctx,
		solicitud,
	)
	if !errors.Is(
		err,
		errPersistenciaDecisionCoberturaDurableNoDisponible,
	) || errors.Is(err, cobertura.ErrClaveOperacionDecisionCoberturaUsada) {
		t.Fatalf("corrupción no falló cerrada: %v", err)
	}
}

func probarACLRLSPreparacionDecisionCobertura(
	t *testing.T,
	ctx context.Context,
	admin *pgxpool.Pool,
) {
	t.Helper()
	var tablasSeguras, aclTablas, aclFunciones, funcionesSeguras bool
	if err := admin.QueryRow(ctx, `
		WITH tablas(nombre) AS (VALUES
		  ('control_migracion_cobertura_o4'),
		  ('reserva_operacion_decision_cobertura'),
		  ('alias_operacion_decision_cobertura'),
		  ('reserva_operacion_decision_cobertura_version'),
		  ('reserva_operacion_decision_cobertura_actual'),
		  ('confirmacion_operacion_decision_cobertura'),
		  ('terminal_operacion_decision_cobertura')
		)
		SELECT (
		         SELECT pg_catalog.bool_and(
		                  c.relrowsecurity AND c.relforcerowsecurity
		                )
		           FROM tablas t
		           JOIN pg_catalog.pg_class c ON c.relname=t.nombre
		           JOIN pg_catalog.pg_namespace n ON n.oid=c.relnamespace
		          WHERE n.nspname='vec_contratacion_temporal'
		       ),
		       (
		         SELECT pg_catalog.bool_and(
		           NOT pg_catalog.has_table_privilege(
		             r.rol,t.nombre_cualificado,'SELECT,INSERT,UPDATE,DELETE,TRUNCATE'
		           )
		         )
		           FROM (VALUES
		             ('public'),
		             ('vec_contratacion_temporal_ejecutor'),
		             ('vec_contratacion_temporal_migrador')
		           ) r(rol)
		           CROSS JOIN LATERAL (
		             SELECT 'vec_contratacion_temporal.'||nombre
		                      AS nombre_cualificado
		               FROM tablas
		           ) t
		       ),
		       pg_catalog.has_function_privilege(
		         'vec_contratacion_temporal_ejecutor',
		         'vec_contratacion_temporal.preparar_operacion_decision_cobertura_v1(jsonb,jsonb)',
		         'EXECUTE'
		       )
		       AND pg_catalog.has_function_privilege(
		         'vec_contratacion_temporal_ejecutor',
		         'vec_contratacion_temporal.consultar_operacion_decision_cobertura_confirmada_v1(jsonb)',
		         'EXECUTE'
		       )
		       AND NOT pg_catalog.has_function_privilege(
		         'vec_contratacion_temporal_ejecutor',
		         'vec_contratacion_temporal.o404c_carga_terminal_v1(text)',
		         'EXECUTE'
		       )
		       AND NOT pg_catalog.has_function_privilege(
		         'vec_contratacion_temporal_ejecutor',
		         'vec_contratacion_temporal.leer_terminal_primario_decision_cobertura_o404c_v1(jsonb)',
		         'EXECUTE'
		       )
		       AND NOT pg_catalog.has_function_privilege(
		         'public',
		         'vec_contratacion_temporal.preparar_operacion_decision_cobertura_v1(jsonb,jsonb)',
		         'EXECUTE'
		       )
		       AND NOT pg_catalog.has_function_privilege(
		         'public',
		         'vec_contratacion_temporal.consultar_operacion_decision_cobertura_confirmada_v1(jsonb)',
		         'EXECUTE'
		       ),
		       (
		         SELECT pg_catalog.bool_and(
		           p.proowner =
		             'vec_contratacion_temporal_propietario'::regrole
		           AND p.prosecdef
		           AND p.proconfig @> ARRAY[
		             'search_path=pg_catalog',
		             'row_security=on',
		             'TimeZone=UTC',
		             'lock_timeout=2s'
		           ]::text[]
		         )
		           FROM pg_catalog.pg_proc p
		           JOIN pg_catalog.pg_namespace n ON n.oid=p.pronamespace
		          WHERE n.nspname='vec_contratacion_temporal'
		            AND p.proname IN (
		              'preparar_operacion_decision_cobertura_v1',
		              'consultar_operacion_decision_cobertura_confirmada_v1',
		              'leer_terminal_primario_decision_cobertura_o404c_v1'
		            )
		       )`).Scan(
		&tablasSeguras,
		&aclTablas,
		&aclFunciones,
		&funcionesSeguras,
	); err != nil || !tablasSeguras || !aclTablas ||
		!aclFunciones || !funcionesSeguras {
		t.Fatalf(
			"ACL/RLS/FORCE O4-04C incompletas: tablas=%t acl_tablas=%t acl_funciones=%t funciones=%t err=%v",
			tablasSeguras,
			aclTablas,
			aclFunciones,
			funcionesSeguras,
			err,
		)
	}
}

func probarBarreraPreparacionDecisionCobertura(
	t *testing.T,
	ctx context.Context,
	admin *pgxpool.Pool,
	preparador *PreparadorOperacionDecisionCoberturaDurablePostgreSQL,
	consulta cobertura.SolicitudConsultarOperacionDecisionCoberturaConfirmada,
) {
	t.Helper()
	if _, err := admin.Exec(ctx, `
		BEGIN;
		SET LOCAL ROLE vec_contratacion_temporal_propietario;
		UPDATE vec_contratacion_temporal.control_migracion_cobertura_o4
		   SET version_esquema=2,
		       actualizada_en=pg_catalog.date_trunc(
		         'microseconds',pg_catalog.clock_timestamp()
		       )
		 WHERE control AND version_esquema=3;
		COMMIT`); err != nil {
		t.Fatal(err)
	}
	_, _, err :=
		preparador.ConsultarOperacionDecisionCoberturaConfirmada(
			ctx,
			consulta,
		)
	if !errors.Is(
		err,
		errPersistenciaDecisionCoberturaDurableNoDisponible,
	) {
		t.Fatalf("lector atravesó barrera degradada: %v", err)
	}
	if _, err := admin.Exec(ctx, `
		BEGIN;
		SET LOCAL ROLE vec_contratacion_temporal_propietario;
		UPDATE vec_contratacion_temporal.control_migracion_cobertura_o4
		   SET version_esquema=3,
		       actualizada_en=pg_catalog.date_trunc(
		         'microseconds',pg_catalog.clock_timestamp()
		       )
		 WHERE control AND version_esquema=2;
		COMMIT`); err != nil {
		t.Fatal(err)
	}
	_, existe, err :=
		preparador.ConsultarOperacionDecisionCoberturaConfirmada(
			ctx,
			consulta,
		)
	if err != nil || !existe {
		t.Fatalf("lector no volvió tras restaurar barrera: %t / %v", existe, err)
	}
}
