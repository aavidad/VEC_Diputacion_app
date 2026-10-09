#!/usr/bin/env python3
"""Ensayo sintético AUT67 sobre una restauración privada P1 de PostgreSQL 18.

Ejemplo, desde un árbol que reúna las cinco migraciones nuevas:
  python3 deploy/postgresql/autorizacion/pruebas_sql/aut67_efecto_p1_pg18.py \
    --tar /var/tmp/vec-b1-rat-fresh-20261009/base_postAUT65_P1_cold_SOLO_ENSAYO.tar \
    --scratch /var/tmp/vec-aut67-reproduccion-nueva

La fuente P1 contiene AUT59/AUT64 instaladas, ADMIN v7 y solo datos sintéticos.
El guion exige su SHA, restaura un PGDATA nuevo, ejecuta AD228→AD229→AUT66→
AD231, prepara v8/v9 con aprobaciones sintéticas, instala AUT67 y comprueba
v10, dos AsignacionID, CAS, denegación, replay y recuperación tras reinicio.
El LOGIN técnico tiene CONNECTION LIMIT 1: los intentos simultáneos de esa
misma operación quedan excluidos por la propia precondición de AUT67.
La fuente AUT58 de Bolsa con 15 entradas se aprueba/publica en otro circuito;
este ensayo y AUT67 solo conceden las dos acciones ADMIN de inscripción.
No conecta al servidor principal ni publica nada. --keep conserva el scratch.
"""

import argparse
import hashlib
import json
import os
from pathlib import Path
import re
import shutil
import subprocess
import threading
import time
import uuid


P1_SHA = "323fb79f789120a4e1241f50b8365b6323500c94c0a6aeb516ebc67494b59356"
SOURCE_ROOT = Path(__file__).resolve().parents[4]
MIGRATIONS = {
    "ad228": "deploy/postgresql/autorizacion_atestada_v3/migraciones/000228_inscripcion_v3_auditoria.up.sql",
    "ad229": "deploy/postgresql/autorizacion_atestada_v3/migraciones/000229_incorporacion_inscripcion_v3.up.sql",
    "aut66": "deploy/postgresql/autorizacion/migraciones/000066_gobierno_inscripcion_base.up.sql",
    "ad231": "deploy/postgresql/autorizacion_atestada_v3/migraciones/000231_inscripcion_gobierno_v3.up.sql",
    "aut67": "deploy/postgresql/autorizacion/migraciones/000067_mantenimiento_inscripcion_admin.up.sql",
}
PHASES = {
    4: (7, 8, "concesiones_gobierno_definiciones_admin_v1", "preimagen_mantenimiento_gobierno_definiciones_admin_v1", "config_mantenimiento_gobierno_definiciones_admin_v1", "vec_admin_mantenimiento_gobierno_definiciones_ejecutor", "mantener_version_perfil_fijo_gobierno_definiciones_admin_v1"),
    5: (8, 9, "concesiones_version_bolsa_admin_v1", "preimagen_mantenimiento_version_bolsa_admin_v1", "config_mantenimiento_version_bolsa_admin_v1", "vec_admin_mantenimiento_version_bolsa_ejecutor", "mantener_version_perfil_fijo_version_bolsa_admin_v1"),
    6: (9, 10, "concesiones_version_inscripcion_admin_v1", "preimagen_mantenimiento_version_inscripcion_admin_v1", "config_mantenimiento_version_inscripcion_admin_v1", "vec_admin_mantenimiento_version_inscripcion_ejecutor", "mantener_version_perfil_fijo_version_inscripcion_admin_v1"),
}


def sha256(data):
    return hashlib.sha256(data).hexdigest()


def run(argv, *, input_data=None):
    result = subprocess.run(argv, input=input_data, capture_output=True, text=True)
    if result.returncode:
        raise RuntimeError(f"{argv[0]} terminó {result.returncode}: {result.stderr[-4000:]}")
    return result.stdout


def sql_json_literal(value):
    # El plan procede de la base aislada y es JSON canónico. Evita interpolar
    # una secuencia que cierre el delimitador usado por psql.
    marker = "$vecplan$"
    if marker in value:
        raise ValueError("plan con delimitador reservado")
    return marker + value + marker


