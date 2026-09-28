#!/usr/bin/env python3
"""Ensayo HTTPS aislado. Sólo acepta certificados y referencias sintéticas.

Variables obligatorias: VEC_AUDITORIA_HTTP_URL (https://localhost),
VEC_AUDITORIA_HTTP_CA, VEC_AUDITORIA_HTTP_CERT_{CT,BOLSA},
VEC_AUDITORIA_HTTP_KEY_{CT,BOLSA}, VEC_AUDITORIA_EXP_{CT,BOLSA}.
OPCIONES usa la identidad CT de la composición; CERT/KEY_OPCIONES pueden
declararse para una sesión CT distinta con el mismo perfil nominal.
Opcionales: VEC_AUDITORIA_HTTP_DESDE/HASTA (intervalo máximo de 31 días),
VEC_AUDITORIA_HTTP_RUTA_CONSULTA (ruta registrada por la composición).
Los recibos y referencias no se imprimen. El fichero de comparación se borra
al terminar el guion de PG18.
"""

import datetime as dt
import json
import os
import pathlib
import ssl
import sys
import urllib.error
import urllib.parse
import urllib.request


def requerido(nombre):
    valor = os.environ.get(nombre, "")
    if not valor:
        raise RuntimeError(f"falta {nombre}")
    return valor


def contexto(perfil):
    certificado = os.environ.get(f"VEC_AUDITORIA_HTTP_CERT_{perfil}")
    clave = os.environ.get(f"VEC_AUDITORIA_HTTP_KEY_{perfil}")
    if perfil == "OPCIONES" and not certificado and not clave:
        certificado = requerido("VEC_AUDITORIA_HTTP_CERT_CT")
        clave = requerido("VEC_AUDITORIA_HTTP_KEY_CT")
    else:
        certificado = requerido(f"VEC_AUDITORIA_HTTP_CERT_{perfil}")
        clave = requerido(f"VEC_AUDITORIA_HTTP_KEY_{perfil}")
    ctx = ssl.create_default_context(cafile=requerido("VEC_AUDITORIA_HTTP_CA"))
    ctx.load_cert_chain(certificado, clave)
    return ctx


def pedir(perfil, ruta, cuerpo=None):
    base = requerido("VEC_AUDITORIA_HTTP_URL").rstrip("/")
    url = urllib.parse.urlparse(base)
    if url.scheme != "https" or url.query or url.fragment or url.username or url.password:
        raise RuntimeError("la URL de prueba debe ser HTTPS local sin credenciales")
    if url.hostname not in ("localhost", "127.0.0.1", "::1"):
        raise RuntimeError("el recorrido HTTP sólo admite loopback")
    datos = None if cuerpo is None else json.dumps(cuerpo, separators=(",", ":")).encode()
    req = urllib.request.Request(
        base + ruta,
        data=datos,
        method="GET" if cuerpo is None else "POST",
        headers={} if cuerpo is None else {"Content-Type": "application/json"},
    )
    try:
        with urllib.request.urlopen(req, context=contexto(perfil), timeout=15) as respuesta:
            if respuesta.headers.get("Set-Cookie"):
                raise RuntimeError("la respuesta creó cookie")
            return respuesta.status, json.load(respuesta)
    except urllib.error.HTTPError as error:
        # El cuerpo de un rechazo podría contener datos privados: no se registra.
        error.close()
        return error.code, None


def exigir(condicion, mensaje):
    if not condicion:
        raise RuntimeError(mensaje)


def instante(valor):
    exigir(isinstance(valor, str) and valor.endswith("Z"), "instante sin UTC")
    return dt.datetime.fromisoformat(valor.replace("Z", "+00:00"))


def comprobar_registro(fuente, ref, registro):
    campos = (
        "id", "fuente", "modulo_id", "accion", "actor_ref", "ocurrido_en",
        "resultado", "expediente_ref", "recibo_ref", "antes_sha256",
        "despues_sha256", "motivo", "antes", "despues", "datos_disponibles",
    )
    exigir(isinstance(registro, dict) and all(c in registro for c in campos), "proyección incompleta")
    exigir(registro["fuente"] == fuente and registro["expediente_ref"] == ref, "expediente o fuente ajenos")
    exigir(all(isinstance(registro[c], str) for c in ("id", "accion", "actor_ref", "resultado", "motivo")), "tipo de dato de historia inválido")
    exigir(registro["id"] and registro["accion"] and registro["resultado"], "historia sin identidad")
    instante(registro["ocurrido_en"])
    exigir(registro["antes"] is None or isinstance(registro["antes"], dict), "preimagen inválida")
    exigir(registro["despues"] is None or isinstance(registro["despues"], dict), "postimagen inválida")
    exigir(isinstance(registro["datos_disponibles"], bool), "disponibilidad inválida")


