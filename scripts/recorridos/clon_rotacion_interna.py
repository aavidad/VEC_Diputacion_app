#!/usr/bin/env python3
"""Archive an owned stopped projection and copy its data to the next source.

The caller creates the new projection through the normal material driver. This
helper never stops a process, changes operator material, or connects to SQL.
"""
import argparse
from contextlib import contextmanager
import ctypes
import errno
import fcntl
import hashlib
import importlib.util
import json
import os
from pathlib import Path
import re
import stat
import tempfile

OWNER = 'Codex-M'
KINDS = {'documentos', 'imagenes', 'data', 'comunicaciones'}
RECEIPT = 'rotacion-interna.json'


class RotationError(Exception):
    pass


def fail(code):
    raise RotationError(code)


def encoded(value):
    return (json.dumps(value, sort_keys=True, indent=2) + '\n').encode()


def digest(value):
    return hashlib.sha256(value).hexdigest()


def source(value):
    if not isinstance(value, str) or not re.fullmatch('[0-9a-f]{40}', value):
        fail('source_ref_invalid')
    return value


def path_check(path, directory=False):
    path = Path(path)
    if not path.is_absolute() or '..' in path.parts:
        fail('path_not_canonical')
    for part in (path, *path.parents):
        if part.is_symlink():
            fail('symlink_forbidden')
    info = path.lstat()
    if info.st_uid != os.getuid():
        fail('foreign_file_owner')
    if directory:
        if not stat.S_ISDIR(info.st_mode):
            fail('directory_required')
    elif not stat.S_ISREG(info.st_mode) or info.st_nlink != 1:
        fail('regular_single_link_file_required')
    return info


def read(path):
    path_check(path)
    fd = os.open(path, os.O_RDONLY | os.O_NOFOLLOW | os.O_NONBLOCK)
    with os.fdopen(fd, 'rb') as stream:
        info = os.fstat(stream.fileno())
        if not stat.S_ISREG(info.st_mode) or info.st_nlink != 1 or info.st_uid != os.getuid():
            fail('unsafe_open_file')
        return stream.read()


def private_json(path):
    if path_check(path).st_mode & 0o077:
        fail('private_metadata_required')
    value = json.loads(read(path))
    if not isinstance(value, dict):
        fail('metadata_object_required')
    return value


def write_cas(path, value, previous=None):
    if previous is None and (path.exists() or path.is_symlink()):
        fail('receipt_already_exists')
    if previous is not None and read(path) != previous:
        fail('receipt_preimage_changed')
    fd, name = tempfile.mkstemp(prefix='.rotation-', dir=path.parent)
    temporary = Path(name)
    try:
        with os.fdopen(fd, 'wb') as stream:
            stream.write(encoded(value))
            stream.flush()
            os.fsync(stream.fileno())
        if previous is None:
            rename_exclusive(temporary, path)
        else:
            if read(path) != previous:
                fail('receipt_preimage_changed')
            os.replace(temporary, path)
    finally:
        temporary.unlink(missing_ok=True)


def snapshot_file(path, data, workspace):
    """Publish a whole snapshot or leave its final name absent for recovery."""
    fd, name = tempfile.mkstemp(prefix='.rotation-metadata-', dir=workspace)
    temporary = Path(name)
    try:
        with os.fdopen(fd, 'wb') as stream:
            stream.write(data)
            stream.flush()
            os.fsync(stream.fileno())
        rename_exclusive(temporary, path)
    finally:
        temporary.unlink(missing_ok=True)


def rename_exclusive(old, new):
    """Linux same-filesystem rename, with no replacement or hard links."""
    libc = ctypes.CDLL(None, use_errno=True)
    result = libc.renameat2(-100, os.fsencode(old), -100, os.fsencode(new), 1)
    if result:
        number = ctypes.get_errno()
        raise OSError(number, 'exclusive_rename_failed')


def metadata(info, kind):
    return {'type': kind, 'mode': stat.S_IMODE(info.st_mode), 'uid': info.st_uid,
            'gid': info.st_gid, 'size': info.st_size if kind == 'file' else 0}


