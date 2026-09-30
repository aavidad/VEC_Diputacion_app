#!/usr/bin/env python3
"""Opaque, bounded loopback TCP relay; TLS is terminated only by the pinned app.

The caller supplies the externally approved clone identity. The foreground
process publishes one private durable receipt and a separate readiness marker.
It never logs traffic, inspects HTTP, chooses a target from client bytes, or
creates, changes, removes, or publishes Docker containers.
"""
from __future__ import annotations

import argparse
from contextlib import contextmanager
from dataclasses import dataclass
import hashlib
import json
import os
from pathlib import Path
import re
import secrets
import select
import signal
import socket
import stat
import subprocess
import sys
import threading
import time

HEX64 = re.compile(r'[a-f0-9]{64}\Z')
IMAGE = re.compile(r'sha256:[a-f0-9]{64}\Z')
RELAY = '/opt/vec/relay_tcp'
DOCKER = '/usr/bin/docker'
RECEIPT = 'relay-host.json'
READY = 'relay-host-ready.json'
MAX_JSON = 2 * 1024 * 1024


class Refused(RuntimeError):
    pass


def require(condition, code):
    if not condition:
        raise Refused(code)


@contextmanager
def directory(path):
    path = Path(path)
    require(path.is_absolute() and '..' not in path.parts, 'relay_invalid_path')
    fd = os.open('/', os.O_RDONLY | os.O_DIRECTORY | os.O_CLOEXEC)
    try:
        for part in path.parts[1:]:
            child = os.open(part, os.O_RDONLY | os.O_DIRECTORY | os.O_NOFOLLOW |
                            os.O_CLOEXEC, dir_fd=fd)
            os.close(fd)
            fd = child
        yield fd
    finally:
        os.close(fd)


def signature(info):
    return (info.st_dev, info.st_ino, info.st_size, info.st_mtime_ns, info.st_ctime_ns)


def read_file(path, limit, private=False):
    path = Path(path)
    with directory(path.parent) as parent:
        fd = os.open(path.name, os.O_RDONLY | os.O_NOFOLLOW | os.O_NONBLOCK |
                     os.O_CLOEXEC, dir_fd=parent)
        with os.fdopen(fd, 'rb') as stream:
            info = os.fstat(stream.fileno())
            require(stat.S_ISREG(info.st_mode) and info.st_nlink == 1 and
                    info.st_uid == os.getuid() and not info.st_mode &
                    (0o077 if private else 0o022) and 0 < info.st_size <= limit,
                    'relay_unsafe_file')
            data = stream.read(limit + 1)
            require(len(data) == info.st_size and signature(info) ==
                    signature(os.fstat(stream.fileno())), 'relay_file_changed')
    return data


def unique(pairs):
    value = {}
    for key, item in pairs:
        require(key not in value, 'relay_duplicate_json_key')
        value[key] = item
    return value


def decode(data):
    try:
        return json.loads(data, object_pairs_hook=unique,
                          parse_constant=lambda _: (_ for _ in ()).throw(ValueError()))
    except (ValueError, UnicodeError):
        raise Refused('relay_invalid_json') from None


def load_approval(path, expected_sha256, state_dir):
    require(HEX64.fullmatch(expected_sha256 or ''), 'relay_invalid_approval_pin')
    path, state_dir = Path(path), Path(state_dir)
    require(path != state_dir and state_dir not in path.parents,
            'relay_approval_must_be_external')
    data = read_file(path, MAX_JSON, private=True)
    require(hashlib.sha256(data).hexdigest() == expected_sha256,
            'relay_approval_hash_mismatch')
    value = decode(data)
    require(isinstance(value, dict), 'relay_invalid_approval')
    return value


