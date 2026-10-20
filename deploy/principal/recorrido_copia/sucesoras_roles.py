#!/usr/bin/env python3
"""Sucesoras de los ocho cargos existentes, desde sus huellas publicadas."""
import argparse
import json
import os
import re
from pathlib import Path

ROLES = ("tecnico", "jefatura", "diputacion", "secretaria")


def leer_privado(path):
    p = Path(path)
    if not p.is_absolute() or not p.is_file() or p.is_symlink() or p.stat().st_mode & 0o077:
        raise ValueError("archivo privado 0600 requerido")
    return json.loads(p.read_bytes())


def guardar(path, value):
    with path.open("xb") as f:
        f.write(json.dumps(value, ensure_ascii=False, separators=(",", ":")).encode())
        f.flush()
        os.fsync(f.fileno())
    path.chmod(0o600)


def main():
    ap = argparse.ArgumentParser()
    ap.add_argument("--plantilla", required=True)
    ap.add_argument("--roles-v1", required=True)
    ap.add_argument("--salida", required=True)
    a = ap.parse_args()
    os.umask(0o077)
    plantilla = json.loads(Path(a.plantilla).read_bytes())
    huellas = leer_privado(a.roles_v1)
    destino = Path(a.salida)
    if not destino.is_absolute() or not destino.is_dir() or destino.is_symlink() or destino.stat().st_mode & 0o077:
        raise ValueError("salida privada 0700 requerida")
    if plantilla.get("esquema") != "vec.admin.cargos-firma.cargos.v2" or len(plantilla.get("cargos", [])) != 16:
        raise ValueError("plantilla de 16 cargos inválida")
    if not isinstance(huellas, dict) or len(huellas) != 8:
        raise ValueError("ocho huellas v1 de esta copia requeridas")
    for tipo in ("firma", "consulta"):
        nuevos = []
        for papel in ROLES:
            rol_id = f"ct_cargo_{papel}_{tipo}_dev"
            encontrados = [x for x in plantilla["cargos"] if x.get("rol_id") == rol_id]
            anterior = huellas.get(rol_id, "")
            if len(encontrados) != 1 or not re.fullmatch(r"[0-9a-f]{64}", anterior):
                raise ValueError(f"cargo v1 sin preimagen única: {rol_id}")
            cargo = dict(encontrados[0])
            if cargo.get("version") != 1:
                raise ValueError("plantilla inicial cambiada")
            cargo["version"] = 2
            cargo["version_anterior_sha256"] = anterior
            operaciones = list(cargo["operaciones_v2"])
            if tipo == "firma":
                if "recuperar_r5" in operaciones:
                    raise ValueError("operación ya incluida en plantilla")
                operaciones.append("recuperar_r5")
            else:
                if operaciones.count("consultar_detalle") != 1 or "consultar_detalle_selectores" in operaciones:
                    raise ValueError("operación de consulta inesperada")
                operaciones[operaciones.index("consultar_detalle")] = "consultar_detalle_selectores"
            cargo["operaciones_v2"] = operaciones
            nuevos.append(cargo)
        guardar(destino / f"cargos-{tipo}-v2.json", {"esquema": plantilla["esquema"], "cargos": nuevos})
    print("Dos fuentes de cuatro cargos v2 preparadas desde las preimágenes publicadas")


if __name__ == "__main__":
    main()
