#!/usr/bin/env python3
"""Build private synthetic external mTLS material offline after source accreditation.

No SQL, provisioning, alias export, READY update or runtime activation occurs.
Requires the reviewed sibling clon_fuente_acreditada and local cryptography.
Custody and runtime are sibling trees; only runtime-externo may be mounted in
the external process. An interrupted build remains pending and cannot regenerate
keys. A completed build can only be verified byte for byte.
"""
from __future__ import annotations

import argparse
from contextlib import contextmanager
from datetime import datetime, timedelta, timezone
import fcntl
import hashlib
import importlib.util
import ipaddress
import json
import os
from pathlib import Path
import re
import secrets
import stat
import sys

from cryptography import x509
from cryptography.hazmat.primitives import hashes, serialization
from cryptography.hazmat.primitives.asymmetric import ec
from cryptography.hazmat.primitives.serialization import pkcs12
from cryptography.x509.oid import ExtendedKeyUsageOID, NameOID

MAX_FILE = 256 * 1024
SOURCE_SHA = "f7fcd35db52d1776e7f0ea34dabbe5c86466c24b352b0007690470f32a4df226"
SUBJECT = "desarrollo:clon-recorridos:candidato"
PENDING = "material-externo.pending.json"
RECEIPT = "material-externo-offline.json"
LOCK = ".material-externo.lock"
RUNTIME = "runtime-externo/"
CUSTODY = "custodia/"
RUNTIME_FILES = frozenset({"ca/ca.crt", "tls/servidor.crt", "tls/servidor.key",
    "mtls/candidato.crt", "identidad/candidato.json", "identidad/bolsa-candidato.json",
    "portal-proceso.json", "manifiesto.json", "idempotencia/configuracion.json",
    *{f"idempotencia/g{g}-{kind}.bin" for g in (1, 2) for kind in ("localizador", "huella-solicitud")}})
CUSTODY_FILES = frozenset({"ca/ca.crt", "ca/ca.key", "mtls/candidato.crt", "mtls/candidato.key",
    "mtls/candidato.p12", "mtls/candidato.p12.password"})
FILES = frozenset({RUNTIME + name for name in RUNTIME_FILES} | {CUSTODY + name for name in CUSTODY_FILES})
META_KEYS = {"version", "source_sha256", "autoridad_maestra", "responsable", "acreditado_en",
             "refs", "sujeto", "vigente_desde", "vigente_hasta"}


class OfflineMaterialError(RuntimeError):
    pass


def fail(code: str) -> None:
    raise OfflineMaterialError(code)


def digest(data: bytes) -> str:
    return hashlib.sha256(data).hexdigest()


def encoded(value: object) -> bytes:
    return (json.dumps(value, ensure_ascii=False, sort_keys=True, indent=2) + "\n").encode()


def unique_object(pairs: list) -> dict:
    result = {}
    for key, value in pairs:
        if key in result:
            fail("duplicate_json_key")
        result[key] = value
    return result


def decoded(data: bytes) -> dict:
    result = json.loads(data, object_pairs_hook=unique_object,
                        parse_constant=lambda _: fail("json_number_invalid"))
    if not isinstance(result, dict):
        fail("json_object_required")
    return result


def instant(value: str) -> datetime:
    if not isinstance(value, str) or not re.fullmatch(r"\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}(?:\.\d{6})?Z", value):
        fail("instant_invalid")
    return datetime.fromisoformat(value.replace("Z", "+00:00"))


def file_info(fd: int, *, empty: bool = False) -> os.stat_result:
    info = os.fstat(fd)
    if (not stat.S_ISREG(info.st_mode) or stat.S_IMODE(info.st_mode) != 0o600 or
            info.st_uid != os.getuid() or info.st_nlink != 1 or
            not (0 if empty else 1) <= info.st_size <= MAX_FILE):
        fail("private_file_invalid")
    return info


