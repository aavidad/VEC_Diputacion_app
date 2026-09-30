#!/usr/bin/env python3
"""Plan documental H6+AD132 hacia un commit explícito de origin/main.

Request recibe las huellas esperadas de la lista y las seis SQL en orden causal.
Sólo lee objetos Git locales: nunca lee recibos, aplica SQL ni consulta servicios.
Las huellas comprueban integridad; el resultado siempre necesita aprobación.
La ejecución futura conserva H6 histórico y emite un segundo recibo postmain.
"""
from __future__ import annotations

import argparse
from dataclasses import dataclass
import hashlib
import json
from pathlib import Path
import re
import subprocess
import sys

SOURCE = "73e56c106d12fdda0bd16d6fe573503c42c5495f"
LIST = "deploy/principal/lista_sql_trabajo_codexd_rpt_escritura_v3_20260930.txt"
CAT = "deploy/postgresql/catalogos_configurables/"
AD = "deploy/postgresql/autorizacion_atestada_v3/"
SQL_PATHS = (
    CAT + "roles_up.sql",
    CAT + "migraciones/000001_autoridad_categorias.up.sql",
    CAT + "migraciones/000002_lecturas_nominales.up.sql",
    AD + "migraciones/000117_lecturas_categorias_rpt.up.sql",
    CAT + "migraciones/000003_usos_nominales.up.sql",
    AD + "migraciones/000126_usos_categorias_rpt.up.sql",
)
# Cada acompañante tiene un vínculo nominal cerrado con su UP. Nunca se ejecuta.
COMPANIONS = {
    CAT + "roles_down.sql": SQL_PATHS[0],
    CAT + "pruebas_sql/roles_up_acl_public.sql": SQL_PATHS[0],
    CAT + "migraciones/000001_autoridad_categorias.down.sql": SQL_PATHS[1],
    CAT + "pruebas_sql/000001_autoridad_categorias.sql": SQL_PATHS[1],
    CAT + "migraciones/000002_lecturas_nominales.down.sql": SQL_PATHS[2],
    CAT + "pruebas_sql/000002_lecturas_nominales.sql": SQL_PATHS[2],
    AD + "pruebas_sql/ad3_117_lecturas_categorias_rpt.sql": SQL_PATHS[3],
    CAT + "migraciones/000003_usos_nominales.down.sql": SQL_PATHS[4],
    CAT + "pruebas_sql/000003_usos_nominales.sql": SQL_PATHS[4],
    AD + "migraciones/000126_usos_categorias_rpt.down.sql": SQL_PATHS[5],
    AD + "pruebas_sql/000126_usos_categorias_rpt.sql": SQL_PATHS[5],
}
HEX40 = re.compile(r"[0-9a-f]{40}\Z")
HEX64 = re.compile(r"[0-9a-f]{64}\Z")
LIMIT = 4 * 1024 * 1024
GIT_ENV = {"PATH": "/usr/bin:/bin", "HOME": "/nonexistent", "LC_ALL": "C",
           "GIT_CONFIG_NOSYSTEM": "1", "GIT_CONFIG_GLOBAL": "/dev/null",
           "GIT_OPTIONAL_LOCKS": "0", "GIT_TERMINAL_PROMPT": "0",
           "GIT_NO_LAZY_FETCH": "1"}


class Refused(RuntimeError):
    """Código nominal; los errores de Git nunca se publican."""


@dataclass(frozen=True)
class Request:
    repo: Path
    target_commit: str
    expected_list_sha256: str
    expected_sql_sha256: tuple[str, ...]


def _require(condition, code):
    if not condition:
        raise Refused(code)


def _sha(data):
    return hashlib.sha256(data).hexdigest()


def canonical(value):
    return (json.dumps(value, sort_keys=True, ensure_ascii=True,
                       separators=(",", ":"), allow_nan=False) + "\n").encode("ascii")


def _path(value):
    _require(isinstance(value, str) and len(value) <= 512 and
             re.fullmatch(r"[A-Za-z0-9_./-]+", value) and
             not value.startswith("/") and
             all(p not in ("", ".", "..") for p in value.split("/")),
             "postmain_invalid_relative_path")
    return value


def _git(repo, *args):
    try:
        result = subprocess.run(
            ["/usr/bin/git", "--no-replace-objects", "-c", "core.fsmonitor=false",
             "-c", "core.hooksPath=/dev/null", "-c", "protocol.allow=never",
             "-C", str(repo), *args],
            stdin=subprocess.DEVNULL, stdout=subprocess.PIPE,
            stderr=subprocess.DEVNULL, check=False, timeout=20, env=GIT_ENV)
    except (OSError, subprocess.TimeoutExpired):
        raise Refused("postmain_local_git_unavailable") from None
    _require(result.returncode == 0 and len(result.stdout) <= LIMIT,
             "postmain_git_read_failed")
    return result.stdout


