#!/usr/bin/env python3
"""Instala deltas canónicos de identidad/consulta en una preimagen acreditada.

No reconstruye una base histórica desde cero. Exige que la base ya contenga
la cadena previa, ensaya todo con ROLLBACK y aplica idénticos bytes una vez.
"""

from __future__ import annotations

import argparse
import hashlib
import json
from pathlib import Path
import re
import sys

from aprovisionar import atomic_private, private_dir, psql, read_private, fail


SOURCE = Path(__file__).resolve().parents[3] / "deploy" / "postgresql"
DELTAS = (
    "contexto_actor_v1/migraciones/000006_vinculos_efectivos_temporales.up.sql",
    "identidad_sesiones_v1/migraciones/000006_politica_certificado_personal_desarrollo.up.sql",
    "personal/migraciones/000010a_lectura_incorporacion_certificado_desarrollo.up.sql",
    "contratacion_temporal/migraciones_identidad/000002_consulta_rrhh_certificado_desarrollo.up.sql",
    "autorizacion_atestada_v3/migraciones/000050a_preflight_material_interno.up.sql",
)


def body(contents: bytes) -> str:
    raw = contents.decode("utf-8")
    lines = raw.splitlines(keepends=True)
    starts = [i for i, line in enumerate(lines) if line.strip() == "BEGIN;"]
    ends = [i for i, line in enumerate(lines) if line.strip() == "COMMIT;"]
    if len(starts) != 1 or len(ends) != 1 or starts[0] >= ends[0]:
        fail("migración canónica sin envoltura transaccional simple")
    for i, line in enumerate(lines):
        if line.lstrip().startswith(("\\i ", "\\ir ")):
            fail("migración con componente externo: requiere instalación propia")
        if i in (starts[0], ends[0]) or line.strip() == r"\set ON_ERROR_STOP on":
            lines[i] = ""
    return "".join(lines)


def build(finish: str) -> tuple[str, dict[str, str]]:
    sql = ["\\set ON_ERROR_STOP on", "BEGIN;", "SET LOCAL search_path = pg_catalog;",
           "SET LOCAL lock_timeout = '5s';", "SET LOCAL statement_timeout = '180s';",
           """DO $preimagen$ BEGIN
 IF current_setting('server_version_num')::integer / 10000 <> 18
 OR current_setting('server_encoding') <> 'UTF8'
 OR to_regnamespace('vec_identidad_sesiones_v1') IS NULL
 OR to_regnamespace('vec_contexto_actor_v1') IS NULL
 OR to_regnamespace('vec_autorizacion') IS NULL
 OR to_regnamespace('vec_autorizacion_atestada_v3') IS NULL
 OR to_regnamespace('vec_personal') IS NULL
 OR to_regnamespace('vec_contratacion_temporal') IS NULL
 OR to_regclass('vec_personal.org_nodo_historia') IS NULL
 OR NOT EXISTS (SELECT 1 FROM pg_catalog.pg_class c
   WHERE c.oid='vec_personal.org_nodo_historia'::regclass
     AND c.relowner='vec_personal_propietario'::regrole
     AND c.relrowsecurity AND c.relforcerowsecurity)
 OR to_regprocedure('vec_personal.consultar_organizacion_historica_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea,bytea,bytea)') IS NULL
 OR to_regrole('vec_contratacion_temporal_registrador_frontera') IS NULL
 OR to_regprocedure('vec_contratacion_temporal.registrar_auditoria_frontera_ruta_exacta_v1(text,text,text,text,text)') IS NULL
 OR to_regprocedure('vec_contexto_actor_v1.acreditar_uso_registro_contexto_actor_v2(text,text,text,text,text,text,numeric,text,numeric,text,numeric,text,numeric,text,text,timestamptz,timestamptz)') IS NULL
 OR to_regprocedure('vec_identidad_sesiones_v1.revalidar_contexto_corporativo_rrhh_v1(text,text)') IS NULL
 OR to_regclass('vec_identidad_sesiones_v1.politica_certificado_personal_desarrollo_v1') IS NOT NULL
 OR to_regrole('vec_autorizacion_atestada_v3_preflight_interno') IS NOT NULL
 OR to_regprocedure('vec_autorizacion_atestada_v3.comprobar_material_emision_interna_v1(jsonb)') IS NOT NULL
 THEN RAISE EXCEPTION 'preimagen interna ausente o delta ya instalado' USING ERRCODE='55000'; END IF;
END $preimagen$;"""]
    hashes = {}
    for relative in DELTAS:
        path = SOURCE / relative
        if not path.is_file() or path.is_symlink() or SOURCE not in path.resolve().parents:
            fail("fuente canónica de migración ausente")
        contents = path.read_bytes()
        hashes[relative] = hashlib.sha256(contents).hexdigest()
        sql += [f"-- INICIO {relative}", body(contents), f"-- FIN {relative}"]
    sql.append(finish + ";")
    return "\n".join(sql), hashes


def ejecutar_con_ensayo(ejecutor) -> dict[str, str]:
    dry, hashes = build("ROLLBACK")
    ejecutor(dry)
    # La segunda lectura ocurre DESPUÉS del ensayo. Si la fuente cambia,
    # no se ejecuta COMMIT. En cada lectura SQL y huella usan los mismos bytes.
    apply, hashes_again = build("COMMIT")
    if hashes != hashes_again:
        fail("fuente canónica cambió entre ensayo y aplicación")
    ejecutor(apply)
    return hashes


def main() -> None:
    parser = argparse.ArgumentParser(description="Deltas de composición interna sobre preimagen canónica")
    parser.add_argument("--material-dir", required=True)
    parser.add_argument("--database", required=True)
    parser.add_argument("--admin-user", required=True)
    parser.add_argument("--pg-container")
    parser.add_argument("--container-engine", choices=("docker", "podman"), default="docker")
    args = parser.parse_args()
    if not re.fullmatch(r"[A-Za-z][A-Za-z0-9_-]{0,62}", args.database) or not re.fullmatch(r"[A-Za-z][A-Za-z0-9_-]{0,62}", args.admin_user):
        fail("base o usuario administrativo inválido")
    material = private_dir(args.material_dir)
    receipt = material / "esquema-interno.json"
    if receipt.exists():
        read_private(receipt)
        fail("delta ya registrado; no se reaplica migración con historia")
    hashes = ejecutar_con_ensayo(
        lambda sql: psql(sql, args.pg_container, args.container_engine,
                         args.database, args.admin_user))
    atomic_private(receipt, (json.dumps({"version": 1, "database": args.database,
                                        "migraciones_sha256": hashes}, sort_keys=True,
                                       separators=(",", ":")) + "\n").encode())
    print("Deltas canónicos instalados una vez tras ensayo ROLLBACK; recibo privado escrito")


if __name__ == "__main__":
    try:
        main()
    except (OSError, ValueError, RuntimeError) as error:
        print(f"vec-interno: {error}", file=sys.stderr)
        sys.exit(1)
