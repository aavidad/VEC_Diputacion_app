import importlib.util
import json
import os
from pathlib import Path
import signal
import socket
import subprocess
import sys
import tempfile
import unittest
from unittest.mock import patch

spec = importlib.util.spec_from_file_location('clon_runtime', Path(__file__).with_name('clon_runtime.py'))
runtime = importlib.util.module_from_spec(spec)
spec.loader.exec_module(runtime)


class RuntimeTests(unittest.TestCase):
    def setUp(self):
        self.temporary = tempfile.TemporaryDirectory()
        self.addCleanup(self.temporary.cleanup)
        self.root = Path(self.temporary.name)
        self.root.chmod(0o700)
        self.source = self.root / 'source'
        (self.source / 'config').mkdir(parents=True)
        self.material = self.root / 'material'
        for name in ('ca/ca.crt', 'tls/servidor.crt', 'tls/servidor.key'):
            path = self.material / name
            path.parent.mkdir(parents=True, exist_ok=True)
            path.write_text('synthetic')
        self.values = {
            'VEC_EXECUTION_PROFILE': 'desarrollo', 'VEC_AUTH_MODE': 'desarrollo',
            'VEC_DEVELOPMENT_GUARD': 'ACEPTO_CREDENCIALES_NO_AUTORITATIVAS_SOLO_DESARROLLO',
            'VEC_DEVELOPMENT_MATERIAL_DIR': str(self.material),
            'VEC_TLS_CERT_FILE': str(self.material / 'tls/servidor.crt'),
            'VEC_TLS_KEY_FILE': str(self.material / 'tls/servidor.key'),
            'VEC_CT_DATABASE_URL': 'postgres://actor:dummy@127.0.0.1:55531/synthetic?sslmode=verify-full',
            'VEC_CT_REGISTRO_IDENTIDAD_DATABASE_URL': 'postgres://actor:dummy@127.0.0.1:55531/synthetic?sslmode=verify-full',
            'VEC_CT_AUDITORIA_FRONTERA_DATABASE_URL': 'postgres://actor:dummy@127.0.0.1:55531/synthetic?sslmode=verify-full',
        }
        (self.source / 'config/config.go').write_text('\n'.join('"' + key + '"' for key in self.values))
        self.config = self.root / 'runtime-config.json'
        self.save()

    def save(self):
        self.config.write_text(json.dumps(self.values))
        self.config.chmod(0o600)

    def test_material_rejects_remote_dsn_in_auxiliary_json(self):
        auxiliary = self.material / 'users.json'
        auxiliary.write_text(json.dumps({'nested': {'dsn': 'postgres://dummy@remote.invalid:55531/synthetic?sslmode=verify-full'}}))
        manifest = {'owner': 'Codex-M', 'target': {'source_commit': 'a' * 40, 'app_port': 18531, 'pg_port': 55531},
                    'files': {'material/users.json': runtime.digest(auxiliary)}}
        runtime.write_json(self.root / 'material-manifest.json', manifest)
        with self.assertRaises(runtime.RuntimeErrorLocal):
            runtime.validate_material(self.root, 'a' * 40, 18531, 55531)
        auxiliary.write_text('{}')
        with self.assertRaises(runtime.RuntimeErrorLocal):
            runtime.validate_material(self.root, 'a' * 40, 18531, 55531)

    def smtp_fixture(self):
        ca = self.material / 'ca/ca.crt'
        ca.chmod(0o600)
        self.values.update({'VEC_SMTP_HOST': '127.0.0.1', 'VEC_SMTP_PORT': '11025',
                            'VEC_SMTP_FROM': 'rrhh@example.test', 'VEC_SMTP_CA_FILE': str(ca),
                            'VEC_SMTP_MODO_TLS': 'starttls'})
        with (self.source / 'config/config.go').open('a') as stream:
            stream.write('\n'.join('"' + k + '"' for k in self.values if k.startswith('VEC_SMTP_')))
        proxy = self.root / 'comunicaciones/proxy-11025.json'
        proxy.parent.mkdir()
        proxy.write_text('{}')
        proxy.chmod(0o600)
        proof = {'smtp_ready': True, 'smtp_scope': 'synthetic_local_sink', 'corporate_delivery': False,
                 'starttls_verified': True, 'external_recipient_rejected': True,
                 'mailpit_container': 'vec-codexm-recorridos-mailpit-20260930',
                 'loopback_proxies': [{'port': 11025, 'pid': 41, 'record': str(proxy)}]}
        profiles = self.root / 'perfiles.json'
        profiles.write_text(json.dumps({'profiles': {'usuarios_comunicaciones': proof}}))
        profiles.chmod(0o600)
        self.save()
        labels = {'vec.recorridos.owner': 'Codex-M', 'vec.recorridos.state': str(self.root)}
        network_name = 'vec-codexm-recorridos-mailpit-red-20260930'
        sink = {'Image': 'fixture-image', 'State': {'Running': True},
                'Config': {'Labels': labels, 'Cmd': ['--disable-version-check', '--block-remote-css-and-fonts', '--smtp-disable-rdns', '--smtp-require-starttls', '--smtp-tls-cert', '/tls/servidor.crt', '--smtp-tls-key', '/tls/servidor.key', '--smtp-allowed-recipients', r'^[A-Za-z0-9._+\-]+@example\.test$', '--smtp', '0.0.0.0:1025', '--listen', '0.0.0.0:8025', '--database', '/data/mailpit.db', '--max', '100', '--quiet']},
                'HostConfig': {'ReadonlyRootfs': True, 'NetworkMode': network_name,
                               'PortBindings': {'1025/tcp': [{'HostIp': '127.0.0.1', 'HostPort': '11025'}],
                                                '8025/tcp': [{'HostIp': '127.0.0.1', 'HostPort': '18532'}]}},
                'Mounts': [{'Destination': '/tls', 'Source': str(self.root / 'material/comunicaciones'), 'RW': False}],
                'NetworkSettings': {'Networks': {network_name: {'IPAddress': '172.31.0.2'}}}}
        network = {'Labels': labels, 'Internal': True}
        return sink, network, {'Id': 'fixture-image'}

    def test_owned_smtp_fixture_uses_explicit_local_tls_and_ignores_ambient(self):
        resources = self.smtp_fixture()
        with patch.object(runtime, 'inspect_smtp_resource', side_effect=resources), patch.object(runtime, 'smtp_proxy_identity', return_value=41), patch.dict(os.environ, {'VEC_SMTP_HOST': 'remote.invalid', 'VEC_SMTP_PASSWORD': 'dummy'}):
            environment, _ = runtime.runtime_environment(self.source, self.root, 18531, 55531)
        self.assertEqual(environment['VEC_SMTP_HOST'], '127.0.0.1')
        self.assertEqual(environment['VEC_SMTP_PORT'], '11025')
        self.assertEqual(environment['VEC_SMTP_MODO_TLS'], 'starttls')
        self.assertNotIn('VEC_SMTP_PASSWORD', environment)

    def test_smtp_rejects_remote_other_ports_cleartext_and_foreign_ca(self):
        self.smtp_fixture()
        for field, value in [('VEC_SMTP_HOST', 'remote.invalid'), ('VEC_SMTP_HOST', 'localhost'),
                             ('VEC_SMTP_PORT', '25'), ('VEC_SMTP_PORT', '11026'),
                             ('VEC_SMTP_MODO_TLS', 'none'), ('VEC_SMTP_MODO_TLS', 'tls'),
                             ('VEC_SMTP_CA_FILE', '/outside/ca.crt'), ('VEC_SMTP_FROM', 'rrhh@external.invalid')]:
            changed = dict(self.values, **{field: value})
            with self.subTest(field=field, value=value), self.assertRaises(runtime.RuntimeErrorLocal):
                runtime.validate_smtp(changed, self.root)

    def test_smtp_rejects_missing_proof_external_network_and_foreign_container(self):
        resources = self.smtp_fixture()
        self.assertIsNone(runtime.validate_smtp({}, self.root))
        (self.root / 'perfiles.json').unlink()
        with self.assertRaises(runtime.RuntimeErrorLocal):
            runtime.validate_smtp(self.values, self.root)
        (self.root / 'perfiles.json').write_text(json.dumps({'profiles': {'usuarios_comunicaciones': {}}}))
        (self.root / 'perfiles.json').chmod(0o600)
        with self.assertRaises(runtime.RuntimeErrorLocal):
            runtime.validate_smtp(self.values, self.root)

    def test_smtp_rejects_extra_network_and_changed_server_command(self):
        for failure in ['network', 'command', 'listener']:
            with self.subTest(failure=failure):
                resources = list(self.smtp_fixture())
                if failure == 'network':
                    resources[1]['Internal'] = False
                elif failure == 'command':
                    resources[0]['Config']['Cmd'].append('--smtp-disable-starttls')
                else:
                    resources[0]['HostConfig']['PortBindings']['1025/tcp'][0]['HostIp'] = '0.0.0.0'
                with patch.object(runtime, 'inspect_smtp_resource', side_effect=resources), self.assertRaises(runtime.RuntimeErrorLocal):
                    runtime.validate_smtp(self.values, self.root)
                (self.root / 'comunicaciones/proxy-11025.json').unlink()
                (self.root / 'comunicaciones').rmdir()

    def test_smtp_rejects_foreign_sink_before_proxy_checks(self):
        resources = list(self.smtp_fixture())
        resources[0]['Config']['Labels']['vec.recorridos.owner'] = 'other'
        with patch.object(runtime, 'inspect_smtp_resource', side_effect=resources), self.assertRaises(runtime.RuntimeErrorLocal):
            runtime.validate_smtp(self.values, self.root)

    def test_private_state_cannot_be_a_repository_or_symlink(self):
        repo = self.root / 'repo'
        repo.mkdir()
        (repo / '.git').mkdir()
        with self.assertRaises(runtime.RuntimeErrorLocal):
            runtime.validate_state(repo, repo / 'private')
        link = self.root / 'link'
        link.symlink_to(self.material)
        with self.assertRaises(runtime.RuntimeErrorLocal):
            runtime.validate_state(repo, link)

    def test_runtime_ignores_ambient_credentials(self):
        with patch.dict(os.environ, {'VEC_CT_DATABASE_URL': 'postgres://remote.invalid/private',
                                     'AWS_SECRET_ACCESS_KEY': 'dummy', 'PGHOST': 'remote.invalid'}):
            env, _ = runtime.runtime_environment(self.source, self.root, 18531, 55531)
        self.assertEqual(env['VEC_HTTP_ADDR'], '127.0.0.1:18531')
        self.assertNotIn('AWS_SECRET_ACCESS_KEY', env)
        self.assertNotIn('PGHOST', env)
        self.assertIn('127.0.0.1:55531', env['VEC_CT_DATABASE_URL'])

    def test_dsn_rejects_remote_other_local_ports_and_overrides(self):
        bad = [
            'postgres://dummy@remote.invalid:55531/synthetic?sslmode=verify-full',
            'postgres://dummy@127.0.0.1:55441/synthetic?sslmode=verify-full',
            'postgres://dummy@127.0.0.1:55531/synthetic?sslmode=disable',
            'postgres://dummy@127.0.0.1:55531/synthetic?sslmode=require',
            'postgres://dummy@127.0.0.1:55531/synthetic?sslmode=verify-full&host=remote.invalid',
            'postgres://dummy@127.0.0.1:55531/synthetic?sslmode=verify-full&hostaddr=192.0.2.1',
            'postgres://dummy@127.0.0.1:55531/synthetic?sslmode=verify-full&sslmode=disable',
            'postgres://dummy@127.0.0.1:55531/synthetic?sslmode=verify-full&sslrootcert=/outside/file',
        ]
        for value in bad:
            with self.subTest(value=value), self.assertRaises(runtime.RuntimeErrorLocal):
                runtime.validate_dsn(value, 55531, self.root)

    def test_material_path_cannot_follow_symlink(self):
        link = self.material / 'tls/link.key'
        link.symlink_to(self.material / 'tls/servidor.key')
        self.values['VEC_TLS_KEY_FILE'] = str(link)
        self.save()
        with self.assertRaises(runtime.RuntimeErrorLocal):
            runtime.runtime_environment(self.source, self.root, 18531, 55531)

    def test_bootstrap_selectors_need_an_actual_environment_read(self):
        bootstrap = self.source / 'internal/app/bootstrap'
        bootstrap.mkdir(parents=True)
        (bootstrap / 'users.go').write_text('const envUsers = "VEC_USUARIOS_PREFERENCIAS_ENABLED"\n'
                                           'const unused = "VEC_UNUSED_SELECTOR"\n'
                                           'func selectorCapacidadRRHHDesarrollo(cfg config.Config, nombre string) {os.Getenv(nombre)}\n'
                                           'selectorCapacidadRRHHDesarrollo(cfg, envUsers)')
        self.values['VEC_USUARIOS_PREFERENCIAS_ENABLED'] = 'true'
        self.save()
        env, _ = runtime.runtime_environment(self.source, self.root, 18531, 55531)
        self.assertEqual(env['VEC_USUARIOS_PREFERENCIAS_ENABLED'], 'true')
        self.values['VEC_UNUSED_SELECTOR'] = 'true'
        self.save()
        with self.assertRaises(runtime.RuntimeErrorLocal):
            runtime.runtime_environment(self.source, self.root, 18531, 55531)

    def test_allowlist_accepts_only_the_exact_loopback_cidr(self):
        self.values['VEC_HTTP_ALLOWED_CIDRS'] = '127.0.0.1/32'
        with (self.source / 'config/config.go').open('a') as out:
            out.write('"VEC_HTTP_ALLOWED_CIDRS"')
        self.save()
        env, _ = runtime.runtime_environment(self.source, self.root, 18531, 55531)
        self.assertEqual(env['VEC_HTTP_ALLOWED_CIDRS'], '127.0.0.1/32')
        for value in ['0.0.0.0/0', '127.0.0.0/8', '127.0.0.1/32,192.0.2.0/24']:
            self.values['VEC_HTTP_ALLOWED_CIDRS'] = value
            self.save()
            with self.subTest(value=value), self.assertRaises(runtime.RuntimeErrorLocal):
                runtime.runtime_environment(self.source, self.root, 18531, 55531)

    def test_unknown_variables_and_outbound_endpoints_rejected(self):
        self.values['LD_PRELOAD'] = '/dummy'
        self.save()
        with self.assertRaises(runtime.RuntimeErrorLocal):
            runtime.runtime_environment(self.source, self.root, 18531, 55531)
        self.values.pop('LD_PRELOAD')
        self.values['VEC_SMTP_HOST'] = 'remote.invalid'
        self.save()
        with (self.source / 'config/config.go').open('a') as out:
            out.write('"VEC_SMTP_HOST"')
        with self.assertRaises(runtime.RuntimeErrorLocal):
            runtime.runtime_environment(self.source, self.root, 18531, 55531)

    def test_env_parser_does_not_evaluate_shell(self):
        self.config.unlink()
        env = self.root / 'runtime.env'
        env.write_text("VEC_AUTH_MODE=$(touch /dummy)\n")
        env.chmod(0o600)
        with self.assertRaises(runtime.RuntimeErrorLocal):
            runtime.read_environment(self.root)
        env.write_text("VEC_AUTH_MODE='desarrollo'\n")
        self.assertEqual(runtime.read_environment(self.root)[0], {'VEC_AUTH_MODE': 'desarrollo'})

    def test_stop_refuses_unrelated_pid_even_if_record_exists(self):
        identity = runtime.process_identity(os.getpid())
        record = dict(identity, start_ticks='wrong', binary_sha256='dummy')
        runtime.write_json(self.root / 'runtime-process.json', record)
        with patch.object(signal, 'pidfd_send_signal') as send_signal:
            with self.assertRaises(runtime.RuntimeErrorLocal):
                runtime.stop(self.root)
            send_signal.assert_not_called()

    def test_pinned_main_rejects_branch_revision(self):
        repo = self.root / 'repo'
        subprocess.run(['git', 'init', '-q', '-b', 'main', str(repo)], check=True)
        subprocess.run(['git', '-C', str(repo), '-c', 'user.name=Fixture', '-c', 'user.email=fixture@example.invalid',
                        'commit', '-qm', 'base', '--allow-empty'], check=True)
        commit = runtime.git(repo, 'rev-parse', 'HEAD')
        self.assertEqual(runtime.pinned_main(repo, commit), commit)
        subprocess.run(['git', '-C', str(repo), 'switch', '-qc', 'task'], check=True)
        subprocess.run(['git', '-C', str(repo), '-c', 'user.name=Fixture', '-c', 'user.email=fixture@example.invalid',
                        'commit', '-qm', 'task', '--allow-empty'], check=True)
        with self.assertRaises(runtime.RuntimeErrorLocal):
            runtime.pinned_main(repo, runtime.git(repo, 'rev-parse', 'HEAD'))


if __name__ == '__main__':
    unittest.main()
