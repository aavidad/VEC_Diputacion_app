#!/usr/bin/env python3
"""Validate an acknowledged synthetic source and report offline blockers.

The acknowledgement is pinned to Direction's exact message, not a caller flag.
CAS/material/alias JSON is untrusted schema input. Approved producer receipts,
physical RESTORE/CID binding and a verified alias export are pending. This cut
cannot emit executable snapshots, regardless of the caller's input files.
"""
from __future__ import annotations

import argparse
import copy
import ctypes
from datetime import datetime, timezone
import fcntl
import hashlib
import json
import os
from pathlib import Path
import re
import secrets
import stat
import sys

SOURCE_SHA256 = "f7fcd35db52d1776e7f0ea34dabbe5c86466c24b352b0007690470f32a4df226"
ACK_MESSAGE_SHA256 = "5a06d3b70cf746a10e742a8227b63f47c6a1d669dabb66203963b0a5426b1c17"
AUTHORITY = "direccion-vec:sintetico:20261001"
RESPONSIBLE = "Dirección VEC"
ACK_TIME = "2026-09-30T23:53:00Z"
MAX_FILE = 256 * 1024
POPULATIONS = ("candidato", "usuarios")
CAS_FIELDS = {"revision_control_rol", "huella_control_rol", "version_asignacion",
              "huella_asignacion", "version_contexto", "huella_contexto", "secuencia_motivos"}
MEASUREMENTS = {"cuenta", "alias", "contexto", "rol", "control_rol", "asignacion", "checkpoint", "catalogos"}
CATALOGS = {"motivos_mi_bolsa_desarrollo", "motivos_historial_mi_bolsa_desarrollo", "motivos_portal_mi_bolsa_desarrollo"}
ROLE_REF = "rol:candidato_bolsa_portal_historial_propio_desarrollo:v1"
TRUST_BLOCKERS = ("recibos_productores_aprobados_pendientes", "enlace_fisico_restore_cid_pendiente",
                  "alias_exportado_verificado_pendiente")
BLOCKERS = set(TRUST_BLOCKERS) | {"preimagen_cas_nominal_pendiente", "contexto_clon_nominal_pendiente",
                               "material_externo_validado_pendiente", "alias_externo_nominal_pendiente"}


class AccreditationError(RuntimeError):
    """Only nominal codes may escape to the command's output."""


def require(condition: bool, code: str) -> None:
    if not condition:
        raise AccreditationError(code)


def sha(data: bytes) -> str:
    return hashlib.sha256(data).hexdigest()


def encoded(value: dict) -> bytes:
    return (json.dumps(value, sort_keys=True, indent=2, ensure_ascii=False) + "\n").encode()


def is_hash(value: object) -> bool:
    return isinstance(value, str) and re.fullmatch(r"[0-9a-f]{64}", value) is not None


def integer(value: object, maximum: int) -> bool:
    return type(value) is int and 0 <= value < maximum


def keys(value: object, expected: set[str], code: str) -> None:
    require(isinstance(value, dict) and set(value) == expected, code)


def unique(pairs: list) -> dict:
    result = {}
    for key, value in pairs:
        require(key not in result, "duplicate_json_key")
        result[key] = value
    return result


def decoded(data: bytes) -> dict:
    require(0 < len(data) <= MAX_FILE, "json_size_invalid")
    value = json.loads(data, object_pairs_hook=unique,
                       parse_constant=lambda _: require(False, "json_number_invalid"))
    require(isinstance(value, dict), "json_object_required")
    return value


def instant(value: str) -> datetime:
    require(isinstance(value, str) and bool(re.fullmatch(
        r"\d{4}-\d\d-\d\dT\d\d:\d\d:\d\d(?:\.\d{6})?Z", value)), "instant_invalid")
    return datetime.fromisoformat(value.replace("Z", "+00:00"))


def reference(prefix: str, component: str) -> str:
    # Same deterministic reference contract as fuente_sintetica_candidatos.
    return prefix + sha((AUTHORITY + "\0" + component).encode())[:32]


