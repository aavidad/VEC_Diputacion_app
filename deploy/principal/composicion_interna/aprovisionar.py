#!/usr/bin/env python3
"""Aprovisiona material privado y LOGIN nominales de vec-interno en desarrollo.

El certificado de persona se emite a partir de un CSR creado en un token
PKCS#11: la clave privada personal nunca entra en este proceso ni en Git.
"""

from __future__ import annotations

import argparse
import ctypes as c
from datetime import datetime, timezone
import fcntl
import hashlib
import ipaddress
import json
import os
from pathlib import Path
import re
import secrets
import shutil
import subprocess
import sys
import tempfile
from urllib.parse import urlsplit
from urllib.parse import quote, urlencode


ROLES = {
    "alta_personal": "vec_personal_ejecutor",
    "registro_ct": "vec_contratacion_temporal_ejecutor",
    "inicial_ct": "vec_contratacion_temporal_lector_raices_historicas",
    "localizador_ct": "vec_contratacion_temporal_localizador_incorporacion",
    "localizador_personal": "vec_personal_localizador_solicitud_alta",
    "lectura_personal": "vec_personal_ejecutor",
    "historia_registro_ct": "vec_contratacion_temporal_lector_historia_incorporacion",
    "historia_autenticacion": "vec_identidad_sesiones_v1_lector_historico",
    "historia_contexto": "vec_contexto_actor_v1_lector_historico",
    "historia_evaluacion": "vec_autorizacion_evaluacion_historica_lector",
    "historia_concesion": "vec_autorizacion_registro",
}

FUNCTIONS = {
    "alta_personal": "vec_personal.registrar_alta_ejercicio_v1(jsonb,bytea,bytea,bytea,bytea,bigint,bigint,bytea,bytea,bytea,bytea)",
    "registro_ct": "vec_contratacion_temporal.registrar_incorporacion_ejercicio_v2(jsonb,text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea,bytea,bytea,bytea,bytea,bigint,bigint,bytea,bytea,bytea,bytea,jsonb)",
    "inicial_ct": "vec_contratacion_temporal.leer_raices_incorporacion_ejercicio_v2(text,text,text)",
    "localizador_ct": "vec_contratacion_temporal.localizar_incorporacion_original_v2(text,text,text)",
    "localizador_personal": "vec_personal.localizar_solicitud_alta_ejercicio_v1(text,text,text)",
    "lectura_personal": "vec_personal.acreditar_alta_ejercicio_v2(text,text,text,bigint,text,text,text,text,text,text,bytea,bytea,bytea,bytea,bigint,bigint,bytea,bytea,bytea,bytea)",
    "historia_registro_ct": "vec_contratacion_temporal.leer_historia_incorporacion_original_v2(text,text,text)",
    "historia_autenticacion": "vec_identidad_sesiones_v1.leer_autenticacion_original_v1(text,text,text)",
    "historia_contexto": "vec_contexto_actor_v1.leer_contexto_original_v2(text,text,text)",
    "historia_evaluacion": "vec_autorizacion.leer_evaluacion_original_contexto_actor_v3(text,text,text)",
    "historia_concesion": "vec_autorizacion.leer_concesion_historica_contexto_actor_v3(bytea,bytea,numeric,numeric)",
}

IDENTITY_ROLES = {
    "registro": "vec_identidad_sesiones_v1_registrador",
    "revalidacion": "vec_identidad_sesiones_v1_revalidador",
    "contexto": "vec_contexto_actor_v1_runtime",
    "auditoria": "vec_contratacion_temporal_registrador_frontera",
}
IDENTITY_FUNCTIONS = {
    "registro": [
        "vec_identidad_sesiones_v1.registrar_sesion_v1(text,text,text,text,bigint,bytea,bytea,bytea,bytea,bytea,boolean,text,text,text,text,timestamptz,timestamptz,timestamptz,text,text)",
        "vec_identidad_sesiones_v1.reconciliar_registro_sesion_v1(text,text,text,text,bigint,bytea,bytea,bytea,bytea,bytea,boolean,text,text,text,text,timestamptz,timestamptz,timestamptz,text,text)",
    ],
    "revalidacion": [
        "vec_identidad_sesiones_v1.revalidar_sesion_y_cuentas_v1(text,text,text,text,text,text,boolean,text,text,text,text,text,timestamptz,timestamptz,text,text,text,text,timestamptz,timestamptz)",
        "vec_identidad_sesiones_v1.revalidar_autenticacion_actor_v1(text,text)",
        "vec_identidad_sesiones_v1.coincide_politica_certificado_desarrollo_v1(text,text,timestamptz)",
    ],
    "contexto": [
        "vec_contexto_actor_v1.acreditar_runtime_contexto_actor_v1()",
        "vec_contexto_actor_v1.resolver_y_registrar_contexto_actor_v2(text,text,text,text,text,text,timestamptz)",
        "vec_contexto_actor_v1.reconciliar_contexto_actor_v2(text,text,text,text,text,text,timestamptz)",
    ],
    "auditoria": [
        "vec_contratacion_temporal.registrar_auditoria_frontera_ruta_exacta_v1(text,text,text,text,text)",
    ],
}

V3_ROLES = {
    "fuente_autorizacion": "vec_autorizacion_fuente",
    "registro_autorizacion": "vec_autorizacion_registro",
    "motivos_autorizacion": "vec_autorizacion_motivos_evaluador",
    "motivos_rrhh": "vec_autorizacion_motivos_rrhh_resolutor",
    "consulta_rrhh": "vec_contratacion_temporal_consultor_rrhh",
    "gobierno_v3": "vec_autorizacion_atestada_v3_preflight_interno",
}
V3_FUNCTIONS = {
    "fuente_autorizacion": "vec_autorizacion.obtener_instantanea(text,text)",
    "registro_autorizacion": "vec_autorizacion.registrar_decision_contexto_actor_v3(bytea,bytea,numeric,numeric)",
    "motivos_autorizacion": "vec_autorizacion.resolver_motivo_autorizacion_v2_historico(text,integer,text,text,timestamptz)",
    "motivos_rrhh": ["vec_autorizacion.resolver_motivo_cuadro_rrhh_v1(timestamptz)",
                     "vec_autorizacion.resolver_motivo_detalle_rrhh_v1(timestamptz)"],
    "consulta_rrhh": [
        "vec_contratacion_temporal.consultar_cuadro_rrhh_atestado_v1(vec_contratacion_temporal.alcance_consulta_rrhh_v1,vec_contratacion_temporal.consulta_cuadro_rrhh_v1,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)",
        "vec_contratacion_temporal.consultar_detalle_rrhh_atestado_v1(vec_contratacion_temporal.alcance_consulta_rrhh_v1,vec_contratacion_temporal.consulta_detalle_rrhh_v1,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)",
        "vec_contratacion_temporal.consultar_resumen_seguimiento_rrhh_atestado_v1(vec_contratacion_temporal.alcance_consulta_rrhh_v1,vec_contratacion_temporal.consulta_detalle_rrhh_v1,text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)",
    ],
    "gobierno_v3": [
        "vec_autorizacion_atestada_v3.comprobar_material_emision_interna_v1(jsonb)",
        "vec_autorizacion_atestada_v3.leer_configuracion_interna_v1(jsonb)",
    ],
}


def fail(message: str) -> None:
    raise RuntimeError(message)


def run(args: list[str], *, input_text: str | None = None,
        input_bytes: bytes | None = None, env: dict[str, str] | None = None) -> bytes:
    if input_text is not None and input_bytes is not None:
        fail("entrada ambigua")
    payload = input_text.encode() if input_text is not None else input_bytes
    result = subprocess.run(args, input=payload, env=env,
                            stdout=subprocess.PIPE, stderr=subprocess.PIPE, check=False)
    if result.returncode:
        # La salida de OpenSSL/psql puede contener material; no se propaga.
        fail(f"falló {Path(args[0]).name} (código {result.returncode})")
    return result.stdout


def private_dir(path: str) -> Path:
    target = Path(path).expanduser().absolute()
    if target.is_symlink():
        fail("directorio privado enlazado")
    if ".worktrees" in target.parts:
        fail("el material privado no puede estar en un worktree")
    probe = target
    while not probe.exists():
        if probe == probe.parent:
            fail("destino privado inválido")
        probe = probe.parent
    git_preflight = subprocess.run(["git", "-C", str(probe), "rev-parse", "--show-toplevel"],
                                   stdout=subprocess.PIPE, stderr=subprocess.DEVNULL, check=False)
    if git_preflight.returncode == 0 and git_preflight.stdout.strip():
        fail("el material privado debe estar fuera de cualquier repositorio Git")
    if not target.exists():
        target.mkdir(mode=0o700, parents=True)
    target = target.resolve(strict=True)
    if ".worktrees" in target.parts:
        fail("el material privado no puede estar en un worktree")
    git = subprocess.run(["git", "-C", str(target), "rev-parse", "--show-toplevel"],
                         stdout=subprocess.PIPE, stderr=subprocess.DEVNULL, check=False)
    if git.returncode == 0 and git.stdout.strip():
        fail("el material privado debe estar fuera de cualquier repositorio Git")
    inside_git_dir = subprocess.run(["git", "-C", str(target), "rev-parse", "--is-inside-git-dir"],
                                    stdout=subprocess.PIPE, stderr=subprocess.DEVNULL, check=False)
    if inside_git_dir.returncode == 0 and inside_git_dir.stdout.strip() == b"true":
        fail("el material privado no puede estar en metadatos Git")
    if not target.is_dir() or target.stat().st_uid != os.getuid() or target.stat().st_mode & 0o777 != 0o700:
        fail("directorio privado inseguro; exige propietario actual y modo 0700")
    return target


