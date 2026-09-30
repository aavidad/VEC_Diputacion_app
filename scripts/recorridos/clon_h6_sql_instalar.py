#!/usr/bin/env python3
"""One-shot SQL62 for an owned fresh H1; AD132 and readiness are separate.

All artifact digests and the image digest are external inputs. The restore
receipt identifies PG by immutable ID. Existing SQL state is never resumed.
An attempt marker survives every failure; recovery requires a new clone.
"""
from __future__ import annotations

import argparse
from contextlib import contextmanager
import fcntl
import json
import os
from pathlib import Path
import re
import stat
import sys

try:
    from . import clon_sql, clon_h6_kit
except ImportError:
    import clon_sql
    import clon_h6_kit

SOURCE = '73e56c106d12fdda0bd16d6fe573503c42c5495f'
ATTEMPT = 'sql62-intento.json'
LOCK = 'sql62-instalar.lock'
BLOCKERS = ('sql-journal.json', '.sql-confirming', ATTEMPT,
            'DB_READY.json', 'READY.json', 'runtime-process.json')
DIR_FLAGS = os.O_RDONLY | os.O_DIRECTORY | os.O_NOFOLLOW | os.O_CLOEXEC
FILE_FLAGS = os.O_RDONLY | os.O_NOFOLLOW | os.O_NONBLOCK | os.O_CLOEXEC


class Refused(RuntimeError):
    """Only fixed public error codes belong to this exception."""


def require(condition, code):
    if not condition:
        raise Refused(code)


def identity(info):
    return (info.st_dev, info.st_ino, info.st_uid, info.st_mode,
            info.st_nlink, info.st_size, info.st_mtime_ns, info.st_ctime_ns)


def canonical_path(path):
    path = Path(path)
    require(path.is_absolute() and '..' not in path.parts and path != Path('/'), 'private_path')
    return path


@contextmanager
def private_state(path):
    """Retain every trusted ancestor; never create or adopt private state."""
    path = canonical_path(path)
    descriptors, snapshots = [], []
    try:
        descriptors.append(os.open('/', DIR_FLAGS))
        for index in range(len(path.parts)):
            fd = descriptors[-1]
            info = os.fstat(fd)
            require(stat.S_ISDIR(info.st_mode) and info.st_uid in (0, os.getuid())
                    and not info.st_mode & 0o022, 'state_ancestor')
            try:
                os.stat('.git', dir_fd=fd, follow_symlinks=False)
            except FileNotFoundError:
                pass
            else:
                raise Refused('state_inside_git')
            snapshots.append((info.st_dev, info.st_ino, info.st_uid, info.st_mode))
            if index < len(path.parts) - 1:
                descriptors.append(os.open(path.parts[index + 1], DIR_FLAGS, dir_fd=fd))
        require(info.st_uid == os.getuid() and stat.S_IMODE(info.st_mode) == 0o700, 'state_owner_mode')
        yield descriptors[-1], (descriptors, snapshots)
    finally:
        for fd in reversed(descriptors):
            os.close(fd)


def stable_state(path, retained):
    descriptors, expected = retained
    for fd, snapshot in zip(descriptors, expected, strict=True):
        now = os.fstat(fd)
        require((now.st_dev, now.st_ino, now.st_uid, now.st_mode) == snapshot, 'state_changed')
    with private_state(path) as (_, (_, current)):
        require(current == expected, 'state_replaced')


def present(directory, name):
    try:
        os.stat(name, dir_fd=directory, follow_symlinks=False)
    except FileNotFoundError:
        return False
    return True


def fresh(directory):
    require(not any(present(directory, name) for name in BLOCKERS), 'sql_state_exists_new_clone_required')


def read_owned(directory, name, limit=16384):
    fd = os.open(name, FILE_FLAGS, dir_fd=directory)
    try:
        before = os.fstat(fd)
        require(stat.S_ISREG(before.st_mode) and before.st_nlink == 1
                and before.st_uid == os.getuid() and stat.S_IMODE(before.st_mode) == 0o600
                and 0 < before.st_size <= limit, 'private_file_metadata')
        data = bytearray()
        while len(data) <= limit:
            chunk = os.read(fd, min(65536, limit + 1 - len(data)))
            if not chunk:
                break
            data.extend(chunk)
        require(len(data) == before.st_size and identity(before) == identity(os.fstat(fd))
                and identity(before) == identity(os.stat(name, dir_fd=directory, follow_symlinks=False)),
                'private_file_changed')
        return bytes(data)
    finally:
        os.close(fd)


