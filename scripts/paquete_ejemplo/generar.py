#!/usr/bin/env python3
"""Genera el paquete de datos de ejemplo de VEC (Bolsa y Contratación temporal).

Es un paquete separable, como los datos de ejemplo de un gestor de contenidos:
todo lo que produce lleva el identificador de paquete en su procedencia
(`paquete.json`, referencias de bolsa y de persona) y se carga solo en entornos
de desarrollo o presentación. Producción se instala sin él. No carga nada en
ninguna base: escribe ficheros en `--salida`.

Salida:
- `convoca/<categoria>/resumen.xls` y `detalle.xls`: exportaciones con el formato
  de Convoca que acepta el importador B1 (`vec-server importar-convoca`);
- `personas.json`: todas las personas del paquete, con sus datos completos;
- `contratacion/expedientes.json`: escenarios de solicitud de Contratación temporal;
- `identidades.json`: identidades de certificado de desarrollo con CN legible;
- `paquete.json`: manifiesto con semilla, rangos, huellas y órdenes de carga.

Determinista: la misma semilla y los mismos parámetros producen los mismos bytes.
"""
from __future__ import annotations

import argparse
import hashlib
import json
import random
from datetime import date, timedelta
from pathlib import Path

from . import catalogos
from .personas import GeneradorPersonas, Persona, sin_tildes

RAIZ = Path(__file__).resolve().parents[2]
CATALOGO_RPT = RAIZ / "data" / "catalogos" / "rpt" / "v1.rpt-2026.json"
PAQUETE_POR_DEFECTO = "paquete:ejemplo:vec:v1"
SEMILLA_POR_DEFECTO = 20260928

# Cabeceras que acepta hoy el importador (`internal/modules/bolsa/domain/importacionconvoca`).
CABECERAS_VEC = {
    "resumen": ["DNI/NIE", "Primer Apellido", "Segundo Apellido", "Nombre", "Turno",
                "Experiencia", "Formacion", "Total"],
    "detalle": ["DNI/NIE", "Primer Apellido", "Segundo Apellido", "Nombre", "Turno", "Grupo",
                "Descripcion del grupo", "Orden grupo", "Descripcion del merito",
                "Puntos autobaremacion", "Puntos tribunal", "Motivo"],
}
# Cabeceras literales de una exportación real de Convoca (solo la fila de títulos).
CABECERAS_CONVOCA = {
    "resumen": ["DNI/NIE enmascarado", "Primer Apellido", "Segundo Apellido", "Nombre", "Turno",
                "Experiencia", "Formación", "Total"],
    "detalle": ["DNI/NIE enmascarado", "Primer Apellido", "Segundo Apellido", "Nombre", "Turno", "Grupo",
                "Descripción del grupo", "Orden grupo", "Descripción del mérito",
                "Puntos autobaremación", "Puntos tribunal", "Motivo"],
}
HOJAS = {"resumen": "grupo de méritos (Tribunal) (1)", "detalle": "méritos (1)"}


def sufijo_paquete(paquete: str) -> str:
    """`paquete:ejemplo:vec:v1` → `ejemplo-vec-v1`, válido dentro de una referencia opaca."""
    partes = paquete.split(":")
    if len(partes) < 3 or partes[0] != "paquete" or not all(p.isalnum() for p in partes[1:]):
        raise ValueError("el identificador de paquete debe tener la forma paquete:<a>:<b>[:...]")
    return "-".join(partes[1:]).lower()


def categorias_rpt(numero: int, claves: list[str] | None) -> list[dict]:
    rpt = json.loads(CATALOGO_RPT.read_text(encoding="utf-8"))
    todas = [c for c in rpt["categorias"] if c["origen"] == "categoria"]
    if claves:
        por_clave = {c["clave"]: c for c in todas}
        faltan = [c for c in claves if c not in por_clave]
        if faltan:
            raise ValueError(f"categorías que no están en la RPT: {', '.join(faltan)}")
        return [por_clave[c] for c in claves]
    return todas[:numero]


def repartir(azar: random.Random, total: float, partes: int) -> list[float]:
    """Divide `total` en `partes` importes con dos decimales que suman exactamente `total`."""
    centimos = round(total * 100)
    cortes = sorted(azar.sample(range(1, centimos), partes - 1)) if centimos > partes else []
    limites = [0, *cortes, centimos]
    importes = [(limites[i + 1] - limites[i]) / 100 for i in range(len(limites) - 1)]
    return importes + [0.0] * (partes - len(importes))