class Clone:
    def __init__(self, scratch):
        self.scratch = scratch
        self.name = "vec-aut67-repro-" + uuid.uuid4().hex[:12]
        self.socket = scratch / "socket"
        self.data = scratch / "data"
        self.seller_stop = threading.Event()
        self.seller = None
        self.started = False

    def start(self):
        run(["docker", "run", "-d", "--rm", "--name", self.name,
             "--network", "none", "--memory", "2g", "--user", f"{os.getuid()}:{os.getgid()}",
             "-v", f"{self.data}:/var/lib/postgresql/18/docker",
             "-v", f"{self.socket}:/socket", "postgres:18.4", "postgres",
             "-D", "/var/lib/postgresql/18/docker", "-c", "unix_socket_directories=/socket",
             "-c", "listen_addresses=", "-c", "port=5432"])
        self.started = True
        for _ in range(100):
            ready = subprocess.run(["docker", "exec", self.name, "psql", "-X",
                                    "-h", "/socket", "-p", "5432", "-U", "postgres",
                                    "-d", "postgres", "-Atqc", "SELECT 1"], capture_output=True)
            if ready.returncode == 0:
                return
            time.sleep(0.1)
        raise RuntimeError("PostgreSQL aislado no arrancó")

    def stop(self):
        self.stop_seller()
        if self.started:
            subprocess.run(["docker", "stop", self.name], capture_output=True)
            self.started = False

    def psql(self, sql, user="postgres"):
        return run(["docker", "exec", "-i", self.name, "psql", "-X", "-qAt",
                    "-v", "ON_ERROR_STOP=1", "-h", "/socket", "-p", "5432",
                    "-U", user, "-d", "postgres"], input_data=sql).strip()

    def apply(self, path):
        self.psql(path.read_text())

    def start_seller(self):
        self.seller_stop.clear()
        # AD207 exige latido actual incluso para el primer intento de la CLI.
        self.psql("SELECT vec_autorizacion_atestada_v3.sellar_cadena_auditoria_v5(500)")

        def loop():
            while not self.seller_stop.wait(1):
                try:
                    self.psql("SELECT vec_autorizacion_atestada_v3.sellar_cadena_auditoria_v5(500)")
                except RuntimeError:
                    pass

        self.seller = threading.Thread(target=loop, daemon=True)
        self.seller.start()

    def stop_seller(self):
        self.seller_stop.set()
        if self.seller:
            self.seller.join(timeout=3)
            self.seller = None