def subdir(root: Path, name: str) -> Path:
    path = root / name
    if path.is_symlink():
        fail("subdirectorio privado enlazado")
    if not path.exists():
        path.mkdir(mode=0o700)
    if path.stat().st_uid != os.getuid() or path.stat().st_mode & 0o777 != 0o700:
        fail("subdirectorio privado inseguro")
    return path


def read_private(path: Path) -> bytes:
    if path.is_symlink() or not path.is_file() or path.stat().st_uid != os.getuid() or path.stat().st_mode & 0o777 != 0o600:
        fail("fichero privado inexistente o inseguro")
    return path.read_bytes()


def atomic_private(path: Path, data: bytes) -> None:
    if path.is_symlink() or path.exists():
        fail("material ya existente: no se sobrescribe")
    fd, temp = tempfile.mkstemp(prefix=".vec-interno-", dir=path.parent)
    try:
        with os.fdopen(fd, "wb") as file:
            file.write(data)
            file.flush()
            os.fsync(file.fileno())
        os.chmod(temp, 0o600)
        os.link(temp, path)
    finally:
        os.unlink(temp)


def replace_private(path: Path, data: bytes) -> None:
    if path.is_symlink():
        fail("destino privado enlazado")
    if path.exists():
        read_private(path)
    fd, temp = tempfile.mkstemp(prefix=".vec-interno-reemplazo-", dir=path.parent)
    try:
        with os.fdopen(fd, "wb") as file:
            file.write(data)
            file.flush()
            os.fsync(file.fileno())
        os.chmod(temp, 0o600)
        os.replace(temp, path)
    finally:
        if os.path.exists(temp):
            os.unlink(temp)


def check_tool(name: str) -> None:
    if not shutil.which(name):
        fail(f"falta {name}")


def sql_quote(value: str) -> str:
    return "'" + value.replace("'", "''") + "'"


def name(value: str) -> str:
    if not re.fullmatch(r"[a-z][a-z0-9_]{0,62}", value):
        fail("identificador SQL inválido")
    return value


def digest_cert(path: Path) -> str:
    der = run(["openssl", "x509", "-in", str(path), "-outform", "DER"])
    return hashlib.sha256(der).hexdigest()


def token_environment(pin_file: Path) -> dict[str, str]:
    pin = read_private(pin_file).decode().strip()
    if not re.fullmatch(r"[ -~]{8,128}", pin):
        fail("PIN privado inválido")
    env = os.environ.copy()
    env["GNUTLS_PIN"] = pin
    return env


def create_csr(root: Path, subject_id: str, token_url: str, pkcs11_module: Path,
               p11tool: Path, certtool: Path, pin_file: Path) -> None:
    if not re.fullmatch(r"per_[a-z0-9_]{22,128}", subject_id):
        fail("sujeto opaco inválido")
    if not token_url.startswith("pkcs11:") or "type=private" not in token_url or "object=" not in token_url:
        fail("URL PKCS#11 debe señalar la clave privada nominal")
    if any(p.is_symlink() or not p.is_file() for p in (pkcs11_module, p11tool, certtool)):
        fail("proveedor y herramientas PKCS#11 ausentes")
    env = token_environment(pin_file)
    people = subdir(root, "personas")
    for existing in people.glob("*.token.json"):
        if existing.name != f"{subject_id}.token.json" and json.loads(read_private(existing)).get("token_url") == token_url:
            fail("la clave del token ya pertenece a otra persona sintética")
    dest = people / f"{subject_id}.csr"
    if dest.exists():
        read_private(dest)
        run(["openssl", "req", "-in", str(dest), "-verify", "-noout"])
        print("CSR nominal existente sin cambios")
        return
    token_base = token_url.split(";object=", 1)[0]
    label = token_url.split(";object=", 1)[1].split(";", 1)[0]
    if not re.fullmatch(r"[A-Za-z0-9_.-]{1,64}", label):
        fail("etiqueta de token inválida")
    with tempfile.TemporaryDirectory(prefix=".csr-", dir=people) as temporary:
        tmp = Path(temporary)
        probe = subprocess.run([str(p11tool), "--provider", str(pkcs11_module), "--login",
                                "--info", token_url], env=env, stdout=subprocess.PIPE,
                               stderr=subprocess.PIPE, check=False)
        if probe.returncode:
            run([str(p11tool), "--provider", str(pkcs11_module), "--login",
                 "--generate-privkey=ECDSA", "--curve=secp256r1", f"--label={label}",
                 f"--outfile={tmp / 'public.pem'}", token_base], env=env)
        info = run([str(p11tool), "--provider", str(pkcs11_module), "--login", "--info", token_url], env=env)
        if b"CKA_NEVER_EXTRACTABLE" not in info or b"CKA_SENSITIVE" not in info or b"CKA_PRIVATE" not in info:
            fail("clave del token exportable o sin PIN")
        (tmp / "csr.tmpl").write_text('cn = "vec-rrhh-sintetico"\norganization = "VEC Desarrollo"\nunit = "Sintetico"\n')
        run([str(certtool), "--provider", str(pkcs11_module), "--generate-request",
             "--load-privkey", token_url, "--template", str(tmp / "csr.tmpl"),
             "--outfile", str(tmp / "persona.csr")], env=env)
        run(["openssl", "req", "-in", str(tmp / "persona.csr"), "-verify", "-noout"])
        atomic_private(dest, (tmp / "persona.csr").read_bytes())
    print("CSR emitido desde clave no exportable del token, fuera de Git")


def init_ca(root: Path, server_name: str) -> None:
    check_tool("openssl")
    if not re.fullmatch(r"[a-zA-Z0-9.-]{1,253}", server_name) or server_name.startswith("-"):
        fail("nombre DNS de servidor inválido")
    ca, tls, identity = (subdir(root, x) for x in ("ca", "tls", "identidad"))
    targets = [ca / "ac.key", ca / "ac.crt", ca / "ac.password", tls / "servidor.key",
               tls / "servidor.crt", identity / "emisor.key", identity / "emisor.pub",
               ca / "ca.cnf", ca / "index.txt", ca / "serial", ca / "crlnumber",
               identity / "clientes.crl"]
    if any(p.exists() or p.is_symlink() for p in targets):
        if not all(p.is_file() and not p.is_symlink() for p in targets):
            fail("material CA/servidor parcial; intervención manual")
        for p in targets:
            read_private(p)
        if read_private(tls / "servidor.crt").count(b"-----BEGIN CERTIFICATE-----") != 2:
            fail("servidor TLS existente sin cadena completa hoja+AC")
        run(["openssl", "verify", "-purpose", "sslserver", "-CAfile", str(ca / "ac.crt"),
             str(tls / "servidor.crt")])
        print("Material CA/servidor existente y privado; sin cambios")
        return
    with tempfile.TemporaryDirectory(prefix=".ca-", dir=root) as temporary:
        tmp = Path(temporary)
        password = secrets.token_urlsafe(48).encode() + b"\n"
        (tmp / "ac.password").write_bytes(password)
        os.chmod(tmp / "ac.password", 0o600)
        run(["openssl", "genpkey", "-algorithm", "EC", "-pkeyopt", "ec_paramgen_curve:P-256",
             "-aes-256-cbc", "-pass", f"file:{tmp / 'ac.password'}", "-out", str(tmp / "ac.key")])
        run(["openssl", "req", "-new", "-x509", "-sha256", "-days", "397",
             "-key", str(tmp / "ac.key"), "-passin", f"file:{tmp / 'ac.password'}",
             "-subj", "/CN=VEC AC desarrollo interno/O=VEC Desarrollo/OU=Sintetico",
             "-addext", "basicConstraints=critical,CA:TRUE,pathlen:0",
             "-addext", "keyUsage=critical,keyCertSign,cRLSign", "-out", str(tmp / "ac.crt")])
        run(["openssl", "genpkey", "-algorithm", "EC", "-pkeyopt", "ec_paramgen_curve:P-256",
             "-out", str(tmp / "servidor.key")])
        run(["openssl", "req", "-new", "-sha256", "-key", str(tmp / "servidor.key"),
             "-subj", f"/CN={server_name}/O=VEC Desarrollo/OU=Sintetico",
             "-out", str(tmp / "servidor.csr")])
        (tmp / "servidor.ext").write_text("basicConstraints=critical,CA:FALSE\n"
            "keyUsage=critical,digitalSignature,keyEncipherment\n"
            "extendedKeyUsage=serverAuth\n" f"subjectAltName=DNS:{server_name}\n")
        run(["openssl", "x509", "-req", "-sha256", "-days", "397", "-in", str(tmp / "servidor.csr"),
             "-CA", str(tmp / "ac.crt"), "-CAkey", str(tmp / "ac.key"),
             "-passin", f"file:{tmp / 'ac.password'}", "-set_serial", "0x" + secrets.token_hex(16),
             "-extfile", str(tmp / "servidor.ext"), "-out", str(tmp / "servidor.crt")])
        run(["openssl", "genpkey", "-algorithm", "ED25519", "-out", str(tmp / "emisor.key")])
        run(["openssl", "pkey", "-in", str(tmp / "emisor.key"), "-pubout", "-out", str(tmp / "emisor.pub")])
        for source, dest in (("ac.key", ca), ("ac.crt", ca), ("ac.password", ca),
                             ("servidor.key", tls), ("emisor.key", identity),
                             ("emisor.pub", identity)):
            atomic_private(dest / source, (tmp / source).read_bytes())
        # La raíz interna exige cadena servidor hoja+AC, no solo la hoja.
        fullchain = (tmp / "servidor.crt").read_bytes().rstrip() + b"\n" + \
                    (tmp / "ac.crt").read_bytes().rstrip() + b"\n"
        atomic_private(tls / "servidor.crt", fullchain)
    if any(ch in str(ca) for ch in "\n\r$#"):
        fail("ruta de CA incompatible con configuración OpenSSL")
    ca_config = f"""[ ca ]
default_ca = vec_interno_ca
[ vec_interno_ca ]
database = {ca / 'index.txt'}
new_certs_dir = {ca / 'newcerts'}
certificate = {ca / 'ac.crt'}
private_key = {ca / 'ac.key'}
serial = {ca / 'serial'}
crlnumber = {ca / 'crlnumber'}
default_md = sha256
policy = politica_sintetica
unique_subject = no
default_crl_days = 45
copy_extensions = none
[ politica_sintetica ]
commonName = supplied
organizationName = optional
organizationalUnitName = optional
"""
    subdir(ca, "newcerts")
    atomic_private(ca / "ca.cnf", ca_config.encode())
    atomic_private(ca / "index.txt", b"")
    atomic_private(ca / "serial", secrets.token_hex(16).upper().encode() + b"\n")
    atomic_private(ca / "crlnumber", b"1000\n")
    refresh_crl(root)
    print("CA propia, servidor TLS y emisor de aserciones creados fuera de Git")


