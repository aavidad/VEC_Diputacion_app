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
    def run_cli(self, schema, database='', types='', *, b119=False):
        with tempfile.TemporaryDirectory() as directory:
            paths = []
            for index, content in enumerate((schema, database, types)):
                path = Path(directory, f'{index}.sql')
                path.write_text(content, encoding='utf-8')
                paths.append(str(path))
            return subprocess.run([sys.executable, str(SCRIPT), *paths,
                                   *(['--roles-b119'] if b119 else [])],
                                  text=True, capture_output=True, check=False)

    def test_b119_seeds_missing_names_without_login_or_privileges(self):
        result = self.run_cli('', b119=True)
        self.assertEqual(result.returncode, 0, result.stderr)
        self.assertEqual(result.stdout.splitlines(), [
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
        self.assertEqual(result.stdout.splitlines(), [
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
                 GRANT SELECT ON TABLE x TO postgres, pg_read_all_data, PUBLIC, "Mi Rol";'''
        result = self.run_cli(sql)
        self.assertEqual(result.returncode, 0, result.stderr)
        self.assertEqual(result.stdout.splitlines(), [
            'CREATE ROLE "Mi Rol" NOLOGIN;',
            'CREATE ROLE "evil""; DROP ROLE victim; --" NOLOGIN;',
        ])

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
        self.assertEqual(result.stdout.splitlines(), [
            'CREATE ROLE "coautor" NOLOGIN;',
            'CREATE ROLE "donante" NOLOGIN;',
            'CREATE ROLE "dueño" NOLOGIN;',
            'CREATE ROLE "lector" NOLOGIN;',
            'CREATE ROLE "sin permiso" NOLOGIN;',
        ])


if __name__ == '__main__':
    unittest.main()
