#!/usr/bin/env python3
"""Prepara el rol sintético de inscripción solo en la copia local O."""

import argparse
from datetime import datetime, timezone
import json
import os
from pathlib import Path
import re
import stat
import subprocess


ROL_ID = "candidato_bolsa_portal_historial_propio_desarrollo"
ROL_ANTERIOR = "rol:" + ROL_ID + ":v1"
ROL_NUEVO = "rol:" + ROL_ID + ":v2"
ACCIONES = (
    "bolsa.inscripcion.convocatorias.listar",
    "bolsa.inscripcion.convocatoria.consultar",
    "bolsa.inscripcion.propias.listar",
    "bolsa.inscripcion.propia.consultar",
    "bolsa.inscripcion.presentar",
)
FUENTE_HUELLA = "07bf028039044c91a23f09a2d5d5fe0d2af254e2fe85aa7fbbacea7c473487fe"
OPERADOR = "operacion:ensayo:aspirante"
REF = re.compile(r"^[a-z]+_[A-Za-z0-9_-]{22,128}$")


def exigir(condicion, clave):
    if not condicion:
        raise ValueError(clave)


def privado(ruta, directorio=False):
    dato = ruta.lstat()
    exigir(dato.st_uid == os.getuid() and not ruta.is_symlink() and not dato.st_mode & 0o077,
           "ruta_no_privada")
    exigir(stat.S_ISDIR(dato.st_mode) if directorio else stat.S_ISREG(dato.st_mode),
           "tipo_de_ruta_invalido")


def literal(valor):
    exigir(isinstance(valor, str) and "\x00" not in valor, "texto_invalido")
    return "'" + valor.replace("'", "''") + "'"


def psql(pg, sql):
    resultado = subprocess.run(
        ["docker", "exec", "-i", pg, "psql", "-U", "postgres", "-d", "postgres",
         "-X", "-w", "-qAt", "-v", "ON_ERROR_STOP=1", "-v", "VERBOSITY=sqlstate"],
        input=sql, text=True, capture_output=True, timeout=30, check=False,
    )
    if resultado.returncode:
        codigo = re.search(r"\b[0-9]{2}[A-Z0-9]{3}\b", resultado.stderr)
        raise ValueError("sql_copia_rechazada_" + (codigo.group(0) if codigo else "sin_sqlstate"))
    return resultado.stdout.strip()


def estado(pg, perfil):
    sql = ("SELECT jsonb_build_object('rol_v1',r1.documento,'rol_v2',r2.documento,"
           "'control_v1',c1.documento,'control_v2',c2.documento,"
           "'asignacion',a.documento,'asignacion_huella',a.huella_sha256)::text "
           "FROM vec_autorizacion.version_rol r1 "
           "JOIN vec_autorizacion.control_vigencia_version_rol_actual p1 "
           "ON p1.version_rol_ref=r1.version_rol_ref "
           "JOIN vec_autorizacion.control_vigencia_version_rol c1 "
           "ON c1.version_rol_ref=p1.version_rol_ref AND c1.revision=p1.revision "
           "JOIN vec_autorizacion.asignacion_perfil_actual_externa ap ON ap.perfil_activo_ref=" + literal(perfil) + " "
           "JOIN vec_autorizacion.asignacion_perfil_externa a USING(asignacion_ref) "
           "LEFT JOIN vec_autorizacion.version_rol r2 ON r2.version_rol_ref=" + literal(ROL_NUEVO) + " "
           "LEFT JOIN vec_autorizacion.control_vigencia_version_rol_actual p2 "
           "ON p2.version_rol_ref=r2.version_rol_ref "
           "LEFT JOIN vec_autorizacion.control_vigencia_version_rol c2 "
           "ON c2.version_rol_ref=p2.version_rol_ref AND c2.revision=p2.revision "
           "WHERE r1.version_rol_ref=" + literal(ROL_ANTERIOR) + ";")
    dato = psql(pg, sql)
    exigir(bool(dato), "base_rol_o_asignacion_ausente")
    return json.loads(dato)


def fuentes(repo):
    archivo = repo / "deploy/principal/inscripcion_recorrido_20261013/inscripcion_bolsa_actos_v1.json"
    fuente = json.loads(archivo.read_text())
    entradas = [e for e in fuente["entradas"] if e["concesion"]["accion"] in ACCIONES]
    exigir(fuente["huella_sha256"] == FUENTE_HUELLA and
           [e["concesion"]["accion"] for e in entradas] == list(ACCIONES) and
           all(e["fuente_huella_sha256"] == FUENTE_HUELLA and
               e["dimensiones_ambito"] == ["candidato_ref"] and
               e["clase_control"] == "ordinario" for e in entradas),
           "fuente_inscripcion_divergente")
    return [e["concesion"] for e in entradas]