def refresh_crl(root: Path) -> None:
    ca = root / "ca"
    identity = subdir(root, "identidad")
    for file in (ca / "ca.cnf", ca / "ac.crt", ca / "ac.key", ca / "ac.password", ca / "index.txt"):
        read_private(file)
    with tempfile.TemporaryDirectory(prefix=".crl-", dir=identity) as temporary:
        generated = Path(temporary) / "clientes.crl"
        run(["openssl", "ca", "-gencrl", "-batch", "-config", str(ca / "ca.cnf"),
             "-passin", f"file:{ca / 'ac.password'}", "-out", str(generated)])
        run(["openssl", "crl", "-in", str(generated), "-noout"])
        replace_private(identity / "clientes.crl", generated.read_bytes())
    print("CRL de la AC de desarrollo firmada y renovada")


def issue_person(root: Path, csr_path: Path, subject_id: str, token_url: str,
                 pkcs11_module: Path, p11tool: Path, pin_file: Path) -> None:
    check_tool("openssl")
    if not re.fullmatch(r"per_[a-z0-9_]{22,128}", subject_id):
        fail("sujeto opaco inválido")
    ca, people = root / "ca", subdir(root, "personas")
    for existing in people.glob("*.token.json"):
        if existing.name != f"{subject_id}.token.json" and json.loads(read_private(existing)).get("token_url") == token_url:
            fail("la clave del token ya pertenece a otra persona sintética")
    read_private(ca / "ac.key")
    read_private(ca / "ac.password")
    read_private(ca / "ac.crt")
    if csr_path.is_symlink() or not csr_path.is_file():
        fail("CSR de token ausente")
    if not token_url.startswith("pkcs11:") or "type=private" not in token_url:
        fail("URL PKCS#11 de clave privada inválida")
    if pkcs11_module.is_symlink() or not pkcs11_module.is_file() or p11tool.is_symlink() or not p11tool.is_file():
        fail("módulo o herramienta PKCS#11 ausentes")
    env = token_environment(pin_file)
    info = run([str(p11tool), "--provider", str(pkcs11_module), "--login", "--info", token_url], env=env)
    if b"CKA_NEVER_EXTRACTABLE" not in info or b"CKA_SENSITIVE" not in info or b"CKA_PRIVATE" not in info:
        fail("la clave del token no acredita protección y no exportabilidad")
    public_token = run([str(p11tool), "--provider", str(pkcs11_module), "--login",
                        "--export-pubkey", token_url], env=env)
    public_csr = run(["openssl", "req", "-in", str(csr_path), "-pubkey", "-noout"])
    der_token = run(["openssl", "pkey", "-pubin", "-outform", "DER"], input_bytes=public_token)
    der_csr = run(["openssl", "pkey", "-pubin", "-outform", "DER"], input_bytes=public_csr)
    if der_token != der_csr:
        fail("CSR no corresponde a la clave no exportable del token")
    # OpenSSL comprueba la prueba de posesión de clave que contiene el CSR.
    run(["openssl", "req", "-in", str(csr_path), "-verify", "-noout"])
    subject = run(["openssl", "req", "-in", str(csr_path), "-noout", "-subject",
                   "-nameopt", "RFC2253"]).decode().strip()
    if not re.search(r"(?:^|[=,])CN=vec-rrhh-sintetico(?:,|$)", subject):
        fail("CSR sin identidad sintética esperada")
    dest = people / f"{subject_id}.crt"
    proof = people / f"{subject_id}.token.json"
    if dest.exists() or dest.is_symlink():
        read_private(dest)
        run(["openssl", "verify", "-purpose", "sslclient", "-CAfile", str(ca / "ac.crt"), str(dest)])
        cert_public = run(["openssl", "x509", "-in", str(dest), "-pubkey", "-noout"])
        if run(["openssl", "pkey", "-pubin", "-outform", "DER"], input_bytes=cert_public) != der_token:
            fail("certificado existente no corresponde al token")
        expected = {"version": 1, "cert_sha256": digest_cert(dest), "token_url": token_url,
                    "module": str(pkcs11_module)}
        if proof.exists():
            if json.loads(read_private(proof)) != expected:
                fail("prueba de token existente incompatible")
        else:
            atomic_private(proof, (json.dumps(expected, separators=(",", ":")) + "\n").encode())
        print(f"Certificado ya emitido; huella SHA256: {digest_cert(dest)}")
        return
    with tempfile.TemporaryDirectory(prefix=".persona-", dir=people) as temporary:
        tmp = Path(temporary)
        (tmp / "persona.ext").write_text("basicConstraints=critical,CA:FALSE\n"
             "keyUsage=critical,digitalSignature\nextendedKeyUsage=clientAuth\n"
             f"subjectAltName=URI:urn:vec:desarrollo:persona:{subject_id}\n")
        run(["openssl", "ca", "-batch", "-config", str(ca / "ca.cnf"),
             "-passin", f"file:{ca / 'ac.password'}", "-days", "37",
             "-in", str(csr_path), "-extfile", str(tmp / "persona.ext"),
             "-out", str(tmp / "persona.crt")])
        atomic_private(dest, (tmp / "persona.crt").read_bytes())
    attestation = {"version": 1, "cert_sha256": digest_cert(dest), "token_url": token_url,
                   "module": str(pkcs11_module)}
    atomic_private(proof, (json.dumps(attestation, separators=(",", ":")) + "\n").encode())
    print(f"Certificado emitido desde CSR de token; huella SHA256: {digest_cert(dest)}")


def register_person(root: Path, subject_id: str, account_id: str) -> None:
    for value, prefix in ((subject_id, "per_"), (account_id, "cta_")):
        if not re.fullmatch(re.escape(prefix) + r"[a-z0-9_]{22,128}", value):
            fail("referencia opaca inválida")
    ca = root / "ca" / "ac.crt"
    cert = root / "personas" / f"{subject_id}.crt"
    proof = root / "personas" / f"{subject_id}.token.json"
    read_private(ca)
    read_private(cert)
    attest = json.loads(read_private(proof))
    if attest.get("version") != 1 or attest.get("cert_sha256") != digest_cert(cert):
        fail("certificado sin prueba de emisión con token")
    run(["openssl", "verify", "-purpose", "sslclient", "-CAfile", str(ca), str(cert)])
    huella = "sha256:" + digest_cert(cert)
    identity = subdir(root, "identidad")
    path = identity / "certificados.json"
    record = {"huella_sha256": huella, "sujeto_id": subject_id, "cuenta_id": account_id,
              "proteccion_clave_ref": "pkcs11-pin-no-exportable", "activo": True}
    if path.exists():
        current = json.loads(read_private(path))
        if current.get("version") != 1 or not isinstance(current.get("certificados"), list):
            fail("registro de certificados incompatible")
        for old in current["certificados"]:
            if old.get("huella_sha256") == huella or old.get("sujeto_id") == subject_id or old.get("cuenta_id") == account_id:
                if old != record:
                    fail("registro existente distinto; la revocación no se revierte automáticamente")
                print("Vínculo técnico existente sin cambios")
                return
        # El registro se relee por petición. La actualización es atómica y
        # mantiene cualquier revocación previa. Nunca reactiva una entrada.
        current["certificados"].append(record)
        data = (json.dumps(current, ensure_ascii=False, separators=(",", ":")) + "\n").encode()
        fd, temp = tempfile.mkstemp(prefix=".certificados-", dir=identity)
        try:
            with os.fdopen(fd, "wb") as file:
                file.write(data)
                file.flush()
                os.fsync(file.fileno())
            os.chmod(temp, 0o600)
            os.replace(temp, path)
        finally:
            if os.path.exists(temp):
                os.unlink(temp)
    else:
        atomic_private(path, (json.dumps({"version": 1, "certificados": [record]},
                                         ensure_ascii=False, separators=(",", ":")) + "\n").encode())
    print("Vínculo técnico de certificado registrado fuera de Git; falta acreditar F1/V3")


