#!/usr/bin/env python3
"""Autoridad única de DB_READY para el clon H6 paquete62.

La aprobación es externa al estado y se carga mediante load_approval. Las APIs
validate_pre_ad132(state, approval, live=True) y validate_h6_ready(...)
son de lectura; live=False comprueba documentos y nunca acredita PostgreSQL.
complete_h6(state, approval) siempre revalida AD132 vivo, confirma sólo la fase
del journal privado y publica DB_READY v2. No aplica SQL ni arranca aplicaciones.

El pending propio del adaptador AD132 se conserva: su recibo revalidado acredita
el éxito. Un pending SQL o .sql-confirming bloquea siempre. Recuperar un journal
ya confirmado no reaplica nada; publica el mismo sello determinista.
"""
from __future__ import annotations

import argparse
from contextlib import contextmanager
import hashlib
import json
import os
from pathlib import Path
import re
import stat
import subprocess
import sys
import uuid

try:
    from . import clon_sql, clon_ad132_recibo
except ImportError:
    import clon_sql
    import clon_ad132_recibo

HEX64 = re.compile(r"[0-9a-f]{64}\Z")
HEX40 = re.compile(r"[0-9a-f]{40}\Z")
IMAGE = re.compile(r"sha256:[0-9a-f]{64}\Z")
PATHS = {"repo", "h6_package", "h6_lock", "h1_state_file", "canary_plan",
         "canary_plan_receipt", "ca_path", "binary_path"}
PINS = {"approved_package_sha256", "approved_lock_sha256", "estado_h1_sha",
        "identidad_clon", "pg_container_id", "approved_canary_plan_sha256",
        "approved_canary_receipt_sha256", "arranque_sha256", "ca_sha256",
        "binary_sha256", "material_manifest_sha256", "runtime_config_sha256",
        "projection_manifest_sha256", "projection_config_sha256"}
APPROVAL_FIELDS = PATHS | PINS | {"source_commit", "container", "pg_image_id",
    "pg_volume", "pg_port", "app_port", "ad132_request", "canary_image_id"}
CANARY_FIELDS = {"ad132_request", "canary_plan", "canary_plan_receipt",
    "approved_canary_plan_sha256", "approved_canary_receipt_sha256", "canary_image_id",
    "arranque_sha256"}
PRE_APPROVAL_FIELDS = APPROVAL_FIELDS - CANARY_FIELDS
REQUEST_PATHS = {"package_tar", "package_root", "release_lock", "plan",
                 "plan_receipt", "approval", "receipt", "apply_output", "pending_path"}
PLAN_FIELDS = ("source_ref", "approved_sql_ref", "plan_family", "file_count",
               "plan_sha", "inventory_sha", "package_sha", "lock_sha", "list_sha",
               "release_sha", "estado_h1_sha")
MATERIAL_FILES = {"material-manifest.json": "material_manifest_sha256",
                  "runtime-config.json": "runtime_config_sha256",
                  "runtime-interno/material-manifest.json": "projection_manifest_sha256",
                  "runtime-interno/runtime-config.json": "projection_config_sha256"}


class Refused(RuntimeError):
    """Error nominal; no contiene salidas de proveedores ni material privado."""


def require(condition, code):
    if not condition:
        raise Refused(code)


def canonical(value):
    return (json.dumps(value, ensure_ascii=False, sort_keys=True,
                       separators=(",", ":"), allow_nan=False) + "\n").encode()


def sha(data):
    return hashlib.sha256(data).hexdigest()


def unique(pairs):
    value = {}
    for key, item in pairs:
        require(key not in value, "h6_duplicate_json_key")
        value[key] = item
    return value


def decode(data):
    try:
        return json.loads(data, object_pairs_hook=unique,
                          parse_constant=lambda _: (_ for _ in ()).throw(ValueError()))
    except (ValueError, UnicodeError):
        raise Refused("h6_invalid_json") from None


def nominal(path):
    path = Path(path)
    require(path.is_absolute() and ".." not in path.parts and path != Path("/"),
            "h6_path_not_absolute")
    return path


@contextmanager
def directory(path):
    """Recorrer por descriptor evita enlaces y sustituciones de antecesores."""
    path = nominal(path)
    descriptor = os.open("/", os.O_RDONLY | os.O_DIRECTORY | os.O_CLOEXEC)
    try:
        for part in path.parts[1:]:
            child = os.open(part, os.O_RDONLY | os.O_DIRECTORY | os.O_NOFOLLOW |
                            os.O_CLOEXEC, dir_fd=descriptor)
            os.close(descriptor)
            descriptor = child
        yield descriptor
    finally:
        os.close(descriptor)