@dataclass(frozen=True)
class Config:
    app_id: str
    pg_id: str
    app_image_id: str
    pg_image_id: str
    relay_sha256: str
    relay_source: Path
    local_port: int
    target_port: int
    state_dir: Path
    approval_sha256: str
    max_clients: int = 8
    idle_timeout: float = 30.0
    connection_timeout: float = 300.0
    verify_interval: float = 1.0

    def validate(self, approval):
        require(isinstance(approval, dict), 'relay_invalid_approval')
        require(all(isinstance(x, str) and HEX64.fullmatch(x) for x in
                    (self.app_id, self.pg_id, self.relay_sha256, self.approval_sha256)),
                'relay_invalid_identity_pin')
        require(self.app_id != self.pg_id and all(isinstance(x, str) and
                    IMAGE.fullmatch(x) for x in (self.app_image_id, self.pg_image_id)),
                'relay_invalid_image_pin')
        require(type(self.local_port) is int and type(self.target_port) is int and
                    1024 <= self.local_port <= 65535 and
                    self.local_port == self.target_port == approval.get('app_port') and
                    type(approval.get('app_port')) is int and
                    self.pg_id == approval.get('pg_container_id') and
                    self.pg_image_id == approval.get('pg_image_id'),
                'relay_unapproved_endpoint')
        require(type(self.max_clients) is int and 1 <= self.max_clients <= 32 and
                    0 < self.idle_timeout <= 60 and
                    self.idle_timeout <= self.connection_timeout <= 600 and
                    0 < self.verify_interval <= 5, 'relay_invalid_limits')
        for path in (self.state_dir, self.relay_source):
            require(Path(path).is_absolute() and '..' not in Path(path).parts,
                    'relay_invalid_path')


class State:
    """Keep directory descriptors; do not overwrite or delete foreign records."""
    def __init__(self, path):
        self.path = Path(path)
        self.context = directory(path)
        self.fd = self.context.__enter__()
        try:
            info = os.fstat(self.fd)
            require(info.st_uid == os.getuid() and stat.S_IMODE(info.st_mode) == 0o700,
                    'relay_state_not_private')
            self.identity = (info.st_dev, info.st_ino)
            self.files = {}
            os.mkdir('docker-client', mode=0o700, dir_fd=self.fd)
            self.docker_fd = os.open('docker-client', os.O_RDONLY | os.O_DIRECTORY |
                                     os.O_NOFOLLOW | os.O_CLOEXEC, dir_fd=self.fd)
        except BaseException:
            self.context.__exit__(*sys.exc_info())
            raise

    def check(self):
        with directory(self.path) as current:
            info = os.fstat(current)
            require((info.st_dev, info.st_ino) == self.identity and
                    stat.S_IMODE(info.st_mode) == 0o700 and info.st_uid == os.getuid(),
                    'relay_state_changed')
            info = os.stat('docker-client', dir_fd=current, follow_symlinks=False)
            original = os.fstat(self.docker_fd)
            require((info.st_dev, info.st_ino) == (original.st_dev, original.st_ino) and
                    stat.S_ISDIR(info.st_mode) and stat.S_IMODE(info.st_mode) == 0o700 and
                    info.st_uid == os.getuid() and not os.listdir(self.docker_fd),
                    'relay_docker_config_changed')

    def publish(self, name, value):
        data = (json.dumps(value, sort_keys=True, separators=(',', ':')) + '\n').encode()
        temporary = '.relay-' + secrets.token_hex(16)
        fd = os.open(temporary, os.O_WRONLY | os.O_CREAT | os.O_EXCL |
                     os.O_NOFOLLOW | os.O_CLOEXEC, 0o600, dir_fd=self.fd)
        try:
            with os.fdopen(fd, 'wb') as stream:
                stream.write(data)
                stream.flush()
                os.fsync(stream.fileno())
                info = os.fstat(stream.fileno())
            os.link(temporary, name, src_dir_fd=self.fd, dst_dir_fd=self.fd,
                    follow_symlinks=False)
            self.files[name] = (info.st_dev, info.st_ino)
            os.fsync(self.fd)
        finally:
            os.unlink(temporary, dir_fd=self.fd)

    def remove(self, name):
        owned = self.files.get(name)
        if owned is None:
            return
        try:
            info = os.stat(name, dir_fd=self.fd, follow_symlinks=False)
            require((info.st_dev, info.st_ino) == owned, 'relay_record_changed')
            os.unlink(name, dir_fd=self.fd)
            os.fsync(self.fd)
            self.files.pop(name)
        except FileNotFoundError:
            self.files.pop(name)

    def close(self):
        os.close(self.docker_fd)
        self.context.__exit__(None, None, None)


