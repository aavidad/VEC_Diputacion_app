#!/usr/bin/env python3
"""Prepara el peer documental local solo después de instalar sus migraciones."""

import argparse
import json
import os
from pathlib import Path
import re
import stat
import subprocess
from urllib.parse import parse_qs, urlsplit


LOGIN = {
    "datos": ("vec_externo_documentos_datos", "vec_documentos_subida_datos"),
    "frontera": ("vec_externo_documentos_frontera", "vec_documentos_frontera_externa"),
}


def exigir(condicion, clave):
    if not condicion:
        raise ValueError(clave)


def psql(pg, sql):
    r = subprocess.run(["docker", "exec", "-i", pg, "psql", "-U", "postgres", "-d", "postgres",
                        "-X", "-w", "-qAt", "-v", "ON_ERROR_STOP=1", "-v", "VERBOSITY=sqlstate"],
                       input=sql, text=True, capture_output=True, timeout=25, check=False)
    if r.returncode:
        codigo = re.search(r"\b[0-9]{2}[A-Z0-9]{3}\b", r.stderr)
        raise ValueError("sql_documentos_rechazado_" + (codigo.group(0) if codigo else "sin_sqlstate"))
    return r.stdout.strip()


def privado(ruta, directorio=False):
    info = ruta.lstat()
    exigir(info.st_uid == os.getuid() and not ruta.is_symlink() and
           (stat.S_ISDIR(info.st_mode) if directorio else stat.S_ISREG(info.st_mode)) and
           stat.S_IMODE(info.st_mode) == (0o700 if directorio else 0o600), "material_no_privado")


def crear_directorio(ruta, aplicar):
    if not ruta.exists():
        exigir(aplicar and not ruta.is_symlink(), "directorio_documental_pendiente")
        ruta.mkdir(mode=0o700)
    privado(ruta, True)


def escribir_exclusivo(ruta, contenido, aplicar):
    if ruta.exists() or ruta.is_symlink():
        privado(ruta)
        exigir(ruta.read_bytes() == contenido, "material_documental_divergente")
        return
    exigir(aplicar, "material_documental_pendiente")
    fd = os.open(ruta, os.O_WRONLY | os.O_CREAT | os.O_EXCL | os.O_NOFOLLOW, 0o600)
    with os.fdopen(fd, "wb") as salida:
        salida.write(contenido)
        salida.flush()
        os.fsync(salida.fileno())
    privado(ruta)


def preparar_login(pg, aplicar):
    grupos = psql(pg, "SELECT count(*) FROM pg_roles WHERE rolname IN "
                 "('vec_documentos_subida_datos','vec_documentos_frontera_externa')")
    esquema = psql(pg, "SELECT to_regclass('vec_documentos.frontera_subida_local') IS NOT NULL")
    exigir(grupos == "2" and esquema == "t", "migraciones_documentales_pendientes")
    if aplicar:
        sentencias = ["BEGIN;", "SET LOCAL search_path=pg_catalog;", "SET LOCAL lock_timeout='3s';",
                      "SELECT pg_advisory_xact_lock(hashtextextended('vec:ensayo:documentos:login',0));"]
        for usuario, grupo in LOGIN.values():
            sentencias.append(
                f"DO $g$ DECLARE r record; extras integer; BEGIN "
                f"IF to_regrole('{usuario}') IS NULL THEN "
                f"CREATE ROLE {usuario} LOGIN INHERIT NOSUPERUSER NOCREATEDB NOCREATEROLE "
                "NOREPLICATION NOBYPASSRLS; ELSE "
                "SELECT rolcanlogin,rolinherit,rolsuper,rolcreatedb,rolcreaterole,"
                f"rolreplication,rolbypassrls,rolconfig INTO r FROM pg_roles WHERE rolname='{usuario}'; "
                f"SELECT count(*) INTO extras FROM pg_auth_members WHERE member=to_regrole('{usuario}') "
                f"AND (roleid<>to_regrole('{grupo}') OR NOT inherit_option OR set_option OR admin_option); "
                "IF NOT r.rolcanlogin OR NOT r.rolinherit OR r.rolsuper OR r.rolcreatedb OR "
                "r.rolcreaterole OR r.rolreplication OR r.rolbypassrls OR r.rolconfig IS NOT NULL OR extras<>0 "
                "THEN RAISE EXCEPTION 'login_documental_preimagen_divergente' USING ERRCODE='55000'; "
                "END IF; END IF; "
                f"IF NOT EXISTS(SELECT 1 FROM pg_auth_members WHERE member=to_regrole('{usuario}') "
                f"AND roleid=to_regrole('{grupo}')) THEN "
                f"GRANT {grupo} TO {usuario} WITH INHERIT TRUE, SET FALSE, ADMIN FALSE; "
                "END IF; END $g$;")
            sentencias.append(
                f"DO $post$ BEGIN IF (SELECT count(*) FROM pg_auth_members WHERE member=to_regrole('{usuario}') "
                f"AND roleid=to_regrole('{grupo}') AND inherit_option AND NOT set_option AND NOT admin_option)<>1 "
                f"OR (SELECT count(*) FROM pg_auth_members WHERE member=to_regrole('{usuario}'))<>1 "
                "THEN RAISE EXCEPTION 'login_documental_postimagen_divergente' USING ERRCODE='55000'; "
                "END IF; END $post$;")
        sentencias.append("COMMIT;")
        psql(pg, "\n".join(sentencias))
    for usuario, grupo in LOGIN.values():
        q = ("SELECT l.rolcanlogin,l.rolinherit,l.rolsuper,l.rolcreatedb,l.rolcreaterole,"
             "l.rolreplication,l.rolbypassrls,l.rolconfig IS NULL,count(m.roleid),"
             "coalesce(string_agg(g.rolname||':'||m.inherit_option||':'||m.set_option||':'||m.admin_option,','),'') "
             "FROM pg_roles l LEFT JOIN pg_auth_members m ON m.member=l.oid "
             "LEFT JOIN pg_roles g ON g.oid=m.roleid WHERE l.rolname='" + usuario + "' "
             "GROUP BY l.rolcanlogin,l.rolinherit,l.rolsuper,l.rolcreatedb,"
             "l.rolcreaterole,l.rolreplication,l.rolbypassrls,l.rolconfig")
        campos = psql(pg, q).split("|")
        exigir(campos == ["t", "t", "f", "f", "f", "f", "f", "t", "1",
                           grupo + ":true:false:false"], "login_documental_no_nominal")


