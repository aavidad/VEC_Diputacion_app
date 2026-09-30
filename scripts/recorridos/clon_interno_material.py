#!/usr/bin/env python3
"""Project sealed synthetic operator material into an internal-only runtime root."""
from __future__ import annotations

import argparse
import fcntl
import hashlib
import json
import os
from pathlib import Path
import re
import shlex
import stat
import tempfile
from urllib.parse import parse_qsl, urlencode, urlsplit, urlunsplit

OWNER = "Codex-M"
GUARD = "ACEPTO_CREDENCIALES_NO_AUTORITATIVAS_SOLO_DESARROLLO"
REQUIRED = (
    "ca/ca.crt", "tls/servidor.crt", "tls/servidor.key", "mtls/cliente.crt", "mtls/intervencion.crt",
    "identidad/identidad.json", "identidad/intervencion.json", "manifiesto.json",
    "kms/clave-maestra.bin", "kms/atestacion-ed25519.key", "kms/atestacion-ed25519.pub",
    "kms/revalidacion-ed25519.key", "kms/revalidacion-ed25519.pub", "tsa/clave-hmac.bin",
    "idempotencia/configuracion.json", "idempotencia/g1-localizador.bin", "idempotencia/g1-huella-solicitud.bin",
    "idempotencia/g2-localizador.bin", "idempotencia/g2-huella-solicitud.bin", "pg/ca.crt",
    "identidad/usuarios-preferencias-interna.json",
)
OPTIONAL = (
    "mtls/solicitante.crt", "mtls/ratificador.crt", "identidad/solicitante.json", "identidad/ratificador.json",
    "identidad/centros.json", "identidad/consultas-rrhh.json", "identidad/bolsa-bback.json", "identidad/documentos.json",
)
DATABASE_KEYS = (
    "VEC_CT_DATABASE_URL", "VEC_CT_GOBIERNO_DATABASE_URL", "VEC_CT_REGISTRO_AUTORIZACION_DATABASE_URL",
    "VEC_CT_CONFIRMADOR_DATABASE_URL", "VEC_CT_LECTOR_RESULTADO_DATABASE_URL", "VEC_BOLSA_LLAMAMIENTOS_DATABASE_URL",
    "VEC_CT_CONSULTAS_RRHH_DATABASE_URL", "VEC_CT_MOTIVOS_RRHH_DATABASE_URL", "VEC_CT_REGISTRO_IDENTIDAD_DATABASE_URL",
    "VEC_CT_REVALIDACION_IDENTIDAD_DATABASE_URL", "VEC_CT_CONTEXTO_ACTOR_DATABASE_URL", "VEC_CT_AUDITORIA_FRONTERA_DATABASE_URL",
    "VEC_AUTORIZACION_FUENTE_DATABASE_URL", "VEC_AUTORIZACION_MOTIVOS_EVALUADOR_DATABASE_URL",
    "VEC_BOLSA_AUDITORIA_FRONTERA_DATABASE_URL", "VEC_BOLSA_POLITICA_OFERTAS_CALCULADOR_DATABASE_URL",
    "VEC_BOLSA_IMPORTACION_CONVOCA_DATABASE_URL",
)
ENV_KEYS = set(DATABASE_KEYS) | {
    "VEC_EXECUTION_PROFILE", "VEC_AUTH_MODE", "VEC_DEVELOPMENT_GUARD", "VEC_HTTP_ADDR", "VEC_HTTP_IDLE_TIMEOUT",
    "VEC_PERSONAL_ORGANIZACION_POSTGRESQL", "VEC_PERSONAL_ORGANIZACION_VERSION", "VEC_BOLSA_BORRADORES_ENABLED",
    "VEC_BOLSA_POLITICA_OFERTAS_ENABLED", "VEC_USUARIOS_PREFERENCIAS_ENABLED", "VEC_USUARIOS_CORREOS_ENABLED",
    "VEC_USUARIOS_IMAGEN_ENABLED", "VEC_SMTP_HOST", "VEC_SMTP_PORT", "VEC_SMTP_FROM", "VEC_SMTP_MODO_TLS",
    "VEC_DOCUMENTOS_ENABLED", "VEC_RRHH_AUDITORIA_ENABLED", "VEC_AUDITORIA_CONSULTA_EXPEDIENTE_CT",
    "VEC_AUDITORIA_CONSULTA_EXPEDIENTE_BOLSA", "VEC_CT_FIRMA_REGISTRO_ENABLED", "VEC_CT_SEGUIMIENTO_CESE_ENABLED",
    "VEC_CT_CANCELACION_ENABLED", "VEC_CT_INCORPORACION_ACREDITADA_ENABLED",
}
RW_ENV = {"VEC_BOLSA_DATA_DIR": "data/bolsa", "VEC_BOLSA_DATA_PATH": "data/bolsa/bolsa_store.json",
          "VEC_PERSONAL_CATALOG_PATH": "data/personal-catalog.json", "VEC_BOLSA_IMPORTACION_CONVOCA_CUSTODIA_DIR": "data/importaciones"}
