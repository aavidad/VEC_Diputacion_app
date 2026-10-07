#!/usr/bin/env python3
"""Driver mTLS local: conserva y recupera una preparación S2 original."""

import argparse
import datetime
import hashlib
import http.client
import json
import os
from pathlib import Path
import ssl
import stat
import sys
from urllib.parse import urlsplit


GUARDAR = "/api/vec/seleccion/preparacion-bases/guardar"
CONSULTAR = "/api/vec/seleccion/preparacion-bases/consultar"
LIMITE = 1024 * 1024


def exigir(condicion, codigo):
    if not condicion:
        raise ValueError(codigo)


def objeto(pares):
    resultado = {}
    for clave, valor in pares:
        exigir(clave not in resultado, "clave_duplicada")
        resultado[clave] = valor
    return resultado


def privado(nombre, lectura=True):
    ruta = Path(nombre)
    exigir(ruta.is_absolute(), "ruta_privada_absoluta_requerida")
    for componente in [ruta, *ruta.parents]:
        exigir(not componente.is_symlink(), "enlace_privado_no_admitido")
        exigir(not (componente / ".git").exists(), "material_dentro_git")
    if lectura:
        s = ruta.stat()
        exigir(stat.S_ISREG(s.st_mode) and s.st_uid == os.getuid(), "archivo_privado_invalido")
        exigir(s.st_mode & 0o077 == 0 and s.st_size <= LIMITE, "archivo_privado_expuesto_o_grande")
    else:
        s = ruta.parent.stat()
        exigir(s.st_uid == os.getuid() and s.st_mode & 0o077 == 0, "directorio_privado_requerido")
    return ruta


def leer(nombre):
    with privado(nombre).open("rb") as f:
        datos = f.read(LIMITE + 1)
    exigir(len(datos) <= LIMITE, "archivo_grande")
    return json.loads(datos, object_pairs_hook=objeto)


def canon(datos):
    return json.dumps(datos, ensure_ascii=False, sort_keys=True, separators=(",", ":")).encode()


def huella(datos):
    return hashlib.sha256(canon(datos)).hexdigest()


def guardar(nombre, datos):
    ruta = privado(nombre, False)
    fd = os.open(ruta, os.O_WRONLY | os.O_CREAT | os.O_EXCL | os.O_NOFOLLOW, 0o600)
    with os.fdopen(fd, "wb") as f:
        f.write(canon(datos))
        f.flush()
        os.fsync(f.fileno())


def post(cfg, ruta, entrada, esperado):
    destino = urlsplit(cfg["origen"])
    exigir(destino.scheme == "https" and destino.hostname in {"localhost", "127.0.0.1", "::1"}, "solo_https_loopback")
    exigir(destino.path in {"", "/"} and not destino.query and not destino.fragment and not destino.username and not destino.password, "origen_invalido")
    contexto = ssl.create_default_context(cafile=str(privado(cfg["ca"])))
    contexto.load_cert_chain(str(privado(cfg["certificado"])), str(privado(cfg["clave"])))
    conexion = http.client.HTTPSConnection(destino.hostname, destino.port or 443, context=contexto, timeout=25)
    try:
        conexion.request("POST", ruta, canon(entrada), {"Content-Type": "application/json", "Cache-Control": "no-store"})
        respuesta = conexion.getresponse()
        exigir(respuesta.status == esperado, "http_inesperado_" + str(respuesta.status))
        exigir(respuesta.getheader("Set-Cookie") is None and respuesta.getheader("Cache-Control") == "no-store", "cabeceras_fuera_contrato")
        cuerpo = respuesta.read(LIMITE + 1)
        exigir(len(cuerpo) <= LIMITE, "respuesta_grande")
        return json.loads(cuerpo, object_pairs_hook=objeto)
    finally:
        conexion.close()


def negocio(respuesta):
    exigir(respuesta.get("estado") in {"guardada", "recuperada", "obtenida"}, "estado_fuera_contrato")
    exigir(set(respuesta) == {"estado", "preparacion", "pendientes", "recibo", "acceso"}, "respuesta_fuera_contrato")
    exigir(bool(respuesta["recibo"].get("recibo_ref")), "recibo_ausente")
    return {k: respuesta[k] for k in ("preparacion", "pendientes", "recibo")}