def dir_info(fd: int, *, leaf: bool) -> tuple:
    info = os.fstat(fd)
    if (not stat.S_ISDIR(info.st_mode) or info.st_uid not in {0, os.getuid()} or
            info.st_mode & 0o022 or (leaf and
            (info.st_uid != os.getuid() or stat.S_IMODE(info.st_mode) != 0o700))):
        fail("private_directory_invalid")
    try:
        os.stat(".git", dir_fd=fd, follow_symlinks=False)
    except FileNotFoundError:
        pass
    else:
        fail("private_directory_inside_git")
    return info.st_dev, info.st_ino


def open_root(path: Path, *, create: bool = False) -> tuple[int, tuple]:
    if not path.is_absolute() or len(path.parts) < 2 or ".." in path.parts:
        fail("private_path_invalid")
    flags = os.O_RDONLY | os.O_DIRECTORY | os.O_NOFOLLOW
    fd = os.open("/", flags)
    identities = []
    try:
        identities.append(dir_info(fd, leaf=False))
        for index, name in enumerate(path.parts[1:], 1):
            leaf = index == len(path.parts) - 1
            if leaf and create:
                try:
                    os.mkdir(name, mode=0o700, dir_fd=fd)
                    os.fsync(fd)
                except FileExistsError:
                    pass
            child = os.open(name, flags, dir_fd=fd)
            os.close(fd)
            fd = child
            identities.append(dir_info(fd, leaf=leaf))
        return fd, tuple(identities)
    except BaseException:
        os.close(fd)
        raise


def revalidate(path: Path, identities: tuple) -> None:
    fd, current = open_root(path)
    try:
        if identities != current:
            fail("private_directory_changed")
    finally:
        os.close(fd)


@contextmanager
def parent_fd(root_fd: int, relative: str, *, create: bool = False):
    parts = relative.split("/")
    if any(part in ("", ".", "..") for part in parts):
        fail("relative_path_invalid")
    fd = os.dup(root_fd)
    try:
        for name in parts[:-1]:
            if create:
                try:
                    os.mkdir(name, mode=0o700, dir_fd=fd)
                    os.fsync(fd)
                except FileExistsError:
                    pass
            child = os.open(name, os.O_RDONLY | os.O_DIRECTORY | os.O_NOFOLLOW, dir_fd=fd)
            os.close(fd)
            fd = child
            dir_info(fd, leaf=True)
        yield fd, parts[-1]
    finally:
        os.close(fd)


def read_at(root_fd: int, relative: str) -> bytes:
    with parent_fd(root_fd, relative) as (parent, leaf):
        fd = os.open(leaf, os.O_RDONLY | os.O_NOFOLLOW | os.O_NONBLOCK, dir_fd=parent)
        try:
            before = file_info(fd)
            data = bytearray()
            while len(data) <= MAX_FILE:
                chunk = os.read(fd, min(65536, MAX_FILE + 1 - len(data)))
                if not chunk:
                    break
                data.extend(chunk)
            after = file_info(fd)
            fields = ("st_dev", "st_ino", "st_mode", "st_uid", "st_nlink", "st_size", "st_mtime_ns", "st_ctime_ns")
            if len(data) != before.st_size or any(getattr(before, f) != getattr(after, f) for f in fields):
                fail("private_file_changed")
            return bytes(data)
        finally:
            os.close(fd)


def read_private(path: Path) -> bytes:
    fd, identities = open_root(path.parent)
    try:
        data = read_at(fd, path.name)
        revalidate(path.parent, identities)
        return data
    finally:
        os.close(fd)


def write_at(root_fd: int, relative: str, data: bytes) -> None:
    if not 0 < len(data) <= MAX_FILE:
        fail("output_size_invalid")
    with parent_fd(root_fd, relative, create=True) as (parent, leaf):
        fd = os.open(leaf, os.O_WRONLY | os.O_CREAT | os.O_EXCL | os.O_NOFOLLOW, 0o600, dir_fd=parent)
        try:
            os.fchmod(fd, 0o600)
            view = memoryview(data)
            while view:
                written = os.write(fd, view)
                if written < 1:
                    fail("output_write_failed")
                view = view[written:]
            os.fsync(fd)
            file_info(fd)
        finally:
            os.close(fd)
        os.fsync(parent)