def metadata(info):
    return (info.st_dev, info.st_ino, info.st_mode, info.st_uid, info.st_nlink,
            info.st_size, info.st_mtime_ns, info.st_ctime_ns)


def read(path, limit=2 * 1024 * 1024, *, private=True, expected=None):
    path = nominal(path)
    with directory(path.parent) as parent:
        descriptor = os.open(path.name, os.O_RDONLY | os.O_NOFOLLOW |
                             os.O_NONBLOCK | os.O_CLOEXEC, dir_fd=parent)
        with os.fdopen(descriptor, "rb") as stream:
            before = os.fstat(stream.fileno())
            require(stat.S_ISREG(before.st_mode) and before.st_nlink == 1 and
                    before.st_uid == os.getuid() and not before.st_mode &
                    (0o077 if private else 0o022) and 0 < before.st_size <= limit,
                    "h6_unsafe_file")
            data = stream.read(limit + 1)
            require(len(data) == before.st_size and metadata(before) ==
                    metadata(os.fstat(stream.fileno())), "h6_file_changed")
    if expected is not None:
        require(sha(data) == expected, "h6_approval_hash_mismatch")
    return data


def validate_approval(approval, *, pre_ad132=False):
    require(isinstance(approval, dict) and (set(approval) == APPROVAL_FIELDS or
            pre_ad132 and set(approval) == PRE_APPROVAL_FIELDS),
            "h6_external_approval_missing_or_unknown")
    result = dict(approval)
    for key in PATHS & approval.keys():
        result[key] = nominal(approval[key])
    for key in PINS & approval.keys():
        require(isinstance(approval[key], str) and HEX64.fullmatch(approval[key]),
                "h6_invalid_external_pin")
    require(isinstance(approval["source_commit"], str) and
            HEX40.fullmatch(approval["source_commit"]), "h6_invalid_source")
    for key in ("pg_image_id", "canary_image_id"):
        require(key not in approval or isinstance(approval[key], str) and IMAGE.fullmatch(approval[key]),
                "h6_invalid_image_pin")
    require(isinstance(approval["container"], str) and
            re.fullmatch(r"vec-[a-z0-9_-]+", approval["container"]), "h6_invalid_container")
    for key in ("pg_port", "app_port"):
        require(type(approval[key]) is int and 1024 <= approval[key] <= 65535,
                "h6_invalid_port")
    require(approval["pg_port"] != approval["app_port"], "h6_duplicate_port")
    volume = approval["pg_volume"]
    require(isinstance(volume, dict) and set(volume) == {"source", "dev", "ino"},
            "h6_invalid_volume_pin")
    source = nominal(volume["source"])
    require(source.parent == Path("/dev/shm") and
            re.fullmatch(r"vec-recorridos-[A-Za-z0-9_-]+", source.name) and
            type(volume["dev"]) is int and volume["dev"] >= 0 and
            type(volume["ino"]) is int and volume["ino"] > 0, "h6_invalid_volume_pin")
    result["pg_volume"] = {**volume, "source": str(source)}
    if pre_ad132 and set(approval) == PRE_APPROVAL_FIELDS:
        return result
    request = approval["ad132_request"]
    if isinstance(request, dict):
        require(set(request) == set(clon_ad132_recibo.Request.__dataclass_fields__),
                "h6_invalid_ad132_request")
        request = clon_ad132_recibo.Request(**{
            key: nominal(value) if key in REQUEST_PATHS else value
            for key, value in request.items()})
    require(isinstance(request, clon_ad132_recibo.Request), "h6_invalid_ad132_request")
    for key in REQUEST_PATHS:
        nominal(getattr(request, key))
    for key in ("approved_package_sha256", "approved_lock_sha256",
                "approved_release_sha256", "approved_plan_sha256", "approval_sha256",
                "container"):
        require(isinstance(getattr(request, key), str) and
                HEX64.fullmatch(getattr(request, key)), "h6_invalid_ad132_request")
    require(request.package_tar == result["h6_package"] and
            request.release_lock == result["h6_lock"] and
            request.approved_package_sha256 == approval["approved_package_sha256"] and
            request.approved_lock_sha256 == approval["approved_lock_sha256"] and
            request.source_commit == approval["source_commit"] and
            request.container == approval["pg_container_id"] and
            request.plan == result["canary_plan"] and
            request.plan_receipt == result["canary_plan_receipt"] and
            request.approved_plan_sha256 == approval["approved_canary_plan_sha256"],
            "h6_ad132_request_mismatch")
    result["ad132_request"] = request
    return result