def metadata(directory, state, receipt, request):
    raw = read_owned(directory, 'clon.json')
    value = json.loads(raw, object_pairs_hook=clon_h6_kit.unique)
    expected = {'propietario': 'Codex-M', 'estado': str(state),
                'contenedor': receipt['pg_container_id'], 'puerto_pg': 5432, 'commit': SOURCE}
    require(isinstance(value, dict) and all(value.get(key) == item for key, item in expected.items()),
            'clone_inventory_mismatch')
    require(type(value['puerto_pg']) is int, 'clone_inventory_mismatch')
    if 'identidad_clon' in value:
        require(value['identidad_clon'] == request.approved_restore_receipt_sha256, 'clone_identity_mismatch')
    return clon_sql.sha(raw)


@contextmanager
def installer_lock(directory):
    fd = os.open(LOCK, os.O_RDWR | os.O_CREAT | os.O_NOFOLLOW | os.O_NONBLOCK | os.O_CLOEXEC,
                 0o600, dir_fd=directory)
    try:
        info = os.fstat(fd)
        require(stat.S_ISREG(info.st_mode) and info.st_uid == os.getuid()
                and info.st_nlink == 1 and stat.S_IMODE(info.st_mode) == 0o600
                and info.st_size == 0, 'installer_lock_metadata')
        fcntl.flock(fd, fcntl.LOCK_EX | fcntl.LOCK_NB)
        yield
    finally:
        os.close(fd)


def reserve_attempt(directory, request, plan, receipt, image, inventory_sha):
    record = {'version': 1, 'kind': 'sql62_attempt_reserved', 'source_ref': SOURCE,
              'restore_receipt_sha256': request.approved_restore_receipt_sha256,
              'plan_sha256': plan['plan_sha'], 'package_sha256': plan['package_sha'],
              'lock_sha256': plan['lock_sha'], 'clon_sha256': inventory_sha,
              'pg_container_id': receipt['pg_container_id'], 'pg_image_id': image}
    data = clon_h6_kit.canonical(record)
    fd = os.open(ATTEMPT, os.O_WRONLY | os.O_CREAT | os.O_EXCL | os.O_NOFOLLOW | os.O_CLOEXEC,
                 0o600, dir_fd=directory)
    try:
        remaining = memoryview(data)
        while remaining:
            count = os.write(fd, remaining)
            require(count > 0, 'attempt_write_failed')
            remaining = remaining[count:]
        os.fsync(fd)
    finally:
        os.close(fd)
    os.fsync(directory)


