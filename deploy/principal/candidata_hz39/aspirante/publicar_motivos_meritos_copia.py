#!/usr/bin/env python3
"""Publica dos catálogos V2 sintéticos de motivos en el clon O."""

import argparse
from datetime import datetime, timedelta, timezone
import hashlib
import json
from pathlib import Path
import re
import subprocess


CATALOGOS = ("motivos_autorizacion_meritos_cartera", "motivos_meritos_declaracion")


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
        raise ValueError("sql_motivos_rechazado_" + (codigo.group(0) if codigo else "sin_sqlstate"))
    return r.stdout.strip()


def catalogo_sintetico(catalogo_id):
    desde = (datetime.now(timezone.utc) - timedelta(minutes=1)).replace(microsecond=0)
    clave = "motivo_" + hashlib.sha256(("vec:ensayo:meritos:" + catalogo_id).encode()).hexdigest()[:32]
    entrada = {"clave": clave, "vigente_desde": desde.strftime("%Y-%m-%dT%H:%M:%S.%fZ"),
               "vigente_hasta": None}
    canon = {"catalogo_id": catalogo_id, "catalogo_version": 1, "entradas": [entrada]}
    huella = hashlib.sha256(json.dumps(canon, sort_keys=True, separators=(",", ":"),
                                      ensure_ascii=False).encode()).hexdigest()
    return {**canon, "catalogo_huella_publicada_sha256": huella}


def privado(ruta, valor, aplicar):
    if ruta.exists():
        exigir(not ruta.is_symlink() and not ruta.stat().st_mode & 0o077 and
               json.loads(ruta.read_text()) == valor, "catalogo_privado_divergente")
        return
    exigir(aplicar, "catalogo_privado_pendiente")
    with ruta.open("x") as salida:
        json.dump(valor, salida, ensure_ascii=False, separators=(",", ":"))
        salida.write("\n")
    ruta.chmod(0o600)


def publicado(pg, nombre):
    q = ("SELECT c.catalogo_huella_publicada_sha256,e.entrada_clave "
         "FROM vec_autorizacion.motivo_v2_catalogo_publicado c "
         "JOIN vec_autorizacion.motivo_v2_entrada e USING(catalogo_id,catalogo_version) "
         "WHERE c.catalogo_id=" + literal(nombre) + " AND c.catalogo_version=1;")
    return psql(pg, q)


def main():
    p = argparse.ArgumentParser(description=__doc__)
    p.add_argument("--ensayo", type=Path, required=True)
    p.add_argument("--pg", required=True)
    p.add_argument("--aplicar", action="store_true")
    a = p.parse_args()
    exigir(a.pg.startswith("vec-codexo-aspirante-pg-") and re.fullmatch(r"[a-z0-9-]+", a.pg) and
           a.ensayo.is_absolute() and Path(__file__).resolve().parents[4] not in a.ensayo.parents and
           not a.ensayo.stat().st_mode & 0o077, "solo_clon_privado")
    actos = a.ensayo / "incorporacion/actos"
    referencias = {}
    for nombre in CATALOGOS:
        ruta = actos / ("catalogo-" + nombre + ".json")
        if ruta.exists():
            exigir(not ruta.is_symlink() and not ruta.stat().st_mode & 0o077,
                   "catalogo_no_privado")
            catalogo = json.loads(ruta.read_text())
        else:
            exigir(a.aplicar, "catalogo_pendiente")
            catalogo = catalogo_sintetico(nombre)
            privado(ruta, catalogo, a.aplicar)
        canon = {k: catalogo[k] for k in ("catalogo_id", "catalogo_version", "entradas")}
        huella = hashlib.sha256(json.dumps(canon, sort_keys=True, separators=(",", ":"),
                                          ensure_ascii=False).encode()).hexdigest()
        exigir(catalogo["catalogo_id"] == nombre and catalogo["catalogo_version"] == 1 and
               catalogo["catalogo_huella_publicada_sha256"] == huella and
               len(catalogo["entradas"]) == 1, "catalogo_motivos_divergente")
        referencia = {"catalogo_id": nombre, "catalogo_version": 1,
                      "catalogo_huella_sha256": huella,
                      "entrada_clave": catalogo["entradas"][0]["clave"]}
        referencias[nombre] = referencia
        actual = publicado(a.pg, nombre)
        if actual:
            exigir(actual == huella + "|" + referencia["entrada_clave"],
                   "motivo_actual_divergente")
            continue
        exigir(a.aplicar, "motivo_pendiente")
        secuencia = int(psql(a.pg, "SELECT ultima_secuencia+1 FROM "
                            "vec_autorizacion.motivo_v2_checkpoint_origen WHERE control_id=true;"))
        evento_huella = hashlib.sha256((huella + ":" + str(secuencia)).encode()).hexdigest()
        evento = "evento_" + evento_huella[:32]
        entradas = literal(json.dumps(catalogo["entradas"], ensure_ascii=False,
                                      separators=(",", ":")))
        sql = ("BEGIN ISOLATION LEVEL SERIALIZABLE; SET LOCAL search_path=pg_catalog; "
               "SET LOCAL timezone='UTC'; SET LOCAL lock_timeout='3s'; "
               "SELECT pg_advisory_xact_lock(hashtextextended('vec:ensayo:meritos:motivos',0)); "
               "DO $pre$ BEGIN IF (SELECT ultima_secuencia FROM "
               "vec_autorizacion.motivo_v2_checkpoint_origen WHERE control_id=true)<>" +
               str(secuencia - 1) + " THEN RAISE EXCEPTION 'checkpoint_motivos_divergente' "
               "USING ERRCODE='55000'; END IF; END $pre$; "
               "SELECT vec_autorizacion.publicar_motivos_autorizacion_v2(" +
               literal(evento) + "," + str(secuencia) + "," + literal(evento_huella) + "," +
               literal(nombre) + ",1," + literal(huella) + "," +
               literal(catalogo["entradas"][0]["vigente_desde"]) + "::timestamptz," +
               entradas + "::jsonb); COMMIT;")
        recibo = psql(a.pg, sql)
        exigir("t" in recibo.splitlines() and
               publicado(a.pg, nombre) == huella + "|" + referencia["entrada_clave"],
               "motivo_no_cotejado")
    resultado = {"version": 1, "motivo_retirada": referencias[CATALOGOS[0]],
                 "motivo_declaracion": referencias[CATALOGOS[1]]}
    privado(actos / "motivos-meritos.preparado.json", resultado, a.aplicar)
    print("motivos_meritos=publicados")


if __name__ == "__main__":
    try:
        main()
    except (OSError, ValueError, KeyError, TypeError, json.JSONDecodeError,
            subprocess.TimeoutExpired) as error:
        raise SystemExit("motivos_meritos=error clave=" +
                         str(error).split(":", 1)[0]) from None
