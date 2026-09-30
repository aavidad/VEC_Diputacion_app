"""Read-only gate for definitive material before the offline H6 canary."""
from __future__ import annotations

import importlib.util
import json
import os
from pathlib import Path
import stat
import sys
import uuid


class PreparationError(RuntimeError):
    pass


def read(path, limit=2 * 1024 * 1024):
    path = Path(path).absolute()
    if path != path.resolve():
        raise PreparationError("pre_ad132_symlink_path")
    fd = os.open(path, os.O_RDONLY | os.O_NOFOLLOW | os.O_NONBLOCK)
    with os.fdopen(fd, "rb") as stream:
        info = os.fstat(stream.fileno())
        if (not stat.S_ISREG(info.st_mode) or info.st_nlink != 1 or info.st_uid != os.getuid()
                or info.st_mode & 0o077 or not 0 < info.st_size <= limit):
            raise PreparationError("pre_ad132_unsafe_file")
        data = stream.read(limit + 1)
        if len(data) != info.st_size:
            raise PreparationError("pre_ad132_file_changed")
        return data


def load_sql():
    path = Path(__file__).with_name("clon_sql.py")
    if not path.is_file() or path.is_symlink():
        raise PreparationError("pre_ad132_sql_validator_missing")
    spec = importlib.util.spec_from_file_location("clon_sql_pre_ad132", path)
    module = importlib.util.module_from_spec(spec)
    sys.modules[spec.name] = module
    spec.loader.exec_module(module)
    return module


def validate(repo, state, container, pg_port, source, approval, *, pre_ad132=True):
    """Validate external pins and closed journal; never lock/write/query the DB."""
    state = Path(state).absolute()
    if (state != state.resolve() or not state.is_dir() or state.stat().st_uid != os.getuid()
            or state.stat().st_mode & 0o077
            or any((parent / ".git").exists() for parent in (state, *state.parents))):
        raise PreparationError("pre_ad132_invalid_private_root")
    if not isinstance(approval, dict):
        raise PreparationError("pre_ad132_external_approval_missing")
    required = ("h6_package", "h6_lock", "approved_package_sha256", "approved_lock_sha256",
                "h1_state_file", "estado_h1_sha", "identidad_clon")
    if any(not approval.get(key) for key in required):
        raise PreparationError("pre_ad132_external_approval_missing")
    sql = load_sql()
    plan, _ = sql.preflight_h6_package(*(approval[key] for key in required[:6]),
                                     source_ref=source, git_repo=repo)
    context = {key: plan[key] for key in (*sql.CONTEXT_KEYS[1:], "lock_sha")}
    context["identidad_clon"] = approval["identidad_clon"]
    journal = json.loads(read(state / "sql-journal.json", sql.MAX_JOURNAL))
    if (state / ".sql-confirming").exists() or (state / ".sql-confirming").is_symlink():
        raise PreparationError("pre_ad132_uncertain_sql_confirmation")
    if not isinstance(journal, dict) or journal.get("journal_sha") != sql.record_hash(journal):
        raise PreparationError("pre_ad132_journal_corrupt")
    uuid.UUID(journal["run_id"])
    sql.validate_record(journal, plan, context)
    phase = "awaiting_ad132" if pre_ad132 else "ad132_confirmed"
    if (plan.get("plan_family") != sql.H6_PACKAGE_FAMILY or plan.get("file_count") != 62
            or journal.get("phase") != phase or len(journal.get("installed", [])) != 62):
        raise PreparationError("pre_ad132_requires_closed_package62")
    metadata = json.loads(read(state / "clon.json"))
    expected = {"propietario": "Codex-M", "estado": str(state), "contenedor": container,
                "puerto_pg": pg_port, "commit": source}
    if any(metadata.get(key) != value for key, value in expected.items()):
        raise PreparationError("pre_ad132_clone_inventory_mismatch")
    if "identidad_clon" in metadata and metadata["identidad_clon"] != context["identidad_clon"]:
        raise PreparationError("pre_ad132_clone_identity_mismatch")
    markers = ("DB_READY.json", "READY.json")
    if pre_ad132:
        if any((state / name).exists() or (state / name).is_symlink() for name in markers):
            raise PreparationError("pre_ad132_ready_already_exists")
        process = state / "runtime-process.json"
        if process.exists() or process.is_symlink():
            raise PreparationError("pre_ad132_application_reservation_exists")
    else:
        ready = json.loads(read(state / "DB_READY.json"))
        if (any(ready.get(key) != value for key, value in expected.items())
                or ready.get("puerto_web") != metadata.get("puerto_web") or ready.get("sql_instaladas") != 62):
            raise PreparationError("pre_ad132_final_ready_mismatch")
    # Stable across the AD132 transition: it binds the approved inputs, not
    # a journal postimage or a fabricated database readiness claim.
    return {key: plan[key] for key in ("source_ref", "approved_sql_ref", "plan_family", "file_count",
            "plan_sha", "inventory_sha", "package_sha", "lock_sha", "list_sha", "release_sha", "estado_h1_sha")} | {
                "identidad_clon": context["identidad_clon"]}
