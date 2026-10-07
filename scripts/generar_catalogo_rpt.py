#!/usr/bin/env python3
"""Genera el catálogo público de la RPT (puestos y categorías) para VEC.

Lee la importación local `config/rpt_positions_import.json` (no versionada: nace del
PDF de la RPT publicada) y escribe `data/catalogos/rpt/v1.rpt-2026.json` con lo que es
público y útil para Contratación, Bolsa, Cronos y Dietas: puestos con su centro, grupo,
escala, nivel de destino, complemento específico anual y dotación; y categorías
derivadas de las denominaciones con su grupo. No incluye ocupantes, rutas locales,
páginas ni texto en bruto.
"""
from __future__ import annotations

import argparse
import json
import re
import sys
import unicodedata
from datetime import date
from pathlib import Path

RAIZ = Path(__file__).resolve().parents[1]
ENTRADA = RAIZ / "config" / "rpt_positions_import.json"
SALIDA = RAIZ / "data" / "catalogos" / "rpt" / "v1.rpt-2026.json"
GRUPOS_VALIDOS = ("A1", "A2", "B", "C1", "C2", "AP")


def clave(texto: str) -> str:
    plano = unicodedata.normalize("NFKD", texto).encode("ascii", "ignore").decode()
    plano = re.sub(r"[^a-z0-9]+", "-", plano.lower()).strip("-")
    return plano[:60]


CODIGO_CATEGORIA = re.compile(r"(?<!\S)(\d{1,3})\s+(?=[A-ZÁÉÍÓÚÜÑ])")


def limpiar_denominacion(texto: str) -> str:
    """Quita el código numérico inicial, los sufijos de tabla («2A», «(B)») y los
    espacios que el PDF mete tras la barra («TÉCNICO/ A»)."""
    n = re.sub(r"\s+", " ", (texto or "").strip().upper())
    n = re.sub(r"^\d+\s+", "", n)
    n = re.sub(r"\s+2A$", "", n)
    n = re.sub(r"\s*\(B\)$", "", n)
    n = re.sub(r"/\s+A\b", "/A", n)
    return n.strip()


def denominaciones_categoria(valor: str) -> list[str]:
    """Separa las alternativas numeradas de una misma celda de categoría."""
    texto = (valor or "").strip().upper()
    codigos = list(CODIGO_CATEGORIA.finditer(texto))
    if len(codigos) < 2:
        nombre = limpiar_denominacion(texto)
        return [nombre] if nombre else []
    nombres = []
    for indice, codigo in enumerate(codigos):
        fin = codigos[indice + 1].start() if indice + 1 < len(codigos) else None
        nombre = limpiar_denominacion(texto[codigo.start():fin])
        if nombre:
            nombres.append(nombre)
    return nombres


def grupos_de(valor: str) -> list[str]:
    partes = [p.strip().upper() for p in re.split(r"[/,]", valor or "") if p.strip()]
    return [p for p in partes if p in GRUPOS_VALIDOS]


