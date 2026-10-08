"""Casos sintéticos de la puerta SQL; no se conecta a PostgreSQL."""

import unittest
from pathlib import Path

from scripts.verificar_search_path_definer import AD225_PATH, added_lines, inspect_sql


class VerificarSearchPathDefiner(unittest.TestCase):
    def test_definer_nuevo_correcto_multilinea(self):
        sql = '''CREATE OR REPLACE FUNCTION "vec prueba"."función"()
RETURNS integer LANGUAGE plpgsql SECURITY DEFINER
SET search_path = pg_catalog,
    pg_temp SET row_security = on AS $body$
BEGIN
  RETURN 1;
END
$body$;'''
        self.assertEqual(inspect_sql(sql, {6}), [])
        self.assertEqual(inspect_sql("CREATE FUNCTION a() RETURNS int AS $$ SELECT 1 $$ LANGUAGE sql SECURITY DEFINER SET search_path=pg_catalog,pg_temp;", {1}), [])

    def test_definer_nuevo_incorrecto(self):
        for path in ("", "SET search_path=public", "SET search_path=pg_temp,pg_catalog",
                     "SET search_path=pg_catalog,pg_temp,public", "SET search_path=pg_catalog"):
            with self.subTest(path=path):
                sql = f"CREATE FUNCTION a() RETURNS int LANGUAGE sql SECURITY DEFINER {path} AS $$ SELECT 1 $$;"
                self.assertEqual(len(inspect_sql(sql, {1})), 1)
        body_only = "CREATE FUNCTION a() RETURNS int SECURITY DEFINER AS $$ SELECT 'SET search_path=pg_catalog,pg_temp' $$;"
        self.assertEqual(len(inspect_sql(body_only, {1})), 1)

    def test_invocadora_y_historico_intacto(self):
        self.assertEqual(inspect_sql("CREATE FUNCTION a() RETURNS int LANGUAGE sql SECURITY INVOKER AS $$SELECT 1$$;", {1}), [])
        sql = "CREATE FUNCTION antigua() RETURNS int SECURITY DEFINER AS $$SELECT 1$$;\nSELECT 1;"
        self.assertEqual(inspect_sql(sql, {2}), [])

    def test_comentarios_literales_e_identificadores_no_son_definiciones(self):
        sql = '''-- CREATE FUNCTION falsa() SECURITY DEFINER;
/* CREATE FUNCTION falsa2() SECURITY DEFINER; /* comentario anidado */ */
SELECT 'CREATE FUNCTION falsa3() SECURITY DEFINER',
       "CREATE FUNCTION falsa4() SECURITY DEFINER";
DO $body$ BEGIN RAISE NOTICE 'CREATE FUNCTION falsa5() SECURITY DEFINER'; END $body$;'''
        self.assertEqual(inspect_sql(sql, set(range(1, 6))), [])

    def test_alter_path(self):
        for kind in ("FUNCTION", "PROCEDURE", "ROUTINE"):
            with self.subTest(kind=kind):
                self.assertEqual(inspect_sql(f"ALTER {kind} a() SET search_path TO pg_catalog, pg_temp;", {1}), [])
                self.assertEqual(len(inspect_sql(f"ALTER {kind} a() SET search_path TO public;", {1})), 1)
                self.assertEqual(len(inspect_sql(f"ALTER {kind} a() RESET search_path;", {1})), 1)
                self.assertEqual(len(inspect_sql(f"ALTER {kind} a() RESET ALL;", {1})), 1)
        self.assertEqual(inspect_sql("ALTER ROUTINE a() RESET lock_timeout;", {1}), [])

    def test_search_path_equivalente_con_literales_o_identificadores_citados(self):
        for clause in (
            "SET search_path TO 'pg_catalog', 'pg_temp'",
            'SET search_path = "pg_catalog", "pg_temp"',
            "SET search_path TO 'pg_catalog', 'pg_temp' SET lock_timeout='2s' AS $body$ SELECT 1 $body$",
        ):
            with self.subTest(clause=clause):
                suffix = "" if " AS $body$" in clause else " AS $$ SELECT 1 $$"
                create = f"CREATE FUNCTION a() RETURNS int SECURITY DEFINER {clause}{suffix};"
                self.assertEqual(inspect_sql(create, {1}), [])
                if suffix:
                    self.assertEqual(inspect_sql(f"ALTER ROUTINE a() {clause};", {1}), [])
        for clause in (
            "SET search_path TO 'pg_temp', 'pg_catalog'",
            "SET search_path TO 'pg_catalog', 'pg_temp', 'public'",
            "SET search_path TO 'pg_catalog, pg_temp'",
            "SET search_path TO 'pg_catalog', valor_variable",
            "SET search_path TO 'PG_CATALOG', 'pg_temp'",
            'SET search_path TO "pg_catalog", "pg_temp", "public"',
            "SET search_path TO 'pg_catalog'\n '_extra', 'pg_temp'",
            "SET search_path TO 'pg_catalog', 'pg_temp'\n ',public'",
            "SET search_path TO 'pg_catalog', 'pg_temp' || valor_variable",
            "SET search_path TO 'pg_catalog', 'pg_temp',",
        ):
            with self.subTest(clause=clause):
                create = f"CREATE FUNCTION a() RETURNS int SECURITY DEFINER {clause} AS $$ SELECT 1 $$;"
                self.assertTrue(inspect_sql(create, {1}))

    def test_reconstruccion_dinamica(self):
        sql = '''DO $body$
DECLARE original text;
BEGIN
 SELECT pg_get_functiondef('a()'::regprocedure) INTO original;
 EXECUTE original;
END
$body$;'''
        self.assertTrue(any("reconstrucción" in reason for _, reason in inspect_sql(sql, {5})))
        self.assertEqual(inspect_sql(sql, set()), [])

    def test_cadena_ejecutada_con_definicion(self):
        sql = "DO $$ BEGIN EXECUTE 'CREATE FUNCTION a() RETURNS int SECURITY DEFINER AS $f$ SELECT 1 $f$'; END $$;"
        self.assertTrue(inspect_sql(sql, {1}))
        sql_format = "DO $$ BEGIN EXECUTE format('CREATE FUNCTION %I() RETURNS int SECURITY DEFINER AS $f$ SELECT 1 $f$', 'a'); END $$;"
        self.assertTrue(inspect_sql(sql_format, {1}))
        sql_ok = "DO $$ BEGIN EXECUTE 'CREATE FUNCTION a() RETURNS int SECURITY DEFINER SET search_path=pg_catalog,pg_temp AS $f$ SELECT 1 $f$'; END $$;"
        self.assertEqual(inspect_sql(sql_ok, {1}), [])

    def test_ddl_dinamico_fragmentado(self):
        for sql in (
            "DO $$ BEGIN EXECUTE 'CREATE ' || 'FUNCTION a() RETURNS int SECURITY DEFINER AS $f$ SELECT 1 $f$'; END $$;",
            "DO $$ DECLARE ddl text; BEGIN ddl := 'CREATE FUNCTION a() RETURNS int SECURITY ' || 'DEFINER AS $f$ SELECT 1 $f$'; EXECUTE ddl; END $$;",
            "DO $$ BEGIN EXECUTE format('CREATE %s a() RETURNS int SECURITY DEFINER AS $f$ SELECT 1 $f$', 'FUNCTION'); END $$;",
            "DO $$ BEGIN EXECUTE 'ALTER ' || 'FUNCTION a() RESET search_path'; END $$;",
            "DO $$ BEGIN EXECUTE format('ALTER %s a() RESET ALL', 'ROUTINE'); END $$;",
            "DO $$ BEGIN EXECUTE format('ALTER FUNCTION %I() %s', 'a', 'RESET ALL'); END $$;",
            "DO $$ BEGIN EXECUTE format('ALTER FUNCTION %I() RESET %I', 'a', 'search_path'); END $$;",
            "DO $$ DECLARE ddl text; BEGIN ddl := 'ALTER PROCEDURE a() RESET ALL'; EXECUTE ddl; END $$;",
            "DO $outer$ BEGIN EXECUTE 'ALTER ' || $ddl$ROUTINE a() RESET ALL$ddl$; END $outer$;",
        ):
            with self.subTest(sql=sql):
                self.assertTrue(any("no verificable" in reason for _, reason in inspect_sql(sql, {1})))
        dollar_quoted = "DO $outer$ BEGIN EXECUTE $ddl$ALTER ROUTINE a() RESET ALL$ddl$; END $outer$;"
        self.assertTrue(inspect_sql(dollar_quoted, {1}))
        alter_ok = "DO $$ BEGIN EXECUTE format('ALTER ROUTINE %I() SET search_path=pg_catalog,pg_temp', 'a'); END $$;"
        self.assertEqual(inspect_sql(alter_ok, {1}), [])
        self.assertEqual(inspect_sql("DO $$ BEGIN EXECUTE 'DROP FUNCTION a()'; END $$;", {1}), [])

    def test_ad225_reconstruccion_exacta_revisada(self):
        repo = Path(__file__).resolve().parents[2]
        sql = (repo / AD225_PATH).read_text(encoding="utf-8")
        self.assertEqual(inspect_sql(sql, {10}, filename=AD225_PATH), [])
        self.assertEqual(inspect_sql(sql, {4}, filename=AD225_PATH), [])
        self.assertTrue(any("reconstrucción dinámica" in reason
                            for _, reason in inspect_sql(sql, {10}, filename="otro/archivo.sql")))
        self.assertTrue(any("reconstrucción dinámica" in reason
                            for _, reason in inspect_sql(sql, {10})))

    def test_ad225_mutaciones_y_ddl_ajeno_se_rechazan(self):
        repo = Path(__file__).resolve().parents[2]
        sql = (repo / AD225_PATH).read_text(encoding="utf-8")
        cambios = (
            ("p.proconfig=ARRAY['search_path=pg_catalog, pg_temp'",
             "p.proconfig=ARRAY['search_path=public, pg_temp'"),
            ("nuevo:=replace(original,marca,ampliacion);", "nuevo:=original;"),
            ("EXECUTE nuevo;", "EXECUTE original;"),
            ("replace(actual,ampliacion,marca) IS DISTINCT FROM original",
             "replace(actual,marca,ampliacion) IS DISTINCT FROM original"),
            ("79d2f29752235a01716d49095777fe8e890a6deed5269a8647d1f866d4b671b5",
             "559555ec535ad40cc3aad6361286899c28aede91ff73b2901eb30970d95ac986"),
            ("p_perfil_mutacion IS NOT DISTINCT FROM 'reanudacion_seleccion'",
             "p_perfil_mutacion IS NOT DISTINCT FROM 'reanudacion_orden_abierta'"),
            ("strpos(original,'reanudacion_solicitud_llamamiento')<>0",
             "strpos(original,'reanudacion_solicitud_llamamiento')=0"),
        )
        for original, alterado in cambios:
            with self.subTest(cambio=original[:35]):
                self.assertIn(original, sql)
                fallos = inspect_sql(sql.replace(original, alterado, 1), {10}, filename=AD225_PATH)
                self.assertTrue(any("reconstrucción dinámica" in reason for _, reason in fallos))
        no_aprobado = "DO $$ DECLARE ddl text; BEGIN SELECT pg_get_functiondef('a()'::regprocedure) INTO ddl; EXECUTE ddl; END $$;"
        self.assertTrue(inspect_sql(no_aprobado, {1}, filename=AD225_PATH))
        for original, alterado, linea in (
            ("SET LOCAL search_path = pg_catalog;", "SET LOCAL search_path = public, pg_catalog;", 4),
            ("SET LOCAL ROLE vec_autorizacion_atestada_v3_propietario;",
             "SET LOCAL ROLE vec_bolsa_llamamientos_propietario;", 3),
        ):
            with self.subTest(preambulo=original):
                self.assertIn(original, sql)
                fallos = inspect_sql(sql.replace(original, alterado, 1), {linea}, filename=AD225_PATH)
                self.assertTrue(any("preámbulo" in reason for _, reason in fallos))
        cabecera = "SET search_path = pg_catalog, pg_temp SET lock_timeout = '2s'"
        self.assertIn(cabecera, sql)
        insegura = sql.replace(cabecera, "SET search_path = public SET lock_timeout = '2s'", 1)
        linea = insegura[:insegura.index("SET search_path = public SET lock_timeout")].count("\n") + 1
        self.assertTrue(any("SECURITY DEFINER" in reason
                            for _, reason in inspect_sql(insegura, {linea}, filename=AD225_PATH)))

    def test_hunks_solo_lineas_añadidas(self):
        diff = "@@ -2,0 +3,2 @@\n+a\n+b\n@@ -8 +10 @@\n-x\n+y\n@@ -12 +13,0 @@\n-SET search_path=pg_catalog,pg_temp\n"
        self.assertEqual(added_lines(diff), {3, 4, 10, 13})


if __name__ == "__main__":
    unittest.main()
