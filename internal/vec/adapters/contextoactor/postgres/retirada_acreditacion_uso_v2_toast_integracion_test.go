package postgres

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
)

const comentarioIndiceToastCT126 = "ct126_metadato_toast_sintetico"

type objetosToastControlV2 struct {
	tablaOID    uint32
	indiceOID   uint32
	tablaSQL    string
	indiceSQL   string
	indiceLocal string
}

func TestRetiradaAcreditacionUsoV2RechazaDependenciaExtensionToast(t *testing.T) {
	dsn := os.Getenv("VEC_CONTEXTO_ACTOR_MIGRACION_POSTGRES_DSN")
	if dsn == "" {
		t.Skip("requiere PostgreSQL 18 efímero de contexto actor")
	}
	down := leerDocumentoRetiradaAcreditacionUsoV2(t)
	up := leerDocumentoAltaAcreditacionUsoV2(t)
	asegurarAcreditacionUsoV2InstaladaAlFinal(t, dsn, up)
	limpiarDerivasToastAlFinal(t, dsn)

	ctx, cancelar := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancelar()
	conexion := abrirConexionRetiradaAcreditacionUsoV2(t, ctx, dsn, "TOAST e/x")
	defer cerrarConexionRetiradaAcreditacionUsoV2(t, conexion)
	objetos := descubrirObjetosToastControlV2(t, ctx, conexion)

	if _, err := conexion.Exec(ctx, fmt.Sprintf(
		"ALTER INDEX %s DEPENDS ON EXTENSION plpgsql", objetos.indiceSQL,
	)); err != nil {
		t.Fatalf("añadir dependencia e/x al índice TOAST: %v", err)
	}
	exigirRechazoConObjetoConservado(
		t, ctx, conexion, down, "dependencia e/x exacta del índice TOAST",
		fmt.Sprintf(`SELECT EXISTS (
			SELECT 1 FROM pg_catalog.pg_depend AS d
			  JOIN pg_catalog.pg_extension AS e ON e.oid=d.refobjid
			 WHERE d.classid='pg_catalog.pg_class'::regclass
			   AND d.objid=%d AND d.objsubid=0
			   AND d.refclassid='pg_catalog.pg_extension'::regclass
			   AND d.deptype='x' AND e.extname='plpgsql'
		)`, objetos.indiceOID),
	)
	if _, err := conexion.Exec(ctx, fmt.Sprintf(
		"ALTER INDEX %s NO DEPENDS ON EXTENSION plpgsql", objetos.indiceSQL,
	)); err != nil {
		t.Fatalf("limpiar dependencia e/x del índice TOAST: %v", err)
	}
	retirarYRestaurarAcreditacionUsoV2Toast(t, ctx, conexion, down, up)
}

func TestRetiradaAcreditacionUsoV2RechazaMetadatosToastCombinados(t *testing.T) {
	dsn := os.Getenv("VEC_CONTEXTO_ACTOR_MIGRACION_POSTGRES_DSN")
	if dsn == "" {
		t.Skip("requiere PostgreSQL 18 efímero de contexto actor")
	}
	down := leerDocumentoRetiradaAcreditacionUsoV2(t)
	up := leerDocumentoAltaAcreditacionUsoV2(t)
	asegurarAcreditacionUsoV2InstaladaAlFinal(t, dsn, up)
	limpiarDerivasToastAlFinal(t, dsn)

	ctx, cancelar := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancelar()
	conexion := abrirConexionRetiradaAcreditacionUsoV2(t, ctx, dsn, "metadatos TOAST")
	defer cerrarConexionRetiradaAcreditacionUsoV2(t, conexion)
	objetos := descubrirObjetosToastControlV2(t, ctx, conexion)

	deriva := fmt.Sprintf(`
		ALTER TABLE vec_contexto_actor_v1.control_generacion_punteros_actuales_v2
		  SET (toast.autovacuum_enabled=false);
		COMMENT ON INDEX %s IS '%s';
		GRANT SELECT ON TABLE %s TO PUBLIC;
		CLUSTER %s USING %s
	`, objetos.indiceSQL, comentarioIndiceToastCT126, objetos.tablaSQL,
		objetos.tablaSQL, objetos.indiceLocal)
	if _, err := conexion.Exec(ctx, deriva); err != nil {
		t.Fatalf("crear metadatos TOAST combinados: %v", err)
	}
	consultaDeriva := fmt.Sprintf(`SELECT
		'autovacuum_enabled=false'=ANY(coalesce(t.reloptions, ARRAY[]::text[]))
		AND EXISTS (
			SELECT 1 FROM pg_catalog.aclexplode(t.relacl) AS a
			 WHERE a.grantee=0 AND a.privilege_type='SELECT'
		)
		AND pg_catalog.obj_description(%d, 'pg_class')='%s'
		AND i.indisclustered
		FROM pg_catalog.pg_class AS t
		  JOIN pg_catalog.pg_index AS i ON i.indrelid=t.oid
		WHERE t.oid=%d AND i.indexrelid=%d`,
		objetos.indiceOID, comentarioIndiceToastCT126,
		objetos.tablaOID, objetos.indiceOID)
	exigirRechazoConObjetoConservado(
		t, ctx, conexion, down, "reloptions, ACL, comentario y CLUSTER TOAST",
		consultaDeriva,
	)
	limpiarDerivasToast(t, ctx, conexion, objetos)
	retirarYRestaurarAcreditacionUsoV2Toast(t, ctx, conexion, down, up)
}

