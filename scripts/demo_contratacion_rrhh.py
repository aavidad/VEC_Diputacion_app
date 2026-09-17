#!/usr/bin/env python3
"""Prepara el conjunto sintético de Contratación temporal para una demo RRHH.

No usa SQL ni añade autoridad por cabeceras. La ejecución real es opt-in y
conserva cada recibo devuelto por la API para que un reintento no duplique altas.
"""

from __future__ import annotations

import argparse
import hashlib
import json
import ssl
import sys
import uuid
from dataclasses import dataclass
from datetime import date, timedelta
from pathlib import Path
from typing import Any
from urllib.error import HTTPError, URLError
from urllib.request import Request, build_opener, HTTPSHandler


API = "/api/vec/contratacion-temporal"
RUTAS = {
    "catalogos": f"{API}/catalogos-alta",
    "configuracion_analisis": f"{API}/configuracion-analisis",
    "alta": f"{API}/solicitudes",
    "analisis": f"{API}/analisis/registros",
    "propuesta_cobertura": f"{API}/cobertura/propuesta",
    "decision_cobertura": f"{API}/cobertura/decisiones",
    "asignacion": f"{API}/asignaciones",
    "informe": f"{API}/informes-juridicos/preparaciones",
    "fiscalizacion": f"{API}/fiscalizaciones/resultados",
    "subsanacion": f"{API}/subsanacion-reparos",
    "seleccion": f"{API}/llamamientos/seleccion",
    "detalle": f"{API}/expedientes/consultas",
    "comunicacion": f"{API}/llamamientos/comunicaciones",
}
ESPACIO_CLAVES = uuid.UUID("f7ce9ce2-6057-45bc-a08a-922d7e24fa4a")


@dataclass(frozen=True)
class Caso:
    codigo: str
    punto: str
    detalle: str
    jornada: int
    reparo: bool = False


CASOS = (
    Caso("01", "solicitud", "Sustitución temporal en atención administrativa.", 100),
    Caso("02", "solicitud", "Refuerzo temporal para gestión de expedientes.", 50),
    Caso("03", "solicitud", "Cobertura temporal de servicio público.", 75),
    Caso("04", "analisis", "Análisis previo para cobertura temporal.", 100),
    Caso("05", "analisis", "Análisis de necesidad de carácter temporal.", 50),
    Caso("06", "asignacion", "Cobertura y asignación a unidad competente.", 75),
    Caso("07", "asignacion", "Cobertura de vacante temporal y asignación.", 100),
    Caso("08", "fiscalizacion", "Fiscalización favorable de la cobertura temporal.", 50),
    Caso("09", "fiscalizacion", "Fiscalización con reparo para subsanación.", 75, True),
    Caso("10", "llamamiento", "Llamamiento por bolsa vigente de carácter sintético.", 100),
    Caso("11", "llamamiento", "Llamamiento temporal con jornada parcial.", 50),
    Caso("12", "nombramiento", "Propuesta de nombramiento de desarrollo sintética.", 75),
)
ORDEN = {"solicitud": 0, "analisis": 1, "asignacion": 2, "fiscalizacion": 3, "llamamiento": 4, "nombramiento": 5}


class ErrorAPI(RuntimeError):
    def __init__(self, ruta: str, estado: int, cuerpo: Any):
        super().__init__(f"{ruta}: HTTP {estado}")
        self.ruta, self.estado, self.cuerpo = ruta, estado, cuerpo


def clave(caso: Caso, operacion: str) -> str:
    """UUID v4 determinista: repetición segura y válida para el contrato."""
    digest = bytearray(hashlib.sha256(f"C6:{caso.codigo}:{operacion}".encode()).digest()[:16])
    digest[6] = (digest[6] & 0x0F) | 0x40
    digest[8] = (digest[8] & 0x3F) | 0x80
    return str(uuid.UUID(bytes=bytes(digest)))