def load_approval(path, *, pre_ad132=False):
    """JSON privado canónico cerrado; nunca reconstruye pins desde DB_READY."""
    data = read(path, 64_000)
    value = decode(data)
    require(data == canonical(value), "h6_approval_not_canonical")
    return validate_approval(value, pre_ad132=pre_ad132)


def state_root(state):
    state = nominal(state)
    require(not any((parent / ".git").exists() for parent in (state, *state.parents)),
            "h6_state_in_git")
    with directory(state) as descriptor:
        info = os.fstat(descriptor)
        require(info.st_uid == os.getuid() and stat.S_IMODE(info.st_mode) == 0o700,
                "h6_state_not_private")
    return state


def exists(path):
    # lstat incluye enlaces rotos: una marca incierta no desaparece por resolverla.
    try:
        path.lstat()
        return True
    except FileNotFoundError:
        return False


def plan_and_journal(state, approval, phase):
    plan, _ = clon_sql.preflight_h6_package(approval["h6_package"], approval["h6_lock"],
        approval["approved_package_sha256"], approval["approved_lock_sha256"],
        approval["h1_state_file"], approval["estado_h1_sha"],
        source_ref=approval["source_commit"], git_repo=approval["repo"])
    require(plan.get("plan_family") == clon_sql.H6_PACKAGE_FAMILY and
            plan.get("file_count") == 62, "h6_requires_package62")
    context = {key: plan[key] for key in (*clon_sql.CONTEXT_KEYS[1:], "lock_sha")}
    context["identidad_clon"] = approval["identidad_clon"]
    require(not exists(state / ".sql-confirming"), "h6_sql_confirmation_pending")
    data = read(state / "sql-journal.json", clon_sql.MAX_JOURNAL)
    journal = decode(data)
    require(isinstance(journal, dict) and journal.get("journal_sha") ==
            clon_sql.record_hash(journal), "h6_journal_corrupt")
    try:
        require(str(uuid.UUID(journal["run_id"])) == journal["run_id"], "h6_invalid_run_id")
    except (KeyError, ValueError, TypeError, AttributeError):
        raise Refused("h6_invalid_run_id") from None
    clon_sql.validate_record(journal, plan, context)
    clon_sql.validate_receipts(journal.get("installed", []), plan, complete=True)
    require(journal.get("phase") in phase and len(journal["installed"]) == 62,
            "h6_journal_not_complete")
    if "ad132_request" in approval:
        require(approval["ad132_request"].approved_release_sha256 == plan["release_sha"],
                "h6_ad132_release_mismatch")
    return plan, journal, sha(data)


def clone_record(state, approval):
    value = decode(read(state / "clon.json", 64_000))
    expected = {"propietario": "Codex-M", "estado": str(state),
                "contenedor": approval["container"], "puerto_pg": approval["pg_port"],
                "puerto_web": approval["app_port"], "commit": approval["source_commit"],
                "pgdata": approval["pg_volume"]["source"]}
    require(isinstance(value, dict) and all(value.get(key) == item for key, item in
            expected.items()), "h6_clone_inventory_mismatch")
    if "identidad_clon" in value:
        require(value["identidad_clon"] == approval["identidad_clon"], "h6_clone_identity_mismatch")
    return expected


