import importlib.util
import subprocess
import sys
import tempfile
import unittest
from pathlib import Path


SCRIPT = Path(__file__).with_name('copia_local_roles.py')
SPEC = importlib.util.spec_from_file_location('copia_local_roles', SCRIPT)
roles = importlib.util.module_from_spec(SPEC)
sys.modules[SPEC.name] = roles
SPEC.loader.exec_module(roles)


class RolesVolcadoTests(unittest.TestCase):
    def run_cli(self, schema, database='', types='', *, b119=False, sources=(), migrations=()):
        with tempfile.TemporaryDirectory() as directory:
            paths = []
            for index, content in enumerate((schema, database, types)):
                path = Path(directory, f'{index}.sql')
                path.write_text(content, encoding='utf-8')
                paths.append(str(path))
            extras = []
            for flag, contents in (('--fuente-roles', sources), ('--migracion', migrations)):
                for content in contents:
                    path = Path(directory, f'{flag[2:]}-{len(extras)}.sql')
                    path.write_text(content, encoding='utf-8')
                    extras.extend((flag, str(path)))
            return subprocess.run([sys.executable, str(SCRIPT), *paths, *extras,
                                   *(['--roles-b119'] if b119 else [])],
                                  text=True, capture_output=True, check=False)

    @staticmethod
    def created_lines(result):
        return [line.strip() for line in result.stdout.splitlines()
                if line.strip().startswith('CREATE ROLE ')]

    def test_b119_seeds_missing_names_without_login_or_privileges(self):
        result = self.run_cli('', b119=True)
        self.assertEqual(result.returncode, 0, result.stderr)
        self.assertEqual(self.created_lines(result), [
            'CREATE ROLE "vec_externo_documentos_bolsa_datos" NOLOGIN;',
            'CREATE ROLE "vec_interno_documentos_bolsa_rrhh" NOLOGIN;',
        ])

    def test_b119_name_already_in_dump_is_created_once(self):
        result = self.run_cli('GRANT SELECT ON TABLE t TO vec_externo_documentos_bolsa_datos;', b119=True)
        self.assertEqual(result.returncode, 0, result.stderr)
        self.assertEqual(result.stdout.count('CREATE ROLE "vec_externo_documentos_bolsa_datos"'), 1)

    def test_b119_invalid_dump_still_produces_no_partial_sql(self):
        result = self.run_cli('ALTER TABLE t OWNER TO CURRENT_USER;', b119=True)
        self.assertNotEqual(result.returncode, 0)
        self.assertEqual(result.stdout, '')

    def test_extracts_owners_acl_and_grantors_without_memberships(self):
        schema = '''
            \\restrict hidden-token
            -- ALTER TABLE fake OWNER TO false_comment;
            CREATE FUNCTION f() RETURNS text LANGUAGE plpgsql AS $body$
            BEGIN
              PERFORM 'ALTER TABLE fake OWNER TO false_string;';
              PERFORM 'GRANT SELECT ON fake TO false_body;';
              RETURN 'x';
            END $body$;
            ALTER FUNCTION public.f() OWNER TO "Propietario Uno";
            /* nested /* GRANT SELECT ON TABLE x TO false_comment; */ comment */
            GRANT SELECT, UPDATE ON TABLE public.t TO "lector, uno", publico
              WITH GRANT OPTION GRANTED BY "otorgante";
            GRANT role_a TO member_a;
            REVOKE ALL ON SCHEMA public FROM PUBLIC, "revocado" GRANTED BY admin;
            \\unrestrict hidden-token
        '''
        database = 'GRANT CONNECT ON DATABASE vec TO "base lector" GRANTED BY db_admin;'
        types = 'GRANT USAGE ON TYPE public.t TO "tipo lector";'
        result = self.run_cli(schema, database, types)
        self.assertEqual(result.returncode, 0, result.stderr)
        self.assertEqual(self.created_lines(result), [
            'CREATE ROLE "Propietario Uno" NOLOGIN;',
            'CREATE ROLE "admin" NOLOGIN;',
            'CREATE ROLE "base lector" NOLOGIN;',
            'CREATE ROLE "db_admin" NOLOGIN;',
            'CREATE ROLE "lector, uno" NOLOGIN;',
            'CREATE ROLE "otorgante" NOLOGIN;',
            'CREATE ROLE "publico" NOLOGIN;',
            'CREATE ROLE "revocado" NOLOGIN;',
            'CREATE ROLE "tipo lector" NOLOGIN;',
        ])
        self.assertNotIn('hidden-token', result.stdout + result.stderr)

    def test_quotes_role_names_and_skips_builtins(self):
        sql = '''ALTER TABLE owner ALTER COLUMN owner SET DEFAULT 'OWNER TO false';
                 ALTER TABLE x OWNER TO "evil""; DROP ROLE victim; --";
                 GRANT SELECT ON TABLE x TO postgres, pg_read_all_data, PUBLIC, "Mi Rol";
                 GRANT SELECT ON TABLE x TO "O'Brien\\dept$vec_copia_roles$";'''
        result = self.run_cli(sql)
        self.assertEqual(result.returncode, 0, result.stderr)
        self.assertEqual(self.created_lines(result), [
            'CREATE ROLE "Mi Rol" NOLOGIN;',
            'CREATE ROLE "O\'Brien\\dept$vec_copia_roles$" NOLOGIN;',
            'CREATE ROLE "evil""; DROP ROLE victim; --" NOLOGIN;',
        ])
        self.assertIn("rolname = E'O''Brien\\\\dept$vec_copia_roles$'", result.stdout)
        self.assertNotIn('DO $vec_copia_roles$ BEGIN', result.stdout)

    def test_rejects_ambiguous_role_without_partial_output(self):
        result = self.run_cli('ALTER TABLE x OWNER TO CURRENT_USER;')
        self.assertNotEqual(result.returncode, 0)
        self.assertEqual(result.stdout, '')
        self.assertNotIn('CURRENT_USER', result.stderr)

    def test_rejects_unclosed_literal_and_non_restrict_metacommand(self):
        for sql in ("CREATE FUNCTION f() RETURNS text AS $$ no end;",
                    '\\include private.sql\nALTER TABLE x OWNER TO valid;'):
            with self.subTest(sql=sql):
                result = self.run_cli(sql)
                self.assertNotEqual(result.returncode, 0)
                self.assertEqual(result.stdout, '')

    def test_default_privileges_and_revoke_variants(self):
        sql = '''ALTER DEFAULT PRIVILEGES FOR ROLE "dueño", "coautor" IN SCHEMA public
                 GRANT SELECT ON TABLES TO "lector" GRANTED BY "donante";
                 REVOKE GRANT OPTION FOR USAGE ON TYPE public.t
                 FROM "sin permiso" GRANTED BY "donante" CASCADE;'''
        result = self.run_cli(sql)
        self.assertEqual(result.returncode, 0, result.stderr)
        self.assertEqual(self.created_lines(result), [
            'CREATE ROLE "coautor" NOLOGIN;',
            'CREATE ROLE "donante" NOLOGIN;',
            'CREATE ROLE "dueño" NOLOGIN;',
            'CREATE ROLE "lector" NOLOGIN;',
            'CREATE ROLE "sin permiso" NOLOGIN;',
        ])

    def test_meritos_canonical_declaration_and_function_reference(self):
        base = Path(__file__).parents[1] / 'postgresql/meritos'
        source = (base / 'roles_up.sql').read_text()
        consulta = (base / 'roles_consulta_propia_up.sql').read_text()
        schema = '''CREATE FUNCTION public.comprobar() RETURNS boolean AS $f$
          BEGIN
            RETURN pg_has_role(session_user, 'vec_meritos_interno', 'MEMBER')
               OR to_regrole('vec_meritos_externo') IS NOT NULL
               OR 'vec_meritos_ejecutor'::regrole IS NOT NULL
               OR to_regrole('vec_meritos_consulta_propia_interno') IS NOT NULL;
          END $f$ LANGUAGE plpgsql;'''
        result = self.run_cli(schema, sources=[source, consulta])
        self.assertEqual(result.returncode, 0, result.stderr)
        self.assertEqual(self.created_lines(result), [
            'CREATE ROLE "vec_meritos_consulta_propia_interno" NOLOGIN;',
            'CREATE ROLE "vec_meritos_ejecutor" NOLOGIN;',
            'CREATE ROLE "vec_meritos_externo" NOLOGIN;',
            'CREATE ROLE "vec_meritos_interno" NOLOGIN;',
        ])
        self.assertEqual(result.stdout.count('IF NOT EXISTS'), 4)
        self.assertNotIn('GRANT ', result.stdout)
        self.assertNotIn(' LOGIN;', result.stdout)

    def test_references_are_roles_only_and_future_delta_creates_its_own(self):
        source = '''CREATE ROLE vec_citado NOLOGIN;
                    CREATE ROLE vec_delta NOLOGIN;
                    CREATE ROLE vec_literal NOLOGIN;
                    CREATE ROLE vec_login LOGIN;
                    -- CREATE ROLE vec_comentario NOLOGIN;
                    SELECT 'CREATE ROLE vec_falso NOLOGIN;';'''
        schema = '''CREATE TABLE vec_esquema (vec_citado text);
                    -- SELECT to_regrole('vec_comentario');
                    SELECT 'to_regrole(''vec_literal'')';
                    SELECT to_regrole('vec_login');
                    SELECT pg_has_role('vec_citado', 'MEMBER');'''
        delta = '''DO $pre$ BEGIN
                     IF to_regrole('vec_delta') IS NOT NULL THEN
                       RAISE EXCEPTION 'ya existe'; END IF;
                   END $pre$;
                   CREATE ROLE vec_delta NOLOGIN;'''
        result = self.run_cli(schema, sources=[source], migrations=[delta])
        self.assertEqual(result.returncode, 0, result.stderr)
        self.assertEqual(self.created_lines(result), ['CREATE ROLE "vec_citado" NOLOGIN;'])
        self.assertNotIn('vec_esquema', result.stdout)
        self.assertNotIn('vec_literal', result.stdout)
        self.assertNotIn('vec_login', result.stdout)

    def test_supplementary_deduplicates_dump_and_b119(self):
        source = 'CREATE ROLE vec_externo_documentos_bolsa_datos NOLOGIN;'
        schema = '''GRANT SELECT ON TABLE t TO vec_externo_documentos_bolsa_datos;
                    SELECT to_regrole('vec_externo_documentos_bolsa_datos');'''
        result = self.run_cli(schema, sources=[source], b119=True)
        self.assertEqual(result.returncode, 0, result.stderr)
        self.assertEqual(len(self.created_lines(result)), 2)

    def test_do_declarations_are_detected_without_reading_strings_or_comments(self):
        source = '''DO $body$ BEGIN
                      IF true THEN CREATE ROLE vec_do NOLOGIN; END IF;
                      PERFORM 'CREATE ROLE vec_falso NOLOGIN;';
                      -- CREATE ROLE vec_comentario NOLOGIN;
                    END $body$;'''
        schema = "SELECT to_regrole('vec_do'); SELECT to_regrole('vec_falso');"
        result = self.run_cli(schema, sources=[source])
        self.assertEqual(result.returncode, 0, result.stderr)
        self.assertEqual(self.created_lines(result), ['CREATE ROLE "vec_do" NOLOGIN;'])
        delta = '''DO $body$ BEGIN
                     IF true THEN CREATE ROLE vec_do NOLOGIN; END IF;
                   END $body$;'''
        result = self.run_cli(schema, sources=[source], migrations=[delta])
        self.assertEqual(result.returncode, 0, result.stderr)
        self.assertEqual(self.created_lines(result), [])

    def test_delta_login_excludes_canonical_nologin_name(self):
        source = 'CREATE ROLE vec_grupo NOLOGIN;'
        schema = "SELECT to_regrole('vec_grupo');"
        delta = 'CREATE ROLE vec_grupo LOGIN;'
        result = self.run_cli(schema, sources=[source], migrations=[delta])
        self.assertEqual(result.returncode, 0, result.stderr)
        self.assertEqual(self.created_lines(result), [])

    def test_pg_has_role_accepts_only_closed_literal_casts_in_function_body(self):
        source = '''CREATE ROLE vec_texto NOLOGIN;
                    CREATE ROLE vec_nombre NOLOGIN;
                    CREATE ROLE vec_regrole NOLOGIN;
                    CREATE ROLE vec_expresion NOLOGIN;'''
        schema = '''CREATE FUNCTION f() RETURNS boolean AS $func$
                    BEGIN
                      RETURN pg_catalog.pg_has_role(session_user, 'vec_texto'::text, 'MEMBER')
                         OR pg_has_role(session_user, 'vec_nombre'::name, 'MEMBER')
                         OR pg_has_role('vec_regrole'::regrole, 'MEMBER')
                         OR pg_has_role(session_user, 'vec_expresion'::varchar, 'MEMBER');
                    END $func$ LANGUAGE plpgsql;'''
        result = self.run_cli(schema, sources=[source])
        self.assertEqual(result.returncode, 0, result.stderr)
        self.assertEqual(self.created_lines(result), [
            'CREATE ROLE "vec_nombre" NOLOGIN;',
            'CREATE ROLE "vec_regrole" NOLOGIN;',
            'CREATE ROLE "vec_texto" NOLOGIN;',
        ])


if __name__ == '__main__':
    unittest.main()
