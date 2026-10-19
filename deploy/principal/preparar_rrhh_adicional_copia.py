#!/usr/bin/env python3
"""Prepara en la copia local una segunda identidad técnica de RRHH."""

import argparse
import hashlib
import json
import os
from pathlib import Path
import re
import secrets
import stat
import subprocess
import tempfile


ARCHIVOS = (
    "mtls/tecnico-rrhh-adicional.crt",
    "mtls/tecnico-rrhh-adicional.key",
    "mtls/tecnico-rrhh-adicional.p12",
    "mtls/tecnico-rrhh-adicional.p12.password",
    "identidad/tecnico-rrhh-adicional-identidad.json",
    "identidad/tecnico-rrhh-adicional.json",
)


def privado(ruta: Path, directorio=False):
    info = ruta.lstat()
    if (info.st_uid != os.getuid() or info.st_mode & 0o077
            or (not stat.S_ISDIR(info.st_mode) if directorio
                else not stat.S_ISREG(info.st_mode) or info.st_nlink != 1)):
        raise ValueError("La ruta del material debe ser propia y privada")


def leer_json(ruta: Path):
    privado(ruta)
    def unico(pares):
        objeto = {}
        for clave, valor in pares:
            if clave in objeto:
                raise ValueError("Hay una clave JSON repetida")
            objeto[clave] = valor
        return objeto
    dato = json.loads(ruta.read_text(encoding="utf-8"), object_pairs_hook=unico)
    if not isinstance(dato, dict):
        raise ValueError("Se esperaba un objeto JSON")
    return dato


def ejecutar(*args, entrada=None):
    proceso = subprocess.run(["openssl", *map(str, args)], input=entrada,
                            stdout=subprocess.PIPE, stderr=subprocess.PIPE,
                            timeout=20, check=False)
    if proceso.returncode:
        raise ValueError("OpenSSL no pudo verificar o preparar el certificado")
    return proceso.stdout


def huella(ruta):
    return hashlib.sha256(ejecutar("x509", "-in", ruta, "-outform", "DER")).hexdigest()


def cotejar_clave(certificado, clave):
    if ejecutar("pkey", "-in", clave, "-pubout") != ejecutar("x509", "-in", certificado, "-pubkey", "-noout"):
        raise ValueError("La clave no corresponde al certificado")


def referencia(prefijo, material):
    return prefijo + hashlib.sha256(("vec.ct.alta.desarrollo.v1\0" + material).encode()).hexdigest()[:32]


def cotejar_origen(material: Path, sujeto: str):
    privado(material, True)
    privado(material / "ca", True)
    privado(material / "mtls", True)
    privado(material / "identidad", True)
    for ruta in ("ca/ca.crt", "ca/ca.key", "mtls/cliente.crt",
                 "mtls/cliente.key", "identidad/identidad.json", "manifiesto.json"):
        privado(material / ruta)
    raiz = material / "ca/ca.crt"
    ejecutar("verify", "-purpose", "sslclient", "-CAfile", raiz, material / "mtls/cliente.crt")
    original = leer_json(material / "identidad/identidad.json")
    manifiesto = leer_json(material / "manifiesto.json")
    original_huella = huella(material / "mtls/cliente.crt")
    if (manifiesto.get("perfil") != "desarrollo" or manifiesto.get("autoridad") != "no_autoritativo"
            or manifiesto.get("migrable_a_produccion") is not False
            or manifiesto.get("huella_ca_sha256", "").lower() != huella(raiz)
            or manifiesto.get("huella_cliente_sha256", "").lower() != original_huella
            or original.get("certificate_sha256") != original_huella
            or original.get("roles") != ["tecnico_rrhh"]
            or original.get("subject") == sujeto):
        raise ValueError("La identidad original o la CA no coinciden con el material de desarrollo")
    cotejar_clave(raiz, material / "ca/ca.key")
    for identidad in (material / "identidad").glob("*.json"):
        dato = leer_json(identidad)
        if dato.get("subject") == sujeto:
            raise ValueError("El sujeto ya existe en otra identidad")
    return original, original_huella


