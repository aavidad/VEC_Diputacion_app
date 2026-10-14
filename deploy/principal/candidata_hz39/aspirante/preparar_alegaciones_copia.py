#!/usr/bin/env python3
"""Prepara el motivo y material de Alegaciones ya definido para el clon O."""

import argparse
import hashlib
import json
from pathlib import Path
import re
import subprocess


def exigir(condicion, clave):
    if not condicion:
        raise ValueError(clave)


def literal(valor):
    exigir(isinstance(valor, str) and "\x00" not in valor, "texto_invalido")
    return "'" + valor.replace("'", "''") + "'"


def psql(pg, sql):
    r = subprocess.run(["docker", "exec", "-i", pg, "psql", "-U", "postgres", "-d", "postgres",
                        "-X", "-w", "-qAt", "-v", "ON_ERROR_STOP=1", "-v", "VERBOSITY=sqlstate"],
                       input=sql, text=True, capture_output=True, timeout=20, check=False)
    if r.returncode:
        codigo = re.search(r"\b[0-9]{2}[A-Z0-9]{3}\b", r.stderr)
        raise ValueError("sql_alegaciones_rechazado_" + (codigo.group(0) if codigo else "sin_sqlstate"))
    return r.stdout.strip()


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
    fuente = repo / "deploy/principal/alegaciones_bolsa"
    catalogo = json.loads((fuente / "motivos_alegaciones_bolsa_v1.json").read_text())
    motivo = json.loads((fuente / "bolsa-alegaciones.ejemplo.json").read_text())
    referencia = motivo["motivo_presentacion"]
    canon = {k: catalogo[k] for k in ("catalogo_id", "catalogo_version", "entradas")}
    huella = hashlib.sha256(json.dumps(canon, sort_keys=True, separators=(",", ":"),
                                      ensure_ascii=False).encode()).hexdigest()
    exigir(huella == catalogo["catalogo_huella_publicada_sha256"] ==
           referencia["catalogo_huella_sha256"] and
           catalogo["catalogo_id"] == referencia["catalogo_id"] and
           catalogo["catalogo_version"] == referencia["catalogo_version"] and
           len(catalogo["entradas"]) == 1 and
           catalogo["entradas"][0]["clave"] == referencia["entrada_clave"],
           "fuente_alegaciones_divergente")
    inventario = json.loads((a.ensayo / "externo/externo/v3/inventario.json").read_text())
    audiencias = [v["audiencia"] for v in inventario["consumidores"].get("inscripcion_bolsa", [])]
    exigir(audiencias == ["vec_bolsa_llamamientos.inscripcion.presentar.v1",
                          "vec_bolsa_llamamientos.alegacion.presentar.v1"],
           "audiencia_alegaciones_ausente")
    actual = psql(a.pg, "SELECT c.catalogo_huella_publicada_sha256,e.entrada_clave "
                  "FROM vec_autorizacion.motivo_v2_catalogo_publicado c "
                  "JOIN vec_autorizacion.motivo_v2_entrada e USING(catalogo_id,catalogo_version) "
                  "WHERE c.catalogo_id=" + literal(catalogo["catalogo_id"]) +
                  " AND c.catalogo_version=" + str(catalogo["catalogo_version"]) + ";")
    esperado = huella + "|" + referencia["entrada_clave"]
    if actual:
        exigir(actual == esperado, "catalogo_alegaciones_actual_divergente")
    else:
        exigir(a.aplicar, "catalogo_alegaciones_pendiente")
        secuencia = int(psql(a.pg, "SELECT ultima_secuencia+1 FROM "
                            "vec_autorizacion.motivo_v2_checkpoint_origen WHERE control_id=true;"))
        evento_huella = hashlib.sha256((huella + ":" + str(secuencia)).encode()).hexdigest()
        entradas = literal(json.dumps(catalogo["entradas"], ensure_ascii=False,
                                      separators=(",", ":")))
        sql = ("BEGIN ISOLATION LEVEL SERIALIZABLE; SET LOCAL search_path=pg_catalog; "
               "SET LOCAL timezone='UTC'; SET LOCAL lock_timeout='3s'; "
               "SELECT pg_advisory_xact_lock(hashtextextended('vec:ensayo:alegaciones:motivos',0)); "
               "DO $pre$ BEGIN IF (SELECT ultima_secuencia FROM vec_autorizacion.motivo_v2_checkpoint_origen "
               "WHERE control_id=true)<>" + str(secuencia - 1) +
               " THEN RAISE EXCEPTION 'checkpoint_alegaciones_divergente' USING ERRCODE='55000'; "
               "END IF; END $pre$; SELECT vec_autorizacion.publicar_motivos_autorizacion_v2(" +
               literal("evento_" + evento_huella[:32]) + "," + str(secuencia) + "," +
               literal(evento_huella) + "," + literal(catalogo["catalogo_id"]) + "," +
               str(catalogo["catalogo_version"]) + "," + literal(huella) + "," +
               literal(catalogo["entradas"][0]["vigente_desde"]) + "::timestamptz," +
               entradas + "::jsonb); COMMIT;")
        exigir("t" in psql(a.pg, sql).splitlines(), "motivo_alegaciones_no_confirmado")
    destino = a.ensayo / "externo/identidad/bolsa-alegaciones.json"
    if destino.exists():
        exigir(not destino.is_symlink() and not destino.stat().st_mode & 0o077 and
               json.loads(destino.read_text()) == motivo, "material_alegaciones_divergente")
    else:
        exigir(a.aplicar, "material_alegaciones_pendiente")
        with destino.open("x") as salida:
            json.dump(motivo, salida, ensure_ascii=False, separators=(",", ":"))
            salida.write("\n")
        destino.chmod(0o600)
    exigir(psql(a.pg, "SELECT count(*) FROM vec_autorizacion.motivo_v2_catalogo_publicado "
                "WHERE catalogo_id=" + literal(catalogo["catalogo_id"]) +
                " AND catalogo_version=" + str(catalogo["catalogo_version"]) +
                " AND catalogo_huella_publicada_sha256=" + literal(huella) + ";") == "1",
           "motivo_alegaciones_no_cotejado")
    print("alegaciones_material=preparado")


if __name__ == "__main__":
    try:
        main()
    except (OSError, ValueError, KeyError, TypeError, json.JSONDecodeError,
            subprocess.TimeoutExpired) as error:
        raise SystemExit("alegaciones_material=error clave=" +
                         str(error).split(":", 1)[0]) from None
