#!/usr/bin/env python3
"""Inspect the owned clone's candidate material without activating an absent authority."""
from __future__ import annotations

import argparse
import fcntl
import hashlib
import json
import os
from pathlib import Path
import re
import stat
import subprocess
import sys
import tempfile

OWNER = "Codex-M"
SUBJECT = "desarrollo:clon-recorridos:candidato"
BLOCKER = "contexto_externo_provision_autoridad_ausente_main"
MAX_FILE = 64 * 1024


class CandidateMaterialError(RuntimeError):
    pass


def fail(code: str) -> None:
    raise CandidateMaterialError(code)


def canonical(path: Path) -> Path:
    path = Path(os.path.abspath(path))
    if path.resolve() != path:
        fail("path_symlink")
    return path


def private_dir(path: Path) -> Path:
    path = canonical(path)
    info = path.stat()
    if not stat.S_ISDIR(info.st_mode) or info.st_uid != os.getuid() or info.st_mode & 0o077:
        fail("directory_not_private")
    return path


def read_private(path: Path) -> bytes:
    canonical(path)
    fd = os.open(path, os.O_RDONLY | os.O_NOFOLLOW | os.O_NONBLOCK)
    try:
        info = os.fstat(fd)
        if not stat.S_ISREG(info.st_mode) or info.st_nlink != 1 or info.st_uid != os.getuid() or info.st_mode & 0o077:
            fail("file_not_private")
        if info.st_size > MAX_FILE:
            fail("file_too_large")
        data = os.read(fd, MAX_FILE + 1)
        after = os.fstat(fd)
        unchanged = (info.st_dev, info.st_ino, info.st_mode, info.st_nlink, info.st_uid,
                     info.st_size, info.st_mtime_ns) == (
                     after.st_dev, after.st_ino, after.st_mode, after.st_nlink, after.st_uid,
                     after.st_size, after.st_mtime_ns)
        if len(data) != info.st_size or not unchanged:
            fail("file_changed")
        return data
    finally:
        os.close(fd)


def unique_object(pairs: list[tuple[str, object]]) -> dict:
    value = {}
    for key, item in pairs:
        if key in value:
            fail("duplicate_json_key")
        value[key] = item
    return value


def read_json(path: Path) -> dict:
    value = json.loads(read_private(path), object_pairs_hook=unique_object)
    if not isinstance(value, dict):
        fail("json_object_required")
    return value


def run(argv: list[str], data: bytes | None = None) -> bytes:
    completed = subprocess.run(argv, input=data, stdout=subprocess.PIPE, stderr=subprocess.PIPE,
                               timeout=20, check=False)
    if completed.returncode:
        # Never propagate command arguments, database output or PEM through errors.
        fail("subprocess_failed")
    return completed.stdout


def validate_container(info: dict, state: Path, port: int) -> None:
    labels = info.get("Config", {}).get("Labels", {})
    if labels.get("vec.recorridos.owner") != OWNER or labels.get("vec.recorridos.state") != str(state):
        fail("clone_ownership_mismatch")
    if info.get("State", {}).get("Running") is not True:
        fail("clone_not_running")
    if info.get("NetworkSettings", {}).get("Ports") != {
        "5432/tcp": [{"HostIp": "127.0.0.1", "HostPort": str(port)}]
    }:
        fail("clone_loopback_port_mismatch")


def reference(prefix: str, value: str) -> str:
    digest = hashlib.sha256(("vec.ct.alta.desarrollo.v1\0" + value).encode()).hexdigest()
    return prefix + digest[:32]


