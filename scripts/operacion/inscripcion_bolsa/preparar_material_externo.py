#!/usr/bin/env python3
"""Prepara la base privada de un portal externo; no provisiona identidad ni permisos."""

import argparse
import hashlib
import json
import os
from pathlib import Path
import re
import secrets
import stat
import subprocess
import sys
import tempfile


COPIAS = (
    "ca/ca.crt", "mtls/candidato.crt", "identidad/candidato.json",
    "idempotencia/configuracion.json",
)
ESQUEMA_HMAC = "vec.bolsa.convocatoria.idempotencia-hmac.desarrollo.v1"


class MaterialInvalido(Exception):
    pass


def rechazar():
    raise MaterialInvalido("material_externo_invalido")


def ruta_canonica(valor):
    ruta = Path(valor)
    if not ruta.is_absolute() or str(ruta) != os.path.abspath(valor):
        rechazar()
    for parte in (ruta, *ruta.parents):
        if parte.exists() and parte.is_symlink():
            rechazar()
    if ruta.resolve(strict=False) != ruta:
        rechazar()
    return ruta


def directorio_privado(ruta):
    try:
        dato = ruta.lstat()
    except OSError:
        rechazar()
    if not stat.S_ISDIR(dato.st_mode) or dato.st_uid != os.getuid() or dato.st_mode & 0o077:
        rechazar()


def fichero_privado(ruta, limite=65536):
    try:
        dato = ruta.lstat()
    except OSError:
        rechazar()
    if (not stat.S_ISREG(dato.st_mode) or dato.st_nlink != 1 or dato.st_uid != os.getuid()
            or dato.st_mode & 0o077 or not 0 < dato.st_size <= limite):
        rechazar()
    return ruta.read_bytes()


def json_privado(ruta):
    def unico(pares):
        resultado = {}
        for clave, valor in pares:
            if clave in resultado:
                rechazar()
            resultado[clave] = valor
        return resultado
    try:
        valor = json.loads(fichero_privado(ruta), object_pairs_hook=unico)
    except (UnicodeError, ValueError):
        rechazar()
    if not isinstance(valor, dict):
        rechazar()
    return valor


def ejecutar(*argumentos):
    try:
        hecho = subprocess.run(argumentos, stdout=subprocess.PIPE, stderr=subprocess.DEVNULL,
                               timeout=20, check=False)
    except (OSError, subprocess.TimeoutExpired):
        rechazar()
    if hecho.returncode != 0:
        rechazar()
    return hecho.stdout


def huella_der(certificado):
    return hashlib.sha256(ejecutar("openssl", "x509", "-in", str(certificado), "-outform", "DER")).hexdigest()


def validar_identidad(raiz, externo=False):
    ca = raiz / "ca/ca.crt"
    cert = raiz / "mtls/candidato.crt"
    identidad = json_privado(raiz / "identidad/candidato.json")
    candidato = json_privado(raiz / "identidad/bolsa-candidato.json")
    fichero_privado(ca)
    fichero_privado(cert)
    if (identidad.get("version"), identidad.get("autoridad"), identidad.get("roles")) != (1, "no_autoritativo", ["candidato_bolsa"]):
        rechazar()
    if identidad.get("certificate_sha256") != huella_der(cert) or not identidad.get("subject"):
        rechazar()
    if (candidato.get("version"), candidato.get("autoridad"), candidato.get("sujeto"),
            candidato.get("certificado"), candidato.get("identidad")) != (
            1, "no_autoritativo", identidad["subject"], "mtls/candidato.crt", "identidad/candidato.json"):
        rechazar()
    for clave, prefijo in (("cuenta_ref", "cta_"), ("persona_ref", "per_"),
                          ("perfil_ref", "prf_"), ("candidato_ref", "can_")):
        if not isinstance(candidato.get(clave), str) or not re.fullmatch(prefijo + r"[A-Za-z0-9_-]{22,128}", candidato[clave]):
            rechazar()
    ejecutar("openssl", "verify", "-purpose", "sslclient", "-CAfile", str(ca), str(cert))
    if externo and (raiz / "mtls/candidato.key").exists():
        rechazar()
    return identidad, candidato


def generaciones(raiz):
    datos = json_privado(raiz / "idempotencia/configuracion.json")
    if (datos.get("version"), datos.get("esquema"), datos.get("autoridad"),
            datos.get("version_esquema_hmac")) != (1, ESQUEMA_HMAC, "no_autoritativo", 2):
        rechazar()
    grupos = datos.get("generaciones")
    if not isinstance(grupos, list) or not 2 <= len(grupos) <= 4:
        rechazar()
    anterior = 1 << 32
    for g in grupos:
        if not isinstance(g, dict) or set(g) != {"generacion", "referencia_localizador", "referencia_huella_solicitud"}:
            rechazar()
        n = g["generacion"]
        if not isinstance(n, int) or not 0 < n < anterior:
            rechazar()
        anterior = n
        if (g["referencia_localizador"] != f"clave:hmac:convocatorias:localizador:desarrollo:v{n}"
                or g["referencia_huella_solicitud"] != f"clave:hmac:convocatorias:huella:desarrollo:v{n}"):
            rechazar()
    return [g["generacion"] for g in grupos]


