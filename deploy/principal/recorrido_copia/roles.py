"""LOGIN de origen exigidos por AD278 antes de instalar las migraciones."""

import json
import hashlib
import os
from pathlib import Path
import re
import secrets
import subprocess


ORIGENES = (
    ("VEC_CT_DATABASE_URL", "vec_contratacion_temporal_ejecutor",
     "vec_contratacion_temporal.vinculo_emision_bolsa.v1", "contratacion_temporal.bolsa.vincular"),
    ("VEC_BOLSA_LLAMAMIENTOS_DATABASE_URL", "vec_bolsa_llamamientos_ejecutor",
     "vec_bolsa_llamamientos.llamamiento.emitir.v1", "llamamiento.emitir.v1"),
)
NOMBRE = re.compile(r"vec_[a-z0-9_]{2,62}\Z")
HUELLA_PLANTILLA = "249aac6d2c196a160b05b5188a769c0b7dd0ab748041119d856fb3d66b394db1"


def preparar_roles(copia) -> None:
    """Prepara exclusivamente los dos LOGIN del origen en la copia nueva."""
    estado = copia.leer()
    if estado.get("preparada") or not copia.existe() or not copia.raiz.is_dir():
        raise ValueError("La copia debe estar recién restaurada y activa")
    destino = copia.raiz / "roles-previos.json"
    if destino.exists():
        raise ValueError("Los roles previos ya se prepararon")
    credenciales = {}
    for variable, _, audiencia, operacion in ORIGENES:
        consulta = ("SELECT login_nombre FROM vec_autorizacion_atestada_v3.configuracion_origen_consumos_v1 "
                    "WHERE audiencia_consumo='" + audiencia + "' AND operacion='" + operacion +
                    "' AND canal_permitido='interna_corporativa';")
        resultado = subprocess.run(
            ["docker", "exec", "-i", "-u", "postgres", copia.nombre, "psql", "-X", "-At",
             "-v", "ON_ERROR_STOP=1", "-U", "postgres", "-d", "postgres"],
            input=consulta.encode(), stdout=subprocess.PIPE, stderr=subprocess.PIPE,
            check=True, timeout=30)
        filas = resultado.stdout.decode().splitlines()
        if len(filas) != 1 or not NOMBRE.fullmatch(filas[0]):
            raise ValueError("Origen nominal AD278 ambiguo")
        credenciales[variable] = {"login": filas[0], "password": secrets.token_hex(32)}
    if credenciales[ORIGENES[0][0]]["login"] == credenciales[ORIGENES[1][0]]["login"]:
        raise ValueError("Los orígenes AD278 comparten un LOGIN")
    plantilla_bytes = (Path(__file__).parent / "datos/origenes_ad278.sql").read_bytes()
    if hashlib.sha256(plantilla_bytes).hexdigest() != HUELLA_PLANTILLA:
        raise ValueError("Contrato SQL de orígenes modificado")
    plantilla = plantilla_bytes.decode()
    if plantilla.count("COMMIT;") != 1:
        raise ValueError("Contrato SQL de orígenes inválido")
    cierre = "\n".join("ALTER ROLE " + dato["login"] + " PASSWORD '" + dato["password"] + "';"
                       for dato in credenciales.values())
    copia.sql_bytes(plantilla.replace("COMMIT;", cierre + "\nCOMMIT;").encode(), "datos-roles-previos.log")
    # El SQL viaja por stdin; el archivo de credenciales se crea tras el COMMIT.
    with open(destino, "x", encoding="utf-8", opener=lambda p, flags: os.open(p, flags, 0o600)) as salida:
        json.dump(credenciales, salida)
        salida.write("\n")
