#!/usr/bin/env python3
"""Publica la asignación externa sintética de la copia mediante AUT16."""

import argparse
from collections import OrderedDict
from datetime import datetime, timezone
import hashlib
import json
import os
from pathlib import Path
import re
import stat
import subprocess


ROL = "rol:candidato_bolsa_portal_historial_propio_desarrollo:v1"
OPERADOR = "operacion:ensayo:aspirante"
ACTO = "acto:ensayo:asignacion_candidato_externo"
REFERENCIA = re.compile(r"^[a-z]+_[A-Za-z0-9_-]{22,128}$")


def exigir(condicion, clave):
    if not condicion:
        raise ValueError(clave)


def privado(ruta, directorio=False):
    dato = ruta.lstat()
    exigir(dato.st_uid == os.getuid() and not ruta.is_symlink() and not dato.st_mode & 0o077,
           "ruta_no_privada")
    exigir(stat.S_ISDIR(dato.st_mode) if directorio else stat.S_ISREG(dato.st_mode),
           "tipo_de_ruta_invalido")


def leer_json(ruta):
    privado(ruta)
    return json.loads(ruta.read_text())


def consultar(pg, sql):
    resultado = subprocess.run(
        ["docker", "exec", "-i", pg, "psql", "-U", "postgres", "-d", "postgres",
         "-X", "-w", "-qAt", "-v", "ON_ERROR_STOP=1"], input=sql,
        text=True, capture_output=True, timeout=15, check=False,
    )
    exigir(resultado.returncode == 0, "lectura_pg_fallida")
    return resultado.stdout.strip()


def ejecutar_nominal(pg, ca, sql):
    resultado = subprocess.run(
        ["docker", "run", "--rm", "-i", "--network", "container:" + pg,
         "-v", str(ca) + ":/vec-pg-ca.crt:ro",
         "-e", "PGHOST=127.0.0.1", "-e", "PGPORT=5432", "-e", "PGDATABASE=postgres",
         "-e", "PGUSER=vec_prov_externo_autorizacion", "-e", "PGSSLMODE=verify-full",
         "-e", "PGSSLROOTCERT=/vec-pg-ca.crt", "postgres:18.4", "psql", "-X", "-w",
         "-qAt", "-v", "ON_ERROR_STOP=1", "-v", "VERBOSITY=sqlstate"], input=sql, text=True,
        capture_output=True, timeout=30, check=False,
    )
    if resultado.returncode != 0:
        codigo = re.search(r"\b[0-9]{2}[A-Z0-9]{3}\b", resultado.stderr)
        raise ValueError("publicacion_aut16_rechazada_" + (codigo.group(0) if codigo else "sin_sqlstate"))
    return resultado.stdout.strip()


def literal(valor):
    exigir(isinstance(valor, str) and "\x00" not in valor, "texto_sql_invalido")
    return "'" + valor.replace("'", "''") + "'"


def estado_actual(pg, perfil):
    sql = ("SELECT a.documento::text,a.huella_sha256 FROM vec_autorizacion.asignacion_perfil_actual_externa p "
           "JOIN vec_autorizacion.asignacion_perfil_externa a USING(asignacion_ref) "
           "WHERE p.perfil_activo_ref=" + literal(perfil) + ";")
    resultado = consultar(pg, sql)
    if not resultado:
        return None
    documento, huella = resultado.split("|", 1)
    return json.loads(documento), huella


def canon_dominio(documento):
    orden = ("asignacion_id", "version", "perfil_activo_ref", "principal_id",
             "version_rol_ref", "estado", "ambitos", "vigente_desde", "vigente_hasta",
             "emitida_por", "emitida_en", "revocada_en")
    salida = OrderedDict((k, documento[k]) for k in orden)
    salida["ambitos"] = [OrderedDict((k, ambito[k]) for k in ("clave", "valores"))
                         for ambito in documento["ambitos"]]
    return json.dumps(salida, ensure_ascii=False, separators=(",", ":")).encode()


def documento_valido(documento, huella):
    return (documento["vigente_desde"] >= documento["emitida_en"] and
            documento.get("revocada_en") == "0001-01-01T00:00:00Z" and
            hashlib.sha256(canon_dominio(documento)).hexdigest() == huella)


def construir_documento(pg, interno, externo):
    perfil = interno["perfil_ref"]
    sql = ("SELECT a.documento::text FROM vec_autorizacion.asignacion_perfil_actual p "
           "JOIN vec_autorizacion.asignacion_perfil a USING(asignacion_ref) "
           "WHERE p.perfil_activo_ref=" + literal(perfil) + ";")
    fuente = json.loads(consultar(pg, sql))
    exigir(fuente["principal_id"] == interno["persona_ref"] and
           fuente["ambitos"] == [{"clave": "candidato_ref", "valores": [interno["candidato_ref"]]}],
           "asignacion_interna_no_coincide")
    fuente["asignacion_id"] = ("candidato_externo_" +
                              hashlib.sha256(externo["perfil_ref"].encode()).hexdigest()[:32])
    fuente["version"] = 1
    fuente["perfil_activo_ref"] = externo["perfil_ref"]
    fuente["version_rol_ref"] = ROL
    fuente["estado"] = "activa"
    fuente["emitida_por"] = OPERADOR
    instante = datetime.now(timezone.utc).replace(microsecond=0).strftime("%Y-%m-%dT%H:%M:%SZ")
    fuente["emitida_en"] = instante
    fuente["vigente_desde"] = instante
    fuente["revocada_en"] = "0001-01-01T00:00:00Z"
    return fuente