SOURCE_PATHS = (
    "config/portal_proceso.go", "internal/app/separacionportales/material.go",
    "internal/app/bootstrap/material_desarrollo.go", "internal/app/bootstrap/usuarios_preferencias_config_identidad.go",
    "internal/app/bootstrap/documentos_montaje.go", "internal/app/bootstrap/usuarios_imagen_montaje.go",
    "config/postgresql_importacion_convoca.go", "internal/app/bootstrap/bolsa_importacion_convoca_pool.go",
    "internal/app/bootstrap/bolsa_importacion_convoca_custodia.go",
)
# Reviewed portal/material contracts at main 8fc0b534; SQL lineage alone
# cannot approve changes to these Go loaders or their classification.
APPROVED_CONTRACTS = dict(zip(SOURCE_PATHS, (
    "083e46c4ac92b85d32386820038b3f5264f6af974550cea1850b7bfaf713ea89",
    "db7f6f31d99173185774cf046f9e0e1abd55362c2d0741e4ca131592eb9a541f",
    "6f267a8a72b137e672289c9381d7a4ad71b033c4d69ead5c0d2a875e0fe066a6",
    "6fb7e6f198f30613040210088f0e285bfbff11db8e0161936888b7bcbf8c0e32",
    "3ee0ac2917e03b1093b528781b2af25238600b7400d4d30f78c836d26355665a",
    "dcfb4d6c6eb4f438993de0ab6c027b21b429826a601519fafdeec3ef2ed68bf8",
    "b30ac7f1c8f92d95a251704f0c128315070eafc2a0c743a08c627b72e7e39ece",
    "5707fbbe5c78c4b7b48267b8bb39f0095e224eddbd3b16c071d216daad1b72be",
    "8b3c2909bc3278c9937f45935f54774edd141230e8824022657742a159549d04",
)))


class ProjectionError(RuntimeError):
    pass


def fail(code):
    raise ProjectionError(code)


def canonical(path):
    path = Path(path).absolute()
    if path.resolve() != path:
        fail("projection_symlink_path")
    return path


def read(path):
    canonical(path)
    fd = os.open(path, os.O_RDONLY | os.O_NOFOLLOW | os.O_NONBLOCK)
    with os.fdopen(fd, "rb") as stream:
        info = os.fstat(stream.fileno())
        if not stat.S_ISREG(info.st_mode) or info.st_nlink != 1 or info.st_uid != os.getuid() or info.st_mode & 0o077 or not 0 < info.st_size <= 262144:
            fail("projection_unsafe_source_file")
        return stream.read(262145)


def digest(data):
    return hashlib.sha256(data).hexdigest()


def json_bytes(value):
    return (json.dumps(value, ensure_ascii=False, indent=2) + "\n").encode()


def operator_digest(principal):
    normalized = {key: value for key, value in principal.items() if key != "runtime_interno"}
    return digest(json.dumps(normalized, sort_keys=True, separators=(",", ":")).encode())


def private_dirs(path):
    missing = []
    while not path.exists():
        missing.append(path)
        path = path.parent
    for directory in reversed(missing):
        directory.mkdir(mode=0o700)


