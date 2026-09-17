#!/usr/bin/env python3
"""Genera el juego de datos sintético de bolsas de empleo para las demostraciones.

Parte del catálogo público de la RPT (`data/catalogos/rpt/v1.rpt-2026.json`): una bolsa
por cada una de las categorías con más dotación, con candidaturas sintéticas ordenadas,
su estado y su último llamamiento. Determinista (semilla fija). Ninguna persona real:
nombres y apellidos se combinan al azar de listas cortas y el documento va enmascarado.

Produce dos salidas coherentes entre sí:
- `data/demo/bolsa/v1.bolsas-demo.json`, que lee el área personal en perfil de desarrollo;
- `data/demo/bolsa/convoca/<categoria>/{resumen,detalle}.csv` con las cabeceras literales
  de Convoca que acepta `internal/modules/bolsa/adapters/xlsconvoca`, para cargar la
  principal por el importador real. Con `--xls`, convierte cada CSV a BIFF8 con xlwt.
"""
from __future__ import annotations

import argparse
import csv
import json
import random
import shutil
import subprocess
from datetime import date, datetime, timedelta, timezone
from pathlib import Path

RAIZ = Path(__file__).resolve().parents[1]
CATALOGO_RPT = RAIZ / "data" / "catalogos" / "rpt" / "v1.rpt-2026.json"
SALIDA = RAIZ / "data" / "demo" / "bolsa" / "v1.bolsas-demo.json"
SALIDA_CONVOCA = RAIZ / "data" / "demo" / "bolsa" / "convoca"
CABECERA_RESUMEN = ["DNI/NIE", "Primer Apellido", "Segundo Apellido", "Nombre", "Turno", "Experiencia", "Formacion", "Total"]
CABECERA_DETALLE = ["DNI/NIE", "Primer Apellido", "Segundo Apellido", "Nombre", "Turno", "Grupo", "Descripcion del grupo",
                    "Orden grupo", "Descripcion del merito", "Puntos autobaremacion", "Puntos tribunal", "Motivo"]
NOMBRES = ["Ana", "Bruno", "Carla", "Daniel", "Elena", "Fabio", "Gloria", "Hugo", "Irene", "Jorge", "Lucía", "Marcos",
           "Nuria", "Óscar", "Paula", "Raúl", "Sara", "Tomás", "Úrsula", "Víctor", "Ximena", "Yago", "Zaira", "Adrián",
           "Beatriz", "Claudio", "Dolores", "Emilio", "Fátima", "Gonzalo", "Helena", "Iván", "Julia", "Kevin", "Laura",
           "Manuel", "Noelia", "Pablo", "Rocío", "Sergio"]
APELLIDOS = ["Alcalde", "Barranco", "Cuevas", "Delgado", "Escudero", "Fuentes", "Garrido", "Herrera", "Ibáñez", "Jurado",
             "Lozano", "Medina", "Navarro", "Ortega", "Pastor", "Quintana", "Redondo", "Salinas", "Torralba", "Uceda",
             "Valverde", "Zamora", "Aguilera", "Bermejo", "Castaño", "Domínguez", "Espinosa", "Ferrer", "Guerrero",
             "Hidalgo", "Iglesias", "Lara", "Molina", "Nieto", "Olmo", "Peñalver", "Rubio", "Serrano", "Tejada", "Vidal"]
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


def escribir_convoca(clave: str, candidaturas: list[dict], azar: random.Random) -> None:
    """Escribe resumen.csv y detalle.csv con las cabeceras literales de Convoca."""
    carpeta = SALIDA_CONVOCA / clave
    carpeta.mkdir(parents=True, exist_ok=True)
    with (carpeta / "resumen.csv").open("w", encoding="utf-8", newline="") as f:
        w = csv.writer(f)
        w.writerow(CABECERA_RESUMEN)
        for c in candidaturas:
            cv = c["convoca"]
            w.writerow([c["documento_enmascarado"], cv["primer_apellido"], cv["segundo_apellido"], cv["nombre"], cv["turno"],
                        f"{cv['experiencia']:g}", f"{cv['formacion']:g}", f"{round(cv['experiencia'] + cv['formacion'], 2):g}"])
    with (carpeta / "detalle.csv").open("w", encoding="utf-8", newline="") as f:
        w = csv.writer(f)
        w.writerow(CABECERA_DETALLE)
        for c in candidaturas:
            cv = c["convoca"]
            persona = [c["documento_enmascarado"], cv["primer_apellido"], cv["segundo_apellido"], cv["nombre"], cv["turno"]]
            for grupo, descripcion, total, meritos in (
                ("EXP", "Experiencia profesional", cv["experiencia"],
                 ["Servicios en administración local", "Servicios en otra administración", "Servicios en el sector privado"]),
                ("FOR", "Formacion", cv["formacion"], ["Curso de formación específica", "Titulación adicional", "Jornadas y seminarios"]),
            ):
                partes = azar.randint(1, len(meritos))
                pesos = [azar.random() for _ in range(partes)]
                suma = sum(pesos) or 1.0
                restante = round(total, 2)
                for orden, (merito, peso) in enumerate(zip(meritos, pesos), start=1):
                    puntos = restante if orden == partes else round(total * peso / suma, 2)
                    restante = round(restante - puntos, 2)
                    w.writerow(persona + [grupo, descripcion, orden, merito, f"{puntos:g}", f"{puntos:g}", ""])


