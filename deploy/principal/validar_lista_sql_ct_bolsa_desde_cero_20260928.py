#!/usr/bin/env python3
"""Comprueba cobertura rastreada y aristas causales del plan CT/Bolsa.

No ejecuta SQL ni acredita las preimágenes PostgreSQL.
"""
from collections import Counter
from pathlib import Path
import subprocess
import sys

RAIZ = Path(__file__).resolve().parents[2]
PLAN = Path(__file__).with_name(
    "lista_sql_ct_bolsa_llamamientos_desde_cero_20260928.txt"
)
RLS_B1 = (
    "deploy/postgresql/bolsa_llamamientos/dba/"
    "20260928_b1_rls_propietario/01_cerrar_politicas.sql"
)
FAMILIAS = ("contratacion_temporal", "bolsa_llamamientos")


def nombre(familia: str, prefijo: str, subdir: str = "migraciones") -> str:
    candidatas = sorted((RAIZ / "deploy/postgresql" / familia / subdir).glob(
        f"{prefijo}_*.up.sql"
    ))
    if len(candidatas) != 1:
        raise ValueError(f"{familia}/{subdir}/{prefijo}: {len(candidatas)} candidatas")
    return candidatas[0].relative_to(RAIZ).as_posix()


def validar() -> None:
    lineas = PLAN.read_text(encoding="utf-8").splitlines()
    rutas = [s.strip() for s in lineas if s.strip() and not s.lstrip().startswith("#")]
    duplicados = [ruta for ruta, veces in Counter(rutas).items() if veces != 1]
    if duplicados:
        raise ValueError(f"rutas duplicadas: {duplicados}")
    rastreados = set(
        subprocess.check_output(
            ["git", "-C", str(RAIZ), "ls-files", "deploy/postgresql"],
            text=True,
        ).splitlines()
    )
    for ruta in rutas:
        if not ruta.startswith("deploy/postgresql/") or ruta not in rastreados:
            raise ValueError(f"ruta no rastreada: {ruta}")
        if not (ruta.endswith(".up.sql") or ruta.endswith("_up.sql") or ruta == RLS_B1):
            raise ValueError(f"tipo de archivo fuera del plan: {ruta}")
    propios = {
        ruta
        for ruta in rastreados
        if any(ruta.startswith(f"deploy/postgresql/{familia}/") for familia in FAMILIAS)
        and (ruta.endswith(".up.sql") or ruta.endswith("_up.sql"))
    }
    presentes = propios.intersection(rutas)
    if presentes != propios:
        raise ValueError(f"faltan propios: {sorted(propios - presentes)}")
    if len(propios) != 210:
        raise ValueError(f"inventario propio cambió: {len(propios)}")
    indice = {ruta: i for i, ruta in enumerate(rutas)}
    aristas = [
        (nombre("autorizacion", "000004"), nombre("bolsa_llamamientos", "000001")),
        (nombre("bolsa_llamamientos", "000001"), RLS_B1),
        (RLS_B1, "deploy/postgresql/contexto_actor_v1/roles_contexto_corporativo_rrhh_selector_v1_up.sql"),
        ("deploy/postgresql/contexto_actor_v1/acl_tipos_preselector_c3_v1.up.sql",
         "deploy/postgresql/contexto_actor_v1/roles_contexto_corporativo_rrhh_selector_v1_up.sql"),
        (nombre("contexto_actor_v1", "000003"),
         "deploy/postgresql/contexto_actor_v1/roles_contexto_corporativo_rrhh_selector_v1_up.sql"),
        (nombre("autorizacion_atestada_v3", "000090"), nombre("bolsa_llamamientos", "000044")),
        (nombre("contratacion_temporal", "000138"), nombre("autorizacion_atestada_v3", "000104")),
        (nombre("autorizacion_atestada_v3", "000104"), nombre("contratacion_temporal", "000139")),
        (nombre("contratacion_temporal", "000139"), nombre("autorizacion_atestada_v3", "000105")),
        (nombre("autorizacion_atestada_v3", "000105"), nombre("contratacion_temporal", "000140")),
    ]
    for anterior, posterior in aristas:
        if anterior not in indice or posterior not in indice or indice[anterior] >= indice[posterior]:
            raise ValueError(f"orden inválido: {anterior} -> {posterior}")
    print(f"Cobertura estática: {len(propios)} propios, {len(rutas) - len(propios)} dependencias; sin duplicados")
    print("NO-GO dinámico: ACL CT/AD3 pendiente y huella Contexto4a incompatible")


if __name__ == "__main__":
    try:
        validar()
    except (OSError, ValueError, subprocess.CalledProcessError) as error:
        print(f"ERROR: {error}", file=sys.stderr)
        sys.exit(1)