def escribir(raiz, relativa, contenido):
    destino = raiz / relativa
    destino.parent.mkdir(mode=0o700, parents=True, exist_ok=True)
    os.chmod(destino.parent, 0o700)
    with destino.open("xb") as salida:
        salida.write(contenido)
    os.chmod(destino, 0o600)


def perfil_externo_valido(perfil, interno):
    return isinstance(perfil, str) and re.fullmatch(r"prf_[A-Za-z0-9_-]{22,128}", perfil) is not None and perfil != interno


# ContextoActor segrega las cuentas: la externa es propia y distinta de la
# interna; persona, candidatura, sujeto y certificado son los mismos.
def cuenta_externa_valida(cuenta, interna):
    return isinstance(cuenta, str) and re.fullmatch(r"cta_[A-Za-z0-9_-]{22,128}", cuenta) is not None and cuenta != interna


def verificar(interno, externo, perfil_externo, cuenta_externa):
    directorio_privado(interno)
    directorio_privado(externo)
    if json_privado(externo / "portal-proceso.json") != {"version": 1, "portal": "externo"}:
        rechazar()
    for relativa in COPIAS:
        if fichero_privado(interno / relativa) != fichero_privado(externo / relativa):
            rechazar()
    identidad_interna, candidato_interno = validar_identidad(interno)
    identidad_externa, candidato_externo = validar_identidad(externo, True)
    if not perfil_externo_valido(perfil_externo, candidato_interno["perfil_ref"]) or \
            not cuenta_externa_valida(cuenta_externa, candidato_interno["cuenta_ref"]):
        rechazar()
    esperado = dict(candidato_interno)
    esperado["perfil_ref"] = perfil_externo
    esperado["cuenta_ref"] = cuenta_externa
    if identidad_interna != identidad_externa or candidato_externo != esperado:
        rechazar()
    ca = externo / "ca/ca.crt"
    servidor = externo / "tls/servidor.crt"
    clave = externo / "tls/servidor.key"
    fichero_privado(servidor)
    fichero_privado(clave)
    ejecutar("openssl", "verify", "-purpose", "sslserver", "-verify_hostname", "localhost",
            "-CAfile", str(ca), str(servidor))
    pub_cert = ejecutar("openssl", "x509", "-in", str(servidor), "-pubkey", "-noout")
    pub_clave = ejecutar("openssl", "pkey", "-in", str(clave), "-pubout")
    if pub_cert != pub_clave or fichero_privado(interno / "tls/servidor.key") == fichero_privado(clave):
        rechazar()
    manifiesto = json_privado(externo / "manifiesto.json")
    if (manifiesto.get("perfil"), manifiesto.get("autoridad"), manifiesto.get("migrable_a_produccion"),
            manifiesto.get("huella_ca_sha256"), manifiesto.get("huella_servidor_sha256")) != (
            "desarrollo", "no_autoritativo", False, huella_der(ca), huella_der(servidor)):
        rechazar()
    grupos = generaciones(externo)
    internos = generaciones(interno)
    if grupos != internos:
        rechazar()
    huellas_internas = set()
    huellas_externas = set()
    for n in grupos:
        for nombre in ("localizador", "huella-solicitud"):
            rel = f"idempotencia/g{n}-{nombre}.bin"
            propio = fichero_privado(externo / rel)
            interno_clave = fichero_privado(interno / rel)
            if len(propio) != 32 or len(interno_clave) != 32:
                rechazar()
            huellas_externas.add(hashlib.sha256(propio).digest())
            huellas_internas.add(hashlib.sha256(interno_clave).digest())
    if len(huellas_externas) != 2 * len(grupos) or huellas_externas & huellas_internas:
        rechazar()
    permitidos = set(COPIAS) | {"identidad/bolsa-candidato.json", "portal-proceso.json", "manifiesto.json", "tls/servidor.crt", "tls/servidor.key"}
    permitidos.update(f"idempotencia/g{n}-{nombre}.bin" for n in grupos for nombre in ("localizador", "huella-solicitud"))
    for ruta in externo.rglob("*"):
        relativa = ruta.relative_to(externo).as_posix()
        if ruta.is_symlink() or (not ruta.is_dir() and not ruta.is_file()):
            rechazar()
        if ruta.is_dir():
            directorio_privado(ruta)
            continue
        if relativa not in permitidos and not re.fullmatch(r"externo/v3/(inventario[.]json|raiz[.]bin|[a-z_]+-[0-9]{1,2}[.]bin)", relativa):
            rechazar()
        fichero_privado(ruta)