def immutable_material(state, approval):
    hashes, documents = {}, {}
    for relative, key in MATERIAL_FILES.items():
        data = read(state / relative, expected=approval[key])
        hashes[key] = sha(data)
        documents[relative] = decode(data)
    for relative in ("material-manifest.json", "runtime-interno/material-manifest.json"):
        manifest = documents[relative]
        target = {"source_commit": approval["source_commit"],
                  "pg_port": approval["pg_port"], "app_port": approval["app_port"]}
        require(isinstance(manifest, dict) and manifest.get("owner") == "Codex-M" and
                isinstance(manifest.get("target"), dict) and
                all(manifest["target"].get(k) == v for k, v in target.items()),
                "h6_material_target_mismatch")
        require(manifest.get("status") in ("prepared", "pending_ad132") and
                manifest.get("blockers", []) == [], "h6_material_incomplete")
        inventory = manifest.get("files")
        require(isinstance(inventory, dict) and 0 < len(inventory) <= 2048,
                "h6_material_inventory_missing")
        root = (state / relative).parent
        for name, digest in inventory.items():
            require(isinstance(name, str) and name and not Path(name).is_absolute() and
                    all(part not in ("", ".", "..") for part in name.split("/")) and
                    isinstance(digest, str) and HEX64.fullmatch(digest),
                    "h6_invalid_material_inventory")
            read(root / name, 32 * 1024 * 1024, expected=digest)
    principal = documents["material-manifest.json"]
    descriptor = principal.get("runtime_interno", {})
    require(isinstance(descriptor, dict) and
            descriptor.get("manifest") == "runtime-interno/material-manifest.json" and
            descriptor.get("config") == "runtime-interno/runtime-config.json" and
            descriptor.get("source_commit") == approval["source_commit"] and
            descriptor.get("manifest_sha256") == approval["projection_manifest_sha256"],
            "h6_projection_not_bound")
    for relative in ("runtime-config.json", "runtime-interno/runtime-config.json"):
        config = documents[relative]
        require(isinstance(config, dict) and config and all(
            isinstance(k, str) and re.fullmatch(r"VEC_[A-Z0-9_]+", k) and
            isinstance(v, str) and "\x00" not in v and "\n" not in v
            for k, v in config.items()), "h6_invalid_runtime_config")
    require(approval["ca_path"].is_relative_to(state) and
            approval["binary_path"].is_relative_to(state), "h6_material_outside_state")
    hashes["ca_sha256"] = sha(read(approval["ca_path"], 1_000_000, private=False,
                                   expected=approval["ca_sha256"]))
    hashes["binary_sha256"] = sha(read(approval["binary_path"], 128 * 1024 * 1024,
                                       private=False, expected=approval["binary_sha256"]))
    return hashes


def docker_record(approval):
    result = subprocess.run(["/usr/bin/docker", "inspect", "--type", "container",
            approval["pg_container_id"]], stdin=subprocess.DEVNULL,
            stdout=subprocess.PIPE, stderr=subprocess.DEVNULL, timeout=15, check=False,
            env={"PATH": "/usr/bin:/bin", "LC_ALL": "C"})
    require(result.returncode == 0 and len(result.stdout) <= 1024 * 1024,
            "h6_live_container_unavailable")
    value = decode(result.stdout)
    require(isinstance(value, list) and len(value) == 1 and isinstance(value[0], dict),
            "h6_live_container_invalid")
    return value[0]


def live_container(state, approval):
    value = docker_record(approval)
    labels = value.get("Config", {}).get("Labels") or {}
    require(value.get("Id") == approval["pg_container_id"] and
            value.get("Image") == approval["pg_image_id"] and
            value.get("Name") == "/" + approval["container"] and
            value.get("Config", {}).get("Image") == "postgres:18.4" and
            value.get("State", {}).get("Running") is True and
            labels.get("vec.recorridos.owner") == "Codex-M" and
            labels.get("vec.recorridos.state") == str(state), "h6_live_container_mismatch")
    bindings = value.get("NetworkSettings", {}).get("Ports") or {}
    require(all(binding.get("HostIp") in ("127.0.0.1", "::1") and
                binding.get("HostPort") == str(approval["pg_port"])
                for entries in bindings.values() for binding in (entries or [])),
            "h6_live_container_ports_mismatch")
    mounts = [mount for mount in value.get("Mounts", [])
              if mount.get("Destination") == "/var/lib/postgresql"]
    require(len(mounts) == 1 and mounts[0].get("Type") == "bind" and
            mounts[0].get("Source") == approval["pg_volume"]["source"] and
            mounts[0].get("RW") is True, "h6_live_volume_mismatch")
    with directory(approval["pg_volume"]["source"]) as descriptor:
        info = os.fstat(descriptor)
        require((info.st_dev, info.st_ino) == (approval["pg_volume"]["dev"],
                approval["pg_volume"]["ino"]), "h6_live_volume_replaced")