def inventory(root_fd: int, prefix: str = "") -> set[str]:
    result = set()
    allowed_dirs = {str(parent) for name in FILES for parent in Path(name).parents if str(parent) != "."}
    for name in os.listdir(root_fd):
        info = os.stat(name, dir_fd=root_fd, follow_symlinks=False)
        relative = prefix + name
        if stat.S_ISDIR(info.st_mode):
            if relative not in allowed_dirs:
                fail("material_directory_unexpected")
            child = os.open(name, os.O_RDONLY | os.O_DIRECTORY | os.O_NOFOLLOW, dir_fd=root_fd)
            try:
                dir_info(child, leaf=True)
                result.update(inventory(child, relative + "/"))
            finally:
                os.close(child)
        elif stat.S_ISREG(info.st_mode):
            result.add(relative)
        else:
            fail("material_entry_invalid")
        if len(result) > len(FILES) + 3:
            fail("material_inventory_excessive")
    return result


def accreditation(source_bytes: bytes, ack: dict, now: datetime) -> dict:
    # This is trusted repository code, never a plugin path supplied by input.
    path = Path(__file__).with_name("clon_fuente_acreditada.py")
    if not path.is_file() or path.is_symlink():
        fail("source_accreditation_authority_absent")
    spec = importlib.util.spec_from_file_location("vec_source_accreditation_offline", path)
    module = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(module)
    validator = getattr(module, "validate_accreditation", None)
    if not callable(validator):
        fail("source_accreditation_contract_absent")
    return validator(source_bytes, ack, now)


def binding(source_bytes: bytes, ack_bytes: bytes, now: datetime) -> dict:
    if digest(source_bytes) != SOURCE_SHA:
        fail("source_sha_mismatch")
    ack = decoded(ack_bytes)
    meta = accreditation(source_bytes, ack, now)
    if not isinstance(meta, dict) or set(meta) != META_KEYS or type(meta["version"]) is not int or meta["version"] != 1 or meta["source_sha256"] != SOURCE_SHA:
        fail("source_accreditation_metadata_invalid")
    master = decoded(source_bytes)
    if meta["sujeto"] != SUBJECT or master.get("datos_sinteticos") is not True:
        fail("synthetic_subject_invalid")
    refs = meta["refs"]
    prefixes = {"cuenta": "cta_", "persona": "per_", "perfil": "prf_", "contexto": "vca_",
                "vinculo": "vin_", "candidato": "can_", "procedencia": "prc_"}
    if not isinstance(refs, dict) or set(refs) != set(prefixes) or any(
            not isinstance(refs[k], str) or not re.fullmatch(p + r"[0-9a-f]{32}", refs[k])
            for k, p in prefixes.items()):
        fail("source_refs_invalid")
    if master.get("candidatoBolsa") != {k + "_ref": refs[k] for k in ("cuenta", "persona", "perfil", "candidato")}:
        fail("source_refs_mismatch")
    persons = master.get("personas")
    if not isinstance(persons, list) or len(persons) != 1 or persons[0].get("persona_ref") != refs["persona"]:
        fail("source_person_invalid")
    display = persons[0].get("nombre_visible")
    if not isinstance(display, str) or not display.strip() or len(display) > 160 or any(ord(c) < 32 for c in display):
        fail("source_display_name_invalid")
    start, end = instant(meta["vigente_desde"]), instant(meta["vigente_hasta"])
    if not start <= now < end or end - start > timedelta(days=31):
        fail("source_validity_invalid")
    return {"version": 1, "kind": "material_externo_pending_v1", "fuente_sha256": SOURCE_SHA,
            "acuse_sha256": digest(ack_bytes), **{k: meta[k] for k in META_KEYS - {"version", "source_sha256"}},
            "nombre_visible": display}


def subject_name(common_name: str) -> x509.Name:
    return x509.Name([x509.NameAttribute(NameOID.COMMON_NAME, common_name),
                      x509.NameAttribute(NameOID.ORGANIZATION_NAME, "VEC Desarrollo"),
                      x509.NameAttribute(NameOID.ORGANIZATIONAL_UNIT_NAME, "NO AUTORITATIVO")])