def validate_accreditation(source_bytes: bytes, ack: dict, now: datetime) -> dict:
    """Return source metadata only; it grants no permission or executable snapshot."""
    require(sha(source_bytes) == SOURCE_SHA256, "source_sha_mismatch")
    source = decoded(source_bytes)
    keys(source, {"version", "esquema", "estado", "autoridad_maestra", "responsable",
                  "procedencia_ref", "procedencia_version", "datos_sinteticos", "personas",
                  "cuentas", "candidatoBolsa", "vigente_desde_propuesto",
                  "vigente_hasta_propuesto", "acreditacion"}, "source_schema_invalid")
    require(type(source["version"]) is int and source["version"] == 1 and
            source["esquema"] == "vec.fuente.sintetica.externa.propuesta.v1" and
            source["estado"] == "propuesta" and source["datos_sinteticos"] is True and
            source["autoridad_maestra"] == AUTHORITY and source["responsable"] == RESPONSIBLE and
            source["acreditacion"] is None, "source_authority_invalid")
    require(isinstance(source["personas"], list) and len(source["personas"]) == 1 and
            isinstance(source["cuentas"], list) and len(source["cuentas"]) == 1,
            "source_population_ambiguous")
    person, account, candidate = source["personas"][0], source["cuentas"][0], source["candidatoBolsa"]
    keys(person, {"persona_ref", "nombre_visible"}, "source_person_invalid")
    keys(account, {"cuenta_ref", "persona_ref", "sujeto", "poblaciones"}, "source_account_invalid")
    keys(candidate, {"cuenta_ref", "persona_ref", "perfil_ref", "candidato_ref"}, "source_candidate_invalid")
    refs = {key: reference(prefix, key) for key, prefix in
            (("cuenta", "cta_"), ("persona", "per_"), ("perfil", "prf_"),
             ("contexto", "vca_"), ("vinculo", "vin_"), ("candidato", "can_"), ("procedencia", "prc_"))}
    require(all(candidate[key + "_ref"] == refs[key] for key in ("cuenta", "persona", "perfil", "candidato")) and
            account["cuenta_ref"] == refs["cuenta"] and account["persona_ref"] == refs["persona"] and
            person["persona_ref"] == refs["persona"] and
            account["poblaciones"] == list(POPULATIONS) and
            account["sujeto"] == "desarrollo:clon-recorridos:candidato" and
            isinstance(person["nombre_visible"], str) and bool(person["nombre_visible"].strip()) and
            source["procedencia_ref"] == refs["procedencia"] and
            type(source["procedencia_version"]) is int and source["procedencia_version"] == 1,
            "source_references_invalid")
    since, until = source["vigente_desde_propuesto"], source["vigente_hasta_propuesto"]
    require(now.tzinfo is not None and instant(since) <= now < instant(until), "source_not_current")
    keys(ack, {"version", "kind", "fuente_sha256", "autoridad_maestra", "responsable",
               "acreditado_en", "mensaje", "mensaje_sha256"}, "ack_schema_invalid")
    require(type(ack["version"]) is int and ack["version"] == 1 and
            ack["kind"] == "acuse_fuente_sintetica_v1" and ack["fuente_sha256"] == SOURCE_SHA256 and
            ack["autoridad_maestra"] == AUTHORITY and ack["responsable"] == RESPONSIBLE and
            ack["acreditado_en"] == ACK_TIME and instant(ACK_TIME) <= now and
            isinstance(ack["mensaje"], str) and
            sha(ack["mensaje"].encode()) == ack["mensaje_sha256"] == ACK_MESSAGE_SHA256,
            "ack_mismatch")
    return {"version": 1, "source_sha256": SOURCE_SHA256, "autoridad_maestra": AUTHORITY,
            "responsable": RESPONSIBLE, "acreditado_en": ACK_TIME, "refs": refs,
            "sujeto": account["sujeto"], "vigente_desde": since, "vigente_hasta": until}


def validate_proposal(value: dict, population: str, accredited: dict) -> dict:
    keys(value, {"version", "snapshot", "preimagen"}, "proposal_schema_invalid")
    require(type(value["version"]) is int and value["version"] == 1 and value["preimagen"] is None,
            "proposal_cas_not_placeholder")
    snapshot = value["snapshot"]
    keys(snapshot, {"provision_ref", "poblacion", "estado", "cuenta", "persona", "perfil",
                    "contexto", "vinculo_candidato"}, "snapshot_schema_invalid")
    require(snapshot["provision_ref"] == reference("pce_" if population == "candidato" else "pue_", population) and
            snapshot["poblacion"] == population and snapshot["estado"] == "activo", "snapshot_nominal_invalid")
    for name in ("cuenta", "persona", "perfil", "contexto", "vinculo"):
        component = snapshot["vinculo_candidato"] if name == "vinculo" else snapshot[name]
        if name == "vinculo" and population == "usuarios":
            require(component is None, "users_candidate_link_invalid")
            continue
        expected = {"referencia": accredited["refs"][name], "version": 1,
                    "procedencia_ref": accredited["refs"]["procedencia"], "procedencia_version": 1,
                    "procedencia_huella_sha256": SOURCE_SHA256, "procedencia_autoridad": "propuesta",
                    "estado": "activo", "vigente_desde": accredited["vigente_desde"],
                    "vigente_hasta": accredited["vigente_hasta"]}
        if name == "vinculo":
            expected["candidato_ref"] = accredited["refs"]["candidato"]
        require(component == expected and type(component["version"]) is int and
                type(component["procedencia_version"]) is int, "snapshot_reference_changed")
    return copy.deepcopy(snapshot)


