#!/usr/bin/env python3
"""H6 composition and one-shot H1/SQL62 phase for a private local clone.

Full preparation remains blocked. preparar-sql restores H1 and installs SQL62
only in an empty private state. verificar-sql is read-only and requires the
externally retained phase receipt digest. Neither action approves AD132 or READY.
The concrete operation order is recorded here so absent transport, preview,
material and runtime authorities cannot silently become executable adapters.
API presence is documentary evidence only; it never means reviewed or approved.
"""
from __future__ import annotations

import argparse
import ast
from dataclasses import dataclass
import json
import os
from pathlib import Path
import re
import stat
import sys

SOURCE = '73e56c106d12fdda0bd16d6fe573503c42c5495f'
MAX_SOURCE = 256 * 1024
PLAN_INPUTS = ('package_tar', 'release_lock', 'approved_package_sha256',
               'approved_lock_sha256', 'h1_state_file', 'approved_h1_sha256')


class Refused(RuntimeError):
    pass


@dataclass(frozen=True)
class Operation:
    id: str
    api: str
    effect: str
    prerequisite: str
    recovery: str


# No operation is implemented by a caller-selected callback, module or command.
OPERATIONS = (
    Operation('h1', 'clon_h1_restore.restore', 'fresh_restore',
              'external_h1_sha_and_clean_shutdown', 'new_state_only'),
    Operation('sql62', 'clon_sql.apply', 'original_sql_bytes',
              'H6Kit_preimage_H3_8_H4_9_H6_45', 'never_retry'),
    Operation('material_pre_ad132', 'clon_material.prepare_staged', 'private_files',
              'definitive_material_already_complete_no_profiles', 'preserve'),
    Operation('canary', 'clon_h6_archive.import_inputs/export_output', 'offline_uid10002',
              'reviewed_controller_pull_never_no_network_fixed_socket', 'new_state_only'),
    Operation('preview_ad132', 'D_cli_preview', 'read_only',
              'pinned_original_D_cli_plan_and_receipt', 'read_only'),
    Operation('approval_ad132', 'external_approval', 'none',
              'operator_approval_of_exact_preview', 'never_synthesize'),
    Operation('apply_ad132', 'clon_ad132_recibo.apply_and_confirm', 'D_cli_once',
              'external_approval_and_durable_pending', 'revalidar_only'),
    Operation('db_ready', 'clon_h6_ready.complete_h6', 'journal_and_receipt',
              'live_AD132_original_receipt_and_62_confirmations', 'read_only_then_same_receipt'),
    Operation('material_final', 'clon_material.prepare_staged', 'read_only',
              'DB_READY_and_same_definitive_material', 'read_only'),
    Operation('runtime', 'clon_runtime.start', 'own_runtime',
              'reviewed_PG_namespace_relay_and_approved_binary_source', 'no_SQL'),
)

# These are unresolved composition boundaries, not guessed provider APIs.
# Remove a blocker only in a reviewed change that installs its concrete adapter.
COMPOSITION_BLOCKERS = (
    'fresh_definitive_material_authority_missing',
    'archive_Docker_controller_and_canonical_pair_not_connected',
    'pinned_D_preview_and_external_approval_handoff_not_connected',
    'PG_namespace_runtime_and_pinned_loopback_relay_not_connected',
    'H1_ownership_restart_and_retirement_controller_not_connected',
    'whole_composition_independent_reviews_pending',
)

API_REQUIREMENTS = {
    'clon_h1_restore.py': ('restore',),
    'clon_h6_kit.py': ('Request', 'H6Kit.validate', 'H6Kit.identity',
                       'H6Kit.confirm', 'H6Kit.verify'),
    'clon_sql.py': ('preflight_h6_package', 'DockerDB', 'ReadOnlyDB', 'apply'),
    'clon_material.py': ('prepare_staged', 'reject_staged_mutations'),
    'clon_h6_archive.py': ('import_inputs', 'export_output'),
    'clon_ad132_recibo.py': ('Request', 'apply_and_confirm', 'revalidar'),
    'clon_h6_ready.py': ('load_approval', 'validate_pre_ad132',
                         'complete_h6', 'validate_h6_ready'),
    'clon_runtime.py': ('load_readiness_approval', 'validate_database_ready', 'start'),
}