def register_context(root: Path, account_id: str, profile_id: str, organization: str,
                     unit: str) -> None:
    for value, prefix in ((account_id, "cta_"), (profile_id, "prf_")):
        if not re.fullmatch(re.escape(prefix) + r"[a-z0-9_]{22,128}", value):
            fail("referencia de contexto inválida")
    if not (re.fullmatch(r"ref:[0-9a-f]{64}", organization) or
            re.fullmatch(r"organizacion:[a-z0-9_:-]{4,128}", organization)):
        fail("organización no corresponde a referencia CT")
    # Mismo contrato que UnidadSeguimientoValida: referencia opaca o nominal
    # «unidad:…» ya usada por el seguimiento CT existente.
    if not (re.fullmatch(r"ref:[0-9a-f]{64}", unit) or
            re.fullmatch(r"unidad:[a-z0-9_:-]{4,128}", unit)):
        fail("unidad sin referencia CT válida")
    # La cuenta del selector es la referencia interna de Identidad/F1 que
    # devuelve el registro de sesión. Nunca coincide con la cuenta externa del
    # certificado (la sesión rechaza referencias que procedan de la entrada);
    # ambas se enlazan por el alias HMAC registrado en Identidad.
    certs = json.loads(read_private(root / "identidad" / "certificados.json"))
    if any(c.get("cuenta_id") == account_id for c in certs.get("certificados", [])):
        fail("la cuenta del selector debe ser la interna de F1, no la del certificado")
    if not any(c.get("activo") is True for c in certs.get("certificados", [])):
        fail("sin certificado activo en el material privado")
    path = root / "identidad" / "contextos.json"
    record = {"cuenta_ref": account_id, "perfil_ref": profile_id,
              "organizacion_ref": organization, "unidad_ref": unit}
    if path.exists():
        current = json.loads(read_private(path))
        if current.get("version") != 1 or not isinstance(current.get("contextos"), list):
            fail("selector de contexto incompatible")
        for old in current["contextos"]:
            if old.get("cuenta_ref") == account_id:
                if old != record:
                    fail("cuenta ya asociada a otro perfil; revocar en F1 y reprovisionar")
                print("Selector nominal existente sin cambios")
                return
        current["contextos"].append(record)
        data = (json.dumps(current, ensure_ascii=False, separators=(",", ":")) + "\n").encode()
        fd, temp = tempfile.mkstemp(prefix=".contextos-", dir=path.parent)
        try:
            with os.fdopen(fd, "wb") as file:
                file.write(data)
                file.flush()
                os.fsync(file.fileno())
            os.chmod(temp, 0o600)
            os.replace(temp, path)
        finally:
            if os.path.exists(temp):
                os.unlink(temp)
    else:
        atomic_private(path, (json.dumps({"version": 1, "contextos": [record]},
                                         ensure_ascii=False, separators=(",", ":")) + "\n").encode())
    print("Selector nominal guardado; la resolución y vigencia corresponden a F1")


def revoke_person(root: Path, subject_id: str) -> None:
    if not re.fullmatch(r"per_[a-z0-9_]{22,128}", subject_id):
        fail("sujeto opaco inválido")
    path = root / "identidad" / "certificados.json"
    current = json.loads(read_private(path))
    if current.get("version") != 1 or not isinstance(current.get("certificados"), list):
        fail("registro de certificados incompatible")
    matches = [c for c in current["certificados"] if c.get("sujeto_id") == subject_id]
    if len(matches) != 1:
        fail("vínculo de certificado ausente o ambiguo")
    if matches[0].get("activo") not in (True, False):
        fail("estado de admisión inválido")
    if matches[0]["activo"]:
        # Denegación primero: ante fallo posterior de AC o interrupción, el
        # registro vivo corta el acceso y repetir completa la revocación X.509.
        matches[0]["activo"] = False
        data = (json.dumps(current, ensure_ascii=False, separators=(",", ":")) + "\n").encode()
        replace_private(path, data)
    cert = root / "personas" / f"{subject_id}.crt"
    read_private(cert)
    ca = root / "ca"
    index = read_private(ca / "index.txt").decode()
    serial = run(["openssl", "x509", "-in", str(cert), "-noout", "-serial"]).decode().strip().split("=", 1)[-1].upper()
    rows = [row.split("\t") for row in index.splitlines() if row]
    found = [row for row in rows if len(row) >= 4 and row[3].lstrip("0") == serial.lstrip("0")]
    if len(found) != 1:
        fail("certificado no inscrito de forma única en la AC de desarrollo")
    if found[0][0] == "V":
        run(["openssl", "ca", "-batch", "-config", str(ca / "ca.cnf"),
             "-passin", f"file:{ca / 'ac.password'}", "-revoke", str(cert)])
    elif found[0][0] != "R":
        fail("estado de certificado en AC incompatible")
    refresh_crl(root)
    crl_text = run(["openssl", "crl", "-in", str(root / "identidad" / "clientes.crl"),
                    "-noout", "-text"]).decode()
    if not re.search(r"Serial Number:\s*0*" + re.escape(serial.lstrip("0")) + r"(?:\s|$)", crl_text, re.IGNORECASE):
        fail("CRL firmada sin serial revocado")
    print("Certificado excluido y serial publicado en CRL firmada de la AC de desarrollo")


def role_sql(passwords: dict[str, str], database: str, finish: str,
             roles_map: dict[str, str] = ROLES,
             functions_map: dict[str, str | list[str]] = FUNCTIONS,
             login_prefix: str = "vec_interno_",
             login_names: dict[str, str] | None = None) -> str:
    lines = ["BEGIN;", "SET LOCAL search_path = pg_catalog;", "SET LOCAL lock_timeout = '5s';"]
    lines.append("DO $version$ BEGIN IF pg_catalog.current_setting('server_version_num')::integer / 10000 <> 18 THEN RAISE EXCEPTION 'se exige PostgreSQL 18' USING ERRCODE='55000'; END IF; END $version$;")
    for key, role in roles_map.items():
        login = name(login_names[key] if login_names and key in login_names else f"{login_prefix}{key}_desarrollo")
        group = name(role)
        functions = functions_map[key]
        if isinstance(functions, str):
            functions = [functions]
        acl_checks = "\n".join(f"""  IF pg_catalog.to_regprocedure({sql_quote(function)}) IS NULL
    OR NOT pg_catalog.has_function_privilege({sql_quote(group)}, {sql_quote(function)}, 'EXECUTE')
  THEN RAISE EXCEPTION 'función/ACL requerida ausente: {key}' USING ERRCODE='55000'; END IF;"""
                              for function in functions)
        password = sql_quote(passwords[key])
        lines.append(f"""DO $provision$
DECLARE v_login oid; v_group oid;
BEGIN
  SELECT oid INTO v_group FROM pg_catalog.pg_roles WHERE rolname={sql_quote(group)};
  IF v_group IS NULL THEN RAISE EXCEPTION 'falta rol propietario requerido: {group}' USING ERRCODE='55000'; END IF;
{acl_checks}
  SELECT oid INTO v_login FROM pg_catalog.pg_roles WHERE rolname={sql_quote(login)};
  IF v_login IS NULL THEN
    EXECUTE pg_catalog.format('CREATE ROLE %I LOGIN PASSWORD %L NOSUPERUSER NOCREATEDB NOCREATEROLE INHERIT NOREPLICATION NOBYPASSRLS',
      {sql_quote(login)}, {password});
    EXECUTE pg_catalog.format('GRANT %I TO %I WITH ADMIN FALSE, INHERIT TRUE, SET FALSE',
      {sql_quote(group)}, {sql_quote(login)});
  END IF;
  IF NOT EXISTS (
    SELECT 1 FROM pg_catalog.pg_roles l JOIN pg_catalog.pg_auth_members m ON m.member=l.oid
    JOIN pg_catalog.pg_roles g ON g.oid=m.roleid
    WHERE l.rolname={sql_quote(login)} AND l.rolcanlogin AND l.rolinherit
      AND NOT (l.rolsuper OR l.rolcreatedb OR l.rolcreaterole OR l.rolreplication OR l.rolbypassrls)
      AND l.rolconfig IS NULL AND l.rolconnlimit=-1 AND l.rolvaliduntil IS NULL
      AND g.rolname={sql_quote(group)} AND NOT g.rolcanlogin
      AND NOT (g.rolsuper OR g.rolcreatedb OR g.rolcreaterole OR g.rolreplication OR g.rolbypassrls)
      AND NOT m.admin_option AND m.inherit_option AND NOT m.set_option
      AND (SELECT count(*) FROM pg_catalog.pg_auth_members WHERE member=l.oid)=1
      AND NOT EXISTS (SELECT 1 FROM pg_catalog.pg_auth_members WHERE roleid=l.oid)
      AND NOT EXISTS (SELECT 1 FROM pg_catalog.pg_shdepend WHERE refobjid=l.oid)
      AND NOT EXISTS (SELECT 1 FROM pg_catalog.pg_db_role_setting WHERE setrole=l.oid)
  ) THEN RAISE EXCEPTION 'LOGIN nominal incompatible: {login}' USING ERRCODE='42501'; END IF;
  EXECUTE pg_catalog.format('ALTER ROLE %I PASSWORD %L', {sql_quote(login)}, {password});
END $provision$;""")
        lines.append(f"DO $connect$ BEGIN EXECUTE pg_catalog.format('GRANT CONNECT ON DATABASE %I TO %I', {sql_quote(database)}, {sql_quote(group)}); END $connect$;")
    lines.append(finish + ";")
    return "\n".join(lines)


def psql(sql: str, container: str | None, engine: str, database: str, admin_user: str) -> None:
    if container:
        if engine not in ("docker", "podman"):
            fail("motor de contenedor inválido")
        check_tool(engine)
        if not re.fullmatch(r"[A-Za-z0-9_.-]+", container):
            fail("contenedor inválido")
        command = [engine, "exec", "-i", container, "psql", "-X", "-q", "-v", "ON_ERROR_STOP=1",
                   "-U", admin_user, "-d", database]
    else:
        check_tool("psql")
        command = ["psql", "-X", "-q", "-v", "ON_ERROR_STOP=1", "-U", admin_user, "-d", database]
    run(command, input_text=sql)