def validate_preimage(preimage: dict) -> None:
    keys(preimage, CAS_FIELDS, "cas_fields_invalid")
    for number, fingerprint in (("revision_control_rol", "huella_control_rol"),
                                ("version_asignacion", "huella_asignacion"),
                                ("version_contexto", "huella_contexto")):
        require(integer(preimage[number], 1 << 31) and
                ((preimage[number] == 0 and preimage[fingerprint] == "") or
                 (preimage[number] > 0 and is_hash(preimage[fingerprint]))), "cas_value_invalid")
    require(integer(preimage["secuencia_motivos"], 1 << 62), "cas_sequence_invalid")


def validate_observations(act: dict, context: dict, snapshots: dict, now: datetime) -> dict:
    """Check untrusted structure; no live CAS or physical clone is accredited."""
    keys(act, {"version", "kind", "source_sha256", "observado_en", "pgid", "journal_sha256",
               "restore_receipt_sha256", "observaciones"}, "observations_schema_invalid")
    keys(context, {"version", "pgid", "journal_sha256", "restore_receipt_sha256"}, "clone_context_invalid")
    require(type(act["version"]) is int and act["version"] == 1 and
            type(context["version"]) is int and context["version"] == 1 and
            act["kind"] == "preimagenes_nominales_ro_v1" and act["source_sha256"] == SOURCE_SHA256 and
            instant(ACK_TIME) <= instant(act["observado_en"]) <= now, "observations_binding_invalid")
    keys(act["pgid"], {"system_identifier", "database_name", "database_oid", "pg_container_id",
                       "pg_image", "pg_image_id", "pg_volume"}, "pgid_invalid")
    require(all(isinstance(v, (str, int)) and not isinstance(v, bool) and str(v) for v in act["pgid"].values()) and
            is_hash(act["journal_sha256"]) and is_hash(act["restore_receipt_sha256"]) and
            all(act[name] == context[name] for name in ("pgid", "journal_sha256", "restore_receipt_sha256")),
            "clone_context_mismatch")
    keys(act["observaciones"], set(POPULATIONS), "observations_population_invalid")
    result = {}
    for population in POPULATIONS:
        row = act["observaciones"][population]
        keys(row, {"cuenta_ref", "perfil_ref", "provision_ref", "preimagen", "mediciones"}, "observations_row_invalid")
        snapshot = snapshots[population]
        require(row["cuenta_ref"] == snapshot["cuenta"]["referencia"] and
                row["perfil_ref"] == snapshot["perfil"]["referencia"] and
                row["provision_ref"] == snapshot["provision_ref"], "observations_reference_changed")
        p = row["preimagen"]
        validate_preimage(p)
        keys(row["mediciones"], MEASUREMENTS, "cas_not_measured")
        for name, measurement in row["mediciones"].items():
            keys(measurement, {"estado", "consulta_sha256", "valor"}, "measurement_invalid")
            require(is_hash(measurement["consulta_sha256"]) and measurement["estado"] in {"ausente", "presente"},
                    "cas_not_measured")
            absent = [] if name == "catalogos" else None
            require((measurement["estado"] == "ausente" and measurement["valor"] == absent) or
                    (measurement["estado"] == "presente" and measurement["valor"] is not None and measurement["valor"] != []),
                    "measurement_state_incoherent")
        for name, number, fingerprint, value_number, value_hash in (
                ("control_rol", "revision_control_rol", "huella_control_rol", "revision", "huella_control"),
                ("asignacion", "version_asignacion", "huella_asignacion", "version", "huella_sha256"),
                ("contexto", "version_contexto", "huella_contexto", "version", "huella_sha256")):
            m = row["mediciones"][name]
            if m["estado"] == "ausente":
                require(p[number] == 0 and p[fingerprint] == "", "cas_absence_incoherent")
            else:
                expected_keys = {value_number, value_hash} | ({"provision_ref"} if name == "contexto" else set())
                keys(m["valor"], expected_keys, "measurement_projection_invalid")
                require(p[number] > 0 and type(m["valor"][value_number]) is int and m["valor"][value_number] == p[number] and
                        m["valor"][value_hash] == p[fingerprint], "cas_present_incoherent")
                if name == "contexto":
                    require(m["valor"]["provision_ref"] == snapshot["provision_ref"], "context_reference_changed")
        checkpoint = row["mediciones"]["checkpoint"]
        if checkpoint["estado"] == "ausente":
            require(p["secuencia_motivos"] == 0, "checkpoint_absence_incoherent")
        else:
            keys(checkpoint["valor"], {"secuencia"}, "checkpoint_projection_invalid")
            require(integer(checkpoint["valor"]["secuencia"], 1 << 62) and
                    checkpoint["valor"]["secuencia"] == p["secuencia_motivos"], "checkpoint_present_incoherent")
        # The identity CLI has an explicit absence preimage, not an update/replay API.
        require(row["mediciones"]["cuenta"]["estado"] == "ausente" and
                row["mediciones"]["alias"]["estado"] == "ausente", "identity_preexisting_reconcile")
        require((row["mediciones"]["rol"]["estado"] == "ausente") ==
                (row["mediciones"]["control_rol"]["estado"] == "ausente"), "role_control_incoherent")
        role = row["mediciones"]["rol"]
        if role["estado"] == "presente":
            keys(role["valor"], {"version_rol_ref", "version", "huella_sha256"}, "role_projection_invalid")
            require(role["valor"]["version_rol_ref"] == ROLE_REF and type(role["valor"]["version"]) is int and
                    role["valor"]["version"] == 1 and is_hash(role["valor"]["huella_sha256"]), "role_nominal_invalid")
        catalogs = row["mediciones"]["catalogos"]["valor"]
        require(isinstance(catalogs, list) and len(catalogs) <= 3, "catalog_projection_invalid")
        seen = set()
        for catalog in catalogs:
            keys(catalog, {"catalogo_id", "version", "huella_sha256", "secuencia", "retirado", "entradas"},
                 "catalog_projection_invalid")
            require(isinstance(catalog["catalogo_id"], str) and catalog["catalogo_id"] in CATALOGS and
                    catalog["catalogo_id"] not in seen and type(catalog["version"]) is int and catalog["version"] == 1 and
                    is_hash(catalog["huella_sha256"]) and integer(catalog["secuencia"], 1 << 62) and
                    0 < catalog["secuencia"] <= p["secuencia_motivos"] and type(catalog["retirado"]) is bool and
                    integer(catalog["entradas"], 1 << 31) and catalog["entradas"] > 0,
                    "catalog_nominal_invalid")
            seen.add(catalog["catalogo_id"])
        result[population] = copy.deepcopy(p)
    for name in MEASUREMENTS - {"contexto"}:
        require(act["observaciones"]["candidato"]["mediciones"][name] ==
                act["observaciones"]["usuarios"]["mediciones"][name], "shared_measurement_changed")
    return result


