#!/usr/bin/env python3
"""Read-only composition contract for the local H6 clone.

This gate does not restore H1, execute providers, connect Docker/PostgreSQL,
create private state, approve AD132, repair profiles or publish READY.
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


def main(argv=None):
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('action', choices=('preflight', 'preparar', 'reiniciar'))
    parser.add_argument('--source-ref', default=SOURCE)
    args = parser.parse_args(argv)
    if not re.fullmatch('[0-9a-f]{40}', args.source_ref):
        raise Refused('source_not_canonical')
    value = composition(source_ref=args.source_ref)
    print(json.dumps(value, ensure_ascii=True, sort_keys=True, separators=(',', ':')))
    if args.action != 'preflight':
        require_complete(args.action, source_ref=args.source_ref)


if __name__ == '__main__':
    try:
        main()
    except Exception as error:
        code = str(error) if isinstance(error, Refused) else type(error).__name__
        print('H6-NO-GO ' + code, file=sys.stderr)
        raise SystemExit(1)
