//go:build integracion_postgresql_o405

package cobertura_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/postgres"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/cobertura"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

const (
	rutaFixturePostgreSQLO405 = "/run/vec-o405/fixture.json"
	maximoBytesFixtureO405    = 8 * 1024
)

type fixturePostgreSQLO405 struct {
	OrganizacionRef string   `json:"organizacion_ref"`
	ExpedienteRef   string   `json:"expediente_ref"`
	AmbitosHMAC     []string `json:"ambitos_idempotencia_hmac"`
}

func TestIntegracionPostgreSQLO405LectorNominativo(
	t *testing.T,
) {
	exigirIntegracionPostgreSQLO405(t)
	fixture := cargarFixturePostgreSQLO405(t)
	solicitud := solicitudFixturePostgreSQLO405(
		t,
		fixture.OrganizacionRef,
		fixture.ExpedienteRef,
		fixture.AmbitosHMAC,
	)
	pool := nuevoPoolPostgreSQLO405(t)
	ctxProduccion, cancelarProduccion := context.WithTimeout(
		context.Background(),
		10*time.Second,
	)
	defer cancelarProduccion()
	if ejecutor, err :=
		postgres.NuevoEjecutorLecturaResultadoHistoricoOperacionDecisionCoberturaPostgreSQL(
			ctxProduccion,
			pool,
		); ejecutor != nil || !errors.Is(
		err,
		cobertura.ErrLecturaResultadoHistoricoOperacionDecisionCoberturaNoDisponible,
	) {
		t.Fatalf(
			"constructor de producción aceptó socket sin TLS: ejecutor=%v err=%v",
			ejecutor,
			err,
		)
	}
	lector := nuevoLectorPostgreSQLO405(t, pool)

	ctx, cancelar := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancelar()
	resultado, err :=
		lector.LeerResultadoHistoricoOperacionDecisionCobertura(ctx, solicitud)
	if err != nil {
		t.Fatalf("lectura confirmada O4-05: %v", err)
	}
	if _, confirmado := resultado.ReciboConfirmadoPara(solicitud); !confirmado {
		t.Fatal("el lector TCB no construyó la rama confirmado")
	}

	solicitudAusente := solicitudFixturePostgreSQLO405(
		t,
		fixture.OrganizacionRef,
		"expediente:o405:ausente",
		fixture.AmbitosHMAC,
	)
	resultado, err =
		lector.LeerResultadoHistoricoOperacionDecisionCobertura(
			ctx,
			solicitudAusente,
		)
	if err != nil {
		t.Fatalf("lectura no observable O4-05: %v", err)
	}
	if !resultado.NoObservablePara(solicitudAusente) {
		t.Fatal("el lector TCB no construyó la rama no_observable")
	}
}

func TestIntegracionPostgreSQLO405LoginSinGrantO405FallaCerrado(
	t *testing.T,
) {
	exigirIntegracionPostgreSQLO405(t)
	pool := nuevoPoolPostgreSQLO405(t)

	ctx, cancelar := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancelar()
	ejecutor, err :=
		postgres.NuevoEjecutorLecturaResultadoHistoricoOperacionDecisionCoberturaPostgreSQLParaIntegracionSocketUnix(
			ctx,
			pool,
		)
	if !errors.Is(
		err,
		cobertura.ErrLecturaResultadoHistoricoOperacionDecisionCoberturaNoDisponible,
	) {
		t.Fatalf("pool sin grant O4-05 no falló cerrado: %v", err)
	}
	if ejecutor != nil {
		t.Fatal("pool sin grant O4-05 obtuvo ejecutor")
	}
}