def file_signature(info: os.stat_result) -> tuple:
    return (info.st_dev, info.st_ino, info.st_mode, info.st_uid, info.st_nlink,
            info.st_size, info.st_mtime_ns, info.st_ctime_ns)


def safe_file(fd: int) -> os.stat_result:
    info = os.fstat(fd)
    require(stat.S_ISREG(info.st_mode) and stat.S_IMODE(info.st_mode) == 0o600 and
            info.st_uid == os.getuid() and info.st_nlink == 1 and info.st_size <= MAX_FILE,
            "private_file_invalid")
    return info


class PrivateDirectory:
    """Retain every ancestor descriptor, validate links, never reopen by path."""

    def __init__(self, path: Path, *, create: bool = False):
        require(path.is_absolute() and len(path.parts) > 1 and ".." not in path.parts, "private_path_invalid")
        self.chain: list[tuple[int, str | None, tuple]] = []
        flags = os.O_RDONLY | os.O_DIRECTORY | os.O_NOFOLLOW
        try:
            fd = os.open("/", flags)
            self.chain.append((fd, None, self.identity(fd, False)))
            for index, name in enumerate(path.parts[1:], 1):
                leaf = index == len(path.parts) - 1
                if leaf and create:
                    try:
                        os.mkdir(name, 0o700, dir_fd=fd)
                        os.fsync(fd)
                    except FileExistsError:
                        pass
                fd = os.open(name, flags, dir_fd=fd)
                try:
                    identity = self.identity(fd, leaf)
                except BaseException:
                    os.close(fd)
                    raise
                self.chain.append((fd, name, identity))
            self.check()
        except BaseException:
            self.close()
            raise

    @staticmethod
    def identity(fd: int, leaf: bool) -> tuple:
        info = os.fstat(fd)
        require(stat.S_ISDIR(info.st_mode) and info.st_uid in {0, os.getuid()} and
                not info.st_mode & 0o022 and
                (not leaf or (info.st_uid == os.getuid() and stat.S_IMODE(info.st_mode) == 0o700)),
                "private_directory_invalid")
        try:
            os.stat(".git", dir_fd=fd, follow_symlinks=False)
        except FileNotFoundError:
            pass
        else:
            raise AccreditationError("private_path_inside_git")
        return info.st_dev, info.st_ino

    @property
    def fd(self) -> int:
        return self.chain[-1][0]

    def check(self) -> None:
        for index, (fd, name, identity) in enumerate(self.chain):
            require(self.identity(fd, index == len(self.chain) - 1) == identity, "private_directory_changed")
            if index:
                info = os.stat(name, dir_fd=self.chain[index - 1][0], follow_symlinks=False)
                require(stat.S_ISDIR(info.st_mode) and (info.st_dev, info.st_ino) == identity,
                        "private_directory_changed")

    def read(self, name: str) -> bytes:
        require(Path(name).name == name and name not in {"", ".", ".."}, "private_name_invalid")
        self.check()
        fd = os.open(name, os.O_RDONLY | os.O_NOFOLLOW | os.O_NONBLOCK, dir_fd=self.fd)
        try:
            before = safe_file(fd)
            chunks, total = [], 0
            while True:
                block = os.read(fd, min(65536, MAX_FILE + 1 - total))
                if not block:
                    break
                chunks.append(block)
                total += len(block)
                require(total <= MAX_FILE, "private_file_size_changed")
            after = safe_file(fd)
            linked = os.stat(name, dir_fd=self.fd, follow_symlinks=False)
            require(file_signature(before) == file_signature(after) == file_signature(linked) and
                    total == before.st_size, "private_file_changed")
            self.check()
            return b"".join(chunks)
        finally:
            os.close(fd)

    def close(self) -> None:
        for fd, _, _ in reversed(self.chain):
            os.close(fd)
        self.chain.clear()

    def __enter__(self):
        return self

    def __exit__(self, *_):
        self.close()