def file_info(path):
    info = path_check(path)
    h = hashlib.sha256()
    fd = os.open(path, os.O_RDONLY | os.O_NOFOLLOW | os.O_NONBLOCK)
    with os.fdopen(fd, 'rb') as stream:
        before = os.fstat(stream.fileno())
        if not stat.S_ISREG(before.st_mode) or before.st_nlink != 1 or before.st_uid != os.getuid():
            fail('unsafe_open_file')
        for block in iter(lambda: stream.read(1024 * 1024), b''):
            h.update(block)
        after = os.fstat(stream.fileno())
    if (info.st_ino, info.st_dev) != (before.st_ino, before.st_dev) or (before.st_ino, before.st_dev, before.st_size, before.st_mtime_ns, before.st_ctime_ns) != (after.st_ino, after.st_dev, after.st_size, after.st_mtime_ns, after.st_ctime_ns):
        fail('inventory_file_changed')
    return dict(metadata(after, 'file'), sha256=h.hexdigest())


def inventory(root):
    result = {}
    def visit(path):
        info = path_check(path, directory=path.is_dir())
        relative = str(path.relative_to(root))
        if stat.S_ISDIR(info.st_mode):
            result[relative] = metadata(info, 'dir')
            for child in sorted(path.iterdir()):
                visit(child)
        else:
            result[relative] = file_info(path)
    visit(root)
    return result


def runtime_module():
    spec = importlib.util.spec_from_file_location('rotation_runtime', Path(__file__).with_name('clon_runtime.py'))
    module = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(module)
    return module.container_module()


def stopped(state, expected, restoring=False, archived_history=None):
    """Reuse exact ownership guards; inspect only, never signal or delete."""
    module = runtime_module()
    if (state / 'runtime-container-intent.json').exists():
        fail('runtime_creation_intent_pending')
    for name in ('runtime-process.json', 'runtime-container.json'):
        path = state / name
        if not path.exists():
            continue
        record = private_json(path)
        module.verify_record_boundary(state, record)
        if record['source_commit'] != expected:
            fail('foreign_runtime_source')
        try:
            module.inspect(state, record['container_id'])
        except module.DockerNotFound:
            if module.process_identity(record['pid']) is not None:
                fail('saved_process_still_present')
        else:
            fail('runtime_container_still_present')
    # Catch even an interrupted launch without its final reservation. Other
    # containers (including PostgreSQL) are neither inspected nor modified.
    objects = module.docker(state, 'container', 'ls', '-aq', '--no-trunc', '--filter',
                            'label=' + module.PREFIX + 'state=' + str(state))
    if objects.strip():
        fail('runtime_object_still_present')
    if restoring:
        for path in state.glob('runtime-container-stopped-*.json'):
            record = private_json(path)
            if record.get('source_commit') == expected:
                module.verify_record_boundary(state, record)
                if archived_history is not None:
                    snapshot = archived_history / path.name
                    if snapshot.exists() and read(snapshot) == read(path):
                        continue
                fail('new_runtime_already_started')


def marker(root, expected):
    value = private_json(root / 'material-manifest.json')
    if value.get('owner') != OWNER or value.get('portal') != 'interno' or value.get('target', {}).get('source_commit') != expected:
        fail('projection_source_or_owner_mismatch')
    return value


def layout(state, root, expected):
    marker(root, expected)
    parent = private_json(state / 'material-manifest.json')
    descriptor = parent.get('runtime_interno', {})
    if descriptor.get('mode') != 'interno' or descriptor.get('portal') != 'interno' or descriptor.get('source_commit') != expected or descriptor.get('manifest_sha256') != digest(read(root / 'material-manifest.json')):
        fail('projection_descriptor_mismatch')
    result = {}
    for entry in descriptor.get('rw', []):
        kind, relative = entry.get('kind'), entry.get('source')
        if kind not in KINDS or kind in result or not isinstance(relative, str):
            fail('rw_inventory_invalid')
        absolute = state / relative
        allowed = [root / 'rw' / kind]
        if kind == 'comunicaciones':
            allowed.append(root / 'material/comunicaciones')
        target = root / 'material/comunicaciones' if kind == 'comunicaciones' else absolute
        if absolute not in allowed or entry.get('target') != str(target):
            fail('rw_reference_escapes_projection')
        path_check(absolute, directory=True)
        result[kind] = str(absolute.relative_to(root))
    if set(result) != KINDS:
        fail('four_rw_roots_required')
    return result


