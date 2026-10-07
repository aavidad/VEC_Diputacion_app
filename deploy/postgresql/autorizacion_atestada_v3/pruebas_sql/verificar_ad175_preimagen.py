#!/usr/bin/env python3
"""Comprueba la preimagen PG18 posterior a AD211 y el parche reversible AD175."""

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
    operaciones = (
        "nuevo:=replace(original,runtime,runtime_nuevo);",
        "nuevo:=replace(nuevo,excl,excl_nuevo);",
        "nuevo:=replace(nuevo,marca,extension||marca);",
        "EXECUTE nuevo;",
    )
    if any(sql.count(op) != 1 for op in operaciones) or [sql.index(op) for op in operaciones] != sorted(sql.index(op) for op in operaciones):
        raise ValueError("secuencia de sustituciones AD175 divergente")
    if sql.count("replace(replace(replace(actual,extension||marca,marca),excl_nuevo,excl),runtime_nuevo,runtime)") != 1:
        raise ValueError("inversión SQL AD175 divergente")
    marca = bloque(sql, "marca")
    cambios = [
        (bloque(sql, "runtime"), bloque(sql, "runtime_nuevo")),
        (bloque(sql, "excl"), bloque(sql, "excl_nuevo")),
        (marca, bloque(sql, "extension") + marca),
    ]
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
    check_sha = "d54d76c9f34009f3313e5f236b78cfc173e6547ec4bfbebe2be3785cd13f4a17"
    if (sql.count(f"esperada_audiencia_sha256 constant text:='{check_sha}';") != 2
            or sql.count("pg_get_constraintdef(c.oid,false)") != 3
            or not sql.index("LOCK TABLE vec_autorizacion_atestada_v3.clave_capacidad_version")
                < sql.index("DO $pre$") < sql.index("h:=CASE WHEN d IS NULL") < sql.index("EXECUTE nuevo;")
            or "esperada_audiencia_sha256 constant text:=NULL" in sql
            or "vec_personal.vinculo_propio.crn11.v1" in sql):
        raise ValueError("audiencia no medida antes de DDL o dependencia ajena")
    if ("nueva:='CHECK (('||substr(d,8,length(d)-8)||') OR audiencia_consumo = " not in sql
            or sql.count("vec_personal.registro_empleado.ficha_propia.servicios.exportar.v1") < 3):
        raise ValueError("CHECK de exportación no está reanclado al formato actual")
    print("AD175-PREIMAGEN-ESTATICA-OK; PG18 POST211, sin ensayo SQL")


if __name__ == "__main__":
    if len(sys.argv) != 3:
        raise SystemExit("uso: verificar_ad175_preimagen.py nucleo.json 000175.up.sql")
    verificar(*(pathlib.Path(p) for p in sys.argv[1:]))