def candidaturas_bolsa(azar: random.Random, personas: GeneradorPersonas, tamano: int,
                       ya_inscritas: list[Persona]) -> list[dict]:
    elegidas: list[Persona] = []
    for _ in range(tamano):
        repetibles = [p for p in ya_inscritas if p not in elegidas]
        if repetibles and azar.random() < 0.12:
            elegidas.append(azar.choice(repetibles))  # misma persona inscrita en otra bolsa
        else:
            elegidas.append(personas.nueva())
    filas = []
    for persona in elegidas:
        experiencia = round(azar.uniform(0.4, 20.0), 2)
        formacion = round(azar.uniform(0.2, 8.0), 2)
        filas.append({"persona": persona, "turno": "Discapacidad" if azar.random() < 0.07 else "Libre",
                      "experiencia": experiencia, "formacion": formacion,
                      "total": round(experiencia + formacion, 2)})
    filas.sort(key=lambda f: (-f["total"], sin_tildes(f["persona"].primer_apellido),
                              sin_tildes(f["persona"].segundo_apellido), sin_tildes(f["persona"].nombre)))
    return filas


def filas_detalle(azar: random.Random, fila: dict) -> list[list]:
    persona = fila["persona"]
    identidad = [persona.documento_enmascarado, persona.primer_apellido, persona.segundo_apellido,
                 persona.nombre, fila["turno"]]
    salida = []
    for grupo, descripcion, total, meritos in (
        ("EXP", "Experiencia profesional", fila["experiencia"], catalogos.MERITOS_EXPERIENCIA),
        ("FOR", "Formación", fila["formacion"], catalogos.MERITOS_FORMACION),
    ):
        elegidos = azar.sample(meritos, azar.randint(1, min(3, len(meritos))))
        for orden, (merito, tribunal) in enumerate(zip(elegidos, repartir(azar, total, len(elegidos))), start=1):
            autobaremo, motivo = tribunal, ""
            if azar.random() < 0.1:
                autobaremo = round(tribunal + azar.choice([0.25, 0.5, 1.0]), 2)
                motivo = azar.choice(catalogos.MOTIVOS_TRIBUNAL)
            salida.append(identidad + [grupo, descripcion, orden, merito, autobaremo, tribunal, motivo])
    return salida


def escribir_xls(ruta: Path, hoja: str, cabecera: list[str], filas: list[list]) -> None:
    import xlwt  # dependencia de desarrollo ya usada por scripts/generar_bolsa_demo.py

    libro = xlwt.Workbook(encoding="utf-8")
    h = libro.add_sheet(hoja)
    for columna, titulo in enumerate(cabecera):
        h.write(0, columna, titulo)
    for numero, celdas in enumerate(filas, start=1):
        for columna, valor in enumerate(celdas):
            if valor != "":
                h.write(numero, columna, valor)
    ruta.parent.mkdir(parents=True, exist_ok=True)
    libro.save(str(ruta))


def identidad_certificado(persona: Persona, rol: str, cargo: str) -> dict:
    """CN al estilo de un certificado de persona física: APELLIDOS NOMBRE - NIF."""
    apellidos = sin_tildes(f"{persona.primer_apellido} {persona.segundo_apellido}").upper()
    nombre = sin_tildes(persona.nombre).upper()
    cn = f"{apellidos} {nombre} - {persona.documento}"
    return {
        "rol": rol, "cargo": cargo, "persona_ref": persona.referencia,
        "display_name": persona.nombre_completo, "correo": persona.correo, "cn": cn,
        "openssl_subj": f"/C=ES/serialNumber=IDCES-{persona.documento}/GN={nombre}/SN={apellidos}/CN={cn}",
    }


