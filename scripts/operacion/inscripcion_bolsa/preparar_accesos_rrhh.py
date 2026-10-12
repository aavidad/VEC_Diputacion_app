#!/usr/bin/env python3
"""Compone el material RRHH de inscripción desde fuentes privadas existentes."""

import argparse
import json
import os
from pathlib import Path
import re
import tempfile

DSNS = (
    "dsn_registro_identidad", "dsn_revalidacion_identidad", "dsn_contexto",
    "dsn_fuente_autorizacion", "dsn_registro_autorizacion", "dsn_motivos",
)
CUENTA = ("certificado_sha256", "sujeto", "cuenta_ref", "perfil_ref")
MOTIVO = ("catalogo_id", "catalogo_version", "catalogo_huella_sha256", "entrada_clave")


def privado(ruta: Path, directorio=False):
    info = ruta.lstat()
    if ruta.is_symlink() or info.st_uid != os.getuid() or info.st_mode & 0o077:
        raise ValueError("la ruta debe ser propia y privada")
    if directorio != ruta.is_dir():
        raise ValueError("tipo de ruta inesperado")


def leer(ruta: Path):
    privado(ruta)
    def claves_unicas(pares):
        dato = {}
        for clave, valor in pares:
            if clave in dato:
                raise ValueError("clave JSON repetida")
            dato[clave] = valor
        return dato
    with ruta.open(encoding="utf-8") as entrada:
        dato = json.load(entrada, object_pairs_hook=claves_unicas)
    if not isinstance(dato, dict):
        raise ValueError("se esperaba un objeto JSON")
    return dato


def preparar(identidad, conexiones, motivos):
    cuentas = identidad.get("cuentas")
    if not isinstance(cuentas, list) or not 1 <= len(cuentas) <= 64:
        raise ValueError("faltan cuentas RRHH nominales")
    vistos = set()
    for cuenta in cuentas:
        if not isinstance(cuenta, dict) or set(cuenta) != set(CUENTA):
            raise ValueError("cuenta RRHH incompleta")
        if not re.fullmatch(r"[0-9a-f]{64}", cuenta["certificado_sha256"]):
            raise ValueError("huella de certificado inválida")
        if any(not isinstance(cuenta[k], str) or not cuenta[k] for k in CUENTA):
            raise ValueError("referencia de cuenta ausente")
        if cuenta["certificado_sha256"] in vistos:
            raise ValueError("certificado repetido")
        vistos.add(cuenta["certificado_sha256"])
    salida = {"version": 1, "autoridad": "no_autoritativo", "cuentas": cuentas}
    for clave in DSNS:
        valor = conexiones.get(clave)
        if not isinstance(valor, str) or not valor.strip():
            raise ValueError("falta una conexión privada")
        salida[clave] = valor
    seleccion = {k: motivos.get(k) for k in ("decidir", "incorporar")}
    for motivo in seleccion.values():
        if not isinstance(motivo, dict) or set(motivo) != set(MOTIVO):
            raise ValueError("referencia de motivo incompleta")
        if type(motivo["catalogo_version"]) is not int or motivo["catalogo_version"] < 1:
            raise ValueError("versión de motivo inválida")
        if not re.fullmatch(r"[0-9a-f]{64}", motivo["catalogo_huella_sha256"]):
            raise ValueError("huella de motivo inválida")
        if any(not isinstance(motivo[k], str) or not motivo[k] for k in ("catalogo_id", "entrada_clave")):
            raise ValueError("clave de motivo inválida")
    if seleccion["decidir"]["catalogo_id"] != seleccion["incorporar"]["catalogo_id"] or seleccion["decidir"] == seleccion["incorporar"]:
        raise ValueError("los motivos deben ser distintos y del mismo catálogo")
    salida["motivos"] = seleccion
    return salida


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--material-dir", type=Path, required=True)
    parser.add_argument("--identidad", type=Path, required=True)
    parser.add_argument("--conexiones", type=Path, required=True)
    parser.add_argument("--motivos", type=Path, required=True)
    args = parser.parse_args()
    privado(args.material_dir, directorio=True)
    destino_dir = args.material_dir / "identidad"
    privado(destino_dir, directorio=True)
    destino = destino_dir / "inscripcion-bolsa-rrhh.json"
    contenido = (json.dumps(preparar(leer(args.identidad), leer(args.conexiones), leer(args.motivos)),
                           ensure_ascii=False, separators=(",", ":")) + "\n").encode()
    if destino.exists():
        privado(destino)
        if destino.read_bytes() != contenido:
            raise ValueError("el material existente difiere; revisar fuentes antes de sustituir")
        print("Material RRHH privado ya preparado.")
        return
    descriptor, temporal = tempfile.mkstemp(prefix=".inscripcion-rrhh-", dir=destino_dir)
    try:
        os.fchmod(descriptor, 0o600)
        with os.fdopen(descriptor, "wb") as salida:
            salida.write(contenido)
            salida.flush()
            os.fsync(salida.fileno())
        os.link(temporal, destino)
    finally:
        os.unlink(temporal)
    print("Material RRHH privado preparado.")


if __name__ == "__main__":
    main()