func TestIntegracionPostgreSQLO405ReemplazoMismoOIDNoPublicaYRestaura(
	t *testing.T,
) {
	exigirIntegracionPostgreSQLO405(t)
	fixture := cargarFixturePostgreSQLO405(t)
	solicitud := solicitudFixturePostgreSQLO405(
		t,
		fixture.OrganizacionRef,
		fixture.ExpedienteRef,
		fixture.AmbitosHMAC,
	)
	pool := nuevoPoolPostgreSQLO405(t)
	lector := nuevoLectorPostgreSQLO405(t, pool)
	ctx, cancelar := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancelar()
	admin, err := pgx.Connect(
		ctx,
		"host=/var/run/postgresql dbname=postgres "+
			"user=postgres sslmode=disable",
	)
	if err != nil {
		t.Fatal("administrador O4-05 no disponible")
	}
	defer admin.Close(context.Background())

	const firma = "" +
		"vec_contratacion_temporal." +
		"recuperar_resultado_propio_decision_cobertura_o405_v1(jsonb)"
	var oidOriginal uint32
	var definicionOriginal string
	var cuerpoOriginal string
	if err := admin.QueryRow(
		ctx,
		`SELECT p.oid::oid,
		        pg_catalog.pg_get_functiondef(p.oid),
		        p.prosrc
		   FROM pg_catalog.pg_proc AS p
		  WHERE p.oid=pg_catalog.to_regprocedure($1)`,
		firma,
	).Scan(
		&oidOriginal,
		&definicionOriginal,
		&cuerpoOriginal,
	); err != nil {
		t.Fatal("definición O4-05 original no disponible")
	}
	restaurada := false
	defer func() {
		if !restaurada {
			_, _ = admin.Exec(context.Background(), definicionOriginal)
		}
	}()
	_, err = admin.Exec(ctx, `
		CREATE OR REPLACE FUNCTION vec_contratacion_temporal
		  .recuperar_resultado_propio_decision_cobertura_o405_v1(
		    p_consulta jsonb
		)
		RETURNS TABLE(resultado_json jsonb)
		LANGUAGE plpgsql
		STABLE
		SECURITY DEFINER
		SET search_path=pg_catalog
		SET row_security='on'
		SET timezone='UTC'
		SET lock_timeout='2s'
		AS $hostil$
		BEGIN
		  RETURN QUERY SELECT pg_catalog.jsonb_build_object(
		    'esquema',
		    'vec.contratacion-temporal.resultado-recuperacion-propia-' ||
		      'decision-cobertura.o4-05.v1',
		    'estado','no_observable',
		    'observada_en','2026-07-26T10:02:00Z'
		  );
		END
		$hostil$`)
	if err != nil {
		t.Fatal("reemplazo hostil O4-05 no creado")
	}
	var oidReemplazo uint32
	var metadatosIguales bool
	var cuerpoReemplazo string
	if err := admin.QueryRow(
		ctx,
		`SELECT p.oid::oid,
		        n.nspname='vec_contratacion_temporal'
		        AND p.proname=
		          'recuperar_resultado_propio_decision_cobertura_o405_v1'
		        AND p.prokind='f'
		        AND r.rolname='vec_contratacion_temporal_propietario'
		        AND p.prosecdef AND p.provolatile='s'
		        AND l.lanname='plpgsql' AND p.probin IS NULL
		        AND pg_catalog.pg_get_function_identity_arguments(p.oid)=
		          'p_consulta jsonb'
		        AND pg_catalog.pg_get_function_result(p.oid)=
		          'TABLE(resultado_json jsonb)'
		        AND p.proconfig=ARRAY[
		          'search_path=pg_catalog','row_security=on',
		          'TimeZone=UTC','lock_timeout=2s']::text[],
		        p.prosrc
		   FROM pg_catalog.pg_proc AS p
		   JOIN pg_catalog.pg_namespace AS n ON n.oid=p.pronamespace
		   JOIN pg_catalog.pg_roles AS r ON r.oid=p.proowner
		   JOIN pg_catalog.pg_language AS l ON l.oid=p.prolang
		  WHERE p.oid=pg_catalog.to_regprocedure($1)`,
		firma,
	).Scan(
		&oidReemplazo,
		&metadatosIguales,
		&cuerpoReemplazo,
	); err != nil || oidReemplazo != oidOriginal ||
		!metadatosIguales || cuerpoReemplazo == cuerpoOriginal {
		t.Fatalf(
			"adversario no conservó OID/metadatos: oid=%d/%d meta=%t",
			oidReemplazo,
			oidOriginal,
			metadatosIguales,
		)
	}
	resultado, err :=
		lector.LeerResultadoHistoricoOperacionDecisionCobertura(ctx, solicitud)
	if !errors.Is(
		err,
		cobertura.ErrLecturaResultadoHistoricoOperacionDecisionCoberturaNoConfiable,
	) || resultado.NoObservablePara(solicitud) {
		t.Fatalf("reemplazo publicó datos: resultado=%v err=%v", resultado, err)
	}

	if _, err := admin.Exec(ctx, definicionOriginal); err != nil {
		t.Fatal("definición O4-05 no restaurada")
	}
	restaurada = true
	var cuerpoRestaurado string
	if err := admin.QueryRow(
		ctx,
		`SELECT p.prosrc
		   FROM pg_catalog.pg_proc AS p
		  WHERE p.oid=pg_catalog.to_regprocedure($1)`,
		firma,
	).Scan(&cuerpoRestaurado); err != nil ||
		cuerpoRestaurado != cuerpoOriginal {
		t.Fatal("restauración O4-05 no fue exacta")
	}
	aclRevocada := false
	defer func() {
		if aclRevocada {
			_, _ = admin.Exec(
				context.Background(),
				`GRANT EXECUTE ON FUNCTION
				   vec_contratacion_temporal
				    .recuperar_resultado_propio_decision_cobertura_o405_v1(jsonb)
				 TO vec_contratacion_temporal_lector_resultado_cobertura`,
			)
		}
	}()
	if _, err := admin.Exec(
		ctx,
		`REVOKE EXECUTE ON FUNCTION
		   vec_contratacion_temporal
		    .recuperar_resultado_propio_decision_cobertura_o405_v1(jsonb)
		 FROM vec_contratacion_temporal_lector_resultado_cobertura`,
	); err != nil {
		t.Fatal("deriva ACL O4-05 no creada")
	}
	aclRevocada = true
	resultado, err =
		lector.LeerResultadoHistoricoOperacionDecisionCobertura(ctx, solicitud)
	if !errors.Is(
		err,
		cobertura.ErrLecturaResultadoHistoricoOperacionDecisionCoberturaNoConfiable,
	) || resultado.NoObservablePara(solicitud) {
		t.Fatalf("deriva ACL publicó datos: resultado=%v err=%v", resultado, err)
	}
	if _, err := admin.Exec(
		ctx,
		`GRANT EXECUTE ON FUNCTION
		   vec_contratacion_temporal
		    .recuperar_resultado_propio_decision_cobertura_o405_v1(jsonb)
		 TO vec_contratacion_temporal_lector_resultado_cobertura`,
	); err != nil {
		t.Fatal("ACL O4-05 no restaurada")
	}
	aclRevocada = false
	membresiaRevocada := false
	defer func() {
		if membresiaRevocada {
			_, _ = admin.Exec(
				context.Background(),
				`GRANT vec_contratacion_temporal_lector_resultado_cobertura
				   TO vec_o405_lector
				   WITH ADMIN FALSE,INHERIT TRUE,SET FALSE`,
			)
		}
	}()
	if _, err := admin.Exec(
		ctx,
		`REVOKE vec_contratacion_temporal_lector_resultado_cobertura
		   FROM vec_o405_lector`,
	); err != nil {
		t.Fatal("deriva de membresía O4-05 no creada")
	}
	membresiaRevocada = true
	resultado, err =
		lector.LeerResultadoHistoricoOperacionDecisionCobertura(ctx, solicitud)
	if !errors.Is(
		err,
		cobertura.ErrLecturaResultadoHistoricoOperacionDecisionCoberturaNoConfiable,
	) || resultado.NoObservablePara(solicitud) {
		t.Fatalf(
			"deriva de rol publicó datos: resultado=%v err=%v",
			resultado,
			err,
		)
	}
	if _, err := admin.Exec(
		ctx,
		`GRANT vec_contratacion_temporal_lector_resultado_cobertura
		   TO vec_o405_lector
		   WITH ADMIN FALSE,INHERIT TRUE,SET FALSE`,
	); err != nil {
		t.Fatal("membresía O4-05 no restaurada")
	}
	membresiaRevocada = false
	resultado, err =
		lector.LeerResultadoHistoricoOperacionDecisionCobertura(ctx, solicitud)
	if err != nil {
		t.Fatalf("recuperación restaurada O4-05: %v", err)
	}
	if _, confirmada := resultado.ReciboConfirmadoPara(solicitud); !confirmada {
		t.Fatal("restauración O4-05 no recuperó recibo confirmado")
	}
}