def inspect_identity(material: Path) -> dict:
    identity = read_json(material / "identidad/candidato.json")
    expected_keys = {"version", "autoridad", "certificate_sha256", "subject", "display_name", "roles"}
    if set(identity) != expected_keys or identity["version"] != 1 or identity["autoridad"] != "no_autoritativo":
        fail("candidate_identity_invalid")
    if identity["subject"] != SUBJECT or identity["roles"] != ["candidato_bolsa"]:
        fail("candidate_identity_not_nominal")
    cert = read_private(material / "mtls/candidato.crt")
    read_private(material / "mtls/candidato.key")
    read_private(material / "ca/ca.crt")
    run(["openssl", "verify", "-purpose", "sslclient", "-CAfile", str(material / "ca/ca.crt"),
         str(material / "mtls/candidato.crt")])
    der = run(["openssl", "x509", "-outform", "DER"], cert)
    fingerprint = hashlib.sha256(der).hexdigest()
    if identity["certificate_sha256"] != fingerprint:
        fail("candidate_certificate_mismatch")
    san = run(["openssl", "x509", "-noout", "-ext", "subjectAltName"], cert).decode().splitlines()
    if len(san) != 2 or san[1].strip() != "URI:urn:vec:" + SUBJECT:
        fail("candidate_certificate_subject_mismatch")
    public_cert = run(["openssl", "x509", "-pubkey", "-noout"], cert)
    public_key = run(["openssl", "pkey", "-pubout"], read_private(material / "mtls/candidato.key"))
    if public_cert != public_key:
        fail("candidate_private_key_mismatch")
    base = SUBJECT + "\0" + fingerprint
    return {"subject": SUBJECT, "role": "candidato_bolsa", "certificate_sha256": fingerprint,
            "cert": "material/mtls/candidato.crt", "key": "material/mtls/candidato.key",
            "pkcs12": "material/mtls/candidato.p12", "pkcs12_password_file": "material/mtls/candidato.p12.password",
            "expected_cuenta_ref": reference("cta_", base + "\0cuenta"),
            "expected_persona_ref": reference("per_", base + "\0persona"),
            "expected_fixture_perfil_ref": reference("prf_", base + "\0perfil"),
            "status": "blocked_authority", "durable_provisioned": False}


INVENTORY_SQL = b"""BEGIN READ ONLY;
SET LOCAL search_path=pg_catalog;
SET LOCAL statement_timeout='5s';
SELECT jsonb_build_object(
 'bolsa_candidate_count',(SELECT count(DISTINCT candidato_ref) FROM vec_bolsa_llamamientos.vinculo_candidato),
 'candidate_context_count',(SELECT count(*) FROM vec_contexto_actor_v1.vinculo_referencia_versiones WHERE tipo='candidato'),
 'context_apis',(SELECT jsonb_agg(p.proname ORDER BY p.proname)
  FROM pg_proc p JOIN pg_namespace n ON n.oid=p.pronamespace WHERE n.nspname='vec_contexto_actor_v1')
)::text;
ROLLBACK;
"""


def validate_result_update(old: dict, result: dict, repo: Path) -> None:
    old_target, new_target = old.get("target", {}), result.get("target", {})
    if not isinstance(old_target, dict) or not isinstance(new_target, dict):
        fail("candidate_result_preimage_mismatch")
    immutable_old = {key: value for key, value in old.items() if key not in ("target", "inventory")}
    immutable_new = {key: value for key, value in result.items() if key not in ("target", "inventory")}
    if immutable_old != immutable_new or (
        {key: value for key, value in old_target.items() if key != "source_commit"} !=
        {key: value for key, value in new_target.items() if key != "source_commit"}
    ):
        fail("candidate_result_preimage_mismatch")
    old_source, new_source = old_target.get("source_commit", ""), new_target.get("source_commit", "")
    if not all(isinstance(source, str) and re.fullmatch(r"[0-9a-f]{40}", source)
               for source in (old_source, new_source)):
        fail("candidate_result_source_invalid")
    if old_source != new_source:
        run(["git", "-C", str(repo), "merge-base", "--is-ancestor", old_source, new_source])


def write_result(path: Path, result: dict, repo: Path) -> None:
    path = canonical(path)
    private_dir(path.parent)
    encoded = (json.dumps(result, sort_keys=True, indent=2) + "\n").encode()
    lock = path.parent / "result.lock"
    canonical(lock)
    fd = os.open(lock, os.O_RDWR | os.O_CREAT | os.O_NOFOLLOW | os.O_NONBLOCK, 0o600)
    try:
        info = os.fstat(fd)
        if not stat.S_ISREG(info.st_mode) or info.st_nlink != 1 or info.st_uid != os.getuid() or info.st_mode & 0o077:
            fail("candidate_result_lock_invalid")
        fcntl.flock(fd, fcntl.LOCK_EX | fcntl.LOCK_NB)
        if path.exists():
            old = read_json(path)
            # Only source ancestry and the read-only inventory may advance.
            validate_result_update(old, result, repo)
            if old == result:
                return
        temporary_fd, temporary_name = tempfile.mkstemp(prefix=".result-", dir=path.parent)
        try:
            with os.fdopen(temporary_fd, "wb") as stream:
                stream.write(encoded)
                stream.flush()
                os.fsync(stream.fileno())
            os.replace(temporary_name, path)
        finally:
            if os.path.exists(temporary_name):
                os.unlink(temporary_name)
    finally:
        os.close(fd)