@contextmanager
def locked(state):
    state = Path(state)
    info = path_check(state, directory=True)
    if info.st_mode & 0o077 or any((p / '.git').exists() for p in (state, *state.parents)):
        fail('owned_private_state_required')
    clone = private_json(state / 'clon.json')
    if clone.get('propietario') != OWNER or clone.get('estado') != str(state):
        fail('foreign_clone_reference')
    if (state / 'READY.json').exists() or (state / 'READY.json').is_symlink():
        fail('ready_must_be_withdrawn')
    fd = os.open(state / 'runtime.lock', os.O_RDWR | os.O_CREAT | os.O_NOFOLLOW, 0o600)
    try:
        info = os.fstat(fd)
        if not stat.S_ISREG(info.st_mode) or info.st_nlink != 1 or info.st_uid != os.getuid() or info.st_mode & 0o077:
            fail('unsafe_runtime_lock')
        fcntl.flock(fd, fcntl.LOCK_EX | fcntl.LOCK_NB)
        yield state
    finally:
        os.close(fd)


def receipt(state):
    path = state / RECEIPT
    value = private_json(path)
    if value.get('version') != 1 or value.get('owner') != OWNER or value.get('state') != str(state):
        fail('foreign_rotation_receipt')
    source(value.get('old_source')); source(value.get('new_source'))
    kind = value.get('rotation_kind', 'source_advance')
    if kind not in ('source_advance', 'config_same_source') or ((value['old_source'] == value['new_source']) != (kind == 'config_same_source')):
        fail('rotation_kind_source_mismatch')
    name = value['old_source'] + '-' + value['old_manifest_sha256']
    if not re.fullmatch('[0-9a-f]{40}-[0-9a-f]{64}', name) or value.get('archive') != 'proyecciones/' + name:
        fail('archive_reference_invalid')
    return value


def summary(value):
    return {key: value[key] for key in ('phase', 'old_source', 'new_source', 'archive', 'old_manifest_sha256')}


def verify_archive(state, value):
    archived = state / value['archive']
    marker(archived, value['old_source'])
    if digest(read(archived / 'material-manifest.json')) != value['old_manifest_sha256'] or inventory(archived) != value['inventory']:
        fail('archived_projection_changed')
    if inventory(state / (value['archive'] + '.metadata')) != value['metadata_inventory']:
        fail('archived_metadata_changed')
    return archived