def datos(envoltorio: Any) -> Any:
    if not isinstance(envoltorio, dict) or "data" not in envoltorio:
        raise RuntimeError("la API no devolvió el envoltorio data esperado")
    return envoltorio["data"]


class Cliente:
    def __init__(self, url_base: str, cert: str | None, clave_privada: str | None):
        self.url_base = url_base.rstrip("/")
        if self.url_base.startswith("https://"):
            if not cert or not clave_privada:
                raise ValueError("HTTPS exige --cert-rrhh/--clave-rrhh y certificados de Intervención")
            contexto = ssl.create_default_context()
            contexto.load_cert_chain(cert, clave_privada)
            self.opener = build_opener(HTTPSHandler(context=contexto))
        else:
            self.opener = build_opener()

    def pedir(self, metodo: str, ruta: str, cuerpo: dict[str, Any] | None = None) -> Any:
        contenido = None if cuerpo is None else json.dumps(cuerpo, ensure_ascii=False, separators=(",", ":")).encode()
        peticion = Request(self.url_base + ruta, data=contenido, method=metodo)
        peticion.add_header("Accept", "application/json")
        if contenido is not None:
            peticion.add_header("Content-Type", "application/json")
        try:
            with self.opener.open(peticion, timeout=20) as respuesta:
                bruto = respuesta.read()
                valor = json.loads(bruto) if bruto else {}
                if not 200 <= respuesta.status < 300:
                    raise ErrorAPI(ruta, respuesta.status, valor)
                return valor
        except HTTPError as error:
            bruto = error.read()
            try:
                valor = json.loads(bruto) if bruto else {}
            except json.JSONDecodeError:
                valor = {"respuesta": bruto.decode("utf-8", "replace")}
            raise ErrorAPI(ruta, error.code, valor) from error
        except URLError as error:
            raise RuntimeError(f"no se pudo conectar con {self.url_base}: {error.reason}") from error


def primera(lista: Any, campo: str) -> dict[str, Any]:
    if not isinstance(lista, list) or not lista or not isinstance(lista[0], dict):
        raise RuntimeError(f"la configuración no contiene {campo}")
    return lista[0]


def planificar(catalogos: dict[str, Any]) -> list[dict[str, Any]]:
    centros, categorias = catalogos.get("centros"), catalogos.get("categorias")
    if not isinstance(centros, list) or len(centros) < len(CASOS):
        raise RuntimeError("se requieren doce centros del catálogo para la demostración")
    if not isinstance(categorias, list) or len(categorias) < 6:
        raise RuntimeError("se requieren las seis categorías del catálogo para la demostración")
    salida = []
    inicio = date(2026, 10, 1)
    for indice, caso in enumerate(CASOS):
        centro, categoria = centros[indice], categorias[indice % 6]
        contactos = centro.get("contactos") if isinstance(centro, dict) else None
        grupos = categoria.get("grupos_subgrupos") if isinstance(categoria, dict) else None
        contacto, grupo = primera(contactos, "contacto"), primera(grupos, "grupo/subgrupo")
        desde = inicio + timedelta(days=14 * indice)
        salida.append({
            "codigo": caso.codigo, "objetivo": caso.punto, "reparo": caso.reparo,
            "centro": {"referencia": centro["referencia"], "etiqueta": centro.get("etiqueta", "")},
            "categoria": {"referencia": categoria["referencia"], "etiqueta": categoria.get("etiqueta", ""), "grupo_subgrupo": grupo["clave"]},
            "contacto_ref": contacto["referencia"], "jornada_porcentaje": caso.jornada,
            "periodo": {"inicio": f"{desde.isoformat()}T00:00:00Z", "fin": f"{(desde + timedelta(days=90)).isoformat()}T00:00:00Z"},
            "detalle": caso.detalle,
        })
    return salida


