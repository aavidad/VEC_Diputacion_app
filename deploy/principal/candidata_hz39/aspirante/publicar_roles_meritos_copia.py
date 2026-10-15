#!/usr/bin/env python3
"""Publica roles sintéticos externos de Usuarios y Méritos en el clon O."""

import argparse
from collections import OrderedDict
from datetime import datetime, timezone
import hashlib
import json
import os
from pathlib import Path
import re
import subprocess


ROLES = {
    "usuarios": "candidato_usuarios_propios_desarrollo",
    "meritos": "persona_meritos_cartera",
}
OPERADOR = "operacion:ensayo:aspirante"
MERITOS_ACCIONES = ("meritos.cartera.consultar_propia", "meritos.cartera.retirar_propio",
                    "meritos.hecho.declarar")


def exigir(condicion, clave):
    if not condicion:
        raise ValueError(clave)


def literal(valor):
    exigir(isinstance(valor, str) and "\x00" not in valor, "texto_invalido")
    return "'" + valor.replace("'", "''") + "'"


def psql(pg, sql):
    r = subprocess.run(["docker", "exec", "-i", pg, "psql", "-U", "postgres", "-d", "postgres",
                        "-X", "-w", "-qAt", "-v", "ON_ERROR_STOP=1", "-v", "VERBOSITY=sqlstate"],
                       input=sql, text=True, capture_output=True, timeout=30, check=False)
    if r.returncode:
        codigo = re.search(r"\b[0-9]{2}[A-Z0-9]{3}\b", r.stderr)
        raise ValueError("sql_rol_rechazado_" + (codigo.group(0) if codigo else "sin_sqlstate"))
    return r.stdout.strip()


def canon_rol(documento):
    orden = ("rol_id", "version", "nombre", "estado", "concesiones", "publicada_por",
             "publicada_en", "retirada_en")
    rol = OrderedDict((k, documento.get(k, "0001-01-01T00:00:00Z") if k == "retirada_en"
                       else documento[k]) for k in orden)
    grants = ("accion", "modulo_id", "tipo_recurso", "finalidades", "garantia_minima",
              "campos_permitidos", "obligaciones")
    rol["concesiones"] = [OrderedDict((k, c[k]) for k in grants if k in c)
                          for c in documento["concesiones"]]
    return json.dumps(rol, ensure_ascii=False, separators=(",", ":")).encode()


def canon_control(documento):
    orden = ("version_rol_ref", "revision", "estado", "actualizado_por", "actualizado_en")
    return json.dumps(OrderedDict((k, documento[k]) for k in orden),
                      ensure_ascii=False, separators=(",", ":")).encode()


def privado(ruta, dato, aplicar):
    if ruta.exists():
        exigir(not ruta.is_symlink() and not ruta.stat().st_mode & 0o077 and
               json.loads(ruta.read_text()) == dato, "rol_privado_divergente")
        return
    exigir(aplicar, "rol_privado_pendiente")
    with ruta.open("x") as salida:
        json.dump(dato, salida, ensure_ascii=False, separators=(",", ":"))
        salida.write("\n")
    ruta.chmod(0o600)


def fuente_usuarios(pg):
    q = "SELECT documento::text FROM vec_autorizacion.version_rol WHERE rol_id='usuarios_preferencias_externa_h3' AND version=2;"
    dato = psql(pg, q)
    exigir(bool(dato), "rol_usuarios_fuente_ausente")
    rol = json.loads(dato)
    exigir(len(rol["concesiones"]) == 10, "rol_usuarios_fuente_divergente")
    return rol


def fuente_meritos(repo):
    ruta = repo / "deploy/principal/fuentes/meritos_cartera_propia_v1.json"
    fuente = json.loads(ruta.read_text())
    concesiones = [e["concesion"] for e in fuente["entradas"]]
    exigir([c["accion"] for c in concesiones] == list(MERITOS_ACCIONES) and
           all(e["fuente_huella_sha256"] == fuente["huella_sha256"] and
               e["dimensiones_ambito"] == ["persona_ref"] for e in fuente["entradas"]),
           "fuente_meritos_divergente")
    return concesiones


