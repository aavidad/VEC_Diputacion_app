#!/usr/bin/env python3
"""Aprovisionamiento gobernado de empleados sintéticos de Personal (B2).

Da de alta, por la API interna real y con el certificado mTLS de una persona
de RRHH, a personas sintéticas ya acreditadas en ContextoActor. Cada alta es
el mismo acto que el portal: autorización V3 nominal, consumo en la
transacción de Personal 000019, recibo durable y publicación de la proyección
persona→empleado (Personal 000016) que consumen Cronos y Dietas.

No escribe en PostgreSQL, no firma nada por su cuenta y no guarda datos
civiles: el plan privado solo contiene referencias opacas. Es idempotente:
cada alta y cada entrada de catálogo usan una clave que incluye el SHA-256 del
cuerpo enviado, de modo que repetir el mismo plan devuelve el mismo recibo
(200). Un 409 significa entonces que ya existe algo distinto de lo que declara
el plan (otro contenido o un alta por otra vía): se informa como divergencia y
el plan se detiene con código 1, sin crear nada.

Uso (sin secretos en la línea de órdenes):
  VEC_ALTAS_BASE_URL=https://localhost:8443 \
  VEC_ALTAS_CLIENT_CERT=... VEC_ALTAS_CLIENT_KEY=... VEC_ALTAS_CA_CERT=... \
  personal_altas_sinteticas.py PLAN.json [--comprobar]

El plan es un JSON privado (0600, fuera del repositorio):
  {"version": 1, "organismo_ref": "...",
   "procedencia": {"acto_ref", "fuente_ref", "fuente_version", "fuente_huella_sha256"},
   "catalogo": [{"tipo", "ref", "version", "denominacion", "vigente_desde", "vigente_hasta"}],
   "altas": [{"persona_ref", "unidad_ref", "regimen": {"ref", "version"},
              "modalidad": {"ref", "version"}, "vigente_desde", "vigente_hasta"}]}
"""

from __future__ import annotations

import datetime as _dt
import hashlib
import http.client
import json
import os
from pathlib import Path
import re
import ssl
import stat
import sys
from typing import Any, Callable
from urllib.parse import urlsplit

RUTA_ALTA = "/api/vec/personal/empleados"
RUTA_CATALOGO = "/api/vec/personal/catalogos-registro-empleado"
MAXIMO_PLAN = 256 * 1024
MAXIMO_RESPUESTA = 512 * 1024
MAXIMO_ALTAS = 500

REF = re.compile(r"^[a-z][a-z0-9_:-]{2,159}$")
PERSONA = re.compile(r"^per_[A-Za-z0-9_-]{22,128}$")
EMPLEADO = re.compile(r"^emp_[A-Za-z0-9_-]{22,128}$")
HUELLA = re.compile(r"^[0-9a-f]{64}$")
FECHA = re.compile(r"^\d{4}-\d{2}-\d{2}$")
TIPOS = ("regimen", "modalidad", "situacion", "clase_servicio")


class PlanInvalido(ValueError):
    """El plan no cumple el contrato; no se envía nada."""


def _fecha(valor: Any, vacia: bool = False) -> str:
    if vacia and valor == "":
        return ""
    if not isinstance(valor, str) or not FECHA.match(valor):
        raise PlanInvalido("fecha no válida")
    try:
        _dt.date.fromisoformat(valor)
    except ValueError as error:
        raise PlanInvalido("fecha no válida") from error
    return valor


def _entero(valor: Any) -> int:
    if not isinstance(valor, int) or isinstance(valor, bool) or not 1 <= valor <= 2147483647:
        raise PlanInvalido("versión no válida")
    return valor


def _ref(valor: Any, patron: re.Pattern[str] = REF) -> str:
    if not isinstance(valor, str) or not patron.match(valor):
        raise PlanInvalido("referencia no válida")
    return valor


def _denominacion(valor: Any) -> str:
    if (not isinstance(valor, str) or not valor or valor != valor.strip()
            or len(valor.encode("utf-8")) > 256
            or any(ord(c) < 0x20 or ord(c) == 0x7F for c in valor)):
        raise PlanInvalido("denominación no válida")
    return valor


