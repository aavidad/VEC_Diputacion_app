#!/usr/bin/env python3
"""Publica snapshots CTX15 sintéticos por su función de gobierno nominal."""

import argparse
import json
import os
from pathlib import Path
import re
import subprocess


def exigir(condicion, clave):
    if not condicion:
        raise ValueError(clave)


def literal(valor):
    exigir(isinstance(valor, str) and "\x00" not in valor, "texto_invalido")
    return "'" + valor.replace("'", "''") + "'"


def admin(pg, sql):
    r = subprocess.run(["docker", "exec", "-i", pg, "psql", "-U", "postgres", "-d", "postgres",
                        "-X", "-w", "-qAt", "-v", "ON_ERROR_STOP=1"], input=sql,
                       text=True, capture_output=True, timeout=15, check=False)
    exigir(r.returncode == 0, "consulta_ctx15_fallida")
    return r.stdout.strip()


def nominal(pg, ca, sql):
    r = subprocess.run(["docker", "run", "--rm", "-i", "--network", "container:" + pg,
                        "-v", str(ca) + ":/vec-pg-ca.crt:ro",
                        "-e", "PGHOST=127.0.0.1", "-e", "PGPORT=5432",
                        "-e", "PGDATABASE=postgres", "-e", "PGUSER=vec_prov_externo_contexto",
                        "-e", "PGSSLMODE=verify-full", "-e", "PGSSLROOTCERT=/vec-pg-ca.crt",
                        "postgres:18.4", "psql", "-X", "-w", "-qAt", "-v", "ON_ERROR_STOP=1",
                        "-v", "VERBOSITY=sqlstate"], input=sql, text=True,
                       capture_output=True, timeout=30, check=False)
    if r.returncode:
        codigo = re.search(r"\b[0-9]{2}[A-Z0-9]{3}\b", r.stderr)
        raise ValueError("publicacion_ctx15_rechazada_" + (codigo.group(0) if codigo else "sin_sqlstate"))
    return r.stdout.strip()


def actual(pg, ref):
    q = ("SELECT v.snapshot::text,v.huella_sha256 FROM vec_contexto_actor_v1.contexto_externo_actual a "
         "JOIN vec_contexto_actor_v1.contexto_externo_versiones v USING(provision_ref,version) "
         "WHERE v.provision_ref=" + literal(ref) + ";")
    dato = admin(pg, q)
    if not dato:
        return None
    texto, huella = dato.split("|", 1)
    return json.loads(texto), huella


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
    actos = a.ensayo / "incorporacion/actos"
    for etiqueta in ("usuarios", "meritos"):
        snapshot = json.loads((actos / ("snapshot-" + etiqueta + "-externo.json")).read_text())
        ref = snapshot["provision_ref"]
        exigir(re.fullmatch(r"pue_[A-Za-z0-9_-]{22,128}", ref) and
               snapshot["poblacion"] == "usuarios" and snapshot["vinculo_candidato"] is None,
               "snapshot_privado_invalido")
        existente = actual(a.pg, ref)
        if existente is not None:
            exigir(existente[0] == snapshot, "snapshot_actual_divergente")
            continue
        exigir(a.aplicar, "snapshot_usuarios_pendiente")
        canon = json.dumps(snapshot, ensure_ascii=False, separators=(",", ":"))
        huella = admin(a.pg, "SELECT vec_contexto_actor_v1.huella_snapshot_contexto_externo_v1(" +
                       literal(canon) + "::jsonb,1);")
        exigir(re.fullmatch(r"[0-9a-f]{64}", huella), "huella_ctx15_invalida")
        sql = ("BEGIN ISOLATION LEVEL SERIALIZABLE; SET LOCAL search_path=pg_catalog; "
               "SET LOCAL timezone='UTC'; SET LOCAL lock_timeout='3s'; "
               "SELECT provision_ref,version,huella_sha256 FROM "
               "vec_contexto_actor_v1.publicar_snapshot_contexto_externo_v1(" +
               literal(canon) + "::jsonb,0,NULL," + literal(huella) + "); COMMIT;")
        recibo = nominal(a.pg, a.ca, sql)
        exigir(huella in recibo and actual(a.pg, ref) == (snapshot, huella),
               "recibo_ctx15_no_cotejado")
    print("contextos_usuarios_meritos=publicados")


if __name__ == "__main__":
    try:
        main()
    except (OSError, ValueError, KeyError, TypeError, json.JSONDecodeError,
            subprocess.TimeoutExpired) as error:
        raise SystemExit("contextos_usuarios_meritos=error clave=" +
                         str(error).split(":", 1)[0]) from None