def preparar(material: Path, sujeto: str, nombre: str):
    if not re.fullmatch(r"desarrollo:[a-z0-9][a-z0-9:._-]{2,110}", sujeto):
        raise ValueError("El sujeto debe ser una referencia sintética de desarrollo")
    if not isinstance(nombre, str) or not 3 <= len(nombre) <= 120 or any(ord(c) < 32 for c in nombre):
        raise ValueError("El nombre de la persona es inválido")
    material = material.absolute()
    if material.resolve(strict=True) != material:
        raise ValueError("La ruta del material no debe contener enlaces")
    if any((carpeta / ".git").exists() for carpeta in (material, *material.parents)):
        raise ValueError("El material privado debe estar fuera de Git")
    original, original_huella = cotejar_origen(material, sujeto)
    existentes = [material / nombre for nombre in ARCHIVOS]
    if any(ruta.exists() or ruta.is_symlink() for ruta in existentes):
        raise ValueError("El segundo RRHH ya tiene material: no se sobrescribe")
    with tempfile.TemporaryDirectory(prefix=".rrhh-adicional-", dir=material) as temporal:
        t = Path(temporal)
        key, csr, ext, crt = (t / nombre for nombre in ("hoja.key", "hoja.csr", "hoja.ext", "hoja.crt"))
        ejecutar("genpkey", "-algorithm", "EC", "-pkeyopt", "ec_paramgen_curve:P-256", "-out", key)
        ejecutar("req", "-new", "-sha256", "-key", key, "-subj",
                "/CN=tecnico-rrhh-adicional/O=VEC Desarrollo/OU=NO AUTORITATIVO", "-out", csr)
        ext.write_text("basicConstraints=critical,CA:FALSE\nkeyUsage=critical,digitalSignature\n"
                       "extendedKeyUsage=clientAuth\nsubjectAltName=URI:urn:vec:" + sujeto + "\n"
                       "subjectKeyIdentifier=hash\nauthorityKeyIdentifier=keyid,issuer\n", encoding="utf-8")
        ejecutar("x509", "-req", "-sha256", "-days", "90", "-in", csr,
                "-CA", material / "ca/ca.crt", "-CAkey", material / "ca/ca.key",
                "-set_serial", "0x" + secrets.token_hex(19), "-extfile", ext, "-out", crt)
        ejecutar("verify", "-purpose", "sslclient", "-CAfile", material / "ca/ca.crt", crt)
        cotejar_clave(crt, key)
        nueva_huella = huella(crt)
        if nueva_huella == original_huella:
            raise ValueError("El certificado nuevo coincide con el original")
        for existente in (material / "mtls").glob("*.crt"):
            privado(existente)
            if nueva_huella == huella(existente):
                raise ValueError("El certificado nuevo ya existe")
        password = t / "hoja.p12.password"
        password.write_text(secrets.token_hex(32) + "\n", encoding="ascii")
        p12 = t / "hoja.p12"
        ejecutar("pkcs12", "-export", "-out", p12, "-inkey", key, "-in", crt,
                "-certfile", material / "ca/ca.crt", "-name", "tecnico-rrhh-adicional",
                "-passout", "file:" + str(password))
        ejecutar("pkcs12", "-in", p12, "-passin", "file:" + str(password), "-noout")
        base = sujeto + "\0" + nueva_huella
        cuenta = referencia("cta_", base + "\0cuenta")
        persona = referencia("per_", base + "\0persona")
        perfil = referencia("prf_", base + "\0perfil")
        base_original = original["subject"] + "\0" + original_huella
        if cuenta == referencia("cta_", base_original + "\0cuenta") or persona == referencia("per_", base_original + "\0persona"):
            raise ValueError("La persona o la cuenta coinciden con el RRHH original")
        identidad = {"version": 1, "autoridad": "no_autoritativo", "certificate_sha256": nueva_huella,
                     "subject": sujeto, "display_name": nombre, "roles": ["tecnico_rrhh"]}
        manifiesto = {"version": 1, "autoridad": "no_autoritativo",
                      "certificado": "mtls/tecnico-rrhh-adicional.crt",
                      "identidad": "identidad/tecnico-rrhh-adicional-identidad.json",
                      "sujeto": sujeto, "cuenta_ref": cuenta, "persona_ref": persona, "perfil_ref": perfil}
        identidad_ruta = t / "identidad.json"
        manifiesto_ruta = t / "manifiesto.json"
        for ruta, dato in ((identidad_ruta, identidad), (manifiesto_ruta, manifiesto)):
            ruta.write_text(json.dumps(dato, ensure_ascii=False, separators=(",", ":")) + "\n", encoding="utf-8")
        fuente = (crt, key, p12, password, identidad_ruta, manifiesto_ruta)
        creados = []
        try:
            for origen, destino in zip(fuente, existentes):
                origen.chmod(0o600)
                os.link(origen, destino)
                creados.append(destino)
        except OSError:
            for destino in creados:
                destino.unlink()
            raise
    return manifiesto


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--material-dir", required=True, type=Path)
    parser.add_argument("--sujeto", required=True)
    parser.add_argument("--nombre", required=True)
    args = parser.parse_args()
    preparar(args.material_dir, args.sujeto, args.nombre)
    print("Segunda identidad técnica RRHH preparada en material privado; publicación administrativa pendiente.")


if __name__ == "__main__":
    main()
