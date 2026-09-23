#!/usr/bin/env python3
"""Ensayo real de comisión calculada con identidad mTLS sintética en cidonia.

Repetir con la misma VEC_DIETAS_TEST_KEY tras reiniciar aplicación y PG.
"""

import json
import os
import re
import ssl
import sys
from urllib.error import HTTPError, URLError
from urllib.parse import quote
from urllib.request import Request, urlopen


def requerido(nombre):
    valor = os.environ.get(nombre, "")
    if not valor:
        raise ValueError(f"Falta {nombre}")
    return valor


def peticion(contexto, base, metodo, ruta, cuerpo=None):
    datos = None if cuerpo is None else json.dumps(cuerpo, ensure_ascii=False).encode("utf-8")
    cabeceras = {"Accept": "application/json"}
    if datos is not None:
        cabeceras["Content-Type"] = "application/json; charset=utf-8"
    try:
        with urlopen(Request(base + ruta, data=datos, headers=cabeceras, method=metodo),
                     context=contexto, timeout=30) as respuesta:
            bruto = respuesta.read(131073)
            if len(bruto) > 131072:
                raise ValueError("Respuesta excesiva")
            return respuesta.status, json.loads(bruto)
    except HTTPError as error:
        raise ValueError(f"{metodo} {ruta}: HTTP {error.code}") from error
    except URLError as error:
        raise ValueError(f"{metodo} {ruta}: transporte no disponible") from error


def comprobar(estado, item):
    if estado not in (200, 201) or not isinstance(item, dict):
        raise ValueError(f"Alta incompatible: HTTP {estado}")
    comision, recibo = item.get("comision"), item.get("recibo")
    if not isinstance(comision, dict) or not isinstance(recibo, dict):
        raise ValueError("Faltan comisión o recibo")
    calculo = comision.get("calculo", {})
    ruta = calculo.get("tramos_ruta", [])
    grupos = calculo.get("opciones_dieta", [])
    if (comision.get("codigos_ruta") != ["18087", "18175"]
            or not re.fullmatch(r"dco_[A-Za-z0-9_-]{22,128}", comision.get("referencia", ""))
            or not re.fullmatch(r"rcd_[A-Za-z0-9_-]{22,128}", recibo.get("referencia", ""))
            or calculo.get("procedencia") != "osrm_interno"
            or not calculo.get("version_grafo")
            or not calculo.get("version_tarifa", "").startswith("provisional:")
            or not isinstance(ruta, list) or len(ruta) != 1
            or [ruta[0].get("origen_codigo"), ruta[0].get("destino_codigo")] != ["18087", "18175"]
            or not isinstance(grupos, list) or [g.get("grupo") for g in grupos] != [1, 2, 3]
            or float(calculo.get("kilometros", 0)) <= 0
            or int(calculo.get("importe_kilometraje_centimos", -1)) < 0):
        raise ValueError("Cálculo OSRM, ruta o tramos incompatibles")
    return comision, recibo


def main():
    base = requerido("VEC_VERIFY_BASE_URL").rstrip("/")
    if not base.startswith("https://"):
        raise ValueError("VEC_VERIFY_BASE_URL debe usar HTTPS")
    clave = requerido("VEC_DIETAS_TEST_KEY")
    if not re.fullmatch(r"[A-Za-z0-9_-]{16,128}", clave):
        raise ValueError("VEC_DIETAS_TEST_KEY no válida")
    contexto = ssl.create_default_context(cafile=requerido("VEC_VERIFY_CA_CERT"))
    contexto.load_cert_chain(requerido("VEC_VERIFY_CLIENT_CERT"), requerido("VEC_VERIFY_CLIENT_KEY"))
    cuerpo = {
        "clave_idempotencia": clave,
        "fecha_inicio": os.environ.get("VEC_DIETAS_TEST_FECHA", "2026-09-24"),
        "fecha_fin": os.environ.get("VEC_DIETAS_TEST_FECHA", "2026-09-24"),
        "hora_inicio": "08:00", "hora_fin": "18:00",
        "motivo": "Ensayo sintético de comisión calculada",
        "codigos_ruta": ["18087", "18175"],
    }
    relacion = os.environ.get("VEC_DIETAS_TEST_RELACION", "")
    if relacion:
        cuerpo["relacion_ref"] = relacion
    ruta = "/api/vec/dietas/comisiones"
    estado, alta = peticion(contexto, base, "POST", ruta, cuerpo)
    comision, recibo = comprobar(estado, alta)
    consulta = ruta + "/" + quote(comision["referencia"])
    if relacion:
        consulta += "?relacion_ref=" + quote(relacion)
    estado_detalle, detalle = peticion(contexto, base, "GET", consulta)
    comision_detalle, recibo_detalle = comprobar(estado_detalle, detalle)
    estado_replay, replay = peticion(contexto, base, "POST", ruta, cuerpo)
    comision_replay, recibo_replay = comprobar(estado_replay, replay)
    for otro, otro_recibo in ((comision_detalle, recibo_detalle), (comision_replay, recibo_replay)):
        if otro != comision or {k: v for k, v in otro_recibo.items() if k != "repeticion"} != {k: v for k, v in recibo.items() if k != "repeticion"}:
            raise ValueError("Detalle o replay no conserva cálculo y recibo")
    if estado_detalle != 200 or estado_replay != 200 or not recibo_replay.get("repeticion"):
        raise ValueError("Replay no confirmado")
    print(f"Dietas R1: POST {estado}, GET 200, replay 200; {comision['referencia']}; "
          f"{comision['calculo']['kilometros']} km; recibo {recibo['referencia']}")


if __name__ == "__main__":
    try:
        main()
    except (ValueError, OSError, TypeError) as error:
        print(f"Dietas R1: {error}", file=sys.stderr)
        sys.exit(1)