def consultar(cfg, conservado):
    estado = conservado["preparacion"]["estado"]
    for modo in ("actual", "exacta"):
        entrada = {"modo": modo, "preparacion_ref": estado["preparacion_ref"], "revision": 0, "huella_material_sha256": ""}
        if modo == "exacta":
            entrada.update(estado)
        respuesta = post(cfg, CONSULTAR, entrada, 200)
        exigir(negocio(respuesta) == conservado, "consulta_cambio_original_" + modo)
        exigir(bool(respuesta["acceso"].get("auditoria_ref")), "consulta_sin_auditoria")


def instante(valor):
    exigir(isinstance(valor, str) and valor.endswith("Z"), "instante_reinicio_invalido")
    return datetime.datetime.fromisoformat(valor.removesuffix("Z") + "+00:00")


def ejecutar(cfg, fase):
    exigir(cfg.get("esquema") == "vec.seleccion.preparacion-bases.ensayo-http.v1", "configuracion_invalida")
    original = leer(cfg["guardar"])
    exigir(set(original) == {"esperada", "material", "clave_operacion"}, "entrada_original_invalida")
    continuidad = cfg["continuidad"]
    if fase == "guardar":
        exigir(not privado(continuidad, False).exists(), "continuidad_existente_recuperar_original")
        respuesta = post(cfg, GUARDAR, original, 201)
        conservado = negocio(respuesta)
        exigir(respuesta["estado"] == "guardada", "alta_no_confirmada")
        guardar(continuidad, {"solicitud_sha256": huella(original), "original": conservado})
    else:
        previa = leer(continuidad)
        exigir(previa["solicitud_sha256"] == huella(original), "intencion_original_modificada")
        conservado = previa["original"]
        if fase in {"recuperar", "reinicio"}:
            recuperada = post(cfg, GUARDAR, original, 200)
            exigir(recuperada["estado"] == "recuperada" and negocio(recuperada) == conservado, "replay_cambio_original")
        elif fase == "cas":
            colision = leer(cfg["cas"])
            exigir(colision["esperada"] == original["esperada"] and colision["clave_operacion"] != original["clave_operacion"], "cas_no_disputa_preimagen_original")
            rechazada = post(cfg, GUARDAR, colision, 409)
            exigir(rechazada.get("error") == "version_en_conflicto" and "preparacion" not in rechazada, "cas_no_cerrado")
    consultar(cfg, conservado)
    salida = {"fase": fase, "http": "comprobado", "original_sha256": huella(conservado), "historia_sql": "pendiente_owner"}
    if fase == "reinicio":
        prueba = leer(cfg["evidencia_reinicio_owner"])
        exigir(prueba.get("esquema") == "vec.ensayo.reinicio-owner.v1", "evidencia_owner_invalida")
        for componente in ("postgresql", "aplicacion"):
            exigir(instante(prueba[componente]["despues"]) > instante(prueba[componente]["antes"]), "reinicio_owner_no_acreditado")
        salida["reinicio"] = "segun_evidencia_owner"
        salida["reinicio_owner_sha256"] = huella(prueba)
    return salida


def main():
    argumentos = argparse.ArgumentParser()
    argumentos.add_argument("fase", choices=("guardar", "recuperar", "cas", "consultar", "reinicio"))
    argumentos.add_argument("configuracion")
    q = argumentos.parse_args()
    try:
        resultado = ejecutar(leer(q.configuracion), q.fase)
    except (ValueError, OSError, KeyError, TypeError, http.client.HTTPException):
        # Las excepciones de TLS, HTTP o fichero pueden contener rutas privadas.
        print('{"error":"ensayo_no_confirmado"}', file=sys.stderr)
        return 1
    print(json.dumps(resultado, sort_keys=True))
    return 0


if __name__ == "__main__":
    sys.exit(main())
