"""Dummy process/socket fixtures only: no Docker, TCP listener, or child commands."""
import copy
from dataclasses import replace
import hashlib
import json
import os
from pathlib import Path
import socket
import subprocess
import tempfile
import threading
import unittest
from unittest.mock import Mock, patch

import clon_relay_host as relay


class FakeSocket:
    def __init__(self, chunks=()):
        self.chunks = list(chunks)
        self.sent = bytearray()
        self.closed = False
        self.shutdowns = []
        self.bound = None

    def setblocking(self, blocking):
        self.blocking = blocking

    def recv(self, maximum):
        data = self.chunks.pop(0)
        if len(data) > maximum:
            self.chunks.insert(0, data[maximum:])
        return data[:maximum]

    def send(self, data):
        size = min(3, len(data))
        self.sent.extend(data[:size])
        return size

    def shutdown(self, how):
        self.shutdowns.append(how)

    def close(self):
        self.closed = True

    def bind(self, address):
        self.bound = address

    def listen(self, backlog):
        self.backlog = backlog

    def settimeout(self, timeout):
        self.timeout = timeout


class FakePipe:
    def __init__(self, number):
        self.number, self.closed = number, False

    def fileno(self):
        return self.number

    def close(self):
        self.closed = True


class FakeProcess:
    def __init__(self):
        self.stdin, self.stdout = FakePipe(91), FakePipe(92)
        self.alive = True
        self.terminated = self.killed = False

    def poll(self):
        return None if self.alive else 0

    def terminate(self):
        self.terminated = True
        self.alive = False

    def kill(self):
        self.killed = True
        self.alive = False

    def wait(self, timeout):
        return 0