def consultar(fuente, perfil, ref, opciones, actor="", cursor=""):
    desde = os.environ.get("VEC_AUDITORIA_HTTP_DESDE", "2026-09-01T00:00:00Z")
    hasta = os.environ.get("VEC_AUDITORIA_HTTP_HASTA", "2026-10-01T00:00:00Z")
    exigir(dt.timedelta(0) < instante(hasta) - instante(desde) <= dt.timedelta(days=31), "intervalo inválido")
    cuerpo = {
        "fuente": fuente, "expediente_ref": ref, "actor_ref": actor,
        "desde": desde, "hasta": hasta, "limite": 1, "cursor": cursor,
        "finalidad_ref": opciones["finalidad_ref"], "motivo_ref": opciones["motivo_ref"],
    }
    ruta = os.environ.get("VEC_AUDITORIA_HTTP_RUTA_CONSULTA", "/api/vec/auditoria/consultas")
    return pedir(perfil, ruta, cuerpo)


def recorrido():
    estado, opciones = pedir("OPCIONES", "/api/vec/auditoria/opciones")
    exigir(estado == 200 and isinstance(opciones, dict), "GET opciones no disponible")
    exigir(opciones.get("permiso_requerido") == "vec.auditoria.consultar", "permiso de opciones incorrecto")
    exigir(opciones.get("fuentes") == ["ct", "bolsa"], "fuentes de opciones incorrectas")
    exigir(opciones.get("finalidad_ref") and opciones.get("motivo_ref"), "falta finalidad o motivo")
    exigir(opciones.get("es_ejemplo") is True, "el catálogo debe identificarse como ejemplo")
    salida = {}
    for fuente, perfil in (("ct", "CT"), ("bolsa", "BOLSA")):
        ref = requerido("VEC_AUDITORIA_EXP_CT" if fuente == "ct" else "VEC_AUDITORIA_EXP_BOLSA")
        estado, primera = consultar(fuente, perfil, ref, opciones)
        exigir(estado == 200 and isinstance(primera, dict), f"POST {fuente} no disponible")
        registros = primera.get("registros")
        cursor = primera.get("siguiente_cursor")
        exigir(isinstance(registros, list) and len(registros) == 1 and isinstance(cursor, str) and cursor, f"{fuente} sin paginación")
        estado, segunda = consultar(fuente, perfil, ref, opciones, cursor=cursor)
        exigir(estado == 200 and isinstance(segunda, dict) and len(segunda.get("registros", [])) == 1, f"segunda página {fuente} fallida")
        par = registros + segunda["registros"]
        for registro in par:
            comprobar_registro(fuente, ref, registro)
        clave_primera = (instante(par[0]["ocurrido_en"]), par[0]["id"])
        clave_segunda = (instante(par[1]["ocurrido_en"]), par[1]["id"])
        exigir(clave_primera > clave_segunda, f"orden o cursor duplicado en {fuente}")
        exigir(any(r["actor_ref"] for r in par), f"{fuente} no muestra actor")
        actor = next(r["actor_ref"] for r in par if r["actor_ref"])
        estado, filtrada = consultar(fuente, perfil, ref, opciones, actor=actor)
        exigir(estado == 200 and isinstance(filtrada, dict) and all(r["actor_ref"] == actor for r in filtrada.get("registros", [])), f"filtro actor fallido en {fuente}")
        salida[fuente] = par
    # El mismo certificado no hereda el perfil lector de la otra fuente.
    for fuente, perfil, ref in (("ct", "BOLSA", requerido("VEC_AUDITORIA_EXP_CT")),
                                ("bolsa", "CT", requerido("VEC_AUDITORIA_EXP_BOLSA"))):
        estado, _ = consultar(fuente, perfil, ref, opciones)
        exigir(estado == 403, f"perfil ajeno obtuvo {estado} en {fuente}")
    return salida


def main():
    exigir(len(sys.argv) == 3 and sys.argv[1] in ("before", "after"), "uso: recorrido_http.py before|after EVIDENCIA")
    evidencia = pathlib.Path(sys.argv[2])
    exigir(evidencia.parent == pathlib.Path("/dev/shm"), "la evidencia debe ser efímera")
    resultado = recorrido()
    if sys.argv[1] == "before":
        os.umask(0o077)
        with evidencia.open("x", encoding="utf-8") as archivo:
            json.dump(resultado, archivo, sort_keys=True)
        evidencia.chmod(0o600)
    else:
        anterior = json.loads(evidencia.read_text(encoding="utf-8"))
        exigir(resultado == anterior, "la historia/recibos/instantes cambiaron tras reiniciar")
    print(f"OK HTTP {sys.argv[1]}: opciones, CT, Bolsa, dos páginas, actor y 403 cruzado")


if __name__ == "__main__":
    try:
        main()
    except (RuntimeError, OSError, ValueError, KeyError, TypeError) as error:
        print(f"FALLO recorrido HTTP: {error}", file=sys.stderr)
        sys.exit(1)
