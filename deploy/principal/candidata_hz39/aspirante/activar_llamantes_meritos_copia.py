#!/usr/bin/env python3
"""Admite dos LOGIN de Méritos en Identidad para la copia ficticia O."""

import argparse
from pathlib import Path
import re

from publicar_roles_meritos_copia import exigir, literal, psql


LLAMANTES = (
    ("vec_externo_meritos_cartera", "vec_meritos_cartera_externo"),
    ("vec_externo_meritos_registro_autorizacion",
     "vec_autorizacion_registro_meritos_externo"),
)


def consultar(pg):
    filas = psql(pg, "SELECT login::text || '|' || grupo_esperado::text || '|' || "
                 "aprobacion_ref FROM vec_identidad_externa_v1.llamante_autorizacion "
                 "WHERE login IN ('vec_externo_meritos_cartera', "
                 "'vec_externo_meritos_registro_autorizacion') ORDER BY login;")
    return {fila.split("|", 1)[0]: tuple(fila.split("|", 1)[1].split("|", 1))
            for fila in filas.splitlines() if fila}


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--ensayo", type=Path, required=True)
    parser.add_argument("--pg", required=True)
    parser.add_argument("--referencia-datos", required=True)
    parser.add_argument("--referencia-registro", required=True)
    parser.add_argument("--aplicar", action="store_true")
    args = parser.parse_args()
    repo = Path(__file__).resolve().parents[4]
    exigir(args.pg == "vec-codexo-aspirante-pg-20261014" and
           args.ensayo.is_absolute() and args.ensayo.is_dir() and
           repo not in args.ensayo.parents and not args.ensayo.is_symlink() and
           not args.ensayo.stat().st_mode & 0o077,
           "solo_copia_ficticia_o")
    referencias = (args.referencia_datos, args.referencia_registro)
    exigir(all(re.fullmatch(r"apr_[A-Za-z0-9_-]{22,128}", ref) for ref in referencias)
           and referencias[0] != referencias[1], "referencias_de_copia_invalidas")
    esperadas = {login: (grupo, ref) for (login, grupo), ref in zip(LLAMANTES, referencias)}
    actuales = consultar(args.pg)
    exigir(all(actuales[login] == esperado for login, esperado in esperadas.items()
               if login in actuales), "llamante_preexistente_divergente")
    if len(actuales) == len(LLAMANTES):
        print("llamantes_meritos=preparados")
        return
    exigir(args.aplicar, "llamantes_meritos_pendientes")
    # Solo la copia experimental admite estas referencias como datos ficticios.
    # No representan una firma ni un acto de Identidad en cidonia.
    comprobaciones = []
    inserciones = []
    for login, grupo in LLAMANTES:
        login_lit, grupo_lit = literal(login), literal(grupo)
        comprobaciones.append(f"""NOT EXISTS (
          SELECT 1 FROM pg_roles l JOIN pg_auth_members m ON m.member=l.oid
          JOIN pg_roles g ON g.oid=m.roleid
          WHERE l.rolname={login_lit} AND g.rolname={grupo_lit}
            AND l.rolcanlogin AND l.rolinherit AND l.rolconfig IS NULL
            AND NOT(l.rolsuper OR l.rolcreatedb OR l.rolcreaterole OR
                    l.rolreplication OR l.rolbypassrls)
            AND NOT(g.rolcanlogin OR g.rolsuper OR g.rolcreatedb OR
                    g.rolcreaterole OR g.rolreplication OR g.rolbypassrls)
            AND g.rolconfig IS NULL AND m.inherit_option
            AND NOT m.set_option AND NOT m.admin_option
            AND (SELECT count(*) FROM pg_auth_members x WHERE x.member=l.oid)=1
        )""")
        grupo, ref = esperadas[login]
        inserciones.append(f"""INSERT INTO vec_identidad_externa_v1.llamante_autorizacion
          (login,grupo_esperado,aprobacion_ref)
          SELECT {login_lit},{grupo_lit},{literal(ref)}
          WHERE NOT EXISTS (SELECT 1 FROM vec_identidad_externa_v1.llamante_autorizacion
                            WHERE login={login_lit});""")
    sql = """BEGIN ISOLATION LEVEL SERIALIZABLE;
SET LOCAL ROLE vec_identidad_sesiones_v1_propietario;
SET LOCAL search_path=pg_catalog,pg_temp;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='3s';
SET LOCAL statement_timeout='15s';
SELECT pg_advisory_xact_lock(hashtextextended('vec:ensayo:o:identidad:meritos',0));
DO $pre$ BEGIN
 IF """ + " OR ".join(comprobaciones) + """ THEN
   RAISE EXCEPTION 'llamantes_meritos_topologia_divergente' USING ERRCODE='55000';
 END IF;
END $pre$;
""" + "\n".join(inserciones) + "\nCOMMIT;"
    psql(args.pg, sql)
    exigir(consultar(args.pg) == esperadas, "llamantes_meritos_postimagen_divergente")
    print("llamantes_meritos=preparados")


if __name__ == "__main__":
    try:
        main()
    except (OSError, ValueError, KeyError, TypeError) as error:
        raise SystemExit("llamantes_meritos=error clave=" + str(error).split(":", 1)[0]) from None
