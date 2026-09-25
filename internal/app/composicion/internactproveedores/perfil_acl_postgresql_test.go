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
