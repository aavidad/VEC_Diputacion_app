#!/usr/bin/env python3
"""Preflight reproducible de la cadena de inscripción sobre P1 sintética.

Ejemplo:
  python3 inscripcion_gobierno_cadena_hz10_pg18.py \
    --tar /var/tmp/vec-b1-rat-fresh-20261009/base_postAUT65_P1_cold_SOLO_ENSAYO.tar \
    --scratch /var/tmp/vec-inscripcion-hz10-inventario-nuevo \
    --ad153-sql /ruta/versionada/000153_consumidores_preparacion_bases_bolsa.up.sql

Este corte solo lee SQL y catálogos de un PGDATA copiado. No ejecuta UP,
sesiones ADMIN, mTLS ni contratos HTTP. La cadena causal pendiente empieza
authBC1→rolesBC→BC1…BC7→AD153→BC8; CC1 ya existe en P1. AD153 exige una
preimagen literal del núcleo V3. Una divergencia detiene la preparación antes
de cualquier migración. Los hashes finales y la ruta ADMIN B1 siguen bajo
decisión de Dirección; una presencia de tabla no acredita una fase instalada.
"""

import argparse
import hashlib
import json
import os
from pathlib import Path
import re
import shutil
import stat
import subprocess
import sys
import time
import uuid


P1_SHA256 = "323fb79f789120a4e1241f50b8365b6323500c94c0a6aeb516ebc67494b59356"
ORDER_EDGES = [
    ("auth_bc1", "roles_bc"), ("roles_bc", "bc1"),
    ("bc1", "bc2"), ("bc2", "bc3"), ("bc3", "bc4"),
    ("bc4", "bc5"), ("bc5", "bc6"), ("bc6", "bc7"),
    ("bc7", "ad153"), ("ad153", "bc8"),
    ("ad228", "ad229"), ("ad229", "aut66"),
    ("aut66", "ad231"), ("ad231", "ca39"),
    ("ca39", "aut68"), ("personal31", "aut68"),
    ("aut67", "aut68"),
]
KNOWN_STEPS = {
    "auth_bc1", "roles_bc", *(f"bc{i}" for i in range(1, 10)),
    "ad153", "ad228", "ad229", "aut66", "ad231", "ca39",
    "aut67", "aut68", "personal31", "ca38", "cc11", "b96", "b97",
}
AD153_DEF_RE = re.compile(r"esperada_def_sha256 text:=\$esperada_def_sha256\$([0-9a-f]{64})\$esperada_def_sha256\$")
AD153_SOURCE_RE = re.compile(r"esperada_fuente_sha256 text:=\$esperada_fuente_sha256\$([0-9a-f]{64})\$esperada_fuente_sha256\$")


def sha256_file(path):
    digest = hashlib.sha256()
    with path.open("rb") as stream:
        for block in iter(lambda: stream.read(1024 * 1024), b""):
            digest.update(block)
    return digest.hexdigest()


def freeze_regular(path):
    descriptor = os.open(path, os.O_RDONLY | os.O_NOFOLLOW | os.O_CLOEXEC)
    try:
        before = os.fstat(descriptor)
        if not stat.S_ISREG(before.st_mode):
            raise ValueError(f"archivo no regular: {path}")
        with os.fdopen(descriptor, "rb", closefd=False) as stream:
            data = stream.read()
        after = os.fstat(descriptor)
    finally:
        os.close(descriptor)
    identity = lambda item: (item.st_dev, item.st_ino, item.st_size, item.st_mtime_ns)
    if identity(before) != identity(after) or len(data) != before.st_size:
        raise RuntimeError(f"archivo cambió durante lectura: {path}")
    return data, hashlib.sha256(data).hexdigest()


def run(argv, *, input_data=None):
    result = subprocess.run(argv, input=input_data, capture_output=True, text=True)
    if result.returncode:
        raise RuntimeError(f"{argv[0]} terminó {result.returncode}: {result.stderr[-1200:]}")
    return result.stdout


