#!/usr/bin/env python3
"""Prepara los siete LOGIN runtime de Méritos en el clon sintético O."""

import argparse
from pathlib import Path
import re
import subprocess
from urllib.parse import parse_qs, urlsplit, urlunsplit


LOGIN = {
    "operacion.dsn": ("vec_externo_meritos_operacion", ("vec_meritos_ejecutor", "vec_meritos_externo")),
    "cartera.dsn": ("vec_externo_meritos_cartera", ("vec_meritos_cartera_externo",)),
    "frontera.dsn": ("vec_externo_meritos_frontera", ("vec_meritos_frontera_externo",)),
    "fuente_autorizacion.dsn": ("vec_externo_meritos_fuente_autorizacion", ("vec_autorizacion_fuente_meritos_externa",)),
    "registro_autorizacion.dsn": ("vec_externo_meritos_registro_autorizacion", ("vec_autorizacion_registro_meritos_externo",)),
    "motivos_autorizacion.dsn": ("vec_externo_meritos_motivos_autorizacion", ("vec_autorizacion_motivos_meritos_externo",)),
    "contexto.dsn": ("vec_externo_meritos_contexto", ("vec_contexto_actor_v1_usuarios_externo",)),
}


def exigir(condicion, clave):
    if not condicion:
        raise ValueError(clave)


def psql(pg, sql):
    r = subprocess.run(["docker", "exec", "-i", pg, "psql", "-U", "postgres", "-d", "postgres",
                        "-X", "-w", "-qAt", "-v", "ON_ERROR_STOP=1", "-v", "VERBOSITY=sqlstate"],
                       input=sql, text=True, capture_output=True, timeout=20, check=False)
    if r.returncode:
        codigo = re.search(r"\b[0-9]{2}[A-Z0-9]{3}\b", r.stderr)
        raise ValueError("sql_login_rechazado_" + (codigo.group(0) if codigo else "sin_sqlstate"))
    return r.stdout.strip()


def preparar_sql():
    partes = ["BEGIN;", "SET LOCAL search_path=pg_catalog;", "SET LOCAL lock_timeout='3s';",
              "SELECT pg_advisory_xact_lock(hashtextextended('vec:ensayo:meritos:login',0));"]
    for login, grupos in LOGIN.values():
        partes.append(f"DO $g$ BEGIN IF to_regrole('{login}') IS NULL THEN "
                      f"CREATE ROLE {login} LOGIN INHERIT NOSUPERUSER NOCREATEDB NOCREATEROLE "
                      f"NOREPLICATION NOBYPASSRLS; END IF; END $g$;")
        for grupo in grupos:
            partes.append(f"DO $g$ BEGIN IF NOT EXISTS(SELECT 1 FROM pg_auth_members "
                          f"WHERE member=to_regrole('{login}') AND roleid=to_regrole('{grupo}')) "
                          f"THEN GRANT {grupo} TO {login} WITH INHERIT TRUE, SET FALSE, ADMIN FALSE; "
                          f"END IF; END $g$;")
    partes.append("COMMIT;")
    return "\n".join(partes)


def comprobar_pg(pg):
    nombres = ",".join("'" + login + "'" for login, _ in LOGIN.values())
    q = ("SELECT l.rolname,l.rolcanlogin,l.rolinherit,l.rolsuper,l.rolcreatedb,l.rolcreaterole,"
         "l.rolreplication,l.rolbypassrls,count(m.roleid),"
         "coalesce(string_agg(g.rolname||':'||m.inherit_option||':'||m.set_option||':'||m.admin_option,',' ORDER BY g.rolname),'') "
         "FROM pg_roles l LEFT JOIN pg_auth_members m ON m.member=l.oid "
         "LEFT JOIN pg_roles g ON g.oid=m.roleid WHERE l.rolname IN (" + nombres + ") "
         "GROUP BY l.rolname,l.rolcanlogin,l.rolinherit,l.rolsuper,l.rolcreatedb,l.rolcreaterole,"
         "l.rolreplication,l.rolbypassrls ORDER BY l.rolname;")
    encontrados = {}
    for linea in psql(pg, q).splitlines():
        partes = linea.split("|")
        exigir(len(partes) == 10, "login_diagnostico_invalido")
        encontrados[partes[0]] = partes
    exigir(len(encontrados) == len(LOGIN), "login_meritos_ausente")
    for login, grupos in LOGIN.values():
        dato = encontrados[login]
        membresias = ",".join(grupo + ":true:false:false" for grupo in sorted(grupos))
        exigir(dato[1:8] == ["t", "t", "f", "f", "f", "f", "f"] and
               int(dato[8]) == len(grupos) and dato[9] == membresias,
               "login_meritos_no_nominal")


def preparar_dsn(ensayo):
    referencia = ensayo / "incorporacion/externo-dsn/preflight.dsn"
    dsn = urlsplit(referencia.read_text().strip())
    claves = parse_qs(dsn.query)
    exigir(dsn.scheme == "postgresql" and dsn.hostname == "localhost" and dsn.port == 5432 and
           dsn.path == "/postgres" and
           claves.get("sslmode") == ["verify-full"] and
           claves.get("sslrootcert") == ["/vec-pg-ca.crt"], "dsn_base_invalida")
    destino = ensayo / "incorporacion/externo-dsn/meritos"
    destino.mkdir(mode=0o700, exist_ok=True)
    exigir(not destino.stat().st_mode & 0o077, "directorio_dsn_no_privado")
    for nombre, (usuario, _) in LOGIN.items():
        valor = urlunsplit(("postgresql", usuario + "@127.0.0.1:5432", "/postgres",
                            "sslmode=verify-full&sslrootcert=/vec-pg-ca.crt", "")) + "\n"
        archivo = destino / nombre
        if archivo.exists():
            exigir(not archivo.is_symlink() and not archivo.stat().st_mode & 0o077 and
                   archivo.read_text() == valor, "dsn_meritos_divergente")
        else:
            with archivo.open("x") as salida:
                salida.write(valor)
            archivo.chmod(0o600)


def main():
    p = argparse.ArgumentParser(description=__doc__)
    p.add_argument("--ensayo", type=Path, required=True)
    p.add_argument("--pg", required=True)
    p.add_argument("--aplicar", action="store_true")
    a = p.parse_args()
    exigir(a.pg.startswith("vec-codexo-aspirante-pg-") and re.fullmatch(r"[a-z0-9-]+", a.pg) and
           a.ensayo.is_absolute() and Path(__file__).resolve().parents[4] not in a.ensayo.parents and
           not a.ensayo.stat().st_mode & 0o077, "solo_clon_privado")
    if a.aplicar:
        psql(a.pg, preparar_sql())
    comprobar_pg(a.pg)
    if a.aplicar:
        preparar_dsn(a.ensayo)
    print("login_meritos=preparados")


if __name__ == "__main__":
    try:
        main()
    except (OSError, ValueError, KeyError, TypeError, subprocess.TimeoutExpired) as error:
        raise SystemExit("login_meritos=error clave=" + str(error).split(":", 1)[0]) from None