def generar_identidades(personas: GeneradorPersonas, candidato: Persona, centros: int) -> list[dict]:
    def femenino(p: Persona, masculino: str, femenino_: str) -> str:
        return femenino_ if p.sexo == "M" else masculino

    salida = []
    for cargo_h, cargo_m in (("Jefe de Servicio de Recursos Humanos", "Jefa de Servicio de Recursos Humanos"),
                             ("Técnico de Recursos Humanos", "Técnica de Recursos Humanos"),
                             ("Técnico de Recursos Humanos", "Técnica de Recursos Humanos")):
        p = personas.nueva(30, 62)
        salida.append(identidad_certificado(p, "tecnico_rrhh", femenino(p, cargo_h, cargo_m)))
    for cargo_h, cargo_m in (("Interventor", "Interventora"), ("Técnico de Intervención", "Técnica de Intervención")):
        p = personas.nueva(32, 62)
        salida.append(identidad_certificado(p, "intervencion", femenino(p, cargo_h, cargo_m)))
    for indice in range(centros):  # mismo índice de centro que los expedientes
        p = personas.nueva(28, 60)
        salida.append(identidad_certificado(p, "solicitante_centro", femenino(p, "Responsable de unidad", "Responsable de unidad"))
                      | {"centro_indice": indice})
        p = personas.nueva(35, 62)
        salida.append(identidad_certificado(p, "ratificador_centro", femenino(p, "Director del centro", "Directora del centro"))
                      | {"centro_indice": indice})
    for _ in range(3):
        p = personas.nueva(24, 62)
        salida.append(identidad_certificado(p, "empleado", femenino(p, "Empleado público", "Empleada pública")))
    salida.append(identidad_certificado(candidato, "candidato_bolsa", femenino(candidato, "Candidato", "Candidata")))
    return salida


def generar_expedientes(azar: random.Random, personas: GeneradorPersonas, numero: int) -> list[dict]:
    salida = []
    inicio = date(2026, 10, 5)
    for indice in range(numero):
        sustituida = personas.nueva(26, 64)
        causa = catalogos.CAUSAS_SUSTITUCION[indice % len(catalogos.CAUSAS_SUSTITUCION)]
        desde = inicio + timedelta(days=7 * indice + azar.randint(0, 4))
        duracion = {0: 60, 1: 112, 2: 30, 3: 180, 4: 150, 5: 120}[indice % len(catalogos.CAUSAS_SUSTITUCION)]
        salida.append({
            "codigo": f"{indice + 1:02d}", "motivo_clave": "sustitucion",
            "centro_indice": indice, "categoria_indice": indice % 6,
            "persona_sustituida_ref": sustituida.referencia,
            "detalle": f"Sustitución de {sustituida.tratamiento} {sustituida.nombre_completo} {causa}.",
            "observaciones": azar.choice(catalogos.OBSERVACIONES_SOLICITUD),
            "observaciones_analisis": azar.choice(catalogos.OBSERVACIONES_ANALISIS),
            "jornada_porcentaje": azar.choice([100, 100, 100, 75, 50]),
            "periodo": {"inicio": f"{desde.isoformat()}T00:00:00Z",
                        "fin": f"{(desde + timedelta(days=duracion)).isoformat()}T00:00:00Z"},
        })
    return salida


def escribir_json(ruta: Path, datos) -> None:
    ruta.parent.mkdir(parents=True, exist_ok=True)
    ruta.write_text(json.dumps(datos, ensure_ascii=False, indent=1) + "\n", encoding="utf-8")


