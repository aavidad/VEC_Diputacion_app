package bootstrap

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Exige una base PostgreSQL 18 efímera llamada vec_pref_lote_* con la
// preimagen canónica AD3 y AD3-106 ya instaladas. No utiliza el clon privado
// conservado ni crea roles o migraciones. El runner externo aporta dos LOGIN:
// DBA de esa base desechable y gobierno nominal de vec-server.
func TestPreferenciasLoteGobiernoPostgreSQL18(t *testing.T) {
	if os.Getenv("VEC_PREF_LOTE_PG_DESECHABLE") != "si" {
		t.Skip("requiere PostgreSQL 18 desechable exclusivo")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	admin, err := pgxpool.New(ctx, os.Getenv("VEC_PREF_LOTE_ADMIN_DSN"))
	if err != nil {
		t.Fatal("DBA efímero no disponible")
	}
	defer admin.Close()
	var version int
	var base string
	if err = admin.QueryRow(ctx, `SELECT current_setting('server_version_num')::int,current_database()`).Scan(&version, &base); err != nil || version/10000 != 18 || !strings.HasPrefix(base, "vec_pref_lote_") {
		t.Fatalf("base no desechable PostgreSQL18: %d %q %v", version, base, err)
	}
	gobierno, _, err := abrirPoolPostgreSQLContratacionTemporalDesarrollo(ctx, os.Getenv("VEC_PREF_LOTE_GOBIERNO_DSN"), "vec-pref-lote-pg18", rolGobiernoPostgreSQLContratacionTemporalDesarrollo)
	if err != nil {
		t.Fatal("LOGIN de gobierno nominal no acreditado")
	}
	defer gobierno.Close()
	const contarUsuarios = `SELECT count(*) FROM vec_autorizacion_atestada_v3.clave_capacidad_version WHERE audiencia_consumo LIKE 'vec_usuarios.preferencias.%'`
	var antes int
	if err = admin.QueryRow(ctx, contarUsuarios).Scan(&antes); err != nil || antes != 0 {
		t.Fatalf("la base efímera ya contiene claves Usuarios: %d %v", antes, err)
	}
	if err = admin.QueryRow(ctx, `SELECT count(*) FROM vec_autorizacion_atestada_v3.clave_capacidad_version`).Scan(&antes); err != nil || antes != 0 {
		t.Fatalf("el gobierno efímero no está vacío: %d %v", antes, err)
	}
	material := materialRenovableCTPrueba(t, time.Now().UTC().Truncate(time.Microsecond).Add(-24*time.Hour))
	if err = publicarGobiernoAtestacionContratacionTemporalDesarrollo(ctx, gobierno, &material); err != nil {
		t.Fatalf("clave CT base no publicable: %v", err)
	}
	catalogo, err := nuevoCatalogoMaterialAutorizacionComunDesarrollo(descriptoresMaterialPreferenciasUsuariosDesarrollo())
	if err != nil {
		t.Fatal(err)
	}
	const huellaFunciones = `SELECT md5(string_agg(pg_get_functiondef(p.oid),E'\n' ORDER BY p.oid::regprocedure::text))
 FROM pg_catalog.pg_proc p JOIN pg_catalog.pg_namespace n ON n.oid=p.pronamespace
 WHERE n.nspname='vec_autorizacion_atestada_v3' AND p.proname IN ('consumir_decision_mutacion_v3_interna','texto_tecnico_valido')`
	var funcionesAntes, funcionesDespues string
	if err = admin.QueryRow(ctx, huellaFunciones).Scan(&funcionesAntes); err != nil || funcionesAntes == "" {
		t.Fatalf("funciones previas ausentes: %v", err)
	}
	const restriccion = `ALTER TABLE vec_autorizacion_atestada_v3.clave_capacidad_version ADD CONSTRAINT vec_pref_lote_fallo_tercera
 CHECK (audiencia_consumo <> 'vec_usuarios.preferencias.consultar.externa_personal.v1')`
	const quitar = `ALTER TABLE vec_autorizacion_atestada_v3.clave_capacidad_version DROP CONSTRAINT vec_pref_lote_fallo_tercera`
	if _, err = admin.Exec(ctx, restriccion); err != nil {
		t.Fatalf("no se pudo provocar rechazo de tercera audiencia: %v", err)
	}
	defer func() {
		limpiar, cancelar := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancelar()
		_, _ = admin.Exec(limpiar, quitar)
	}()
	if _, err = publicarMaterialPreferenciasUsuariosEnLote(ctx, gobierno, material, relojContratacionTemporalDesarrollo{}, catalogo); err == nil {
		t.Fatal("tercera publicación rechazada aceptó lote")
	}
	var trasFallo int
	if err = admin.QueryRow(ctx, contarUsuarios).Scan(&trasFallo); err != nil || trasFallo != 0 {
		t.Fatalf("ROLLBACK dejó %d claves Usuarios: %v", trasFallo, err)
	}
	if err = admin.QueryRow(ctx, huellaFunciones).Scan(&funcionesDespues); err != nil || funcionesDespues != funcionesAntes {
		t.Fatalf("fallo cambió funciones anteriores: %v", err)
	}
	if _, err = admin.Exec(ctx, quitar); err != nil {
		t.Fatalf("retirar restricción efímera: %v", err)
	}
	proveedores, err := publicarMaterialPreferenciasUsuariosEnLote(ctx, gobierno, material, relojContratacionTemporalDesarrollo{}, catalogo)
	if err != nil {
		t.Fatalf("cuatro publicaciones atómicas: %v", err)
	}
	for i, p := range proveedores {
		if p == nil {
			t.Fatalf("proveedor %d no construido tras COMMIT", i)
		}
	}
	var confirmadas int
	if err = admin.QueryRow(ctx, contarUsuarios).Scan(&confirmadas); err != nil || confirmadas != 4 {
		t.Fatalf("COMMIT confirmó %d claves Usuarios: %v", confirmadas, err)
	}
	filas, err := admin.Query(ctx, `SELECT audiencia_consumo,count(*) FROM vec_autorizacion_atestada_v3.clave_capacidad_version
 WHERE audiencia_consumo LIKE 'vec_usuarios.preferencias.%' GROUP BY audiencia_consumo`)
	if err != nil {
		t.Fatal(err)
	}
	observadas := map[string]int64{}
	for filas.Next() {
		var audiencia string
		var n int64
		if err = filas.Scan(&audiencia, &n); err != nil {
			filas.Close()
			t.Fatal(err)
		}
		observadas[audiencia] = n
	}
	err = filas.Err()
	filas.Close()
	if err != nil {
		t.Fatal(err)
	}
	for _, d := range descriptoresMaterialPreferenciasUsuariosDesarrollo() {
		if observadas[d.Audiencia] != 1 {
			t.Fatalf("audiencia %s publicada %d veces", d.Audiencia, observadas[d.Audiencia])
		}
	}
	if err = admin.QueryRow(ctx, huellaFunciones).Scan(&funcionesDespues); err != nil || funcionesDespues != funcionesAntes {
		t.Fatalf("éxito alteró funciones previas: %v", err)
	}
}
