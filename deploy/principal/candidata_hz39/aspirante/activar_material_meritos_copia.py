#!/usr/bin/env python3
"""Activa solo el material externo aprobado de Usuarios y cartera."""

import argparse
import json
import os
from pathlib import Path
import re


def exigir(condicion, clave):
    if not condicion:
        raise ValueError(clave)


def leer(ruta):
    exigir(ruta.is_file() and not ruta.is_symlink() and not ruta.stat().st_mode & 0o077,
           "material_no_privado")
    return json.loads(ruta.read_text())


def instalar(ruta, valor, aplicar):
    if ruta.exists():
        exigir(leer(ruta) == valor, "material_externo_divergente")
        return
    exigir(aplicar, "material_externo_pendiente")
    with ruta.open("x") as salida:
        json.dump(valor, salida, ensure_ascii=False, separators=(",", ":"))
        salida.write("\n")
    ruta.chmod(0o600)


def main():
    p = argparse.ArgumentParser(description=__doc__)
    p.add_argument("--ensayo", type=Path, required=True)
    p.add_argument("--aplicar", action="store_true")
    a = p.parse_args()
    exigir(a.ensayo.is_absolute() and Path(__file__).resolve().parents[4] not in a.ensayo.parents and
           a.ensayo.is_dir() and not a.ensayo.stat().st_mode & 0o077,
           "solo_ensayo_privado")
    actos = a.ensayo / "incorporacion/actos"
    externo = a.ensayo / "externo/identidad"
    exigir(externo.is_dir() and not externo.stat().st_mode & 0o077, "destino_no_privado")
    usuarios = leer(actos / "usuarios-preferencias-externa.preparado.json")
    cuentas = leer(actos / "meritos-cuentas.preparado.json")
    motivos = leer(actos / "motivos-meritos.preparado.json")
    manifiesto = {"version": 1, "motivo_retirada": motivos["motivo_retirada"],
                  "motivo_declaracion": motivos["motivo_declaracion"],
                  "cuentas": cuentas["cuentas"]}
    exigir(len(usuarios["cuentas"]) == 1 and len(manifiesto["cuentas"]) == 1 and
           usuarios["cuentas"][0]["certificado_sha256"] ==
           manifiesto["cuentas"][0]["certificado_sha256"] and
           usuarios["cuentas"][0]["cuenta_ref"] != manifiesto["cuentas"][0]["cuenta_ref"] and
           usuarios["cuentas"][0]["perfil_ref"] != manifiesto["cuentas"][0]["perfil_ref"] and
           manifiesto["motivo_retirada"]["catalogo_id"] == "motivos_autorizacion_meritos_cartera" and
           manifiesto["motivo_declaracion"]["catalogo_id"] == "motivos_meritos_declaracion",
           "material_meritos_divergente")
    inventario = leer(a.ensayo / "externo/externo/v3/inventario.json")
    exigir(len(inventario["consumidores"].get("meritos_cartera", [])) == 2,
           "material_v3_meritos_ausente")
    instalar(externo / "usuarios-preferencias-externa.json", usuarios, a.aplicar)
    instalar(externo / "meritos-cartera-externa.json", manifiesto, a.aplicar)
    print("material_usuarios_meritos=activo")


if __name__ == "__main__":
    try:
        main()
    except (OSError, ValueError, KeyError, TypeError, json.JSONDecodeError) as error:
        raise SystemExit("material_usuarios_meritos=error clave=" + str(error).split(":", 1)[0]) from None