def install(request, state, expected_image_id):
    """Apply only from absent journal and matching H1 preimage, once.

    No caller-selected factory, callback, command or approval object is accepted.
    DockerDB must accept immutable CID; its older constructor refuses before SQL.
    """
    require(isinstance(request, clon_h6_kit.Request) and request.ad132_request is None, 'request_contract')
    require(isinstance(expected_image_id, str)
            and re.fullmatch(r'sha256:[0-9a-f]{64}', expected_image_id) is not None, 'external_image_pin')
    state = canonical_path(state)
    for path in (request.package_tar, request.release_lock, request.h1_state_file,
                 request.git_repo, request.restore_receipt, request.normalizer_path, request.guiones_manifest):
        canonical_path(path)
    require(Path(request.restore_receipt).parent == state, 'restore_receipt_state')
    with private_state(state) as (directory, retained):
        fresh(directory)
        with installer_lock(directory):
            fresh(directory)
            receipt = clon_h6_kit.restore_receipt(request)
            require(isinstance(receipt.get('pg_container_id'), str)
                    and re.fullmatch(r'[0-9a-f]{64}', receipt['pg_container_id']) is not None,
                    'restore_immutable_cid')
            require(receipt['pg_image_id'] == expected_image_id, 'restore_image_mismatch')
            inventory_sha = metadata(directory, state, receipt, request)
            plan, rows = clon_sql.preflight_h6_package(
                request.package_tar, request.release_lock, request.approved_package_sha256,
                request.approved_lock_sha256, request.h1_state_file, request.approved_h1_sha256,
                source_ref=SOURCE, git_repo=request.git_repo)
            require(plan.get('source_ref') == SOURCE and plan.get('approved_sql_ref') == SOURCE
                    and plan.get('plan_family') == 'h6_package_62' and plan.get('file_count') == 62
                    and len(rows) == 62, 'package62_contract')
            context = {key: plan[key] for key in (*clon_sql.CONTEXT_KEYS[1:], 'lock_sha')}
            context['identidad_clon'] = request.approved_restore_receipt_sha256
            context = clon_sql.validate_context(context)
            kit = clon_h6_kit.H6Kit(request)
            clon_sql.require_kit(kit, plan, context)
            # No mutable Docker name fallback, even with the older CID constructor.
            db = clon_sql.DockerDB(receipt['pg_container_id'], state, expected_image_id=expected_image_id)
            db.check_owner()
            ro = clon_sql.ReadOnlyDB(db)
            require(kit.identity(ro, context) == context['identidad_clon'], 'restore_live_identity')
            require(kit.confirm(ro, None, 0, context) is True, 'restore_live_preimage')
            stable_state(state, retained)
            require(metadata(directory, state, receipt, request) == inventory_sha, 'clone_inventory_changed')
            fresh(directory)
            reserve_attempt(directory, request, plan, receipt, expected_image_id, inventory_sha)
            # Recheck only journal guards: the durable attempt is intentionally present.
            require(not any(present(directory, name) for name in BLOCKERS if name != ATTEMPT),
                    'sql_state_exists_new_clone_required')
            stable_state(state, retained)
            record = clon_sql.apply(db, rows, state, SOURCE, plan, context, kit=kit)
            clon_sql.validate_record(record, plan, context)
            clon_sql.validate_receipts(record.get('installed'), plan)
            require(record.get('phase') == 'awaiting_ad132' and record.get('pending') is None
                    and len(record.get('installed', [])) == 62
                    and not present(directory, '.sql-confirming'), 'sql62_confirmation_incomplete')
            raw = read_owned(directory, 'sql-journal.json', clon_sql.MAX_JOURNAL)
            require(json.loads(raw, object_pairs_hook=clon_h6_kit.unique) ==
                    {**record, 'journal_sha': clon_sql.record_hash(record)},
                    'sql62_journal_confirmation_mismatch')
            return {'journal_path': str(state / 'sql-journal.json'), 'journal_sha256': clon_sql.sha(raw),
                    'sql_count': 62, 'phase': 'awaiting_ad132'}


def status(state):
    """File presence only; no lock, Docker, live DB or inferred confirmation."""
    with private_state(state) as (directory, _):
        return 'state_present' if any(present(directory, name) for name in BLOCKERS) else 'journal_absent'


class Parser(argparse.ArgumentParser):
    def error(self, message):
        raise Refused('arguments_invalid')


def main(argv=None):
    parser = Parser(description=__doc__)
    commands = parser.add_subparsers(dest='action', required=True)
    inspect = commands.add_parser('status')
    inspect.add_argument('--state-dir', type=Path, required=True)
    apply = commands.add_parser('install')
    apply.add_argument('--state-dir', type=Path, required=True)
    apply.add_argument('--expected-pg-image-id', required=True)
    paths = ('package-tar', 'release-lock', 'h1-state-file', 'git-repo',
             'restore-receipt', 'normalizer-path', 'guiones-manifest')
    digests = ('package', 'lock', 'h1', 'restore-receipt', 'normalizer', 'guiones')
    for field in paths:
        apply.add_argument('--' + field, type=Path, required=True)
    for field in digests:
        apply.add_argument('--approved-' + field + '-sha256', required=True)
    args = parser.parse_args(argv)
    if args.action == 'status':
        print('SQL62 ' + status(args.state_dir))
        return
    request = clon_h6_kit.Request(**{field: getattr(args, field) for field in clon_h6_kit.Request.__dataclass_fields__
                                   if field != 'ad132_request'})
    print('SQL62 preflight')
    value = install(request, args.state_dir, args.expected_pg_image_id)
    print('SQL62 62/62 awaiting_ad132')
    print(value['journal_path'] + ' SHA256 ' + value['journal_sha256'])


if __name__ == '__main__':
    try:
        main()
    except Exception as error:
        # Never print provider/subprocess exception messages, SQL or private JSON.
        code = str(error) if isinstance(error, Refused) else 'installation_refused'
        print('SQL62-NO-GO ' + code + ' preserve_state_new_clone_required', file=sys.stderr)
        raise SystemExit(1)