func TestRetiradaAcreditacionUsoV2RechazaFormaIndiceToast(t *testing.T) {
	dsn := os.Getenv("VEC_CONTEXTO_ACTOR_MIGRACION_POSTGRES_DSN")
	if dsn == "" {
		t.Skip("requiere PostgreSQL 18 efímero de contexto actor")
	}
	down := leerDocumentoRetiradaAcreditacionUsoV2(t)
	up := leerDocumentoAltaAcreditacionUsoV2(t)
	asegurarAcreditacionUsoV2InstaladaAlFinal(t, dsn, up)
	limpiarDerivasToastAlFinal(t, dsn)

	ctx, cancelar := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancelar()
	conexion := abrirConexionRetiradaAcreditacionUsoV2(t, ctx, dsn, "forma de índice TOAST")
	defer cerrarConexionRetiradaAcreditacionUsoV2(t, conexion)
	objetos := descubrirObjetosToastControlV2(t, ctx, conexion)

	if _, err := conexion.Exec(ctx, `
		UPDATE pg_catalog.pg_index
		   SET indoption='1 0'::int2vector
		 WHERE indexrelid=$1
	`, objetos.indiceOID); err != nil {
		t.Fatalf("derivar opciones del índice TOAST: %v", err)
	}
	exigirRechazoConObjetoConservado(
		t, ctx, conexion, down, "forma exacta del índice TOAST",
		fmt.Sprintf(`SELECT indoption='1 0'::int2vector
			FROM pg_catalog.pg_index WHERE indexrelid=%d`, objetos.indiceOID),
	)
	limpiarDerivasToast(t, ctx, conexion, objetos)
	retirarYRestaurarAcreditacionUsoV2Toast(t, ctx, conexion, down, up)
}

func descubrirObjetosToastControlV2(
	t *testing.T,
	ctx context.Context,
	conexion *pgx.Conn,
) objetosToastControlV2 {
	t.Helper()
	var objetos objetosToastControlV2
	var esquemaTabla, nombreTabla, esquemaIndice, nombreIndice string
	if err := conexion.QueryRow(ctx, `
		SELECT t.oid, i.oid, nt.nspname, t.relname, ni.nspname, i.relname
		  FROM pg_catalog.pg_class AS c
		  JOIN pg_catalog.pg_class AS t ON t.oid=c.reltoastrelid
		  JOIN pg_catalog.pg_index AS x ON x.indrelid=t.oid AND x.indisprimary
		  JOIN pg_catalog.pg_class AS i ON i.oid=x.indexrelid
		  JOIN pg_catalog.pg_namespace AS nt ON nt.oid=t.relnamespace
		  JOIN pg_catalog.pg_namespace AS ni ON ni.oid=i.relnamespace
		 WHERE c.oid=
		   'vec_contexto_actor_v1.control_generacion_punteros_actuales_v2'::regclass
	`).Scan(
		&objetos.tablaOID, &objetos.indiceOID,
		&esquemaTabla, &nombreTabla, &esquemaIndice, &nombreIndice,
	); err != nil {
		t.Fatalf("descubrir clausura TOAST del control: %v", err)
	}
	objetos.tablaSQL = pgx.Identifier{esquemaTabla, nombreTabla}.Sanitize()
	objetos.indiceSQL = pgx.Identifier{esquemaIndice, nombreIndice}.Sanitize()
	objetos.indiceLocal = pgx.Identifier{nombreIndice}.Sanitize()
	return objetos
}

