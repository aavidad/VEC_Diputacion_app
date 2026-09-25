package internactproveedores

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
)

// Requiere una base PG18 vacía y desechable. Ninguna migración ni historia se
// aplica o revierte: crea solo los objetos mínimos para probar la sonda ACL.
func TestPerfilEfectivoPostgreSQL18RechazaConcesionesExtra(t *testing.T) {
	dsn := os.Getenv("VEC_CT_ACL_FOCAL_PG_DSN")
	if dsn == "" {
		t.Skip("PostgreSQL desechable no configurado")
	}
	if os.Getenv("VEC_CT_ACL_FOCAL_PG_DESECHABLE") != "si" {
		t.Fatal("se exige instancia desechable")
	}
	ctx, cancelar := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancelar()
	admin, err := pgx.Connect(ctx, dsn)
	if err != nil {
		t.Fatal("conexion PG18 no disponible")
	}
	defer admin.Close(context.Background())
	var version int
	if err := admin.QueryRow(ctx, "SELECT current_setting('server_version_num')::integer").Scan(&version); err != nil || version/10000 != 18 {
		t.Fatal("se exige PostgreSQL 18")
	}
	for _, sql := range []string{
		"CREATE ROLE vec_autorizacion_fuente NOLOGIN INHERIT NOSUPERUSER NOCREATEDB NOCREATEROLE NOREPLICATION NOBYPASSRLS",
		"CREATE ROLE vec_ct_acl_focal LOGIN INHERIT NOSUPERUSER NOCREATEDB NOCREATEROLE NOREPLICATION NOBYPASSRLS",
		"GRANT vec_autorizacion_fuente TO vec_ct_acl_focal WITH ADMIN FALSE, INHERIT TRUE, SET FALSE",
		"CREATE SCHEMA vec_autorizacion",
		"REVOKE ALL ON SCHEMA public FROM PUBLIC",
		"REVOKE ALL ON DATABASE postgres FROM PUBLIC",
		"GRANT CONNECT ON DATABASE postgres TO vec_autorizacion_fuente",
		"CREATE FUNCTION vec_autorizacion.obtener_instantanea(text,text) RETURNS integer LANGUAGE sql AS 'SELECT 1'",
		"REVOKE ALL ON FUNCTION vec_autorizacion.obtener_instantanea(text,text) FROM PUBLIC",
		"GRANT USAGE ON SCHEMA vec_autorizacion TO vec_autorizacion_fuente",
		"GRANT EXECUTE ON FUNCTION vec_autorizacion.obtener_instantanea(text,text) TO vec_autorizacion_fuente",
	} {
		if _, err := admin.Exec(ctx, sql); err != nil {
			t.Fatal("preparacion PG18 invalida")
		}
	}
	config, err := pgx.ParseConfig(dsn)
	if err != nil {
		t.Fatal(err)
	}
	config.User = "vec_ct_acl_focal"
	lector, err := pgx.ConnectConfig(ctx, config)
	if err != nil {
		t.Fatal("LOGIN nominal PG18 no disponible")
	}
	defer lector.Close(context.Background())
	perfil := perfilPool{nombre: "fuente_autorizacion", rol: "vec_autorizacion_fuente", funcion: "vec_autorizacion.obtener_instantanea(text,text)"}
	if err := acreditarPerfilEfectivo(ctx, lector, perfil); err != nil {
		t.Fatal("perfil mínimo rechazado")
	}
	for _, sql := range []string{
		"CREATE FUNCTION vec_autorizacion.funcion_extra() RETURNS integer LANGUAGE sql AS 'SELECT 1'",
		"REVOKE ALL ON FUNCTION vec_autorizacion.funcion_extra() FROM PUBLIC",
		"GRANT EXECUTE ON FUNCTION vec_autorizacion.funcion_extra() TO vec_autorizacion_fuente",
	} {
		if _, err := admin.Exec(ctx, sql); err != nil {
			t.Fatal("preparacion EXECUTE invalida")
		}
	}
	if err := acreditarPerfilEfectivo(ctx, lector, perfil); err == nil {
		t.Fatal("EXECUTE extra del grupo admitido")
	}
	if _, err := admin.Exec(ctx, "REVOKE EXECUTE ON FUNCTION vec_autorizacion.funcion_extra() FROM vec_autorizacion_fuente"); err != nil {
		t.Fatal(err)
	}
	if err := acreditarPerfilEfectivo(ctx, lector, perfil); err != nil {
		t.Fatal("perfil restaurado rechazado")
	}
	for _, sql := range []string{
		"CREATE TABLE vec_autorizacion.tabla_extra(id integer)",
		"REVOKE ALL ON TABLE vec_autorizacion.tabla_extra FROM PUBLIC",
		"GRANT INSERT ON TABLE vec_autorizacion.tabla_extra TO vec_autorizacion_fuente",
	} {
		if _, err := admin.Exec(ctx, sql); err != nil {
			t.Fatal("preparacion INSERT invalida")
		}
	}
	if err := acreditarPerfilEfectivo(ctx, lector, perfil); err == nil {
		t.Fatal("INSERT extra del grupo admitido")
	}
	if _, err := admin.Exec(ctx, "REVOKE INSERT ON TABLE vec_autorizacion.tabla_extra FROM vec_autorizacion_fuente"); err != nil {
		t.Fatal(err)
	}
	if err := acreditarPerfilEfectivo(ctx, lector, perfil); err != nil {
		t.Fatal("perfil restaurado tras INSERT rechazado")
	}
	var objeto uint32
	if err := admin.QueryRow(ctx, "SELECT pg_catalog.lo_create(0)").Scan(&objeto); err != nil {
		t.Fatal("LO sintético no disponible")
	}
	if _, err := admin.Exec(ctx, fmt.Sprintf("GRANT UPDATE ON LARGE OBJECT %d TO vec_autorizacion_fuente", objeto)); err != nil {
		t.Fatal("GRANT LO no disponible")
	}
	if err := acreditarPerfilEfectivo(ctx, lector, perfil); err == nil {
		t.Fatal("UPDATE LO heredado admitido")
	}
	if _, err := admin.Exec(ctx, fmt.Sprintf("REVOKE UPDATE ON LARGE OBJECT %d FROM vec_autorizacion_fuente", objeto)); err != nil {
		t.Fatal(err)
	}
	if err := acreditarPerfilEfectivo(ctx, lector, perfil); err != nil {
		t.Fatal("perfil restaurado tras LO rechazado")
	}
}