def preparar(ensayo, aplicar):
    base = ensayo / "externo/externo/documentos"
    crear_directorio(ensayo / "documentos", aplicar)
    crear_directorio(ensayo / "documentos/custodia", aplicar)
    crear_directorio(ensayo / "externo/externo", aplicar)
    crear_directorio(base, aplicar)
    nombres_clave = ("correlacion.bin", "seudonimo.bin", "solicitud.bin")
    claves = []
    for nombre in nombres_clave:
        clave = base / nombre
        if not clave.exists():
            exigir(aplicar, "clave_documental_pendiente")
            escribir_exclusivo(clave, os.urandom(32), True)
        privado(clave)
        contenido = clave.read_bytes()
        exigir(len(contenido) == 32 and contenido != bytes(32), "clave_documental_invalida")
        claves.append(contenido)
    exigir(len(set(claves)) == 3, "claves_documentales_repetidas")
    dsn_base = (ensayo / "incorporacion/externo-dsn/preflight.dsn").read_text().strip()
    partes = urlsplit(dsn_base)
    query = parse_qs(partes.query)
    exigir(partes.scheme == "postgresql" and partes.hostname == "localhost" and
           partes.port == 5432 and partes.path == "/postgres" and
           query.get("sslmode") == ["verify-full"] and
           query.get("sslrootcert") == ["/vec-pg-ca.crt"], "dsn_base_invalida")
    dsns = {clave_dsn: "postgresql://" + usuario + "@127.0.0.1:5432/postgres"
            "?sslmode=verify-full&sslrootcert=/vec-pg-ca.crt"
            for clave_dsn, (usuario, _) in LOGIN.items()}
    for ruta in ("ca.crt", "cliente.crt", "cliente.key"):
        certificado = ensayo / "servicios/analizador/tls" / ruta
        exigir(certificado.is_file() and not certificado.is_symlink(), "cliente_analizador_pendiente")
    politica = json.loads((ensayo / "material/identidad/documentos.json").read_text())
    retencion = politica["almacen"]["retencion_minima_dias"]
    exigir(type(retencion) is int and retencion == 0, "retencion_existente_divergente")
    conf = {"version": 1, "dsn_datos": dsns["datos"], "dsn_frontera": dsns["frontera"],
            "directorio": "/vec-documentos-custodia", "retencion_minima_dias": retencion,
            "clave_correlacion_archivo": nombres_clave[0],
            "clave_seudonimo_archivo": nombres_clave[1],
            "clave_solicitud_archivo": nombres_clave[2],
            "analizador": {"endpoint": "https://analizador.vec.local:19055",
                            "nombre_servidor": "analizador.vec.local",
                            "ruta_ca": "/vec-analizador-ca.crt",
                            "ruta_certificado": "/vec-analizador-cliente.crt",
                            "ruta_clave": "/vec-analizador-cliente.key",
                            "conector_id": "vec-analizador-local"}}
    escribir_exclusivo(base / "subidas.json",
                       (json.dumps(conf, ensure_ascii=False, separators=(",", ":")) + "\n").encode(), aplicar)
    plan = {"version": 1, "directorio_rw": {"origen": str(ensayo / "documentos/custodia"),
            "destino": "/vec-documentos-custodia"}, "certificados_ro": [
                {"origen": str(ensayo / "servicios/analizador/tls" / nombre), "destino": destino}
                for nombre, destino in (("ca.crt", "/vec-analizador-ca.crt"),
                                        ("cliente.crt", "/vec-analizador-cliente.crt"),
                                        ("cliente.key", "/vec-analizador-cliente.key"))]}
    escribir_exclusivo(ensayo / "mountplan-documentos-aspirante.json",
                       (json.dumps(plan, ensure_ascii=False, separators=(",", ":")) + "\n").encode(), aplicar)


def main():
    p = argparse.ArgumentParser(description=__doc__)
    p.add_argument("--ensayo", type=Path, required=True)
    p.add_argument("--pg", required=True)
    p.add_argument("--aplicar", action="store_true")
    a = p.parse_args()
    raiz = Path(__file__).resolve().parents[4]
    exigir(a.pg.startswith("vec-codexo-aspirante-pg-") and re.fullmatch(r"[a-z0-9-]+", a.pg) and
           a.ensayo.is_absolute() and raiz not in a.ensayo.parents, "solo_clon_privado")
    privado(a.ensayo, True)
    preparar_login(a.pg, a.aplicar)
    preparar(a.ensayo, a.aplicar)
    print("subida_documental=preparada")


if __name__ == "__main__":
    try:
        main()
    except (OSError, ValueError, KeyError, TypeError, json.JSONDecodeError,
            subprocess.TimeoutExpired) as error:
        raise SystemExit("subida_documental=error clave=" + str(error).split(":", 1)[0]) from None
