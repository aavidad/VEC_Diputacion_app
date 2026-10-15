#!/usr/bin/env python3
"""Prepara en PG O solo los permisos propios de Preferencias y Alegaciones."""

import argparse
from datetime import datetime, timedelta, timezone
import json
from pathlib import Path
import re
import subprocess


ROL = "candidato_bolsa_portal_historial_propio_desarrollo"
OPERADOR = "operacion:ensayo:aspirante"
PREFERENCIAS = ("bolsa.preferencias_bolsas.consultar_propias",
                "bolsa.preferencias_bolsas.ordenar_propias")
ALEGACIONES = ("bolsa.alegacion.propias.consultar",
               "bolsa.inscripcion.alegacion.presentar")


def exigir(condicion, clave):
    if not condicion:
        raise ValueError(clave)


def lit(valor):
    exigir(isinstance(valor, str) and "\x00" not in valor, "texto_invalido")
    return "'" + valor.replace("'", "''") + "'"


def jlit(valor):
    return lit(json.dumps(valor, ensure_ascii=False, separators=(",", ":"))) + "::jsonb"


def psql(pg, sql):
    r = subprocess.run(["docker", "exec", "-i", pg, "psql", "-U", "postgres", "-d", "postgres",
                        "-X", "-w", "-qAt", "-v", "ON_ERROR_STOP=1", "-v", "VERBOSITY=sqlstate"],
                       input=sql, text=True, capture_output=True, timeout=30, check=False)
    if r.returncode:
        codigo = re.search(r"\b[0-9]{2}[A-Z0-9]{3}\b", r.stderr)
        raise ValueError("sql_alegaciones_rechazado_" + (codigo.group(0) if codigo else "sin_sqlstate"))
    return r.stdout.strip()


def actual(pg, perfil):
    dato = psql(pg, "SELECT jsonb_build_object('rol',r.documento,'rol_huella',r.huella_sha256,"
                "'asignacion',a.documento,'asignacion_huella',a.huella_sha256,"
                "'control',c.documento,'acto_asignacion',p.acto_ref,"
                "'huellas_validas',r.huella_sha256=encode(sha256(convert_to("
                "vec_autorizacion.canon_version_rol_admin_v1(r.documento),'UTF8')),'hex') "
                "AND a.huella_sha256=encode(sha256(convert_to("
                "vec_autorizacion.canon_asignacion_perfil_admin_v1(a.documento),'UTF8')),'hex') "
                "AND c.huella_sha256=encode(sha256(convert_to("
                "vec_autorizacion.canon_control_rol_admin_v1(c.documento),'UTF8')),'hex'))::text "
                "FROM vec_autorizacion.asignacion_perfil_actual_externa p "
                "JOIN vec_autorizacion.asignacion_perfil_externa a USING(asignacion_ref) "
                "JOIN vec_autorizacion.version_rol r USING(version_rol_ref) "
                "JOIN vec_autorizacion.control_vigencia_version_rol_actual ca "
                "ON ca.version_rol_ref=r.version_rol_ref "
                "JOIN vec_autorizacion.control_vigencia_version_rol c "
                "ON c.version_rol_ref=ca.version_rol_ref AND c.revision=ca.revision "
                "WHERE p.perfil_activo_ref=" + lit(perfil) + " AND r.rol_id=" + lit(ROL))
    exigir(bool(dato), "asignacion_candidato_ausente")
    actual = json.loads(dato)
    exigir(actual["huellas_validas"] is True and actual["control"]["estado"] == "habilitada" and
           actual["control"]["version_rol_ref"] == actual["asignacion"]["version_rol_ref"] and
           actual["asignacion"]["estado"] == "activa" and
           actual["asignacion"]["perfil_activo_ref"] == perfil,
           "instantanea_candidato_no_consumible")
    ahora = datetime.now(timezone.utc)
    inicio = datetime.fromisoformat(actual["asignacion"]["vigente_desde"].replace("Z", "+00:00"))
    fin = datetime.fromisoformat(actual["asignacion"]["vigente_hasta"].replace("Z", "+00:00"))
    exigir(inicio <= ahora < fin, "ventana_candidato_no_vigente")
    return actual


