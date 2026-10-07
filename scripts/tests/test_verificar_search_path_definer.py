"""Casos sintéticos de la puerta SQL; no se conecta a PostgreSQL."""

import unittest

from scripts.verificar_search_path_definer import added_lines, inspect_sql


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
        self.assertEqual(inspect_sql("ALTER FUNCTION a() SET search_path TO pg_catalog, pg_temp;", {1}), [])
        self.assertEqual(len(inspect_sql("ALTER PROCEDURE a() SET search_path TO public;", {1})), 1)

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

    def test_hunks_solo_lineas_añadidas(self):
        diff = "@@ -2,0 +3,2 @@\n+a\n+b\n@@ -8 +10 @@\n-x\n+y\n@@ -12 +13,0 @@\n-SET search_path=pg_catalog,pg_temp\n"
        self.assertEqual(added_lines(diff), {3, 4, 10, 13})


if __name__ == "__main__":
    unittest.main()
