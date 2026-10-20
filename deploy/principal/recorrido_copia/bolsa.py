"""Manifiesto B-BACK y selectores de Bolsa para la copia propia."""

import importlib.util
import json
import os
from pathlib import Path
import subprocess


class _Consulta:
    def __init__(self, copia):
        self.copia = copia

    def sql(self, consulta):
        sentencia = "BEGIN READ ONLY; SET LOCAL statement_timeout='5s'; " + consulta + " ROLLBACK;"
        resultado = subprocess.run(
            ["docker", "exec", "-i", "-u", "postgres", self.copia.nombre, "psql", "-X", "-q", "-At",
             "-v", "ON_ERROR_STOP=1", "-U", "postgres", "-d", "postgres"],
            input=sentencia.encode(), stdout=subprocess.PIPE, stderr=subprocess.PIPE,
            check=True, timeout=30)
        return resultado.stdout.decode().strip()


def _historico(repo: Path):
    # Este módulo versionado no inicia procesos al importarse. Se comparte su
    # cálculo histórico, que conserva el perfil y los ámbitos publicados.
    ruta = repo / "scripts/recorridos/clon_bolsa_material.py"
    spec = importlib.util.spec_from_file_location("vec_clon_bolsa_material", ruta)
    if spec is None or spec.loader is None:
        raise ValueError("No está el productor versionado de Bolsa")
    modulo = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(modulo)
    return modulo._historical_manifest


def preparar_bolsa(copia) -> None:
    estado = copia.leer()
    if estado.get("preparada") or not copia.existe():
        raise ValueError("La copia no es nueva o no está activa")
    repo = Path(estado["fuente"])
    destino = copia.material / "identidad/bolsa-bback.json"
    if estado.get("material_modo") == "rrhh_original":
        if not destino.is_file():
            raise ValueError("Falta el manifiesto B-BACK original")
    else:
        manifiesto = _historico(repo)(_Consulta(copia), copia.material)
        with open(destino, "x", encoding="utf-8", opener=lambda p, f: os.open(p, f, 0o600)) as salida:
            json.dump(manifiesto, salida, ensure_ascii=False)
            salida.write("\n")
    ruta = copia.raiz / "entorno_modulos.json"
    entorno = json.loads(ruta.read_text())
    # Los cinco pools de Bolsa ya salen de copia_local_conexiones.preparar;
    # reutilizarlos evita crear LOGIN paralelos para el mismo grupo.
    for clave in ("VEC_BOLSA_BORRADORES_EJECUTOR_CONSULTA_DATABASE_URL",
                  "VEC_BOLSA_BORRADORES_PROYECTOR_GOBIERNO_DATABASE_URL",
                  "VEC_BOLSA_BORRADORES_VERIFICADOR_RECIBO_DATABASE_URL",
                  "VEC_BOLSA_IMPORTACION_CONVOCA_DATABASE_URL",
                  "VEC_BOLSA_B103_FUENTE_AUTORIZACION_DATABASE_URL"):
        if clave not in entorno:
            raise ValueError("Falta una conexión nominal de Bolsa")
    entorno.update(VEC_BOLSA_BORRADORES_ENABLED="true",
                   VEC_PERSONAL_ORGANIZACION_SOURCE_PATH=str(
                       repo / "data/catalogos/estructura-organizativa/v1.rpt-publica.json"),
                   VEC_RPT_CATALOGO_PATH=str(repo / "data/catalogos/rpt/v1.rpt-2026.json"))
    from .datos import _huella, _privado
    _privado(ruta, entorno)
    estado["entorno_modulos_sha256"] = _huella(ruta)
    _privado(copia.estado, estado)