class Docker:
    def __init__(self, state, runner=subprocess.run, popen=subprocess.Popen):
        self.state, self.runner, self.popen = state, runner, popen

    def command(self, *args):
        self.state.check()
        config = str(self.state.path / 'docker-client')
        return [DOCKER, '--config', config, '--host', 'unix:///var/run/docker.sock', *args]

    def environment(self):
        return {'PATH': '/usr/bin:/bin', 'HOME': str(self.state.path / 'docker-client')}

    def inspect(self, container_id):
        require(HEX64.fullmatch(container_id), 'relay_invalid_container_id')
        result = self.runner(self.command('container', 'inspect', container_id),
                             env=self.environment(), stdin=subprocess.DEVNULL,
                             stdout=subprocess.PIPE, stderr=subprocess.DEVNULL,
                             timeout=5, check=False)
        require(result.returncode == 0 and len(result.stdout) <= MAX_JSON,
                'relay_inspect_failed')
        value = decode(result.stdout)
        require(isinstance(value, list) and len(value) == 1 and
                isinstance(value[0], dict) and value[0].get('Id') == container_id,
                'relay_container_changed')
        return value[0]

    def binary_sha(self, container_id):
        result = self.runner(self.command('exec', container_id, RELAY, '--sha256'),
                             env=self.environment(), stdin=subprocess.DEVNULL,
                             stdout=subprocess.PIPE, stderr=subprocess.DEVNULL,
                             timeout=5, check=False)
        require(result.returncode == 0 and len(result.stdout) <= 65,
                'relay_binary_probe_failed')
        try:
            value = result.stdout.decode('ascii')
            require(HEX64.fullmatch(value.removesuffix('\n')),
                    'relay_binary_probe_failed')
            return value.removesuffix('\n')
        except UnicodeError:
            raise Refused('relay_binary_probe_failed') from None

    def connect(self, config):
        return self.popen(self.command('exec', '-i', config.app_id, RELAY,
                                       str(config.target_port)), env=self.environment(),
                          stdin=subprocess.PIPE, stdout=subprocess.PIPE,
                          stderr=subprocess.DEVNULL, bufsize=0, close_fds=True)


def verify(config, docker):
    app = docker.inspect(config.app_id)
    pg = docker.inspect(config.pg_id)
    require(app.get('Id') == config.app_id and pg.get('Id') == config.pg_id,
            'relay_container_changed')
    for proof, image in ((app, config.app_image_id), (pg, config.pg_image_id)):
        host, state, network = (proof.get(field) for field in
                               ('HostConfig', 'State', 'NetworkSettings'))
        require(all(isinstance(field, dict) for field in (host, state, network)) and
                isinstance(host.get('PortBindings') or {}, dict) and
                isinstance(network.get('Ports') or {}, dict), 'relay_invalid_inspection')
        require(proof.get('Image') == image and state.get('Running') is True
                and type(state.get('Pid')) is int and state['Pid'] > 0
                and host.get('Privileged') is False and
                not any((host.get('PortBindings') or {}).values()) and
                not any((network.get('Ports') or {}).values()),
                'relay_container_identity_drift')
    require(app['HostConfig'].get('NetworkMode') == 'container:' + config.pg_id and
            pg['HostConfig'].get('NetworkMode') == 'none', 'relay_namespace_drift')
    require(app['HostConfig'].get('ReadonlyRootfs') is True,
            'relay_application_root_writable')
    mounts = app.get('Mounts', [])
    require(isinstance(mounts, list) and all(isinstance(mount, dict) and
            isinstance(mount.get('Destination'), str) and
            Path(mount['Destination']).is_absolute() and
            '..' not in Path(mount['Destination']).parts for mount in mounts),
            'relay_invalid_mount_inspection')
    targets = [m for m in mounts if m.get('Destination') == RELAY]
    require(len(targets) == 1 and targets[0].get('Type') == 'bind' and
            targets[0].get('Source') == str(config.relay_source) and
            targets[0].get('RW') is False and targets[0].get('Propagation') == 'rprivate',
            'relay_mount_drift')
    require(not any(Path(m.get('Destination', '/')).is_relative_to(Path(RELAY)) or
                    Path(RELAY).is_relative_to(Path(m.get('Destination', '/')))
                    for m in mounts if m not in targets), 'relay_overlapping_mount')
    data = read_file(config.relay_source, 32 * 1024 * 1024)
    require(hashlib.sha256(data).hexdigest() == config.relay_sha256 and
            docker.binary_sha(config.app_id) == config.relay_sha256,
            'relay_binary_hash_drift')
    return (app['State']['Pid'], pg['State']['Pid'])


