#!/usr/bin/env python3
"""Prepara el rol de comprobantes de Méritos en la copia ficticia de O."""

import argparse
from datetime import datetime, timezone
import hashlib
import json
from pathlib import Path
import re

from publicar_roles_meritos_copia import exigir, literal, psql


def jlit(dato):
    return literal(json.dumps(dato, ensure_ascii=False, separators=(",", ":"))) + "::jsonb"


def consultar(pg, perfil):
    texto = psql(pg, "SELECT jsonb_build_object('asignacion',a.documento,'rol',r.documento)::text "
                 "FROM vec_autorizacion.asignacion_perfil_actual_externa p "
                 "JOIN vec_autorizacion.asignacion_perfil_externa a USING(asignacion_ref) "
                 "JOIN vec_autorizacion.version_rol r USING(version_rol_ref) "
                 "WHERE p.perfil_activo_ref=" + literal(perfil) + ";")
    exigir(bool(texto), "asignacion_meritos_ausente")
    return json.loads(texto)


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--ensayo", type=Path, required=True)
    parser.add_argument("--pg", required=True)
    parser.add_argument("--aplicar", action="store_true")
    args = parser.parse_args()
    repo = Path(__file__).resolve().parents[4]
    exigir(re.fullmatch(r"vec-codexo-aspirante-pg-[a-z0-9-]+", args.pg) and
           args.ensayo.is_absolute() and repo not in args.ensayo.parents and
           not args.ensayo.is_symlink() and not args.ensayo.stat().st_mode & 0o077,
           "solo_clon_privado")
    fuente = json.loads((repo / "deploy/principal/fuentes/meritos_cartera_propia_v2.json").read_text())
    exigir(fuente["version"] == 2 and
           hashlib.sha256(fuente["entradas_canon"].encode()).hexdigest() == fuente["huella_sha256"],
           "fuente_comprobantes_divergente")
    canon = json.loads(fuente["entradas_canon"])
    exigir(canon == [{k: v for k, v in entrada.items() if k != "fuente_huella_sha256"}
                      for entrada in fuente["entradas"]], "entradas_comprobantes_divergentes")
    concesiones = [entrada["concesion"] for entrada in fuente["entradas"]]
    exigir(len(concesiones) == 5 and all(entrada["fuente_huella_sha256"] == fuente["huella_sha256"]
                                       for entrada in fuente["entradas"]), "concesiones_divergentes")
    config = json.loads((args.ensayo / "externo/identidad/meritos-cartera-externa.json").read_text())
    exigir(len(config["cuentas"]) == 1, "cuenta_meritos_ambigua")
    cuenta = config["cuentas"][0]
    actual = consultar(args.pg, cuenta["perfil_ref"])
    if actual["rol"]["concesiones"] == concesiones:
        exigir(actual["rol"]["rol_id"] == "persona_meritos_cartera", "rol_comprobantes_divergente")
        print("permisos_comprobantes=preparados")
        return
    exigir(args.aplicar and actual["rol"]["rol_id"] == "persona_meritos_cartera" and
           actual["rol"]["version"] == 1 and len(actual["rol"]["concesiones"]) == 3,
           "preimagen_comprobantes_pendiente_o_divergente")
    marca = datetime.now(timezone.utc).replace(microsecond=0).strftime("%Y-%m-%dT%H:%M:%SZ")
    rol_ref = "rol:persona_meritos_cartera:v2"
    rol = dict(actual["rol"], version=2, concesiones=concesiones,
               publicada_en=marca, publicada_por="operacion:ensayo:aspirante")
    asignacion = dict(actual["asignacion"], version=actual["asignacion"]["version"] + 1,
                      version_rol_ref=rol_ref, emitida_en=marca,
                      emitida_por="operacion:ensayo:aspirante")
    control = {"version_rol_ref": rol_ref, "revision": 1, "estado": "habilitada",
               "actualizado_por": "operacion:ensayo:aspirante", "actualizado_en": marca}
    r, a, c = jlit(rol), jlit(asignacion), jlit(control)
    previa = "asignacion:" + actual["asignacion"]["asignacion_id"] + ":v" + str(actual["asignacion"]["version"])
    nueva = "asignacion:" + asignacion["asignacion_id"] + ":v" + str(asignacion["version"])
    sql = f"""BEGIN ISOLATION LEVEL SERIALIZABLE;
SET LOCAL search_path=pg_catalog;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='3s';
SELECT pg_advisory_xact_lock(hashtextextended('vec:ensayo:meritos:comprobantes',0));
DO $pre$ BEGIN
 IF vec_autorizacion.rol_meritos_cartera_cinco_v2({r}) IS NOT TRUE OR
    NOT EXISTS(SELECT 1 FROM vec_autorizacion.asignacion_perfil_actual_externa
     WHERE perfil_activo_ref={literal(cuenta['perfil_ref'])} AND asignacion_ref={literal(previa)})
 THEN RAISE EXCEPTION 'preimagen_comprobantes_divergente' USING ERRCODE='55000'; END IF;
END $pre$;
INSERT INTO vec_autorizacion.version_rol(version_rol_ref,rol_id,version,huella_sha256,publicada_en,documento)
 SELECT {literal(rol_ref)},'persona_meritos_cartera',2,
 encode(sha256(convert_to(vec_autorizacion.canon_version_rol_admin_v1({r}),'UTF8')),'hex'),
 ({r}->>'publicada_en')::timestamptz,{r};
INSERT INTO vec_autorizacion.control_vigencia_version_rol(version_rol_ref,revision,estado,huella_sha256,actualizado_en,documento)
 SELECT {literal(rol_ref)},1,'habilitada',
 encode(sha256(convert_to(vec_autorizacion.canon_control_rol_admin_v1({c}),'UTF8')),'hex'),
 ({c}->>'actualizado_en')::timestamptz,{c};
INSERT INTO vec_autorizacion.control_vigencia_version_rol_actual
 VALUES({literal(rol_ref)},1,{literal(marca)}::timestamptz,'operacion:ensayo:aspirante','acto:ensayo:meritos:comprobantes');
INSERT INTO vec_autorizacion.asignacion_perfil_externa(asignacion_ref,asignacion_id,version,perfil_activo_ref,principal_id,version_rol_ref,huella_sha256,emitida_en,documento)
 SELECT {literal(nueva)},({a}->>'asignacion_id'),{asignacion['version']},({a}->>'perfil_activo_ref'),
 ({a}->>'principal_id'),{literal(rol_ref)},
 encode(sha256(convert_to(vec_autorizacion.canon_asignacion_perfil_admin_v1({a}),'UTF8')),'hex'),
 ({a}->>'emitida_en')::timestamptz,{a};
UPDATE vec_autorizacion.asignacion_perfil_actual_externa SET asignacion_ref={literal(nueva)},
 actualizada_en={literal(marca)}::timestamptz,actualizada_por='operacion:ensayo:aspirante',
 acto_ref='acto:ensayo:meritos:comprobantes'
 WHERE perfil_activo_ref={literal(cuenta['perfil_ref'])} AND asignacion_ref={literal(previa)};
COMMIT;"""
    psql(args.pg, sql)
    exigir(consultar(args.pg, cuenta["perfil_ref"]) == {"asignacion": asignacion, "rol": rol},
           "postimagen_comprobantes_divergente")
    print("permisos_comprobantes=preparados")


if __name__ == "__main__":
    try:
        main()
    except (OSError, ValueError, KeyError, TypeError, json.JSONDecodeError) as error:
        raise SystemExit("permisos_comprobantes=error clave=" + str(error).split(":", 1)[0]) from None