def source_api(path):
    """Inspect syntax without importing or executing source/provider code."""
    fd = os.open(path, os.O_RDONLY | os.O_NOFOLLOW | os.O_NONBLOCK | os.O_CLOEXEC)
    try:
        info = os.fstat(fd)
        if not (stat.S_ISREG(info.st_mode) and info.st_nlink == 1 and
                info.st_uid in (0, os.getuid()) and not info.st_mode & 0o022 and
                0 < info.st_size <= MAX_SOURCE):
            raise Refused('source_metadata')
        data = os.read(fd, MAX_SOURCE + 1)
        if len(data) != info.st_size or os.read(fd, 1):
            raise Refused('source_changed')
        after = os.fstat(fd)
        if (info.st_dev, info.st_ino, info.st_size, info.st_mtime_ns, info.st_ctime_ns) != (
                after.st_dev, after.st_ino, after.st_size, after.st_mtime_ns, after.st_ctime_ns):
            raise Refused('source_changed')
        tree = ast.parse(data, filename=path.name)
        names = set()
        for node in tree.body:
            if isinstance(node, (ast.FunctionDef, ast.AsyncFunctionDef, ast.ClassDef)):
                names.add(node.name)
            if isinstance(node, ast.ClassDef):
                names.update(node.name + '.' + child.name for child in node.body
                             if isinstance(child, (ast.FunctionDef, ast.AsyncFunctionDef)))
        return names
    finally:
        os.close(fd)


def composition(scripts=None, source_ref=SOURCE):
    scripts = Path(scripts or Path(__file__).parent)
    if source_ref != SOURCE:
        raise Refused('source_not_approved_H6_SQL')
    apis, blockers = {}, list(COMPOSITION_BLOCKERS)
    for name, required in API_REQUIREMENTS.items():
        try:
            available = source_api(scripts / name)
        except (OSError, SyntaxError, UnicodeError, Refused):
            available = set()
        apis[name] = {key: key in available for key in required}
        if not all(apis[name].values()):
            blockers.append('api_missing:' + name)
    return {'version': 1, 'kind': 'h6_composition_preflight',
            'source_sql': SOURCE, 'sql_count': 62, 'ad132_separate': True,
            'plan_status': 'external_inputs_required', 'plan_inputs': list(PLAN_INPUTS),
            'postgres_transport': {'network': 'none', 'logical_port': 5432,
                'host_published': False, 'app_network': 'container:<PGID>',
                'host_app_access': 'pinned_loopback_exec_relay'},
            'ready': False, 'executable': False, 'apis': apis,
            'blockers': blockers,
            'operations': [dict(id=op.id, api=op.api, effect=op.effect,
                                prerequisite=op.prerequisite, recovery=op.recovery)
                           for op in OPERATIONS]}


def require_complete(action, scripts=None, source_ref=SOURCE):
    if action not in ('preparar', 'reiniciar'):
        raise Refused('unknown_action')
    # No environment flag, legacy READY, private JSON, API presence or caller
    # callback can remove an outstanding operational/review blocker.
    value = composition(scripts, source_ref)
    raise Refused('composition_incomplete:' + ','.join(value['blockers']))


# Imported only for the explicit SQL phase; preflight composition stays documentary.
def phase_apis():
    try:
        from . import clon_h1_restore, clon_h6_sql_instalar, clon_h6_kit, clon_sql
    except ImportError:
        import clon_h1_restore, clon_h6_sql_instalar, clon_h6_kit, clon_sql
    return clon_h1_restore, clon_h6_sql_instalar, clon_h6_kit, clon_sql


