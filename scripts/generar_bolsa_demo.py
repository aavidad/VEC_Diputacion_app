#!/usr/bin/env python3
"""Genera el juego de datos sintético de bolsas de empleo para las demostraciones.

Parte del catálogo público de la RPT (`data/catalogos/rpt/v1.rpt-2026.json`): una bolsa
por cada una de las categorías con más dotación, con candidaturas sintéticas ordenadas,
su estado y su último llamamiento. Determinista (semilla fija). Ninguna persona real:
las candidaturas se identifican por un número y un nombre de demostración.
"""
from __future__ import annotations

import json
import random
from datetime import date, datetime, timedelta, timezone
from pathlib import Path

RAIZ = Path(__file__).resolve().parents[1]
CATALOGO_RPT = RAIZ / "data" / "catalogos" / "rpt" / "v1.rpt-2026.json"
SALIDA = RAIZ / "data" / "demo" / "bolsa" / "v1.bolsas-demo.json"
NUM_BOLSAS = 12
ESTADOS = [  # peso, clave, etiqueta
    (58, "disponible", "Disponible"),
    (14, "trabajando", "Trabajando"),
    (10, "no_disponible", "No disponible (pausa voluntaria)"),
    (6, "pendiente_incorporacion", "Pendiente de incorporación"),
    (5, "disponible_desde", "Disponible a partir de una fecha"),
    (4, "renuncia", "Renuncia"),
    (3, "excluido", "Excluido"),
]


def main() -> int:
    rpt = json.loads(CATALOGO_RPT.read_text(encoding="utf-8"))
    categorias = [c for c in rpt["categorias"] if c["origen"] == "categoria"][:NUM_BOLSAS]
    azar = random.Random(20260917)
    base = datetime(2026, 9, 1, 9, 0, tzinfo=timezone.utc)
    bolsas, candidaturas, llamamientos = [], [], []
    siguiente_candidatura = 1
    for indice, categoria in enumerate(categorias, start=1):
        bolsa_ref = f"bolsa:demo:{categoria['clave']}"
        tamano = azar.randint(18, 42)
        constituida = date(2025, 1 + (indice % 12), 1 + (indice * 3) % 27)
        bolsas.append({
            "bolsa_ref": bolsa_ref,
            "categoria_ref": f"categoria:rpt:{categoria['clave']}",
            "categoria": categoria["denominacion"],
            "grupos": categoria["grupos"],
            "constituida_en": constituida.isoformat(),
            "vigente_desde": constituida.isoformat(),
            "vigente_hasta": None,
            "tipo_lista": "rotatoria" if indice % 3 else "cerrada",
            "candidaturas": tamano,
        })
        for orden in range(1, tamano + 1):
            numero = siguiente_candidatura
            siguiente_candidatura += 1
            estado = azar.choices([e[1] for e in ESTADOS], weights=[e[0] for e in ESTADOS])[0]
            desde = base - timedelta(days=azar.randint(3, 400))
            candidatura = {
                "candidatura_ref": f"candidatura:demo:{numero:04d}",
                "sujeto_ref": f"sujeto:demo:{numero:04d}",
                "nombre_visible": f"Candidatura de demostración {numero:04d}",
                "bolsa_ref": bolsa_ref,
                "orden": orden,
                "puntuacion": round(max(2.0, 30.0 - orden * azar.uniform(0.35, 0.75)), 2),
                "estado_clave": estado,
                "estado": next(e[2] for e in ESTADOS if e[1] == estado),
                "estado_desde": desde.isoformat(),
                "disponible_desde": (base + timedelta(days=azar.randint(10, 90))).date().isoformat() if estado == "disponible_desde" else None,
                "contactos_previos": azar.randint(0, 3),
            }
            candidaturas.append(candidatura)
            if estado in ("trabajando", "pendiente_incorporacion", "renuncia") or (estado == "disponible" and azar.random() < 0.12):
                fecha = desde + timedelta(days=azar.randint(1, 20))
                resultado = {"trabajando": "aceptado", "pendiente_incorporacion": "aceptado", "renuncia": "renuncia"}.get(estado, "sin_respuesta")
                llamamientos.append({
                    "llamamiento_ref": f"llamamiento:demo:{len(llamamientos) + 1:04d}",
                    "candidatura_ref": candidatura["candidatura_ref"],
                    "bolsa_ref": bolsa_ref,
                    "puesto": f"{categoria['denominacion'].title()} — sustitución temporal",
                    "centro_ref": None,
                    "comunicado_en": fecha.isoformat(),
                    "canal": azar.choice(["correo", "telefono", "correo"]),
                    "plazo_respuesta_hasta": (fecha + timedelta(days=2)).isoformat(),
                    "resultado": resultado,
                })
    salida = {
        "esquema": "vec.demo.bolsa.v1",
        "aviso": "Datos sintéticos de demostración: bolsas por categoría de la RPT y candidaturas numeradas sin personas reales. Sin validez administrativa.",
        "generado_en": "2026-09-17T00:00:00Z",
        "resumen": {"bolsas": len(bolsas), "candidaturas": len(candidaturas), "llamamientos": len(llamamientos)},
        "bolsas": bolsas,
        "candidaturas": candidaturas,
        "llamamientos": llamamientos,
    }
    SALIDA.parent.mkdir(parents=True, exist_ok=True)
    SALIDA.write_text(json.dumps(salida, ensure_ascii=False, indent=1) + "\n", encoding="utf-8")
    print(f"{SALIDA.relative_to(RAIZ)}: {salida['resumen']}")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
