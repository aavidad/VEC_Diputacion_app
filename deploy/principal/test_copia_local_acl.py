import importlib.util
import subprocess
import sys
import tempfile
import unittest
from pathlib import Path


SCRIPT = Path(__file__).with_name('copia_local_acl.py')
SPEC = importlib.util.spec_from_file_location('copia_local_acl', SCRIPT)
acl = importlib.util.module_from_spec(SPEC)
SPEC.loader.exec_module(acl)


class ACLTests(unittest.TestCase):
    def test_fixed_annex_forms(self):
        grants = acl.parse_base('''-- anexo
            BEGIN;
            REVOKE ALL ON DATABASE postgres FROM PUBLIC;
            GRANT CONNECT ON DATABASE postgres TO "lector";
            GRANT CREATE, CONNECT ON DATABASE postgres TO "nombre""doble";
            COMMIT;
        ''')
        self.assertEqual(grants, {('lector', 'CONNECT'), ('nombre"doble', 'CREATE'),
                                  ('nombre"doble', 'CONNECT')})
        types = acl.parse_types('''REVOKE ALL ON TYPE vec_bolsa_llamamientos.origen_oferta_ct_v1 FROM PUBLIC;
            REVOKE ALL ON TYPE x.y FROM PUBLIC;''')
        self.assertEqual(types, {('vec_bolsa_llamamientos', 'origen_oferta_ct_v1'), ('x', 'y')})

    def test_rejects_unexpected_or_duplicate_sql(self):
        for base in ('GRANT ALL ON DATABASE postgres TO "lector";',
                     'GRANT CONNECT ON DATABASE otra TO "lector";',
                     'BEGIN;\nREVOKE ALL ON DATABASE postgres FROM PUBLIC;',
                     'REVOKE ALL ON DATABASE postgres FROM PUBLIC;\nCOMMIT;',
                     'BEGIN;\nCOMMIT;\nGRANT CONNECT ON DATABASE postgres TO "lector";',
                     'REVOKE ALL ON DATABASE postgres FROM PUBLIC;\nGRANT CREATE, CREATE ON DATABASE postgres TO "lector";',
                     'REVOKE ALL ON DATABASE postgres FROM PUBLIC;\nGRANT CREATE, CONNECT, TEMPORARY ON DATABASE postgres TO "lector";',
                     'REVOKE ALL ON DATABASE postgres FROM PUBLIC;\nGRANT CONNECT ON DATABASE postgres TO "lector" WITH GRANT OPTION;',
                     'REVOKE ALL ON DATABASE postgres FROM PUBLIC;\nGRANT CONNECT ON DATABASE postgres TO lector;',
                     'GRANT CONNECT ON DATABASE postgres TO "lector";'):
            with self.subTest(base=base):
                with self.assertRaises(ValueError):
                    acl.parse_base(base)
        for types in ('REVOKE ALL ON TYPE x.y FROM PUBLIC;\nREVOKE ALL ON TYPE x.y FROM PUBLIC;',
                      'GRANT USAGE ON TYPE x.y TO PUBLIC;',
                      'REVOKE ALL ON TYPE "x".y FROM PUBLIC;'):
            with self.subTest(types=types):
                with self.assertRaises(ValueError):
                    acl.parse_types(types)

    def test_render_checks_exact_and_subset_modes(self):
        grants = {('lector', 'CONNECT')}
        types = {('x', 'y')}
        strict = acl.render(grants, types, False)
        extended = acl.render(grants, types, True)
        for sql in (strict, extended):
            self.assertTrue(sql.startswith('BEGIN READ ONLY;'))
            self.assertIn('SET LOCAL search_path = pg_catalog, pg_temp;', sql)
            self.assertIn("('postgres', 'TEMPORARY', false)", sql)
            self.assertIn("('lector', 'CONNECT', false)", sql)
            self.assertIn("ARRAY['x.y']", sql)
            self.assertIn('a.grantee = 0', sql)
            self.assertIn('v_missing', sql)
            self.assertIn('origen_oferta_ct_outbox_v1', sql)
            self.assertIn("OR a.privilege_type <> 'USAGE' OR a.is_grantable", sql)
            self.assertIn('ROLLBACK;', sql)
            self.assertNotIn('GRANT CONNECT ON DATABASE', sql)
        self.assertIn('IF v_extra THEN', strict)
        self.assertNotIn('IF v_extra THEN', extended)

    def test_cli_rejects_nonofficial_annex_without_output(self):
        with tempfile.TemporaryDirectory() as directory:
            base = Path(directory, 'base.sql')
            types = Path(directory, 'types.sql')
            base.write_text('REVOKE ALL ON DATABASE postgres FROM PUBLIC;')
            types.write_text('REVOKE ALL ON TYPE x.y FROM PUBLIC;')
            result = subprocess.run([sys.executable, str(SCRIPT), str(base), str(types)],
                                    text=True, capture_output=True, check=False)
        self.assertEqual(result.returncode, 1)
        self.assertEqual(result.stdout, '')
        self.assertNotIn(directory, result.stderr)


if __name__ == '__main__':
    unittest.main()
