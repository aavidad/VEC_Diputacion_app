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

import json
import re
import sys
import unicodedata
from collections import defaultdict
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


def grupos_de(valor: str) -> list[str]:
    partes = [p.strip().upper() for p in re.split(r"[/,]", valor or "") if p.strip()]
    return [p for p in partes if p in GRUPOS_VALIDOS]


def main() -> int:
    if not ENTRADA.exists():
        print(f"falta la importación local {ENTRADA.relative_to(RAIZ)}", file=sys.stderr)
        return 2
    datos = json.loads(ENTRADA.read_text(encoding="utf-8"))
    puestos_salida: list[dict] = []
    categorias: dict[str, dict] = {}
    for p in datos["positions"]:
        grupos = grupos_de(p.get("group", ""))
        denominacion = (p.get("category_code") or p.get("name") or "").strip()
        categoria_clave = clave(denominacion) if denominacion and not denominacion[0].isdigit() else ""
        puestos_salida.append({
            "codigo": p["code"],
            "denominacion": p["name"].strip(),
            "centro_codigo": p.get("center_code", ""),
            "centro": p.get("center_name", "").strip(),
            "delegacion": p.get("delegation", "").strip(),
            "grupos": grupos,
            "escala": (p.get("scale") or "").strip() if (p.get("scale") or "") in ("AE", "AG", "AGAE", "HN") else "",
            "categoria_clave": categoria_clave,
            "nivel_destino": int(p.get("destination_level") or 0),
            "complemento_especifico_anual_centimos": int(p.get("annual_amount_cents") or 0),
            "dotacion": int(p.get("dot") or 0),
            "tipo": p.get("type", ""),
            "provision": p.get("provision", ""),
        })
        if categoria_clave and grupos:
            c = categorias.setdefault(categoria_clave, {
                "clave": categoria_clave, "denominacion": denominacion, "grupos": [],
                "escalas": [], "dotacion": 0, "puestos": 0, "niveles_destino": [],
                "complementos_especificos_anuales_centimos": [],
            })
            for g in grupos:
                if g not in c["grupos"]:
                    c["grupos"].append(g)
            esc = puestos_salida[-1]["escala"]
            if esc and esc not in c["escalas"]:
                c["escalas"].append(esc)
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
            "clave": c["clave"], "denominacion": c["denominacion"], "grupos": orden_grupos,
            "escalas": sorted(c["escalas"]), "dotacion": c["dotacion"], "puestos": c["puestos"],
            "nivel_destino_mediana": mediana(c["niveles_destino"]),
            "complemento_especifico_anual_centimos_mediana": mediana(c["complementos_especificos_anuales_centimos"]),
        })
    salida = {
        "esquema": "vec.catalogo.rpt.v1",
        "fuente": {
            "documento": "Relación de Puestos de Trabajo de la Diputación de Granada 2026 (publicada), revisión 2026-05-07",
            "importacion": datos.get("version", ""),
            "generado_en": date.today().isoformat(),
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
    }
    SALIDA.parent.mkdir(parents=True, exist_ok=True)
    SALIDA.write_text(json.dumps(salida, ensure_ascii=False, indent=1) + "\n", encoding="utf-8")
    print(f"{SALIDA.relative_to(RAIZ)}: {salida['resumen']}")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
