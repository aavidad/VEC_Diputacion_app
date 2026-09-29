import importlib.util
import json
from pathlib import Path
import tempfile
import unittest

spec = importlib.util.spec_from_file_location('clon_usuarios', Path(__file__).with_name('clon_usuarios.py'))
users = importlib.util.module_from_spec(spec)
spec.loader.exec_module(users)


class UsersTests(unittest.TestCase):
    def test_remap_changes_only_the_canonical_account(self):
        before = b'{"version":1,"cuentas":[{"cuenta_ref":"cta_original","sujeto":"subject","perfil_ref":"prf_original","certificado_sha256":"dummy"}],"dsn":"private-dsn"}'
        after = users.account_only(before, 'cta_canonical')
        expected = json.loads(before)
        expected['cuentas'][0]['cuenta_ref'] = 'cta_canonical'
        self.assertEqual(json.loads(after), expected)
        self.assertIn(b'"sujeto":"subject"', after)

    def test_remap_rejects_more_than_one_account_or_ambiguous_reference(self):
        for value in [b'{"cuentas":[]}', b'{"cuentas":[{"cuenta_ref":"cta_old"},{"cuenta_ref":"cta_other"}]}',
                      b'{"cuentas":[{"cuenta_ref":"cta_old"}],"extra":{"cuenta_ref":"cta_old"}}']:
            with self.subTest(value=value), self.assertRaises(users.UsersError):
                users.account_only(value, 'cta_new')

    def test_private_inputs_reject_symlinks_and_public_permissions(self):
        with tempfile.TemporaryDirectory() as folder:
            path = Path(folder) / 'private'
            path.write_text('synthetic')
            with self.assertRaises(users.UsersError):
                users.private(path)
            path.chmod(0o600)
            self.assertEqual(users.private(path), b'synthetic')
            link = Path(folder) / 'link'
            link.symlink_to(path)
            with self.assertRaises(users.UsersError):
                users.private(link)

    def test_historical_function_extraction_does_not_execute_old_tests(self):
        source = 'func authority() error {\n return nil\n}\nfunc TestOld(t *testing.T) {\n panic("must not execute")\n}\n'
        selected = users.selected_go_functions(source, ['authority'])
        self.assertIn('func authority()', selected)
        self.assertNotIn('TestOld', selected)
        with self.assertRaises(users.UsersError):
            users.selected_go_functions(source, ['unknown'])

    def test_technical_logins_match_only_the_private_H1_contract(self):
        configs = {}
        statements = ['BEGIN;']
        base = {'dsn_registro_identidad': 'vec_identidad_sesiones_v1_registrador',
                'dsn_revalidacion_identidad': 'vec_identidad_sesiones_v1_revalidador',
                'dsn_contexto': 'vec_contexto_actor_v1_runtime', 'dsn_fuente_autorizacion': 'vec_autorizacion_fuente',
                'dsn_registro_autorizacion': 'vec_autorizacion_registro', 'dsn_motivos': 'vec_autorizacion_motivos_evaluador'}
        for surface in ['interna', 'externa']:
            suffix = 'interno' if surface == 'interna' else 'externo'
            groups = dict(base, dsn_usuarios='vec_usuarios_ejecutor_' + suffix,
                          dsn_usuarios_frontera='vec_usuarios_registrador_frontera_' + suffix)
            config = {}
            for index, (field, group) in enumerate(groups.items()):
                name = 'fixture_' + surface + '_' + str(index)
                config[field] = 'postgres://' + name + '@127.0.0.1:55531/postgres?sslmode=verify-full'
                statements.append('CREATE ROLE ' + name + ' LOGIN INHERIT NOSUPERUSER NOCREATEDB NOCREATEROLE NOREPLICATION NOBYPASSRLS;')
                statements.append('GRANT ' + group + ' TO ' + name + ' WITH ADMIN FALSE, INHERIT TRUE, SET FALSE;')
            configs[surface] = config
        statements.append('COMMIT;')
        source = '\n'.join(statements).encode()
        self.assertEqual(len(users.login_plan(source, configs)), 16)
        with self.assertRaises(users.UsersError):
            users.login_plan(source + b'ALTER ROLE fixture_interna_0 SUPERUSER;', configs)
        configs['externa']['dsn_contexto'] = configs['interna']['dsn_contexto']
        with self.assertRaises(users.UsersError):
            users.login_plan(source, configs)

    def test_harness_preserves_original_person_and_emits_fresh_short_session(self):
        generated = users.make_harness(Path(__file__).parents[2])
        self.assertIn('original,historic', generated)
        self.assertIn('users.resolverSesion(', generated)
        self.assertIn('certificate trust', generated)
        self.assertNotIn('func TestProvisionarIdentidadPreferenciasHito1', generated)
        self.assertNotIn('INSERT INTO', generated)
        dto = users.selected_go_functions(generated, ['construirCuentaProvisionPreferenciasHito1'])
        self.assertNotIn('CrearVinculoAutenticacionActorV2ConResultado', dto)
        self.assertNotIn('SesionValidaHasta', dto)


if __name__ == '__main__':
    unittest.main()