def alta(cliente: Cliente, caso: Caso, fila: dict[str, Any]) -> dict[str, Any]:
    return datos(cliente.pedir("POST", RUTAS["alta"], {
        "clave_idempotencia": clave(caso, "alta"),
        "solicitud": {
            "centro_ref": fila["centro"]["referencia"], "contacto_ref": fila["contacto_ref"],
            "categoria_ref": fila["categoria"]["referencia"], "grupo_subgrupo": fila["categoria"]["grupo_subgrupo"],
            "motivo_clave": "sustitucion", "detalle": fila["detalle"], "periodo": fila["periodo"],
            "rc": {"existe": False}, "documentos_adjuntos": [], "observaciones": "Demostración RRHH; datos exclusivamente sintéticos.",
        },
    }))


def analisis(cliente: Cliente, caso: Caso, fila: dict[str, Any], recibo_alta: dict[str, Any], configuracion: dict[str, Any]) -> dict[str, Any]:
    modalidad = primera(configuracion.get("modalidades"), "modalidad")["clave"]
    causa = primera(configuracion.get("causas"), "causa")["clave"]
    rc = primera(configuracion.get("entradas_rc"), "entrada RC")
    return datos(cliente.pedir("POST", RUTAS["analisis"], {
        "expediente_ref": recibo_alta["expediente_ref"], "version_esperada": recibo_alta["version"],
        "clave_idempotencia": clave(caso, "analisis"), "artefacto_ref": configuracion["artefacto_ref"],
        "analisis": {"modalidad_clave": modalidad, "categoria_ref": fila["categoria"]["referencia"],
            "grupo_subgrupo": fila["categoria"]["grupo_subgrupo"], "causa_clave": causa,
            "periodo": fila["periodo"], "porcentaje_jornada": fila["jornada_porcentaje"] * 100,
            "entrada_rc": {"referencia": rc["referencia"], "huella_sha256": rc["huella_sha256"]},
            "observaciones": f"Análisis de demostración RRHH C6-{caso.codigo}; necesidad temporal verificada.",},
    }))


def cobertura_y_asignacion(cliente: Cliente, caso: Caso, analisis_recibo: dict[str, Any]) -> dict[str, Any]:
    expediente, version = analisis_recibo["expediente_ref"], analisis_recibo["version_resultante"]
    propuesta = datos(cliente.pedir("POST", RUTAS["propuesta_cobertura"], {"expediente_ref": expediente, "version_esperada": version}))
    via = propuesta.get("via_recomendada")
    if propuesta.get("estado") != "viable" or not isinstance(via, str):
        raise RuntimeError("la propuesta de cobertura no ofrece una vía viable")
    cobertura = datos(cliente.pedir("POST", RUTAS["decision_cobertura"], {
        "expediente_ref": expediente, "version_esperada": version, "clave_idempotencia": clave(caso, "cobertura"),
        "identidad_semantica": propuesta["identidad_semantica"], "via_elegida": via, "motivo_clave": "",
    }))
    asignacion = datos(cliente.pedir("POST", RUTAS["asignacion"], {
        "expediente_ref": expediente, "version_esperada": cobertura["version_resultante"],
        "clave_idempotencia": clave(caso, "asignacion"), "unidad_ref": "unidad:desarrollo:rrhh",
        "responsable_ref": "persona:responsable-sintetica-001",
    }))
    return {"cobertura": cobertura, "asignacion": asignacion}


