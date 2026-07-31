package postgres

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
)

func TestRetiradaAcreditacionUsoV2RechazaDependenciasImplicitas(t *testing.T) {
	dsn := os.Getenv("VEC_CONTEXTO_ACTOR_MIGRACION_POSTGRES_DSN")
	if dsn == "" {
		t.Skip("requiere PostgreSQL 18 efímero de contexto actor")
	}
	down := leerDocumentoRetiradaAcreditacionUsoV2(t)
	up := leerDocumentoAltaAcreditacionUsoV2(t)
	asegurarAcreditacionUsoV2InstaladaAlFinal(t, dsn, up)
	limpiarDependenciasImplicitasAlFinal(t, dsn)

	ctx, cancelar := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancelar()
	conexion := abrirConexionRetiradaAcreditacionUsoV2(
		t, ctx, dsn, "de dependencias implícitas",
	)
	defer cerrarConexionRetiradaAcreditacionUsoV2(t, conexion)

	if _, err := conexion.Exec(ctx, `
		ALTER INDEX
		  vec_contexto_actor_v1.control_generacion_punteros_actuales_v2_pkey
		  DEPENDS ON EXTENSION plpgsql
	`); err != nil {
		t.Fatalf("añadir dependencia sintética de extensión: %v", err)
	}
	exigirRechazoConObjetoConservado(
		t, ctx, conexion, down, "dependencia DEPENDS ON EXTENSION",
		`SELECT EXISTS (
		   SELECT 1 FROM pg_catalog.pg_depend AS d
		     JOIN pg_catalog.pg_extension AS e ON e.oid=d.refobjid
		    WHERE d.classid='pg_catalog.pg_class'::regclass
		      AND d.objid='vec_contexto_actor_v1.control_generacion_punteros_actuales_v2_pkey'::regclass
		      AND d.refclassid='pg_catalog.pg_extension'::regclass
		      AND d.deptype='x' AND e.extname='plpgsql'
		 )`,
	)
	if _, err := conexion.Exec(ctx, `
		ALTER INDEX
		  vec_contexto_actor_v1.control_generacion_punteros_actuales_v2_pkey
		  NO DEPENDS ON EXTENSION plpgsql;
		CREATE STATISTICS vec_contexto_actor_v1.ct126_estadistica_control
		  ON generacion, actualizada_en
		  FROM vec_contexto_actor_v1.control_generacion_punteros_actuales_v2
	`); err != nil {
		t.Fatalf("sustituir dependencia por estadística extendida: %v", err)
	}
	exigirRechazoConObjetoConservado(
		t, ctx, conexion, down, "estadística extendida dependiente",
		`SELECT EXISTS (
		   SELECT 1 FROM pg_catalog.pg_statistic_ext
		    WHERE stxnamespace='vec_contexto_actor_v1'::regnamespace
		      AND stxname='ct126_estadistica_control'
		 )`,
	)
	if _, err := conexion.Exec(ctx, `
		DROP STATISTICS vec_contexto_actor_v1.ct126_estadistica_control
	`); err != nil {
		t.Fatalf("limpiar estadística extendida: %v", err)
	}
	configurarOptInRetiradaUsoV2(t, ctx, conexion, optInRetiradaAcreditacionUsoV2)
	if _, err := conexion.Exec(ctx, string(down)); err != nil {
		t.Fatalf("retirar tras limpiar dependencias implícitas: %v", err)
	}
	comprobarBaseTrasRetiradaAcreditacionUsoV2(t, ctx, conexion)
	if _, err := conexion.Exec(ctx, string(up)); err != nil {
		t.Fatalf("restaurar 000002 tras dependencias implícitas: %v", err)
	}
}

func exigirRechazoConObjetoConservado(
	t *testing.T,
	ctx context.Context,
	conexion *pgx.Conn,
	down []byte,
	descripcion string,
	consultaCausa string,
) {
	t.Helper()
	var causaPrevia bool
	if err := conexion.QueryRow(ctx, consultaCausa).Scan(&causaPrevia); err != nil {
		t.Fatalf("comprobar existencia previa de %s: %v", descripcion, err)
	}
	if !causaPrevia {
		t.Fatalf("no se materializó %s antes de la retirada", descripcion)
	}
	configurarOptInRetiradaUsoV2(t, ctx, conexion, optInRetiradaAcreditacionUsoV2)
	if _, err := conexion.Exec(ctx, string(down)); err == nil {
		t.Fatalf("la retirada aceptó %s", descripcion)
	}
	if err := sanearConexionRetiradaAcreditacionUsoV2(conexion); err != nil {
		t.Fatalf("sanear rechazo de %s: %v", descripcion, err)
	}
	var controlConservado, causaConservada bool
	if err := conexion.QueryRow(ctx, `
		SELECT pg_catalog.to_regclass(
		  'vec_contexto_actor_v1.control_generacion_punteros_actuales_v2'
		) IS NOT NULL
	`).Scan(&controlConservado); err != nil {
		t.Fatalf("comprobar rollback de %s: %v", descripcion, err)
	}
	if !controlConservado {
		t.Fatalf("el rechazo de %s dejó una retirada parcial", descripcion)
	}
	if err := conexion.QueryRow(ctx, consultaCausa).Scan(&causaConservada); err != nil {
		t.Fatalf("comprobar conservación de %s: %v", descripcion, err)
	}
	if !causaConservada {
		t.Fatalf("el rechazo no conservó %s", descripcion)
	}
}

func limpiarDependenciasImplicitasAlFinal(t *testing.T, dsn string) {
	t.Helper()
	t.Cleanup(func() {
		ctx, cancelar := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancelar()
		conexion, err := pgx.Connect(ctx, dsn)
		if err != nil {
			t.Errorf("abrir conexión para limpiar dependencias: %v", err)
			return
		}
		defer func() { _ = conexion.Close(ctx) }()
		_, _ = conexion.Exec(ctx, `
			DROP STATISTICS IF EXISTS
			  vec_contexto_actor_v1.ct126_estadistica_control
		`)
		_, _ = conexion.Exec(ctx, `
			ALTER INDEX
			  vec_contexto_actor_v1.control_generacion_punteros_actuales_v2_pkey
			  NO DEPENDS ON EXTENSION plpgsql
		`)
	})
}
