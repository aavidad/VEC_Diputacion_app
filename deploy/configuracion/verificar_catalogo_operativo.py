#!/usr/bin/env python3
"""Valida y proyecta el catálogo operativo sin leer ni escribir secretos."""

import argparse
import json
import re
import sys
from pathlib import Path

NOMBRES_OBLIGATORIOS = {
    "nombre", "lector", "superficies", "estado", "requerido",
    "valor_predeterminado", "tipo", "secreto", "descripcion",
}
ESTADOS = {"activo", "prohibido", "alias", "heredado"}
SUPERFICIES = {"vec-publico", "vec-interno", "vec-emisor-capacidad-v4"}
PATRON_NOMBRE = re.compile(r"^(?:VEC|BOLSA)_[A-Z0-9_]+$")
PATRON_LITERAL = re.compile(r'"((?:VEC|BOLSA)_[A-Z0-9_]+)"')


def raiz_repositorio() -> Path:
    return Path(__file__).resolve().parents[2]


def argumentos() -> argparse.Namespace:
    parser = argparse.ArgumentParser(
        description="Valida el catálogo VEC/Bolsa o genera una plantilla mínima.")
    parser.add_argument("--catalogo", type=Path,
                        default=Path(__file__).with_name("catalogo_operativo.json"),
                        help="Ruta al catálogo JSON (por defecto, el distribuido).")
    grupo = parser.add_mutually_exclusive_group(required=True)
    grupo.add_argument("--validar", action="store_true",
                        help="Comprueba estructura, paridad y ausencia de secretos.")
    grupo.add_argument("--plantilla", choices=sorted(SUPERFICIES),
                        help="Escribe en stdout la plantilla de una superficie.")
    return parser.parse_args()


def cargar_catalogo(ruta: Path) -> dict:
    try:
        with ruta.open(encoding="utf-8") as archivo:
            return json.load(archivo)
    except (OSError, json.JSONDecodeError) as error:
        raise ValueError(f"CAT001 catálogo JSON no legible: {error}") from error


def contiene_campo_valor(objeto: object) -> bool:
    if isinstance(objeto, dict):
        return "valor" in objeto or any(contiene_campo_valor(v) for v in objeto.values())
    if isinstance(objeto, list):
        return any(contiene_campo_valor(v) for v in objeto)
    return False


def literales_productivos(raiz: Path) -> set[str]:
    rutas = [raiz / "config", raiz / "internal/app/composicion",
             raiz / "cmd/vec-emisor-capacidad-v4"]
    nombres: set[str] = set()
    for base in rutas:
        for ruta in base.rglob("*.go"):
            if ruta.name.endswith("_test.go"):
                continue
            nombres.update(PATRON_LITERAL.findall(ruta.read_text(encoding="utf-8")))
    return nombres


def validar(catalogo: dict, raiz: Path) -> list[str]:
    errores: list[str] = []
    if contiene_campo_valor(catalogo):
        errores.append("CAT002 el catálogo no admite el campo 'valor'.")
    variables = catalogo.get("variables")
    if not isinstance(variables, list):
        return ["CAT003 'variables' debe ser una lista."]
    nombres: list[str] = []
    for indice, variable in enumerate(variables, start=1):
        prefijo = f"CAT{100 + indice:03d} variable {indice}"
        if not isinstance(variable, dict) or set(variable) != NOMBRES_OBLIGATORIOS:
            errores.append(f"{prefijo}: campos obligatorios incompletos o desconocidos.")
            continue
        nombre = variable["nombre"]
        nombres.append(nombre)
        if not isinstance(nombre, str) or not PATRON_NOMBRE.fullmatch(nombre):
            errores.append(f"{prefijo}: nombre inválido.")
        if variable["estado"] not in ESTADOS:
            errores.append(f"{prefijo}: estado inválido.")
        if not isinstance(variable["superficies"], list) or not variable["superficies"]:
            errores.append(f"{prefijo}: debe declarar lector o superficie.")
        if not isinstance(variable["requerido"], bool) or not isinstance(variable["secreto"], bool):
            errores.append(f"{prefijo}: requerido y secreto deben ser booleanos.")
        if not isinstance(variable["lector"], str) or not isinstance(variable["tipo"], str):
            errores.append(f"{prefijo}: lector y tipo deben ser texto.")
        if not isinstance(variable["descripcion"], str) or not variable["descripcion"].strip():
            errores.append(f"{prefijo}: falta descripción.")
        if variable["secreto"] and variable["valor_predeterminado"] is not None:
            errores.append(f"{prefijo}: una variable secreta no puede tener valor predeterminado.")
        if variable["estado"] == "prohibido" and variable["requerido"]:
            errores.append(f"{prefijo}: una variable prohibida no puede ser requerida.")
        desconocidas = set(variable["superficies"]) - (SUPERFICIES | {"configuracion-general"})
        if desconocidas:
            errores.append(f"{prefijo}: superficie desconocida: {sorted(desconocidas)}.")
    repetidos = sorted({nombre for nombre in nombres if nombres.count(nombre) > 1})
    if repetidos:
        errores.append(f"CAT004 nombres duplicados: {', '.join(repetidos)}.")
    catalogados = set(nombres)
    literales = literales_productivos(raiz)
    if faltan := sorted(literales - catalogados):
        errores.append(f"CAT005 faltan literales productivos: {', '.join(faltan)}.")
    if sobran := sorted(catalogados - literales):
        errores.append(f"CAT006 sobran nombres sin literal productivo: {', '.join(sobran)}.")
    return errores


def generar_plantilla(catalogo: dict, superficie: str) -> str:
    activas = [v for v in catalogo["variables"]
               if superficie in v["superficies"] and v["estado"] == "activo" and v["requerido"]]
    lineas = [f"# Plantilla mínima de {superficie}; exportar solo en la sesión del proceso."]
    for variable in activas:
        lineas.append(f"{variable['nombre']}=")
    return "\n".join(lineas) + "\n"


def main() -> int:
    opciones = argumentos()
    try:
        catalogo = cargar_catalogo(opciones.catalogo)
        errores = validar(catalogo, raiz_repositorio())
    except ValueError as error:
        print(error, file=sys.stderr)
        return 2
    if errores:
        print("\n".join(errores), file=sys.stderr)
        return 1
    if opciones.validar:
        print("CAT000 catálogo operativo válido: paridad productiva confirmada.")
    else:
        sys.stdout.write(generar_plantilla(catalogo, opciones.plantilla))
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