def _claves(objeto: Any, esperadas: set[str]) -> dict[str, Any]:
    if not isinstance(objeto, dict) or set(objeto) != esperadas:
        raise PlanInvalido("claves del plan no válidas")
    return objeto


def validar_plan(plan: Any) -> dict[str, Any]:
    """Comprueba la forma cerrada del plan y devuelve una copia normalizada."""
    plan = _claves(plan, {"version", "organismo_ref", "procedencia", "catalogo", "altas"})
    if plan["version"] != 1:
        raise PlanInvalido("versión de plan no admitida")
    organismo = _ref(plan["organismo_ref"])
    if len(organismo) > 127:
        raise PlanInvalido("organismo demasiado largo")
    proc = _claves(plan["procedencia"], {"acto_ref", "fuente_ref", "fuente_version", "fuente_huella_sha256"})
    procedencia = {
        "acto_ref": _ref(proc["acto_ref"]), "fuente_ref": _ref(proc["fuente_ref"]),
        "fuente_version": _entero(proc["fuente_version"]),
        "fuente_huella_sha256": _ref(proc["fuente_huella_sha256"], HUELLA),
    }
    if not isinstance(plan["catalogo"], list) or not isinstance(plan["altas"], list) \
            or len(plan["catalogo"]) > 200 or not 1 <= len(plan["altas"]) <= MAXIMO_ALTAS:
        raise PlanInvalido("colecciones del plan no válidas")
    catalogo, vistas = [], set()
    for bruta in plan["catalogo"]:
        e = _claves(bruta, {"tipo", "ref", "version", "denominacion", "vigente_desde", "vigente_hasta"})
        if e["tipo"] not in TIPOS:
            raise PlanInvalido("tipo de catálogo no válido")
        entrada = {"tipo": e["tipo"], "ref": _ref(e["ref"]), "version": _entero(e["version"]),
                   "denominacion": _denominacion(e["denominacion"]),
                   "vigente_desde": _fecha(e["vigente_desde"]),
                   "vigente_hasta": _fecha(e["vigente_hasta"], vacia=True)}
        if entrada["vigente_hasta"] and entrada["vigente_hasta"] <= entrada["vigente_desde"]:
            raise PlanInvalido("vigencia de catálogo no válida")
        clave = (entrada["tipo"], entrada["ref"], entrada["version"])
        if clave in vistas:
            raise PlanInvalido("entrada de catálogo repetida")
        vistas.add(clave)
        catalogo.append(entrada)
    altas, personas = [], set()
    for bruta in plan["altas"]:
        a = _claves(bruta, {"persona_ref", "unidad_ref", "regimen", "modalidad", "vigente_desde", "vigente_hasta"})
        alta = {"persona_ref": _ref(a["persona_ref"], PERSONA), "unidad_ref": _ref(a["unidad_ref"]),
                "vigente_desde": _fecha(a["vigente_desde"]), "vigente_hasta": _fecha(a["vigente_hasta"], vacia=True)}
        for nombre in ("regimen", "modalidad"):
            c = _claves(a[nombre], {"ref", "version"})
            alta[nombre] = {"ref": _ref(c["ref"]), "version": _entero(c["version"])}
        if alta["vigente_hasta"] and alta["vigente_hasta"] <= alta["vigente_desde"]:
            raise PlanInvalido("vigencia de alta no válida")
        if alta["persona_ref"] in personas:
            raise PlanInvalido("persona repetida en el plan")
        personas.add(alta["persona_ref"])
        altas.append(alta)
    return {"organismo_ref": organismo, "procedencia": procedencia, "catalogo": catalogo, "altas": altas}


def clave_idempotencia(*partes: str) -> str:
    """UUID v4 de formato, derivado del contenido: la misma alta, la misma clave."""
    digest = bytearray(hashlib.sha256("\n".join(partes).encode("utf-8")).digest()[:16])
    digest[6] = (digest[6] & 0x0F) | 0x40
    digest[8] = (digest[8] & 0x3F) | 0x80
    h = digest.hex()
    return f"{h[:8]}-{h[8:12]}-{h[12:16]}-{h[16:20]}-{h[20:]}"