def roles(root: Path, args: argparse.Namespace, *, identity: bool = False,
          v3: bool = False) -> None:
    if not re.fullmatch(r"[A-Za-z][A-Za-z0-9_-]{0,62}", args.database) or not re.fullmatch(r"[A-Za-z][A-Za-z0-9_-]{0,62}", args.admin_user):
        fail("base o usuario administrativo inválidos")
    if not re.fullmatch(r"[A-Za-z0-9.-]{1,253}", args.db_host) or not 1 <= args.db_port <= 65535:
        fail("destino PostgreSQL inválido")
    db_ca = Path(args.db_ca).absolute()
    if db_ca.is_symlink() or not db_ca.is_file():
        fail("CA de PostgreSQL ausente")
    if identity and v3:
        fail("conjunto de pools ambiguo")
    base = subdir(root, "identidad") if identity else root
    state_path = base / ("ct-v3-pools-state.json" if v3 else "pools-state.json")
    output_path = base / ("ct_v3_pools.json" if v3 else "pools.json")
    role_map = V3_ROLES if v3 else IDENTITY_ROLES if identity else ROLES
    function_map = V3_FUNCTIONS if v3 else IDENTITY_FUNCTIONS if identity else FUNCTIONS
    prefix = "vec_interno_v3_" if v3 else "vec_interno_identidad_" if identity else "vec_interno_"
    login_names = {"gobierno_v3": "vec_interno_preflight_v3_desarrollo"} if v3 else None
    if state_path.exists():
        state = json.loads(read_private(state_path))
        if state.get("version") != 1 or state.get("database") != args.database or set(state.get("passwords", {})) != set(role_map):
            fail("estado de pools incompatible")
    else:
        state = {"version": 1, "database": args.database,
                 "passwords": {key: secrets.token_urlsafe(36) for key in role_map}}
        atomic_private(state_path, (json.dumps(state, separators=(",", ":")) + "\n").encode())
    passwords = state["passwords"]
    psql(role_sql(passwords, args.database, "ROLLBACK", role_map, function_map, prefix, login_names), args.pg_container, args.container_engine, args.database, args.admin_user)
    psql(role_sql(passwords, args.database, "COMMIT", role_map, function_map, prefix, login_names), args.pg_container, args.container_engine, args.database, args.admin_user)
    pools = {}
    for key in role_map:
        login = login_names[key] if login_names and key in login_names else f"{prefix}{key}_desarrollo"
        netloc = f"{quote(login)}:{quote(passwords[key], safe='')}@{args.db_host}:{args.db_port}"
        query = urlencode({"sslmode": "verify-full", "sslrootcert": str(db_ca)})
        pools[key] = {"login": login, "dsn": f"postgresql://{netloc}/{quote(args.database)}?{query}"}
    generated = (json.dumps({"version": 1, "pools": pools}, separators=(",", ":")) + "\n").encode()
    if output_path.exists():
        if read_private(output_path) != generated:
            fail("pools.json existente difiere del estado; no se sobrescribe")
    else:
        atomic_private(output_path, generated)
    print(f"{len(role_map)} LOGIN nominales ensayados y confirmados; pools.json privado preparado")


def policy(root: Path, args: argparse.Namespace) -> None:
    if not re.fullmatch(r"pga_[A-Za-z0-9_-]{22,128}", args.reference):
        fail("referencia de política inválida")
    if not re.fullmatch(r"[A-Za-z0-9_-]{8,128}", args.key_id):
        fail("identificador de emisor inválido")
    if not re.fullmatch(r"[0-9a-f]{64}", args.fingerprint) or args.fingerprint == "0" * 64:
        fail("huella de política inválida")
    if not re.fullmatch(r"\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}Z", args.expires_at):
        fail("retirada debe usar segundo UTC RFC3339 exacto")
    try:
        expires = datetime.fromisoformat(args.expires_at.replace("Z", "+00:00"))
    except ValueError:
        fail("fecha de retirada inválida")
    if expires.tzinfo is None or expires.utcoffset().total_seconds() != 0 or expires <= datetime.now(timezone.utc):
        fail("retirada debe ser futura y UTC")
    state = {"version": 1, "politica_ref": args.reference, "huella_sha256": args.fingerprint,
             "retirar_en": args.expires_at, "clave_id": args.key_id}
    path = subdir(root, "identidad") / "politica.json"
    read_private(root / "identidad" / "emisor.key")
    if path.exists() and json.loads(read_private(path)) != state:
        fail("política privada existente distinta; no se sustituye")
    metadata = {"version": 1, "clave_id": args.key_id, "politica_ref": args.reference,
                "politica_huella": "sha256:" + args.fingerprint}
    issuer_path = root / "identidad" / "emisor.json"
    encoded = (json.dumps(metadata, separators=(",", ":")) + "\n").encode()
    if issuer_path.exists() and read_private(issuer_path) != encoded:
        fail("metadatos de emisor existentes distintos")
    sql = f"""BEGIN;
SET LOCAL ROLE vec_identidad_sesiones_v1_propietario;
SET LOCAL search_path = pg_catalog;
SET LOCAL lock_timeout = '5s';
INSERT INTO vec_identidad_sesiones_v1.politica_certificado_personal_desarrollo_v1
  (singleton, politica_ref, huella_sha256, retirar_en, activa)
VALUES (true,{sql_quote(args.reference)},{sql_quote(args.fingerprint)},
  {sql_quote(args.expires_at)}::timestamptz,true)
ON CONFLICT (singleton) DO NOTHING;
DO $comprobar$ BEGIN
  IF NOT EXISTS (SELECT 1 FROM vec_identidad_sesiones_v1.politica_certificado_personal_desarrollo_v1
    WHERE singleton=true AND politica_ref={sql_quote(args.reference)}
      AND huella_sha256={sql_quote(args.fingerprint)}
      AND retirar_en={sql_quote(args.expires_at)}::timestamptz AND activa=true)
  THEN RAISE EXCEPTION 'política de certificado distinta o revocada' USING ERRCODE='55000'; END IF;
END $comprobar$;
FINALIZAR;"""
    for final in ("ROLLBACK", "COMMIT"):
        psql(sql.replace("FINALIZAR", final), args.pg_container, args.container_engine,
             args.database, args.admin_user)
    if not path.exists():
        atomic_private(path, (json.dumps(state, separators=(",", ":")) + "\n").encode())
    if not issuer_path.exists():
        atomic_private(issuer_path, encoded)
    print("Política de certificado de desarrollo instalada y cotejada; no se sobrescribió historia")


def hmac_token(root: Path, args: argparse.Namespace) -> None:
    import pkcs11_hmac

    module = Path(args.pkcs11_module).absolute()
    pin_path = Path(args.pin_file).absolute()
    if pin_path != root / "identidad" / "hmac.pin":
        fail("PIN HMAC debe estar en identidad/hmac.pin del material privado")
    if not re.fullmatch(r"[A-Za-z0-9_.-]{1,32}", args.token_label) or \
       not re.fullmatch(r"[A-Za-z0-9]{1,16}", args.token_serial):
        fail("token PKCS#11 inválido")
    if not re.fullmatch(r"[0-9a-f]{16,64}", args.object_id_hex) or len(args.object_id_hex) % 2:
        fail("ID de objeto PKCS#11 inválido")
    if not re.fullmatch(r"[A-Za-z0-9_-]{8,128}", args.key_id) or args.key_version < 1:
        fail("versión o ID de clave inválido")
    if not re.fullmatch(r"idh_[a-z0-9_]{22,128}", args.domain_ref) or not args.identity_space.startswith("https://"):
        fail("dominio/espacio de identidad inválido")
    pin = bytearray(read_private(pin_path).strip())
    if len(pin) < 8 or len(pin) > 128:
        fail("PIN HMAC inválido")
    identity = subdir(root, "identidad")
    metadata = {"version": 1, "modulo": str(module), "token_label": args.token_label,
                "token_serial": args.token_serial, "objeto_id_hex": args.object_id_hex,
                "clave_id": args.key_id, "clave_version": args.key_version,
                "dominio_ref": args.domain_ref, "espacio_identidad": args.identity_space,
                "pin_fichero": str(pin_path)}
    path = identity / "hmac.json"
    encoded = (json.dumps(metadata, separators=(",", ":")) + "\n").encode()
    if path.exists() and read_private(path) != encoded:
        fail("configuración HMAC privada existente incompatible")
    try:
        pkcs11_hmac.provision(module, args.token_label, args.token_serial,
                              bytes.fromhex(args.object_id_hex), args.key_id, bytes(pin))
    finally:
        for i in range(len(pin)):
            pin[i] = 0
    if not path.exists():
        atomic_private(path, encoded)
    print("Clave HMAC SHA256 no exportable verificada dentro del token; metadatos privados escritos")


def mensaje_hmac_identidad(metadata: dict, proposito: str, identificador: str) -> bytes:
    """Mismos seis campos y longitudes big-endian que mensajeCanonico de Go."""
    partes = ("vec.identidad.hmac-sha256.v1", metadata["dominio_ref"],
              metadata["espacio_identidad"], str(metadata["clave_version"]),
              proposito, identificador)
    if len(identificador.encode("utf-8")) > 4096:
        fail("identificador de identidad demasiado largo")
    return b"".join(len(parte.encode("utf-8")).to_bytes(4, "big") + parte.encode("utf-8")
                    for parte in partes)