def read_private(path: Path) -> bytes:
    with PrivateDirectory(path.parent) as parent:
        return parent.read(path.name)


def rename_exclusive(parent: int, source: str, destination: str) -> None:
    # Linux renameat2 RENAME_NOREPLACE prevents replacement of an existing
    # directory, including one created concurrently by a noncooperating writer.
    libc = ctypes.CDLL(None, use_errno=True)
    rename = getattr(libc, "renameat2", None)
    require(rename is not None, "exclusive_rename_unavailable")
    rename.argtypes = (ctypes.c_int, ctypes.c_char_p, ctypes.c_int, ctypes.c_char_p, ctypes.c_uint)
    rename.restype = ctypes.c_int
    if rename(parent, os.fsencode(source), parent, os.fsencode(destination), 1):
        raise AccreditationError("exclusive_output_commit_failed")


def write_blocked_plan(root: Path, plan: dict) -> None:
    """Only a blocked, nonexecutable plan may be persisted; exact replay is read-only."""
    keys(plan, {"estado", "source_sha256", "snapshots", "provision_ejecutada", "blockers"}, "plan_invalid")
    require(plan["estado"] == "bloqueado" and plan["source_sha256"] == SOURCE_SHA256 and
            type(plan["snapshots"]) is int and plan["snapshots"] == 0 and plan["provision_ejecutada"] is False and
            isinstance(plan["blockers"], list) and bool(plan["blockers"]) and
            all(isinstance(code, str) and code in BLOCKERS for code in plan["blockers"]) and
            set(TRUST_BLOCKERS) <= set(plan["blockers"]), "plan_executable_rejected")
    outputs = {"plan.bloqueado-v1.json": encoded({"version": 1, "kind": "plan_fuente_bloqueado_v1", **plan})}
    with PrivateDirectory(root, create=True) as destination:
        lock = os.open(".acreditacion.lock", os.O_RDWR | os.O_CREAT | os.O_NOFOLLOW | os.O_NONBLOCK,
                       0o600, dir_fd=destination.fd)
        try:
            safe_file(lock)
            fcntl.flock(lock, fcntl.LOCK_EX | fcntl.LOCK_NB)
            destination.check()
            try:
                final = os.open("plan-v1", os.O_RDONLY | os.O_DIRECTORY | os.O_NOFOLLOW,
                                dir_fd=destination.fd)
            except FileNotFoundError:
                final = None
            if final is not None:
                try:
                    PrivateDirectory.identity(final, True)
                    require(set(os.listdir(final)) == set(outputs), "replay_bundle_changed")
                    for name, expected in outputs.items():
                        fd = os.open(name, os.O_RDONLY | os.O_NOFOLLOW | os.O_NONBLOCK, dir_fd=final)
                        try:
                            before = safe_file(fd)
                            require(os.read(fd, MAX_FILE + 1) == expected and
                                    file_signature(before) == file_signature(safe_file(fd)) ==
                                    file_signature(os.stat(name, dir_fd=final, follow_symlinks=False)),
                                    "replay_bundle_changed")
                        finally:
                            os.close(fd)
                    linked = os.stat("plan-v1", dir_fd=destination.fd, follow_symlinks=False)
                    require(PrivateDirectory.identity(final, True) == (linked.st_dev, linked.st_ino) and
                            stat.S_ISDIR(linked.st_mode), "replay_directory_changed")
                    destination.check()
                    return
                finally:
                    os.close(final)
            temporary = ".acreditacion-" + secrets.token_hex(16)
            os.mkdir(temporary, 0o700, dir_fd=destination.fd)
            stage = os.open(temporary, os.O_RDONLY | os.O_DIRECTORY | os.O_NOFOLLOW, dir_fd=destination.fd)
            try:
                stage_identity = PrivateDirectory.identity(stage, True)
                for name, data in outputs.items():
                    destination.check()
                    fd = os.open(name, os.O_WRONLY | os.O_CREAT | os.O_EXCL | os.O_NOFOLLOW,
                                 0o600, dir_fd=stage)
                    with os.fdopen(fd, "wb") as stream:
                        stream.write(data)
                        stream.flush()
                        os.fsync(stream.fileno())
                        safe_file(stream.fileno())
                os.fsync(stage)
                destination.check()
                linked = os.stat(temporary, dir_fd=destination.fd, follow_symlinks=False)
                require(stat.S_ISDIR(linked.st_mode) and (linked.st_dev, linked.st_ino) == stage_identity,
                        "stage_directory_changed")
                rename_exclusive(destination.fd, temporary, "plan-v1")
                destination.check()
                linked = os.stat("plan-v1", dir_fd=destination.fd, follow_symlinks=False)
                require(stat.S_ISDIR(linked.st_mode) and (linked.st_dev, linked.st_ino) == stage_identity,
                        "output_directory_changed")
                os.fsync(destination.fd)
            finally:
                os.close(stage)
        finally:
            os.close(lock)


