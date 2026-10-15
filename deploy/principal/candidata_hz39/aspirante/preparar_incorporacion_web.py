#!/usr/bin/env python3
"""Compone el montaje web desde una lista positiva de archivos runtime."""

import argparse
import json
import os
from pathlib import Path
import stat


MERITOS_RUNTIME = (
    "operacion.dsn", "cartera.dsn", "frontera.dsn", "fuente_autorizacion.dsn",
    "registro_autorizacion.dsn", "motivos_autorizacion.dsn", "contexto.dsn",
)
GENERADOS_RUNTIME = ("conexiones_referencia.sh", "arrancar_combinado.sh")


def exigir(condicion, clave):
    if not condicion:
        raise ValueError(clave)


def comprobar(ruta, directorio=False):
    dato = ruta.lstat()
    exigir(dato.st_uid == os.getuid() and not ruta.is_symlink(), "ruta_no_propia")
    exigir(stat.S_ISDIR(dato.st_mode) if directorio else stat.S_ISREG(dato.st_mode),
           "tipo_invalido")


def copiar(raiz, destino, relativo, aplicar):
    origen = raiz / relativo
    comprobar(origen)
    exigir(not origen.stat().st_mode & 0o077, "origen_no_privado")
    salida = destino / relativo
    contenido = origen.read_bytes()
    if salida.exists():
        comprobar(salida)
        exigir(not salida.stat().st_mode & 0o077, "web_divergente")
        if salida.read_bytes() == contenido:
            return
        raise ValueError("web_divergente")
    exigir(aplicar, "web_pendiente")
    salida.parent.mkdir(mode=0o700, parents=True, exist_ok=True)
    salida.parent.chmod(0o700)
    salida.write_bytes(contenido)
    salida.chmod(0o700 if salida.name == "arrancar_combinado.sh" else 0o600)


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--respaldo", type=Path, required=True)
    parser.add_argument("--ensayo", type=Path, required=True)
    parser.add_argument("--aplicar", action="store_true")
    args = parser.parse_args()
    respaldo, ensayo = args.respaldo, args.ensayo
    exigir(respaldo.is_absolute() and ensayo.is_absolute() and
           Path(__file__).resolve().parents[4] not in ensayo.parents,
           "rutas_no_admitidas")
    comprobar(respaldo, True)
    comprobar(ensayo, True)
    origen = ensayo / "incorporacion"
    comprobar(origen, True)
    plantilla = respaldo / "incorporacion-servidor-20260910/config"
    comprobar(plantilla, True)
    relativos = []
    for ruta in plantilla.rglob("*"):
        comprobar(ruta, ruta.is_dir())
        if ruta.is_file() and "portal-externo" not in ruta.relative_to(plantilla).parts:
            relativos.append(ruta.relative_to(plantilla))
    relativos.extend(Path(nombre) for nombre in GENERADOS_RUNTIME)
    carpeta_meritos = origen / "externo-dsn/meritos"
    if carpeta_meritos.exists():
        comprobar(carpeta_meritos, True)
        exigir(all((carpeta_meritos / nombre).is_file() for nombre in MERITOS_RUNTIME),
               "meritos_runtime_incompleto")
        relativos.extend(Path("externo-dsn/meritos") / nombre for nombre in MERITOS_RUNTIME)
    destino = ensayo / "incorporacion_web"
    if args.aplicar:
        destino.mkdir(mode=0o700, exist_ok=True)
    comprobar(destino, True)
    exigir(not destino.stat().st_mode & 0o077, "web_no_privada")
    for relativo in relativos:
        copiar(origen, destino, relativo, args.aplicar)
    permitidos = {p.as_posix() for p in relativos}
    existentes = {p.relative_to(destino).as_posix() for p in destino.rglob("*") if p.is_file()}
    exigir(existentes == permitidos, "web_contiene_fichero_fuera_de_lista")
    plan = {"version": 1, "origen": str(destino), "destino": "/vec-incorporacion",
            "solo_lectura": True, "archivos": len(permitidos),
            "material_interno": str(ensayo / "material"),
            "material_externo": str(ensayo / "externo")}
    archivo = ensayo / "mountplan-aspirante.json"
    if archivo.exists():
        comprobar(archivo)
        anterior = json.loads(archivo.read_text())
        exigir(not archivo.stat().st_mode & 0o077 and
               all(anterior.get(k) == plan[k] for k in plan if k != "archivos") and
               isinstance(anterior.get("archivos"), int) and
               anterior["archivos"] <= plan["archivos"], "mountplan_divergente")
        if anterior != plan:
            exigir(args.aplicar, "mountplan_pendiente")
            archivo.write_text(json.dumps(plan, ensure_ascii=False, separators=(",", ":")) + "\n")
    elif args.aplicar:
        archivo.write_text(json.dumps(plan, ensure_ascii=False, separators=(",", ":")) + "\n")
        archivo.chmod(0o600)
    else:
        raise ValueError("mountplan_pendiente")
    print("incorporacion_web=preparada" if args.aplicar else "incorporacion_web=comprobada")
    print("archivos_runtime=" + str(len(permitidos)))


if __name__ == "__main__":
    try:
        main()
    except (OSError, ValueError, KeyError, TypeError, json.JSONDecodeError) as error:
        raise SystemExit("incorporacion_web=error clave=" + str(error).split(":", 1)[0]) from None