def build_plan(clone, phase):
    src, dst, additions, prefn, config, group, facade = PHASES[phase]
    login = f"vec_aut67_ensayo_v{dst}"
    query = r'''
WITH tm AS (
 SELECT to_char((clock_timestamp()-interval '5 seconds') AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS"Z"') prep,
        to_char((clock_timestamp()+interval '20 minutes') AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS"Z"') cad
), r AS (
 SELECT * FROM vec_autorizacion.version_rol WHERE version_rol_ref='rol:administracion_perfiles:v@SRC@'
), ctl AS (
 SELECT c.* FROM vec_autorizacion.control_vigencia_version_rol_actual p
 JOIN vec_autorizacion.control_vigencia_version_rol c USING(version_rol_ref,revision)
 WHERE p.version_rol_ref='rol:administracion_perfiles:v@SRC@'
), cat AS (
 SELECT encode(sha256(convert_to(COALESCE(jsonb_agg(to_jsonb(c) ORDER BY c.accion_ref,c.version),'[]'::jsonb)::text,'UTF8')),'hex') sha
 FROM vec_autorizacion.catalogo_accion_nominal_v1 c WHERE c.version_rol_ref='rol:administracion_perfiles:v@SRC@'
), targets AS (
 SELECT jsonb_agg(jsonb_build_object(
  'perfil_ref',a.perfil_activo_ref,'asignacion_origen_ref',a.asignacion_ref,
  'asignacion_origen_sha256',a.huella_sha256,'persona_ref',a.principal_id,
  'cuenta_ref',v.cuenta_ref,'vinculo_ref',v.vinculo_ref,
  'cuenta_version',ca.version,'persona_version',pa.version,
  'perfil_version',pf.version,'vinculo_version',va.version,
  'ambitos_fuente',jsonb_build_array(
   jsonb_build_object('dimension','organizacion_ref','valores',jsonb_build_array(o.organizacion_ref),
    'fuente',jsonb_build_object('referencia',ov.procedencia_ref,'version',ov.procedencia_version,'huella_sha256',ov.procedencia_huella_sha256)),
   jsonb_build_object('dimension','unidad_ref','valores',jsonb_build_array(unit.unidad_ref),
    'fuente',jsonb_build_object('referencia',unit.fuente_ref,'version',unit.revision,'huella_sha256',unit.huella_fuente_sha256))
  )) ORDER BY a.perfil_activo_ref) asig
 FROM vec_autorizacion.asignacion_perfil_actual ap
 JOIN vec_autorizacion.asignacion_perfil a USING(perfil_activo_ref,asignacion_ref)
 JOIN vec_contexto_actor_v1.vinculo_contexto_versiones v ON v.perfil_ref=a.perfil_activo_ref AND v.persona_ref=a.principal_id
 JOIN vec_contexto_actor_v1.vinculo_contexto_actual va ON va.vinculo_ref=v.vinculo_ref AND va.version=v.version
 JOIN vec_contexto_actor_v1.proyeccion_cuenta_actual ca ON ca.cuenta_ref=v.cuenta_ref
 JOIN vec_contexto_actor_v1.persona_actual pa ON pa.persona_ref=v.persona_ref
 JOIN vec_contexto_actor_v1.perfil_actual pf ON pf.perfil_ref=v.perfil_ref
 JOIN vec_contexto_actor_v1.organizacion_actual o ON o.organizacion_ref=a.documento#>>'{ambitos,0,valores,0}'
 JOIN vec_contexto_actor_v1.organizacion_versiones ov ON ov.organizacion_ref=o.organizacion_ref AND ov.version=o.version
 JOIN LATERAL (
  SELECT h.* FROM vec_personal.org_nodo_historia h
  WHERE h.organismo_ref=o.organizacion_ref AND h.unidad_ref=a.documento#>>'{ambitos,1,valores,0}'
  ORDER BY h.conocido_desde DESC,h.revision DESC LIMIT 1
 ) unit ON true
 WHERE a.version_rol_ref='rol:administracion_perfiles:v@SRC@'
), doc AS (
 SELECT jsonb_set(jsonb_set(jsonb_set(jsonb_set(r.documento,'{version}',to_jsonb(@DST@)),
   '{concesiones}',r.documento->'concesiones'||vec_autorizacion.@ADDITIONS@()),
   '{publicada_por}',to_jsonb('mantenimiento_operador:@LOGIN@'::text)),
   '{publicada_en}',to_jsonb(tm.prep)) destination
 FROM r,tm
)
SELECT jsonb_build_object('version',@PHASE@,'operacion_ref','pmf_'||replace(gen_random_uuid()::text,'-',''),
 'preparado_en',tm.prep,'caduca_en',tm.cad,'rol_origen_sha256',r.huella_sha256,
 'control_revision_esperada',ctl.revision,'control_huella_sha256',ctl.huella_sha256,
 'catalogo_sha256',cat.sha,'rol_destino_doc',doc.destination,
 'asignaciones',targets.asig)::text
FROM tm,r,ctl,cat,targets,doc;
'''
    for token, value in {"@SRC@": src, "@DST@": dst, "@PHASE@": phase,
                         "@ADDITIONS@": additions, "@LOGIN@": login}.items():
        query = query.replace(token, str(value))
    plan = clone.psql(query)
    doc = json.loads(plan)
    if doc["version"] != phase or len(doc["asignaciones"]) != 2:
        raise AssertionError("plan sintético incompleto")
    plan_sha = sha256(plan.encode())
    pre_sha = clone.psql("BEGIN ISOLATION LEVEL SERIALIZABLE READ WRITE;"
                          "SET LOCAL TIME ZONE 'UTC';"
                          f"SELECT encode(sha256(convert_to(vec_autorizacion.{prefn}({sql_json_literal(plan)}::jsonb)::text,'UTF8')),'hex');"
                          "ROLLBACK;").splitlines()[-1]
    if not re.fullmatch(r"[0-9a-f]{64}", pre_sha):
        raise AssertionError("preimagen sintética ausente")
    approval_sha = sha256(json.dumps({"huella_plan_sha256": plan_sha}, separators=(",", ":")).encode())
    clone.psql(f"""
      REVOKE TEMP ON DATABASE postgres FROM PUBLIC;
      GRANT CONNECT ON DATABASE postgres TO {group};
      CREATE ROLE {login} LOGIN INHERIT NOSUPERUSER NOCREATEROLE NOCREATEDB
       NOREPLICATION NOBYPASSRLS CONNECTION LIMIT 1;
      GRANT {group} TO {login} WITH INHERIT TRUE, SET FALSE, ADMIN FALSE;
      SET ROLE vec_autorizacion_propietario;
      INSERT INTO vec_autorizacion.{config}
       (login_nombre,plan_sha256,preimagen_sha256,catalogo_sha256,aprobacion_ref,
        aprobacion_sha256,entorno,vigente_desde,vigente_hasta)
      VALUES ('{login}','{plan_sha}','{pre_sha}','{doc['catalogo_sha256']}',
       'SOLO_ENSAYO:AUT67:v{dst}','{approval_sha}','desarrollo',
       clock_timestamp()-interval '1 minute',('{doc['caduca_en']}'::timestamptz)+interval '1 minute');
      RESET ROLE;
    """)
    return plan, plan_sha, login, facade, doc