def read_relative(root: Path, name: str) -> bytes:
    parts = name.split("/")
    require(all(part not in {"", ".", ".."} for part in parts) and not name.startswith("/"),
            "material_relative_path_invalid")
    return read_private(root.joinpath(*parts))


def validate_material(root: Path, act_bytes: bytes, ack_bytes: bytes, accredited: dict) -> dict:
    """Inspect untrusted inventory and hashes; this does not verify cryptography or origin."""
    act = decoded(act_bytes)
    keys(act, {"version", "kind", "estado", "fuente_sha256", "acuse_sha256", "autoridad_maestra",
               "responsable", "acreditado_en", "refs", "sujeto", "vigente_desde", "vigente_hasta",
               "huella_ca_sha256", "huella_servidor_sha256", "certificate_sha256", "archivos",
               "exportador_ejecutado", "provision_ejecutada", "runtime_activado"}, "material_act_invalid")
    require(type(act["version"]) is int and act["version"] == 1 and act["kind"] == "material_externo_offline_v1" and
            act["estado"] == "preparado_offline" and act["fuente_sha256"] == SOURCE_SHA256 and
            act["acuse_sha256"] == sha(ack_bytes) and
            all(act[name] == accredited[name] for name in
                ("autoridad_maestra", "responsable", "acreditado_en", "refs", "sujeto", "vigente_desde", "vigente_hasta")) and
            all(is_hash(act[name]) for name in ("huella_ca_sha256", "huella_servidor_sha256", "certificate_sha256")) and
            all(act[name] is False for name in ("exportador_ejecutado", "provision_ejecutada", "runtime_activado")),
            "material_act_binding_invalid")
    required = {"runtime-externo/ca/ca.crt", "runtime-externo/tls/servidor.crt", "runtime-externo/tls/servidor.key",
                "runtime-externo/mtls/candidato.crt", "runtime-externo/identidad/candidato.json",
                "runtime-externo/identidad/bolsa-candidato.json", "runtime-externo/idempotencia/configuracion.json",
                "runtime-externo/manifiesto.json", "runtime-externo/portal-proceso.json",
                *{f"runtime-externo/idempotencia/g{generation}-{kind}.bin"
                  for generation in (1, 2) for kind in ("localizador", "huella-solicitud")},
                *{"custodia/" + name for name in ("ca/ca.crt", "ca/ca.key", "mtls/candidato.crt",
                  "mtls/candidato.key", "mtls/candidato.p12", "mtls/candidato.p12.password")}}
    require(isinstance(act["archivos"], dict) and set(act["archivos"]) == required, "material_inventory_incomplete")
    for name, entry in act["archivos"].items():
        require(isinstance(name, str) and name.startswith(("runtime-externo/", "custodia/")), "material_inventory_path_invalid")
        keys(entry, {"sha256", "bytes"}, "material_inventory_entry_invalid")
        require(is_hash(entry["sha256"]) and integer(entry["bytes"], MAX_FILE + 1) and entry["bytes"] > 0,
                "material_inventory_entry_invalid")
        data = read_relative(root, name)
        require(len(data) == entry["bytes"] and sha(data) == entry["sha256"], "material_preimage_changed")
    bolsa = decoded(read_relative(root, "runtime-externo/identidad/bolsa-candidato.json"))
    require(all(bolsa.get(name + "_ref") == accredited["refs"][name] for name in
                ("cuenta", "persona", "perfil", "candidato")) and bolsa.get("sujeto") == accredited["sujeto"],
            "material_references_changed")
    identity = decoded(read_relative(root, "runtime-externo/identidad/candidato.json"))
    require(identity.get("subject") == accredited["sujeto"] and
            identity.get("certificate_sha256") == act["certificate_sha256"], "material_identity_changed")
    return act


