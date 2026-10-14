#!/usr/bin/env python3
"""Publica ID7 de Usuarios y Méritos mediante el CLI nominal existente."""

import argparse
import json
import os
from pathlib import Path
import re
import subprocess
from urllib.parse import urlunsplit


def exigir(condicion, clave):
    if not condicion:
        raise ValueError(clave)


def consultar(pg, cuenta):
    exigir(re.fullmatch(r"cta_[A-Za-z0-9_-]{22,128}", cuenta), "cuenta_invalida")
    sql = ("SELECT coalesce((SELECT CASE WHEN a.cuenta_ref IS NULL THEN 'cuenta_sin_alias' "
           "ELSE encode(a.cuenta_id_hmac,'hex')||':'||encode(a.sujeto_id_hmac,'hex')||':'||"
           "coalesce(e.estado,'sin_estado') END FROM vec_identidad_externa_v1.cuenta c "
           "LEFT JOIN vec_identidad_externa_v1.alias_cuenta a USING(cuenta_ref) "
           "LEFT JOIN vec_identidad_externa_v1.estado_actual p USING(cuenta_ref) "
           "LEFT JOIN vec_identidad_externa_v1.estado_cuenta e ON e.cuenta_ref=p.cuenta_ref "
           "AND e.revision=p.revision WHERE c.cuenta_ref='" + cuenta + "'),'');")
    r = subprocess.run(["docker", "exec", "-i", pg, "psql", "-U", "postgres", "-d", "postgres",
                        "-X", "-w", "-qAt", "-v", "ON_ERROR_STOP=1"], input=sql,
                       text=True, capture_output=True, timeout=10, check=False)
    exigir(r.returncode == 0, "consulta_id7_fallida")
    return r.stdout.strip()


def cli(pg, ca, binario, actos, etiqueta, plan=None):
    cmd = ["docker", "run", "--rm", "--network", "container:" + pg,
           "-v", str(actos) + ":/vec-actos:ro", "-v", str(binario) + ":/vec-cli:ro",
           "-v", str(ca) + ":/vec-pg-ca.crt:ro",
           "-e", "VEC_PORTAL_PROCESO=interno", "-e", "VEC_EXECUTION_PROFILE=desarrollo",
           "-e", "VEC_AUTH_MODE=desarrollo",
           "-e", "VEC_DEVELOPMENT_GUARD=ACEPTO_CREDENCIALES_NO_AUTORITATIVAS_SOLO_DESARROLLO",
           "postgres:18.4", "/vec-cli", "--fuente",
           "/vec-actos/fuente-identidad-" + etiqueta + "-externo.json", "--fase", "identidad",
           "--alias-archivo", "/vec-actos/alias-" + etiqueta + "-externo.json"]
    if plan is not None:
        cmd.extend(["--dsn-archivo", "/vec-actos/gobierno-identidad.dsn",
                    "--aprobar", plan["huella_sha256"], "--preimagen", plan["preimagen_sha256"]])
    r = subprocess.run(cmd, capture_output=True, text=True, timeout=45, check=False)
    exigir(r.returncode == 0, "cli_id7_no_confirmado_reconciliar")
    return json.loads(r.stdout)


def main():
    p = argparse.ArgumentParser(description=__doc__)
    p.add_argument("--ensayo", type=Path, required=True)
    p.add_argument("--pg", required=True)
    p.add_argument("--ca", type=Path, required=True)
    p.add_argument("--binario", type=Path, required=True)
    p.add_argument("--aplicar", action="store_true")
    a = p.parse_args()
    exigir(a.pg.startswith("vec-codexo-aspirante-pg-") and re.fullmatch(r"[a-z0-9-]+", a.pg) and
           a.ensayo.is_absolute() and Path(__file__).resolve().parents[4] not in a.ensayo.parents and
           not a.ensayo.stat().st_mode & 0o077 and a.ca.is_file() and a.binario.is_file(),
           "solo_clon_privado")
    actos = a.ensayo / "incorporacion/actos"
    dsn = actos / "gobierno-identidad.dsn"
    contenido = (urlunsplit(("postgresql", "vec_prov_externo_identidad@127.0.0.1:5432",
                             "/postgres", "sslmode=verify-full&sslrootcert=/vec-pg-ca.crt", "")) + "\n")
    if dsn.exists():
        exigir(not dsn.is_symlink() and not dsn.stat().st_mode & 0o077 and
               dsn.read_text() == contenido, "dsn_gobierno_divergente")
    else:
        exigir(a.aplicar, "dsn_gobierno_pendiente")
        with dsn.open("x") as salida:
            salida.write(contenido)
        dsn.chmod(0o600)
    for etiqueta in ("usuarios", "meritos"):
        fuente = json.loads((actos / ("fuente-identidad-" + etiqueta + "-externo.json")).read_text())
        alias = json.loads((actos / ("alias-" + etiqueta + "-externo.json")).read_text())
        cuenta = fuente["snapshot"]["cuenta"]["referencia"]
        esperado = [x for x in alias["cuentas"] if x["cuenta_ref"] == cuenta]
        exigir(len(esperado) == 1, "alias_privado_no_unico")
        huellas = (esperado[0]["cuenta_id_hmac"] + ":" +
                   esperado[0]["sujeto_id_hmac"] + ":activa")
        actual = consultar(a.pg, cuenta)
        if actual:
            exigir(actual == huellas, "identidad_externa_divergente")
            continue
        exigir(a.aplicar, "identidad_externa_pendiente")
        plan = cli(a.pg, a.ca, a.binario, actos, etiqueta)
        exigir(plan.get("fase") == "identidad" and plan.get("estado") == "preparado" and
               re.fullmatch(r"[0-9a-f]{64}", plan.get("huella_sha256", "")) and
               re.fullmatch(r"[0-9a-f]{64}", plan.get("preimagen_sha256", "")),
               "plan_id7_invalido")
        recibo = cli(a.pg, a.ca, a.binario, actos, etiqueta, plan)
        exigir(recibo.get("fase") == "identidad" and recibo.get("estado") == "confirmado" and
               consultar(a.pg, cuenta) == huellas, "recibo_id7_no_cotejado")
    print("identidades_usuarios_meritos=publicadas")


if __name__ == "__main__":
    try:
        main()
    except (OSError, ValueError, KeyError, TypeError, json.JSONDecodeError,
            subprocess.TimeoutExpired) as error:
        raise SystemExit("identidades_usuarios_meritos=error clave=" +
                         str(error).split(":", 1)[0]) from None