// perfilPreflight es el perfil gobierno_v3 exacto que abre Construir.
var perfilPreflight = perfilPool{nombre: "gobierno_v3", rol: "vec_autorizacion_atestada_v3_preflight_interno",
	funcion: "vec_autorizacion_atestada_v3.leer_configuracion_interna_v1(jsonb)"}

// Con objetos mínimos: el manifiesto del preflight admite exactamente las
// cuatro funciones v1+v2 que deja AD3-69; sobra o falta una y falla cerrado.
func TestPerfilPreflightPostgreSQL18ManifiestoExactoAD369(t *testing.T) {
	dsn := os.Getenv("VEC_CT_ACL_FOCAL_PG_DSN")
	if dsn == "" {
		t.Skip("PostgreSQL desechable no configurado")
	}
	if os.Getenv("VEC_CT_ACL_FOCAL_PG_DESECHABLE") != "si" {
		t.Fatal("se exige instancia desechable")
	}
	ctx, cancelar := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancelar()
	admin, err := pgx.Connect(ctx, dsn)
	if err != nil {
		t.Fatal("conexion PG18 no disponible")
	}
	defer admin.Close(context.Background())
	var version int
	if err := admin.QueryRow(ctx, "SELECT current_setting('server_version_num')::integer").Scan(&version); err != nil || version/10000 != 18 {
		t.Fatal("se exige PostgreSQL 18")
	}
	const esquema = "vec_autorizacion_atestada_v3"
	const grupo = "vec_autorizacion_atestada_v3_preflight_interno"
	preparar := []string{
		"CREATE ROLE " + grupo + " NOLOGIN INHERIT NOSUPERUSER NOCREATEDB NOCREATEROLE NOREPLICATION NOBYPASSRLS",
		"CREATE ROLE vec_ct_acl_preflight_focal LOGIN INHERIT NOSUPERUSER NOCREATEDB NOCREATEROLE NOREPLICATION NOBYPASSRLS",
		"GRANT " + grupo + " TO vec_ct_acl_preflight_focal WITH ADMIN FALSE, INHERIT TRUE, SET FALSE",
		"CREATE SCHEMA " + esquema,
		"REVOKE ALL ON SCHEMA public FROM PUBLIC",
		"REVOKE ALL ON DATABASE postgres FROM PUBLIC",
		"GRANT CONNECT ON DATABASE postgres TO " + grupo,
		"GRANT USAGE ON SCHEMA " + esquema + " TO " + grupo,
	}
	for _, f := range []struct{ firma, tipo string }{
		{"comprobar_material_emision_interna_v1(jsonb)", "boolean"}, {"leer_configuracion_interna_v1(jsonb)", "jsonb"},
		{"comprobar_material_emision_interna_v2(text,jsonb)", "boolean"}, {"leer_configuracion_interna_v2(text,jsonb)", "jsonb"},
		{"funcion_ajena_v1()", "boolean"},
	} {
		preparar = append(preparar, "CREATE FUNCTION "+esquema+"."+f.firma+" RETURNS "+f.tipo+" LANGUAGE sql AS 'SELECT NULL::"+f.tipo+"'",
			"REVOKE ALL ON FUNCTION "+esquema+"."+f.firma+" FROM PUBLIC")
	}
	for _, f := range funcionesEsperadasPerfil(perfilPreflight) {
		preparar = append(preparar, "GRANT EXECUTE ON FUNCTION "+f+" TO "+grupo)
	}
	for _, sql := range preparar {
		if _, err := admin.Exec(ctx, sql); err != nil {
			t.Fatalf("preparacion PG18 invalida: %s", sql)
		}
	}
	config, err := pgx.ParseConfig(dsn)
	if err != nil {
		t.Fatal(err)
	}
	config.User = "vec_ct_acl_preflight_focal"
	lector, err := pgx.ConnectConfig(ctx, config)
	if err != nil {
		t.Fatal("LOGIN nominal PG18 no disponible")
	}
	defer lector.Close(context.Background())
	if err := acreditarPerfilEfectivo(ctx, lector, perfilPreflight); err != nil {
		t.Fatal("manifiesto exacto AD3-69 rechazado")
	}
	leerV2 := esquema + ".leer_configuracion_interna_v2(text,jsonb)"
	cambios := []struct{ nombre, rompe, repara string }{
		{"sobra una funcion", "GRANT EXECUTE ON FUNCTION " + esquema + ".funcion_ajena_v1() TO " + grupo,
			"REVOKE EXECUTE ON FUNCTION " + esquema + ".funcion_ajena_v1() FROM " + grupo},
		{"falta leer v2", "REVOKE EXECUTE ON FUNCTION " + leerV2 + " FROM " + grupo,
			"GRANT EXECUTE ON FUNCTION " + leerV2 + " TO " + grupo},
		{"falta comprobar v1", "REVOKE EXECUTE ON FUNCTION " + esquema + ".comprobar_material_emision_interna_v1(jsonb) FROM " + grupo,
			"GRANT EXECUTE ON FUNCTION " + esquema + ".comprobar_material_emision_interna_v1(jsonb) TO " + grupo},
		{"v2 con opcion de concesion", "GRANT EXECUTE ON FUNCTION " + esquema + ".comprobar_material_emision_interna_v2(text,jsonb) TO " + grupo + " WITH GRANT OPTION",
			"REVOKE GRANT OPTION FOR EXECUTE ON FUNCTION " + esquema + ".comprobar_material_emision_interna_v2(text,jsonb) FROM " + grupo},
		{"estado previo a AD3-69", "DROP FUNCTION " + leerV2,
			"CREATE FUNCTION " + leerV2 + " RETURNS jsonb LANGUAGE sql AS 'SELECT NULL::jsonb';" +
				"REVOKE ALL ON FUNCTION " + leerV2 + " FROM PUBLIC;" +
				"GRANT EXECUTE ON FUNCTION " + leerV2 + " TO " + grupo},
	}
	for _, c := range cambios {
		if _, err := admin.Exec(ctx, c.rompe); err != nil {
			t.Fatalf("%s: preparacion invalida", c.nombre)
		}
		if err := acreditarPerfilEfectivo(ctx, lector, perfilPreflight); err == nil {
			t.Fatalf("%s: manifiesto admitido", c.nombre)
		}
		if _, err := admin.Exec(ctx, c.repara); err != nil {
			t.Fatalf("%s: restauracion invalida", c.nombre)
		}
		if err := acreditarPerfilEfectivo(ctx, lector, perfilPreflight); err != nil {
			t.Fatalf("%s: perfil restaurado rechazado", c.nombre)
		}
	}
}

// Contra la cadena AD3 real con AD3-69 instalada: el LOGIN nominal de
// preflight supera el manifiesto exacto que exige el binario.
func TestPerfilPreflightPostgreSQL18CadenaRealConAD369(t *testing.T) {
	dsn := os.Getenv("VEC_CT_ACL_AD369_PG_DSN")
	if dsn == "" {
		t.Skip("cadena AD3 con AD3-69 no configurada")
	}
	if os.Getenv("VEC_CT_ACL_FOCAL_PG_DESECHABLE") != "si" {
		t.Fatal("se exige instancia desechable")
	}
	ctx, cancelar := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancelar()
	con, err := pgx.Connect(ctx, dsn)
	if err != nil {
		t.Fatal("LOGIN de preflight PG18 no disponible")
	}
	defer con.Close(context.Background())
	var login string
	if err := con.QueryRow(ctx, "SELECT session_user::text").Scan(&login); err != nil || login != "vec_interno_preflight_v3_desarrollo" {
		t.Fatal("se exige el LOGIN nominal de preflight")
	}
	if err := acreditarPerfilEfectivo(ctx, con, perfilPreflight); err != nil {
		t.Fatal("la cadena real con AD3-69 no supera el manifiesto exacto")
	}
}