def manifest_inventory(path):
    if path is None:
        return {"estado": "sin_lista_final", "pasos": []}
    raw, file_sha = freeze_regular(path)
    document = json.loads(raw)
    if set(document) != {"esquema", "pasos"} or document["esquema"] != "vec.inscripcion.hz10.sql.v1":
        raise ValueError("manifiesto de cadena no canónico")
    if not isinstance(document["pasos"], list):
        raise ValueError("pasos de cadena no son una lista")
    seen = set()
    frozen = []
    for step in document["pasos"]:
        if set(step) != {"id", "ruta", "sha256"} or step["id"] not in KNOWN_STEPS:
            raise ValueError("paso de cadena no permitido")
        if step["id"] == "cc1" or step["id"] in seen:
            raise ValueError("CC1 instalado o paso duplicado")
        if not re.fullmatch(r"[0-9a-f]{64}", step["sha256"]):
            raise ValueError("SHA de paso inválido")
        source = Path(step["ruta"])
        if not source.is_absolute():
            raise ValueError("ruta de SQL debe ser absoluta")
        data, observed_sha = freeze_regular(source)
        if observed_sha != step["sha256"]:
            raise ValueError(f"SHA divergente antes de SQL: {step['id']}")
        frozen.append({"id": step["id"], "ruta": str(source),
                       "sha256": observed_sha, "bytes": len(data)})
        seen.add(step["id"])
    order = {step["id"]: index for index, step in enumerate(frozen)}
    for previous, following in ORDER_EDGES:
        if previous in order and following in order and order[previous] >= order[following]:
            raise ValueError(f"orden causal invertido: {previous}→{following}")
    return {"estado": "lista_congelada_sin_ejecutar", "sha256": file_sha,
            "pasos": frozen, "faltantes": sorted(KNOWN_STEPS - seen)}