def validate_alias(alias_bytes: bytes, receipt: dict, material_bytes: bytes, accredited: dict) -> dict:
    """Inspect untrusted export structure; a label or digest is not an exporter receipt."""
    keys(receipt, {"version", "kind", "fuente_sha256", "material_acta_sha256", "exportador", "salida_sha256"},
         "alias_receipt_invalid")
    require(type(receipt["version"]) is int and receipt["version"] == 1 and
            receipt["kind"] == "exportacion_seudonimos_externos_v1" and receipt["fuente_sha256"] == SOURCE_SHA256 and
            receipt["material_acta_sha256"] == sha(material_bytes) and
            receipt["exportador"] == "cmd/vec-server exportar-seudonimos-portal-externo" and
            receipt["salida_sha256"] == sha(alias_bytes), "alias_receipt_binding_invalid")
    alias = decoded(alias_bytes)
    keys(alias, {"version", "cuentas"}, "alias_schema_invalid")
    require(type(alias["version"]) is int and alias["version"] == 1 and
            isinstance(alias["cuentas"], list) and len(alias["cuentas"]) == 1, "alias_account_ambiguous")
    account = alias["cuentas"][0]
    keys(account, {"cuenta_ref", "esquema", "dominio_ref", "clave_id", "clave_version",
                   "cuenta_id_hmac", "sujeto_id_hmac"}, "alias_account_invalid")
    domain = "idh_" + sha(("vec.ct.alta.desarrollo.v1\0https://localhost/vec/desarrollo/identidad").encode())[:32]
    require(account["cuenta_ref"] == accredited["refs"]["cuenta"] and account["esquema"] == "vec.identidad.hmac-sha256.v1" and
            account["dominio_ref"] == domain and
            isinstance(account["clave_id"], str) and
            re.fullmatch(r"vec\.identidad\.desarrollo\.externo\.g[1-9][0-9]*", account["clave_id"]) is not None and
            integer(account["clave_version"], 1 << 63) and account["clave_version"] > 0 and
            account["clave_id"] == "vec.identidad.desarrollo.externo.g" + str(account["clave_version"]) and
            all(isinstance(v, str) and all(33 <= ord(c) <= 126 for c in v) for k, v in account.items() if k != "clave_version") and
            is_hash(account["cuenta_id_hmac"]) and is_hash(account["sujeto_id_hmac"]) and
            account["cuenta_id_hmac"] != account["sujeto_id_hmac"], "alias_nominal_invalid")
    return copy.deepcopy(account)