def write(path, data):
    private_dirs(path.parent)
    fd = os.open(path, os.O_WRONLY | os.O_CREAT | os.O_EXCL | os.O_NOFOLLOW, 0o600)
    with os.fdopen(fd, "wb") as stream:
        stream.write(data)
        stream.flush()
        os.fsync(stream.fileno())


def source_contracts(repo, source):
    import subprocess
    result = {}
    contents = {}
    for relative in SOURCE_PATHS:
        completed = subprocess.run(["git", "-C", str(repo), "show", source + ":" + relative], capture_output=True, timeout=20)
        if completed.returncode:
            fail("projection_source_contract_unavailable")
        contents[relative] = completed.stdout.decode()
        result[relative] = digest(completed.stdout)
    if result != APPROVED_CONTRACTS:
        fail("projection_source_contract_review_required")
    if '"VEC_PORTAL_PROCESO"' not in contents[SOURCE_PATHS[0]] or '"interno"' not in contents[SOURCE_PATHS[0]] or "portalProcesoSeparado(cfg)" not in contents[SOURCE_PATHS[2]] or "!superficieExternaUsuariosEnProceso(cfg)" not in contents[SOURCE_PATHS[3]]:
        fail("projection_source_internal_contract_missing")
    return result


def rewrite_dsn(value, old_material, new_material, pg_port):
    parsed = urlsplit(value)
    pairs = parse_qsl(parsed.query, strict_parsing=True)
    parameters = dict(pairs)
    if parsed.scheme not in ("postgres", "postgresql") or parsed.hostname != "127.0.0.1" or parsed.port != pg_port or parsed.path != "/postgres" or not parsed.username or parsed.fragment or len(parameters) != len(pairs):
        fail("projection_dsn_outside_owned_clone")
    if set(parameters) != {"sslmode", "sslrootcert"} or parameters["sslmode"] != "verify-full" or parameters["sslrootcert"] != str(old_material / "pg/ca.crt"):
        fail("projection_dsn_unapproved_parameters")
    parameters["sslrootcert"] = str(new_material / "pg/ca.crt")
    return urlunsplit((parsed.scheme, parsed.netloc, parsed.path, urlencode(parameters), ""))


def rewrite_json(value, old_material, new_material, root, selected, pg_port):
    if isinstance(value, dict):
        return {k: rewrite_json(v, old_material, new_material, root, selected, pg_port) for k, v in value.items()}
    if isinstance(value, list):
        return [rewrite_json(v, old_material, new_material, root, selected, pg_port) for v in value]
    if not isinstance(value, str):
        return value
    if value.startswith(("postgres://", "postgresql://")):
        return rewrite_dsn(value, old_material, new_material, pg_port)
    if value.startswith(("mtls/", "identidad/", "externo/")) and value not in selected:
        fail("projection_json_references_excluded_identity")
    if value.startswith(str(old_material) + "/"):
        relative = str(Path(value).relative_to(old_material))
        if relative not in selected:
            fail("projection_json_references_excluded_path")
        return str(new_material / relative)
    if value.startswith("/"):
        fail("projection_json_absolute_path_not_classified")
    return value