def idempotency_configuration() -> dict:
    return {"version": 1, "esquema": "vec.bolsa.convocatoria.idempotencia-hmac.desarrollo.v1",
            "autoridad": "no_autoritativo", "version_esquema_hmac": 2,
            "generaciones": [{"generacion": g,
                "referencia_localizador": f"clave:hmac:convocatorias:localizador:desarrollo:v{g}",
                "referencia_huella_solicitud": f"clave:hmac:convocatorias:huella:desarrollo:v{g}"} for g in (2, 1)]}


def identity_documents(bound: dict, fingerprint: str) -> tuple[dict, dict]:
    identity = {"version": 1, "autoridad": "no_autoritativo", "certificate_sha256": fingerprint,
                "subject": bound["sujeto"], "display_name": bound["nombre_visible"], "roles": ["candidato_bolsa"]}
    bolsa = {"version": 1, "autoridad": "no_autoritativo", "certificado": "mtls/candidato.crt",
             "identidad": "identidad/candidato.json", "sujeto": bound["sujeto"],
             **{key + "_ref": bound["refs"][key] for key in ("cuenta", "persona", "perfil", "candidato")}}
    return identity, bolsa


def runtime_manifest(ca: str, server: str, client: str) -> dict:
    # Closed Go archivoManifiestoDesarrollo schema; source receipt stays outside runtime.
    return {"version": 4, "perfil": "desarrollo", "autoridad": "no_autoritativo",
            "migrable_a_produccion": False, "huella_ca_sha256": ca,
            "huella_servidor_sha256": server, "huella_cliente_sha256": client,
            "huella_intervencion_sha256": "", "huella_publica_atestacion_kms_sha256": "",
            "huella_publica_revalidacion_kms_sha256": "",
            "proveedores": {"identidad": "identidad-mtls-local-v1",
                "idempotencia_hmac": "idempotencia-hmac-fichero-local-v1", "tls": "tls-ca-local-v1"}}