def solicitud_comunicacion(caso: Caso, fiscalizacion: dict[str, Any], seleccion: dict[str, Any], detalle: dict[str, Any]) -> dict[str, Any]:
    """Forma el único POST de comunicación a partir de sus antecedentes reales."""
    resumen = detalle.get("resumen") if isinstance(detalle, dict) else None
    if not isinstance(resumen, dict) or resumen.get("expediente_ref") != fiscalizacion.get("expediente_ref") \
            or resumen.get("version") != fiscalizacion.get("version_resultante"):
        raise RuntimeError("el detalle RRHH no acredita la versión fiscalizada seleccionada")
    version_llamamiento = seleccion.get("version_llamamiento")
    if not isinstance(version_llamamiento, int) or version_llamamiento < 1:
        raise RuntimeError("la selección no devolvió versión de llamamiento")
    campos = {
        "clave_idempotencia": clave(caso, "comunicacion"),
        "organizacion_ref": seleccion.get("organizacion_ref"),
        "expediente_ref": fiscalizacion.get("expediente_ref"),
        "llamamiento_ref": seleccion.get("llamamiento_ref"),
        # Es versión del llamamiento, no la del expediente que publica el detalle.
        "version_esperada": version_llamamiento,
        "prueba_entrega_ref": seleccion.get("recibo_ref"),
    }
    if not all(isinstance(valor, str) and valor for nombre, valor in campos.items() if nombre != "version_esperada"):
        raise RuntimeError("la selección no devolvió los antecedentes de comunicación completos")
    return campos


def registrar_comunicacion(cliente: Cliente, caso: Caso, fiscalizacion: dict[str, Any], seleccion: dict[str, Any]) -> dict[str, Any]:
    detalle = datos(cliente.pedir("POST", RUTAS["detalle"], {
        "expediente_ref": fiscalizacion["expediente_ref"],
        "version_observada": fiscalizacion["version_resultante"],
    }))
    solicitud = solicitud_comunicacion(caso, fiscalizacion, seleccion, detalle)
    return datos(cliente.pedir("POST", RUTAS["comunicacion"], solicitud))


def fiscalizar(cliente_rrhh: Cliente, cliente_intervencion: Cliente, caso: Caso, asignacion: dict[str, Any]) -> dict[str, Any]:
    expediente, version = asignacion["expediente_ref"], asignacion["version_resultante"]
    informe = datos(cliente_rrhh.pedir("POST", RUTAS["informe"], {
        "expediente_ref": expediente, "version_esperada": version, "clave_idempotencia": clave(caso, "informe"),
    }))
    resultado = "desfavorable" if caso.reparo else "favorable"
    observaciones = "Reparo sintético para mostrar la subsanación de unidad." if caso.reparo else ""
    fiscalizacion = datos(cliente_intervencion.pedir("POST", RUTAS["fiscalizacion"], {
        "expediente_ref": expediente, "version_esperada": informe["version_resultante"],
        "clave_idempotencia": clave(caso, "fiscalizacion"), "resultado": resultado, "observaciones": observaciones,
    }))
    resultado_final: dict[str, Any] = {"informe": informe, "fiscalizacion": fiscalizacion}
    if caso.reparo:
        resultado_final["subsanacion"] = datos(cliente_rrhh.pedir("POST", RUTAS["subsanacion"], {
            "expediente_ref": expediente, "version_esperada": fiscalizacion["version_resultante"],
            "clave_idempotencia": clave(caso, "subsanacion"),
            "observaciones": "Subsanación sintética registrada para la demostración RRHH.",
        }))
    return resultado_final