@dataclass(frozen=True)
class SQLRequest:
    package_tar: Path
    release_lock: Path
    approved_package_sha256: str
    approved_lock_sha256: str
    h1_state_file: Path
    approved_h1_sha256: str
    git_repo: Path
    normalizer_path: Path
    approved_normalizer_sha256: str
    guiones_manifest: Path
    approved_guiones_sha256: str
    expected_pg_image_id: str


PHASE_ATTEMPT = 'sql62-preparar-intento.json'
PHASE_RECEIPT = 'sql62-fase.json'


def phase_preflight(request, source_ref=SOURCE):
    """Freeze original SQL bytes and external pins before any restore effect."""
    h1, installer, kit, sql = phase_apis()
    installer.require(isinstance(request, SQLRequest) and source_ref == SOURCE, 'sql_phase_contract')
    for key in ('package_tar', 'release_lock', 'h1_state_file', 'git_repo',
                'normalizer_path', 'guiones_manifest'):
        installer.canonical_path(getattr(request, key))
    installer.require(request.approved_h1_sha256 == h1.H1_SHA
                      and request.expected_pg_image_id == h1.IMAGE_ID, 'restore_external_pins')
    plan, rows = sql.preflight_h6_package(request.package_tar, request.release_lock,
        request.approved_package_sha256, request.approved_lock_sha256,
        request.h1_state_file, request.approved_h1_sha256,
        source_ref=source_ref, git_repo=request.git_repo)
    installer.require(plan.get('file_count') == 62 and len(rows) == 62
                      and plan.get('plan_family') == sql.H6_PACKAGE_FAMILY, 'package62_contract')
    sql.approved_file(request.normalizer_path, request.approved_normalizer_sha256, 64000)
    scripts = sql.approved_file(request.guiones_manifest, request.approved_guiones_sha256, 64000)
    lock = sql.approved_file(request.release_lock, request.approved_lock_sha256, 1024 * 1024)
    lock_values = dict(line.split() for line in lock.decode('ascii').splitlines())
    listed = {}
    for line in scripts.decode('ascii').splitlines():
        parts = line.split()
        installer.require(len(parts) == 2 and parts[1] not in listed
                          and re.fullmatch('[0-9a-f]{64}', parts[0]) is not None,
                          'guiones_manifest_contract')
        listed[parts[1]] = parts[0]
    installer.require(lock_values.get('KIT_GUIONES_SHA256') == request.approved_guiones_sha256
                      and request.normalizer_path.name == 'h6_normalizar_pg_dump.py'
                      and listed.get(request.normalizer_path.name) == request.approved_normalizer_sha256,
                      'guiones_normalizer_pins')
    return plan, rows


def phase_pins(request):
    return {key: getattr(request, key) for key in SQLRequest.__dataclass_fields__
            if key.startswith('approved_') or key == 'expected_pg_image_id'}


def phase_publish(directory, name, value):
    """Exclusive fsynced evidence; an incomplete write also prevents retry."""
    _, installer, kit, _ = phase_apis()
    fd = os.open(name, os.O_WRONLY | os.O_CREAT | os.O_EXCL | os.O_NOFOLLOW | os.O_CLOEXEC,
                 0o600, dir_fd=directory)
    try:
        data = memoryview(kit.canonical(value))
        while data:
            count = os.write(fd, data)
            installer.require(count > 0, 'phase_write_failed')
            data = data[count:]
        os.fsync(fd)
    finally:
        os.close(fd)
    os.fsync(directory)


def phase_request(request, state, restore_sha):
    _, _, kit, _ = phase_apis()
    fields = {key: getattr(request, key) for key in SQLRequest.__dataclass_fields__
              if key != 'expected_pg_image_id'}
    return kit.Request(**fields, restore_receipt=state / 'h1-restore.json',
                       approved_restore_receipt_sha256=restore_sha)