def huella_cuerpo(cuerpo: dict[str, Any]) -> str:
    """SHA-256 del cuerpo en forma canónica; distingue cualquier cambio del plan."""
    canonico = json.dumps(cuerpo, ensure_ascii=False, sort_keys=True, separators=(",", ":"))
    return hashlib.sha256(canonico.encode("utf-8")).hexdigest()


def huella_catalogo(organismo: str, e: dict[str, Any]) -> str:
    """Misma huella que calcula el portal al publicar una entrada (revisión 1)."""
    partes = ["vec.personal.catalogo-registro-empleado.entrada.v1", organismo, e["tipo"], e["ref"],
              str(e["version"]), "1", e["denominacion"], e["vigente_desde"], e["vigente_hasta"]]
    return hashlib.sha256("\n".join(partes).encode("utf-8")).hexdigest()


def cuerpo_catalogo(organismo: str, e: dict[str, Any]) -> dict[str, Any]:
    return {"operacion": "publicar", "tipo": e["tipo"], "ref": e["ref"], "version": e["version"], "revision": 1,
            "denominacion": e["denominacion"], "huella_sha256": huella_catalogo(organismo, e),
            "vigente_desde": e["vigente_desde"], "vigente_hasta": e["vigente_hasta"]}


def cuerpo_alta(plan: dict[str, Any], alta: dict[str, Any]) -> dict[str, Any]:
    p = plan["procedencia"]
    return {"persona_ref": alta["persona_ref"], "organismo_ref": plan["organismo_ref"], "unidad_ref": alta["unidad_ref"],
            "regimen": alta["regimen"], "modalidad": alta["modalidad"],
            "vigente_desde": alta["vigente_desde"], "vigente_hasta": alta["vigente_hasta"],
            "acto_ref": p["acto_ref"], "fuente_ref": p["fuente_ref"], "fuente_version": p["fuente_version"],
            "fuente_huella_sha256": p["fuente_huella_sha256"]}


Enviar = Callable[[str, dict[str, Any], str], tuple[int, Any]]


def ejecutar(plan: dict[str, Any], enviar: Enviar, informar: Callable[[str], None]) -> int:
    """Publica el catálogo y registra las altas. Devuelve 0 si nada falló.

    201/200 son éxito (nuevo o replay del mismo recibo). Como la clave
    incluye la huella del cuerpo, un 409 es una divergencia entre el plan y
    lo ya registrado: se informa y detiene el plan. También lo detienen la
    caída (estado 0), 401/403 y 5xx: la indisponibilidad nunca es éxito.
    """
    organismo = plan["organismo_ref"]
    for indice, e in enumerate(plan["catalogo"], 1):
        cuerpo = cuerpo_catalogo(organismo, e)
        clave = clave_idempotencia("vec.personal.catalogo-sintetico.v2", organismo, e["tipo"], e["ref"], str(e["version"]),
                                   huella_cuerpo(cuerpo))
        estado, _ = enviar(RUTA_CATALOGO, cuerpo, clave)
        informar(f"catálogo {indice}/{len(plan['catalogo'])} {e['tipo']}:{e['ref']}: HTTP {estado}")
        if estado == 409:
            informar("divergencia: la entrada de catálogo ya existe con contenido distinto al del plan; no se continúa")
            return 1
        if estado not in (200, 201):
            return 1
    fallos = 0
    for indice, alta in enumerate(plan["altas"], 1):
        cuerpo = cuerpo_alta(plan, alta)
        clave = clave_idempotencia("vec.personal.alta-sintetica.v2", organismo, alta["persona_ref"], huella_cuerpo(cuerpo))
        estado, datos = enviar(RUTA_ALTA, cuerpo, clave)
        empleado = ""
        if estado in (200, 201):
            recibo = (datos or {}).get("data", {}).get("recibo", {}) if isinstance(datos, dict) else {}
            empleado = recibo.get("empleado_ref", "") if isinstance(recibo, dict) else ""
            if not EMPLEADO.match(empleado or ""):
                informar(f"alta {indice}/{len(plan['altas'])}: HTTP {estado} sin recibo válido")
                return 1
        informar(f"alta {indice}/{len(plan['altas'])}: HTTP {estado}" + (f" {empleado}" if empleado else ""))
        if estado == 409:
            informar("divergencia: la persona ya tiene empleado con datos distintos a los del plan o dado de alta por otra vía; no se continúa")
            return 1
        if estado not in (200, 201):
            fallos += 1
            if estado == 0 or estado >= 500 or estado in (401, 403):
                return 1
    return 1 if fallos else 0