def canary_evidence(state, approval):
    require(approval["canary_plan"] == state / "plan-conexiones.json" and
            approval["canary_plan_receipt"] == state / "plan-canonico-clon.json",
            "h6_canary_nominal_paths")
    data = read(approval["canary_plan"], 131072,
                expected=approval["approved_canary_plan_sha256"])
    plan = decode(data)
    require(isinstance(plan, dict) and set(plan) == {"conexiones", "huellas"} and
            isinstance(plan["conexiones"], list) and 24 <= len(plan["conexiones"]) <= 96 and
            isinstance(plan["huellas"], dict) and plan["huellas"], "h6_invalid_canary_plan")
    seen = set()
    for entry in plan["conexiones"]:
        require(isinstance(entry, dict) and set(entry) == {"fuente", "login", "rol"},
                "h6_invalid_canary_connection")
        require(isinstance(entry["fuente"], str) and entry["fuente"] not in seen and
                isinstance(entry["login"], str) and
                re.fullmatch(r"[a-z][a-z0-9_]{2,95}", entry["login"]) and
                isinstance(entry["rol"], str) and
                re.fullmatch(r"[a-z][a-z0-9_]{2,95}", entry["rol"]), "h6_invalid_canary_connection")
        seen.add(entry["fuente"])
    require(all(isinstance(k, str) and isinstance(v, str) and HEX64.fullmatch(v)
                for k, v in plan["huellas"].items()), "h6_invalid_canary_inventory")
    expected = {"version": 1, "kind": "clon",
        "package_sha256": approval["approved_package_sha256"],
        "source_commit": approval["source_commit"], "pg_container_id": approval["pg_container_id"],
        "plan_sha256": sha(data), "material_inventory_sha256": sha(canonical(plan["huellas"])),
        "canary_image_id": approval["canary_image_id"], "arranque_sha256": approval["arranque_sha256"],
        "canary_output_sha256": sha(data)}
    receipt = read(approval["canary_plan_receipt"], 8192,
                   expected=approval["approved_canary_receipt_sha256"])
    require(receipt == canonical(expected), "h6_canary_receipt_mismatch")
    return {"canary_plan_sha256": sha(data), "canary_receipt_sha256": sha(receipt),
            "canary_image_id": approval["canary_image_id"], "arranque_sha256": approval["arranque_sha256"]}


def ad132_evidence(state, approval, live):
    request = approval["ad132_request"]
    for key in ("receipt", "apply_output", "pending_path"):
        require(getattr(request, key).is_relative_to(state), "h6_ad132_evidence_outside_state")
    data = read(request.receipt, 4_000_000)
    receipt = decode(data)
    require(isinstance(receipt, dict) and data == canonical(receipt), "h6_ad132_receipt_invalid")
    expected = {"package_sha256": approval["approved_package_sha256"],
        "source_commit": approval["source_commit"], "lock_sha256": approval["approved_lock_sha256"],
        "plan_sha256": approval["approved_canary_plan_sha256"],
        "release_sha256": request.approved_release_sha256,
        "pg_container_id": approval["pg_container_id"], "approval_sha256": request.approval_sha256}
    require(all(receipt.get(k) == v for k, v in expected.items()), "h6_ad132_receipt_mismatch")
    require(read(request.apply_output, 256) == clon_ad132_recibo.SUCCESS,
            "h6_ad132_exit0_evidence_missing")
    # pending es parte de la evidencia canónica del adaptador; no borrarlo.
    read(request.pending_path, 64_000)
    if live:
        require(clon_ad132_recibo.revalidar(request) == receipt, "h6_ad132_live_receipt_changed")
        require(read(request.receipt, 4_000_000) == data, "h6_ad132_receipt_changed")
    return {"ad132_receipt_sha256": sha(data)}


def validated(state, approval, phases, live, *, final):
    require(type(live) is bool, "h6_invalid_live_flag")
    state, approval = state_root(state), validate_approval(approval, pre_ad132=not final)
    if final:
        require(not exists(state / "canario-publicacion-pendiente.json"),
                "h6_canary_publication_pending")
    plan, journal, journal_sha = plan_and_journal(state, approval, phases)
    clone = clone_record(state, approval)
    materials = immutable_material(state, approval)
    if live:
        live_container(state, approval)
    evidence = canary_evidence(state, approval) | ad132_evidence(state, approval, live) if final else {}
    if live and final:
        live_container(state, approval)
    # Detectar cambios en el journal/material durante consultas externas.
    plan2, journal2, journal_sha2 = plan_and_journal(state, approval, phases)
    require(plan2 == plan and journal2 == journal and journal_sha2 == journal_sha and
            immutable_material(state, approval) == materials and
            clone_record(state, approval) == clone, "h6_context_changed")
    if final:
        require(not exists(state / "canario-publicacion-pendiente.json"),
                "h6_canary_publication_pending")
    ready = {"version": 2, "kind": "h6_db_ready", **clone, **{k: plan[k] for k in PLAN_FIELDS},
        "sql_instaladas": 62, "run_id": journal["run_id"], "identidad_clon": approval["identidad_clon"],
        "pg_container_id": approval["pg_container_id"], "pg_image_id": approval["pg_image_id"],
        "pg_volume": approval["pg_volume"], "sql_journal_sha256": journal_sha, **materials, **evidence}
    return state, approval, ready, journal