def preflight(repo, container, state, material, pg_port, source_context):
    repo, state, material = map(canonical, (repo, state, material))
    if material != state / "material" or state.stat().st_uid != os.getuid() or state.stat().st_mode & 0o077:
        fail("projection_invalid_operator_root")
    records = []
    for record in ("clon.json", "DB_READY.json"):
        metadata = json.loads(read(state / record))
        if metadata.get("propietario") != OWNER or metadata.get("estado") != str(state) or metadata.get("contenedor") != container or metadata.get("puerto_pg") != pg_port:
            fail("projection_clone_inventory_mismatch")
        records.append(metadata)
    principal_bytes = read(state / "material-manifest.json")
    principal = json.loads(principal_bytes)
    env_bytes = read(state / "runtime-config.json")
    env = json.loads(env_bytes)
    source = principal["target"]["source_commit"]
    if principal.get("owner") != OWNER or not re.fullmatch(r"[0-9a-f]{40}", source) or principal["target"]["pg_port"] != pg_port or (source_context and source_context["source_ref"] != source):
        fail("projection_parent_target_mismatch")
    if any(record.get("commit") != source for record in records):
        fail("projection_source_marker_mismatch")
    expected_env = {"VEC_EXECUTION_PROFILE": "desarrollo", "VEC_AUTH_MODE": "desarrollo", "VEC_DEVELOPMENT_GUARD": GUARD,
                    "VEC_DEVELOPMENT_MATERIAL_DIR": str(material), "VEC_HTTP_ADDR": "127.0.0.1:" + str(principal["target"]["app_port"])}
    if any(env.get(k) != v for k, v in expected_env.items()) or "runtime-config.json" not in principal["files"]:
        fail("projection_parent_environment_mismatch")
    for relative, expected in principal["files"].items():
        if Path(relative).is_absolute() or ".." in Path(relative).parts or digest(read(state / relative)) != expected:
            fail("projection_parent_seal_changed")
    contracts = source_contracts(repo, source)
    root = canonical(state / "runtime-interno")
    projected_material = root / "material"
    selected = set(REQUIRED) | {p for p in OPTIONAL if (material / p).exists()}
    if env.get("VEC_DOCUMENTOS_ENABLED") == "true" and "identidad/documentos.json" not in selected:
        fail("projection_documents_config_missing")
    payload = {}
    copied = {}
    for relative in sorted(selected):
        if "material/" + relative not in principal["files"]:
            fail("projection_selected_source_not_sealed")
        original = read(material / relative)
        output = original
        if relative.endswith(".json"):
            value = json.loads(original)
            if relative == "identidad/usuarios-preferencias-interna.json" and (value.get("version") != 1 or value.get("autoridad") != "no_autoritativo" or value.get("superficie") != "interna_corporativa" or not value.get("cuentas")):
                fail("projection_internal_Users_surface_changed")
            if relative == "identidad/documentos.json":
                if value.get("almacen", {}).get("tipo", "ficheros") != "ficheros":
                    fail("projection_documents_storage_not_local")
                original_store = value["almacen"].get("directorio")
                if original_store and Path(original_store).exists() and any(Path(original_store).rglob("*")):
                    fail("projection_existing_document_objects_need_positive_snapshot")
                value["almacen"]["directorio"] = str(root / "rw/documentos")
                # Classified writable path, removed before generic path processing.
                writable = value["almacen"].pop("directorio")
                value = rewrite_json(value, material, projected_material, root, selected, pg_port)
                value["almacen"]["directorio"] = writable
            else:
                value = rewrite_json(value, material, projected_material, root, selected, pg_port)
            rewritten = json_bytes(value)
            if value != json.loads(original):
                output = rewritten
        payload["material/" + relative] = output
        copied[relative] = {"source_sha256": digest(original), "projected_sha256": digest(output), "unchanged": output == original}
    projected_env = {key: value for key, value in env.items() if key in ENV_KEYS}
    if not all(isinstance(v, str) and "\0" not in v and "\n" not in v for v in projected_env.values()):
        fail("projection_invalid_environment_value")
    for key in DATABASE_KEYS:
        if key in projected_env:
            projected_env[key] = rewrite_dsn(projected_env[key], material, projected_material, pg_port)
    projected_env.update(VEC_PORTAL_PROCESO="interno", VEC_DEVELOPMENT_MATERIAL_DIR=str(projected_material),
                         VEC_TLS_CERT_FILE=str(projected_material / "tls/servidor.crt"), VEC_TLS_KEY_FILE=str(projected_material / "tls/servidor.key"))
    if env.get("VEC_SMTP_CA_FILE"):
        if env["VEC_SMTP_CA_FILE"] != str(material / "ca/ca.crt"):
            fail("projection_unclassified_SMTP_CA")
        projected_env["VEC_SMTP_CA_FILE"] = str(projected_material / "ca/ca.crt")
    for key, relative in RW_ENV.items():
        if env.get(key) and Path(env[key]).exists():
            existing = canonical(env[key])
            if existing.is_file() or (existing.is_dir() and any(existing.iterdir())):
                fail("projection_existing_runtime_data_need_positive_snapshot")
        projected_env[key] = str(root / "rw" / relative)
    payload["material/portal-proceso.json"] = json_bytes({"version": 1, "portal": "interno"})
    payload["material/desarrollo.env"] = "".join(k + "=" + shlex.quote(v) + "\n" for k, v in sorted(projected_env.items())).encode()
    payload["runtime-config.json"] = json_bytes(projected_env)
    payload["runtime.env"] = payload["material/desarrollo.env"]
    manifest = {"version": 1, "owner": OWNER, "portal": "interno", "mode": "interno", "target": principal["target"],
                "status": "prepared", "files": {k: digest(v) for k, v in payload.items()},
                "source_proof": {"operator_manifest_sha256": operator_digest(principal), "operator_manifest_normalization": "drop_runtime_interno_only", "operator_env_sha256": digest(env_bytes),
                                 "contracts": contracts, "positive_files": copied},
                "source_sql_approval": principal.get("source_sql_approval", {}), "application_started": False,
                "external_profiles_available": False, "operator_material_mounted": False}
    payload["material-manifest.json"] = json_bytes(manifest)
    rw = [{"source": "runtime-interno/rw/" + p, "target": str(root / "rw" / p), "kind": p} for p in ("documentos", "imagenes", "data")]
    rw.append({"source": "runtime-interno/rw/comunicaciones", "target": str(projected_material / "comunicaciones"), "kind": "comunicaciones"})
    return root, payload, {"mode": "interno", "portal": "interno", "material": "runtime-interno/material",
                          "config": "runtime-interno/runtime-config.json", "manifest": "runtime-interno/material-manifest.json",
                          "rw": rw, "manifest_sha256": digest(payload["material-manifest.json"]), "source_commit": source}


