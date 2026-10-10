package bootstrap

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// El guion scripts/probar_portal_externo_acl_pg18.sh crea una base vacía y
// desechable. Las funciones son testigos de catálogo, no dobles de los actos:
// este ensayo verifica el preflight real y las ACL de PostgreSQL, sin ejecutar
// inscripción ni solicitudes documentales.
func TestPortalExternoACLPostgreSQL18(t *testing.T) {
	if os.Getenv("VEC_PORTAL_ACL_PG18_DESECHABLE") != "1" {
		t.Skip("requiere el PostgreSQL 18 desechable del guion de ensayo")
	}
	ctx, cancelar := context.WithTimeout(t.Context(), time.Minute)
	defer cancelar()
	admin, err := pgxpool.New(ctx, os.Getenv("VEC_PORTAL_ACL_PG18_ADMIN_DSN"))
	if err != nil {
		t.Fatal(err)
	}
	defer admin.Close()
	ejecutar := func(sql string) {
		t.Helper()
		if _, err := admin.Exec(ctx, sql); err != nil {
			t.Fatal(err)
		}
	}
	ejecutar(`CREATE ROLE vec_bolsa_llamamientos_portal_externo NOLOGIN;
CREATE ROLE vec_externo_bolsa_desarrollo LOGIN;
GRANT vec_bolsa_llamamientos_portal_externo TO vec_externo_bolsa_desarrollo WITH INHERIT TRUE, SET FALSE;
REVOKE TEMPORARY ON DATABASE postgres FROM PUBLIC;
CREATE SCHEMA vec_bolsa_llamamientos;
REVOKE ALL ON SCHEMA vec_bolsa_llamamientos FROM PUBLIC;
GRANT USAGE ON SCHEMA vec_bolsa_llamamientos TO vec_bolsa_llamamientos_portal_externo;`)
	// Lista del rol B59+B77, independiente de la lista de producción.
	for _, nombre := range []string{
		"consultar_mi_bolsa_v1", "consultar_mi_bolsa_portal_v1", "consultar_historial_mi_bolsa_v1",
		"manifestar_disposicion_oferta_v1", "listar_ofertas_candidato_v1", "solicitar_portal_candidato_v1",
		"responder_llamamiento_portal_v1", "preparar_respuesta_portal_v1", "leer_portal_candidato_v1",
		"confirmar_contacto_propio_v1", "leer_contacto_candidato_v1",
		"solicitar_documental_portal_v1",
	} {
		// Nombres internos fijos: nunca proceden de una entrada del usuario.
		ejecutar(`CREATE FUNCTION vec_bolsa_llamamientos.` + nombre + `() RETURNS integer
LANGUAGE SQL SECURITY DEFINER SET search_path=pg_catalog,pg_temp AS 'SELECT 1';
REVOKE ALL ON FUNCTION vec_bolsa_llamamientos.` + nombre + `() FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_bolsa_llamamientos.` + nombre + `() TO vec_bolsa_llamamientos_portal_externo;`)
	}
	dsn := os.Getenv("VEC_PORTAL_ACL_PG18_EXTERNO_DSN")
	comprobar := func(miBolsa, inscripcion bool) {
		t.Helper()
		inicio := time.Now()
		pool, _, err := abrirBolsaMiBolsaPortalExterno(ctx, dsn)
		if pool != nil {
			pool.Close()
		}
		if (err == nil) != miBolsa {
			t.Fatalf("mi_bolsa permitido=%v esperado=%v error=%v", err == nil, miBolsa, err)
		}
		pool, err = abrirEjecutorExternoInscripcionBolsa(ctx, dsn)
		if pool != nil {
			pool.Close()
		}
		if (err == nil) != inscripcion {
			t.Fatalf("inscripcion permitida=%v esperada=%v error=%v", err == nil, inscripcion, err)
		}
		t.Logf("dos preflights PostgreSQL/TLS: %s", time.Since(inicio))
	}
	t.Run("B77_sin_B96", func(t *testing.T) { comprobar(true, false) })
	ejecutar(`CREATE FUNCTION ` + firmaSolicitarInscripcionPortalExterno + ` RETURNS integer
LANGUAGE SQL SECURITY DEFINER SET search_path=pg_catalog,pg_temp AS 'SELECT 1';
REVOKE ALL ON FUNCTION ` + firmaSolicitarInscripcionPortalExterno + ` FROM PUBLIC;
GRANT EXECUTE ON FUNCTION ` + firmaSolicitarInscripcionPortalExterno + ` TO vec_bolsa_llamamientos_portal_externo;`)
	t.Run("B77_y_B96", func(t *testing.T) { comprobar(true, true) })
	for _, caso := range []struct{ nombre, alterar, restaurar string }{
		{"funcion_extra", `CREATE FUNCTION vec_bolsa_llamamientos.ajena() RETURNS integer LANGUAGE SQL SECURITY DEFINER SET search_path=pg_catalog,pg_temp AS 'SELECT 1'; GRANT EXECUTE ON FUNCTION vec_bolsa_llamamientos.ajena() TO vec_bolsa_llamamientos_portal_externo`, `DROP FUNCTION vec_bolsa_llamamientos.ajena()`},
		{"sobrecarga_B77", `CREATE FUNCTION vec_bolsa_llamamientos.solicitar_documental_portal_v1(text) RETURNS integer LANGUAGE SQL SECURITY DEFINER SET search_path=pg_catalog,pg_temp AS 'SELECT 1'; GRANT EXECUTE ON FUNCTION vec_bolsa_llamamientos.solicitar_documental_portal_v1(text) TO vec_bolsa_llamamientos_portal_externo`, `DROP FUNCTION vec_bolsa_llamamientos.solicitar_documental_portal_v1(text)`},
		{"sobrecarga_B96_sin_grant", `CREATE FUNCTION vec_bolsa_llamamientos.solicitar_inscripcion_v1(text) RETURNS integer LANGUAGE SQL SECURITY DEFINER SET search_path=pg_catalog,pg_temp AS 'SELECT 1'; REVOKE ALL ON FUNCTION vec_bolsa_llamamientos.solicitar_inscripcion_v1(text) FROM PUBLIC`, `DROP FUNCTION vec_bolsa_llamamientos.solicitar_inscripcion_v1(text)`},
		{"tabla", `CREATE TABLE vec_bolsa_llamamientos.privada(id integer); GRANT SELECT ON vec_bolsa_llamamientos.privada TO vec_bolsa_llamamientos_portal_externo`, `DROP TABLE vec_bolsa_llamamientos.privada`},
		{"columna", `CREATE TABLE vec_bolsa_llamamientos.privada(id integer); GRANT SELECT(id) ON vec_bolsa_llamamientos.privada TO vec_bolsa_llamamientos_portal_externo`, `DROP TABLE vec_bolsa_llamamientos.privada`},
		{"secuencia", `CREATE SEQUENCE vec_bolsa_llamamientos.privada; GRANT USAGE ON SEQUENCE vec_bolsa_llamamientos.privada TO vec_bolsa_llamamientos_portal_externo`, `DROP SEQUENCE vec_bolsa_llamamientos.privada`},
		{"crear_en_esquema", `GRANT CREATE ON SCHEMA vec_bolsa_llamamientos TO vec_bolsa_llamamientos_portal_externo`, `REVOKE CREATE ON SCHEMA vec_bolsa_llamamientos FROM vec_bolsa_llamamientos_portal_externo`},
		{"otro_esquema", `CREATE SCHEMA vec_ajeno; GRANT USAGE ON SCHEMA vec_ajeno TO vec_bolsa_llamamientos_portal_externo`, `DROP SCHEMA vec_ajeno`},
		{"B77_ausente", `REVOKE EXECUTE ON FUNCTION vec_bolsa_llamamientos.solicitar_documental_portal_v1() FROM vec_bolsa_llamamientos_portal_externo`, `GRANT EXECUTE ON FUNCTION vec_bolsa_llamamientos.solicitar_documental_portal_v1() TO vec_bolsa_llamamientos_portal_externo`},
		{"B96_publica", `GRANT EXECUTE ON FUNCTION ` + firmaSolicitarInscripcionPortalExterno + ` TO PUBLIC`, `REVOKE EXECUTE ON FUNCTION ` + firmaSolicitarInscripcionPortalExterno + ` FROM PUBLIC`},
		{"B96_sin_grant", `REVOKE EXECUTE ON FUNCTION ` + firmaSolicitarInscripcionPortalExterno + ` FROM vec_bolsa_llamamientos_portal_externo`, `GRANT EXECUTE ON FUNCTION ` + firmaSolicitarInscripcionPortalExterno + ` TO vec_bolsa_llamamientos_portal_externo`},
	} {
		t.Run(caso.nombre, func(t *testing.T) {
			ejecutar(caso.alterar)
			defer ejecutar(caso.restaurar)
			comprobar(false, false)
		})
	}
	t.Run("restauradas", func(t *testing.T) { comprobar(true, true) })
}
