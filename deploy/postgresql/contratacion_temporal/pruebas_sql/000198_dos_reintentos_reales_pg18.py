#!/usr/bin/env python3
"""Comprueba dos reintentos HTTPS sobre una intención CT/Bolsa pendiente real.

Requiere un clon PostgreSQL 18 aislado, app con mTLS y una petición sintética
canónica. La preparación del 503 parcial y la restitución de su permiso se
hacen antes de este ensayo; aquí no se alteran concesiones ni perfiles.
"""

import concurrent.futures
import datetime
import json
import http.client
import os
import pathlib
import re
import ssl
import subprocess
import threading
import urllib.parse
import uuid


def requerido(nombre):
    valor = os.environ.get(nombre, "")
    if not valor:
        raise SystemExit(f"falta {nombre}")
    return valor


contenedor = requerido("VEC_CT198_TEST_PG_CONTAINER")
if not re.fullmatch(r"[A-Za-z0-9_-]+", contenedor):
    raise SystemExit("nombre de contenedor inválido")
url = requerido("VEC_CT198_TEST_URL")
ruta = urllib.parse.urlsplit(url)
if ruta.scheme != "https" or ruta.hostname != "127.0.0.1" or ruta.port is None \
        or ruta.username is not None or ruta.password is not None or ruta.query or ruta.fragment \
        or ruta.path != "/api/vec/contratacion-temporal/llamamientos/seleccion":
    raise SystemExit("la prueba exige HTTPS local y la ruta CT exacta")
peticion = pathlib.Path(requerido("VEC_CT198_TEST_REQUEST"))
cuerpo = peticion.read_bytes()
comando = json.loads(cuerpo)
clave = str(uuid.UUID(comando["clave_idempotencia"]))
if cuerpo != json.dumps(comando, separators=(",", ":"), ensure_ascii=False).encode():
    raise SystemExit("petición no canónica")
contexto_tls = ssl.create_default_context(cafile=requerido("VEC_CT198_TEST_CA"))
contexto_tls.load_cert_chain(requerido("VEC_CT198_TEST_CERT"), requerido("VEC_CT198_TEST_KEY"))


def estado_pg():
    sql = """SELECT jsonb_build_object(
      'ct',(SELECT jsonb_build_object('situacion',situacion,'efecto',efecto,
          'fencing',fencing_version,'lease_vencida',lease_hasta<=clock_timestamp(),
          'ventana_orden',ventana_orden_abierta,
          'ventana_llamamiento',ventana_llamamiento_abierta,
          'recibo',recibo_json IS NOT NULL)
        FROM vec_contratacion_temporal.ejecucion_seleccion_llamamiento_o6
        WHERE clave_idempotencia='%s'::uuid),
      'historia',(SELECT count(*) FROM vec_contratacion_temporal.historia_reanudacion_seleccion_llamamiento
        WHERE clave_idempotencia='%s'::uuid),
      'bolsa',(SELECT count(*) FROM vec_bolsa_llamamientos.integracion_desarrollo),
      'auditoria',(SELECT count(*) FROM vec_bolsa_llamamientos.auditoria_integracion_desarrollo),
      'outbox',(SELECT count(*) FROM vec_bolsa_llamamientos.outbox_integracion_desarrollo),
      'propuesta',(SELECT jsonb_build_object('recibo',recibo_ref,'fecha',confirmada_en)
        FROM vec_bolsa_llamamientos.integracion_desarrollo
        WHERE tipo='propuesta' ORDER BY confirmada_en DESC LIMIT 1))""" % (clave, clave)
    resultado = subprocess.run(
        ["docker", "exec", contenedor, "psql", "-XAt", "-v", "ON_ERROR_STOP=1",
         "-U", "postgres", "-d", "postgres", "-c", sql],
        capture_output=True, text=True, check=False,
    )
    if resultado.returncode:
        raise SystemExit("consulta PostgreSQL fallida")
    return json.loads(resultado.stdout)


previo = estado_pg()
ct = previo["ct"]
if not ct or ct["situacion"] != "indeterminada" or ct["efecto"] != "solicitar_llamamiento" \
        or not ct["lease_vencida"] or not ct["ventana_orden"] or not ct["ventana_llamamiento"] \
        or ct["recibo"] or not previo["propuesta"]:
    raise SystemExit("precondición: falta intención pendiente con propuesta Bolsa")

barrera = threading.Barrier(2)


def post(simultaneo=False):
    if simultaneo:
        barrera.wait(timeout=5)
    conexion = http.client.HTTPSConnection(ruta.hostname, ruta.port, context=contexto_tls, timeout=35)
    try:
        conexion.request("POST", ruta.path, body=cuerpo,
                         headers={"Content-Type": "application/json", "Accept": "application/json"})
        respuesta = conexion.getresponse()
        return respuesta.status, json.loads(respuesta.read())
    finally:
        conexion.close()


with concurrent.futures.ThreadPoolExecutor(max_workers=2) as pool:
    futuros = [pool.submit(post, True), pool.submit(post, True)]
    respuestas = [f.result(timeout=40) for f in futuros]
ganadores = [x[1]["data"] for x in respuestas if x[0] == 200 and x[1].get("data", {}).get("estado") == "confirmado"]
if len(ganadores) not in (1, 2) or any(codigo not in (200, 503) for codigo, _ in respuestas):
    observados = [(codigo, contenido.get("error", {}).get("codigo")) for codigo, contenido in respuestas]
    raise SystemExit(f"clave=resultados_concurrentes actual={observados} esperado=al_menos_un_200_y_solo_200_o_503")
if any(g != ganadores[0] for g in ganadores[1:]):
    raise SystemExit("recibos concurrentes distintos")
codigo_terminal, respuesta_terminal = post()
if codigo_terminal != 200 or respuesta_terminal.get("data") != ganadores[0]:
    raise SystemExit("replay terminal no conservó el recibo")

fecha = lambda valor: datetime.datetime.fromisoformat(valor.replace("Z", "+00:00"))
propuesta = previo["propuesta"]
if ganadores[0]["recibo_ref"] != propuesta["recibo"] or \
        fecha(ganadores[0]["confirmada_en"]) != fecha(propuesta["fecha"]):
    raise SystemExit("recibo o fecha Bolsa original sustituidos")
posterior = estado_pg()
if posterior["ct"]["situacion"] != "confirmada" or posterior["ct"]["fencing"] != ct["fencing"] + 1 \
        or posterior["historia"] != previo["historia"] + 1 \
        or any(posterior[k] != previo[k] for k in ("bolsa", "auditoria", "outbox")):
    raise SystemExit("duplicado de efecto o recuperación sin historia exacta")
print("CT198-CONCURRENCIA-OK", sorted(codigo for codigo, _ in respuestas), "terminal=200",
      "fencing", ct["fencing"], posterior["ct"]["fencing"], "bolsa", posterior["bolsa"])