class RelayTests(unittest.TestCase):
    def setUp(self):
        self.temporary = tempfile.TemporaryDirectory()
        self.addCleanup(self.temporary.cleanup)
        self.root = Path(self.temporary.name)
        self.state_path = self.root / 'attempt'
        self.state_path.mkdir(mode=0o700)
        self.binary = self.root / 'relay'
        self.binary.write_bytes(b'private static fixture relay')
        self.binary.chmod(0o500)
        self.config = relay.Config('a' * 64, 'b' * 64, 'sha256:' + 'c' * 64,
                                  'sha256:' + 'd' * 64,
                                  hashlib.sha256(self.binary.read_bytes()).hexdigest(),
                                  self.binary, 18531, 18531, self.state_path, 'e' * 64)
        self.approval = {'app_port': 18531, 'pg_container_id': self.config.pg_id,
                         'pg_image_id': self.config.pg_image_id}
        self.app = {'Id': self.config.app_id, 'Image': self.config.app_image_id,
                    'State': {'Running': True, 'Pid': 71},
                    'HostConfig': {'NetworkMode': 'container:' + self.config.pg_id,
                                   'ReadonlyRootfs': True, 'Privileged': False,
                                   'PortBindings': {}}, 'NetworkSettings': {'Ports': {}},
                    'Mounts': [{'Type': 'bind', 'Source': str(self.binary),
                                'Destination': relay.RELAY, 'RW': False,
                                'Propagation': 'rprivate'}]}
        self.pg = {'Id': self.config.pg_id, 'Image': self.config.pg_image_id,
                   'State': {'Running': True, 'Pid': 72},
                   'HostConfig': {'NetworkMode': 'none', 'Privileged': False,
                                  'PortBindings': None},
                   'NetworkSettings': {'Ports': {'5432/tcp': None}}}
        self.docker = Mock()
        self.docker.inspect.side_effect = lambda identity: copy.deepcopy(
            self.app if identity == self.config.app_id else self.pg)
        self.docker.binary_sha.return_value = self.config.relay_sha256

    def server(self, config=None):
        server = relay.RelayServer(config or self.config, self.approval,
                                   docker_factory=lambda state: self.docker,
                                   socket_factory=lambda *args: FakeSocket())
        self.addCleanup(server.close)
        return server

    def test_validated_loopback_and_private_durable_receipt(self):
        server = self.server()
        receipt = server.start()
        self.assertEqual(server.listener.bound, ('127.0.0.1', 18531))
        self.assertEqual(receipt['pid'], os.getpid())
        self.assertEqual(receipt['start_ticks'], relay.process_start_ticks())
        self.assertEqual(receipt['approval_sha256'], self.config.approval_sha256)
        self.assertEqual(self.docker.inspect.call_count, 4)
        for name in (relay.RECEIPT, relay.READY):
            file = self.state_path / name
            self.assertEqual(file.stat().st_mode & 0o777, 0o600)
            self.assertEqual(json.loads(file.read_text()), receipt)
        server.close()
        self.assertFalse((self.state_path / relay.READY).exists())
        self.assertTrue((self.state_path / relay.RECEIPT).exists())

    def test_unapproved_and_dynamic_endpoint_rejected_before_state_changes(self):
        for field, value in [('local_port', 18532), ('target_port', 443),
                             ('local_port', True), ('app_id', 'container-name'),
                             ('pg_id', self.config.app_id), ('max_clients', 33),
                             ('idle_timeout', 0), ('verify_interval', 6)]:
            with self.subTest(field=field), self.assertRaises(relay.Refused):
                relay.RelayServer(replace(self.config, **{field: value}), self.approval)
        self.assertEqual(list(self.state_path.iterdir()), [])
        self.docker.inspect.assert_not_called()

    def test_drift_of_every_pinned_boundary_is_rejected(self):
        cases = [('app', 'Id', 'f' * 64), ('app', 'Image', self.config.pg_image_id),
                 ('pg', 'Image', self.config.app_image_id),
                 ('app', 'HostConfig.NetworkMode', 'host'),
                 ('pg', 'HostConfig.NetworkMode', 'bridge'),
                 ('app', 'HostConfig.ReadonlyRootfs', False),
                 ('app', 'HostConfig.Privileged', True),
                 ('app', 'State.Running', False),
                 ('pg', 'HostConfig.PortBindings', {'5432/tcp': [{'HostPort': '55531'}]}),
                 ('app', 'NetworkSettings.Ports', {'18531/tcp': [{'HostPort': '18531'}]}),
                 ('app', 'Mounts', []),
                 ('app', 'Mounts', [dict(self.app['Mounts'][0], RW=True)]),
                 ('app', 'Mounts', [dict(self.app['Mounts'][0], Source='/foreign')]),
                 ('app', 'Mounts', self.app['Mounts'] + [{'Destination': '/opt'}])]
        for subject, field, value in cases:
            app, pg = copy.deepcopy(self.app), copy.deepcopy(self.pg)
            target = app if subject == 'app' else pg
            parts = field.split('.')
            for part in parts[:-1]:
                target = target[part]
            target[parts[-1]] = value
            docker = Mock()
            docker.inspect.side_effect = lambda identity: app if identity == self.config.app_id else pg
            docker.binary_sha.return_value = self.config.relay_sha256
            with self.subTest(field=field), self.assertRaises(relay.Refused):
                relay.verify(self.config, docker)

    def test_hash_drift_on_host_and_inside_app(self):
        self.docker.binary_sha.return_value = 'f' * 64
        with self.assertRaises(relay.Refused):
            relay.verify(self.config, self.docker)
        self.docker.binary_sha.return_value = self.config.relay_sha256
        self.binary.chmod(0o700)
        self.binary.write_bytes(b'replaced')
        with self.assertRaises(relay.Refused):
            relay.verify(self.config, self.docker)

    def test_runtime_drift_stops_listener_and_removes_readiness(self):
        server = self.server()
        server.start()
        listener = server.listener
        self.pg['Image'] = 'sha256:' + 'f' * 64
        with self.assertRaises(relay.Refused):
            server.serve()
        self.assertTrue(listener.closed)
        self.assertTrue(server.stop.is_set())
        self.assertFalse((self.state_path / relay.READY).exists())
        self.assertTrue((self.state_path / relay.RECEIPT).exists())

    def test_foreign_receipt_is_not_replaced(self):
        foreign = self.state_path / relay.RECEIPT
        foreign.write_text('foreign')
        server = self.server()
        with self.assertRaises(FileExistsError):
            server.start()
        self.assertEqual(foreign.read_text(), 'foreign')
        self.assertFalse((self.state_path / relay.READY).exists())

    def test_container_restart_invalidates_ready_receipt(self):
        server = self.server()
        server.start()
        self.app['State']['Pid'] += 1
        with self.assertRaises(relay.Refused):
            server.serve()
        self.assertFalse((self.state_path / relay.READY).exists())

    def test_incomplete_inspection_is_refused_without_traceback(self):
        for field, value in [('HostConfig', None), ('State', []),
                             ('NetworkSettings', None), ('Mounts', [None])]:
            original = self.app[field]
            self.app[field] = value
            with self.subTest(field=field), self.assertRaises(relay.Refused):
                relay.verify(self.config, self.docker)
            self.app[field] = original

    def test_symlink_and_hardlink_sources_are_refused(self):
        alias = self.root / 'alias'
        alias.symlink_to(self.binary)
        with self.assertRaises(OSError):
            relay.read_file(alias, 1024)
        alias.unlink()
        os.link(self.binary, alias)
        with self.assertRaises(relay.Refused):
            relay.read_file(self.binary, 1024)

    def test_external_approval_is_pinned_private_and_duplicate_keys_refused(self):
        path = self.root / 'approval.json'
        path.write_text(json.dumps(self.approval))
        path.chmod(0o600)
        digest = hashlib.sha256(path.read_bytes()).hexdigest()
        self.assertEqual(relay.load_approval(path, digest, self.state_path), self.approval)
        with self.assertRaises(relay.Refused):
            relay.load_approval(path, 'f' * 64, self.state_path)
        path.chmod(0o644)
        with self.assertRaises(relay.Refused):
            relay.load_approval(path, digest, self.state_path)
        with self.assertRaises(relay.Refused):
            relay.decode(b'{"app_port":18531,"app_port":18532}')
        with self.assertRaises(relay.Refused):
            relay.load_approval(self.state_path / 'approval.json', digest, self.state_path)

    def test_fixed_docker_command_allowlist_and_no_traffic_logging(self):
        state = relay.State(self.state_path)
        self.addCleanup(state.close)
        def run(argv, **kwargs):
            self.assertEqual(kwargs['env'], {'PATH': '/usr/bin:/bin',
                'HOME': str(self.state_path / 'docker-client')})
            self.assertEqual(kwargs['stderr'], subprocess.DEVNULL)
            self.assertEqual(kwargs['stdin'], subprocess.DEVNULL)
            output = json.dumps([self.app]).encode() if argv[-3:-1] == ['container', 'inspect'] else (
                     self.config.relay_sha256 + '\n').encode()
            return subprocess.CompletedProcess(argv, 0, output)
        popen = Mock(return_value=FakeProcess())
        runner = Mock(side_effect=run)
        docker = relay.Docker(state, runner=runner, popen=popen)
        with patch.dict(os.environ, {'DOCKER_HOST': 'tcp://foreign',
                                    'DOCKER_CONTEXT': 'foreign', 'HTTP_PROXY': 'foreign'}):
            self.assertEqual(docker.inspect(self.config.app_id), self.app)
            self.assertEqual(docker.binary_sha(self.config.app_id), self.config.relay_sha256)
            docker.connect(self.config)
        argv = popen.call_args.args[0]
        self.assertEqual(argv, [relay.DOCKER, '--config', str(self.state_path / 'docker-client'),
            '--host', 'unix:///var/run/docker.sock', 'exec', '-i', self.config.app_id,
            relay.RELAY, '18531'])
        self.assertNotIn('-t', argv)
        self.assertNotIn('shell', popen.call_args.kwargs)
        self.assertEqual(popen.call_args.kwargs['stderr'], subprocess.DEVNULL)
        self.assertTrue(popen.call_args.kwargs['close_fds'])
        self.assertEqual(set(popen.call_args.kwargs['env']), {'PATH', 'HOME'})
        self.assertEqual(list((self.state_path / 'docker-client').iterdir()), [])

    def test_docker_config_injection_refused_without_running_docker(self):
        state = relay.State(self.state_path)
        self.addCleanup(state.close)
        (self.state_path / 'docker-client/config.json').write_text('{"credsStore":"foreign"}')
        runner = Mock()
        docker = relay.Docker(state, runner=runner)
        with self.assertRaises(relay.Refused):
            docker.inspect(self.config.app_id)
        runner.assert_not_called()

    def test_bridge_preserves_binary_bytes_partial_writes_and_halfclose(self):
        client = FakeSocket([b'\x16\x03\x03\x00\x04\x00\xff\x01\x02', b''])
        process = FakeProcess()
        input_bytes = bytearray()
        response = [b'\x17\x03\x03\x00\x03\xfe\xfd\xfc', b'']
        def wait(reads, writes, errors, timeout):
            self.assertLessEqual(timeout, 0.2)
            self.assertEqual(errors, [])
            # Only let the app finish after receiving the entire client stream.
            selected = [client] if client in reads else []
            if process.stdin.closed and 92 in reads:
                selected.append(92)
            return selected, writes, []
        def write(fd, data):
            self.assertEqual(fd, 91)
            length = min(2, len(data))
            input_bytes.extend(data[:length])
            return length
        with patch.object(relay.os, 'set_blocking'), \
             patch.object(relay.os, 'write', side_effect=write), \
             patch.object(relay.os, 'read', side_effect=lambda fd, maximum: response.pop(0)):
            relay.bridge(client, process, threading.Event(), self.config, wait=wait)
        self.assertEqual(input_bytes, b'\x16\x03\x03\x00\x04\x00\xff\x01\x02')
        self.assertEqual(client.sent, b'\x17\x03\x03\x00\x03\xfe\xfd\xfc')
        self.assertTrue(process.stdin.closed)
        self.assertEqual(client.shutdowns, [socket.SHUT_WR])

    def test_bridge_idle_and_absolute_time_limits(self):
        for configuration, ticks in [(replace(self.config, idle_timeout=1), [0, 0, 2]),
                                      (replace(self.config, idle_timeout=60,
                                               connection_timeout=60), [0, 0, 61])]:
            with self.subTest(configuration=configuration):
                process = FakeProcess()
                with patch.object(relay.os, 'set_blocking'):
                    relay.bridge(FakeSocket(), process, threading.Event(), configuration,
                                 wait=lambda *args: ([], [], []), clock=Mock(side_effect=ticks))
                self.assertFalse(process.terminated)

    def test_normal_browser_reset_does_not_stop_the_listener(self):
        server = self.server()
        server.start()
        client = FakeSocket()
        process = FakeProcess()
        self.docker.connect.return_value = process
        self.assertTrue(server.slots.acquire(blocking=False))
        with patch.object(relay, 'bridge', side_effect=ConnectionResetError()):
            server.client(client)
        self.assertFalse(server.stop.is_set())
        self.assertTrue(client.closed)
        self.assertTrue(process.terminated)
        self.assertTrue(process.stdin.closed)
        self.assertTrue((self.state_path / relay.READY).exists())

    def test_concurrent_client_limit_rejects_extra_connection(self):
        server = self.server(replace(self.config, max_clients=1))
        server.start()
        self.assertTrue(server.slots.acquire(blocking=False))
        extra = FakeSocket()
        def accept():
            server.stop.set()
            return extra, ('127.0.0.1', 54321)
        server.listener.accept = accept
        server.serve()
        self.assertTrue(extra.closed)
        self.docker.connect.assert_not_called()

    def test_stalled_child_is_killed_and_pipes_closed(self):
        process = FakeProcess()
        process.wait = Mock(side_effect=[subprocess.TimeoutExpired('dummy', 1), 0])
        relay.finish(process)
        self.assertTrue(process.terminated)
        self.assertTrue(process.killed)
        self.assertTrue(process.stdin.closed)
        self.assertTrue(process.stdout.closed)


if __name__ == '__main__':
    unittest.main()