def generar(salida: Path, *, paquete: str = PAQUETE_POR_DEFECTO, semilla: int = SEMILLA_POR_DEFECTO,
            bolsas: int = 12, categorias: list[str] | None = None, minimo: int = 25, maximo: int = 45,
            expedientes: int = 12, cabeceras: str = "vec") -> dict:
    if not 1 <= minimo <= maximo or bolsas < 1 or expedientes < 0:
        raise ValueError("parámetros de tamaño incoherentes")
    sufijo = sufijo_paquete(paquete)
    azar = random.Random(semilla)
    personas = GeneradorPersonas(azar, f"persona:{sufijo}:")
    titulos = CABECERAS_VEC if cabeceras == "vec" else CABECERAS_CONVOCA
    inscritas: list[Persona] = []
    resumen_bolsas, ordenes = [], []
    candidato = None
    for categoria in categorias_rpt(bolsas, categorias):
        clave = categoria["clave"]
        filas = candidaturas_bolsa(azar, personas, azar.randint(minimo, maximo), inscritas)
        inscritas.extend(f["persona"] for f in filas if f["persona"] not in inscritas)
        candidato = candidato or filas[min(2, len(filas) - 1)]["persona"]
        carpeta = salida / "convoca" / clave
        escribir_xls(carpeta / "resumen.xls", HOJAS["resumen"], titulos["resumen"], [
            [f["persona"].documento_enmascarado, f["persona"].primer_apellido, f["persona"].segundo_apellido,
             f["persona"].nombre, f["turno"], f["experiencia"], f["formacion"], f["total"]] for f in filas])
        escribir_xls(carpeta / "detalle.xls", HOJAS["detalle"], titulos["detalle"],
                     [linea for f in filas for linea in filas_detalle(azar, f)])
        bolsa_ref = f"bolsa:{clave}:{sufijo}"
        resumen_bolsas.append({"categoria": clave, "denominacion": categoria["denominacion"],
                               "bolsa_ref": bolsa_ref, "candidaturas": len(filas),
                               "personas_ref": [f["persona"].referencia for f in filas]})
        for tipo in ("resumen", "detalle"):
            ordenes.append(f"vec-server importar-convoca --fichero convoca/{clave}/{tipo}.xls "
                           f"--categoria {clave} --bolsa-ref {bolsa_ref}")
        ordenes.append(f"vec-server constituir-bolsa --fichero convoca/{clave}/resumen.xls --categoria {clave}")
    escribir_json(salida / "contratacion" / "expedientes.json",
                  {"paquete": paquete, "expedientes": generar_expedientes(azar, personas, expedientes)})
    escribir_json(salida / "identidades.json",
                  {"paquete": paquete, "identidades": generar_identidades(personas, candidato, expedientes or 1)})
    escribir_json(salida / "personas.json",
                  {"paquete": paquete, "personas": [p.como_dict() for p in personas.personas]})
    ficheros = sorted(p for p in salida.rglob("*") if p.is_file() and p.name != "paquete.json")
    manifiesto = {
        "paquete": paquete, "semilla": semilla, "cabeceras": cabeceras,
        "procedencia": "Datos inventados para desarrollo y presentación; se retiran al pasar a producción.",
        "rangos": {"dni": "99000000–99999999", "nie": "Z9000000–Z9999999", "telefono": "79x xxx xxx",
                   "correo": "@correo.internal", "domicilio": "municipios de la provincia de Granada"},
        "resumen": {"bolsas": len(resumen_bolsas), "personas": len(personas.personas),
                    "candidaturas": sum(b["candidaturas"] for b in resumen_bolsas), "expedientes": expedientes},
        "bolsas": resumen_bolsas,
        "ordenes_carga": ordenes,
        "ficheros": {str(p.relative_to(salida)): hashlib.sha256(p.read_bytes()).hexdigest() for p in ficheros},
    }
    escribir_json(salida / "paquete.json", manifiesto)
    return manifiesto


def main(argv: list[str] | None = None) -> int:
    a = argparse.ArgumentParser(description=__doc__.splitlines()[0])
    a.add_argument("--salida", required=True, type=Path, help="directorio de salida (debe estar vacío o no existir)")
    a.add_argument("--paquete", default=PAQUETE_POR_DEFECTO)
    a.add_argument("--semilla", type=int, default=SEMILLA_POR_DEFECTO)
    a.add_argument("--bolsas", type=int, default=12, help="primeras N categorías de la RPT")
    a.add_argument("--categorias", nargs="*", help="claves RPT concretas (sustituye a --bolsas)")
    a.add_argument("--min-candidaturas", type=int, default=25)
    a.add_argument("--max-candidaturas", type=int, default=45)
    a.add_argument("--expedientes", type=int, default=12)
    a.add_argument("--cabeceras", choices=("vec", "convoca"), default="vec",
                   help="vec: las que acepta hoy el importador; convoca: literales de la exportación real")
    o = a.parse_args(argv)
    if o.salida.exists() and any(o.salida.iterdir()):
        a.error(f"{o.salida} no está vacío")
    m = generar(o.salida, paquete=o.paquete, semilla=o.semilla, bolsas=o.bolsas, categorias=o.categorias,
                minimo=o.min_candidaturas, maximo=o.max_candidaturas, expedientes=o.expedientes, cabeceras=o.cabeceras)
    print(f"{o.salida}: {m['resumen']}")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