def invoke(clone, login, facade, plan, plan_sha):
    sql = ("BEGIN ISOLATION LEVEL SERIALIZABLE READ WRITE; SET LOCAL TIME ZONE 'UTC';"
           f"SELECT vec_autorizacion.{facade}({sql_json_literal(plan)}::text,'{plan_sha}'::text); COMMIT;")
    result = clone.psql(sql, user=login)
    return json.loads(next(line for line in result.splitlines() if line.startswith("{")))


def snapshot(clone):
    return json.loads(clone.psql("""
      SELECT jsonb_build_object(
       'rol_v10',(SELECT count(*) FROM vec_autorizacion.version_rol WHERE version_rol_ref='rol:administracion_perfiles:v10'),
       'historia_rol',(SELECT count(*) FROM vec_autorizacion.version_rol WHERE rol_id='administracion_perfiles'),
       'catalogo_v9',(SELECT count(*) FROM vec_autorizacion.catalogo_accion_nominal_v1 WHERE version_rol_ref='rol:administracion_perfiles:v9'),
       'catalogo_v10',(SELECT count(*) FROM vec_autorizacion.catalogo_accion_nominal_v1 WHERE version_rol_ref='rol:administracion_perfiles:v10'),
       'acciones_inscripcion',(SELECT count(*) FROM vec_autorizacion.catalogo_accion_nominal_v1 WHERE version_rol_ref='rol:administracion_perfiles:v10' AND accion_ref LIKE 'accion:administracion.perfiles.version_inscripcion.%'),
       'asignaciones',(SELECT jsonb_agg(jsonb_build_object('id',a.asignacion_id,'version',a.version) ORDER BY a.asignacion_id) FROM vec_autorizacion.asignacion_perfil_actual q JOIN vec_autorizacion.asignacion_perfil a USING(perfil_activo_ref,asignacion_ref) WHERE a.version_rol_ref='rol:administracion_perfiles:v10'),
       'historia_asignaciones',(SELECT count(*) FROM vec_autorizacion.asignacion_perfil a
         WHERE a.asignacion_id IN ('bootstrap_05065c85da8dc4d75e5bc16016238e1a',
                                    'bootstrap_ce19aba884538a2d376e3686df75e218')),
       'registros',(SELECT count(*) FROM vec_autorizacion.registro_mantenimiento_version_inscripcion_admin_v1),
       'efectos',(SELECT count(*) FROM vec_autorizacion_atestada_v3.auditoria_consumo_v3 a
         JOIN vec_autorizacion.registro_mantenimiento_version_inscripcion_admin_v1 m
           ON m.auditoria_ref=a.auditoria_ref
         WHERE a.tipo_registro='mantenimiento_perfil_fijo_admin'),
       'sellos_asignacion',(SELECT count(*) FROM vec_autorizacion.sello_efecto_admin_tx_v1 s
         JOIN vec_autorizacion.registro_mantenimiento_version_inscripcion_admin_v1 m
           ON m.operacion_ref=s.operacion_ref)
      )::text;
    """))


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--tar", type=Path, required=True)
    parser.add_argument("--scratch", type=Path, required=True)
    parser.add_argument("--source-root", type=Path, default=SOURCE_ROOT)
    for name in MIGRATIONS:
        parser.add_argument(f"--{name}", type=Path)
    parser.add_argument("--keep", action="store_true")
    args = parser.parse_args()
    if args.scratch.exists():
        parser.error("--scratch debe ser una ruta nueva y exclusiva")
    if sha256(args.tar.read_bytes()) != P1_SHA:
        parser.error("la copia P1 no coincide con su SHA256 sintético")
    paths = {name: (getattr(args, name) or args.source_root / relative)
             for name, relative in MIGRATIONS.items()}
    for name, path in paths.items():
        if not path.is_file():
            parser.error(f"falta {name}: {path}")
    args.scratch.mkdir(mode=0o700, parents=True)
    clone = Clone(args.scratch)
    clone.data.mkdir(mode=0o700)
    clone.socket.mkdir(mode=0o700)
    try:
        run(["tar", "-xf", str(args.tar), "-C", str(clone.data)])
        clone.start()
        baseline = clone.psql("SELECT string_agg(version::text,',' ORDER BY version) FROM vec_autorizacion.version_rol WHERE rol_id='administracion_perfiles'")
        if baseline != "1,2,3,4,5,6,7":
            raise AssertionError(f"P1 no inicia en ADMIN v7: {baseline}")
        for name in ("ad228", "ad229", "aut66", "ad231"):
            clone.apply(paths[name])
        clone.start_seller()
        for phase in (4, 5):
            plan, plan_sha, login, facade, _ = build_plan(clone, phase)
            result = invoke(clone, login, facade, plan, plan_sha)
            if result.get("estado") != "permitido" or result.get("replay") is not False:
                raise AssertionError(f"fase {phase} falló: {result.get('codigo')}")
        clone.apply(paths["aut67"])
        # La instalación estructural conserva v9 y no concede CONNECT.
        structure = Path(__file__).with_name("aut67_estructura_clon.sql")
        clone.apply(structure)
        plan, plan_sha, login, facade, doc = build_plan(clone, 6)
        before_ids = sorted(x["asignacion_origen_ref"].split(":")[1] for x in doc["asignaciones"])
        first = invoke(clone, login, facade, plan, plan_sha)
        replay = invoke(clone, login, facade, plan, plan_sha)
        denied = invoke(clone, login, facade, plan, "0" * 64)
        state = snapshot(clone)
        expected_ids = [{"id": x, "version": 7} for x in before_ids]
        if (first.get("estado") != "permitido" or first.get("replay") is not False
                or replay.get("estado") != "permitido" or replay.get("replay") is not True
                or first.get("recibo") != replay.get("recibo")
                or denied.get("estado") != "denegado" or denied.get("recibo") is not None
                or state["rol_v10"] != 1 or state["historia_rol"] != 10
                or state["catalogo_v9"] != 24 or state["catalogo_v10"] != 26
                or state["acciones_inscripcion"] != 2 or state["asignaciones"] != expected_ids
                or state["historia_asignaciones"] != 14
                or state["registros"] != 1 or state["efectos"] != 1
                or state["sellos_asignacion"] != 2):
            raise AssertionError("efecto, replay, denegación o CAS divergente")
        clone.stop()
        clone.start()
        clone.start_seller()
        recovered = snapshot(clone)
        after_restart = invoke(clone, login, facade, plan, plan_sha)
        if (recovered != state or after_restart.get("estado") != "permitido"
                or after_restart.get("replay") is not True
                or after_restart.get("recibo") != first.get("recibo")):
            raise AssertionError("recuperación tras reinicio divergente")
        print(json.dumps({"resultado": "AUT67-ENSAYO-OK", "migraciones_sha256":
                          {name: sha256(path.read_bytes()) for name, path in paths.items()},
                          "rol_v10": state["rol_v10"], "catalogo_v10": state["catalogo_v10"],
                          "dos_aid_cas_v7": state["asignaciones"], "registros": state["registros"],
                          "efectos_auditados": state["efectos"], "sellos_asignacion": state["sellos_asignacion"],
                          "recibo_sha256": sha256(json.dumps(first["recibo"], sort_keys=True, separators=(",", ":")).encode()),
                          "replay": replay["replay"], "denegacion": denied["estado"],
                          "replay_tras_reinicio": after_restart["replay"]}, sort_keys=True))
    finally:
        clone.stop()
        if not args.keep:
            shutil.rmtree(args.scratch)


if __name__ == "__main__":
    main()