class IsolatedClone:
    def __init__(self, scratch):
        self.name = "vec-hz10-inventario-" + uuid.uuid4().hex[:12]
        self.data = scratch / "data"
        self.socket = scratch / "socket"
        self.started = False

    def start(self):
        run(["docker", "run", "-d", "--rm", "--restart=no", "--name", self.name,
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
        raise RuntimeError("PostgreSQL 18 aislado no arrancó")

    def stop(self):
        if self.started:
            subprocess.run(["docker", "stop", self.name], capture_output=True)
            self.started = False

    def psql_read_only(self, statement):
        query = "BEGIN READ ONLY; SET LOCAL search_path=pg_catalog; " + statement + " ROLLBACK;"
        result = run(["docker", "exec", "-i", self.name, "psql", "-X", "-qAt",
                      "-v", "ON_ERROR_STOP=1", "-h", "/socket", "-p", "5432",
                      "-U", "postgres", "-d", "postgres"], input_data=query)
        return next(line for line in result.splitlines() if line.startswith("{"))


INVENTORY_SQL = """
SELECT jsonb_build_object(
 'postgres_version',current_setting('server_version_num'),
 'admin_versiones',(SELECT string_agg(version::text,',' ORDER BY version)
     FROM vec_autorizacion.version_rol WHERE rol_id='administracion_perfiles'),
 'admin_v7_catalogo',(SELECT count(*) FROM vec_autorizacion.catalogo_accion_nominal_v1
     WHERE version_rol_ref='rol:administracion_perfiles:v7'),
 'admin_v7_asignaciones',(SELECT count(*) FROM vec_autorizacion.asignacion_perfil_actual q
     JOIN vec_autorizacion.asignacion_perfil a USING(perfil_activo_ref,asignacion_ref)
     WHERE a.version_rol_ref='rol:administracion_perfiles:v7' AND a.version=4),
 'bc_esquema',to_regnamespace('vec_bolsa_convocatorias') IS NOT NULL,
 'bc_roles',(SELECT count(*) FROM pg_roles WHERE rolname IN
     ('vec_bolsa_convocatorias_propietario','vec_bolsa_convocatorias_migrador',
      'vec_bolsa_convocatorias_ejecutor_consulta','vec_bolsa_convocatorias_proyector_gobierno',
      'vec_bolsa_convocatorias_registrador_atestacion','vec_bolsa_convocatorias_verificador_recibo')),
 'auth_bc1',to_regprocedure('vec_autorizacion.revalidar_decision_bolsa_convocatorias_v1(jsonb,bytea,bytea,text,text,text,jsonb,timestamp with time zone)') IS NOT NULL,
 'bc1',to_regclass('vec_bolsa_convocatorias.version_convocatoria') IS NOT NULL,
 'bc2',(SELECT count(*) FROM pg_proc p JOIN pg_namespace n ON n.oid=p.pronamespace
     WHERE n.nspname='vec_bolsa_convocatorias' AND p.proname='obtener_version_exacta_v1'),
 'bc3',to_regclass('vec_bolsa_convocatorias.material_borrador') IS NOT NULL,
 'bc4',to_regclass('vec_bolsa_convocatorias.preparacion_confirmacion_kms_borrador') IS NOT NULL,
 'bc5',(SELECT count(*) FROM pg_proc p JOIN pg_namespace n ON n.oid=p.pronamespace
     WHERE n.nspname='vec_bolsa_convocatorias' AND p.proname='texto_selector_borradores_valido_v1'),
 'bc6',(SELECT count(*) FROM pg_proc p JOIN pg_namespace n ON n.oid=p.pronamespace
     WHERE n.nspname='vec_bolsa_convocatorias' AND p.proname='preparar_confirmacion_borrador_v2'),
 'bc7',to_regclass('vec_bolsa_convocatorias.lectura_version_v3') IS NOT NULL,
 'ad153',(SELECT count(*) FROM pg_proc p JOIN pg_namespace n ON n.oid=p.pronamespace
     WHERE n.nspname='vec_autorizacion_atestada_v3' AND p.proname IN
      ('consumir_guardar_preparacion_bases_bolsa_v3_atestada',
       'consumir_consultar_preparacion_bases_bolsa_v3_atestada')),
 'bc8',to_regclass('vec_bolsa_convocatorias.preparacion_bases_version_v3') IS NOT NULL,
 'cc1_publicacion',to_regclass('vec_catalogos_configurables.publicacion') IS NOT NULL,
 'cc1_entrada',to_regclass('vec_catalogos_configurables.entrada_publicada') IS NOT NULL,
 'ca38',(SELECT count(*) FROM pg_proc p JOIN pg_namespace n ON n.oid=p.pronamespace
     WHERE n.nspname='vec_contexto_actor_v1' AND p.proname='resolver_candidato_persona_inscripcion_v1'),
 'nucleo_ad3_def_sha256',(SELECT encode(sha256(convert_to(pg_get_functiondef(p.oid),'UTF8')),'hex')
     FROM pg_proc p WHERE p.oid=to_regprocedure('vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_interna(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)')),
 'nucleo_ad3_fuente_sha256',(SELECT encode(sha256(convert_to(p.prosrc,'UTF8')),'hex')
     FROM pg_proc p WHERE p.oid=to_regprocedure('vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_interna(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'))
)::text;
"""


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--tar", type=Path, required=True)
    parser.add_argument("--scratch", type=Path, required=True)
    parser.add_argument("--ad153-sql", type=Path, required=True)
    parser.add_argument("--manifest", type=Path)
    parser.add_argument("--keep", action="store_true")
    args = parser.parse_args()
    if args.scratch.exists():
        parser.error("--scratch debe ser nueva y exclusiva")
    if sha256_file(args.tar) != P1_SHA256:
        parser.error("tar P1 no coincide con SHA sintético acreditado")
    ad153_data, ad153_sha = freeze_regular(args.ad153_sql)
    ad153_text = ad153_data.decode("utf-8")
    expected_def = AD153_DEF_RE.search(ad153_text)
    expected_source = AD153_SOURCE_RE.search(ad153_text)
    if not expected_def or not expected_source:
        parser.error("AD153 sin dos huellas literales esperadas")
    manifest = manifest_inventory(args.manifest)
    args.scratch.mkdir(mode=0o700, parents=True)
    clone = IsolatedClone(args.scratch)
    clone.data.mkdir(mode=0o700)
    clone.socket.mkdir(mode=0o700)
    try:
        run(["tar", "-xf", str(args.tar), "-C", str(clone.data)])
        clone.start()
        inventory = json.loads(clone.psql_read_only(INVENTORY_SQL))
        baseline = (inventory["postgres_version"] == "180004"
                    and inventory["admin_versiones"] == "1,2,3,4,5,6,7"
                    and inventory["admin_v7_catalogo"] == 20
                    and inventory["admin_v7_asignaciones"] == 2
                    and inventory["cc1_publicacion"] is True
                    and inventory["cc1_entrada"] is True)
        compatible = (inventory["nucleo_ad3_def_sha256"] == expected_def.group(1)
                      and inventory["nucleo_ad3_fuente_sha256"] == expected_source.group(1))
        result = {
            "estado": ("PARO_PREIMAGEN_P1" if not baseline else
                       "PARO_AD153_SHA" if not compatible else "PREPARADO_SIN_UP"),
            "tar_sha256": P1_SHA256, "ad153_sql_sha256": ad153_sha,
            "ad153_esperado": {"def": expected_def.group(1), "fuente": expected_source.group(1)},
            "inventario_read_only": inventory, "lista": manifest,
            "sql_up_ejecutado": 0, "limite": "sin_sesiones_admin_ni_http_mtls",
        }
        output = args.scratch / "inventario_hz10.json"
        output.write_text(json.dumps(result, sort_keys=True, indent=2))
        output.chmod(0o600)
        print(json.dumps(result, sort_keys=True))
        return 0 if result["estado"] == "PREPARADO_SIN_UP" else 2
    finally:
        clone.stop()
        if not args.keep:
            shutil.rmtree(args.scratch)


if __name__ == "__main__":
    sys.exit(main())