def generate(bound: dict, now: datetime) -> dict[str, bytes]:
    keys = {kind: ec.generate_private_key(ec.SECP256R1()) for kind in ("ca", "client", "server")}
    start = max(instant(bound["vigente_desde"]), now - timedelta(minutes=5)).replace(microsecond=0)
    end = instant(bound["vigente_hasta"]).replace(microsecond=0)
    ca_name = subject_name("VEC CA externa clon local")

    def issue(kind: str) -> x509.Certificate:
        ca = kind == "ca"
        builder = (x509.CertificateBuilder().subject_name(ca_name if ca else subject_name(
            "candidato-clon-local" if kind == "client" else "localhost")).issuer_name(ca_name)
            .public_key(keys[kind].public_key()).serial_number(x509.random_serial_number())
            .not_valid_before(start).not_valid_after(end)
            .add_extension(x509.BasicConstraints(ca=ca, path_length=0 if ca else None), critical=True)
            .add_extension(x509.KeyUsage(digital_signature=True, content_commitment=False,
                key_encipherment=False, data_encipherment=False, key_agreement=False,
                key_cert_sign=ca, crl_sign=ca, encipher_only=None, decipher_only=None), critical=True)
            .add_extension(x509.SubjectKeyIdentifier.from_public_key(keys[kind].public_key()), critical=False)
            .add_extension(x509.AuthorityKeyIdentifier.from_issuer_public_key(keys["ca"].public_key()), critical=False))
        if not ca:
            san = ([x509.UniformResourceIdentifier("urn:vec:" + bound["sujeto"])] if kind == "client" else
                   [x509.DNSName("localhost"), x509.IPAddress(ipaddress.ip_address("127.0.0.1")),
                    x509.IPAddress(ipaddress.ip_address("::1"))])
            builder = builder.add_extension(x509.SubjectAlternativeName(san), critical=False)
            builder = builder.add_extension(x509.ExtendedKeyUsage([
                ExtendedKeyUsageOID.CLIENT_AUTH if kind == "client" else ExtendedKeyUsageOID.SERVER_AUTH]), critical=False)
        return builder.sign(keys["ca"], hashes.SHA256())

    certs = {kind: issue(kind) for kind in keys}
    fp = {kind: cert.fingerprint(hashes.SHA256()).hex() for kind, cert in certs.items()}
    pem_cert = lambda kind: certs[kind].public_bytes(serialization.Encoding.PEM)
    pem_key = lambda kind: keys[kind].private_bytes(serialization.Encoding.PEM,
        serialization.PrivateFormat.PKCS8, serialization.NoEncryption())
    password = secrets.token_hex(32).encode()
    identity, bolsa = identity_documents(bound, fp["client"])
    outputs = {
        CUSTODY + "ca/ca.crt": pem_cert("ca"), CUSTODY + "ca/ca.key": pem_key("ca"),
        CUSTODY + "mtls/candidato.crt": pem_cert("client"), CUSTODY + "mtls/candidato.key": pem_key("client"),
        CUSTODY + "mtls/candidato.p12.password": password + b"\n",
        CUSTODY + "mtls/candidato.p12": pkcs12.serialize_key_and_certificates(b"candidato", keys["client"],
            certs["client"], [certs["ca"]], serialization.BestAvailableEncryption(password)),
        RUNTIME + "ca/ca.crt": pem_cert("ca"), RUNTIME + "mtls/candidato.crt": pem_cert("client"),
        RUNTIME + "tls/servidor.crt": pem_cert("server"), RUNTIME + "tls/servidor.key": pem_key("server"),
        RUNTIME + "identidad/candidato.json": encoded(identity), RUNTIME + "identidad/bolsa-candidato.json": encoded(bolsa),
        RUNTIME + "portal-proceso.json": encoded({"version": 1, "portal": "externo"}),
        RUNTIME + "manifiesto.json": encoded(runtime_manifest(fp["ca"], fp["server"], fp["client"])),
        RUNTIME + "idempotencia/configuracion.json": encoded(idempotency_configuration()),
    }
    for g in (1, 2):
        for kind in ("localizador", "huella-solicitud"):
            outputs[RUNTIME + f"idempotencia/g{g}-{kind}.bin"] = secrets.token_bytes(32)
    if set(outputs) != FILES:
        fail("generated_inventory_invalid")
    return outputs