def publicar(pg, ca, documento, version_previa, huella_previa):
    canon = canon_dominio(documento)
    huella = hashlib.sha256(canon).hexdigest()
    esperado = "NULL" if huella_previa is None else literal(huella_previa)
    sql = ("BEGIN ISOLATION LEVEL SERIALIZABLE; SELECT asignacion_ref,version,huella_sha256 "
           "FROM vec_autorizacion.publicar_asignacion_candidato_externo_v1(" +
           "decode(" + literal(canon.hex()) + ",'hex')," + literal(huella) + "," +
           str(version_previa) + "," + esperado + "," +
           literal(OPERADOR) + "," + literal(ACTO + ":v" + str(documento["version"])) + "); COMMIT;")
    recibo = ejecutar_nominal(pg, ca, sql)
    exigir(huella in recibo, "recibo_asignacion_no_cotejado")
    return huella


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--ensayo", type=Path, required=True)
    parser.add_argument("--pg", required=True)
    parser.add_argument("--ca", type=Path, required=True)
    parser.add_argument("--aplicar", action="store_true")
    args = parser.parse_args()
    exigir(args.pg.startswith("vec-codexo-aspirante-pg-") and
           re.fullmatch(r"[a-z0-9-]+", args.pg) is not None,
           "solo_clon_aspirante")
    exigir(args.ensayo.is_absolute() and args.ca.is_absolute() and
           Path(__file__).resolve().parents[4] not in args.ensayo.parents,
           "rutas_no_admitidas")
    privado(args.ensayo, True)
    exigir(args.ca.is_file() and not args.ca.is_symlink(), "ca_ausente")
    interno = leer_json(args.ensayo / "material/identidad/bolsa-candidato.json")
    externo = leer_json(args.ensayo / "externo/identidad/bolsa-candidato.json")
    exigir(all(REFERENCIA.fullmatch(interno[k]) for k in
               ("perfil_ref", "persona_ref", "candidato_ref")) and
           REFERENCIA.fullmatch(externo["perfil_ref"]), "referencia_invalida")
    exigir(interno["persona_ref"] == externo["persona_ref"] and
           interno["candidato_ref"] == externo["candidato_ref"] and
           interno["perfil_ref"] != externo["perfil_ref"], "identidad_externa_divergente")
    actual = estado_actual(args.pg, externo["perfil_ref"])
    if actual is not None and documento_valido(*actual):
        exigir(actual[0]["principal_id"] == externo["persona_ref"] and
               actual[0]["perfil_activo_ref"] == externo["perfil_ref"] and
               actual[0]["ambitos"] == [{"clave": "candidato_ref", "valores": [externo["candidato_ref"]]}],
               "asignacion_actual_divergente")
        print("asignacion_externa=confirmada")
        return
    exigir(args.aplicar, "asignacion_externa_pendiente")
    if actual is None:
        documento = construir_documento(args.pg, interno, externo)
        previa, huella_previa = 0, None
    else:
        exigir(actual[0]["version"] == 1 and actual[0]["principal_id"] == externo["persona_ref"] and
               actual[0]["perfil_activo_ref"] == externo["perfil_ref"] and
               actual[0]["version_rol_ref"] == ROL and
               actual[0]["ambitos"] == [{"clave": "candidato_ref", "valores": [externo["candidato_ref"]]}],
               "asignacion_actual_divergente")
        documento = dict(actual[0])
        instante = datetime.now(timezone.utc).replace(microsecond=0).strftime("%Y-%m-%dT%H:%M:%SZ")
        documento.update(version=2, vigente_desde=instante, emitida_en=instante,
                         emitida_por=OPERADOR, revocada_en="0001-01-01T00:00:00Z")
        previa, huella_previa = 1, actual[1]
    archivo = args.ensayo / ("incorporacion/actos/asignacion-candidato-externo-v" +
                             str(documento["version"]) + ".json")
    if archivo.exists():
        documento = leer_json(archivo)
    else:
        with archivo.open("x") as salida:
            json.dump(documento, salida, ensure_ascii=False, separators=(",", ":"))
            salida.write("\n")
        archivo.chmod(0o600)
    exigir(documento["version"] == previa+1 and
           documento["principal_id"] == externo["persona_ref"] and
           documento["perfil_activo_ref"] == externo["perfil_ref"] and
           documento["version_rol_ref"] == ROL and
           documento["ambitos"] == [{"clave": "candidato_ref", "valores": [externo["candidato_ref"]]}] and
           documento_valido(documento, hashlib.sha256(canon_dominio(documento)).hexdigest()),
           "documento_privado_divergente")
    huella = publicar(args.pg, args.ca, documento, previa, huella_previa)
    final = estado_actual(args.pg, externo["perfil_ref"])
    exigir(final == (documento, huella) and documento_valido(*final),
           "recibo_asignacion_no_cotejado")
    print("asignacion_externa=confirmada")


if __name__ == "__main__":
    try:
        main()
    except (OSError, ValueError, KeyError, TypeError, json.JSONDecodeError, subprocess.TimeoutExpired) as error:
        raise SystemExit("asignacion_externa=error clave=" + str(error).split(":", 1)[0]) from None
