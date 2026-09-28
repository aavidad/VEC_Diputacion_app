package bootstrap

// Este test se incorpora solo con go test -overlay al paquete bootstrap.
// El fichero real permanece bajo scripts/rrhh_plantillas y no cambia producto.

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	plantillasapp "vec-diputacion-granada/internal/modules/contrataciontemporal/application/plantillascatalogo"
)

func TestPlantillasPG18PreflightAislado(t *testing.T) {
	dsn := os.Getenv("VEC_PLANTILLAS_PG18_DSN")
	stage := os.Getenv("VEC_PLANTILLAS_PG18_STAGE")
	if dsn == "" || stage == "" {
		t.Skip("solo se ejecuta desde scripts/rrhh_plantillas/probar_pg18.sh")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	err = preflightCatalogoPlantillasGobiernoCT(ctx, pool)
	if stage == "rechazo" {
		if !errors.Is(err, plantillasapp.ErrNoDisponible) {
			t.Fatalf("preflight debía fallar cerrado: %v", err)
		}
		return
	}
	if stage != "aceptado" {
		t.Fatalf("etapa desconocida: %q", stage)
	}
	if err != nil {
		for _, probe := range []struct{ name, sql string }{
			{"ad3_oid", `SELECT to_regprocedure('vec_autorizacion_atestada_v3.registrar_y_consumir_catalogo_plantillas_ct_org_v3_atestada(jsonb,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)')::text`},
			{"ad3_usage_login", `SELECT has_schema_privilege(session_user,'vec_autorizacion_atestada_v3','USAGE')::text`},
			{"ad3_usage_owner", `SELECT has_schema_privilege('vec_contratacion_temporal_propietario','vec_autorizacion_atestada_v3','USAGE')::text`},
			{"ad3_exec_owner", `SELECT has_function_privilege('vec_contratacion_temporal_propietario',to_regprocedure('vec_autorizacion_atestada_v3.registrar_y_consumir_catalogo_plantillas_ct_org_v3_atestada(jsonb,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'),'EXECUTE')::text`},
			{"ct_func", `SELECT has_function_privilege(session_user,to_regprocedure('vec_contratacion_temporal.operar_catalogo_plantillas_v1(jsonb,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'),'EXECUTE')::text`},
			{"rol", `SELECT (pg_has_role(session_user,'vec_contratacion_temporal_ejecutor','USAGE') AND current_user=session_user)::text`},
		} {
			var value string
			e := pool.QueryRow(ctx, probe.sql).Scan(&value)
			t.Logf("%s=%q error=%v", probe.name, value, e)
		}
		t.Fatalf("preflight con CT135/AD3-99 instaladas: %v", err)
	}
	catalogo, err := CargarCatalogoPlantillasCT(os.Getenv("VEC_PLANTILLAS_PG18_CATALOGO"))
	if err != nil {
		t.Fatal(err)
	}
	if err := comprobarPreimagenCatalogoPlantillasCT(ctx, pool, catalogo); err != nil {
		t.Fatalf("preimagen provisionada: %v", err)
	}
}

func TestPlantillasPG18AutoridadesAisladas(t *testing.T) {
	fuenteDSN := os.Getenv("VEC_PLANTILLAS_PG18_FUENTE_DSN")
	motivosDSN := os.Getenv("VEC_PLANTILLAS_PG18_MOTIVOS_DSN")
	stage := os.Getenv("VEC_PLANTILLAS_PG18_AUTORIDADES_STAGE")
	if fuenteDSN == "" || motivosDSN == "" || stage == "" {
		t.Skip("solo se ejecuta desde scripts/rrhh_plantillas/probar_pg18.sh")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	fuente, err := pgxpool.New(ctx, fuenteDSN)
	if err != nil {
		t.Fatal(err)
	}
	defer fuente.Close()
	motivos, err := pgxpool.New(ctx, motivosDSN)
	if err != nil {
		t.Fatal(err)
	}
	defer motivos.Close()
	err = comprobarPreflightAutoridadesPlantillasCT(ctx, fuente, motivos)
	if stage == "rechazo" {
		if !errors.Is(err, plantillasapp.ErrNoDisponible) {
			t.Fatalf("autoridades debían fallar cerrado: %v", err)
		}
		return
	}
	if stage != "aceptado" || err != nil {
		t.Fatalf("preflight de fuente y motivos: etapa=%q error=%v", stage, err)
	}
}