def fuente(repo, nombre, acciones):
    documento = json.loads((repo / "deploy/principal/fuentes" / nombre).read_text())
    entradas = documento["entradas"]
    por_accion = {e["concesion"]["accion"]: e for e in entradas}
    exigir(len(por_accion) == len(entradas) and all(a in por_accion for a in acciones) and
           all(por_accion[a]["fuente_huella_sha256"] == documento["huella_sha256"] and
               por_accion[a]["clase_control"] == "ordinario" for a in acciones),
           "fuente_alegaciones_divergente")
    return [por_accion[a]["concesion"] for a in acciones]


def planificar(base, preferencias, alegaciones, marcas=None):
    rol, asignacion = base["rol"], base["asignacion"]
    exigir(rol["version"] == 2 and asignacion["version"] == 3,
           "preimagen_candidato_divergente")
    if marcas is None:
        instante = datetime.now(timezone.utc).replace(microsecond=0) - timedelta(seconds=2)
        marcas = [instante.strftime("%Y-%m-%dT%H:%M:%SZ"),
                  (instante + timedelta(seconds=1)).strftime("%Y-%m-%dT%H:%M:%SZ")]
    else:
        exigir(isinstance(marcas, list) and len(marcas) == 2 and
               all(isinstance(m, str) and re.fullmatch(r"\d{4}-\d\d-\d\dT\d\d:\d\d:\d\dZ", m)
                   for m in marcas), "marcas_plan_candidato_invalidas")
        inicio = datetime.fromisoformat(marcas[0].replace("Z", "+00:00"))
        fin = datetime.fromisoformat(marcas[1].replace("Z", "+00:00"))
        exigir(fin == inicio + timedelta(seconds=1) and fin < datetime.now(timezone.utc),
               "marcas_plan_candidato_divergentes")
    documentos = {"roles": [], "controles": [], "asignaciones": []}
    previo = rol
    asignacion_previa = asignacion
    for version, nuevas, marca in ((3, preferencias, marcas[0]), (4, alegaciones, marcas[1])):
        rol_nuevo = dict(previo, version=version,
                         concesiones=previo["concesiones"] + nuevas,
                         publicada_por=OPERADOR, publicada_en=marca, estado="publicada")
        ref = "rol:" + ROL + ":v" + str(version)
        control = {"version_rol_ref": ref, "revision": 1, "estado": "habilitada",
                   "actualizado_por": OPERADOR, "actualizado_en": marca}
        asignacion_nueva = dict(asignacion_previa, version=version + 1,
                                version_rol_ref=ref, emitida_por=OPERADOR,
                                emitida_en=marca, vigente_desde=marca, estado="activa")
        documentos["roles"].append(rol_nuevo)
        documentos["controles"].append(control)
        documentos["asignaciones"].append(asignacion_nueva)
        previo, asignacion_previa = rol_nuevo, asignacion_nueva
    return documentos


def insertar_rol(doc, ctrl):
    ref = "rol:" + ROL + ":v" + str(doc["version"])
    d, c = jlit(doc), jlit(ctrl)
    return (
        "INSERT INTO vec_autorizacion.version_rol(version_rol_ref,rol_id,version,huella_sha256,publicada_en,documento) "
        f"SELECT {lit(ref)},{lit(ROL)},{doc['version']},"
        f"encode(sha256(convert_to(vec_autorizacion.canon_version_rol_admin_v1({d}),'UTF8')),'hex'),"
        f"({d}->>'publicada_en')::timestamptz,{d};"
        "INSERT INTO vec_autorizacion.control_vigencia_version_rol(version_rol_ref,revision,estado,huella_sha256,actualizado_en,documento) "
        f"SELECT {lit(ref)},1,'habilitada',"
        f"encode(sha256(convert_to(vec_autorizacion.canon_control_rol_admin_v1({c}),'UTF8')),'hex'),"
        f"({c}->>'actualizado_en')::timestamptz,{c};"
        "INSERT INTO vec_autorizacion.control_vigencia_version_rol_actual(version_rol_ref,revision,actualizada_en,actualizada_por,acto_ref) "
        f"VALUES({lit(ref)},1,({c}->>'actualizado_en')::timestamptz,{lit(OPERADOR)},"
        f"{lit('acto:ensayo:alegaciones:candidato:control:v' + str(doc['version']))});")


