"""Cotejo de los recibos 01 y 02 conservados en la copia PostgreSQL de HZ39."""

from __future__ import annotations

import json
import re
import subprocess


CONTENEDOR = re.compile(r"vec-codexp-hz39-pg-[0-9]{8}\Z")
REFERENCIA = re.compile(r"[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}\Z")
ESTADOS = {"01": (1, "pendiente_ratificacion"), "02": (2, "ratificada")}
CONSULTA = r"""
SELECT jsonb_build_object(
    'peticion_ref', r.peticion_ref,
    'version', r.version,
    'estado', r.estado,
    'recibo_ref', r.recibo_ref,
    'registrado_en', r.recibo_json->>'registrado_en',
    'integridad', r.recibo_json->>'peticion_ref'=r.peticion_ref
        AND r.recibo_json->>'recibo_ref'=r.recibo_ref
        AND (r.recibo_json->>'version')::smallint=r.version
        AND (r.recibo_json->>'registrado_en')::timestamptz=r.registrada_en
)::text
FROM vec_contratacion_temporal.peticion_centro_revision AS r
WHERE r.recibo_ref IN (:'recibo_01', :'recibo_02')
ORDER BY r.version;
"""


class ReciboNoConfirmado(ValueError):
    """La copia no acredita los dos recibos originales exactos."""


def _validar(originales: dict, contenedor: str, puerto: int) -> None:
    if not isinstance(contenedor, str) or not CONTENEDOR.fullmatch(contenedor):
        raise ReciboNoConfirmado("contenedor_fuera_de_la_copia_hz39")
    if type(puerto) is not int or not 1024 <= puerto <= 65535:
        raise ReciboNoConfirmado("puerto_postgresql_invalido")
    if not isinstance(originales, dict) or set(originales) != set(ESTADOS):
        raise ReciboNoConfirmado("recibos_originales_incompletos")
    for paso, (version, _) in ESTADOS.items():
        recibo = originales[paso]
        if not isinstance(recibo, dict) or set(recibo) != {
            "peticion_ref", "recibo_ref", "version", "registrado_en"
        }:
            raise ReciboNoConfirmado("recibo_original_invalido")
        if (
            not all(isinstance(recibo[k], str) and REFERENCIA.fullmatch(recibo[k])
                    for k in ("peticion_ref", "recibo_ref"))
            or type(recibo["version"]) is not int or recibo["version"] != version
            or not isinstance(recibo["registrado_en"], str)
            or not re.fullmatch(r"[0-9T:.+Z-]{20,40}", recibo["registrado_en"])
        ):
            raise ReciboNoConfirmado("recibo_original_invalido")
    if (originales["01"]["peticion_ref"] != originales["02"]["peticion_ref"]
            or originales["01"]["recibo_ref"] == originales["02"]["recibo_ref"]):
        raise ReciboNoConfirmado("recibos_originales_no_relacionados")


def cotejar_recibos(originales: dict, *, contenedor: str, puerto: int = 55639) -> dict:
    """Exige los recibos originales 01/02 y devuelve las dos filas acreditadas.

    Cada entrada lleva peticion_ref, recibo_ref, version y registrado_en tal
    como llegaron en la respuesta HTTP original. No consulta ni repite actos.
    """
    _validar(originales, contenedor, puerto)
    orden = ["docker", "exec", "-i", "--user", "postgres",
             "-e", "PGPASSWORD=", "-e",
             "PGOPTIONS=-c default_transaction_read_only=on -c statement_timeout=5000",
             contenedor, "psql", "-h", "/var/run/postgresql", "-p", str(puerto),
             "-U", "postgres", "-d", "postgres", "-X", "-q", "-A", "-t",
             "-v", "ON_ERROR_STOP=1",
             "-v", "recibo_01=" + originales["01"]["recibo_ref"],
             "-v", "recibo_02=" + originales["02"]["recibo_ref"], "-f", "-"]
    try:
        proceso = subprocess.run(
            orden, input=CONSULTA.encode(), stdout=subprocess.PIPE,
            stderr=subprocess.DEVNULL, timeout=10, check=False,
            env={"PATH": "/usr/bin:/bin", "DOCKER_HOST": "unix:///var/run/docker.sock",
                 "DOCKER_CONTEXT": "default"},
        )
        if proceso.returncode or len(proceso.stdout) > 4096:
            raise ReciboNoConfirmado("consulta_sql_no_confirmada")
        filas = [json.loads(linea) for linea in proceso.stdout.splitlines()]
    except (OSError, subprocess.TimeoutExpired, UnicodeError, json.JSONDecodeError) as exc:
        raise ReciboNoConfirmado("consulta_sql_no_confirmada") from exc
    if len(filas) != 2 or any(not isinstance(fila, dict) for fila in filas):
        raise ReciboNoConfirmado("recibos_sql_ausentes_o_duplicados")
    cotejados = {}
    for paso, fila in zip(ESTADOS, filas):
        esperado = originales[paso]
        version, estado = ESTADOS[paso]
        if (fila.get("integridad") is not True or fila.get("version") != version
                or fila.get("estado") != estado
                or any(fila.get(k) != esperado[k] for k in esperado)):
            raise ReciboNoConfirmado("recibo_sql_distinto_del_original")
        cotejados[paso] = {k: fila[k] for k in
                           ("peticion_ref", "recibo_ref", "version", "estado", "registrado_en")}
    return cotejados