def archive(state, expected_old_source, new_source, allow_same_source=False,
            expected_old_manifest_sha256=None):
    old, new = source(expected_old_source), source(new_source)
    same = old == new
    if same and not allow_same_source:
        fail('source_must_advance')
    if same and (not isinstance(expected_old_manifest_sha256, str) or not re.fullmatch('[0-9a-f]{64}', expected_old_manifest_sha256)):
        fail('same_source_manifest_preimage_required')
    if not same and (allow_same_source or expected_old_manifest_sha256 is not None):
        fail('same_source_opt_in_requires_equal_source')
    with locked(state) as state:
        root, pointer = state / 'runtime-interno', state / RECEIPT
        previous = read(pointer) if pointer.exists() else None
        value = receipt(state) if previous is not None else None
        matches = value is not None and (value['old_source'], value['new_source']) == (old, new) and (not same or value['old_manifest_sha256'] == expected_old_manifest_sha256)
        if value is not None and not matches:
            if value['phase'] != 'restored' or value['new_source'] != old:
                fail('another_rotation_pending')
            historical = state / (value['archive'] + '.receipt.json')
            if historical.exists():
                if read(historical) != previous:
                    fail('historical_receipt_changed')
            else:
                write_cas(historical, value)
            value = None
        if value is None:
            stopped(state, old)
            rw = layout(state, root, old)
            full = inventory(root)
            manifest_sha = digest(read(root / 'material-manifest.json'))
            if same and manifest_sha != expected_old_manifest_sha256:
                fail('same_source_manifest_preimage_changed')
            parent = state / 'proyecciones'
            parent.mkdir(mode=0o700, exist_ok=True)
            path_check(parent, directory=True)
            if parent.stat().st_mode & 0o077 or parent.stat().st_dev != root.stat().st_dev:
                fail('private_same_filesystem_archive_required')
            name = old + '-' + manifest_sha
            destination = parent / name
            if destination.exists() or destination.is_symlink():
                fail('archive_destination_occupied')
            snapshots = parent / (name + '.metadata')
            snapshots.mkdir(mode=0o700, exist_ok=True)
            path_check(snapshots, directory=True)
            if snapshots.stat().st_mode & 0o077:
                fail('private_metadata_directory_required')
            references = ['clon.json', 'material-manifest.json', 'runtime-manifest.json', 'runtime-process.json', 'runtime-container.json', 'perfiles.json']
            references += [p.name for p in state.glob('runtime-container-stopped-*.json')]
            expected_references = {ref for ref in references if (state / ref).exists()}
            if any(p.name not in expected_references for p in snapshots.iterdir()):
                fail('foreign_metadata_snapshot_entry')
            for ref in sorted(set(references)):
                original = state / ref
                if original.exists():
                    if ref.startswith('runtime-container-stopped-'):
                        runtime_module().verify_record_boundary(state, private_json(original))
                    data = read(original)
                    if (snapshots / ref).exists():
                        if read(snapshots / ref) != data:
                            fail('metadata_snapshot_preimage_changed')
                        continue
                    snapshot_file(snapshots / ref, data, parent)
            value = {'version': 1, 'owner': OWNER, 'state': str(state), 'phase': 'archive_pending',
                     'rotation_kind': 'config_same_source' if same else 'source_advance',
                     'old_source': old, 'new_source': new, 'archive': 'proyecciones/' + name,
                     'old_manifest_sha256': manifest_sha, 'inventory': full, 'rw': rw,
                     'metadata_inventory': inventory(snapshots)}
            write_cas(pointer, value, previous)
        if value['phase'] != 'archive_pending':
            verify_archive(state, value)
            return summary(value)
        stopped(state, old)
        destination = state / value['archive']
        if root.exists():
            marker(root, old)
            if inventory(root) != value['inventory'] or destination.exists():
                fail('archive_source_preimage_changed')
            rename_exclusive(root, destination)
        verify_archive(state, value)
        before = read(pointer)
        value['phase'] = 'archived'
        write_cas(pointer, value, before)
        return summary(value)


def copy_file(old, new, info, work):
    """Independent streaming copy; rename cannot replace a concurrent file."""
    if file_info(old) != info:
        fail('archive_copy_preimage_changed')
    fd, name = tempfile.mkstemp(prefix='.rotation-copy-', dir=work)
    temporary = Path(name)
    try:
        original_fd = os.open(old, os.O_RDONLY | os.O_NOFOLLOW | os.O_NONBLOCK)
        with os.fdopen(fd, 'wb') as target, os.fdopen(original_fd, 'rb') as original:
            for block in iter(lambda: original.read(1024 * 1024), b''):
                target.write(block)
            target.flush()
            os.fchown(target.fileno(), info['uid'], info['gid'])
            os.fchmod(target.fileno(), info['mode'])
            os.fsync(target.fileno())
        if file_info(temporary) != info or file_info(old) != info:
            fail('copied_file_does_not_match')
        rename_exclusive(temporary, new)
    finally:
        temporary.unlink(missing_ok=True)


def copy_directory(new, info, work):
    temporary = Path(tempfile.mkdtemp(prefix='.rotation-dir-', dir=work))
    try:
        os.chown(temporary, info['uid'], info['gid'])
        temporary.chmod(info['mode'] | 0o700)
        rename_exclusive(temporary, new)
    finally:
        if temporary.exists():
            temporary.rmdir()