def convertir_a_xls() -> int:
    """Convierte cada CSV a BIFF8 (MS Excel 97) con xlwt; cabeceras y celdas de texto,
    puntuaciones como número, como exporta Convoca."""
    try:
        import xlwt  # type: ignore
    except ImportError:
        print("xlwt no está instalado (pip install xlwt); se dejan solo los CSV.")
        return 1
    convertidos = 0
    for csv_ruta in sorted(SALIDA_CONVOCA.glob("*/*.csv")):
        libro = xlwt.Workbook(encoding="utf-8")
        hoja = libro.add_sheet(csv_ruta.stem)
        with csv_ruta.open(encoding="utf-8", newline="") as f:
            for fila, celdas in enumerate(csv.reader(f)):
                for columna, valor in enumerate(celdas):
                    if fila > 0 and valor != "" and valor.replace(".", "", 1).isdigit():
                        hoja.write(fila, columna, float(valor))
                    elif valor != "":
                        hoja.write(fila, columna, valor)
        libro.save(str(csv_ruta.with_suffix(".xls")))
        convertidos += 1
    print(f"{SALIDA_CONVOCA.relative_to(RAIZ)}: {convertidos} libros XLS")
    return 0


def main() -> int:
    argumentos = argparse.ArgumentParser(description=__doc__.splitlines()[0])
    argumentos.add_argument("--xls", action="store_true", help="convertir los CSV de Convoca a XLS (BIFF8) con xlwt")
    opciones = argumentos.parse_args()
    if SALIDA_CONVOCA.exists():
        shutil.rmtree(SALIDA_CONVOCA)
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
        candidaturas_bolsa = []
        for orden in range(1, tamano + 1):
            numero = siguiente_candidatura
            siguiente_candidatura += 1
            estado = azar.choices([e[1] for e in ESTADOS], weights=[e[0] for e in ESTADOS])[0]
            desde = base - timedelta(days=azar.randint(3, 400))
            nombre = azar.choice(NOMBRES)
            apellido1 = azar.choice(APELLIDOS)
            apellido2 = azar.choice(APELLIDOS) if azar.random() < 0.9 else ""
            experiencia = round(azar.uniform(0.0, 20.0), 2)
            formacion = round(azar.uniform(0.0, 8.0), 2)
            candidatura = {
                "candidatura_ref": f"candidatura:demo:{numero:04d}",
                "sujeto_ref": f"sujeto:demo:{numero:04d}",
                "nombre_visible": " ".join(x for x in (nombre, apellido1, apellido2) if x),
                "documento_enmascarado": f"***{numero:04d}**",
                "convoca": {"primer_apellido": apellido1, "segundo_apellido": apellido2, "nombre": nombre,
                            "turno": "Discapacidad" if azar.random() < 0.07 else "Libre",
                            "experiencia": experiencia, "formacion": formacion},
                "bolsa_ref": bolsa_ref,
                "orden": orden,
                "puntuacion": round(experiencia + formacion, 2),
                "estado_clave": estado,
                "estado": next(e[2] for e in ESTADOS if e[1] == estado),
                "estado_desde": desde.isoformat(),
                "disponible_desde": (base + timedelta(days=azar.randint(10, 90))).date().isoformat() if estado == "disponible_desde" else None,
                "contactos_previos": azar.randint(0, 3),
            }
            candidaturas_bolsa.append(candidatura)
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
        candidaturas_bolsa.sort(key=lambda c: (-c["puntuacion"], c["candidatura_ref"]))
        for orden, candidatura in enumerate(candidaturas_bolsa, start=1):
            candidatura["orden"] = orden
        candidaturas.extend(candidaturas_bolsa)
        escribir_convoca(categoria["clave"], candidaturas_bolsa, azar)
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
    if opciones.xls:
        return convertir_a_xls()
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