def comprobar_publicados(pg, actos, grants_meritos):
    for etiqueta, rol_id in ROLES.items():
        ref = "rol:" + rol_id + ":v1"
        consulta = ("SELECT jsonb_build_object('rol',r.documento,'rol_huella',r.huella_sha256,"
                    "'control',c.documento,'control_huella',c.huella_sha256,'estado',c.estado)::text "
                    "FROM vec_autorizacion.version_rol r "
                    "JOIN vec_autorizacion.control_vigencia_version_rol_actual p USING(version_rol_ref) "
                    "JOIN vec_autorizacion.control_vigencia_version_rol c "
                    "ON c.version_rol_ref=p.version_rol_ref AND c.revision=p.revision "
                    "WHERE r.version_rol_ref=" + literal(ref) + ";")
        respuesta = psql(pg, consulta)
        exigir(bool(respuesta), "rol_o_control_actual_ausente")
        actual = json.loads(respuesta)
        rol_privado = actos / ("rol-" + etiqueta + "-externo-v1.json")
        control_privado = actos / ("control-" + etiqueta + "-externo-v1.json")
        exigir(rol_privado.is_file() and control_privado.is_file() and
               not rol_privado.is_symlink() and not control_privado.is_symlink() and
               not rol_privado.stat().st_mode & 0o077 and
               not control_privado.stat().st_mode & 0o077,
               "rol_privado_ausente")
        esperado_rol = json.loads(rol_privado.read_text())
        esperado_control = json.loads(control_privado.read_text())
        exigir(actual["rol"] == esperado_rol and actual["control"] == esperado_control and
               actual["estado"] == "habilitada" and esperado_rol["version"] == 1 and
               esperado_control["revision"] == 1 and
               actual["rol_huella"] == hashlib.sha256(canon_rol(esperado_rol)).hexdigest() and
               actual["control_huella"] == hashlib.sha256(canon_control(esperado_control)).hexdigest(),
               "rol_o_control_actual_divergente")
        if etiqueta == "usuarios":
            exigir(psql(pg, "SELECT vec_autorizacion.rol_usuarios_externo_acotado_v1(" +
                        literal(json.dumps(esperado_rol, ensure_ascii=False,
                                           separators=(",", ":"))) + "::jsonb);") == "t",
                   "rol_usuarios_fuera_de_contrato")
        else:
            exigir(esperado_rol["concesiones"] == grants_meritos,
                   "rol_meritos_fuera_de_fuente")


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
    actos = a.ensayo / "incorporacion/actos"
    usuarios_base = fuente_usuarios(a.pg)
    meritos_grants = fuente_meritos(repo)
    publicados = psql(a.pg, "SELECT rol_id FROM vec_autorizacion.version_rol WHERE rol_id IN "
                     "('candidato_usuarios_propios_desarrollo','persona_meritos_cartera') ORDER BY rol_id;").splitlines()
    if publicados:
        exigir(publicados == sorted(ROLES.values()), "roles_parciales_o_divergentes")
        comprobar_publicados(a.pg, actos, meritos_grants)
        print("roles_usuarios_meritos=publicados")
        return
    exigir(a.aplicar, "roles_pendientes")
    rutas = [actos / (tipo + "-" + etiqueta + "-externo-v1.json")
             for etiqueta in ROLES for tipo in ("rol", "control")]
    existentes = [ruta.exists() for ruta in rutas]
    exigir(not any(existentes) or all(existentes), "paquete_roles_privado_parcial")
    if all(existentes):
        exigir(all(not ruta.is_symlink() and not ruta.stat().st_mode & 0o077
                   for ruta in rutas), "paquete_roles_no_privado")
        marca = json.loads((actos / "rol-usuarios-externo-v1.json").read_text())["publicada_en"]
    else:
        marca = datetime.now(timezone.utc).replace(microsecond=0).strftime("%Y-%m-%dT%H:%M:%SZ")
    usuarios = {k: usuarios_base[k] for k in
                ("rol_id", "version", "nombre", "estado", "concesiones", "publicada_por", "publicada_en")}
    usuarios.update(rol_id=ROLES["usuarios"], version=1,
                    nombre="areaPersonal.usuarios.rolPropio", estado="publicada",
                    publicada_por="seguridad:desarrollo:no-autoritativa", publicada_en=marca)
    meritos = {"rol_id": ROLES["meritos"], "version": 1,
               "nombre": "areaPersonal.meritos.rolCartera", "estado": "publicada",
               "concesiones": meritos_grants, "publicada_por": OPERADOR,
               "publicada_en": marca, "retirada_en": "0001-01-01T00:00:00Z"}
    documentos = {"usuarios": usuarios, "meritos": meritos}
    partes = ["BEGIN ISOLATION LEVEL SERIALIZABLE;", "SET LOCAL search_path=pg_catalog;",
              "SET LOCAL timezone='UTC';", "SET LOCAL lock_timeout='3s';",
              "SELECT pg_advisory_xact_lock(hashtextextended('vec:ensayo:meritos:roles',0));"]
    for etiqueta, doc in documentos.items():
        ref = "rol:" + ROLES[etiqueta] + ":v1"
        control = {"version_rol_ref": ref, "revision": 1, "estado": "habilitada",
                   "actualizado_por": OPERADOR, "actualizado_en": marca}
        privado(actos / ("rol-" + etiqueta + "-externo-v1.json"), doc, a.aplicar)
        privado(actos / ("control-" + etiqueta + "-externo-v1.json"), control, a.aplicar)
        rol_h = hashlib.sha256(canon_rol(doc)).hexdigest()
        control_h = hashlib.sha256(canon_control(control)).hexdigest()
        dato = literal(json.dumps(doc, ensure_ascii=False, separators=(",", ":")))
        ctrl = literal(json.dumps(control, ensure_ascii=False, separators=(",", ":")))
        partes.append(f"INSERT INTO vec_autorizacion.version_rol(version_rol_ref,rol_id,version,huella_sha256,publicada_en,documento) "
                      f"VALUES({literal(ref)},{literal(ROLES[etiqueta])},1,{literal(rol_h)},{literal(marca)}::timestamptz,{dato}::jsonb);")
        partes.append(f"INSERT INTO vec_autorizacion.control_vigencia_version_rol(version_rol_ref,revision,estado,huella_sha256,actualizado_en,documento) "
                      f"VALUES({literal(ref)},1,'habilitada',{literal(control_h)},{literal(marca)}::timestamptz,{ctrl}::jsonb);")
        partes.append(f"INSERT INTO vec_autorizacion.control_vigencia_version_rol_actual(version_rol_ref,revision,actualizada_en,actualizada_por,acto_ref) "
                      f"VALUES({literal(ref)},1,{literal(marca)}::timestamptz,{literal(OPERADOR)},'acto:ensayo:meritos:rol:{etiqueta}');")
    partes.append("COMMIT;")
    psql(a.pg, "\n".join(partes))
    exigir(psql(a.pg, "SELECT count(*) FROM vec_autorizacion.version_rol WHERE rol_id IN "
                "('candidato_usuarios_propios_desarrollo','persona_meritos_cartera');") == "2",
           "roles_no_cotejados")
    comprobar_publicados(a.pg, actos, meritos_grants)
    print("roles_usuarios_meritos=publicados")


if __name__ == "__main__":
    try:
        main()
    except (OSError, ValueError, KeyError, TypeError, json.JSONDecodeError,
            subprocess.TimeoutExpired) as error:
        raise SystemExit("roles_usuarios_meritos=error clave=" +
                         str(error).split(":", 1)[0]) from None