def check_closed_inventory(root, payload, descriptor):
    """Only declared immutable files/directories and four owned RW roots exist."""
    expected_files = set(payload)
    expected_dirs = {".", "material/comunicaciones"}
    for relative in expected_files:
        expected_dirs.update(str(parent) for parent in Path(relative).parents)
    writable = {str(Path(entry["source"]).relative_to("runtime-interno")) for entry in descriptor["rw"]}
    for relative in writable:
        expected_dirs.add(relative)
        expected_dirs.update(str(parent) for parent in Path(relative).parents)
    def visit(path):
        relative = str(path.relative_to(root))
        info = path.lstat()
        if stat.S_ISLNK(info.st_mode) or info.st_uid != os.getuid() or info.st_mode & 0o077:
            fail("projection_unsafe_inventory_entry")
        if stat.S_ISDIR(info.st_mode):
            if relative not in expected_dirs:
                fail("projection_unlisted_directory")
            if relative not in writable:
                for child in path.iterdir():
                    visit(child)
        elif not stat.S_ISREG(info.st_mode) or info.st_nlink != 1 or relative not in expected_files:
            fail("projection_unlisted_file")
    visit(root)
    for relative in expected_dirs:
        if not (root / relative).is_dir():
            fail("projection_expected_directory_missing")