func exigirIntegracionPostgreSQLO405(t *testing.T) {
	t.Helper()
	if os.Getenv("VEC_O405_INTEGRACION_POSTGRES") != "1" {
		t.Skip("integración PostgreSQL O4-05 no habilitada")
	}
}

func cargarFixturePostgreSQLO405(t *testing.T) fixturePostgreSQLO405 {
	t.Helper()
	archivo, err := os.Open(rutaFixturePostgreSQLO405)
	if err != nil {
		t.Fatal("fixture PostgreSQL O4-05 no disponible")
	}
	defer archivo.Close()
	contenido, err := io.ReadAll(io.LimitReader(
		archivo,
		maximoBytesFixtureO405+1,
	))
	if err != nil || len(contenido) == 0 ||
		len(contenido) > maximoBytesFixtureO405 {
		t.Fatal("fixture PostgreSQL O4-05 fuera de límite")
	}
	var fixture fixturePostgreSQLO405
	decodificador := json.NewDecoder(bytes.NewReader(contenido))
	decodificador.DisallowUnknownFields()
	if err := decodificador.Decode(&fixture); err != nil {
		t.Fatal("fixture PostgreSQL O4-05 inválido")
	}
	if err := decodificador.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		t.Fatal("fixture PostgreSQL O4-05 contiene datos adicionales")
	}
	return fixture
}