def _commit(repo, value):
    _require(isinstance(value, str) and HEX40.fullmatch(value), "postmain_invalid_commit_sha")
    _require(_git(repo, "cat-file", "-t", value) == b"commit\n", "postmain_sha_not_commit")
    tree = _git(repo, "rev-parse", value + "^{tree}").decode("ascii").strip()
    _require(HEX40.fullmatch(tree), "postmain_invalid_tree_sha")
    return {"commit": value, "tree": tree}


def _blob(repo, commit, path, expected=None):
    _path(path)
    entry = _git(repo, "ls-tree", "-z", commit, "--", path)
    _require(entry.endswith(b"\0") and entry.count(b"\0") == 1, "postmain_blob_missing")
    head, actual = entry[:-1].split(b"\t", 1)
    mode, kind, oid = head.decode("ascii").split(" ")
    _require(actual.decode("ascii") == path and mode in ("100644", "100755") and
             kind == "blob" and HEX40.fullmatch(oid), "postmain_not_regular_blob")
    size = _git(repo, "cat-file", "-s", oid).strip()
    _require(size.isdigit() and 0 < int(size) <= LIMIT, "postmain_blob_size")
    # show conserva los bytes originales del commit; nunca usa el árbol de trabajo.
    data = _git(repo, "show", commit + ":" + path)
    _require(len(data) == int(size) and data == _git(repo, "cat-file", "blob", oid),
             "postmain_original_bytes_mismatch")
    digest = _sha(data)
    _require(expected is None or digest == expected, "postmain_expected_sha_mismatch")
    return data, {"blob": oid, "mode": mode, "bytes": len(data), "sha256": digest}


def _list_paths(data):
    try:
        lines = data.decode("utf-8").splitlines()
    except UnicodeError:
        raise Refused("postmain_invalid_causal_list") from None
    result = []
    for line in lines:
        if not line or line.startswith("#"):
            continue
        result.append(_path(line))
    _require(tuple(result) == SQL_PATHS, "postmain_causal_order_mismatch")
    return result


def _diff(repo, target):
    raw = _git(repo, "diff", "--name-status", "-z", "--no-renames",
               "--no-ext-diff", "--no-textconv", "--ignore-submodules=none",
               SOURCE, target, "--", "*.sql")
    parts = raw.split(b"\0")
    _require(parts[-1] == b"" and len(parts[:-1]) % 2 == 0, "postmain_invalid_diff")
    changes, blockers = [], []
    for position in range(0, len(parts) - 1, 2):
        try:
            code = parts[position].decode("ascii")
            path = _path(parts[position + 1].decode("ascii"))
        except UnicodeError:
            raise Refused("postmain_invalid_diff_path") from None
        _require(code in {"A", "M", "D", "T"}, "postmain_invalid_diff_status")
        role = "causal_sql" if path in SQL_PATHS else (
            "companion_non_executable" if path in COMPANIONS else "unknown_sql")
        change = {"path": path, "status": {"A": "added", "M": "modified",
                  "D": "deleted", "T": "type_changed"}[code], "classification": role,
                  "linked_up": COMPANIONS.get(path), "before": None, "after": None}
        for key, commit, present in (("before", SOURCE, code != "A"),
                                     ("after", target, code != "D")):
            if present:
                try:
                    change[key] = _blob(repo, commit, path)[1]
                except Refused:
                    # Un tipo ajeno no puede convertirse en operación ejecutable.
                    blockers.append({"code": "non_regular_sql_blob", "path": path})
        if code in {"D", "T"}:
            blockers.append({"code": "deleted_or_type_changed_sql", "path": path})
        if role == "unknown_sql":
            blockers.append({"code": "unknown_sql_path", "path": path})
            if code == "M":
                blockers.append({"code": "previous_sql_modified", "path": path})
        changes.append(change)
    _require(len(changes) <= 4096, "postmain_diff_size")
    return changes, blockers