def _leer_privado(ruta: str, repo: Path) -> bytes:
    p = Path(ruta)
    if not p.is_absolute() or p.is_symlink() or not p.is_file():
        raise PlanInvalido("el plan debe ser un archivo regular absoluto")
    real = p.resolve()
    if real == repo or repo in real.parents:
        raise PlanInvalido("el plan privado no puede estar dentro del repositorio")
    info = real.stat()
    if info.st_uid != os.getuid() or stat.S_IMODE(info.st_mode) != 0o600 or info.st_size > MAXIMO_PLAN:
        raise PlanInvalido("el plan requiere propietario propio, modo 0600 y tamaño acotado")
    return real.read_bytes()


def _sin_duplicados(pares: list[tuple[str, Any]]) -> dict[str, Any]:
    resultado: dict[str, Any] = {}
    for clave, valor in pares:
        if clave in resultado:
            raise PlanInvalido("clave JSON repetida")
        resultado[clave] = valor
    return resultado


def crear_enviar(base: str, certificado: str, clave: str, ca: str) -> Enviar:
    partes = urlsplit(base)
    if partes.scheme != "https" or not partes.hostname or partes.path not in ("", "/") or partes.query or partes.username:
        raise PlanInvalido("VEC_ALTAS_BASE_URL debe ser https://host[:puerto]")
    contexto = ssl.create_default_context(cafile=ca)
    contexto.minimum_version = ssl.TLSVersion.TLSv1_2
    contexto.load_cert_chain(certificado, clave)

    def enviar(ruta: str, cuerpo: dict[str, Any], idempotencia: str) -> tuple[int, Any]:
        datos = json.dumps(cuerpo, ensure_ascii=False, separators=(",", ":")).encode("utf-8")
        conexion = http.client.HTTPSConnection(partes.hostname, partes.port or 443, context=contexto, timeout=20)
        try:
            conexion.request("POST", ruta, body=datos, headers={
                "Content-Type": "application/json", "Accept": "application/json", "Idempotency-Key": idempotencia})
            respuesta = conexion.getresponse()
            bruto = respuesta.read(MAXIMO_RESPUESTA + 1)
            if len(bruto) > MAXIMO_RESPUESTA:
                return 0, None
            try:
                return respuesta.status, json.loads(bruto.decode("utf-8")) if bruto else None
            except (UnicodeDecodeError, json.JSONDecodeError):
                return respuesta.status, None
        except (OSError, http.client.HTTPException):
            return 0, None
        finally:
            conexion.close()

    return enviar


def main(argv: list[str]) -> int:
    if len(argv) not in (2, 3) or (len(argv) == 3 and argv[2] != "--comprobar"):
        print("uso: personal_altas_sinteticas.py PLAN.json [--comprobar]", file=sys.stderr)
        return 2
    repo = Path(__file__).resolve().parents[2]
    try:
        plan = validar_plan(json.loads(_leer_privado(argv[1], repo).decode("utf-8"), object_pairs_hook=_sin_duplicados))
    except (PlanInvalido, UnicodeDecodeError, json.JSONDecodeError) as error:
        print(f"plan rechazado: {error}", file=sys.stderr)
        return 2
    if len(argv) == 3:
        print(f"plan válido: {len(plan['catalogo'])} entradas de catálogo y {len(plan['altas'])} altas; no se ha enviado nada")
        return 0
    try:
        enviar = crear_enviar(os.environ["VEC_ALTAS_BASE_URL"], os.environ["VEC_ALTAS_CLIENT_CERT"],
                              os.environ["VEC_ALTAS_CLIENT_KEY"], os.environ["VEC_ALTAS_CA_CERT"])
    except KeyError as error:
        print(f"falta la variable {error.args[0]}", file=sys.stderr)
        return 2
    except (PlanInvalido, OSError, ssl.SSLError) as error:
        print(f"conexión no preparada: {error}", file=sys.stderr)
        return 2
    return ejecutar(plan, enviar, print)


if __name__ == "__main__":
    sys.exit(main(sys.argv))
