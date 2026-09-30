import importlib.util
import hashlib
import json
import os
from pathlib import Path
import signal
import shutil
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

    def smtp_fixture(self, smtp_port=11025, http_port=18532):
        ca = self.material / 'ca/ca.crt'
        ca.chmod(0o600)
        self.values.update({'VEC_SMTP_HOST': '127.0.0.1', 'VEC_SMTP_PORT': str(smtp_port),
                            'VEC_SMTP_FROM': 'rrhh@example.test', 'VEC_SMTP_CA_FILE': str(ca),
                            'VEC_SMTP_MODO_TLS': 'starttls'})
        with (self.source / 'config/config.go').open('a') as stream:
            stream.write('\n'.join('"' + k + '"' for k in self.values if k.startswith('VEC_SMTP_')))
        proxy = self.root / ('comunicaciones/proxy-' + str(smtp_port) + '.json')
        proxy.parent.mkdir()
        proxy.write_text('{}')
        proxy.chmod(0o600)
        suffix = hashlib.sha256('\0'.join((str(self.root), 'vec-fixture-pg', '55531')).encode()).hexdigest()[:24]
        target = {'version': 1, 'owner': 'Codex-M', 'state': str(self.root), 'pg_container': 'vec-fixture-pg',
                  'pg_port': 55531, 'smtp_host': '127.0.0.1', 'smtp_port': smtp_port, 'http_port': http_port,
                  'container': 'vec-codexm-mailpit-' + suffix, 'network': 'vec-codexm-mailpit-red-' + suffix}
        target_file = self.root / 'comunicaciones/target.json'
        runtime.write_json(target_file, target)
        runtime.write_json(self.root / 'clon.json', {'propietario': 'Codex-M', 'estado': str(self.root),
                           'contenedor': 'vec-fixture-pg', 'puerto_pg': 55531, 'commit': 'a' * 40})
        proof = {**target, 'source_commit': 'a' * 40, 'target_sha256': runtime.digest(target_file),
                 'image_id': 'fixture-image', 'smtp_ready': True, 'smtp_scope': 'synthetic_local_sink', 'corporate_delivery': False,
                 'starttls_verified': True, 'external_recipient_rejected': True,
                 'mailpit_container': target['container'],
                 'loopback_proxies': [{'port': smtp_port, 'pid': 41, 'record': str(proxy)}]}
        profiles = self.root / 'perfiles.json'
        profiles.write_text(json.dumps({'profiles': {'usuarios_comunicaciones': proof}}))
        profiles.chmod(0o600)
        self.save()
        labels = {'vec.recorridos.owner': 'Codex-M', 'vec.recorridos.state': str(self.root)}
        network_name = target['network']
        sink = {'Image': 'fixture-image', 'State': {'Running': True},
                'Config': {'Labels': labels, 'Cmd': ['--disable-version-check', '--block-remote-css-and-fonts', '--smtp-disable-rdns', '--smtp-require-starttls', '--smtp-tls-cert', '/tls/servidor.crt', '--smtp-tls-key', '/tls/servidor.key', '--smtp-allowed-recipients', r'^[A-Za-z0-9._+\-]+@example\.test$', '--smtp', '0.0.0.0:1025', '--listen', '0.0.0.0:8025', '--database', '/data/mailpit.db', '--max', '100', '--quiet']},
                'HostConfig': {'ReadonlyRootfs': True, 'NetworkMode': network_name,
                               'PortBindings': {'1025/tcp': [{'HostIp': '127.0.0.1', 'HostPort': str(smtp_port)}],
                                                '8025/tcp': [{'HostIp': '127.0.0.1', 'HostPort': str(http_port)}]}},
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

    def test_another_clone_can_use_its_reserved_ports(self):
        resources = self.smtp_fixture(smtp_port=11029, http_port=18537)
        with patch.object(runtime, 'inspect_smtp_resource', side_effect=resources), patch.object(runtime, 'smtp_proxy_identity', return_value=41) as identity:
            environment, _ = runtime.runtime_environment(self.source, self.root, 18531, 55531)
        self.assertEqual(environment['VEC_SMTP_PORT'], '11029')
        self.assertEqual(identity.call_args.args[-1], 11029)

    def test_smtp_rejects_changed_target_or_source_before_resource_lookup(self):
        self.smtp_fixture()
        profiles = self.root / 'perfiles.json'
        original = json.loads(profiles.read_text())
        for field, value in [('state', '/other-clone'), ('source_commit', 'b' * 40), ('target_sha256', '0' * 64)]:
            data = json.loads(json.dumps(original))
            data['profiles']['usuarios_comunicaciones'][field] = value
            profiles.write_text(json.dumps(data))
            with self.subTest(field=field), patch.object(runtime, 'inspect_smtp_resource') as inspect:
                with self.assertRaises(runtime.RuntimeErrorLocal):
                    runtime.validate_smtp(self.values, self.root)
                inspect.assert_not_called()

    def test_smtp_rejects_changed_image_when_resources_are_checked(self):
        resources = self.smtp_fixture()
        profiles = self.root / 'perfiles.json'
        data = json.loads(profiles.read_text())
        data['profiles']['usuarios_comunicaciones']['image_id'] = 'other-image'
        profiles.write_text(json.dumps(data))
        with patch.object(runtime, 'inspect_smtp_resource', side_effect=resources) as inspect:
            with self.assertRaises(runtime.RuntimeErrorLocal):
                runtime.validate_smtp(self.values, self.root)
            self.assertEqual(inspect.call_count, 3)

    def test_atomic_json_write_recovers_stale_fixed_tmp_and_cleans_failure(self):
        target = self.root / 'record.json'
        old_tmp = target.with_suffix('.tmp')
        old_tmp.write_text('old interrupted attempt')
        runtime.write_json(target, {'version': 1})
        self.assertEqual(json.loads(target.read_text()), {'version': 1})
        self.assertEqual(old_tmp.read_text(), 'old interrupted attempt')
        with patch.object(runtime.os, 'replace', side_effect=OSError('fixture failure')):
            with self.assertRaises(OSError):
                runtime.write_json(target, {'version': 2})
        self.assertEqual(json.loads(target.read_text()), {'version': 1})
        self.assertEqual(list(self.root.glob('.record.json-*.tmp')), [])

    def test_static_elf_probe_rejects_an_interpreter(self):
        import struct
        path = self.root / 'fixture-elf'
        header = bytearray(64)
        header[:6] = b'\x7fELF\x02\x01'
        struct.pack_into('<Q', header, 32, 64)
        struct.pack_into('<HH', header, 54, 56, 1)
        program = bytearray(56)
        path.write_bytes(header + program)
        self.assertIsNone(runtime.elf_interpreter(path))
        struct.pack_into('<I', program, 0, 3)
        struct.pack_into('<Q', program, 8, 120)
        struct.pack_into('<Q', program, 32, 8)
        path.write_bytes(header + program + b'/loader\0')
        self.assertEqual(runtime.elf_interpreter(path), '/loader')

    def projection_fixture(self):
        root = self.root / 'runtime-interno'
        root.mkdir(mode=0o700)
        names = ['ca/ca.crt', 'tls/servidor.crt', 'tls/servidor.key']
        positive = {}
        for name in names:
            target = root / 'material' / name
            target.parent.mkdir(parents=True, exist_ok=True)
            target.write_bytes((self.material / name).read_bytes())
            positive[name] = {'source_sha256': runtime.digest(self.material / name),
                              'projected_sha256': runtime.digest(target), 'unchanged': True}
        for kind in ['documentos', 'imagenes', 'data', 'comunicaciones']:
            (root / 'rw' / kind).mkdir(parents=True, mode=0o700)
        (root / 'material/comunicaciones').mkdir(mode=0o700)
        values = {k: v.replace(str(self.material), str(root / 'material')) for k, v in self.values.items()}
        values.update(VEC_PORTAL_PROCESO='interno', VEC_PERSONAL_CATALOG_PATH=str(root / 'rw/data/personal-catalog.json'),
                      VEC_BOLSA_DATA_DIR=str(root / 'rw/data/bolsa'), VEC_BOLSA_DATA_PATH=str(root / 'rw/data/bolsa/bolsa_store.json'),
                      VEC_BOLSA_IMPORTACION_CONVOCA_CUSTODIA_DIR=str(root / 'rw/data/importaciones'))
        runtime.write_json(root / 'runtime-config.json', values)
        (root / 'runtime.env').write_text('fixture')
        (root / 'material/portal-proceso.json').write_text('{"version":1,"portal":"interno"}')
        (root / 'material/desarrollo.env').write_text('fixture')
        with (self.source / 'config/config.go').open('a') as stream:
            stream.write('\n' + '\n'.join('"' + key + '"' for key in values))
        source = self.root / ('source-' + 'a' * 40)
        contracts = {}
        for name in ['config/portal_proceso.go', 'internal/app/separacionportales/material.go',
                     'internal/app/bootstrap/material_desarrollo.go', 'internal/app/bootstrap/usuarios_preferencias_config_identidad.go',
                     'internal/app/bootstrap/documentos_montaje.go', 'internal/app/bootstrap/usuarios_imagen_montaje.go']:
            path = source / name
            path.parent.mkdir(parents=True, exist_ok=True)
            path.write_text('fixture source contract')
            contracts[name] = runtime.digest(path)
        top = {'owner': 'Codex-M', 'target': {'source_commit': 'a' * 40, 'app_port': 18531, 'pg_port': 55531},
               'files': {'material/' + name: runtime.digest(self.material / name) for name in names}}
        proof = {'operator_manifest_normalization': 'drop_runtime_interno_only',
                 'operator_manifest_sha256': hashlib.sha256(json.dumps(top, sort_keys=True, separators=(',', ':')).encode()).hexdigest(),
                 'operator_env_sha256': runtime.digest(self.config), 'positive_files': positive, 'contracts': contracts}
        sealed = {'owner': 'Codex-M', 'portal': 'interno', 'mode': 'interno', 'target': top['target'], 'source_proof': proof,
                  'files': {str(path.relative_to(root)): runtime.digest(path) for path in root.rglob('*') if path.is_file()}}
        runtime.write_json(root / 'material-manifest.json', sealed)
        top['runtime_interno'] = {'mode': 'interno', 'source_commit': 'a' * 40,
                                 'material': 'runtime-interno/material', 'config': 'runtime-interno/runtime-config.json',
                                 'manifest': 'runtime-interno/material-manifest.json', 'manifest_sha256': runtime.digest(root / 'material-manifest.json'),
                                 'rw': [{'source': 'runtime-interno/rw/' + kind, 'target': str(root / ('material/comunicaciones' if kind == 'comunicaciones' else 'rw/' + kind)), 'kind': kind}
                                        for kind in ['documentos', 'imagenes', 'data', 'comunicaciones']]}
        runtime.write_json(self.root / 'material-manifest.json', top)
        return root, values, top, sealed

    def test_projection_accepts_nominal_future_data_and_clean_internal_environment(self):
        root, _, _, _ = self.projection_fixture()
        descriptor = runtime.read_runtime_descriptor(self.root)
        self.assertEqual(len(descriptor['rw']), 4)
        with patch.dict(os.environ, {'VEC_PORTAL_PROCESO': 'externo', 'HOME': '/offline'}):
            environment, _ = runtime.runtime_environment(self.source, self.root, 18531, 55531, root)
        self.assertEqual(environment['VEC_PORTAL_PROCESO'], 'interno')
        self.assertEqual(environment['HOME'], '/home/runtime')
        self.assertEqual(environment['TMPDIR'], '/tmp')
        self.assertEqual(environment['VEC_HTTP_ADDR'], '127.0.0.1:18531')
        self.assertEqual(environment['VEC_HTTP_ALLOWED_CIDRS'], '127.0.0.1/32')
        self.assertFalse(Path(environment['VEC_PERSONAL_CATALOG_PATH']).exists())
        self.assertNotIn(str(self.material), json.dumps(descriptor))

    def test_projection_rejects_extra_offline_key_even_with_resealed_manifest(self):
        root, _, top, sealed = self.projection_fixture()
        extra = root / 'material/ca/ca.key'
        extra.write_text('offline synthetic private key')
        sealed['files']['material/ca/ca.key'] = runtime.digest(extra)
        runtime.write_json(root / 'material-manifest.json', sealed)
        top['runtime_interno']['manifest_sha256'] = runtime.digest(root / 'material-manifest.json')
        runtime.write_json(self.root / 'material-manifest.json', top)
        with self.assertRaises(runtime.RuntimeErrorLocal):
            runtime.read_runtime_descriptor(self.root)

    def test_future_data_rejects_foreign_paths_symlinks_and_missing_ro_file(self):
        root, values, _, _ = self.projection_fixture()
        for name, value in [('VEC_PERSONAL_CATALOG_PATH', str(self.root / 'offline.json')),
                            ('VEC_PERSONAL_CATALOG_PATH', str(root / 'rw/data/other.json')),
                            ('VEC_TLS_KEY_FILE', str(root / 'material/tls/missing.key'))]:
            runtime.write_json(root / 'runtime-config.json', dict(values, **{name: value}))
            with self.subTest(name=name, value=value), self.assertRaises(runtime.RuntimeErrorLocal):
                runtime.runtime_environment(self.source, self.root, 18531, 55531, root)
        runtime.write_json(root / 'runtime-config.json', values)
        (root / 'rw/data/bolsa').symlink_to(self.material, target_is_directory=True)
        with self.assertRaises(runtime.RuntimeErrorLocal):
            runtime.runtime_environment(self.source, self.root, 18531, 55531, root)

    def test_projection_rejects_source_changes_and_incomplete_rw(self):
        root, _, top, _ = self.projection_fixture()
        top['runtime_interno']['rw'].pop()
        runtime.write_json(self.root / 'material-manifest.json', top)
        with self.assertRaises(runtime.RuntimeErrorLocal):
            runtime.read_runtime_descriptor(self.root)
        # Restore only the descriptor; then change its pinned loader content.
        top['runtime_interno']['rw'].append({'source': 'runtime-interno/rw/comunicaciones',
                                            'target': str(root / 'material/comunicaciones'), 'kind': 'comunicaciones'})
        runtime.write_json(self.root / 'material-manifest.json', top)
        (self.root / ('source-' + 'a' * 40) / 'config/portal_proceso.go').write_text('changed source')
        with self.assertRaises(runtime.RuntimeErrorLocal):
            runtime.read_runtime_descriptor(self.root)

    def test_container_publication_failure_stops_only_its_reservation(self):
        from types import SimpleNamespace
        projection = {'root': str(self.root), 'material_path': str(self.material),
                      'runtime_config_path': str(self.config), 'manifest_path': str(self.root / 'material-manifest.json')}
        record = {'container_id': 'own-id', 'container_mode': 'interno', 'pid': 41}
        module = unittest.mock.Mock()
        module.ContainerError = RuntimeError
        module.start.return_value = record
        binary = self.root / 'fixture-bin'
        binary.write_text('ELF fixture')
        with patch.object(runtime, 'read_runtime_descriptor', return_value=projection), patch.object(runtime, 'validate_material', return_value='material'), patch.object(runtime, 'runtime_environment', return_value=(self.values, 'config')), patch.object(runtime, 'own_process', return_value=None), patch.object(runtime, 'elf_interpreter', return_value=None), patch.object(runtime, 'container_module', return_value=module), patch.object(runtime, 'write_json', side_effect=OSError('publish failure')):
            with self.assertRaises(OSError):
                runtime.start(self.source, binary, {'runtime_mode': 'interno', 'cgo_enabled': False, 'source_commit': 'a' * 40}, self.root, 18531, 55531)
        module.stop.assert_called_once_with(self.root)
        self.assertFalse((self.root / 'runtime-process.json').exists())

    def test_internal_start_waits_for_https_then_cleans_only_own_on_timeout(self):
        projection = {'root': str(self.root), 'material_path': str(self.material),
                      'runtime_config_path': str(self.config), 'manifest_path': str(self.root / 'material-manifest.json')}
        binary = self.root / 'fixture-bin'
        binary.write_text('ELF fixture')
        manifest = {'runtime_mode': 'interno', 'cgo_enabled': False, 'source_commit': 'a' * 40}
        for timeout in [False, True]:
            module = unittest.mock.Mock()
            module.ContainerError = RuntimeError
            module.start.return_value = {'container_id': 'own-id', 'container_mode': 'interno', 'pid': 41}
            clock = [0, 61] if timeout else [0, 0, 1]
            health = [ConnectionRefusedError(), None]
            with self.subTest(timeout=timeout), patch.object(runtime, 'read_runtime_descriptor', return_value=projection), patch.object(runtime, 'validate_material', return_value='material'), patch.object(runtime, 'runtime_environment', return_value=(self.values, 'config')), patch.object(runtime, 'own_process', return_value=None), patch.object(runtime, 'elf_interpreter', return_value=None), patch.object(runtime, 'container_module', return_value=module), patch.object(runtime.time, 'monotonic', side_effect=clock), patch.object(runtime.time, 'sleep'), patch.object(runtime, 'check_internal_https', side_effect=health) as check:
                if timeout:
                    with self.assertRaises(runtime.RuntimeErrorLocal):
                        runtime.start(self.source, binary, manifest, self.root, 18531, 55531)
                    module.stop.assert_called_once_with(self.root)
                    self.assertFalse((self.root / 'runtime-process.json').exists())
                else:
                    self.assertEqual(runtime.start(self.source, binary, manifest, self.root, 18531, 55531)['container_id'], 'own-id')
                    self.assertEqual(check.call_count, 2)
                    self.assertEqual(module.verify_record.call_count, 3)
                    module.stop.assert_not_called()
                    (self.root / 'runtime-process.json').unlink()

    def test_verify_rejects_material_config_and_source_changes_without_signal(self):
        record = {'container_mode': 'interno', 'source_commit': 'a' * 40, 'port': 18531, 'pg_port': 55531,
                  'material_sha256': 'material', 'config_sha256': 'config',
                  'runtime_config_path': str(self.config), 'runtime_manifest_path': str(self.root / 'material-manifest.json')}
        projection = {'root': str(self.root), 'runtime_config_path': str(self.config), 'manifest_path': str(self.root / 'material-manifest.json')}
        (self.root / ('source-' + 'a' * 40)).mkdir()
        for failure in ['material', 'config', 'source']:
            module = unittest.mock.Mock()
            module.ContainerError = RuntimeError
            if failure == 'source':
                module.verify_record.side_effect = RuntimeError('source changed')
            with self.subTest(failure=failure), patch.object(runtime, 'own_process', return_value=record), patch.object(runtime, 'read_runtime_descriptor', return_value=projection), patch.object(runtime, 'validate_material', return_value='other' if failure == 'material' else 'material'), patch.object(runtime, 'runtime_environment', return_value=({}, 'other' if failure == 'config' else 'config')), patch.object(runtime, 'container_module', return_value=module), patch.object(runtime, 'check_internal_https') as health:
                with self.assertRaises(runtime.RuntimeErrorLocal):
                    runtime.verify_running(self.root, 'a' * 40, 18531, 55531)
                health.assert_not_called()
                module.stop.assert_not_called()

    def test_stop_checks_exact_record_but_not_source_or_config(self):
        record = {'container_mode': 'interno', 'container_id': 'owned-id', 'instance': 'unique',
                  'pid': 41, 'source_commit': 'a' * 40, 'uid': os.getuid()}
        runtime.write_json(self.root / 'runtime-process.json', record)
        module = unittest.mock.Mock()
        module.ContainerError = RuntimeError
        module.read_record.return_value = dict(record)
        module.stop.return_value = True
        with patch.object(runtime, 'container_module', return_value=module), patch.object(runtime, 'runtime_environment', side_effect=AssertionError('not needed')), patch.object(runtime, 'validate_material', side_effect=AssertionError('not needed')):
            self.assertTrue(runtime.stop(self.root))
        self.assertFalse((self.root / 'runtime-process.json').exists())
        runtime.write_json(self.root / 'runtime-process.json', dict(record, container_id='foreign-id'))
        module.stop.reset_mock()
        with patch.object(runtime, 'container_module', return_value=module):
            with self.assertRaises(runtime.RuntimeErrorLocal):
                runtime.stop(self.root)
            module.stop.assert_not_called()

    def test_stop_recovers_removed_container_receipt_without_signalling_saved_pid(self):
        class MissingContainer(RuntimeError):
            pass
        record = {'owner': 'Codex-M', 'state': str(self.root), 'container_mode': 'interno',
                  'container_id': 'c' * 64, 'instance': 'd' * 32, 'pid': 999999,
                  'start_ticks': '123', 'source_commit': 'a' * 40, 'uid': os.getuid()}
        path = self.root / 'runtime-process.json'
        runtime.write_json(path, record)
        module = unittest.mock.Mock()
        module.ContainerError = RuntimeError
        module.DockerNotFound = MissingContainer
        module.read_record.return_value = None
        module.read_private_json.return_value = dict(record)
        module.inspect.side_effect = MissingContainer()
        with patch.object(runtime, 'container_module', return_value=module):
            self.assertTrue(runtime.stop(self.root))
        self.assertFalse(path.exists())
        module.stop.assert_not_called()
        module.verify_ownership.assert_not_called()
        runtime.write_json(path, record)
        module.read_private_json.return_value = dict(record, instance='e' * 32)
        with patch.object(runtime, 'container_module', return_value=module), self.assertRaises(runtime.RuntimeErrorLocal):
            runtime.stop(self.root)
        self.assertTrue(path.exists())
        module.stop.assert_not_called()

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
                (self.root / 'comunicaciones/target.json').unlink()
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
