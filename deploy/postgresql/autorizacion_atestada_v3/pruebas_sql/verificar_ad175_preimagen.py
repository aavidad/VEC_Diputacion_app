#!/usr/bin/env python3
"""Comprueba fuera de PostgreSQL las mediciones K y la sustitución reversible AD175."""

import hashlib
import json
import pathlib
import sys


def sha(texto: str) -> str:
    return hashlib.sha256(texto.encode("utf-8")).hexdigest()


def bloque(sql: str, nombre: str) -> str:
    inicio, fin = f"{nombre} text:=${nombre}$", f"${nombre}$;"
    if sql.count(inicio) != 1:
        raise ValueError(f"marca {nombre} ausente o repetida")
    resto = sql.split(inicio, 1)[1]
    if resto.count(fin) < 1:
        raise ValueError(f"cierre {nombre} ausente")
    return resto.split(fin, 1)[0]


def verificar(nucleo: pathlib.Path, migracion: pathlib.Path) -> None:
    medido = json.loads(nucleo.read_text(encoding="utf-8"))
    sql = migracion.read_text(encoding="utf-8")
    if (medido["name"] != "vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_interna(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)"
            or medido["owner"] != "vec_autorizacion_atestada_v3_propietario"
            or medido["acl"] != ["vec_autorizacion_atestada_v3_propietario=X/vec_autorizacion_atestada_v3_propietario"]
            or medido["config"] != ["search_path=pg_catalog, pg_temp", "lock_timeout=2s"]
            or sha(medido["definition"]) != medido["def_sha"]
            or sha(medido["source"]) != medido["src_sha"]
            or medido["def_sha"] not in sql or medido["src_sha"] not in sql):
        raise ValueError("núcleo medido o guarda AD175 divergente")
    cambios = [(bloque(sql, a), bloque(sql, b)) for a, b in
               (("runtime", "runtime_nuevo"), ("excl", "excl_nuevo"), ("marca", "extension"))]
    for antiguo, nuevo in cambios:
        if medido["definition"].count(antiguo) != 1 or medido["source"].count(antiguo) != 1:
            raise ValueError("ancla ausente o ambigua en K")
        if "exportacion_servicios_propios" not in nuevo:
            raise ValueError("extensión sin perfil propio")
    candidata = medido["definition"]
    for antiguo, nuevo in cambios:
        candidata = candidata.replace(antiguo, nuevo, 1)
    recuperada = candidata
    for antiguo, nuevo in reversed(cambios):
        if recuperada.count(nuevo) != 1:
            raise ValueError("extensión ambigua")
        recuperada = recuperada.replace(nuevo, antiguo, 1)
    if recuperada != medido["definition"] or candidata.count("exportacion_servicios_propios") != 4:
        raise ValueError("sustitución AD175 no reversible o perfil duplicado")
    if (sql.count("esperada_audiencia_sha256 constant text:=NULL;") != 2
            or not sql.index("DO $pre$") < sql.index("IF esperada_audiencia_sha256 IS NULL") < sql.index("EXECUTE nuevo;")
            or "vec_personal.vinculo_propio.crn11.v1" in sql):
        raise ValueError("audiencia sin PARO previo al DDL o dependencia CRN11 ajena")
    print("AD175-PREIMAGEN-ESTATICA-OK; audiencia de clave pendiente de medición, SQL no ensayado")


if __name__ == "__main__":
    if len(sys.argv) != 3:
        raise SystemExit("uso: verificar_ad175_preimagen.py nucleo.json 000175.up.sql")
    verificar(*(pathlib.Path(p) for p in sys.argv[1:]))