def firmar_alias_hmac(metadata: dict, pin: bytearray, cuenta: str, sujeto: str) -> tuple[bytes, bytes]:
    """Firma en el token; CKA_VALUE nunca se consulta ni sale del proveedor."""
    import pkcs11_hmac as p

    module = Path(metadata["modulo"])
    if not module.is_absolute() or module.is_symlink() or not module.is_file():
        fail("módulo PKCS#11 HMAC ausente o inseguro")
    lib = c.CDLL(str(module))
    lib.C_Initialize.argtypes = [c.c_void_p]
    lib.C_Initialize.restype = p.CK
    lib.C_Finalize.argtypes = [c.c_void_p]
    lib.C_Finalize.restype = p.CK
    lib.C_GetSlotList.argtypes = [p.BOOL, c.POINTER(p.CK), c.POINTER(p.CK)]
    lib.C_GetSlotList.restype = p.CK
    lib.C_GetTokenInfo.argtypes = [p.CK, c.c_void_p]
    lib.C_GetTokenInfo.restype = p.CK
    lib.C_OpenSession.argtypes = [p.CK, p.CK, c.c_void_p, c.c_void_p, c.POINTER(p.CK)]
    lib.C_OpenSession.restype = p.CK
    lib.C_CloseSession.argtypes = [p.CK]
    lib.C_CloseSession.restype = p.CK
    lib.C_Login.argtypes = [p.CK, p.CK, c.c_void_p, p.CK]
    lib.C_Login.restype = p.CK
    lib.C_Logout.argtypes = [p.CK]
    lib.C_Logout.restype = p.CK
    lib.C_FindObjectsInit.argtypes = [p.CK, c.POINTER(p.Attribute), p.CK]
    lib.C_FindObjectsInit.restype = p.CK
    lib.C_FindObjects.argtypes = [p.CK, c.POINTER(p.CK), p.CK, c.POINTER(p.CK)]
    lib.C_FindObjects.restype = p.CK
    lib.C_FindObjectsFinal.argtypes = [p.CK]
    lib.C_FindObjectsFinal.restype = p.CK
    lib.C_GetAttributeValue.argtypes = [p.CK, p.CK, c.POINTER(p.Attribute), p.CK]
    lib.C_GetAttributeValue.restype = p.CK
    lib.C_SignInit.argtypes = [p.CK, c.POINTER(p.Mechanism), p.CK]
    lib.C_SignInit.restype = p.CK
    lib.C_Sign.argtypes = [p.CK, c.c_void_p, p.CK, c.c_void_p, c.POINTER(p.CK)]
    lib.C_Sign.restype = p.CK
    p.check(lib.C_Initialize(None), "inicializar")
    session = None
    logged = False
    try:
        count = p.CK()
        p.check(lib.C_GetSlotList(p.BOOL(1), None, c.byref(count)), "enumerar tokens")
        if not 1 <= count.value <= 64:
            fail("número de tokens PKCS#11 inválido")
        slots = (p.CK * count.value)()
        p.check(lib.C_GetSlotList(p.BOOL(1), slots, c.byref(count)), "leer tokens")
        selected = []
        for slot in slots[:count.value]:
            raw = c.create_string_buffer(256)
            p.check(lib.C_GetTokenInfo(slot, raw), "leer token")
            if (raw.raw[:32].decode("ascii", errors="ignore").strip() == metadata["token_label"] and
                raw.raw[80:96].decode("ascii", errors="ignore").strip() == metadata["token_serial"]):
                selected.append(slot)
        if len(selected) != 1:
            fail("token HMAC ausente o ambiguo")
        handle = p.CK()
        p.check(lib.C_OpenSession(selected[0], p.CKF_SERIAL_SESSION, None, None,
                                  c.byref(handle)), "abrir sesión")
        session = handle.value
        pin_buffer = c.create_string_buffer(bytes(pin), len(pin))
        try:
            result = lib.C_Login(session, p.CKU_USER, pin_buffer, len(pin))
        finally:
            c.memset(pin_buffer, 0, len(pin_buffer))
        if result not in (p.CKR_OK, p.CKR_USER_ALREADY_LOGGED_IN):
            p.check(result, "PIN")
        logged = result == p.CKR_OK
        search, keep = p._attributes([(0x000, p.CK(p.CKO_SECRET_KEY)),
                                       (0x102, bytes.fromhex(metadata["objeto_id_hex"]))])
        p.check(lib.C_FindObjectsInit(session, search, len(search)), "buscar clave")
        try:
            found = (p.CK * 2)()
            found_count = p.CK()
            p.check(lib.C_FindObjects(session, found, 2, c.byref(found_count)), "enumerar claves")
        finally:
            p.check(lib.C_FindObjectsFinal(session), "cerrar búsqueda")
        if found_count.value != 1:
            fail("clave HMAC ausente o ambigua")
        key = found[0]
        for kind, expected in ((0x000, p.CKO_SECRET_KEY), (0x100, p.CKK_GENERIC_SECRET)):
            value = p.CK()
            attr, retained = p._attr(kind, value)
            p.check(lib.C_GetAttributeValue(session, key, c.byref(attr), 1), "verificar tipo de clave")
            if value.value != expected:
                fail("tipo de clave HMAC incompatible")
        for kind, expected in ((0x001, 1), (0x002, 1), (0x103, 1), (0x108, 1),
                               (0x162, 0), (0x163, 1), (0x164, 1), (0x165, 1)):
            value = p.BOOL()
            attr, retained = p._attr(kind, value)
            p.check(lib.C_GetAttributeValue(session, key, c.byref(attr), 1), "verificar clave")
            if value.value != expected:
                fail("atributos de clave HMAC incompatibles")
        size = p.CK()
        attr, retained = p._attr(0x161, size)
        p.check(lib.C_GetAttributeValue(session, key, c.byref(attr), 1), "verificar longitud")
        if size.value < 32:
            fail("clave HMAC demasiado corta")
        signatures = []
        for purpose, identifier in (("cuenta", cuenta), ("sujeto", sujeto)):
            message = bytearray(mensaje_hmac_identidad(metadata, purpose, identifier))
            try:
                source = (c.c_ubyte * len(message)).from_buffer(message)
                result = c.create_string_buffer(32)
                result_len = p.CK(32)
                mechanism = p.Mechanism(p.CKM_SHA256_HMAC, None, 0)
                p.check(lib.C_SignInit(session, c.byref(mechanism), key), "habilitar HMAC")
                p.check(lib.C_Sign(session, source, len(message), result,
                                   c.byref(result_len)), "firmar identidad")
                if result_len.value != 32 or result.raw == bytes(32):
                    fail("huella HMAC inválida")
                signatures.append(result.raw)
            finally:
                c.memset(source, 0, len(message))
        if signatures[0] == signatures[1]:
            fail("huellas de cuenta y sujeto coinciden")
        return signatures[0], signatures[1]
    finally:
        if session is not None:
            if logged:
                lib.C_Logout(session)
            lib.C_CloseSession(session)
        lib.C_Finalize(None)


def sql_alias_hmac(operacion: str, cuenta: str, sujeto: str, perfil: str,
                   organizacion: str, metadata: dict,
                   huella_cuenta: bytes, huella_sujeto: bytes, finish: str) -> str:
    if finish not in ("ROLLBACK", "COMMIT"):
        fail("fin de transacción inválido")
    coordinates = (sql_quote(metadata["dominio_ref"]), sql_quote(metadata["clave_id"]),
                   metadata["clave_version"])
    return f"""BEGIN ISOLATION LEVEL SERIALIZABLE;
SET LOCAL search_path = pg_catalog;
SET LOCAL lock_timeout = '5s';
SET LOCAL statement_timeout = '15s';
SET LOCAL ROLE vec_contexto_actor_v1_propietario;
DO $vinculo$ DECLARE ahora timestamptz := clock_timestamp(); BEGIN
  PERFORM 1
    FROM vec_contexto_actor_v1.vinculo_corporativo_actual AS ca
    JOIN vec_contexto_actor_v1.vinculo_corporativo_versiones AS cv
      ON cv.vinculo_corporativo_ref=ca.vinculo_corporativo_ref AND cv.version=ca.version
     AND cv.cuenta_ref=ca.cuenta_ref AND cv.superficie=ca.superficie AND cv.uso=ca.uso
    JOIN vec_contexto_actor_v1.proyeccion_cuenta_actual AS pca
      ON pca.cuenta_ref=cv.cuenta_ref AND pca.version=cv.cuenta_version
    JOIN vec_contexto_actor_v1.proyeccion_cuenta_versiones AS pc
      ON pc.cuenta_ref=pca.cuenta_ref AND pc.version=pca.version
    JOIN vec_contexto_actor_v1.persona_actual AS pea
      ON pea.persona_ref=cv.persona_ref AND pea.version=cv.persona_version
    JOIN vec_contexto_actor_v1.persona_versiones AS pe
      ON pe.persona_ref=pea.persona_ref AND pe.version=pea.version
    JOIN vec_contexto_actor_v1.perfil_actual AS pfa
      ON pfa.perfil_ref=cv.perfil_ref AND pfa.version=cv.perfil_version
    JOIN vec_contexto_actor_v1.perfil_versiones AS pf
      ON pf.perfil_ref=pfa.perfil_ref AND pf.version=pfa.version
    JOIN vec_contexto_actor_v1.vinculo_contexto_actual AS vca
      ON vca.vinculo_ref=cv.vinculo_contexto_ref AND vca.version=cv.vinculo_contexto_version
    JOIN vec_contexto_actor_v1.vinculo_contexto_versiones AS vc
      ON vc.vinculo_ref=vca.vinculo_ref AND vc.version=vca.version
    JOIN vec_contexto_actor_v1.organizacion_actual AS oa
      ON oa.organizacion_ref=cv.organizacion_ref AND oa.version=cv.organizacion_version
    JOIN vec_contexto_actor_v1.organizacion_versiones AS ov
      ON ov.organizacion_ref=oa.organizacion_ref AND ov.version=oa.version
   WHERE ca.cuenta_ref={sql_quote(cuenta)}
     AND ca.superficie='interna_corporativa' AND ca.uso='consulta_rrhh'
     AND cv.persona_ref={sql_quote(sujeto)} AND cv.perfil_ref={sql_quote(perfil)}
     AND cv.organizacion_ref={sql_quote(organizacion)}
     AND pf.persona_ref=cv.persona_ref
     AND vc.cuenta_ref=cv.cuenta_ref AND vc.perfil_ref=cv.perfil_ref
     AND vc.persona_ref=cv.persona_ref
     AND cv.estado='activo' AND pc.estado='activo' AND pe.estado='activo'
     AND pf.estado='activo' AND vc.estado='activo' AND ov.estado='activo'
     AND cv.procedencia_autoridad='autoridad_maestra_acreditada'
     AND pc.procedencia_autoridad='autoridad_maestra_acreditada'
     AND pe.procedencia_autoridad='autoridad_maestra_acreditada'
     AND pf.procedencia_autoridad='autoridad_maestra_acreditada'
     AND vc.procedencia_autoridad='autoridad_maestra_acreditada'
     AND ov.procedencia_autoridad='autoridad_maestra_acreditada'
     AND ahora >= cv.vigente_desde AND ahora < cv.vigente_hasta
     AND ahora >= pc.vigente_desde AND ahora < pc.vigente_hasta
     AND ahora >= pe.vigente_desde AND ahora < pe.vigente_hasta
     AND ahora >= pf.vigente_desde AND ahora < pf.vigente_hasta
     AND ahora >= vc.vigente_desde AND ahora < vc.vigente_hasta
     AND ahora >= ov.vigente_desde AND ahora < ov.vigente_hasta
   FOR SHARE OF ca, pca, pea, pfa, vca, oa;
  IF NOT FOUND THEN
    RAISE EXCEPTION 'vínculo F1 de persona, cuenta y perfil ausente o no vigente'
      USING ERRCODE='55000';
  END IF;
END $vinculo$;
SET LOCAL ROLE vec_identidad_sesiones_v1_propietario;
LOCK TABLE vec_identidad_sesiones_v1.alias_hmac_cuenta IN SHARE ROW EXCLUSIVE MODE;
DO $alias$ BEGIN
  IF EXISTS (
    SELECT 1 FROM vec_identidad_sesiones_v1.alias_hmac_cuenta AS a
    WHERE a.esquema_hmac='vec.identidad.hmac-sha256.v1'
      AND a.dominio_hmac_ref={coordinates[0]} AND a.clave_hmac_id={coordinates[1]}
      AND a.clave_hmac_version={coordinates[2]}
      AND (a.cuenta_id_hmac=decode('{huella_cuenta.hex()}','hex')
        OR a.sujeto_id_hmac=decode('{huella_sujeto.hex()}','hex')
        OR a.cuenta_ref={sql_quote(cuenta)})
      AND NOT (a.cuenta_ref={sql_quote(cuenta)}
        AND a.cuenta_id_hmac=decode('{huella_cuenta.hex()}','hex')
        AND a.sujeto_id_hmac=decode('{huella_sujeto.hex()}','hex'))
  ) THEN RAISE EXCEPTION 'alias HMAC cruzado' USING ERRCODE='55000'; END IF;
  IF vec_identidad_sesiones_v1.registrar_alias_hmac_cuenta_v1(
    {sql_quote(operacion)}, {sql_quote(cuenta)}, 'vec.identidad.hmac-sha256.v1',
    {coordinates[0]}, {coordinates[1]}, {coordinates[2]},
    decode('{huella_cuenta.hex()}','hex'), decode('{huella_sujeto.hex()}','hex')
  ) IS DISTINCT FROM {sql_quote(cuenta)} THEN
    RAISE EXCEPTION 'alias HMAC rechazado por Identidad' USING ERRCODE='55000';
  END IF;
END $alias$;
{finish};"""


