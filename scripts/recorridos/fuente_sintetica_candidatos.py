#!/usr/bin/env python3
"""Prepare an offline synthetic source proposal; never approve or provision it."""
from __future__ import annotations

import argparse
import ast
import fcntl
import hashlib
import json
import os
from pathlib import Path
import stat
import sys

SOURCE_REF = "direccion-vec:sintetico:20261001"
BASE_COMMIT = "8c816cf872d3af3b8c1c8e030d735151e88bb163"
SUBJECT = "desarrollo:clon-recorridos:candidato"
BLOCKERS = ["acuse_direccion_sha_pendiente", "material_externo_validado_pendiente",
            "alias_externo_nominal_pendiente", "preimagen_cas_nominal_pendiente"]
MAX_FILE = 64 * 1024


class SourceError(RuntimeError):
    pass


def encoded(value: dict) -> bytes:
    return (json.dumps(value, ensure_ascii=False, sort_keys=True, indent=2) + "\n").encode()


def digest(data: bytes) -> str:
    return hashlib.sha256(data).hexdigest()


def reference(prefix: str, component: str) -> str:
    return prefix + digest((SOURCE_REF + "\0" + component).encode())[:32]


def vocabularies(repo: Path) -> tuple[list[str], list[str]]:
    # Reuse only literal dictionaries; do not import the generator, read its
    # demo outputs, run its main, or traverse its CONVOCA export paths.
    tree = ast.parse((repo / "scripts/generar_bolsa_demo.py").read_text())
    values = {}
    for node in tree.body:
        if isinstance(node, ast.Assign) and len(node.targets) == 1:
            target = node.targets[0]
            if isinstance(target, ast.Name) and target.id in {"NOMBRES", "APELLIDOS"}:
                if target.id in values:
                    raise SourceError("vocabulary_ambiguous")
                values[target.id] = ast.literal_eval(node.value)
    if set(values) != {"NOMBRES", "APELLIDOS"} or any(
            not isinstance(value, list) or len(value) < 12 or
            any(not isinstance(item, str) or not item.strip() for item in value)
            for value in values.values()):
        raise SourceError("vocabulary_invalid")
    return values["NOMBRES"], values["APELLIDOS"]