def verify_material(outputs: dict, bound: dict, now: datetime) -> dict:
    if set(outputs) != FILES:
        fail("material_inventory_invalid")
    certs = {}
    for kind, relative in (("ca", "ca/ca.crt"), ("server", "tls/servidor.crt"), ("client", "mtls/candidato.crt")):
        data = outputs[RUNTIME + relative]
        cert = x509.load_pem_x509_certificate(data)
        if cert.public_bytes(serialization.Encoding.PEM) != data:
            fail("certificate_pem_invalid")
        if not isinstance(cert.public_key(), ec.EllipticCurvePublicKey) or not isinstance(cert.public_key().curve, ec.SECP256R1):
            fail("certificate_algorithm_invalid")
        if cert.signature_hash_algorithm.name != "sha256" or not cert.not_valid_before_utc <= now < cert.not_valid_after_utc:
            fail("certificate_validity_invalid")
        if cert.not_valid_before_utc < instant(bound["vigente_desde"]) or cert.not_valid_after_utc > instant(bound["vigente_hasta"]):
            fail("certificate_outside_source_validity")
        certs[kind] = cert
    ca = certs["ca"]
    for kind, cert in certs.items():
        cert.verify_directly_issued_by(ca)
        basic = cert.extensions.get_extension_for_class(x509.BasicConstraints)
        usage = cert.extensions.get_extension_for_class(x509.KeyUsage)
        if not basic.critical or basic.value.ca != (kind == "ca") or not usage.critical or not usage.value.digital_signature:
            fail("certificate_constraints_invalid")
        if usage.value.key_cert_sign != (kind == "ca") or usage.value.crl_sign != (kind == "ca"):
            fail("certificate_usage_invalid")
        if basic.value.path_length != (0 if kind == "ca" else None):
            fail("certificate_path_length_invalid")
        extension_oids = {x509.ExtensionOID.BASIC_CONSTRAINTS, x509.ExtensionOID.KEY_USAGE,
                          x509.ExtensionOID.SUBJECT_KEY_IDENTIFIER, x509.ExtensionOID.AUTHORITY_KEY_IDENTIFIER}
        if kind != "ca":
            extension_oids |= {x509.ExtensionOID.SUBJECT_ALTERNATIVE_NAME, x509.ExtensionOID.EXTENDED_KEY_USAGE}
        if {extension.oid for extension in cert.extensions} != extension_oids:
            fail("certificate_extensions_invalid")
        expected_name = subject_name({"ca": "VEC CA externa clon local", "server": "localhost", "client": "candidato-clon-local"}[kind])
        if cert.subject != expected_name:
            fail("certificate_subject_invalid")
        if kind != "ca":
            expected_san = (x509.SubjectAlternativeName([x509.UniformResourceIdentifier("urn:vec:" + bound["sujeto"])])
                if kind == "client" else x509.SubjectAlternativeName([x509.DNSName("localhost"),
                    x509.IPAddress(ipaddress.ip_address("127.0.0.1")), x509.IPAddress(ipaddress.ip_address("::1"))]))
            if cert.extensions.get_extension_for_class(x509.SubjectAlternativeName).value != expected_san:
                fail("certificate_san_invalid")
            purpose = ExtendedKeyUsageOID.CLIENT_AUTH if kind == "client" else ExtendedKeyUsageOID.SERVER_AUTH
            if list(cert.extensions.get_extension_for_class(x509.ExtendedKeyUsage).value) != [purpose]:
                fail("certificate_purpose_invalid")
    def public(key):
        return key.public_key().public_bytes(serialization.Encoding.DER, serialization.PublicFormat.SubjectPublicKeyInfo)
    for kind, path in (("ca", CUSTODY + "ca/ca.key"), ("client", CUSTODY + "mtls/candidato.key"), ("server", RUNTIME + "tls/servidor.key")):
        key = serialization.load_pem_private_key(outputs[path], password=None)
        if key.private_bytes(serialization.Encoding.PEM, serialization.PrivateFormat.PKCS8,
                             serialization.NoEncryption()) != outputs[path]:
            fail("private_key_pem_invalid")
        if public(key) != certs[kind].public_key().public_bytes(serialization.Encoding.DER, serialization.PublicFormat.SubjectPublicKeyInfo):
            fail("certificate_private_key_mismatch")
    if outputs[CUSTODY + "ca/ca.crt"] != outputs[RUNTIME + "ca/ca.crt"] or outputs[CUSTODY + "mtls/candidato.crt"] != outputs[RUNTIME + "mtls/candidato.crt"]:
        fail("custody_certificate_mismatch")
    password = outputs[CUSTODY + "mtls/candidato.p12.password"]
    if not re.fullmatch(rb"[0-9a-f]{64}\n", password):
        fail("pkcs12_password_invalid")
    p12_key, p12_cert, chain = pkcs12.load_key_and_certificates(outputs[CUSTODY + "mtls/candidato.p12"], password.rstrip(b"\n"))
    if p12_key is None or p12_cert != certs["client"] or chain != [ca] or public(p12_key) != public(serialization.load_pem_private_key(outputs[CUSTODY + "mtls/candidato.key"], None)):
        fail("pkcs12_material_mismatch")
    fp = {kind: cert.fingerprint(hashes.SHA256()).hex() for kind, cert in certs.items()}
    identity, bolsa = identity_documents(bound, fp["client"])
    expected = {"identidad/candidato.json": identity, "identidad/bolsa-candidato.json": bolsa,
                "manifiesto.json": runtime_manifest(fp["ca"], fp["server"], fp["client"]),
                "portal-proceso.json": {"version": 1, "portal": "externo"},
                "idempotencia/configuracion.json": idempotency_configuration()}
    for name, value in expected.items():
        if outputs[RUNTIME + name] != encoded(value):
            fail("runtime_document_mismatch")
    hmac_keys = [outputs[RUNTIME + f"idempotencia/g{g}-{kind}.bin"] for g in (1, 2) for kind in ("localizador", "huella-solicitud")]
    if any(len(k) != 32 or k == bytes(32) for k in hmac_keys) or len(set(hmac_keys)) != 4:
        fail("idempotency_keys_invalid")
    return fp