def registrar_alias_hmac(root: Path, args: argparse.Namespace) -> None:
    if not re.fullmatch(r"per_[a-z0-9_]{22,128}", args.subject_id) or \
       not re.fullmatch(r"cta_[a-z0-9_]{22,128}", args.external_account_id) or \
       not re.fullmatch(r"cta_[a-z0-9_]{22,128}", args.internal_account_ref) or \
       args.external_account_id == args.internal_account_ref or \
       not re.fullmatch(r"org_[a-z0-9]{16,80}", args.corporate_organization_ref):
        fail("referencias de alias inválidas o coincidentes")
    if not re.fullmatch(r"[A-Za-z][A-Za-z0-9_-]{0,62}", args.database) or \
       not re.fullmatch(r"[A-Za-z][A-Za-z0-9_-]{0,62}", args.admin_user):
        fail("base o usuario administrativo inválidos")
    identity = root / "identidad"
    certs = json.loads(read_private(identity / "certificados.json"))
    contexts = json.loads(read_private(identity / "contextos.json"))
    if certs.get("version") != 1 or not isinstance(certs.get("certificados"), list) or \
       contexts.get("version") != 1 or not isinstance(contexts.get("contextos"), list):
        fail("registro local de identidad incompatible")
    if not any(c.get("activo") is True and c.get("sujeto_id") == args.subject_id and
               c.get("cuenta_id") == args.external_account_id for c in certs["certificados"]):
        fail("certificado activo no vinculado a cuenta y sujeto indicados")
    selected = [c for c in contexts["contextos"]
                if c.get("cuenta_ref") == args.internal_account_ref]
    if len(selected) != 1 or not re.fullmatch(r"prf_[a-z0-9_]{22,128}",
                                               selected[0].get("perfil_ref", "")) or \
       not (re.fullmatch(r"ref:[0-9a-f]{64}", selected[0].get("organizacion_ref", "")) or
            re.fullmatch(r"organizacion:[a-z0-9_:-]{4,128}",
                         selected[0].get("organizacion_ref", ""))):
        fail("cuenta interna ausente del selector nominal")
    if any(c.get("cuenta_ref") == args.external_account_id for c in contexts["contextos"]):
        fail("cuenta externa usada como cuenta interna")
    metadata = json.loads(read_private(identity / "hmac.json"))
    expected = {"version", "modulo", "token_label", "token_serial", "objeto_id_hex",
                "clave_id", "clave_version", "dominio_ref", "espacio_identidad", "pin_fichero"}
    if not isinstance(metadata, dict) or set(metadata) != expected or metadata["version"] != 1 or \
       any(not isinstance(metadata[key], str) for key in expected - {"version", "clave_version"}) or \
       not re.fullmatch(r"[0-9a-f]{16,64}", metadata["objeto_id_hex"]) or \
       not isinstance(metadata["clave_version"], int) or isinstance(metadata["clave_version"], bool) or \
       not 1 <= metadata["clave_version"] <= 9223372036854775807 or \
       not re.fullmatch(r"idh_[A-Za-z0-9_-]{22,128}", metadata["dominio_ref"]) or \
       not re.fullmatch(r"[!-~]{1,128}", metadata["clave_id"]) or \
       not re.fullmatch(r"[!-~]{1,32}", metadata["token_label"]) or \
       not re.fullmatch(r"[!-~]{1,32}", metadata["token_serial"]):
        fail("coordenadas HMAC privadas inválidas")
    space = metadata["espacio_identidad"]
    parsed = urlsplit(space)
    if len(space) > 512 or parsed.scheme.lower() != "https" or not parsed.netloc or \
       parsed.username or parsed.password or parsed.query or parsed.fragment or \
       not re.fullmatch(r"[!-~]+", space):
        fail("espacio de identidad HMAC inválido")
    pin_path = identity / "hmac.pin"
    if metadata["pin_fichero"] != str(pin_path):
        fail("ruta de PIN HMAC incompatible")
    pin = bytearray(read_private(pin_path))
    if pin.endswith(b"\n"):
        pin.pop()
    if not 1 <= len(pin) <= 256 or any(byte < 0x21 or byte > 0x7e for byte in pin):
        fail("PIN HMAC inválido")
    try:
        cuenta_hmac, sujeto_hmac = firmar_alias_hmac(metadata, pin,
                                                     args.external_account_id, args.subject_id)
    finally:
        for index in range(len(pin)):
            pin[index] = 0
    operation = "opr_" + hashlib.sha256(
        b"vec.identidad.alias-hmac.v1\0" + args.internal_account_ref.encode() + b"\0" +
        metadata["dominio_ref"].encode() + b"\0" + metadata["clave_id"].encode() + b"\0" +
        str(metadata["clave_version"]).encode() + b"\0" + cuenta_hmac + sujeto_hmac
    ).hexdigest()
    for finish in ("ROLLBACK", "COMMIT"):
        # El selector nominal lleva el ámbito CT («organizacion:…»); el vínculo
        # corporativo de ContextoActor usa su propia referencia «org_…».
        psql(sql_alias_hmac(operation, args.internal_account_ref, args.subject_id,
                           selected[0]["perfil_ref"], args.corporate_organization_ref, metadata,
                           cuenta_hmac, sujeto_hmac, finish), args.pg_container,
             args.container_engine, args.database, args.admin_user)
    print("Alias HMAC de Identidad cotejado y registrado; operación idempotente")