def phase_postimage(db, request):
    _, installer, _, sql = phase_apis()
    ro = sql.ReadOnlyDB(db)
    before = ro.system_identity()
    post = {'schema_sha': ro.schema_digest(request.normalizer_path, request.approved_normalizer_sha256),
            'roles_sha': ro.roles_digest(request.normalizer_path, request.approved_normalizer_sha256),
            'datacl_sha': ro.database_acl_digest()}
    installer.require(before == ro.system_identity()
                      and all(isinstance(v, str) and re.fullmatch('[0-9a-f]{64}', v)
                              for v in post.values()), 'postimage_identity_or_digest')
    return before, post


def phase_files(directory, request, state, plan, restore_sha):
    """Read existing evidence without opening Journal (which creates a lock)."""
    _, installer, kit, sql = phase_apis()
    req = phase_request(request, state, restore_sha)
    receipt = kit.restore_receipt(req)
    installer.require(receipt['pg_image_id'] == request.expected_pg_image_id, 'phase_image')
    h1_records_sha = installer.h1_records(directory, receipt, req)
    volume = json.loads(installer.read_owned(directory, 'h1-volume.json'), object_pairs_hook=kit.unique)
    info = Path(volume['path']).lstat()
    installer.require(stat.S_ISDIR(info.st_mode)
                      and (info.st_dev, info.st_ino) == (volume['device'], volume['inode']), 'phase_volume')
    installer.require(not installer.present(directory, '.sql-confirming'), 'sql_uncertain_new_state_required')
    raw = installer.read_owned(directory, 'sql-journal.json', sql.MAX_JOURNAL)
    record = json.loads(raw, object_pairs_hook=kit.unique)
    context = {key: plan[key] for key in (*sql.CONTEXT_KEYS[1:], 'lock_sha')}
    context['identidad_clon'] = restore_sha
    installer.require(record.get('journal_sha') == sql.record_hash(record), 'phase_journal_hash')
    sql.validate_record(record, plan, context)
    sql.validate_receipts(record.get('installed'), plan)
    installer.require(record.get('phase') == 'awaiting_ad132' and record.get('pending') is None,
                      'phase_awaiting_ad132_required')
    expected_attempt = {'version': 1, 'kind': 'sql62_attempt_reserved', 'source_ref': SOURCE,
        'restore_receipt_sha256': restore_sha, 'plan_sha256': plan['plan_sha'],
        'package_sha256': plan['package_sha'], 'lock_sha256': plan['lock_sha'],
        'h1_records_sha256': h1_records_sha, 'pg_container_id': receipt['pg_container_id'],
        'pg_image_id': request.expected_pg_image_id}
    installer.require(installer.read_owned(directory, installer.ATTEMPT) == kit.canonical(expected_attempt),
                      'phase_sql_attempt')
    return req, receipt, context, sql.sha(raw), h1_records_sha