def finish(process):
    if process.poll() is None:
        process.terminate()
        try:
            process.wait(timeout=1)
        except subprocess.TimeoutExpired:
            process.kill()
            process.wait(timeout=1)
    for pipe in (process.stdin, process.stdout):
        if pipe is not None and not pipe.closed:
            pipe.close()


def bridge(client, process, stop, config, wait=select.select, clock=time.monotonic):
    """One bounded byte stream; backpressure and half-close preserve TLS bytes."""
    client.setblocking(False)
    incoming, outgoing = bytearray(), bytearray()
    source, sink = process.stdout.fileno(), process.stdin.fileno()
    os.set_blocking(source, False)
    os.set_blocking(sink, False)
    started = last = clock()
    client_eof = relay_eof = False
    while not stop.is_set():
        now = clock()
        if now - last >= config.idle_timeout or now - started >= config.connection_timeout:
            return
        reads = ([] if client_eof or len(incoming) >= 65536 else [client])
        if not relay_eof and len(outgoing) < 65536:
            reads.append(source)
        writes = ([sink] if incoming and not process.stdin.closed else [])
        if outgoing:
            writes.append(client)
        ready_read, ready_write, _ = wait(reads, writes, [], min(0.2, config.idle_timeout))
        for item in ready_read:
            try:
                chunk = client.recv(65536 - len(incoming)) if item is client else os.read(source, 65536 - len(outgoing))
            except BlockingIOError:
                continue
            if chunk:
                (incoming if item is client else outgoing).extend(chunk)
                last = clock()
            elif item is client:
                client_eof = True
            else:
                relay_eof = True
        for item in ready_write:
            try:
                written = client.send(outgoing) if item is client else os.write(sink, incoming)
            except BlockingIOError:
                continue
            if written <= 0:
                raise OSError('relay_stream_closed')
            del (outgoing if item is client else incoming)[:written]
            last = clock()
        if client_eof and not incoming and not process.stdin.closed:
            process.stdin.close()
        if relay_eof and not outgoing:
            client.shutdown(socket.SHUT_WR)
            return