def runtime_env(root: Path, args: argparse.Namespace) -> None:
    if not re.fullmatch(r"[A-Za-z0-9.:-]{3,128}", args.listen) or \
       not re.fullmatch(r"[A-Za-z0-9.-]{1,253}", args.server_name):
        fail("escucha o nombre TLS inválidos")
    if not re.fullmatch(r"[A-Za-z0-9._:/-]{8,256}", args.audience) or \
       not re.fullmatch(r"[A-Za-z0-9._:/-]{8,256}", args.issuer):
        fail("audiencia o emisor inválidos")
    networks = args.allowed_cidrs.split(",")
    if not networks or len(networks) > 16:
        fail("redes permitidas inválidas")
    try:
        networks = [str(ipaddress.ip_network(network, strict=True)) for network in networks]
    except ValueError:
        fail("CIDR inválida")
    for network in networks:
        if ipaddress.ip_network(network).prefixlen == 0:
            fail("red global no permitida")
    identity = root / "identidad"
    certs = json.loads(read_private(identity / "certificados.json"))
    policy_state = json.loads(read_private(identity / "politica.json"))
    active = [entry.get("huella_sha256") for entry in certs.get("certificados", []) if entry.get("activo") is True]
    if not active or any(not isinstance(value, str) or not re.fullmatch(r"sha256:[0-9a-f]{64}", value) for value in active):
        fail("certificados activos ausentes o inválidos")
    read_private(identity / "clientes.crl")
    tls_root = Path(args.tls_root).absolute()
    if tls_root != Path(os.path.normpath(tls_root)) or tls_root.is_symlink():
        fail("ruta TLS instalada no absoluta/limpia")
    for directory in (tls_root, *tls_root.parents):
        info = directory.stat()
        if directory.is_symlink() or not directory.is_dir() or info.st_uid != 0 or info.st_mode & 0o022:
            fail("directorio TLS instalado no root-owned")
    for name, mode in (("servidor.crt", 0o644), ("servidor.key", 0o440),
                       ("clientes-ca.crt", 0o644)):
        path = tls_root / name
        if path.is_symlink() or not path.is_file() or path.stat().st_uid != 0 or \
           path.stat().st_mode & 0o777 != mode:
            fail("TLS instalado con propietario/permisos incompatibles")
    for original, installed in ((root / "tls" / "servidor.crt", tls_root / "servidor.crt"),
                                (root / "tls" / "servidor.key", tls_root / "servidor.key"),
                                (root / "ca" / "ac.crt", tls_root / "clientes-ca.crt")):
        if read_private(original) != installed.read_bytes():
            fail("TLS root-owned no coincide con material emitido")
    server_subject = run(["openssl", "x509", "-in", str(tls_root / "servidor.crt"),
                          "-noout", "-ext", "subjectAltName"]).decode()
    if f"DNS:{args.server_name}" not in server_subject:
        fail("nombre de servidor no coincide con SAN TLS")
    if not re.fullmatch(r"\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}Z", policy_state.get("retirar_en", "")):
        fail("retirada de política privada inválida")
    values = {
        "VEC_INTERNO_HTTP_ADDR": args.listen,
        "VEC_INTERNO_HTTP_ALLOWED_CIDRS": ",".join(networks),
        "VEC_INTERNO_TLS_CERT_FILE": str(tls_root / "servidor.crt"),
        "VEC_INTERNO_TLS_KEY_FILE": str(tls_root / "servidor.key"),
        "VEC_INTERNO_TLS_CLIENT_CA_FILE": str(tls_root / "clientes-ca.crt"),
        "VEC_INTERNO_TLS_SERVER_NAME": args.server_name,
        "VEC_INTERNO_IDENTITY_AUDIENCE": args.audience,
        "VEC_INTERNO_IDENTITY_ISSUER": args.issuer,
        "VEC_INTERNO_PROXY_TLS_SHA256": ",".join(active),
        "VEC_INTERNO_CERT_POLICY_EXPIRES_AT": policy_state["retirar_en"],
        "VEC_INTERNO_MATERIAL_DIR": str(root),
    }
    encoded = "".join(f"{key}={value}\n" for key, value in values.items()).encode()
    replace_private(root / "arrancar-interno.env", encoded)
    print("Entorno privado de vec-interno escrito; sin selectores legados")


def main() -> None:
    parser = argparse.ArgumentParser(description="Aprovisionamiento privado de vec-interno para desarrollo")
    parser.add_argument("--material-dir", required=True)
    commands = parser.add_subparsers(dest="command", required=True)
    ca = commands.add_parser("init-ca")
    ca.add_argument("--server-name", required=True)
    csr = commands.add_parser("create-csr")
    csr.add_argument("--subject-id", required=True)
    csr.add_argument("--token-url", required=True)
    csr.add_argument("--pkcs11-module", required=True)
    csr.add_argument("--p11tool", default="/usr/bin/p11tool")
    csr.add_argument("--certtool", default="/usr/bin/certtool")
    csr.add_argument("--pin-file", required=True)
    person = commands.add_parser("issue-person")
    person.add_argument("--csr", required=True)
    person.add_argument("--subject-id", required=True)
    person.add_argument("--token-url", required=True)
    person.add_argument("--pkcs11-module", required=True)
    person.add_argument("--p11tool", default="/usr/bin/p11tool")
    person.add_argument("--pin-file", required=True)
    registry = commands.add_parser("register-person")
    registry.add_argument("--subject-id", required=True)
    registry.add_argument("--account-id", required=True)
    context = commands.add_parser("register-context")
    context.add_argument("--account-id", required=True)
    context.add_argument("--profile-id", required=True)
    context.add_argument("--organization-ref", required=True)
    context.add_argument("--unit-ref", required=True)
    revoke = commands.add_parser("revoke-person")
    revoke.add_argument("--subject-id", required=True)
    commands.add_parser("refresh-crl")
    pg = commands.add_parser("roles")
    pg.add_argument("--database", required=True)
    pg.add_argument("--admin-user", required=True)
    pg.add_argument("--pg-container")
    pg.add_argument("--container-engine", choices=("docker", "podman"), default="docker")
    pg.add_argument("--db-host", required=True)
    pg.add_argument("--db-port", type=int, required=True)
    pg.add_argument("--db-ca", required=True)
    ipg = commands.add_parser("identity-roles")
    for action in pg._actions:
        if action.dest in ("help",):
            continue
        option = action.option_strings[0]
        if action.dest == "container_engine":
            ipg.add_argument(option, choices=("docker", "podman"), default="docker")
        elif action.dest == "db_port":
            ipg.add_argument(option, type=int, required=True)
        elif action.dest == "pg_container":
            ipg.add_argument(option)
        else:
            ipg.add_argument(option, required=True)
    vpg = commands.add_parser("v3-roles")
    for action in pg._actions:
        if action.dest == "help":
            continue
        option = action.option_strings[0]
        if action.dest == "container_engine":
            vpg.add_argument(option, choices=("docker", "podman"), default="docker")
        elif action.dest == "db_port":
            vpg.add_argument(option, type=int, required=True)
        elif action.dest == "pg_container":
            vpg.add_argument(option)
        else:
            vpg.add_argument(option, required=True)
    pol = commands.add_parser("policy")
    pol.add_argument("--database", required=True)
    pol.add_argument("--admin-user", required=True)
    pol.add_argument("--pg-container")
    pol.add_argument("--container-engine", choices=("docker", "podman"), default="docker")
    pol.add_argument("--reference", required=True)
    pol.add_argument("--fingerprint", required=True)
    pol.add_argument("--expires-at", required=True)
    pol.add_argument("--key-id", required=True)
    hmac = commands.add_parser("hmac-token")
    hmac.add_argument("--pkcs11-module", required=True)
    hmac.add_argument("--token-label", required=True)
    hmac.add_argument("--token-serial", required=True)
    hmac.add_argument("--object-id-hex", required=True)
    hmac.add_argument("--key-id", required=True)
    hmac.add_argument("--key-version", type=int, required=True)
    hmac.add_argument("--domain-ref", required=True)
    hmac.add_argument("--identity-space", required=True)
    hmac.add_argument("--pin-file", required=True)
    alias = commands.add_parser("alias-hmac")
    alias.add_argument("--subject-id", required=True)
    alias.add_argument("--external-account-id", required=True)
    alias.add_argument("--internal-account-ref", required=True)
    alias.add_argument("--corporate-organization-ref", required=True,
                       help="organización org_ del vínculo corporativo en ContextoActor")
    alias.add_argument("--database", required=True)
    alias.add_argument("--admin-user", required=True)
    alias.add_argument("--pg-container")
    alias.add_argument("--container-engine", choices=("docker", "podman"), default="docker")
    env = commands.add_parser("runtime-env")
    env.add_argument("--listen", required=True)
    env.add_argument("--allowed-cidrs", required=True)
    env.add_argument("--server-name", required=True)
    env.add_argument("--tls-root", required=True)
    env.add_argument("--audience", required=True)
    env.add_argument("--issuer", required=True)
    args = parser.parse_args()
    os.umask(0o077)
    root = private_dir(args.material_dir)
    with (root / ".aprovisionar.lock").open("a+b") as lock:
        os.chmod(root / ".aprovisionar.lock", 0o600)
        fcntl.flock(lock, fcntl.LOCK_EX)
        if args.command == "init-ca":
            init_ca(root, args.server_name)
        elif args.command == "create-csr":
            create_csr(root, args.subject_id, args.token_url, Path(args.pkcs11_module).absolute(),
                       Path(args.p11tool).absolute(), Path(args.certtool).absolute(), Path(args.pin_file).absolute())
        elif args.command == "issue-person":
            issue_person(root, Path(args.csr).absolute(), args.subject_id, args.token_url,
                         Path(args.pkcs11_module).absolute(), Path(args.p11tool).absolute(),
                         Path(args.pin_file).absolute())
        elif args.command == "register-person":
            register_person(root, args.subject_id, args.account_id)
        elif args.command == "register-context":
            register_context(root, args.account_id, args.profile_id, args.organization_ref,
                             args.unit_ref)
        elif args.command == "revoke-person":
            revoke_person(root, args.subject_id)
        elif args.command == "refresh-crl":
            refresh_crl(root)
        elif args.command == "roles":
            roles(root, args)
        elif args.command == "identity-roles":
            roles(root, args, identity=True)
        elif args.command == "v3-roles":
            roles(root, args, v3=True)
        elif args.command == "policy":
            policy(root, args)
        elif args.command == "hmac-token":
            hmac_token(root, args)
        elif args.command == "alias-hmac":
            registrar_alias_hmac(root, args)
        elif args.command == "runtime-env":
            runtime_env(root, args)


if __name__ == "__main__":
    try:
        main()
    except (OSError, ValueError, KeyError, RuntimeError, json.JSONDecodeError) as error:
        print(f"vec-interno: {error}", file=sys.stderr)
        sys.exit(1)