def preparar_sql(request, state, source_ref=SOURCE):
    """One-shot fresh H1 -> SQL62 -> observed postimage; preserve every failure."""
    h1, installer, kit, sql = phase_apis()
    plan, _ = phase_preflight(request, source_ref)
    state = installer.canonical_path(state)
    with installer.private_state(state) as (directory, retained):
        installer.require(not os.listdir(directory), 'phase_state_exists_new_state_required')
        attempt = {'version': 1, 'kind': 'sql62_phase_reserved', 'source_ref': SOURCE,
                   'state': str(state), 'plan_sha256': plan['plan_sha'], 'pins': phase_pins(request)}
        phase_publish(directory, PHASE_ATTEMPT, attempt)
        installer.stable_state(state, retained)
        observed = h1.restore(state, request.h1_state_file,
                              request.normalizer_path, request.approved_normalizer_sha256)
        # This SHA is an observed output of this exclusive restore, never an input approval.
        raw = installer.read_owned(directory, 'h1-restore.json')
        installer.require(raw == kit.canonical(observed), 'phase_restore_output')
        restore_sha = sql.sha(raw)
        req = phase_request(request, state, restore_sha)
        installed = installer.install(req, state, request.expected_pg_image_id)
        req, receipt, context, journal_sha, records_sha = phase_files(
            directory, request, state, plan, restore_sha)
        installer.require(installed.get('journal_sha256') == journal_sha
                          and installed.get('phase') == 'awaiting_ad132', 'phase_install_output')
        provider = kit.H6Kit(req)
        sql.require_kit(provider, plan, context)
        db = sql.DockerDB(receipt['pg_container_id'], state, expected_image_id=request.expected_pg_image_id)
        db.check_owner()
        installer.require(provider.identity(sql.ReadOnlyDB(db), context) == restore_sha, 'phase_live_identity')
        identity, postimage = phase_postimage(db, request)
        installer.require(identity == {key: receipt[key] for key in kit.IDENTITY_FIELDS}, 'phase_post_identity')
        installer.stable_state(state, retained)
        installer.require(phase_files(directory, request, state, plan, restore_sha)[3:] ==
                          (journal_sha, records_sha), 'phase_evidence_changed')
        value = {'version': 1, 'kind': 'sql62_phase_observed', 'source_ref': SOURCE,
            'state': str(state), 'pins': phase_pins(request), 'plan_sha256': plan['plan_sha'],
            'restore_receipt_sha256': restore_sha, 'h1_records_sha256': records_sha,
            'journal_sha256': journal_sha, 'phase_attempt_sha256': sql.sha(kit.canonical(attempt)),
            'identity': identity, 'postimage': postimage, 'phase': 'awaiting_ad132',
            'sql_count': 62, 'ready': False}
        phase_publish(directory, PHASE_RECEIPT, value)
        return {'phase': 'awaiting_ad132', 'sql_count': 62, 'ready': False,
                'acta_sha256': sql.sha(kit.canonical(value)),
                'restore_receipt_sha256': restore_sha, 'journal_sha256': journal_sha}


def verificar_sql(request, state, approved_acta_sha256, source_ref=SOURCE):
    """Compare a pinned phase acta and live postimage; no locks or file writes."""
    _, installer, kit, sql = phase_apis()
    installer.require(isinstance(approved_acta_sha256, str)
                      and re.fullmatch('[0-9a-f]{64}', approved_acta_sha256) is not None,
                      'external_phase_receipt_pin_required')
    plan, _ = phase_preflight(request, source_ref)
    state = installer.canonical_path(state)
    with installer.private_state(state) as (directory, retained):
        raw = installer.read_owned(directory, PHASE_RECEIPT)
        value = json.loads(raw, object_pairs_hook=kit.unique)
        fields = {'version', 'kind', 'source_ref', 'state', 'pins', 'plan_sha256',
                  'restore_receipt_sha256', 'h1_records_sha256', 'journal_sha256',
                  'phase_attempt_sha256', 'identity', 'postimage', 'phase', 'sql_count', 'ready'}
        installer.require(isinstance(value, dict) and set(value) == fields
            and raw == kit.canonical(value) and sql.sha(raw) == approved_acta_sha256
            and type(value['version']) is int and value['version'] == 1
            and value['kind'] == 'sql62_phase_observed' and value['source_ref'] == SOURCE
            and value['state'] == str(state) and value['pins'] == phase_pins(request)
            and value['plan_sha256'] == plan['plan_sha'] and value['phase'] == 'awaiting_ad132'
            and type(value['sql_count']) is int and value['sql_count'] == 62
            and value['ready'] is False, 'phase_receipt_contract')
        attempt = {'version': 1, 'kind': 'sql62_phase_reserved', 'source_ref': SOURCE,
                   'state': str(state), 'plan_sha256': plan['plan_sha'], 'pins': phase_pins(request)}
        installer.require(installer.read_owned(directory, PHASE_ATTEMPT) == kit.canonical(attempt)
                          and sql.sha(kit.canonical(attempt)) == value['phase_attempt_sha256'], 'phase_attempt')
        req, receipt, context, journal_sha, records_sha = phase_files(
            directory, request, state, plan, value['restore_receipt_sha256'])
        installer.require(journal_sha == value['journal_sha256']
                          and records_sha == value['h1_records_sha256'], 'phase_journal_or_h1_changed')
        provider = kit.H6Kit(req)
        sql.require_kit(provider, plan, context)
        db = sql.DockerDB(receipt['pg_container_id'], state, expected_image_id=request.expected_pg_image_id)
        db.check_owner()
        installer.require(provider.identity(sql.ReadOnlyDB(db), context) == context['identidad_clon'], 'phase_live_identity')
        identity, postimage = phase_postimage(db, request)
        installer.require(identity == value['identity'] == {key: receipt[key] for key in kit.IDENTITY_FIELDS}
                          and postimage == value['postimage'], 'phase_postimage_changed')
        installer.stable_state(state, retained)
        installer.require(phase_files(directory, request, state, plan, value['restore_receipt_sha256'])[3:]
                          == (journal_sha, records_sha)
                          and installer.read_owned(directory, PHASE_RECEIPT) == raw, 'phase_evidence_changed')
        return {'phase': 'awaiting_ad132', 'sql_count': 62, 'ready': False,
                'acta_sha256': approved_acta_sha256, 'verified': True}