def receipt_for(bound: dict, outputs: dict, fp: dict) -> dict:
    return {**{k: v for k, v in bound.items() if k not in {"kind", "nombre_visible"}}, "kind": "material_externo_offline_v1",
            "estado": "preparado_offline", "huella_ca_sha256": fp["ca"], "huella_servidor_sha256": fp["server"],
            "certificate_sha256": fp["client"],
            "archivos": {name: {"sha256": digest(data), "bytes": len(data)} for name, data in sorted(outputs.items())},
            "exportador_ejecutado": False, "provision_ejecutada": False, "runtime_activado": False}


def produce(source: Path, ack: Path, output: Path, *, now: datetime | None = None) -> dict:
    now = now or datetime.now(timezone.utc)
    if now.tzinfo != timezone.utc:
        fail("clock_not_utc")
    # Validate source and explicit receipt before creating even the output directory.
    bound = binding(read_private(source), read_private(ack), now)
    if output == source.parent or output == ack.parent or output in source.parents or output in ack.parents:
        fail("output_overlaps_source")
    fd, identities = open_root(output, create=True)
    lock_fd = None
    try:
        revalidate(output, identities)
        lock_fd = os.open(LOCK, os.O_RDWR | os.O_CREAT | os.O_NOFOLLOW | os.O_NONBLOCK, 0o600, dir_fd=fd)
        file_info(lock_fd, empty=True)
        fcntl.flock(lock_fd, fcntl.LOCK_EX | fcntl.LOCK_NB)
        entries = inventory(fd)
        if entries != {LOCK}:
            if entries != FILES | {LOCK, PENDING, RECEIPT}:
                fail("pending_or_unexpected_material")
            if read_at(fd, PENDING) != encoded(bound):
                fail("material_binding_changed")
            previous = decoded(read_at(fd, RECEIPT))
            outputs = {name: read_at(fd, name) for name in FILES}
            if previous.get("archivos") != {name: {"sha256": digest(data), "bytes": len(data)} for name, data in outputs.items()}:
                fail("material_preimage_changed")
            fp = verify_material(outputs, bound, now)
            if read_at(fd, RECEIPT) != encoded(receipt_for(bound, outputs, fp)):
                fail("material_receipt_changed")
            revalidate(output, identities)
            return previous
        revalidate(output, identities)
        write_at(fd, PENDING, encoded(bound))
        # The pending reservation is durable before the first random key is generated.
        os.fsync(fd)
        revalidate(output, identities)
        outputs = generate(bound, now)
        fp = verify_material(outputs, bound, now)
        for name, data in sorted(outputs.items()):
            revalidate(output, identities)
            write_at(fd, name, data)
        if inventory(fd) != FILES | {LOCK, PENDING} or any(read_at(fd, name) != data for name, data in outputs.items()):
            fail("material_postimage_changed")
        receipt = receipt_for(bound, outputs, fp)
        revalidate(output, identities)
        write_at(fd, RECEIPT, encoded(receipt))
        revalidate(output, identities)
        os.fsync(fd)
        return receipt
    finally:
        if lock_fd is not None:
            os.close(lock_fd)
        os.close(fd)


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__.splitlines()[0])
    parser.add_argument("--fuente", type=Path, required=True)
    parser.add_argument("--acuse", type=Path, required=True)
    parser.add_argument("--directorio", type=Path, required=True)
    args = parser.parse_args()
    try:
        result = produce(args.fuente, args.acuse, args.directorio)
    except Exception:
        # Library/OpenSSL errors may carry sensitive input; never propagate them.
        print('{"error":"material_externo_offline_rechazado"}', file=sys.stderr)
        return 2
    print(json.dumps({"estado": result["estado"], "exportador_ejecutado": False,
                      "provision_ejecutada": False, "runtime_activado": False}))
    return 0


if __name__ == "__main__":
    sys.exit(main())
