#!/usr/bin/env python3
"""Prueba focal de vigencia para el material ficticio de desarrollo."""

import datetime as dt
import hashlib
import os
from pathlib import Path
import subprocess


SCRIPT = Path(__file__).with_name("generar_credenciales_desarrollo.sh")
BASE = Path("/dev/shm/vec/codexl-pruebas-credenciales")


def ejecutar(destino: Path, valor: str | None) -> subprocess.CompletedProcess[str]:
    entorno = os.environ.copy()
    entorno.pop("VEC_DESARROLLO_VIGENCIA_DIAS", None)
    if valor is not None:
        entorno["VEC_DESARROLLO_VIGENCIA_DIAS"] = valor
    return subprocess.run(
        [str(SCRIPT), str(destino)], env=entorno, text=True,
        stdout=subprocess.PIPE, stderr=subprocess.PIPE, check=False,
    )


def huellas(destino: Path) -> dict[str, str]:
    return {
        str(path.relative_to(destino)): hashlib.sha256(path.read_bytes()).hexdigest()
        for path in destino.rglob("*") if path.is_file()
    }


def vigencia(certificado: Path, dias: int) -> None:
    salida = subprocess.check_output(
        ["openssl", "x509", "-in", str(certificado), "-noout", "-dates"],
        text=True,
    )
    fechas = {}
    for linea in salida.splitlines():
        nombre, valor = linea.split("=", 1)
        fechas[nombre] = dt.datetime.strptime(valor, "%b %d %H:%M:%S %Y %Z")
    diferencia = fechas["notAfter"] - fechas["notBefore"]
    assert diferencia.total_seconds() == dias * 86400, certificado


def main() -> None:
    BASE.mkdir(mode=0o700, parents=True, exist_ok=True)
    destino = BASE / "material"
    assert not destino.exists(), "el destino de esta prueba debe estar vacío"
    for indice, valor in enumerate(("59", "91", "$(touch /dev/shm/vec/codexl-inyeccion)", "999999999999999999999")):
        rechazo = BASE / f"rechazo-{indice}" / "material"
        resultado = ejecutar(rechazo, valor)
        assert resultado.returncode != 0, valor
        assert not rechazo.exists() and not rechazo.parent.exists(), valor
    assert not Path("/dev/shm/vec/codexl-inyeccion").exists()

    generado = ejecutar(destino, None)
    assert generado.returncode == 0, generado.stderr
    for relativo in ("ca/ca.crt", "tls/servidor.crt", "mtls/cliente.crt", "mtls/intervencion.crt"):
        vigencia(destino / relativo, 90)
    antes = huellas(destino)
    repetido = ejecutar(destino, "60")
    assert repetido.returncode == 0, repetido.stderr
    assert huellas(destino) == antes, "la segunda ejecución modificó material existente"
    minimo = BASE / "material-60"
    assert not minimo.exists()
    resultado_minimo = ejecutar(minimo, "60")
    assert resultado_minimo.returncode == 0, resultado_minimo.stderr
    for relativo in ("ca/ca.crt", "tls/servidor.crt", "mtls/cliente.crt", "mtls/intervencion.crt"):
        vigencia(minimo / relativo, 60)
    print("vigencia de credenciales de desarrollo: OK")


if __name__ == "__main__":
    main()