def insertar_asignacion(doc):
    ref = "asignacion:" + doc["asignacion_id"] + ":v" + str(doc["version"])
    d = jlit(doc)
    return (
        "INSERT INTO vec_autorizacion.asignacion_perfil_externa(asignacion_ref,asignacion_id,version,"
        "perfil_activo_ref,principal_id,version_rol_ref,huella_sha256,emitida_en,documento) "
        f"SELECT {lit(ref)},({d}->>'asignacion_id'),{doc['version']},({d}->>'perfil_activo_ref'),"
        f"({d}->>'principal_id'),({d}->>'version_rol_ref'),"
        f"encode(sha256(convert_to(vec_autorizacion.canon_asignacion_perfil_admin_v1({d}),'UTF8')),'hex'),"
        f"({d}->>'emitida_en')::timestamptz,{d};")


def publicar(pg, plan, base):
    perfil = base["asignacion"]["perfil_activo_ref"]
    previa = "asignacion:" + base["asignacion"]["asignacion_id"] + ":v3"
    nueva = "asignacion:" + base["asignacion"]["asignacion_id"] + ":v5"
    sql = ["BEGIN ISOLATION LEVEL SERIALIZABLE;", "SET LOCAL search_path=pg_catalog;",
           "SET LOCAL timezone='UTC';", "SET LOCAL lock_timeout='3s';",
           "SELECT pg_advisory_xact_lock(hashtextextended('vec:ensayo:alegaciones:candidato',0));",
           "DO $pre$ BEGIN IF EXISTS(SELECT 1 FROM vec_autorizacion.version_rol WHERE version_rol_ref IN "
           "('rol:candidato_bolsa_portal_historial_propio_desarrollo:v3',"
           "'rol:candidato_bolsa_portal_historial_propio_desarrollo:v4')) OR "
           "NOT EXISTS(SELECT 1 FROM vec_autorizacion.asignacion_perfil_actual_externa "
           f"WHERE perfil_activo_ref={lit(perfil)} AND asignacion_ref={lit(previa)}) "
           "THEN RAISE EXCEPTION 'preimagen_candidato_divergente' USING ERRCODE='55000'; END IF; END $pre$;"]
    for doc, ctrl in zip(plan["roles"], plan["controles"]):
        sql.append(insertar_rol(doc, ctrl))
    for doc in plan["asignaciones"]:
        sql.append(insertar_asignacion(doc))
    sql.append("UPDATE vec_autorizacion.asignacion_perfil_actual_externa SET "
               f"asignacion_ref={lit(nueva)},actualizada_en=clock_timestamp(),"
               f"actualizada_por={lit(OPERADOR)},acto_ref='acto:ensayo:alegaciones:candidato:v5' "
               f"WHERE perfil_activo_ref={lit(perfil)} AND asignacion_ref={lit(previa)};")
    sql.append("DO $post$ BEGIN IF (SELECT count(*) FROM vec_autorizacion.asignacion_perfil_actual_externa "
               f"WHERE perfil_activo_ref={lit(perfil)} AND asignacion_ref={lit(nueva)})<>1 "
               "THEN RAISE EXCEPTION 'postimagen_candidato_divergente' USING ERRCODE='55000'; END IF; END $post$;")
    sql.append("COMMIT;")
    psql(pg, "\n".join(sql))


