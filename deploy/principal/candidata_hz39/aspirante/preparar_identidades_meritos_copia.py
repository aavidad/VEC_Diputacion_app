#!/usr/bin/env python3
"""Deriva fuentes sintéticas externas desde la proyección certificada del clon."""

import argparse
import copy
import hashlib
import json
import os
from pathlib import Path
import re
import stat
import subprocess


def exigir(condicion, clave):
    if not condicion:
        raise ValueError(clave)


def privado(ruta, directorio=False):
    dato = ruta.lstat()
    exigir(dato.st_uid == os.getuid() and not ruta.is_symlink() and not dato.st_mode & 0o077,
           "ruta_no_privada")
    exigir(stat.S_ISDIR(dato.st_mode) if directorio else stat.S_ISREG(dato.st_mode),
           "tipo_de_ruta_invalido")


def leer(ruta):
    privado(ruta)
    return json.loads(ruta.read_text())


def escribir(ruta, valor, aplicar):
    if ruta.exists():
        exigir(leer(ruta) == valor, "fuente_privada_divergente")
        return
    exigir(aplicar, "fuente_privada_pendiente")
    with ruta.open("x") as salida:
        json.dump(valor, salida, ensure_ascii=False, separators=(",", ":"))
        salida.write("\n")
    ruta.chmod(0o600)


def ref(prefijo, etiqueta, semilla):
    return prefijo + hashlib.sha256(("vec.ensayo.codexo.hz39." + etiqueta + "\0" + semilla).encode()).hexdigest()[:32]


def consultar_snapshot(pg, cuenta, perfil):
    exigir(re.fullmatch(r"cta_[A-Za-z0-9_-]{22,128}", cuenta) and
           re.fullmatch(r"prf_[A-Za-z0-9_-]{22,128}", perfil), "referencia_invalida")
    sql = ("SELECT v.snapshot::text FROM vec_contexto_actor_v1.contexto_externo_actual a "
           "JOIN vec_contexto_actor_v1.contexto_externo_versiones v USING(provision_ref,version) "
           "WHERE v.cuenta_ref='" + cuenta + "' AND v.perfil_ref='" + perfil +
           "' AND v.familia='candidato';")
    resultado = subprocess.run(["docker", "exec", "-i", pg, "psql", "-U", "postgres",
                                "-d", "postgres", "-X", "-w", "-qAt", "-v", "ON_ERROR_STOP=1"],
                               input=sql, capture_output=True, text=True, timeout=10, check=False)
    exigir(resultado.returncode == 0 and len(resultado.stdout.splitlines()) == 1,
           "snapshot_candidato_no_unico")
    return json.loads(resultado.stdout)


def snapshot_usuarios(base, etiqueta, persona, cuenta, perfil):
    nuevo = copy.deepcopy(base)
    nuevo["provision_ref"] = ref("pue_", etiqueta + ":provision", persona)
    nuevo["poblacion"] = "usuarios"
    nuevo["estado"] = "activo"
    nuevo["cuenta"]["referencia"] = cuenta
    nuevo["cuenta"]["version"] = 1
    nuevo["perfil"]["referencia"] = perfil
    nuevo["perfil"]["version"] = 1
    nuevo["contexto"]["referencia"] = ref("vca_", etiqueta + ":contexto", persona)
    nuevo["contexto"]["version"] = 1
    nuevo["vinculo_candidato"] = None
    return nuevo


def fuente_identidad(snapshot):
    return {"version": 1, "snapshot": snapshot,
            "preimagen": {"revision_control_rol": 0, "huella_control_rol": "",
                          "version_asignacion": 0, "huella_asignacion": "",
                          "version_contexto": 0, "huella_contexto": "",
                          "secuencia_motivos": 0}}


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--ensayo", type=Path, required=True)
    parser.add_argument("--pg", required=True)
    parser.add_argument("--aplicar", action="store_true")
    args = parser.parse_args()
    exigir(args.pg.startswith("vec-codexo-aspirante-pg-") and re.fullmatch(r"[a-z0-9-]+", args.pg) and
           args.ensayo.is_absolute() and Path(__file__).resolve().parents[4] not in args.ensayo.parents,
           "solo_clon_aspirante")
    privado(args.ensayo, True)
    externo = leer(args.ensayo / "externo/identidad/bolsa-candidato.json")
    usuarios_original = leer(args.ensayo / "material/identidad/usuarios-preferencias-externa.json")
    exigir(len(usuarios_original["cuentas"]) == 1 and
           usuarios_original["cuentas"][0]["sujeto"] == externo["sujeto"] and
           usuarios_original["cuentas"][0]["certificado_sha256"] ==
           leer(args.ensayo / "externo/identidad/candidato.json")["certificate_sha256"],
           "usuarios_no_corresponden_al_candidato")
    base = consultar_snapshot(args.pg, externo["cuenta_ref"], externo["perfil_ref"])
    persona = externo["persona_ref"]
    exigir(base["persona"]["referencia"] == persona and
           base["vinculo_candidato"]["candidato_ref"] == externo["candidato_ref"],
           "snapshot_base_divergente")
    actos = args.ensayo / "incorporacion/actos"
    privado(actos, True)
    cuentas = {}
    for etiqueta in ("usuarios", "meritos"):
        cuenta = ref("cta_", etiqueta + ":cuenta", persona)
        perfil = ref("prf_", etiqueta + ":perfil", persona)
        cuentas[etiqueta] = (cuenta, perfil)
        snapshot = snapshot_usuarios(base, etiqueta, persona, cuenta, perfil)
        escribir(actos / ("snapshot-" + etiqueta + "-externo.json"), snapshot, args.aplicar)
        escribir(actos / ("fuente-identidad-" + etiqueta + "-externo.json"),
                fuente_identidad(snapshot), args.aplicar)
    exigir(len({externo["cuenta_ref"], *[x[0] for x in cuentas.values()]}) == 3 and
           len({externo["perfil_ref"], *[x[1] for x in cuentas.values()]}) == 3,
           "cuentas_o_perfiles_reutilizados")
    usuarios = copy.deepcopy(usuarios_original)
    usuarios["cuentas"][0]["cuenta_ref"], usuarios["cuentas"][0]["perfil_ref"] = cuentas["usuarios"]
    escribir(actos / "usuarios-preferencias-externa.preparado.json", usuarios, args.aplicar)
    meritos = {"version": 1, "cuentas": [{
        "certificado_sha256": usuarios_original["cuentas"][0]["certificado_sha256"],
        "cuenta_ref": cuentas["meritos"][0], "perfil_ref": cuentas["meritos"][1],
    }]}
    escribir(actos / "meritos-cuentas.preparado.json", meritos, args.aplicar)
    print("identidades_usuarios_meritos=preparadas")


if __name__ == "__main__":
    try:
        main()
    except (OSError, ValueError, KeyError, TypeError, json.JSONDecodeError,
            subprocess.TimeoutExpired) as error:
        raise SystemExit("identidades_usuarios_meritos=error clave=" +
                         str(error).split(":", 1)[0]) from None