def ejecutar(cliente_rrhh: Cliente, cliente_intervencion: Cliente, filas: list[dict[str, Any]], configuracion: dict[str, Any]) -> list[dict[str, Any]]:
    resultado = []
    for caso, fila in zip(CASOS, filas, strict=True):
        registro: dict[str, Any] = {"codigo": caso.codigo, "objetivo": caso.punto, "operaciones": {}}
        try:
            recibido_alta = alta(cliente_rrhh, caso, fila); registro["operaciones"]["alta"] = recibido_alta
            if ORDEN[caso.punto] >= ORDEN["analisis"]:
                recibido_analisis = analisis(cliente_rrhh, caso, fila, recibido_alta, configuracion); registro["operaciones"]["analisis"] = recibido_analisis
            else:
                resultado.append(registro); continue
            if ORDEN[caso.punto] >= ORDEN["asignacion"]:
                cadena = cobertura_y_asignacion(cliente_rrhh, caso, recibido_analisis); registro["operaciones"].update(cadena)
            else:
                resultado.append(registro); continue
            if ORDEN[caso.punto] >= ORDEN["fiscalizacion"]:
                cadena = fiscalizar(cliente_rrhh, cliente_intervencion, caso, cadena["asignacion"]); registro["operaciones"].update(cadena)
            else:
                resultado.append(registro); continue
            if caso.reparo:
                registro["detenido_en"] = "subsanacion_unidad"; resultado.append(registro); continue
            if ORDEN[caso.punto] >= ORDEN["llamamiento"]:
                fiscal = cadena["fiscalizacion"]
                seleccion = datos(cliente_rrhh.pedir("POST", RUTAS["seleccion"], {
                    "expediente_ref": fiscal["expediente_ref"], "version_esperada": fiscal["version_resultante"], "clave_idempotencia": clave(caso, "seleccion"),
                }))
                registro["operaciones"]["seleccion_llamamiento"] = seleccion
                registro["operaciones"]["comunicacion_llamamiento"] = registrar_comunicacion(
                    cliente_rrhh, caso, fiscal, seleccion,
                )
            # La propuesta formal requiere una aceptación vinculada, cuya cadena de
            # comunicación se valida por recibos previos; no se inventa esa evidencia.
            if caso.punto == "nombramiento":
                registro["detenido_en"] = "llamamiento"; registro["limite"] = "pendiente de aceptación vinculada por la API de comunicaciones"
        except (ErrorAPI, KeyError, RuntimeError) as error:
            registro["detenido_en"] = "última operación confirmada"
            registro["error"] = {"tipo": type(error).__name__, "detalle": str(error), "respuesta": getattr(error, "cuerpo", None)}
        resultado.append(registro)
    return resultado


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--url-base", required=True)
    parser.add_argument("--url-base-intervencion", help="Base distinta para la identidad de Intervención (por ejemplo, un proxy propio); por defecto, la misma que --url-base")
    parser.add_argument("--cert-rrhh")
    parser.add_argument("--clave-rrhh")
    parser.add_argument("--cert-intervencion")
    parser.add_argument("--clave-intervencion")
    parser.add_argument("--salida", type=Path, required=True)
    parser.add_argument("--solo-inventario", action="store_true", help="Lee catálogos y genera el plan; nunca escribe.")
    parser.add_argument("--ejecutar", action="store_true", help="Permite POST; requiere revisión previa de dirección.")
    args = parser.parse_args()
    if args.solo_inventario and args.ejecutar:
        parser.error("--solo-inventario y --ejecutar son excluyentes")
    if not args.solo_inventario and not args.ejecutar:
        parser.error("use --solo-inventario o --ejecutar; la escritura nunca es el comportamiento por defecto")
    rrhh = Cliente(args.url_base, args.cert_rrhh, args.clave_rrhh)
    intervencion = Cliente(args.url_base_intervencion or args.url_base, args.cert_intervencion or args.cert_rrhh, args.clave_intervencion or args.clave_rrhh)
    try:
        catalogos = datos(rrhh.pedir("GET", RUTAS["catalogos"]))
        filas = planificar(catalogos)
        salida: dict[str, Any] = {"esquema": "vec.ct.demo-rrhh.c6.v1", "modo": "inventario" if args.solo_inventario else "ejecucion", "expedientes": filas}
        if args.ejecutar:
            configuracion = datos(rrhh.pedir("GET", RUTAS["configuracion_analisis"]))
            salida["resultado"] = ejecutar(rrhh, intervencion, filas, configuracion)
        args.salida.parent.mkdir(parents=True, exist_ok=True)
        args.salida.write_text(json.dumps(salida, ensure_ascii=False, indent=2) + "\n", encoding="utf-8")
    except (ErrorAPI, RuntimeError, ValueError, KeyError) as error:
        print(f"error: {error}", file=sys.stderr)
        return 2
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