def prepare(repo: Path) -> dict[str, bytes]:
    names, surnames = vocabularies(repo)
    refs = {key: reference(prefix, key) for key, prefix in
            [("cuenta", "cta_"), ("persona", "per_"), ("perfil", "prf_"),
             ("contexto", "vca_"), ("vinculo", "vin_"), ("candidato", "can_"),
             ("procedencia", "prc_")]}
    master = {
        "version": 1, "esquema": "vec.fuente.sintetica.externa.propuesta.v1",
        "estado": "propuesta", "autoridad_maestra": SOURCE_REF,
        "responsable": "Dirección VEC", "procedencia_ref": refs["procedencia"],
        "procedencia_version": 1, "datos_sinteticos": True,
        "personas": [{"persona_ref": refs["persona"],
                      "nombre_visible": " ".join((names[0], surnames[3], surnames[10]))}],
        "cuentas": [{"cuenta_ref": refs["cuenta"], "persona_ref": refs["persona"],
                     "sujeto": SUBJECT, "poblaciones": ["candidato", "usuarios"]}],
        "candidatoBolsa": {key + "_ref": refs[key] for key in
                           ("cuenta", "persona", "perfil", "candidato")},
        "vigente_desde_propuesto": "2026-10-01T00:00:00.000000Z",
        "vigente_hasta_propuesto": "2026-10-15T00:00:00.000000Z",
        "acreditacion": None,
    }
    source = encoded(master)
    source_sha = digest(source)

    def component(key: str) -> dict:
        return {"referencia": refs[key], "version": 1,
                "procedencia_ref": refs["procedencia"], "procedencia_version": 1,
                "procedencia_huella_sha256": source_sha,
                "procedencia_autoridad": "propuesta", "estado": "activo",
                "vigente_desde": master["vigente_desde_propuesto"],
                "vigente_hasta": master["vigente_hasta_propuesto"]}

    outputs = {"fuente-sintetica-v1.json": source}
    for population, prefix in (("candidato", "pce_"), ("usuarios", "pue_")):
        snapshot = {"provision_ref": reference(prefix, population),
                    "poblacion": population, "estado": "activo",
                    **{key: component(key) for key in ("cuenta", "persona", "perfil", "contexto")},
                    "vinculo_candidato": None}
        if population == "candidato":
            snapshot["vinculo_candidato"] = component("vinculo") | {"candidato_ref": refs["candidato"]}
        # Null marks an unmeasured CAS; Go would decode it as a zero struct.
        # Authority "propuesta" is the guard that rejects the whole proposal.
        # Future accreditation requires a measured CAS before assembling input.
        outputs[population + "-fuente.propuesta-v1.json"] = encoded({
            "version": 1, "snapshot": snapshot, "preimagen": None})
    outputs["bolsa-candidato.propuesta-v1.json"] = encoded({
        "version": 1, "autoridad": "no_autoritativo", "certificado": "mtls/candidato.crt",
        "identidad": "identidad/candidato.json", "sujeto": SUBJECT,
        **master["candidatoBolsa"]})
    # This is an inventory, not runtime material. No credential, certificate,
    # alias HMAC, permission, SQL, or READY marker is emitted.
    outputs["manifiesto.propuesta-v1.json"] = encoded({
        "version": 1, "estado": "propuesta", "source_base_commit": BASE_COMMIT,
        "autoridad_maestra": SOURCE_REF, "responsable": master["responsable"],
        "fuente_sha256": source_sha, "acreditacion": None,
        "procedencia_ref": refs["procedencia"], "procedencia_version": 1,
        "personas": 1, "cuentas": 1, "snapshots": 2, "provision_ejecutada": False,
        "fases_candidato": ["identidad", "contexto", "autorizacion", "motivos"],
        "fases_usuarios": ["identidad"], "blockers": BLOCKERS,
        "preimagen_campos": ["revision_control_rol", "huella_control_rol", "version_asignacion",
                             "huella_asignacion", "version_contexto", "huella_contexto", "secuencia_motivos"],
        "alias_campos": ["cuenta_ref", "esquema", "dominio_ref", "clave_id", "clave_version",
                         "cuenta_id_hmac", "sujeto_id_hmac"],
        "material_externo_requerido": ["ca/ca.crt", "tls/servidor.crt", "tls/servidor.key",
            "manifiesto.json", "idempotencia/configuracion.json", "mtls/candidato.crt",
            "identidad/candidato.json", "identidad/bolsa-candidato.json"],
        "usuarios_alias_adicionales": "identidad/usuarios-preferencias-externa.json",
        "exportador": "cmd/vec-server exportar-seudonimos-portal-externo",
        "provisionador": "cmd/vec-provisionar-candidato-externo",
        "archivos": {name: {"sha256": digest(data), "bytes": len(data)}
                     for name, data in outputs.items()},
    })
    return outputs


def safe_file(fd: int) -> os.stat_result:
    info = os.fstat(fd)
    if (not stat.S_ISREG(info.st_mode) or stat.S_IMODE(info.st_mode) != 0o600 or
            info.st_nlink != 1 or info.st_uid != os.getuid() or info.st_size > MAX_FILE):
        raise SourceError("private_file_invalid")
    return info


def directory_identity(fd: int, *, leaf: bool) -> tuple[int, int]:
    info = os.fstat(fd)
    if (not stat.S_ISDIR(info.st_mode) or info.st_uid not in {0, os.getuid()} or
            info.st_mode & 0o022 or
            (leaf and (stat.S_IMODE(info.st_mode) != 0o700 or info.st_uid != os.getuid()))):
        raise SourceError("private_root_permissions")
    try:
        os.stat(".git", dir_fd=fd, follow_symlinks=False)
    except FileNotFoundError:
        pass
    else:
        raise SourceError("private_root_inside_git")
    return info.st_dev, info.st_ino