func solicitudFixturePostgreSQLO405(
	t *testing.T,
	organizacionRef string,
	expedienteRef string,
	ambitos []string,
) cobertura.SolicitudRecuperacionResultadoOperacionDecisionCobertura {
	t.Helper()
	if len(ambitos) == 0 {
		t.Fatal("fixture PostgreSQL O4-05 sin ámbitos")
	}
	coleccion, err := ports.NuevaColeccionSellosHMAC(
		ambitos[0],
		ambitos[1:],
	)
	if err != nil {
		t.Fatal("fixture PostgreSQL O4-05 con ámbitos inválidos")
	}
	solicitud, err :=
		cobertura.NuevaSolicitudRecuperacionResultadoOperacionDecisionCoberturaIntegracionPrueba(
			organizacionRef,
			expedienteRef,
			coleccion,
		)
	if err != nil {
		t.Fatal("fixture PostgreSQL O4-05 no forma una solicitud nominal")
	}
	return solicitud
}

func nuevoPoolPostgreSQLO405(
	t *testing.T,
) *postgres.PoolRecuperacionCoberturaO405PostgreSQL {
	t.Helper()
	ctx, cancelar := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancelar()
	pool, err :=
		postgres.NuevoPoolRecuperacionCoberturaO405PostgreSQLParaIntegracionSocketUnix(
			ctx,
			"",
		)
	if err != nil {
		t.Fatal("configuración PostgreSQL O4-05 inválida")
	}
	t.Cleanup(pool.Cerrar)
	return pool
}

func nuevoLectorPostgreSQLO405(
	t *testing.T,
	pool *postgres.PoolRecuperacionCoberturaO405PostgreSQL,
) cobertura.LectorResultadoHistoricoOperacionDecisionCobertura {
	t.Helper()
	ctx, cancelar := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancelar()
	ejecutor, err :=
		postgres.NuevoEjecutorLecturaResultadoHistoricoOperacionDecisionCoberturaPostgreSQLParaIntegracionSocketUnix(
			ctx,
			pool,
		)
	if err != nil {
		t.Fatal(err)
	}
	lector, err :=
		cobertura.NuevoLectorResultadoHistoricoOperacionDecisionCoberturaTCB(
			ejecutor,
		)
	if err != nil {
		t.Fatal(err)
	}
	return lector
}
