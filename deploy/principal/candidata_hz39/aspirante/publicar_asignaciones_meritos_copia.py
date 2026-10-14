#!/usr/bin/env python3
"""Asigna Usuarios y Méritos por AUT81 con cuenta y ámbito propios."""

import argparse
from collections import OrderedDict
from datetime import datetime, timezone
import hashlib
import json
import os
from pathlib import Path
import re
import subprocess


LOGIN = "vec_prov_externo_usuarios_autorizacion"
GRUPO = "vec_autorizacion_publicador_usuarios_externo"
OPERADOR = "operacion:ensayo:aspirante"
ROL = {"usuarios": "rol:candidato_usuarios_propios_desarrollo:v1",
       "meritos": "rol:persona_meritos_cartera:v1"}


def exigir(condicion, clave):
    if not condicion:
        raise ValueError(clave)


def literal(valor):
    exigir(isinstance(valor, str) and "\x00" not in valor, "texto_invalido")
    return "'" + valor.replace("'", "''") + "'"


def admin(pg, sql):
    r = subprocess.run(["docker", "exec", "-i", pg, "psql", "-U", "postgres", "-d", "postgres",
                        "-X", "-w", "-qAt", "-v", "ON_ERROR_STOP=1", "-v", "VERBOSITY=sqlstate"],
                       input=sql, text=True, capture_output=True, timeout=20, check=False)
    if r.returncode:
        codigo = re.search(r"\b[0-9]{2}[A-Z0-9]{3}\b", r.stderr)
        raise ValueError("consulta_aut81_rechazada_" + (codigo.group(0) if codigo else "sin_sqlstate"))
    return r.stdout.strip()


def nominal(pg, ca, sql):
    r = subprocess.run(["docker", "run", "--rm", "-i", "--network", "container:" + pg,
                        "-v", str(ca) + ":/vec-pg-ca.crt:ro",
                        "-e", "PGHOST=127.0.0.1", "-e", "PGPORT=5432",
                        "-e", "PGDATABASE=postgres", "-e", "PGUSER=" + LOGIN,
                        "-e", "PGSSLMODE=verify-full", "-e", "PGSSLROOTCERT=/vec-pg-ca.crt",
                        "postgres:18.4", "psql", "-X", "-w", "-qAt", "-v", "ON_ERROR_STOP=1",
                        "-v", "VERBOSITY=sqlstate"], input=sql, text=True,
                       capture_output=True, timeout=30, check=False)
    if r.returncode:
        codigo = re.search(r"\b[0-9]{2}[A-Z0-9]{3}\b", r.stderr)
        raise ValueError("publicacion_aut81_rechazada_" + (codigo.group(0) if codigo else "sin_sqlstate"))
    return r.stdout.strip()


def preparar_login(pg, aplicar):
    q = ("SELECT count(*),coalesce(bool_and(l.rolcanlogin AND l.rolinherit AND "
         "NOT(l.rolsuper OR l.rolcreatedb OR l.rolcreaterole OR l.rolreplication OR l.rolbypassrls) "
         "AND l.rolconfig IS NULL AND m.inherit_option AND NOT m.set_option AND NOT m.admin_option "
         "AND g.rolname='" + GRUPO + "'),false) FROM pg_roles l "
         "LEFT JOIN pg_auth_members m ON m.member=l.oid "
         "LEFT JOIN pg_roles g ON g.oid=m.roleid WHERE l.rolname='" + LOGIN + "';")
    actual = admin(pg, q)
    if actual != "1|t":
        exigir(aplicar and actual == "0|f", "login_gobierno_no_nominal")
        sql = ("BEGIN; SET LOCAL search_path=pg_catalog; "
               "CREATE ROLE " + LOGIN + " LOGIN INHERIT NOSUPERUSER NOCREATEDB NOCREATEROLE "
               "NOREPLICATION NOBYPASSRLS; GRANT " + GRUPO + " TO " + LOGIN +
               " WITH INHERIT TRUE, SET FALSE, ADMIN FALSE; COMMIT;")
        admin(pg, sql)
    exigir(admin(pg, q) == "1|t", "login_gobierno_invalido")