def provision(repo: Path, container: str, state: Path, material: Path,
              pg_port: int, engine: str = "docker") -> dict:
    """Return a blocked profile; no database writes, grants or runtime activation."""
    repo, state, material = canonical(repo), private_dir(state), private_dir(material)
    if material != state / "material" or not repo.is_dir():
        fail("candidate_paths_mismatch")
    if engine not in ("docker", "podman") or not re.fullmatch(r"vec-[a-z0-9_-]+", container):
        fail("clone_arguments_invalid")
    if not 1024 <= pg_port <= 65535:
        fail("clone_port_invalid")
    for ancestor in (state, *state.parents):
        if (ancestor / ".git").exists():
            fail("private_state_inside_git")
    ready = read_json(state / "DB_READY.json")
    if (ready.get("propietario"), ready.get("estado"), ready.get("contenedor"), ready.get("puerto_pg")) != (
            OWNER, str(state), container, pg_port):
        fail("db_readiness_mismatch")
    source = ready.get("commit", "")
    if not re.fullmatch(r"[0-9a-f]{40}", source):
        fail("db_source_invalid")
    run(["git", "-C", str(repo), "cat-file", "-e", source + "^{commit}"])
    info = json.loads(run([engine, "inspect", container]))[0]
    validate_container(info, state, pg_port)
    if (material / "identidad/bolsa-candidato.json").exists():
        fail("candidate_activation_already_present")
    profile = inspect_identity(material)
    inventory = json.loads(run([engine, "exec", "-i", container, "psql", "-X", "-qAt",
                               "-h", "/var/run/postgresql", "-p", "5432", "-v", "ON_ERROR_STOP=1",
                               "-U", "postgres", "-d", "postgres", "-f", "-"],
                              INVENTORY_SQL))
    output = state / "candidato-material"
    output.mkdir(mode=0o700, exist_ok=True)
    private_dir(output)
    result = {"version": 1, "target": {"owner": OWNER, "source_commit": source,
              "container_id": info["Id"], "pg_port": pg_port}, "env": {},
              "profiles": {"candidato": profile}, "blockers": [{"profile": "candidato", "code": BLOCKER,
              "sources": ["internal/app/bootstrap/bolsa_mi_bolsa.go:nuevaRutaMiBolsaDesarrollo",
                          "internal/app/bootstrap/contratacion_temporal_autoridad_postgresql_desarrollo.go:publicarResultadoContextoPostgreSQLDesarrollo",
                          "CANAL_CLAUDE_CODEX.md:2026-09-30T00:29:NO-GO-178"]}],
              "inventory": inventory, "status": "blocked", "sql_written": False,
              "runtime_activated": False, "candidate_ref_assigned": False}
    write_result(output / "result.json", result, repo)
    return result


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--repo", type=Path, required=True)
    parser.add_argument("--container", required=True)
    parser.add_argument("--output", type=Path, required=True)
    parser.add_argument("--pg-port", type=int, required=True)
    parser.add_argument("--result-file", type=Path, required=True)
    parser.add_argument("--engine", choices=("docker", "podman"), default="docker")
    args = parser.parse_args()
    try:
        state = canonical(args.output)
        if canonical(args.result_file) != state / "candidato-material/result.json":
            fail("result_file_outside_owned_directory")
        result = provision(args.repo, args.container, state, state / "material", args.pg_port, args.engine)
        print(json.dumps({"status": result["status"], "blockers": [b["code"] for b in result["blockers"]]}))
        return 3
    except (CandidateMaterialError, OSError, ValueError, KeyError, IndexError, subprocess.TimeoutExpired):
        print(json.dumps({"status": "failed", "code": "candidate_material_validation_failed"}), file=sys.stderr)
        return 1


if __name__ == "__main__":
    sys.exit(main())