def artefacto_privado(ruta, dato, aplicar):
    if ruta.exists():
        privado(ruta)
        return json.loads(ruta.read_text())
    exigir(aplicar, "artefacto_privado_pendiente")
    with ruta.open("x") as salida:
        json.dump(dato, salida, ensure_ascii=False, separators=(",", ":"))
        salida.write("\n")
    ruta.chmod(0o600)
    return dato


def preparar(ensayo, actual, concesiones, aplicar):
    directorio = ensayo / "incorporacion/actos"
    privado(directorio, True)
    marca = datetime.now(timezone.utc).replace(microsecond=0).strftime("%Y-%m-%dT%H:%M:%SZ")
    rol = dict(actual["rol_v1"])
    rol.update(version=2, concesiones=rol["concesiones"] + concesiones,
               publicada_por=OPERADOR, publicada_en=marca)
    rol = artefacto_privado(directorio / "rol-candidato-inscripcion-v2.json", rol, aplicar)
    control = dict(actual["control_v1"])
    control.update(version_rol_ref=ROL_NUEVO, revision=1, actualizado_por=OPERADOR,
                   actualizado_en=rol["publicada_en"])
    control = artefacto_privado(directorio / "control-candidato-inscripcion-v2.json", control, aplicar)
    asignacion = dict(actual["asignacion"])
    asignacion.update(version=actual["asignacion"]["version"] + 1,
                      version_rol_ref=ROL_NUEVO, emitida_por=OPERADOR,
                      emitida_en=rol["publicada_en"], vigente_desde=rol["publicada_en"])
    asignacion = artefacto_privado(directorio /
                                   ("asignacion-candidato-inscripcion-v" +
                                    str(asignacion["version"]) + ".json"), asignacion, aplicar)
    exigir(rol["rol_id"] == ROL_ID and rol["version"] == 2 and
           [x["accion"] for x in rol["concesiones"][-5:]] == list(ACCIONES) and
           len(rol["concesiones"]) == len(actual["rol_v1"]["concesiones"]) + 5 and
           control["version_rol_ref"] == ROL_NUEVO and control["revision"] == 1 and
           asignacion["version_rol_ref"] == ROL_NUEVO and
           asignacion["version"] == actual["asignacion"]["version"] + 1 and
           asignacion["principal_id"] == actual["asignacion"]["principal_id"] and
           asignacion["perfil_activo_ref"] == actual["asignacion"]["perfil_activo_ref"] and
           asignacion["ambitos"] == actual["asignacion"]["ambitos"],
           "artefactos_inscripcion_divergentes")
    return rol, control, asignacion