def canon_asignacion(doc):
    orden = ("asignacion_id", "version", "perfil_activo_ref", "principal_id",
             "version_rol_ref", "estado", "ambitos", "vigente_desde", "vigente_hasta",
             "emitida_por", "emitida_en", "revocada_en")
    salida = OrderedDict((k, doc[k]) for k in orden)
    salida["ambitos"] = [OrderedDict((k, x[k]) for k in ("clave", "valores"))
                         for x in doc["ambitos"]]
    for clave in ("vigente_desde", "vigente_hasta", "emitida_en"):
        salida[clave] = salida[clave].replace(".000000Z", "Z")
    return json.dumps(salida, ensure_ascii=False, separators=(",", ":")).encode()


def actual(pg, perfil):
    q = ("SELECT a.documento::text,a.huella_sha256 FROM vec_autorizacion.asignacion_perfil_actual_externa p "
         "JOIN vec_autorizacion.asignacion_perfil_externa a USING(asignacion_ref) "
         "WHERE p.perfil_activo_ref=" + literal(perfil) + ";")
    resultado = admin(pg, q)
    if not resultado:
        return None
    d, h = resultado.split("|", 1)
    return json.loads(d), h


def asignacion_vigente(doc):
    try:
        desde = datetime.fromisoformat(doc["vigente_desde"].replace("Z", "+00:00"))
        hasta = datetime.fromisoformat(doc["vigente_hasta"].replace("Z", "+00:00"))
    except (KeyError, ValueError, TypeError):
        return False
    ahora = datetime.now(timezone.utc)
    return doc.get("estado") == "activa" and desde <= ahora < hasta


def documento(snapshot, etiqueta):
    perfil = snapshot["perfil"]["referencia"]
    persona = snapshot["persona"]["referencia"]
    ahora = datetime.now(timezone.utc).replace(microsecond=0).strftime("%Y-%m-%dT%H:%M:%SZ")
    return {"asignacion_id": etiqueta + "_externo_" + hashlib.sha256(perfil.encode()).hexdigest()[:32],
            "version": 1, "perfil_activo_ref": perfil, "principal_id": persona,
            "version_rol_ref": ROL[etiqueta], "estado": "activa",
            "ambitos": [{"clave": "persona_ref", "valores": [persona]}],
            "vigente_desde": ahora,
            "vigente_hasta": snapshot["perfil"]["vigente_hasta"].replace(".000000Z", "Z"),
            "emitida_por": OPERADOR, "emitida_en": ahora,
            "revocada_en": "0001-01-01T00:00:00Z"}