def main() -> int:
    parser = argparse.ArgumentParser()
    parser.add_argument("--in", dest="entrada", type=Path, default=ENTRADA)
    parser.add_argument("--out", type=Path, default=SALIDA)
    parser.add_argument("--generated-on", type=date.fromisoformat, default=date.today())
    args = parser.parse_args()
    if not args.entrada.exists():
        print("falta la importación local indicada", file=sys.stderr)
        return 2
    datos = json.loads(args.entrada.read_text(encoding="utf-8"))
    puestos_salida: list[dict] = []
    categorias: dict[str, dict] = {}
    # Un grupo compuesto no identifica el subgrupo de cada alternativa numerada.
    grupos_acreditados: dict[str, set[str]] = {}
    for p in datos["positions"]:
        nombres = denominaciones_categoria(p.get("category_code") or "")
        grupos = grupos_de(p.get("group", ""))
        if len(nombres) == 1 and len(grupos) == 1:
            grupos_acreditados.setdefault(nombres[0], set()).add(grupos[0])
    pendientes: set[str] = set()
    for p in datos["positions"]:
        grupos = grupos_de(p.get("group", ""))
        # Categoría: la columna de categoría de la RPT; si falta, la denominación del
        # puesto solo cuando es un puesto base (tipo N), nunca uno singularizado.
        origen = "categoria"
        denominaciones = denominaciones_categoria(p.get("category_code") or "")
        if not denominaciones and (p.get("type") or "") == "N":
            denominaciones = [limpiar_denominacion(p.get("name") or "")]
            origen = "denominacion"
        denominaciones = [nombre for nombre in denominaciones if nombre]
        claves = [clave(nombre) for nombre in denominaciones]
        categoria_clave = claves[0] if len(claves) == 1 else ""
        puestos_salida.append({
            "codigo": p["code"],
            "denominacion": p["name"].strip(),
            "centro_codigo": p.get("center_code", ""),
            "centro": p.get("center_name", "").strip(),
            "delegacion": p.get("delegation", "").strip(),
            "grupos": grupos,
            "escala": (p.get("scale") or "").strip() if (p.get("scale") or "") in ("AE", "AG", "AGAE", "HN") else "",
            "categoria_clave": categoria_clave,
            "categorias_claves": claves,
            "nivel_destino": int(p.get("destination_level") or 0),
            "complemento_especifico_anual_centimos": int(p.get("annual_amount_cents") or 0),
            "dotacion": int(p.get("dot") or 0),
            "tipo": p.get("type", ""),
            "provision": p.get("provision", ""),
        })
        for denominacion, clave_categoria in zip(denominaciones, claves):
            grupos_categoria = grupos if len(denominaciones) == 1 else sorted(grupos_acreditados.get(denominacion, set()), key=GRUPOS_VALIDOS.index)
            if not grupos_categoria:
                pendientes.add(denominacion)
                continue
            c = categorias.setdefault(clave_categoria, {
                "clave": clave_categoria, "denominacion": denominacion, "origen": origen, "grupos": [],
                "escalas": [], "dotacion": 0, "puestos": 0, "niveles_destino": [],
                "complementos_especificos_anuales_centimos": [],
            })
            for g in grupos_categoria:
                if g not in c["grupos"]:
                    c["grupos"].append(g)
            esc = puestos_salida[-1]["escala"]
            if esc and esc not in c["escalas"]:
                c["escalas"].append(esc)
            # La RPT no distribuye la dotación entre alternativas.
            if len(denominaciones) == 1:
                c["dotacion"] += puestos_salida[-1]["dotacion"]
                c["puestos"] += 1
                if puestos_salida[-1]["nivel_destino"]:
                    c["niveles_destino"].append(puestos_salida[-1]["nivel_destino"])
                if puestos_salida[-1]["complemento_especifico_anual_centimos"]:
                    c["complementos_especificos_anuales_centimos"].append(puestos_salida[-1]["complemento_especifico_anual_centimos"])

    def mediana(valores: list[int]) -> int:
        if not valores:
            return 0
        v = sorted(valores)
        return v[len(v) // 2]

    lista_categorias = []
    for c in sorted(categorias.values(), key=lambda x: (-x["dotacion"], x["denominacion"])):
        orden_grupos = sorted(c["grupos"], key=GRUPOS_VALIDOS.index)
        lista_categorias.append({
            "clave": c["clave"], "denominacion": c["denominacion"], "origen": c["origen"], "grupos": orden_grupos,
            "escalas": sorted(c["escalas"]), "dotacion": c["dotacion"], "puestos": c["puestos"],
            "nivel_destino_mediana": mediana(c["niveles_destino"]),
            "complemento_especifico_anual_centimos_mediana": mediana(c["complementos_especificos_anuales_centimos"]),
        })
    salida = {
        "esquema": "vec.catalogo.rpt.v1",
        "fuente": {
            "documento": "Relación de Puestos de Trabajo de la Diputación de Granada 2026 (publicada), revisión 2026-05-07",
            "importacion": datos.get("version", ""),
            "generado_en": args.generated_on.isoformat(),
            "aviso": "Datos públicos de puestos; no contiene ocupantes ni datos personales. No acredita vigencia administrativa.",
        },
        "resumen": {
            "puestos": len(puestos_salida),
            "dotacion": sum(p["dotacion"] for p in puestos_salida),
            "categorias": len(lista_categorias),
            "centros": len({p["centro_codigo"] for p in puestos_salida if p["centro_codigo"]}),
        },
        "categorias": lista_categorias,
        "puestos": puestos_salida,
        "categorias_pendientes_grupo": sorted(pendientes),
    }
    args.out.parent.mkdir(parents=True, exist_ok=True)
    args.out.write_text(json.dumps(salida, ensure_ascii=False, indent=1) + "\n", encoding="utf-8")
    print(f"{args.out}: {salida['resumen']}")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