class Parser(argparse.ArgumentParser):
    def error(self, message):
        raise Refused('arguments_invalid_external_inputs_required:' + ','.join(PLAN_INPUTS))


def main(argv=None):
    parser = Parser(description=__doc__)
    commands = parser.add_subparsers(dest='action', required=True)
    for action in ('preflight', 'preparar', 'reiniciar'):
        command = commands.add_parser(action)
        command.add_argument('--source-ref', default=SOURCE)
    for action in ('plan', 'preparar-sql', 'verificar-sql'):
        command = commands.add_parser(action)
        command.add_argument('--source-ref', default=SOURCE)
        for key in SQLRequest.__dataclass_fields__:
            command.add_argument('--' + key.replace('_', '-'),
                                 type=Path if key.endswith(('_tar', '_lock', '_file', '_repo', '_path', '_manifest')) else str,
                                 required=True)
        if action != 'plan':
            command.add_argument('--state-dir', type=Path, required=True)
        if action == 'verificar-sql':
            command.add_argument('--approved-acta-sha256', required=True)
    args = parser.parse_args(argv)
    if not re.fullmatch('[0-9a-f]{40}', args.source_ref):
        raise Refused('source_not_canonical')
    if args.action in ('plan', 'preparar-sql', 'verificar-sql'):
        request = SQLRequest(**{key: getattr(args, key) for key in SQLRequest.__dataclass_fields__})
        if args.action == 'plan':
            value, _ = phase_preflight(request, args.source_ref)
        elif args.action == 'preparar-sql':
            value = preparar_sql(request, args.state_dir, args.source_ref)
        else:
            value = verificar_sql(request, args.state_dir, args.approved_acta_sha256, args.source_ref)
        print(json.dumps(value, ensure_ascii=True, sort_keys=True, separators=(',', ':')))
        return
    value = composition(source_ref=args.source_ref)
    print(json.dumps(value, ensure_ascii=True, sort_keys=True, separators=(',', ':')))
    if args.action != 'preflight':
        require_complete(args.action, source_ref=args.source_ref)


if __name__ == '__main__':
    try:
        main()
    except Exception as error:
        code = str(error) if isinstance(error, Refused) else 'sql_phase_refused_preserve_state_new_state_required'
        print('H6-NO-GO ' + code, file=sys.stderr)
        raise SystemExit(1)