def main():
    p = argparse.ArgumentParser(description=__doc__)
    p.add_argument("--ensayo", type=Path, required=True)
    p.add_argument("--pg", required=True)
    p.add_argument("--ca", type=Path, required=True)
    p.add_argument("--aplicar", action="store_true")
    a = p.parse_args()
    exigir(a.pg.startswith("vec-codexo-aspirante-pg-") and re.fullmatch(r"[a-z0-9-]+", a.pg) and
           a.ensayo.is_absolute() and Path(__file__).resolve().parents[4] not in a.ensayo.parents and
           not a.ensayo.stat().st_mode & 0o077 and a.ca.is_file(), "solo_clon_privado")
    preparar_login(a.pg, a.aplicar)
    actos = a.ensayo / "incorporacion/actos"
    for etiqueta in ("usuarios", "meritos"):
        snapshot = json.loads((actos / ("snapshot-" + etiqueta + "-externo.json")).read_text())
        cuenta, perfil, persona = (snapshot[k]["referencia"] for k in ("cuenta", "perfil", "persona"))
        exigir(all(re.fullmatch(r"[a-z]+_[A-Za-z0-9_-]{22,128}", x)
                   for x in (cuenta, perfil, persona)), "referencias_no_validas")
        previo = actual(a.pg, perfil)
        if previo is not None and hashlib.sha256(canon_asignacion(previo[0])).hexdigest() == previo[1]:
            exigir(previo[0]["principal_id"] == persona and previo[0]["version_rol_ref"] == ROL[etiqueta] and
                   previo[0]["ambitos"] == [{"clave": "persona_ref", "valores": [persona]}] and
                   asignacion_vigente(previo[0]),
                   "asignacion_actual_divergente")
            continue
        if previo is not None:
            exigir(previo[0]["version"] == 1 and previo[0]["principal_id"] == persona and
                   previo[0]["perfil_activo_ref"] == perfil and previo[0]["version_rol_ref"] == ROL[etiqueta] and
                   previo[0]["ambitos"] == [{"clave": "persona_ref", "valores": [persona]}] and
                   asignacion_vigente(previo[0]),
                   "asignacion_actual_divergente")
            doc = dict(previo[0])
            marca = datetime.now(timezone.utc).replace(microsecond=0).strftime("%Y-%m-%dT%H:%M:%SZ")
            doc.update(version=2, emitida_por=OPERADOR, emitida_en=marca, vigente_desde=marca,
                       vigente_hasta=doc["vigente_hasta"].replace(".000000Z", "Z"))
        else:
            doc = documento(snapshot, etiqueta)
        archivo = actos / ("asignacion-" + etiqueta + "-externo-v" + str(doc["version"]) + ".json")
        if archivo.exists():
            exigir(not archivo.is_symlink() and not archivo.stat().st_mode & 0o077,
                   "asignacion_privada_invalida")
            doc = json.loads(archivo.read_text())
        else:
            exigir(a.aplicar, "asignacion_pendiente")
            with archivo.open("x") as salida:
                json.dump(doc, salida, ensure_ascii=False, separators=(",", ":"))
                salida.write("\n")
            archivo.chmod(0o600)
        exigir(doc["perfil_activo_ref"] == perfil and doc["principal_id"] == persona and
               doc["version_rol_ref"] == ROL[etiqueta] and doc["version"] == (2 if previo else 1) and
               doc["ambitos"] == [{"clave": "persona_ref", "valores": [persona]}],
               "documento_asignacion_divergente")
        canon = canon_asignacion(doc)
        huella = hashlib.sha256(canon).hexdigest()
        exigir(a.aplicar, "asignacion_pendiente")
        version_previa = previo[0]["version"] if previo else 0
        huella_previa = literal(previo[1]) if previo else "NULL"
        sql = ("BEGIN ISOLATION LEVEL SERIALIZABLE; SET LOCAL search_path=pg_catalog; "
               "SET LOCAL timezone='UTC'; SELECT asignacion_ref,version,huella_sha256 FROM "
               "vec_autorizacion.publicar_asignacion_usuarios_externo_v1(" +
               "decode(" + literal(canon.hex()) + ",'hex')," + literal(huella) + "," +
               str(version_previa) + "," + huella_previa + "," +
               literal(OPERADOR) + "," + literal("acto:ensayo:" + etiqueta +
                                                       ":asignacion:v" + str(doc["version"])) +
               "," + literal(cuenta) + "); COMMIT;")
        recibo = nominal(a.pg, a.ca, sql)
        exigir(huella in recibo and actual(a.pg, perfil) == (doc, huella),
               "recibo_asignacion_no_cotejado")
    print("asignaciones_usuarios_meritos=publicadas")


if __name__ == "__main__":
    try:
        main()
    except (OSError, ValueError, KeyError, TypeError, json.JSONDecodeError,
            subprocess.TimeoutExpired) as error:
        raise SystemExit("asignaciones_usuarios_meritos=error clave=" +
                         str(error).split(":", 1)[0]) from None