func limpiarDerivasToast(
	t *testing.T,
	ctx context.Context,
	conexion *pgx.Conn,
	objetos objetosToastControlV2,
) {
	t.Helper()
	limpieza := fmt.Sprintf(`
		ALTER INDEX %s NO DEPENDS ON EXTENSION plpgsql;
		ALTER TABLE vec_contexto_actor_v1.control_generacion_punteros_actuales_v2
		  RESET (toast.autovacuum_enabled);
		COMMENT ON INDEX %s IS NULL;
		REVOKE SELECT ON TABLE %s FROM PUBLIC;
		UPDATE pg_catalog.pg_class SET relacl=NULL WHERE oid=%d;
		UPDATE pg_catalog.pg_index
		   SET indisclustered=false, indoption='0 0'::int2vector
		 WHERE indexrelid=%d
	`, objetos.indiceSQL, objetos.indiceSQL, objetos.tablaSQL,
		objetos.tablaOID, objetos.indiceOID)
	if _, err := conexion.Exec(ctx, limpieza); err != nil {
		t.Fatalf("limpiar derivas TOAST sintéticas: %v", err)
	}
	var limpia bool
	if err := conexion.QueryRow(ctx, `
		SELECT t.relacl IS NULL AND t.reloptions IS NULL
		  AND pg_catalog.obj_description(i.indexrelid, 'pg_class') IS NULL
		  AND NOT i.indisclustered AND i.indoption='0 0'::int2vector
		  AND NOT EXISTS (
		      SELECT 1 FROM pg_catalog.pg_depend AS d
		       WHERE d.classid='pg_catalog.pg_class'::regclass
		         AND d.objid=i.indexrelid
		         AND d.refclassid='pg_catalog.pg_extension'::regclass
		         AND d.deptype IN ('e', 'x')
		  )
		  FROM pg_catalog.pg_class AS t
		    JOIN pg_catalog.pg_index AS i ON i.indrelid=t.oid
		 WHERE t.oid=$1 AND i.indexrelid=$2
	`, objetos.tablaOID, objetos.indiceOID).Scan(&limpia); err != nil {
		t.Fatalf("verificar limpieza TOAST: %v", err)
	}
	if !limpia {
		t.Fatal("la limpieza dejó deriva en la clausura TOAST")
	}
}

func limpiarDerivasToastAlFinal(t *testing.T, dsn string) {
	t.Helper()
	t.Cleanup(func() {
		ctx, cancelar := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancelar()
		conexion, err := pgx.Connect(ctx, dsn)
		if err != nil {
			t.Errorf("abrir conexión para limpiar TOAST: %v", err)
			return
		}
		defer func() { _ = conexion.Close(ctx) }()
		var objetos objetosToastControlV2
		var esquemaTabla, nombreTabla, esquemaIndice, nombreIndice string
		err = conexion.QueryRow(ctx, `
			SELECT t.oid, i.oid, nt.nspname, t.relname, ni.nspname, i.relname
			  FROM pg_catalog.pg_class AS c
			  JOIN pg_catalog.pg_class AS t ON t.oid=c.reltoastrelid
			  JOIN pg_catalog.pg_index AS x ON x.indrelid=t.oid AND x.indisprimary
			  JOIN pg_catalog.pg_class AS i ON i.oid=x.indexrelid
			  JOIN pg_catalog.pg_namespace AS nt ON nt.oid=t.relnamespace
			  JOIN pg_catalog.pg_namespace AS ni ON ni.oid=i.relnamespace
			 WHERE c.oid=pg_catalog.to_regclass(
			   'vec_contexto_actor_v1.control_generacion_punteros_actuales_v2'
			 )
		`).Scan(
			&objetos.tablaOID, &objetos.indiceOID,
			&esquemaTabla, &nombreTabla, &esquemaIndice, &nombreIndice,
		)
		if err != nil {
			return
		}
		objetos.tablaSQL = pgx.Identifier{esquemaTabla, nombreTabla}.Sanitize()
		objetos.indiceSQL = pgx.Identifier{esquemaIndice, nombreIndice}.Sanitize()
		objetos.indiceLocal = pgx.Identifier{nombreIndice}.Sanitize()
		limpiarDerivasToast(t, ctx, conexion, objetos)
	})
}

func retirarYRestaurarAcreditacionUsoV2Toast(
	t *testing.T,
	ctx context.Context,
	conexion *pgx.Conn,
	down []byte,
	up []byte,
) {
	t.Helper()
	configurarOptInRetiradaUsoV2(t, ctx, conexion, optInRetiradaAcreditacionUsoV2)
	if _, err := conexion.Exec(ctx, string(down)); err != nil {
		t.Fatalf("retirar tras limpiar la clausura TOAST: %v", err)
	}
	comprobarBaseTrasRetiradaAcreditacionUsoV2(t, ctx, conexion)
	if _, err := conexion.Exec(ctx, string(up)); err != nil {
		t.Fatalf("restaurar 000002 tras la clausura TOAST: %v", err)
	}
}