def preparar(interno, externo, perfil_externo, cuenta_externa):
    directorio_privado(interno)
    directorio_privado(externo.parent)
    _, candidato_interno = validar_identidad(interno)
    if not perfil_externo_valido(perfil_externo, candidato_interno["perfil_ref"]) or \
            not cuenta_externa_valida(cuenta_externa, candidato_interno["cuenta_ref"]):
        rechazar()
    grupos = generaciones(interno)
    for relativa in COPIAS:
        fichero_privado(interno / relativa)
    fichero_privado(interno / "ca/ca.key")
    if externo.exists():
        verificar(interno, externo, perfil_externo, cuenta_externa)
        return
    with tempfile.TemporaryDirectory(prefix=".portal-externo-", dir=externo.parent) as temporal:
        trabajo = Path(temporal)
        os.chmod(trabajo, 0o700)
        for relativa in COPIAS:
            escribir(trabajo, relativa, fichero_privado(interno / relativa))
        candidato_externo = dict(candidato_interno)
        candidato_externo["perfil_ref"] = perfil_externo
        candidato_externo["cuenta_ref"] = cuenta_externa
        escribir(trabajo, "identidad/bolsa-candidato.json",
                (json.dumps(candidato_externo, sort_keys=True, separators=(",", ":")) + "\n").encode())
        escribir(trabajo, "portal-proceso.json", b'{"version":1,"portal":"externo"}\n')
        clave = trabajo / "tls/servidor.key"
        clave.parent.mkdir(mode=0o700, parents=True)
        ejecutar("openssl", "genpkey", "-algorithm", "EC", "-pkeyopt", "ec_paramgen_curve:P-256",
                "-out", str(clave))
        os.chmod(clave, 0o600)
        csr = trabajo / "tls/servidor.csr"
        ejecutar("openssl", "req", "-new", "-sha256", "-key", str(clave),
                "-subj", "/CN=localhost/O=VEC Desarrollo/OU=NO AUTORITATIVO", "-out", str(csr))
        ext = trabajo / "tls/servidor.ext"
        ext.write_text("basicConstraints=critical,CA:FALSE\nkeyUsage=critical,digitalSignature,keyEncipherment\n"
                       "extendedKeyUsage=serverAuth\nsubjectAltName=DNS:localhost,IP:127.0.0.1,IP:::1\n"
                       "subjectKeyIdentifier=hash\nauthorityKeyIdentifier=keyid,issuer\n")
        cert = trabajo / "tls/servidor.crt"
        ejecutar("openssl", "x509", "-req", "-sha256", "-days", "397", "-in", str(csr),
                "-CA", str(interno / "ca/ca.crt"), "-CAkey", str(interno / "ca/ca.key"),
                "-set_serial", "0x" + secrets.token_hex(16), "-extfile", str(ext), "-out", str(cert))
        os.chmod(cert, 0o600)
        csr.unlink()
        ext.unlink()
        for n in grupos:
            for nombre in ("localizador", "huella-solicitud"):
                escribir(trabajo, f"idempotencia/g{n}-{nombre}.bin", secrets.token_bytes(32))
        manifiesto = {"version": 4, "perfil": "desarrollo", "autoridad": "no_autoritativo",
                      "migrable_a_produccion": False, "huella_ca_sha256": huella_der(trabajo / "ca/ca.crt"),
                      "huella_servidor_sha256": huella_der(cert),
                      "huella_cliente_sha256": huella_der(trabajo / "mtls/candidato.crt")}
        escribir(trabajo, "manifiesto.json", (json.dumps(manifiesto, sort_keys=True, separators=(",", ":")) + "\n").encode())
        verificar(interno, trabajo, perfil_externo, cuenta_externa)
        externo.mkdir(mode=0o700)
        for ruta in trabajo.rglob("*"):
            if ruta.is_dir():
                (externo / ruta.relative_to(trabajo)).mkdir(mode=0o700, parents=True, exist_ok=True)
            else:
                escribir(externo, ruta.relative_to(trabajo), fichero_privado(ruta))
        verificar(interno, externo, perfil_externo, cuenta_externa)


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--material-interno", required=True)
    parser.add_argument("--material-externo", required=True)
    parser.add_argument("--perfil-externo-ref", required=True)
    parser.add_argument("--cuenta-externa-ref", required=True)
    modo = parser.add_mutually_exclusive_group(required=True)
    modo.add_argument("--comprobar", action="store_true")
    modo.add_argument("--aplicar", action="store_true")
    args = parser.parse_args()
    try:
        interno = ruta_canonica(args.material_interno)
        externo = ruta_canonica(args.material_externo)
        if interno == externo or interno in externo.parents or externo in interno.parents:
            rechazar()
        repo = Path(__file__).resolve().parents[3]
        if repo == interno or repo in interno.parents or repo == externo or repo in externo.parents:
            rechazar()
        if args.aplicar:
            preparar(interno, externo, args.perfil_externo_ref, args.cuenta_externa_ref)
        else:
            verificar(interno, externo, args.perfil_externo_ref, args.cuenta_externa_ref)
    except (MaterialInvalido, OSError, ValueError, KeyError, TypeError):
        print("material_externo_invalido", file=sys.stderr)
        return 1
    print("material_externo_comprobado" if args.comprobar else "material_externo_preparado")
    return 0


if __name__ == "__main__":
    sys.exit(main())