def assemble(source_root: Path, ack_path: Path, *, now: datetime,
             observations_path: Path | None = None, clone_context_path: Path | None = None,
             material_root: Path | None = None, material_act_path: Path | None = None,
             alias_path: Path | None = None, alias_act_path: Path | None = None) -> tuple[dict, dict]:
    ack_bytes = read_private(ack_path)
    with PrivateDirectory(source_root) as source:
        accredited = validate_accreditation(source.read("fuente-sintetica-v1.json"), decoded(ack_bytes), now)
        snapshots = {population: validate_proposal(decoded(source.read(population + "-fuente.propuesta-v1.json")),
                                                  population, accredited) for population in POPULATIONS}
    # None of the following files is a trusted receipt. Their schemas can be
    # inspected for errors, but neither matching hashes nor named exporters
    # authenticate their producer. There is deliberately no success branch,
    # caller-supplied approval flag, or callback that can enable snapshots.
    blockers = list(TRUST_BLOCKERS)
    if observations_path is None:
        blockers.append("preimagen_cas_nominal_pendiente")
    if clone_context_path is None:
        blockers.append("contexto_clon_nominal_pendiente")
    if observations_path is not None and clone_context_path is not None:
        observations = read_private(observations_path)
        clone_context = read_private(clone_context_path)
        validate_observations(decoded(observations), decoded(clone_context), snapshots, now)
    material_bytes = None
    if material_root is None or material_act_path is None:
        blockers.append("material_externo_validado_pendiente")
    else:
        material_bytes = read_private(material_act_path)
        validate_material(material_root, material_bytes, ack_bytes, accredited)
    if alias_path is None or alias_act_path is None or material_bytes is None:
        blockers.append("alias_externo_nominal_pendiente")
    else:
        alias_bytes, alias_receipt = read_private(alias_path), read_private(alias_act_path)
        validate_alias(alias_bytes, decoded(alias_receipt), material_bytes, accredited)
    return {"estado": "bloqueado", "source_sha256": SOURCE_SHA256, "snapshots": 0,
            "provision_ejecutada": False, "blockers": blockers}, {}


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    for name in ("fuente", "acuse", "salida"):
        parser.add_argument("--" + name, type=Path, required=True)
    for name in ("observaciones", "contexto-clon", "material", "acta-material", "alias", "acta-alias"):
        parser.add_argument("--" + name, type=Path)
    args = parser.parse_args()
    try:
        require(args.salida != args.fuente and args.fuente not in args.salida.parents and
                args.salida not in args.fuente.parents, "output_overlaps_immutable_source")
        result, outputs = assemble(args.fuente, args.acuse, now=datetime.now(timezone.utc),
                                   observations_path=args.observaciones, clone_context_path=args.contexto_clon,
                                   material_root=args.material, material_act_path=args.acta_material,
                                   alias_path=args.alias, alias_act_path=args.acta_alias)
        require(not outputs, "executable_snapshots_unavailable")
        write_blocked_plan(args.salida, result)
    except (AccreditationError, OSError, ValueError, TypeError, KeyError) as error:
        code = str(error) if isinstance(error, AccreditationError) else "material_o_evidencia_invalida"
        print(json.dumps({"error": "fuente_acreditada_rechazada", "estado": "bloqueado", "snapshots": 0,
                          "provision_ejecutada": False, "blockers": [code]}, sort_keys=True), file=sys.stderr)
        return 2
    print(json.dumps(result, sort_keys=True))
    return 3


if __name__ == "__main__":
    sys.exit(main())
