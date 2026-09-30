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

    def test_status_allows_concurrent_readers_and_reports_a_writer_without_reading(self):
        repo, state = self.root / 'repo', self.root / 'state'
        repo.mkdir()
        state.mkdir(mode=0o700)
        args = ['runtime', 'status', '--repo', str(repo), '--state', str(state), '--commit', 'a' * 40,
                '--port', '18531', '--pg-port', '55531', '--mode', 'interno']
        # These are real kernel locks in a private fixture, never the live clone.
        with (state / 'runtime.lock').open('a') as reader, (state / 'runtime.lock').open('a') as writer:
            runtime.fcntl.flock(reader, runtime.fcntl.LOCK_SH | runtime.fcntl.LOCK_NB)
            def own_process(owned_state):
                self.assertEqual(owned_state, state)
                with self.assertRaises(BlockingIOError):
                    runtime.fcntl.flock(writer, runtime.fcntl.LOCK_EX | runtime.fcntl.LOCK_NB)
                return {'source_commit': 'a' * 40}
            with patch.object(runtime.sys, 'argv', args), patch.object(runtime, 'own_process', side_effect=own_process) as process, patch('builtins.print') as output:
                runtime.main()
            process.assert_called_once_with(state)
            self.assertTrue(json.loads(output.call_args.args[0])['running'])
            args[1] = 'verify'
            with patch.object(runtime.sys, 'argv', args), patch.object(runtime, 'pinned_main', return_value='a' * 40), \
                    patch.object(runtime, 'verify_running', return_value={'pid': 41}) as verify, patch('builtins.print'):
                runtime.main()
            verify.assert_called_once_with(state, 'a' * 40, 18531, 55531)
            args[1] = 'stop'
            with patch.object(runtime.sys, 'argv', args), patch.object(runtime, 'stop') as stop, \
                    self.assertRaisesRegex(runtime.RuntimeErrorLocal, 'otra operación incompatible'):
                runtime.main()
            stop.assert_not_called()
        with (state / 'runtime.lock').open('a') as writer:
            runtime.fcntl.flock(writer, runtime.fcntl.LOCK_EX | runtime.fcntl.LOCK_NB)
            for action in ['status', 'stop']:
                args[1] = action
                with self.subTest(action=action), patch.object(runtime.sys, 'argv', args), \
                        patch.object(runtime, 'own_process') as process, patch.object(runtime, 'stop') as stop, \
                        self.assertRaisesRegex(runtime.RuntimeErrorLocal, 'otra operación incompatible'):
                    runtime.main()
                process.assert_not_called()
                stop.assert_not_called()

    def test_static_elf_probe_rejects_an_interpreter(self):
        import struct
        path = self.root / 'fixture-elf'
        header = bytearray(64)
        header[:6] = b'\x7fELF\x02\x01'
        struct.pack_into('<H', header, 18, 62)
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

    def artifact_fixture(self):
        import struct
        commit = 'a' * 40
        source = self.root / ('source-' + commit)
        source.mkdir()
        (source / 'go.mod').write_text('module fixture\n')
        snapshot = {'source_commit': commit, 'source_sha256': runtime.source_digest(source)}
        runtime.write_json(self.root / ('source-manifest-' + commit + '.json'), snapshot)
        header = bytearray(64)
        header[:6] = b'\x7fELF\x02\x01'
        struct.pack_into('<H', header, 18, 62)
        struct.pack_into('<Q', header, 32, 64)
        struct.pack_into('<HH', header, 54, 56, 1)
        artifact = self.root / 'approved-artifact'
        artifact.write_bytes(header + bytearray(56) + b'approved fixture bytes')
        artifact.chmod(0o755)
        go = self.root / 'go-inspector'
        go.write_text('fixture tool, not executed')
        info = {'GoVersion': 'go1.26.6', 'Main': {'Path': 'fixture'}, 'Path': 'fixture/cmd/vec-server',
                'Settings': [{'Key': key, 'Value': value} for key, value in {
                    'CGO_ENABLED': '0', 'GOOS': 'linux', 'GOARCH': 'amd64'}.items()]}
        claim = {'path': artifact, 'sha256': runtime.digest(artifact), 'source_commit': commit}
        return source, go, artifact, info, claim

    def test_explicit_static_artifact_is_copied_exactly_without_compiling(self):
        source, go, artifact, info, claim = self.artifact_fixture()
        result = subprocess.CompletedProcess([], 0, json.dumps(info).encode(), b'')
        with patch.object(runtime.subprocess, 'run', return_value=result) as inspect, patch.object(runtime, 'git', return_value='b' * 40):
            extracted, binary, manifest = runtime.build(self.source, self.root, 'a' * 40, str(go), claim)
        self.assertEqual(extracted, source)
        self.assertEqual(binary.read_bytes(), artifact.read_bytes())
        self.assertEqual(binary.stat().st_mode & 0o777, 0o755)
        self.assertEqual(manifest['binary_sha256'], claim['sha256'])
        self.assertEqual(manifest['build_mode'], 'artifact_direction')
        self.assertEqual(manifest['artifact_direction']['source_commit'], 'a' * 40)
        self.assertEqual(manifest['artifact_direction']['source_binding'], 'operator_declared_commit')
        self.assertIsNone(manifest['build_command'])
        inspect.assert_called_once()
        self.assertEqual(inspect.call_args.args[0][1:4], ['version', '-m', '-json'])

    def test_artifact_rejects_wrong_hash_source_links_and_foreign_owner_before_mutation(self):
        _, go, artifact, _, claim = self.artifact_fixture()
        for label, changed in [('sha', dict(claim, sha256='0' * 64)), ('source', dict(claim, source_commit='b' * 40))]:
            with self.subTest(label=label), patch.object(runtime.subprocess, 'run') as inspect, self.assertRaises(runtime.RuntimeErrorLocal):
                runtime.build(self.source, self.root, 'a' * 40, str(go), changed)
            inspect.assert_not_called()
        alias = self.root / 'artifact-link'
        alias.symlink_to(artifact)
        with self.assertRaises(runtime.RuntimeErrorLocal):
            runtime.validate_artifact_claim(dict(claim, path=alias), 'a' * 40)
        with patch.object(runtime.os, 'getuid', return_value=os.getuid() + 1), self.assertRaises(runtime.RuntimeErrorLocal):
            runtime.validate_artifact_claim(claim, 'a' * 40)
        self.assertFalse((self.root / ('vec-server-' + 'a' * 40)).exists())

    def test_artifact_rejects_dynamic_or_incompatible_go_buildinfo(self):
        import struct
        _, go, artifact, info, claim = self.artifact_fixture()
        original = artifact.read_bytes()
        data = bytearray(original)
        struct.pack_into('<I', data, 64, 3)
        struct.pack_into('<Q', data, 72, 120)
        struct.pack_into('<Q', data, 96, 8)
        artifact.write_bytes(data[:120] + b'/loader\0')
        with self.assertRaises(runtime.RuntimeErrorLocal):
            runtime.validate_artifact_claim(dict(claim, sha256=runtime.digest(artifact)), 'a' * 40)
        artifact.write_bytes(original)
        for label, changed in [('cgo', dict(info, Settings=[{'Key': 'CGO_ENABLED', 'Value': '1'}])),
                               ('foreign_package', dict(info, Path='fixture/cmd/other')),
                               ('different_source', dict(info, Settings=info['Settings'] + [{'Key': 'vcs.revision', 'Value': 'b' * 40}]))]:
            result = subprocess.CompletedProcess([], 0, json.dumps(changed).encode(), b'')
            with self.subTest(label=label), patch.object(runtime.subprocess, 'run', return_value=result), self.assertRaises(runtime.RuntimeErrorLocal):
                runtime.build(self.source, self.root, 'a' * 40, str(go), claim)
        self.assertFalse((self.root / ('vec-server-' + 'a' * 40)).exists())

    def test_artifact_import_preserves_previous_binary_and_rolls_back_publication_failure(self):
        source, go, artifact, info, claim = self.artifact_fixture()
        binary = self.root / ('vec-server-' + 'a' * 40)
        old_bytes = artifact.read_bytes() + b'previous build'
        binary.write_bytes(old_bytes)
        old = {'source_commit': 'a' * 40, 'source_sha256': runtime.source_digest(source),
               'binary_sha256': runtime.digest(binary), 'cgo_enabled': False, 'elf_interpreter': None}
        runtime.write_json(self.root / 'runtime-manifest.json', old)
        result = subprocess.CompletedProcess([], 0, json.dumps(info).encode(), b'')
        original_write = runtime.write_json
        def reject_latest(path, value):
            if path.name == 'runtime-manifest.json':
                raise OSError('private publication fixture failure')
            return original_write(path, value)
        with patch.object(runtime.subprocess, 'run', return_value=result), patch.object(runtime, 'git', return_value='b' * 40), \
                patch.object(runtime, 'write_json', side_effect=reject_latest), self.assertRaises(OSError):
            runtime.build(self.source, self.root, 'a' * 40, str(go), claim)
        self.assertEqual(binary.read_bytes(), old_bytes)
        self.assertEqual(json.loads((self.root / 'runtime-manifest.json').read_text()), old)
        with patch.object(runtime.subprocess, 'run', return_value=result), patch.object(runtime, 'git', return_value='b' * 40):
            runtime.build(self.source, self.root, 'a' * 40, str(go), claim)
        self.assertEqual((self.root / ('vec-server-' + 'a' * 40 + '-' + old['binary_sha256'])).read_bytes(), old_bytes)
        self.assertEqual(binary.read_bytes(), artifact.read_bytes())

    def test_artifact_import_refuses_any_active_runtime_reservation(self):
        _, go, _, info, claim = self.artifact_fixture()
        result = subprocess.CompletedProcess([], 0, json.dumps(info).encode(), b'')
        for name in ['runtime-process.json', 'runtime-container.json', 'runtime-container-intent.json']:
            path = self.root / name
            runtime.write_json(path, {'private': 'fixture reservation'})
            with self.subTest(name=name), patch.object(runtime.subprocess, 'run', return_value=result), self.assertRaises(runtime.RuntimeErrorLocal):
                runtime.build(self.source, self.root, 'a' * 40, str(go), claim)
            self.assertFalse((self.root / ('vec-server-' + 'a' * 40)).exists())
            self.assertTrue(path.exists())
            path.unlink()

    def test_wrong_artifact_source_is_rejected_before_restart_stop(self):
        _, go, artifact, _, claim = self.artifact_fixture()
        args = ['runtime', 'restart', '--repo', str(self.source), '--state', str(self.root), '--commit', 'a' * 40,
                '--port', '18531', '--pg-port', '55531', '--mode', 'interno', '--go', str(go),
                '--artifact', str(artifact), '--artifact-sha256', claim['sha256'], '--artifact-source', 'b' * 40]
        with patch.object(runtime.sys, 'argv', args), patch.object(runtime, 'pinned_main', return_value='a' * 40), \
                patch.object(runtime, 'stop') as stop, self.assertRaises(runtime.RuntimeErrorLocal):
            runtime.main()
        stop.assert_not_called()

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
        for name in runtime.projection_module().APPROVED_CONTRACTS:
            path = source / name
            path.parent.mkdir(parents=True, exist_ok=True)
            path.write_bytes((Path(__file__).resolve().parents[2] / name).read_bytes())
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

    def test_legacy_without_public_declarations_remains_valid_even_when_git_contains_catalogs(self):
        _, _, top, _ = self.projection_fixture()
        self.assertNotIn('public_catalogs', top)
        original = json.loads(self.config.read_text())
        self.assertNotIn('VEC_PERSONAL_ORGANIZACION_SOURCE_PATH', original)
        self.assertNotIn('VEC_RPT_CATALOGO_PATH', original)
        source = self.root / ('source-' + 'a' * 40)
        for name in ['data/catalogos/estructura-organizativa/v1.rpt-publica.json', 'data/catalogos/rpt/v1.rpt-2026.json']:
            path = source / name
            path.parent.mkdir(mode=0o700, parents=True, exist_ok=True)
            path.write_bytes((Path(__file__).resolve().parents[2] / name).read_bytes())
        self.assertEqual(runtime.read_runtime_descriptor(self.root)['mode'], 'interno')

    def test_original_partial_file_or_metadata_declaration_cannot_be_erased_as_legacy(self):
        root, _, top, sealed = self.projection_fixture()
        for signal in ['file', 'metadata']:
            parent = json.loads(json.dumps(top))
            if signal == 'file':
                parent['files']['material/catalogos/organizacion-publica.json'] = '0' * 64
            else:
                parent['public_catalogs'] = {}
            normalized = {key: value for key, value in parent.items() if key != 'runtime_interno'}
            sealed['source_proof']['operator_manifest_sha256'] = hashlib.sha256(json.dumps(normalized, sort_keys=True, separators=(',', ':')).encode()).hexdigest()
            runtime.write_json(root / 'material-manifest.json', sealed)
            parent['runtime_interno']['manifest_sha256'] = runtime.digest(root / 'material-manifest.json')
            runtime.write_json(self.root / 'material-manifest.json', parent)
            with self.subTest(signal=signal), self.assertRaisesRegex(runtime.RuntimeErrorLocal, 'fuentes públicas aprobadas completas'):
                runtime.read_runtime_descriptor(self.root)

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

    def test_public_catalogs_require_canonical_bytes_and_exact_projected_selectors(self):
        from types import SimpleNamespace
        root, values, top, sealed = self.projection_fixture()
        public = {
            'catalogos/organizacion-publica.json': {
                'source_path': 'data/catalogos/estructura-organizativa/v1.rpt-publica.json',
                'sha256': '0e52d878526d6a5e7ee4ab6f525ef92a70144aef665f0b031fca6051564e054c'},
            'catalogos/rpt-publica.json': {
                'source_path': 'data/catalogos/rpt/v1.rpt-2026.json',
                'sha256': 'b0685beb5c02b8a30d5e0d6d3d9bceca11ddf76ad4987f4bcb1aa60ac7ebe9a8'},
        }
        source = self.root / ('source-' + 'a' * 40)
        for name, evidence in public.items():
            data = (Path(__file__).resolve().parents[2] / evidence['source_path']).read_bytes()
            self.assertEqual(hashlib.sha256(data).hexdigest(), evidence['sha256'])
            for path in [self.material / name, root / 'material' / name, source / evidence['source_path']]:
                path.parent.mkdir(mode=0o700, parents=True, exist_ok=True)
                path.write_bytes(data)
                path.chmod(0o600)
            top['files']['material/' + name] = evidence['sha256']
            sealed['files']['material/' + name] = evidence['sha256']
            sealed['source_proof']['positive_files'][name] = {'source_sha256': evidence['sha256'], 'projected_sha256': evidence['sha256'], 'unchanged': True}
        selectors = {'VEC_PERSONAL_ORGANIZACION_SOURCE_PATH': 'catalogos/organizacion-publica.json', 'VEC_RPT_CATALOGO_PATH': 'catalogos/rpt-publica.json'}
        self.values.update({key: str(self.material / name) for key, name in selectors.items()})
        self.save()
        values.update({key: str(root / 'material' / name) for key, name in selectors.items()})
        def seal():
            runtime.write_json(root / 'runtime-config.json', values)
            sealed['files']['runtime-config.json'] = runtime.digest(root / 'runtime-config.json')
            sealed['source_proof']['operator_env_sha256'] = runtime.digest(self.config)
            top['files']['runtime-config.json'] = runtime.digest(self.config)
            parent = {key: value for key, value in top.items() if key != 'runtime_interno'}
            sealed['source_proof']['operator_manifest_sha256'] = hashlib.sha256(json.dumps(parent, sort_keys=True, separators=(',', ':')).encode()).hexdigest()
            runtime.write_json(root / 'material-manifest.json', sealed)
            top['runtime_interno']['manifest_sha256'] = runtime.digest(root / 'material-manifest.json')
            runtime.write_json(self.root / 'material-manifest.json', top)
        projection = SimpleNamespace(APPROVED_CONTRACTS=runtime.projection_module().APPROVED_CONTRACTS, APPROVED_PUBLIC_SOURCES=public)
        seal()
        with patch.object(runtime, 'projection_module', return_value=projection):
            self.assertEqual(runtime.read_runtime_descriptor(self.root)['root'], str(root))
            # Removing every projected signal cannot erase the original opt-in.
            for key in selectors:
                values.pop(key)
            for name in public:
                sealed['files'].pop('material/' + name)
                sealed['source_proof']['positive_files'].pop(name)
                (root / 'material' / name).unlink()
            seal()
            with self.assertRaisesRegex(runtime.RuntimeErrorLocal, 'fuentes públicas aprobadas completas'):
                runtime.read_runtime_descriptor(self.root)
            for key, name in selectors.items():
                values[key] = str(root / 'material' / name)
            for name, evidence in public.items():
                (root / 'material' / name).write_bytes((self.material / name).read_bytes())
                sealed['files']['material/' + name] = evidence['sha256']
                sealed['source_proof']['positive_files'][name] = {'source_sha256': evidence['sha256'], 'projected_sha256': evidence['sha256'], 'unchanged': True}
            for key, value in list(self.values.items()):
                if key not in selectors:
                    continue
                self.values[key] = ''
                self.save()
                seal()
                with self.subTest(original_empty=key), self.assertRaisesRegex(runtime.RuntimeErrorLocal, 'fuentes públicas aprobadas completas'):
                    runtime.read_runtime_descriptor(self.root)
                self.values[key] = str(self.material / selectors[key])
            self.values['VEC_RPT_CATALOGO_PATH'] = str(self.material / 'identidad/identidad.json')
            self.save()
            seal()
            with self.assertRaisesRegex(runtime.RuntimeErrorLocal, 'fuentes públicas aprobadas completas'):
                runtime.read_runtime_descriptor(self.root)
            self.values['VEC_RPT_CATALOGO_PATH'] = str(self.material / 'catalogos/rpt-publica.json')
            self.save()
            original = values['VEC_RPT_CATALOGO_PATH']
            values['VEC_RPT_CATALOGO_PATH'] = str(root / 'material/identidad/identidad.json')
            seal()
            with self.assertRaisesRegex(runtime.RuntimeErrorLocal, 'fuentes públicas aprobadas completas'):
                runtime.read_runtime_descriptor(self.root)
            values['VEC_RPT_CATALOGO_PATH'] = original
            name = 'catalogos/rpt-publica.json'
            for path in [self.material / name, root / 'material' / name]:
                path.write_bytes(path.read_bytes() + b'\n')
            altered = runtime.digest(self.material / name)
            top['files']['material/' + name] = altered
            sealed['files']['material/' + name] = altered
            sealed['source_proof']['positive_files'][name] = {'source_sha256': altered, 'projected_sha256': altered, 'unchanged': True}
            seal()
            with self.assertRaisesRegex(runtime.RuntimeErrorLocal, 'copia Git aprobada'):
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

    def test_real_projector_contracts_are_consumed_and_unapproved_proofs_are_denied(self):
        from urllib.parse import urlencode
        projection = runtime.projection_module()
        repo = self.root / 'contract-repo'
        repo.mkdir(mode=0o700)
        git_home = self.root / 'git-home'
        git_home.mkdir(mode=0o700)
        git_env = {'PATH': '/usr/bin:/bin', 'HOME': str(git_home), 'GIT_CONFIG_NOSYSTEM': '1',
                   'GIT_CONFIG_GLOBAL': '/dev/null', 'GIT_CONFIG_SYSTEM': '/dev/null'}
        public_sources = getattr(projection, 'APPROVED_PUBLIC_SOURCES', {})
        source_paths = set(projection.APPROVED_CONTRACTS) | {entry['source_path'] for entry in public_sources.values()}
        for name in source_paths:
            path = repo / name
            path.parent.mkdir(mode=0o700, parents=True, exist_ok=True)
            path.write_bytes((Path(__file__).resolve().parents[2] / name).read_bytes())
        def git(*args):
            return subprocess.run(['git', '-C', str(repo), *args], env=git_env,
                                  capture_output=True, text=True, check=True).stdout.strip()
        git('init', '-q')
        git('add', '.')
        git('-c', 'user.name=fixture', '-c', 'user.email=fixture@example.test', 'commit', '-qm', 'fixture contracts')
        commit = git('rev-parse', 'HEAD')
        source = self.root / ('source-' + commit)
        for name in source_paths:
            path = source / name
            path.parent.mkdir(mode=0o700, parents=True, exist_ok=True)
            path.write_bytes((repo / name).read_bytes())
        for name in projection.REQUIRED:
            path = self.material / name
            path.parent.mkdir(mode=0o700, parents=True, exist_ok=True)
            if name.endswith('.json'):
                value = {'version': 1, 'autoridad': 'no_autoritativo'}
                if name == 'identidad/usuarios-preferencias-interna.json':
                    value.update(superficie='interna_corporativa', cuentas=[{'cuenta_ref': 'fixture', 'perfil_ref': 'fixture'}])
                runtime.write_json(path, value)
            else:
                path.write_bytes(('synthetic fixture ' + name).encode())
                path.chmod(0o600)
        for name, evidence in public_sources.items():
            path = self.material / name
            path.parent.mkdir(mode=0o700, parents=True, exist_ok=True)
            path.write_bytes((repo / evidence['source_path']).read_bytes())
            path.chmod(0o600)
        self.values.update({key: str(self.material / name) for key, name in getattr(projection, 'PUBLIC_ENV_FILES', {}).items()})
        dsn = 'postgres://fixture:dummy@127.0.0.1:55531/postgres?' + urlencode({
            'sslmode': 'verify-full', 'sslrootcert': str(self.material / 'pg/ca.crt')})
        self.values.update(VEC_HTTP_ADDR='127.0.0.1:18531', VEC_BOLSA_IMPORTACION_CONVOCA_DATABASE_URL=dsn)
        self.values.update({name: dsn for name in self.values if name.endswith('_DATABASE_URL')})
        self.save()
        top = {'owner': 'Codex-M', 'target': {'source_commit': commit, 'pg_port': 55531, 'app_port': 18531},
               'files': {str(path.relative_to(self.root)): runtime.digest(path) for path in self.material.rglob('*') if path.is_file()}}
        top['files']['runtime-config.json'] = runtime.digest(self.config)
        runtime.write_json(self.root / 'material-manifest.json', top)
        marker = {'propietario': 'Codex-M', 'estado': str(self.root), 'contenedor': 'vec-fixture', 'puerto_pg': 55531, 'commit': commit}
        for name in ['clon.json', 'DB_READY.json']:
            runtime.write_json(self.root / name, marker)
        descriptor = projection.provision(repo, 'vec-fixture', self.root, self.material, 55531, source_context={'source_ref': commit})
        top['runtime_interno'] = descriptor
        runtime.write_json(self.root / 'material-manifest.json', top)
        root = self.root / 'runtime-interno'
        sealed = json.loads((root / 'material-manifest.json').read_text())
        self.assertEqual(len(sealed['source_proof']['contracts']), 9)
        self.assertEqual(runtime.read_runtime_descriptor(self.root)['runtime_config_path'], str(root / 'runtime-config.json'))
        original = sealed['source_proof']['contracts']
        changed = dict(original, **{'config/postgresql_importacion_convoca.go': '0' * 64})
        extra = dict(original, **{'config/unapproved.go': 'f' * 64})
        missing = {name: sha for name, sha in original.items() if name != 'config/postgresql_importacion_convoca.go'}
        for label, contracts in [('eight', missing), ('ten', extra), ('unapproved_hash', changed)]:
            with self.subTest(label=label):
                sealed['source_proof']['contracts'] = contracts
                runtime.write_json(root / 'material-manifest.json', sealed)
                top['runtime_interno']['manifest_sha256'] = runtime.digest(root / 'material-manifest.json')
                runtime.write_json(self.root / 'material-manifest.json', top)
                with self.assertRaisesRegex(runtime.RuntimeErrorLocal, 'contratos de la fuente fijada'):
                    runtime.read_runtime_descriptor(self.root)
        # Matching an altered local file is insufficient without its approved pin.
        altered = source / 'config/postgresql_importacion_convoca.go'
        altered.write_text('unapproved source contract')
        sealed['source_proof']['contracts'] = dict(original, **{'config/postgresql_importacion_convoca.go': runtime.digest(altered)})
        runtime.write_json(root / 'material-manifest.json', sealed)
        top['runtime_interno']['manifest_sha256'] = runtime.digest(root / 'material-manifest.json')
        runtime.write_json(self.root / 'material-manifest.json', top)
        with self.assertRaisesRegex(runtime.RuntimeErrorLocal, 'contratos de la fuente fijada'):
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
        with patch.object(runtime, 'read_runtime_descriptor', return_value=projection), patch.object(runtime, 'validate_material', return_value='material'), patch.object(runtime, 'runtime_environment', return_value=(self.values, 'config')), patch.object(runtime, 'own_process', return_value=None), patch.object(runtime, 'elf_interpreter', return_value=None), patch.object(runtime, 'container_module', return_value=module), patch.object(runtime.socket, 'socket'), patch.object(runtime, 'write_json', side_effect=OSError('publish failure')):
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
            with self.subTest(timeout=timeout), patch.object(runtime, 'read_runtime_descriptor', return_value=projection), patch.object(runtime, 'validate_material', return_value='material'), patch.object(runtime, 'runtime_environment', return_value=(self.values, 'config')), patch.object(runtime, 'own_process', return_value=None), patch.object(runtime, 'elf_interpreter', return_value=None), patch.object(runtime, 'container_module', return_value=module), patch.object(runtime.socket, 'socket'), patch.object(runtime.time, 'monotonic', side_effect=clock), patch.object(runtime.time, 'sleep'), patch.object(runtime, 'check_internal_https', side_effect=health) as check:
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