def _peticion_unica(respuesta: dict, referencia: str) -> dict:
    if not isinstance(respuesta, dict) or not isinstance(respuesta.get("data"), dict):
        raise ReciboNoConfirmado("bandeja_http_invalida")
    filas = respuesta["data"].get("peticiones")
    if not isinstance(filas, list):
        raise ReciboNoConfirmado("bandeja_http_invalida")
    coincidencias = []
    for fila in filas:
        if not isinstance(fila, dict):
            raise ReciboNoConfirmado("bandeja_http_invalida")
        peticion = fila.get("peticion", fila)
        if not isinstance(peticion, dict):
            raise ReciboNoConfirmado("bandeja_http_invalida")
        if peticion.get("referencia") == referencia:
            coincidencias.append(peticion)
    if len(coincidencias) != 1:
        raise ReciboNoConfirmado("peticion_ausente_o_duplicada_en_bandeja")
    peticion = coincidencias[0]
    if (type(peticion.get("version")) is not int or peticion["version"] < 1
            or not isinstance(peticion.get("estado"), str)
            or not re.fullmatch(r"[a-z][a-z0-9_]{0,63}", peticion["estado"])):
        raise ReciboNoConfirmado("estado_de_bandeja_invalido")
    return {"referencia": referencia, "version": peticion["version"], "estado": peticion["estado"]}


def preparar_recuperacion(previos: dict, valores: dict, checkpoint: dict, *,
                          contenedor: str, puerto: int = 55639) -> dict:
    """Prepara 01/02 con recibos SQL y la bandeja HTTP anterior al reinicio."""
    referencia = valores.get("peticion_ref") if isinstance(valores, dict) else None
    if not isinstance(referencia, str) or not REFERENCIA.fullmatch(referencia):
        raise ReciboNoConfirmado("peticion_ref_original_ausente")
    pasos = previos.get("pasos") if isinstance(previos, dict) else None
    if not isinstance(pasos, list):
        raise ReciboNoConfirmado("pasos_originales_invalidos")
    originales = {}
    for paso in ESTADOS:
        registros = [registro for registro in pasos
                     if isinstance(registro, dict) and registro.get("id") == paso]
        if len(registros) != 1:
            raise ReciboNoConfirmado("paso_original_ausente_o_duplicado")
        recibos = registros[0].get("recibos_efecto")
        if not isinstance(recibos, list) or len(recibos) != 1 or not isinstance(recibos[0], dict):
            raise ReciboNoConfirmado("recibo_efecto_ausente_o_duplicado")
        original = recibos[0]
        if original.get("peticion_ref", referencia) != referencia:
            raise ReciboNoConfirmado("peticion_ref_original_divergente")
        originales[paso] = {"peticion_ref": referencia,
                           **{k: original.get(k) for k in ("recibo_ref", "version", "registrado_en")}}
    peticion = _peticion_unica(checkpoint, referencia)
    return {"recibos": cotejar_recibos(originales, contenedor=contenedor, puerto=puerto),
            "peticion": peticion}


def cotejar_bandeja(actual: dict, preparada: dict) -> dict:
    """Exige la misma referencia, versión y estado vistos antes del reinicio."""
    anterior = preparada.get("peticion") if isinstance(preparada, dict) else None
    if not isinstance(anterior, dict):
        raise ReciboNoConfirmado("preparacion_de_bandeja_invalida")
    referencia = anterior.get("referencia")
    if not isinstance(referencia, str) or not REFERENCIA.fullmatch(referencia):
        raise ReciboNoConfirmado("preparacion_de_bandeja_invalida")
    observada = _peticion_unica(actual, referencia)
    if observada != anterior:
        raise ReciboNoConfirmado("bandeja_distinta_tras_reinicio")
    return observada