def restore(state, expected_new_source):
    new = source(expected_new_source)
    with locked(state) as state:
        value, root = receipt(state), state / 'runtime-interno'
        if value['new_source'] != new or value['phase'] not in ('archived', 'restore_pending', 'restored'):
            fail('rotation_target_or_phase_mismatch')
        original = verify_archive(state, value)
        same = value.get('rotation_kind') == 'config_same_source'
        new_manifest_sha = digest(read(root / 'material-manifest.json')) if same else None
        if same and new_manifest_sha == value['old_manifest_sha256']:
            fail('same_source_projection_must_change')
        if same and 'new_manifest_sha256' in value and value['new_manifest_sha256'] != new_manifest_sha:
            fail('same_source_target_manifest_changed')
        history = state / (value['archive'] + '.metadata') if same else None
        stopped(state, new, restoring=True, archived_history=history)
        rw = layout(state, root, new)
        if rw != value['rw']:
            fail('rw_layout_changed')
        pending = []
        directories = {}
        for relative in rw.values():
            expected = {p: entry for p, entry in value['inventory'].items() if p == relative or p.startswith(relative + '/')}
            directories.update({p: info['mode'] for p, info in expected.items()
                                if info['type'] == 'dir' and info['mode'] != info['mode'] | 0o700})
            current = inventory(root / relative)
            current = {relative if p == '.' else relative + '/' + p: entry for p, entry in current.items()}
            for p, info in current.items():
                compared = dict(info)
                if value['phase'] == 'restore_pending' and p in value.get('directory_modes', {}) and info['mode'] == value['directory_modes'][p] | 0o700:
                    compared['mode'] = value['directory_modes'][p]
                if p not in expected or expected[p] != compared:
                    fail('new_runtime_data_changed')
            pending += [(p, info) for p, info in expected.items() if p not in current]
        if 'directory_modes' in value and value['directory_modes'] != directories:
            fail('temporary_directory_modes_receipt_changed')
        if value['phase'] == 'restored':
            if pending:
                fail('restored_data_missing_no_retransfer')
            return summary(value)
        before = read(state / RECEIPT)
        value['phase'] = 'restore_pending'
        if same:
            value['new_manifest_sha256'] = new_manifest_sha
        value['directory_modes'] = directories
        write_cas(state / RECEIPT, value, before)
        work = state / (value['archive'] + '.copy-work')
        work.mkdir(mode=0o700, exist_ok=True)
        path_check(work, directory=True)
        if work.stat().st_mode & 0o077 or work.stat().st_dev != root.stat().st_dev:
            fail('private_same_filesystem_copy_required')
        for relative, mode in sorted(directories.items(), key=lambda item: len(Path(item[0]).parts)):
            target = root / relative
            if target.exists():
                target.chmod(mode | 0o700)
        for relative, info in sorted(pending, key=lambda item: (len(Path(item[0]).parts), item[0])):
            target = root / relative
            if info['type'] == 'dir':
                copy_directory(target, info, work)
            else:
                copy_file(original / relative, target, info, work)
        for relative, mode in sorted(directories.items(), key=lambda item: len(Path(item[0]).parts), reverse=True):
            (root / relative).chmod(mode)
        for relative in rw.values():
            if inventory(root / relative) != inventory(original / relative):
                fail('restored_inventory_mismatch')
        before = read(state / RECEIPT)
        value['phase'] = 'restored'
        value['restored_inventory_sha256'] = digest(encoded({p: entry for p, entry in value['inventory'].items() if any(p == rel or p.startswith(rel + '/') for rel in rw.values())}))
        write_cas(state / RECEIPT, value, before)
        return summary(value)


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('action', choices=('archive', 'restore'))
    parser.add_argument('--state', required=True, type=Path)
    parser.add_argument('--old-source')
    parser.add_argument('--new-source', required=True)
    parser.add_argument('--allow-same-source', action='store_true')
    parser.add_argument('--expected-old-manifest-sha256')
    args = parser.parse_args()
    if args.action == 'restore' and (args.allow_same_source or args.expected_old_manifest_sha256 is not None):
        fail('archive_only_opt_in')
    result = archive(args.state, args.old_source, args.new_source, args.allow_same_source, args.expected_old_manifest_sha256) if args.action == 'archive' else restore(args.state, args.new_source)
    print(json.dumps(result, sort_keys=True))


if __name__ == '__main__':
    try:
        main()
    except (RotationError, OSError, ValueError, KeyError) as error:
        print(json.dumps({'error': str(error) if isinstance(error, RotationError) else 'rotation_local_failure'}))
        raise SystemExit(2)