def sql_publicacion(perfil, persona, candidato, rol, control, asignacion):
    r, c, a = (literal(json.dumps(x, ensure_ascii=False, separators=(",", ":")))
               for x in (rol, control, asignacion))
    version_previa = asignacion["version"] - 1
    asignacion_previa = "asignacion:" + asignacion["asignacion_id"] + ":v" + str(version_previa)
    asignacion_nueva = "asignacion:" + asignacion["asignacion_id"] + ":v" + str(asignacion["version"])
    retorno = """\nSELECT 'rol_v2',count(*) FROM vec_autorizacion.version_rol WHERE version_rol_ref='rol:candidato_bolsa_portal_historial_propio_desarrollo:v2';
SELECT 'asignacion_actual',count(*) FROM vec_autorizacion.asignacion_perfil_actual_externa p JOIN vec_autorizacion.asignacion_perfil_externa a USING(asignacion_ref) WHERE p.perfil_activo_ref=%s AND a.version=%s;
""" % (literal(perfil), str(asignacion["version"]))
    return f"""BEGIN ISOLATION LEVEL SERIALIZABLE;
SET LOCAL search_path=pg_catalog;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='3s';
SET LOCAL statement_timeout='20s';
SELECT pg_advisory_xact_lock(hashtextextended('vec:ensayo:aspirante:inscripcion',0));
DO $pre$ BEGIN
 IF EXISTS(SELECT 1 FROM vec_autorizacion.version_rol WHERE version_rol_ref={literal(ROL_NUEVO)})
    OR NOT EXISTS(SELECT 1 FROM vec_autorizacion.control_vigencia_version_rol_actual WHERE version_rol_ref={literal(ROL_ANTERIOR)})
    OR EXISTS(SELECT 1 FROM vec_autorizacion.asignacion_perfil_externa WHERE asignacion_ref={literal(asignacion_nueva)})
    OR NOT EXISTS(SELECT 1 FROM vec_autorizacion.asignacion_perfil_actual_externa WHERE perfil_activo_ref={literal(perfil)} AND asignacion_ref={literal(asignacion_previa)})
    OR vec_contexto_actor_v1.acreditar_candidato_externo_v1({literal(persona)},{literal(perfil)},{literal(candidato)}) IS NOT TRUE
 THEN RAISE EXCEPTION 'preimagen_inscripcion_copia_divergente' USING ERRCODE='55000'; END IF;
END $pre$;
INSERT INTO vec_autorizacion.version_rol(version_rol_ref,rol_id,version,huella_sha256,publicada_en,documento)
SELECT {literal(ROL_NUEVO)},{literal(ROL_ID)},2,encode(sha256(convert_to(vec_autorizacion.canon_version_rol_admin_v1({r}::jsonb),'UTF8')),'hex'),
 ({r}::jsonb->>'publicada_en')::timestamptz,{r}::jsonb;
INSERT INTO vec_autorizacion.control_vigencia_version_rol(version_rol_ref,revision,estado,huella_sha256,actualizado_en,documento)
SELECT {literal(ROL_NUEVO)},1,'habilitada',encode(sha256(convert_to(vec_autorizacion.canon_control_rol_admin_v1({c}::jsonb),'UTF8')),'hex'),
 ({c}::jsonb->>'actualizado_en')::timestamptz,{c}::jsonb;
INSERT INTO vec_autorizacion.control_vigencia_version_rol_actual(version_rol_ref,revision,actualizada_en,actualizada_por,acto_ref)
VALUES({literal(ROL_NUEVO)},1,({c}::jsonb->>'actualizado_en')::timestamptz,{literal(OPERADOR)},'acto:ensayo:inscripcion:rol:v2');
INSERT INTO vec_autorizacion.asignacion_perfil_externa(asignacion_ref,asignacion_id,version,perfil_activo_ref,principal_id,version_rol_ref,huella_sha256,emitida_en,documento)
SELECT {literal(asignacion_nueva)},({a}::jsonb->>'asignacion_id'),{asignacion["version"]},{literal(perfil)},{literal(persona)},{literal(ROL_NUEVO)},
 encode(sha256(convert_to(vec_autorizacion.canon_asignacion_perfil_admin_v1({a}::jsonb),'UTF8')),'hex'),
 ({a}::jsonb->>'emitida_en')::timestamptz,{a}::jsonb;
UPDATE vec_autorizacion.asignacion_perfil_actual_externa SET asignacion_ref={literal(asignacion_nueva)},
 actualizada_en=clock_timestamp(),actualizada_por={literal(OPERADOR)},acto_ref='acto:ensayo:inscripcion:asignacion:v{asignacion["version"]}'
WHERE perfil_activo_ref={literal(perfil)} AND asignacion_ref={literal(asignacion_previa)};
DO $post$ BEGIN IF (SELECT count(*) FROM vec_autorizacion.asignacion_perfil_actual_externa
  WHERE perfil_activo_ref={literal(perfil)} AND asignacion_ref={literal(asignacion_nueva)})<>1
 THEN RAISE EXCEPTION 'postimagen_inscripcion_copia_divergente' USING ERRCODE='55000'; END IF; END $post$;
COMMIT;
""" + retorno


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--ensayo", type=Path, required=True)
    parser.add_argument("--pg", required=True)
    parser.add_argument("--aplicar", action="store_true")
    args = parser.parse_args()
    exigir(args.pg.startswith("vec-codexo-aspirante-pg-") and
           re.fullmatch(r"[a-z0-9-]+", args.pg) is not None,
           "solo_clon_aspirante")
    repo = Path(__file__).resolve().parents[4]
    exigir(args.ensayo.is_absolute() and repo not in args.ensayo.parents,
           "destino_no_privado")
    privado(args.ensayo, True)
    identidad = json.loads((args.ensayo / "externo/identidad/bolsa-candidato.json").read_text())
    perfil, persona, candidato = (identidad[k] for k in
                                  ("perfil_ref", "persona_ref", "candidato_ref"))
    exigir(all(REF.fullmatch(x) for x in (perfil, persona, candidato)), "identidad_invalida")
    concesiones = fuentes(repo)
    actual = estado(args.pg, perfil)
    if actual["rol_v2"] is not None:
        exigir(actual["control_v2"] is not None and actual["asignacion"]["version"] >= 2 and
               actual["asignacion"]["version_rol_ref"] == ROL_NUEVO and
               [x["accion"] for x in actual["rol_v2"]["concesiones"][-5:]] == list(ACCIONES),
               "inscripcion_copia_divergente")
        print("inscripcion_externa=preparada")
        return
    exigir(actual["asignacion"]["version"] in (1, 2) and
           actual["asignacion"]["version_rol_ref"] == ROL_ANTERIOR,
           "asignacion_base_no_lista")
    rol, control, asignacion = preparar(args.ensayo, actual, concesiones, args.aplicar)
    exigir(args.aplicar, "inscripcion_externa_pendiente")
    resultado = psql(args.pg, sql_publicacion(perfil, persona, candidato, rol, control, asignacion))
    exigir("rol_v2|1" in resultado and "asignacion_actual|1" in resultado,
           "recibo_inscripcion_no_cotejado")
    print("inscripcion_externa=preparada")


if __name__ == "__main__":
    try:
        main()
    except (OSError, ValueError, KeyError, TypeError, json.JSONDecodeError, subprocess.TimeoutExpired) as error:
        raise SystemExit("inscripcion_externa=error clave=" + str(error).split(":", 1)[0]) from None