def validate_pre_ad132(state, approval, live=True):
    """Antes del canario: 62 confirmaciones, material definitivo, ningún READY."""
    state, _, value, _ = validated(state, approval, {"awaiting_ad132"}, live, final=False)
    require(not any(exists(state / name) for name in
        ("DB_READY.json", "READY.json", "runtime-process.json", "runtime-container.json")),
        "h6_pre_ad132_application_or_ready_exists")
    return {key: value[key] for key in (*PLAN_FIELDS, "identidad_clon", "run_id")}


def validate_h6_ready(state, approval, live=True):
    """El sello nunca suministra su propia aprobación; live=False es documental."""
    state, _, expected, _ = validated(state, approval, {"ad132_confirmed"}, live, final=True)
    data = read(state / "DB_READY.json", 64_000)
    require(data == canonical(expected), "h6_db_ready_stale_or_legacy")
    return expected


def publish(directory_fd, data):
    """Promoción exclusiva atómica; fsync del archivo y del directorio."""
    temporary = ".h6-ready-" + uuid.uuid4().hex
    descriptor = os.open(temporary, os.O_WRONLY | os.O_CREAT | os.O_EXCL |
                         os.O_NOFOLLOW | os.O_CLOEXEC, 0o600, dir_fd=directory_fd)
    try:
        with os.fdopen(descriptor, "wb") as stream:
            stream.write(data)
            stream.flush()
            os.fsync(stream.fileno())
        os.link(temporary, "DB_READY.json", src_dir_fd=directory_fd,
                dst_dir_fd=directory_fd, follow_symlinks=False)
        os.unlink(temporary, dir_fd=directory_fd)
        os.fsync(directory_fd)
    finally:
        try:
            os.unlink(temporary, dir_fd=directory_fd)
        except FileNotFoundError:
            pass


def complete_h6(state, approval):
    """Único cierre H6. Sólo AD132 revalidado vivo permite confirmar y publicar."""
    state, approval = state_root(state), validate_approval(approval)
    with clon_sql.Journal(state) as journal_file:
        _, _, ready, original = validated(state, approval,
            {"awaiting_ad132", "ad132_confirmed"}, True, final=True)
        require(journal_file.load(required=True) == original, "h6_journal_changed")
        if exists(state / "DB_READY.json"):
            require(original["phase"] == "ad132_confirmed", "h6_ready_before_confirmation")
            return validate_h6_ready(state, approval, live=True)
        require(not any(exists(state / name) for name in
            ("READY.json", "runtime-process.json", "runtime-container.json")),
            "h6_application_already_reserved")
        if original["phase"] == "awaiting_ad132":
            record = {**original, "phase": "ad132_confirmed"}
            journal_file.store(record)
        # Si cayó después de store, esta misma lectura/publicación recupera el sello.
        _, _, ready, _ = validated(state, approval, {"ad132_confirmed"}, True, final=True)
        publish(journal_file.directory, canonical(ready))
        return validate_h6_ready(state, approval, live=True)


def main(argv=None):
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("mode", choices=("pre-ad132", "complete", "verify"))
    parser.add_argument("--state", type=Path, required=True)
    parser.add_argument("--approval", type=Path, required=True)
    args = parser.parse_args(argv)
    approval = load_approval(args.approval, pre_ad132=args.mode == "pre-ad132")
    {"pre-ad132": validate_pre_ad132, "complete": complete_h6,
     "verify": validate_h6_ready}[args.mode](args.state, approval)
    print("H6-DB-READY-OK" if args.mode != "pre-ad132" else "H6-PRE-AD132-OK")


if __name__ == "__main__":
    try:
        main()
    except Exception as error:
        # No imprimir excepciones proveedor/JSON/paths ni stdout privados.
        print("H6-NO-GO " + (str(error) if isinstance(error, Refused) else type(error).__name__),
              file=sys.stderr)
        raise SystemExit(1)