def private_root(root: Path, *, create: bool = True) -> tuple[int, tuple]:
    if not root.is_absolute() or len(root.parts) < 2 or ".." in root.parts:
        raise SourceError("private_root_invalid")
    # Resolve each name relative to the retained parent descriptor. Checking
    # resolve() and then opening a complete path would leave ancestors exposed.
    flags = os.O_RDONLY | os.O_DIRECTORY | os.O_NOFOLLOW
    fd = os.open("/", flags)
    identities = []
    try:
        identities.append(directory_identity(fd, leaf=False))
        for index, name in enumerate(root.parts[1:], start=1):
            leaf = index == len(root.parts) - 1
            if leaf and create:
                try:
                    os.mkdir(name, mode=0o700, dir_fd=fd)
                except FileExistsError:
                    pass
            try:
                child_fd = os.open(name, flags, dir_fd=fd)
            except OSError as error:
                raise SourceError("private_root_invalid") from error
            os.close(fd)
            fd = child_fd
            identities.append(directory_identity(fd, leaf=leaf))
        return fd, tuple(identities)
    except BaseException:
        os.close(fd)
        raise


def revalidate_root(root: Path, identities: tuple) -> None:
    fresh_fd, current = private_root(root, create=False)
    try:
        if current != identities:
            raise SourceError("private_root_changed")
    finally:
        os.close(fresh_fd)


def write_proposal(root: Path, outputs: dict[str, bytes]) -> dict:
    root_fd, identities = private_root(root)
    lock_fd = None
    try:
        revalidate_root(root, identities)
        lock_fd = os.open(".fuente.lock", os.O_RDWR | os.O_CREAT | os.O_NOFOLLOW | os.O_NONBLOCK,
                          0o600, dir_fd=root_fd)
        safe_file(lock_fd)
        fcntl.flock(lock_fd, fcntl.LOCK_EX | fcntl.LOCK_NB)
        absent = []
        for name, data in outputs.items():
            if Path(name).name != name or not name.endswith(".json") or len(data) > MAX_FILE:
                raise SourceError("output_invalid")
            try:
                fd = os.open(name, os.O_RDONLY | os.O_NOFOLLOW | os.O_NONBLOCK, dir_fd=root_fd)
            except FileNotFoundError:
                absent.append((name, data))
                continue
            try:
                before = safe_file(fd)
                previous = os.read(fd, MAX_FILE + 1)
                after = os.fstat(fd)
                if (len(previous) != before.st_size or previous != data or
                        (before.st_size, before.st_mtime_ns, before.st_ctime_ns) !=
                        (after.st_size, after.st_mtime_ns, after.st_ctime_ns)):
                    raise SourceError("proposal_preimage_changed")
            finally:
                os.close(fd)
        for name, data in absent:
            revalidate_root(root, identities)
            fd = os.open(name, os.O_WRONLY | os.O_CREAT | os.O_EXCL | os.O_NOFOLLOW,
                         0o600, dir_fd=root_fd)
            with os.fdopen(fd, "wb") as stream:
                stream.write(data)
                stream.flush()
                os.fsync(stream.fileno())
                safe_file(stream.fileno())
        revalidate_root(root, identities)
        os.fsync(root_fd)
    finally:
        if lock_fd is not None:
            os.close(lock_fd)
        os.close(root_fd)
    return {"estado": "propuesta", "personas": 1, "cuentas": 1, "snapshots": 2,
            "fuente_sha256": digest(outputs["fuente-sintetica-v1.json"]),
            "archivos_sha256": {name: digest(data) for name, data in outputs.items()},
            "blockers": BLOCKERS}


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--directorio", type=Path, required=True)
    args = parser.parse_args()
    try:
        result = write_proposal(args.directorio, prepare(Path(__file__).resolve().parents[2]))
    except (SourceError, OSError, ValueError, SyntaxError):
        print('{"error":"fuente_sintetica_propuesta_rechazada"}', file=sys.stderr)
        return 2
    print(json.dumps(result, sort_keys=True))
    return 0


if __name__ == "__main__":
    sys.exit(main())