class RelayServer:
    def __init__(self, config, approval, *, docker_factory=Docker, socket_factory=socket.socket):
        config.validate(approval)
        self.config, self.approval = config, dict(approval)
        self.state = State(config.state_dir)
        self.docker = docker_factory(self.state)
        self.socket_factory = socket_factory
        self.listener = None
        self.stop = threading.Event()
        self.clients = set()
        self.lock = threading.Lock()
        self.slots = threading.BoundedSemaphore(config.max_clients)
        self.threads = set()
        self.verify_lock = threading.Lock()
        self.failed = False
        self.live_identity = None

    def verify(self):
        with self.verify_lock:
            self.config.validate(self.approval)
            identity = verify(self.config, self.docker)
            require(self.live_identity is None or self.live_identity == identity,
                    'relay_container_process_drift')
            self.live_identity = identity

    def start(self):
        try:
            self.verify()
            listener = self.socket_factory(socket.AF_INET, socket.SOCK_STREAM)
            self.listener = listener
            listener.bind(('127.0.0.1', self.config.local_port))
            listener.listen(self.config.max_clients)
            listener.settimeout(0.2)
            self.verify()
            receipt = {'version': 1, 'instance': secrets.token_hex(16),
                       'pid': os.getpid(), 'start_ticks': process_start_ticks(),
                       'uid': os.getuid(), 'app_id': self.config.app_id,
                       'pg_id': self.config.pg_id, 'app_image_id': self.config.app_image_id,
                       'app_pid': self.live_identity[0], 'pg_pid': self.live_identity[1],
                       'pg_image_id': self.config.pg_image_id,
                       'relay_sha256': self.config.relay_sha256,
                       'approval_sha256': self.config.approval_sha256,
                       'listen_host': '127.0.0.1', 'port': self.config.local_port}
            self.state.publish(RECEIPT, receipt)
            self.state.publish(READY, receipt)
            return receipt
        except BaseException:
            self.close()
            raise

    def client(self, client):
        process = None
        try:
            if self.stop.is_set():
                return
            process = self.docker.connect(self.config)
            bridge(client, process, self.stop, self.config)
        except Refused:
            self.failed = True
            self.stop.set()
            try:
                self.state.remove(READY)
            except (OSError, Refused):
                pass
        except (OSError, ValueError, subprocess.SubprocessError):
            # A browser abandoning its TLS socket is local to that connection.
            pass
        finally:
            try:
                if process is not None:
                    finish(process)
            finally:
                client.close()
                with self.lock:
                    self.clients.discard(client)
                    self.threads.discard(threading.current_thread())
                self.slots.release()

    def serve(self):
        require(self.listener is not None, 'relay_not_started')
        next_check = time.monotonic()
        try:
            while not self.stop.is_set():
                if time.monotonic() >= next_check:
                    self.verify()
                    next_check = time.monotonic() + self.config.verify_interval
                try:
                    client, address = self.listener.accept()
                except socket.timeout:
                    continue
                if address[0] != '127.0.0.1' or not self.slots.acquire(blocking=False):
                    client.close()
                    continue
                try:
                    self.verify()
                except BaseException:
                    client.close()
                    self.slots.release()
                    raise
                thread = threading.Thread(target=self.client, args=(client,), daemon=True)
                with self.lock:
                    self.clients.add(client)
                    self.threads.add(thread)
                thread.start()
        finally:
            self.close()

    def close(self):
        self.stop.set()
        if self.state is not None:
            try:
                self.state.remove(READY)
            except (OSError, Refused):
                # Even a replaced record must not prevent closing the listener.
                self.failed = True
        if self.listener is not None:
            self.listener.close()
            self.listener = None
        with self.lock:
            clients, threads = list(self.clients), list(self.threads)
        for client in clients:
            try:
                client.shutdown(socket.SHUT_RDWR)
            except OSError:
                pass
        for thread in threads:
            if thread is not threading.current_thread():
                thread.join(timeout=7)
        if self.state is not None:
            self.state.close()
            self.state = None


def process_start_ticks():
    # /proc comm may contain spaces and parentheses; fields begin after its final ')'.
    return Path('/proc/self/stat').read_text().rsplit(')', 1)[1].split()[19]


def main(argv=None):
    parser = argparse.ArgumentParser()
    for name in ('app-id', 'pg-id', 'app-image-id', 'pg-image-id', 'relay-sha256',
                 'relay-source', 'state-dir', 'approval', 'approval-sha256'):
        parser.add_argument('--' + name, required=True)
    for name in ('local-port', 'target-port'):
        parser.add_argument('--' + name, required=True, type=int)
    args = parser.parse_args(argv)
    server = None
    try:
        approval = load_approval(args.approval, args.approval_sha256, args.state_dir)
        config = Config(**{key: Path(value) if key in ('relay_source', 'state_dir') else value
                           for key, value in vars(args).items() if key != 'approval'})
        server = RelayServer(config, approval)
        for number in (signal.SIGINT, signal.SIGTERM):
            signal.signal(number, lambda _number, _frame: server.stop.set())
        server.start()
        server.serve()
        return 1 if server.failed else 0
    except (Refused, OSError, ValueError, subprocess.SubprocessError):
        print('relay_refused', file=sys.stderr)
        return 1
    finally:
        if server is not None:
            server.close()


if __name__ == '__main__':
    raise SystemExit(main())