def receipt_requirements():
    """Contrato futuro; describe evidencia requerida y no afirma haberla recibido."""
    common = ["package_sha256", "source_commit", "release_sha256", "lock_sha256",
              "plan_sha256", "approval_sha256", "pg_container_id", "cli_sha256",
              "apply_stdout_sha256", "postimage", "installed_anchors"]
    return {
        "status": "not_read_not_validated",
        "h6": {"required_fields": ["version", "kind", "commit", "source_ref",
            "approved_sql_ref", "plan_family", "file_count", "sql_instaladas",
            "run_id", "identidad_clon", "pg_container_id", "pg_image_id", "pg_volume",
            "sql_journal_sha256", "ad132_receipt_sha256", "package_sha", "lock_sha",
            "release_sha", "canary_plan_sha256", "canary_receipt_sha256"],
            "expected": {"version": 2, "kind": "h6_db_ready", "commit": SOURCE,
                "source_ref": SOURCE, "approved_sql_ref": SOURCE,
                "file_count": 62, "sql_instaladas": 62}},
        "ad132": {"required_fields": common, "expected": {"source_commit": SOURCE}},
        "cross_equal": [["h6.pg_container_id", "ad132.pg_container_id"],
            ["h6.commit", "ad132.source_commit"],
            ["h6.package_sha", "ad132.package_sha256"],
            ["h6.lock_sha", "ad132.lock_sha256"],
            ["h6.release_sha", "ad132.release_sha256"],
            ["h6.canary_plan_sha256", "ad132.plan_sha256"]],
        "digest_binding": ["sha256(ad132_original_bytes) == h6.ad132_receipt_sha256",
                           "sha256(h6_original_bytes) == external_approval.h6_receipt_sha256"],
        "future_gates": ["external_approval_binds_plan_sha256_and_both_receipt_sha256",
            "h6_live_validation_before_transition", "ad132_live_validation_before_transition",
            "installed_sql_inventory_matches_h6_journal_and_original_package",
            "no_history_for_six_causal_sql", "second_postmain_receipt_binds_h6_and_ad132",
            "preserve_h6_historical_receipt_and_live_validator"],
    }


def build_plan(request: Request) -> dict:
    _require(type(request) is Request and isinstance(request.repo, Path), "postmain_invalid_request")
    repo = request.repo
    _require(repo.is_absolute() and ".." not in repo.parts and repo.is_dir() and
             not any(p.is_symlink() for p in (repo, *repo.parents)), "postmain_invalid_repo_path")
    _require(isinstance(request.expected_list_sha256, str) and
             HEX64.fullmatch(request.expected_list_sha256) and
             type(request.expected_sql_sha256) is tuple and len(request.expected_sql_sha256) == 6 and
             all(isinstance(pin, str) and HEX64.fullmatch(pin) for pin in request.expected_sql_sha256),
             "postmain_invalid_expected_hashes")
    source = _commit(repo, SOURCE)
    target = _commit(repo, request.target_commit)
    origin = _git(repo, "rev-parse", "--verify", "refs/remotes/origin/main").decode("ascii").strip()
    _require(origin == target["commit"], "postmain_target_not_origin_main")
    _git(repo, "merge-base", "--is-ancestor", SOURCE, target["commit"])
    data, listing = _blob(repo, target["commit"], LIST, request.expected_list_sha256)
    paths = _list_paths(data)
    operations = []
    for position, (path, expected) in enumerate(zip(paths, request.expected_sql_sha256, strict=True), 1):
        _, record = _blob(repo, target["commit"], path, expected)
        operations.append({"position": position, "path": path, **record})
    changes, blockers = _diff(repo, target["commit"])
    _require(_git(repo, "rev-parse", "--verify", "refs/remotes/origin/main").decode("ascii").strip()
             == origin, "postmain_origin_main_changed")
    return {"version": 1, "kind": "clon_postmain_plan", "plan": "pendiente_aprobacion",
        "executable": False, "sql_invoked": False, "source": source, "target": target,
        "origin_main_observed": origin, "causal_list": {"path": LIST, **listing},
        "operations": operations, "sql_diff": changes, "blockers": blockers,
        "receipt_requirements": receipt_requirements()}


def main(argv=None):
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--repo", required=True, type=Path)
    parser.add_argument("--target-commit", required=True)
    parser.add_argument("--expected-list-sha256", required=True)
    parser.add_argument("--expected-sql-sha256", required=True, action="append")
    args = parser.parse_args(argv)
    value = build_plan(Request(args.repo, args.target_commit, args.expected_list_sha256,
                               tuple(args.expected_sql_sha256)))
    sys.stdout.buffer.write(canonical(value))
    return 2 if value["blockers"] else 0


if __name__ == "__main__":
    try:
        raise SystemExit(main())
    except Refused as error:
        print(str(error), file=sys.stderr)
        raise SystemExit(1)
