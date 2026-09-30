#!/usr/bin/env python3
"""Local dummy fixtures; Docker and process signals are always mocked."""

import copy
import json
import os
from pathlib import Path
import struct
import subprocess
import tempfile
import unittest
from unittest.mock import patch

import clon_runtime_container as runtime


class ContainerBoundaryTests(unittest.TestCase):
    def setUp(self):
        self.temporary = tempfile.TemporaryDirectory()
        self.addCleanup(self.temporary.cleanup)
        self.state = Path(self.temporary.name)
        self.commit = 'a' * 40
        self.source = self.state / ('source-' + self.commit)
        self.source.mkdir()
        (self.source / 'source.go').write_text('package fixture\n')
        self.binary = self.state / ('vec-server-' + self.commit)
        self.elf = bytearray(64)
        self.elf[:6] = b'\x7fELF\x02\x01'
        struct.pack_into('<Q', self.elf, 32, 64)
        struct.pack_into('<HH', self.elf, 54, 56, 0)
        self.binary.write_bytes(self.elf)
        self.manifest = {'source_commit': self.commit, 'source_sha256': runtime.source_sha(self.source),
                         'binary_sha256': runtime.sha(self.binary), 'cgo_enabled': False, 'elf_interpreter': None}
        self.root = self.state / 'runtime-interno'
        (self.root / 'material/ca').mkdir(parents=True)
        (self.root / 'material/ca/ca.crt').write_text('dummy public CA')
        (self.root / 'material/server.key').write_text('dummy server key')
        self.environment = {'HOME': runtime.HOME, 'TMPDIR': '/tmp', 'TZ': 'UTC', 'PATH': '/usr/bin:/bin',
                            'VEC_HTTP_ADDR': '127.0.0.1:19443', 'VEC_HTTP_ALLOWED_CIDRS': '127.0.0.1/32'}
        self.envfile = self.root / 'runtime.env'
        self.envfile.write_text(''.join(f'{key}={value}\n' for key, value in sorted(self.environment.items())))
        self.envfile.chmod(0o600)
        (self.root / 'runtime-config.json').write_text(json.dumps(self.environment))
        (self.root / 'material-manifest.json').write_text(json.dumps({'portal': 'interno'}))
        self.projection = {'root': str(self.root), 'mode': 'interno', 'portal': 'interno',
                           'ro': [{'source': str(path), 'target': str(path)} for path in
                                  (self.root / 'material', self.envfile, self.root / 'runtime-config.json',
                                   self.root / 'material-manifest.json')],
                           'rw': []}
        for kind in sorted(runtime.RW_KINDS):
            path = self.root / 'rw' / kind
            path.mkdir(parents=True)
            target = self.root / 'material/comunicaciones' if kind == 'comunicaciones' else path
            self.projection['rw'].append({'source': str(path), 'target': str(target), 'kind': kind})
        self.mounts = self.validate()
        self.container_id = 'b' * 64
        self.image_id = 'sha256:' + 'c' * 64
        self.identity = {'pid': 43210, 'start_ticks': '456', 'exe': str(self.binary), 'uid': os.getuid(),
                         'binary_sha256': self.manifest['binary_sha256']}
        self.record = dict(self.identity, container_id=self.container_id, image_id=self.image_id,
                           source=str(self.source), binary=str(self.binary), source_commit=self.commit,
                           manifest=self.manifest, projection=self.projection, mounts=self.mounts, instance='d' * 32,
                           owner=runtime.OWNER, state=str(self.state), gid=os.getgid(), container_mode='interno',
                           name=runtime.container_name(self.state, self.commit),
                           projection_sha256=runtime.projection_sha(self.projection),
                           env_sha256=runtime.object_sha(sorted(f'{key}={value}' for key, value in self.environment.items())))
        self.proof = self.make_proof()

    def validate(self):
        return runtime.validate_inputs(self.state, self.source, self.binary, self.manifest, self.projection)

    def make_proof(self):
        return {'Id': self.container_id, 'Image': self.image_id, 'Path': str(self.binary), 'Args': [],
                'Config': {'Image': self.image_id, 'User': f'{os.getuid()}:{os.getgid()}',
                           'WorkingDir': str(self.source), 'Entrypoint': [str(self.binary)], 'Cmd': None,
                           'Env': [f'{key}={value}' for key, value in self.environment.items()],
                           'Labels': runtime.expected_labels(self.state, self.record)},
                'HostConfig': {'ReadonlyRootfs': True, 'Privileged': False, 'NetworkMode': 'host', 'PidMode': '',
                               'UsernsMode': '', 'AutoRemove': False,
                               'IpcMode': 'private', 'Init': False, 'CapDrop': ['ALL'], 'CapAdd': None,
                               'SecurityOpt': ['no-new-privileges'], 'Devices': [], 'DeviceRequests': [],
                               'VolumesFrom': None, 'RestartPolicy': {'Name': 'no'}, 'PidsLimit': 64,
                               'Memory': 1073741824, 'NanoCpus': 2000000000,
                               'Ulimits': [{'Name': 'nofile', 'Hard': 1024, 'Soft': 1024},
                                           {'Name': 'fsize', 'Hard': 67108864, 'Soft': 67108864}],
                               'Tmpfs': {'/tmp': 'rw,noexec,nosuid,nodev,size=16m,mode=1777'}},
                'Mounts': [{'Type': 'bind', 'Source': mount['source'], 'Destination': mount['target'],
                            'RW': mount['rw'], 'Propagation': 'rprivate'} for mount in self.mounts],
                'State': {'Running': True, 'Pid': self.identity['pid']}}

    def test_expected_projection_preserves_absolute_source_and_binary_readonly(self):
        for path in (self.source, self.binary):
            self.assertIn({'source': str(path), 'target': str(path), 'rw': False}, self.mounts)
        self.assertNotIn(str(self.state), [mount['source'] for mount in self.mounts])

    def test_docker_error_is_private_0600_and_public_exception_constant(self):
        secret_marker = 'fixture-private-error-not-for-public-output'
        result = subprocess.CompletedProcess(['docker'], 23, stdout='private inspect output', stderr=secret_marker)
        with patch.object(runtime.subprocess, 'run', return_value=result), \
                patch.object(runtime, '_diagnostic_phase', 'scratch_image'), \
                patch.object(runtime, '_diagnostic_attempt', 'e' * 32):
            with self.assertRaises(runtime.ContainerError) as rejected:
                runtime.docker(self.state, 'build', '--network=none')
        self.assertEqual(str(rejected.exception), 'Local Docker operation failed; no secret-bearing output is displayed.')
        self.assertNotIn(secret_marker, str(rejected.exception))
        path = self.state / 'runtime-container-diagnostic.json'
        self.assertEqual(path.stat().st_mode & 0o777, 0o600)
        record = json.loads(path.read_text())
        self.assertEqual(record['stage'], 'scratch_image')
        self.assertEqual(record['cli'], {'operation': 'build', 'exit_code': 23, 'stderr': secret_marker})
        self.assertNotIn('stdout', record['cli'])
        self.assertNotIn('environment', record)

    def test_input_failure_records_exact_stage_without_creating_docker_object(self):
        invalid = dict(self.manifest, cgo_enabled=True)
        with patch.object(runtime, 'docker') as docker, self.assertRaises(runtime.ContainerError):
            runtime.start(self.state, self.source, self.binary, self.environment, invalid, self.projection, 19443, 55531)
        docker.assert_not_called()
        record = json.loads((self.state / 'runtime-container-diagnostic.json').read_text())
        self.assertEqual(record['stage'], 'preflight_inputs')
        self.assertEqual(record['guard'], 'Runtime static binary or source proof does not match.')

    def test_diagnostic_cannot_overwrite_symlink_or_mask_original_error(self):
        foreign = self.state / 'offline-private-marker'
        foreign.write_text('unchanged')
        (self.state / 'runtime-container-diagnostic.json').symlink_to(foreign)
        result = subprocess.CompletedProcess(['docker'], 1, stdout='', stderr='fixture private error')
        with patch.object(runtime.subprocess, 'run', return_value=result), self.assertRaises(runtime.ContainerError):
            runtime.docker(self.state, 'build')
        self.assertEqual(foreign.read_text(), 'unchanged')

    def test_failed_own_process_logs_remain_private_before_immutable_removal(self):
        proof = copy.deepcopy(self.proof)
        proof['State'] = {'Running': False, 'Pid': 0, 'ExitCode': 1, 'Error': ''}
        secret_marker = 'private fixture bootstrap error'
        def fake(state, *args):
            self.assertEqual(args[-1], self.container_id)
            return secret_marker if args[0] == 'logs' else ''
        with patch.object(runtime, 'inspect', return_value=proof), patch.object(runtime, 'docker', side_effect=fake) as docker, \
                patch.object(runtime, '_diagnostic_attempt', 'f' * 32), patch.object(runtime, '_diagnostic_phase', 'verify_live_container'):
            runtime.diagnose(self.state, 'Recorded container is not the current live process.')
            runtime._remove_started(self.state, self.record)
        self.assertEqual([call.args[1] for call in docker.call_args_list], ['logs', 'rm'])
        diagnostic = json.loads((self.state / 'runtime-container-diagnostic.json').read_text())
        self.assertEqual(diagnostic['stage'], 'verify_live_container')
        self.assertEqual(diagnostic['guard'], 'Recorded container is not the current live process.')
        self.assertEqual(diagnostic['process']['exit_code'], 1)
        self.assertFalse(diagnostic['process']['running'])
        self.assertEqual(diagnostic['process']['observed_pid'], 0)
        log = Path(diagnostic['process']['private_log'])
        self.assertEqual(log.read_text(), secret_marker)
        self.assertEqual(log.stat().st_mode & 0o777, 0o600)
        self.assertNotIn(secret_marker, json.dumps(diagnostic))

    def test_failed_process_capture_never_reads_foreign_container_log(self):
        proof = copy.deepcopy(self.proof)
        proof['Config']['Labels'][runtime.PREFIX + 'owner'] = 'foreign'
        with patch.object(runtime, 'inspect', return_value=proof), patch.object(runtime, 'docker') as docker, self.assertRaises(runtime.ContainerError):
            runtime._remove_started(self.state, self.record)
        docker.assert_not_called()
        self.assertEqual(list(self.state.glob('runtime-container-failed-*.log')), [])

    def test_extra_ro_mount_and_state_parent_are_rejected(self):
        for path in (self.state, self.root, self.state / 'material'):
            with self.subTest(path=path):
                projection = copy.deepcopy(self.projection)
                projection['ro'].append({'source': str(path), 'target': str(path)})
                with self.assertRaises(runtime.ContainerError):
                    runtime.validate_inputs(self.state, self.source, self.binary, self.manifest, projection)

    def test_rw_allowlist_cannot_change_source_target_or_kind(self):
        for field, value in [('source', str(self.state)), ('target', '/etc'), ('kind', 'backups')]:
            projection = copy.deepcopy(self.projection)
            projection['rw'][0][field] = value
            with self.subTest(field=field), self.assertRaises(runtime.ContainerError):
                runtime.validate_inputs(self.state, self.source, self.binary, self.manifest, projection)

    def test_symlink_and_hardlinked_projection_are_rejected(self):
        foreign = self.state / 'offline.key'
        foreign.write_text('dummy')
        leaked = self.root / 'material/leaked.key'
        leaked.symlink_to(foreign)
        with self.assertRaises(runtime.ContainerError):
            self.validate()
        leaked.unlink()
        os.link(foreign, leaked)
        with self.assertRaises(runtime.ContainerError):
            self.validate()

    def test_no_client_ca_private_key_p12_or_other_portal_secrets(self):
        for name, contents in [('mtls/cliente.key', 'dummy'), ('ca/ca.key', 'dummy'),
                               ('ca/signing.pem', '-----BEGIN PRIVATE KEY-----'), ('cliente.p12', 'dummy'),
                               ('externo/server.key', 'dummy'), ('admin/runtime.json', '{}'),
                               ('backup/dump.sql', 'dummy')]:
            path = self.root / 'material' / name
            path.parent.mkdir(parents=True, exist_ok=True)
            path.write_text(contents)
            with self.subTest(name=name), self.assertRaises(runtime.ContainerError):
                self.validate()
            path.unlink()

    def test_material_marker_and_projection_mode_are_independent_controls(self):
        (self.root / 'material-manifest.json').write_text(json.dumps({'portal': 'externo'}))
        with self.assertRaises(runtime.ContainerError):
            self.validate()
        (self.root / 'material-manifest.json').write_text(json.dumps({'portal': 'interno'}))
        self.projection['mode'] = 'externo'
        with self.assertRaises(runtime.ContainerError):
            self.validate()

    def test_dynamic_elf_rejected_even_with_static_manifest(self):
        header = self.elf + bytearray(56)
        struct.pack_into('<HH', header, 54, 56, 1)
        struct.pack_into('<I', header, 64, 3)
        self.binary.write_bytes(header)
        self.manifest['binary_sha256'] = runtime.sha(self.binary)
        with self.assertRaises(runtime.ContainerError):
            self.validate()

    def test_changed_source_or_binary_is_rejected(self):
        (self.source / 'source.go').write_text('package changed\n')
        with self.assertRaises(runtime.ContainerError):
            self.validate()
        self.manifest['source_sha256'] = runtime.source_sha(self.source)
        self.binary.write_bytes(self.elf + b'changed')
        with self.assertRaises(runtime.ContainerError):
            self.validate()

    def test_verify_live_process_joins_docker_id_and_host_identity(self):
        with patch.object(runtime, 'inspect', return_value=self.proof), \
                patch.object(runtime, 'process_identity', return_value=self.identity):
            self.assertEqual(runtime.verify_record(self.state, self.record), self.record)

    def test_foreign_pid_uid_exe_or_start_ticks_never_receive_signal(self):
        for key, value in [('pid', 43211), ('uid', 0), ('exe', '/foreign/bin'), ('start_ticks', '457'),
                           ('binary_sha256', 'f' * 64)]:
            identity = dict(self.identity, **{key: value})
            with self.subTest(key=key), patch.object(runtime, 'inspect', return_value=self.proof), \
                    patch.object(runtime, 'process_identity', return_value=identity), \
                    patch.object(runtime.os, 'pidfd_open') as pidfd, \
                    patch.object(runtime.signal, 'pidfd_send_signal') as send:
                with self.assertRaises(runtime.ContainerError):
                    runtime._terminate(self.state, self.record)
                pidfd.assert_not_called()
                send.assert_not_called()

    def test_unapproved_mount_rejected_before_signalling(self):
        proof = copy.deepcopy(self.proof)
        proof['Mounts'].append({'Type': 'bind', 'Source': '/foreign', 'Destination': '/foreign',
                               'RW': False, 'Propagation': 'rprivate'})
        with patch.object(runtime, 'inspect', return_value=proof), patch.object(runtime.os, 'pidfd_open') as pidfd:
            with self.assertRaises(runtime.ContainerError):
                runtime._terminate(self.state, self.record)
            pidfd.assert_not_called()

    def test_changed_container_policy_or_foreign_labels_are_rejected(self):
        changes = [('Config', 'User', '0:0'), ('Config', 'WorkingDir', '/foreign'),
                   ('Config', 'Entrypoint', ['/bin/sh']), ('Config', 'Image', 'foreign:latest'),
                   ('HostConfig', 'ReadonlyRootfs', False), ('HostConfig', 'NetworkMode', 'bridge'),
                   ('HostConfig', 'PidMode', 'host'), ('HostConfig', 'Init', True),
                   ('HostConfig', 'SecurityOpt', ['seccomp=unconfined']), ('HostConfig', 'CapAdd', ['SYS_ADMIN']),
                   ('HostConfig', 'PidsLimit', 0), ('HostConfig', 'Memory', 0),
                   ('HostConfig', 'VolumesFrom', ['foreign'])]
        for group, key, value in changes:
            proof = copy.deepcopy(self.proof)
            proof[group][key] = value
            with self.subTest(key=key), self.assertRaises(runtime.ContainerError):
                runtime.verify_configuration(self.state, self.record, proof)
        self.proof['Config']['Labels'][runtime.PREFIX + 'owner'] = 'foreign'
        with self.assertRaises(runtime.ContainerError):
            runtime.verify_configuration(self.state, self.record, self.proof)

    def test_ro_material_change_invalidates_record(self):
        (self.root / 'material/server.key').write_text('changed dummy')
        with patch.object(runtime, 'inspect') as inspect, self.assertRaises(runtime.ContainerError):
            runtime.verify_record(self.state, self.record)
        inspect.assert_not_called()

    def test_pidfd_identity_is_rechecked_after_open(self):
        with patch.object(runtime, 'verify_ownership', side_effect=[self.record, runtime.ContainerError('changed')]), \
                patch.object(runtime.os, 'pidfd_open', return_value=99), \
                patch.object(runtime.os, 'close') as close, \
                patch.object(runtime.signal, 'pidfd_send_signal') as send:
            with self.assertRaises(runtime.ContainerError):
                runtime._terminate(self.state, self.record)
            send.assert_not_called()
            close.assert_called_once_with(99)

    def test_kernel_status_requires_private_pid_namespace_seccomp_nnp_and_zero_caps(self):
        status = {'Uid': ' '.join([str(os.getuid())] * 4), 'Gid': ' '.join([str(os.getgid())] * 4),
                  'NSpid': '43210 1', 'Seccomp': '2', 'NoNewPrivs': '1'}
        for key in ('CapInh', 'CapPrm', 'CapEff', 'CapBnd', 'CapAmb'):
            status[key] = '0000000000000000'
        self.assertTrue(runtime.isolated_process_status(status))
        for key, value in [('Uid', '0 0 0 0'), ('Gid', '0 0 0 0'), ('NSpid', '43210'),
                           ('NSpid', '43210 2'), ('Seccomp', '0'), ('NoNewPrivs', '0'), ('CapEff', '00000001')]:
            with self.subTest(key=key):
                self.assertFalse(runtime.isolated_process_status(dict(status, **{key: value})))

    def test_stop_removes_only_immutable_owned_id(self):
        runtime.private_json(self.state / 'runtime-container.json', self.record)
        with patch.object(runtime, 'inspect', return_value=self.proof), \
                patch.object(runtime, '_terminate') as terminate, patch.object(runtime, 'docker') as docker:
            self.assertTrue(runtime.stop(self.state))
        terminate.assert_called_once_with(self.state, self.record)
        docker.assert_called_once_with(self.state, 'rm', self.container_id)
        self.assertFalse((self.state / 'runtime-container.json').exists())

    def test_foreign_container_record_cannot_stop_or_remove(self):
        runtime.private_json(self.state / 'runtime-container.json', self.record)
        self.proof['Config']['Labels'][runtime.PREFIX + 'state'] = '/foreign'
        with patch.object(runtime, 'inspect', return_value=self.proof), \
                patch.object(runtime, '_terminate') as terminate, patch.object(runtime, 'docker') as docker:
            with self.assertRaises(runtime.ContainerError):
                runtime.stop(self.state)
        terminate.assert_not_called()
        docker.assert_not_called()

    def test_absent_removed_container_archives_receipt_without_pid_signal(self):
        runtime.private_json(self.state / 'runtime-container.json', self.record)
        with patch.object(runtime, 'inspect', side_effect=runtime.DockerNotFound('removed')), \
                patch.object(runtime, 'process_identity') as identity, \
                patch.object(runtime, '_terminate') as terminate, patch.object(runtime, 'docker') as docker:
            self.assertTrue(runtime.stop(self.state))
        identity.assert_not_called()
        terminate.assert_not_called()
        docker.assert_not_called()
        self.assertFalse((self.state / 'runtime-container.json').exists())
        self.assertEqual(len(list(self.state.glob('runtime-container-stopped-*.json'))), 1)

    def test_daemon_error_preserves_receipt_and_never_signals_pid(self):
        runtime.private_json(self.state / 'runtime-container.json', self.record)
        with patch.object(runtime, 'inspect', side_effect=runtime.ContainerError('daemon unavailable')), \
                patch.object(runtime, '_terminate') as terminate:
            with self.assertRaises(runtime.ContainerError):
                runtime.stop(self.state)
        terminate.assert_not_called()
        self.assertTrue((self.state / 'runtime-container.json').exists())

    def fake_start_docker(self, calls):
        def invoke(state, *args):
            calls.append(args)
            if args[0] == 'create':
                # These labels include the actual mount and image proof.
                return self.container_id
            if args[0] == 'start':
                return self.container_id
            if args[0] == 'rm':
                return self.container_id
            if args[0] == 'logs':
                self.assertEqual(args, ('logs', '--tail', '80', self.container_id))
                return 'private fixture startup failure'
            if args[0] == 'container' and args[1] == 'ls':
                return ''
            raise AssertionError(args)
        return invoke

    def start_runtime(self):
        return runtime.start(self.state, self.source, self.binary, self.environment, self.manifest,
                             self.projection, 19443, 55531)

    def test_start_uses_direct_entrypoint_ro_root_and_exact_mounts(self):
        calls = []
        with patch.object(runtime, 'build_image', return_value=self.image_id), \
                patch.object(runtime.secrets, 'token_hex', return_value='d' * 32), \
                patch.object(runtime, 'docker', side_effect=self.fake_start_docker(calls)), \
                patch.object(runtime, 'inspect', return_value=self.proof), \
                patch.object(runtime, 'process_identity', return_value=self.identity):
            record = self.start_runtime()
        create = calls[0]
        self.assertEqual(create[-1], self.image_id)
        self.assertEqual(create[create.index('--entrypoint') + 1], str(self.binary))
        self.assertIn('--read-only', create)
        self.assertIn('--network=host', create)
        self.assertNotIn('--init', create)
        self.assertNotIn('--privileged', create)
        self.assertEqual(record['container_id'], self.container_id)
        self.assertFalse(record['network_namespace_isolated'])
        self.assertEqual((self.state / 'runtime-container.json').stat().st_mode & 0o777, 0o600)

    def test_publication_failure_cleans_only_newly_created_own_id(self):
        calls = []
        real_private_json = runtime.private_json
        def fail_publication(path, value):
            if path.name == 'runtime-container.json':
                raise OSError('publication failed')
            return real_private_json(path, value)
        with patch.object(runtime, 'build_image', return_value=self.image_id), \
                patch.object(runtime.secrets, 'token_hex', return_value='d' * 32), \
                patch.object(runtime, 'docker', side_effect=self.fake_start_docker(calls)), \
                patch.object(runtime, 'inspect', return_value=self.proof), \
                patch.object(runtime, 'process_identity', return_value=self.identity), \
                patch.object(runtime, '_terminate') as terminate, \
                patch.object(runtime, 'private_json', side_effect=fail_publication):
            with self.assertRaises(OSError):
                self.start_runtime()
        terminate.assert_called_once()
        self.assertEqual(calls[-1], ('rm', self.container_id))

    def test_interrupted_create_recovers_matching_owned_object_by_id(self):
        intent = dict(self.record, name=runtime.container_name(self.state, self.commit))
        intent.pop('container_id')
        for key in self.identity:
            if key != 'uid':
                intent.pop(key, None)
        runtime.private_json(self.state / 'runtime-container-intent.json', intent)
        proof = copy.deepcopy(self.proof)
        proof['State'] = {'Running': False, 'Pid': 0}
        def fake(state, *args):
            if args[:2] == ('container', 'ls'):
                return self.container_id
            if args[0] == 'logs':
                self.assertEqual(args, ('logs', '--tail', '80', self.container_id))
                return ''
            self.assertEqual(args, ('rm', self.container_id))
            return ''
        with patch.object(runtime, 'docker', side_effect=fake) as docker, \
                patch.object(runtime, 'inspect', return_value=proof):
            self.assertTrue(runtime.recover_intent(self.state))
        self.assertEqual(docker.call_args_list[-1].args, (self.state, 'rm', self.container_id))
        self.assertFalse((self.state / 'runtime-container-intent.json').exists())

    def test_interrupted_create_never_removes_foreign_stable_name(self):
        intent = dict(self.record, name=runtime.container_name(self.state, self.commit))
        runtime.private_json(self.state / 'runtime-container-intent.json', intent)
        proof = copy.deepcopy(self.proof)
        proof['Config']['Labels'][runtime.PREFIX + 'instance'] = 'foreign'
        with patch.object(runtime, 'docker', return_value=self.container_id) as docker, \
                patch.object(runtime, 'inspect', return_value=proof), \
                patch.object(runtime, '_remove_started') as remove:
            with self.assertRaises(runtime.ContainerError):
                runtime.recover_intent(self.state)
        remove.assert_not_called()
        self.assertEqual(docker.call_args.args[1:3], ('container', 'ls'))

    def test_owned_process_can_stop_after_material_or_source_changed(self):
        (self.source / 'source.go').write_text('changed after startup')
        (self.root / 'material/server.key').write_text('changed after startup')
        with patch.object(runtime, 'inspect', return_value=self.proof), \
                patch.object(runtime, 'process_identity', return_value=self.identity):
            self.assertEqual(runtime.verify_ownership(self.state, self.record), self.record)
            with self.assertRaises(runtime.ContainerError):
                runtime.verify_record(self.state, self.record)

    def test_required_rw_directories_and_only_communications_overlay(self):
        entry = next(item for item in self.projection['rw'] if item['kind'] == 'comunicaciones')
        self.assertEqual(entry['target'], str(self.root / 'material/comunicaciones'))
        self.projection['rw'].remove(entry)
        with self.assertRaises(runtime.ContainerError):
            self.validate()

    def test_foreign_name_collision_does_not_trigger_cleanup_or_stop(self):
        def fake(state, *args):
            if args[0] == 'container' and args[1] == 'ls':
                return ''
            raise runtime.ContainerError('name in use')
        with patch.object(runtime, 'build_image', return_value=self.image_id), \
                patch.object(runtime, 'docker', side_effect=fake), \
                patch.object(runtime, '_remove_started') as cleanup:
            with self.assertRaises(runtime.ContainerError):
                self.start_runtime()
        cleanup.assert_not_called()

    def test_extra_environment_variable_never_creates_a_container(self):
        self.environment['LD_PRELOAD'] = '/foreign/library'
        with patch.object(runtime, 'build_image', return_value=self.image_id), patch.object(runtime, 'docker') as docker:
            with self.assertRaises(runtime.ContainerError):
                self.start_runtime()
        docker.assert_not_called()

    def test_missing_pidfd_capability_has_no_docker_side_effects(self):
        with patch.object(runtime.os, 'pidfd_open', side_effect=OSError('unavailable')), \
                patch.object(runtime, 'build_image') as build, patch.object(runtime, 'docker') as docker:
            with self.assertRaises(OSError):
                self.start_runtime()
        build.assert_not_called()
        docker.assert_not_called()

    def test_private_reservation_cannot_overwrite_existing_or_symlink(self):
        path = self.state / 'runtime-container.json'
        runtime.private_json(path, self.record)
        with self.assertRaises(FileExistsError):
            runtime.private_json(path, {'foreign': True})
        self.assertEqual(runtime.read_record(self.state), self.record)
        path.unlink()
        path.symlink_to(self.envfile)
        with self.assertRaises(runtime.ContainerError):
            runtime.read_record(self.state)

    def test_image_build_uses_scratch_no_pull_no_network_and_no_binary_copy(self):
        calls = []
        def fake(state, *args):
            calls.append(args)
            if args[0] == 'build':
                context = Path(args[-1])
                dockerfile = (context / 'Dockerfile').read_text()
                self.assertTrue(dockerfile.startswith('FROM scratch\n'))
                self.assertNotIn('ADD', dockerfile)
                self.assertNotIn(str(self.binary), dockerfile)
                self.assertEqual(list((context / 'home/runtime').iterdir()), [])
                self.assertIn(f':{os.getuid()}:{os.getgid()}:', (context / 'passwd').read_text())
                Path(args[args.index('--iidfile') + 1]).write_text(self.image_id)
                return ''
            return json.dumps([{'Id': self.image_id, 'Config': {'Labels': {runtime.PREFIX + 'owner': runtime.OWNER,
                                runtime.PREFIX + 'state': str(self.state), runtime.PREFIX + 'source': self.commit}},
                                'RootFS': {'Type': 'layers'}}])
        with patch.object(runtime, 'docker', side_effect=fake):
            self.assertEqual(runtime.build_image(self.state, self.commit), self.image_id)
        self.assertIn('--pull=false', calls[0])
        self.assertIn('--network=none', calls[0])


if __name__ == '__main__':
    unittest.main()