def main():
    p = argparse.ArgumentParser(description=__doc__)
    p.add_argument("--ensayo", type=Path, required=True)
    p.add_argument("--pg", required=True)
    p.add_argument("--aplicar", action="store_true")
    a = p.parse_args()
    repo = Path(__file__).resolve().parents[4]
    exigir(a.pg.startswith("vec-codexo-aspirante-pg-") and re.fullmatch(r"[a-z0-9-]+", a.pg) and
           a.ensayo.is_absolute() and repo not in a.ensayo.parents and
           not a.ensayo.stat().st_mode & 0o077, "solo_clon_privado")
    candidato = json.loads((a.ensayo / "externo/identidad/bolsa-candidato.json").read_text())
    perfil = candidato["perfil_ref"]
    exigir(re.fullmatch(r"prf_[A-Za-z0-9_-]{22,128}", perfil), "perfil_candidato_invalido")
    preferencias = fuente(repo, "bolsa_preferencias_bolsas_propias_v1.json", PREFERENCIAS)
    alegaciones = fuente(repo, "bolsa_alegaciones_propias_v1.json", ALEGACIONES)
    vigente = actual(a.pg, perfil)
    if vigente["rol"]["version"] == 4:
        exigir(vigente["rol"]["concesiones"][-4:] == preferencias + alegaciones and
               vigente["asignacion"]["version"] == 5 and
               vigente["asignacion"]["version_rol_ref"] == "rol:" + ROL + ":v4" and
               vigente["acto_asignacion"] == "acto:ensayo:alegaciones:candidato:v5",
               "permisos_candidato_divergentes")
        print("permisos_candidato_alegaciones=preparados")
        return
    exigir(vigente["rol"]["version"] == 2, "preimagen_rol_candidato_divergente")
    archivo = a.ensayo / "incorporacion/actos/permisos-candidato-alegaciones.plan.json"
    if archivo.exists():
        exigir(not archivo.is_symlink() and not archivo.stat().st_mode & 0o077,
               "plan_candidato_no_privado")
        paquete = json.loads(archivo.read_text())
        exigir(paquete["preimagen"] == {"rol_huella": vigente["rol_huella"],
                                         "asignacion_huella": vigente["asignacion_huella"]},
               "plan_candidato_preimagen_divergente")
        plan = paquete["documentos"]
        exigir(isinstance(plan, dict) and isinstance(plan.get("roles"), list) and
               len(plan["roles"]) == 2, "plan_candidato_invalido")
        marcas = [plan["roles"][0]["publicada_en"], plan["roles"][1]["publicada_en"]]
        exigir(plan == planificar(vigente, preferencias, alegaciones, marcas),
               "plan_candidato_fuentes_divergentes")
    else:
        exigir(a.aplicar, "plan_candidato_pendiente")
        plan = planificar(vigente, preferencias, alegaciones)
        paquete = {"version": 1, "preimagen": {"rol_huella": vigente["rol_huella"],
                                                 "asignacion_huella": vigente["asignacion_huella"]},
                   "documentos": plan}
        with archivo.open("x") as salida:
            json.dump(paquete, salida, ensure_ascii=False, separators=(",", ":"))
            salida.write("\n")
        archivo.chmod(0o600)
    exigir(a.aplicar, "permisos_candidato_pendientes")
    publicar(a.pg, plan, vigente)
    nuevo = actual(a.pg, perfil)
    exigir(nuevo["rol"] == plan["roles"][1] and
           nuevo["asignacion"] == plan["asignaciones"][1],
           "postimagen_candidato_divergente")
    print("permisos_candidato_alegaciones=preparados")


if __name__ == "__main__":
    try:
        main()
    except (OSError, ValueError, KeyError, TypeError, json.JSONDecodeError,
            subprocess.TimeoutExpired) as error:
        raise SystemExit("permisos_candidato_alegaciones=error clave=" +
                         str(error).split(":", 1)[0]) from None