def refresh_proof_reference(state, root, payload, descriptor):
    """CAS only an updated operator-manifest reference, never runtime bytes."""
    fd = os.open(state / "runtime-interno-proof.lock", os.O_WRONLY | os.O_CREAT | os.O_NOFOLLOW | os.O_NONBLOCK, 0o600)
    try:
        info = os.fstat(fd)
        if not stat.S_ISREG(info.st_mode) or info.st_nlink != 1 or info.st_uid != os.getuid() or info.st_mode & 0o077:
            fail("projection_unsafe_proof_lock")
        fcntl.flock(fd, fcntl.LOCK_EX | fcntl.LOCK_NB)
        old_bytes = read(root / "material-manifest.json")
        old = json.loads(old_bytes)
        new = json.loads(payload["material-manifest.json"])
        old_operator = old["source_proof"]["operator_manifest_sha256"]
        new_operator = new["source_proof"]["operator_manifest_sha256"]
        if not re.fullmatch(r"[0-9a-f]{64}", old_operator):
            fail("projection_invalid_proof_preimage")
        old["source_proof"]["operator_manifest_sha256"] = new_operator
        if old_operator == new_operator or json_bytes(old) != payload["material-manifest.json"]:
            fail("projection_refresh_changes_more_than_operator_reference")
        parent_bytes = read(state / "material-manifest.json")
        parent = json.loads(parent_bytes)
        previous = dict(descriptor, manifest_sha256=digest(old_bytes))
        if parent.get("runtime_interno") != previous or operator_digest(parent) != new_operator:
            fail("projection_refresh_parent_cas_mismatch")
        receipt = {"version": 1, "owner": OWNER, "source_commit": descriptor["source_commit"],
                   "effect": "operator_manifest_reference_only", "preimage_manifest_sha256": digest(old_bytes),
                   "postimage_manifest_sha256": digest(payload["material-manifest.json"]),
                   "preimage_operator_manifest_sha256": old_operator, "postimage_operator_manifest_sha256": new_operator}
        receipt_path = state / ("runtime-interno-proof-refresh-" + digest(old_bytes)[:16] + "-" + digest(payload["material-manifest.json"])[:16] + ".json")
        if receipt_path.exists():
            if read(receipt_path) != json_bytes(receipt):
                fail("projection_refresh_receipt_preimage_changed")
        else:
            write(receipt_path, json_bytes(receipt))
        temporary_fd, name = tempfile.mkstemp(prefix=".runtime-interno-proof-", dir=state)
        temporary = Path(name)
        try:
            with os.fdopen(temporary_fd, "wb") as stream:
                stream.write(payload["material-manifest.json"])
                stream.flush()
                os.fsync(stream.fileno())
            # The parent seal, old proof and immutable inventory are checked
            # again immediately before the single-file atomic replacement.
            check_closed_inventory(root, payload, descriptor)
            if read(root / "material-manifest.json") != old_bytes or read(state / "material-manifest.json") != parent_bytes:
                fail("projection_refresh_cas_preimage_changed")
            for relative, data in payload.items():
                if relative != "material-manifest.json" and read(root / relative) != data:
                    fail("projection_refresh_runtime_bytes_changed")
            os.replace(temporary, root / "material-manifest.json")
        finally:
            temporary.unlink(missing_ok=True)
    finally:
        os.close(fd)


def provision(repo, container, state, material, pg_port, engine="docker", source_context=None, refresh_operator_proof=False):
    """Create an immutable projection; never alter original operator material."""
    if engine not in ("docker", "podman"):
        fail("projection_engine_not_supported")
    root, payload, descriptor = preflight(repo, container, state, material, pg_port, source_context)
    if root.exists():
        canonical(root)
        check_closed_inventory(root, payload, descriptor)
        for relative, data in payload.items():
            if relative == "material-manifest.json" and refresh_operator_proof and read(root / relative) != data:
                continue
            if read(root / relative) != data:
                fail("projection_existing_preimage_changed")
        if read(root / "material-manifest.json") != payload["material-manifest.json"]:
            refresh_proof_reference(canonical(state), root, payload, descriptor)
        return descriptor
    if refresh_operator_proof:
        fail("projection_refresh_requires_existing_sealed_projection")
    # No writes until every source, reference, env, contract and digest passed.
    temporary = Path(tempfile.mkdtemp(prefix=".runtime-interno-", dir=state))
    try:
        for relative, data in payload.items():
            write(temporary / relative, data)
        for p in descriptor["rw"]:
            private_dirs(temporary / Path(p["source"]).relative_to("runtime-interno"))
        for directory in ("rw/data/bolsa", "rw/data/importaciones"):
            private_dirs(temporary / directory)
        (temporary / "material/comunicaciones").mkdir(mode=0o700, exist_ok=True)
        check_closed_inventory(temporary, payload, descriptor)
        temporary.rename(root)
    except BaseException:
        import shutil
        shutil.rmtree(temporary)
        raise
    return descriptor


if __name__ == "__main__":
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--repo", type=Path, required=True)
    parser.add_argument("--state", type=Path, required=True)
    parser.add_argument("--container", required=True)
    parser.add_argument("--pg-port", type=int, required=True)
    parser.add_argument("--refresh-internal-proof", action="store_true")
    args = parser.parse_args()
    result = provision(args.repo, args.container, args.state, args.state / "material", args.pg_port,
                       refresh_operator_proof=args.refresh_internal_proof)
    print(json.dumps({"material": result["material"], "portal": result["portal"], "status": "prepared"}))
