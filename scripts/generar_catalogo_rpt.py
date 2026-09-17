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


# Trozos de la columna de categoría que el PDF parte por el salto de línea y que no
# son categorías por sí mismos (revisados a mano sobre la RPT 2026).
EXCLUIDAS = {
    "ACC.COMUN.", "BOLSAS", "CALAH.", "CIBERSEGURIDAD", "CONTROL CONTABLE", "FINANCIACIÓN AFECTADA",
    "ILLORA", "NO PLANIFICABLE/PERMANENTE", "PLANIFICABLE", "PROFESIONAL", "PROVINCIALES", "PUBLIC.",
    "PUBLICACIONES", "PUBLICAS", "SOCIAL Y PENSIONES", "SOPORTE A TESORERÍA", "SUMINISTROS", "TEMPLE",
    "TEMPORAL", "TERRITITORIALES", "TOPOGRAFÍA", "TRIBUNAL CONTRAT. PUB.", "VIVIENDA",
}

CONTINUACION = re.compile(r"^(Y|DE|DEL|E|EN|LA|LAS|LOS|A|AL|PARA|CON|SIN)\s")


def limpiar_denominacion(texto: str) -> str:
    """Quita el código numérico inicial, los sufijos de tabla («2A», «(B)») y los
    espacios que el PDF mete tras la barra («TÉCNICO/ A»)."""
    n = re.sub(r"\s+", " ", (texto or "").strip().upper())
    n = re.sub(r"^\d+\s+", "", n)
    n = re.sub(r"\s+2A$", "", n)
    n = re.sub(r"\s*\(B\)$", "", n)
    n = re.sub(r"/\s+A\b", "/A", n)
    return n.strip()


def es_fragmento(nombre: str, nombres_completos: set[str]) -> bool:
    """Un trozo de denominación partida por el salto de línea del PDF: pocas letras,
    empieza por preposición o conjunción, o es el final de otra denominación."""
    if len(re.sub(r"[^A-ZÁÉÍÓÚÜÑ]", "", nombre)) < 4 or CONTINUACION.match(nombre):
        return True
    return any(otro != nombre and otro.endswith(" " + nombre) for otro in nombres_completos)


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
    nombres_completos = {limpiar_denominacion(p["name"]) for p in datos["positions"]}
    nombres_completos |= {limpiar_denominacion(p.get("category_code") or "") for p in datos["positions"]}
    frecuencia_categoria: dict[str, int] = defaultdict(int)
    for p in datos["positions"]:
        frecuencia_categoria[limpiar_denominacion(p.get("category_code") or "")] += 1
    for p in datos["positions"]:
        grupos = grupos_de(p.get("group", ""))
        # Categoría: la columna de categoría de la RPT; si falta, la denominación del
        # puesto solo cuando es un puesto base (tipo N), nunca uno singularizado.
        origen = "categoria"
        denominacion = limpiar_denominacion(p.get("category_code") or "")
        if not denominacion and (p.get("type") or "") == "N":
            denominacion = limpiar_denominacion(p.get("name") or "")
            origen = "denominacion"
        # Una categoría muy repetida se acepta aunque sea el final de otra denominación
        # («ADMINISTRATIVO» y «AUXILIAR ADMINISTRATIVO» coexisten); las raras que son
        # final de otra son trozos partidos por el salto de línea del PDF.
        if denominacion and (
            denominacion in EXCLUIDAS
            or CONTINUACION.match(denominacion)
            or len(re.sub(r"[^A-ZÁÉÍÓÚÜÑ]", "", denominacion)) < 4
            or (frecuencia_categoria.get(denominacion, 0) < 10 and es_fragmento(denominacion, nombres_completos))
        ):
            denominacion = ""
        categoria_clave = clave(denominacion) if denominacion else ""
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
                "clave": categoria_clave, "denominacion": denominacion, "origen": origen, "grupos": [],
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
