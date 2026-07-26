#!/usr/bin/env python3
"""Captura y revisa las 17 pantallas de contratación solicitadas por RRHH.

La herramienta trabaja únicamente contra una presentación local, usa capturas
del área visible y no conserva cookies ni almacenamiento del navegador.
"""

from __future__ import annotations

import argparse
import json
import re
import sys
from dataclasses import asdict, dataclass
from datetime import datetime, timezone
from pathlib import Path
from typing import Any, Sequence
from urllib.parse import urlparse


RAIZ_REPOSITORIO = Path(__file__).resolve().parents[1]
SALIDA_PREDETERMINADA = RAIZ_REPOSITORIO / "var" / "revision-web-contratacion-rrhh"
RUTA_MODULO = (
    "/portal-empleado/?presentacion=rrhh&perfil=administrador"
    "#contratacion-temporal"
)


@dataclass(frozen=True, slots=True)
class TamanoCaptura:
    clave: str
    ancho: int
    alto: int


@dataclass(frozen=True, slots=True)
class PantallaRRHH:
    numero: int
    clave: str
    nombre: str
    tipo: str
    referencia_tarea: str = ""
    titulo_esperado: str = ""


TAMANOS: tuple[TamanoCaptura, ...] = (
    TamanoCaptura("referencia-1536", 1536, 1024),
    TamanoCaptura("escritorio-1440", 1440, 1000),
    TamanoCaptura("escritorio-1280", 1280, 900),
)

PANTALLAS: tuple[PantallaRRHH, ...] = (
    PantallaRRHH(1, "cuadro-mando", "Inicio y cuadro de mando", "cuadro"),
    PantallaRRHH(2, "nueva-peticion", "Nueva petición de personal", "alta"),
    PantallaRRHH(3, "analisis-rrhh", "Análisis de RRHH", "tarea", "tarea-analisis", "Análisis de RRHH"),
    PantallaRRHH(4, "gestion-bolsa", "Gestión de bolsa", "tarea", "tarea-cobertura", "Comprobaciones y vía de cobertura"),
    PantallaRRHH(5, "bandeja-unidad", "Bandeja de la unidad", "tarea", "tarea-asignacion", "Asignación a unidad"),
    PantallaRRHH(6, "informe-juridico", "Informe jurídico automático", "tarea", "tarea-informe-juridico", "Informe jurídico"),
    PantallaRRHH(7, "firma-envio", "Firma y envío a Intervención", "tarea", "tarea-envio-intervencion", "Firma y envío a Intervención"),
    PantallaRRHH(8, "fiscalizacion", "Fiscalización por Intervención", "tarea", "tarea-fiscalizacion", "Fiscalización"),
    PantallaRRHH(9, "subsanacion", "Subsanación de reparos", "tarea", "tarea-subsanacion", "Subsanación de reparos"),
    PantallaRRHH(10, "llamamiento", "Llamamiento de la candidatura", "tarea", "tarea-iniciar-llamamiento", "Inicio del llamamiento"),
    PantallaRRHH(11, "seleccion", "Selección de candidatura", "tarea", "tarea-seleccion-candidato", "Selección de candidatura"),
    PantallaRRHH(12, "resultado", "Resultado del llamamiento", "tarea", "tarea-resultado-llamamiento", "Resultado del llamamiento"),
    PantallaRRHH(13, "traslado", "Traslado de la candidatura", "tarea", "tarea-traslado-intervencion", "Traslado de candidatura"),
    PantallaRRHH(14, "documentacion-formalizacion", "Documentación para formalización", "tarea", "tarea-informe-definitivo", "Informe definitivo"),
    PantallaRRHH(15, "datos-ginpix", "Generación de datos para GINPIX", "tarea", "tarea-ginpix", "Preparación de datos para GINPIX"),
    PantallaRRHH(16, "envio-ginpix", "Resumen final y envío a GINPIX", "tarea", "tarea-envio-ginpix", "Resumen final y envío a GINPIX"),
    PantallaRRHH(17, "generacion-documental", "Generación documental para formalización", "tarea", "tarea-formalizacion", "Formalización y firmas"),
)

CSS_CAPTURA_ESTABLE = """
html { scroll-behavior: auto !important; }
*, *::before, *::after {
  animation-duration: 0s !important;
  animation-delay: 0s !important;
  transition-duration: 0s !important;
  caret-color: transparent !important;
}
"""


def validar_matriz(
    pantallas: Sequence[PantallaRRHH] = PANTALLAS,
    tamanos: Sequence[TamanoCaptura] = TAMANOS,
) -> list[str]:
    """Devuelve errores contractuales de la matriz de evidencia."""
    errores: list[str] = []
    if [pantalla.numero for pantalla in pantallas] != list(range(1, 18)):
        errores.append("la numeración debe ser consecutiva del 1 al 17")
    claves = [pantalla.clave for pantalla in pantallas]
    if len(claves) != len(set(claves)):
        errores.append("las claves de pantalla deben ser únicas")
    referencias = [
        pantalla.referencia_tarea
        for pantalla in pantallas
        if pantalla.tipo == "tarea"
    ]
    if len(referencias) != len(set(referencias)):
        errores.append("cada pantalla de tarea debe usar una referencia distinta")
    for pantalla in pantallas:
        if pantalla.tipo not in {"cuadro", "alta", "tarea"}:
            errores.append(f"tipo no admitido en pantalla {pantalla.numero}")
        if pantalla.tipo == "tarea" and (
            not pantalla.referencia_tarea or not pantalla.titulo_esperado
        ):
            errores.append(f"falta contrato de tarea en pantalla {pantalla.numero}")
        if not re.fullmatch(r"[a-z0-9]+(?:-[a-z0-9]+)*", pantalla.clave):
            errores.append(f"clave no canónica en pantalla {pantalla.numero}")
    pares = [(tamano.ancho, tamano.alto) for tamano in tamanos]
    if pares != [(1536, 1024), (1440, 1000), (1280, 900)]:
        errores.append("los tamaños deben conservar la referencia y las dos pasadas de escritorio")
    return errores


def normalizar_url_base(valor: str) -> str:
    """Acepta solo un servidor local explícito y sin credenciales."""
    url = valor.strip().rstrip("/")
    analizada = urlparse(url)
    if analizada.scheme not in {"http", "https"}:
        raise ValueError("la URL local debe usar http o https")
    if analizada.hostname not in {"127.0.0.1", "localhost", "::1"}:
        raise ValueError("la revisión de RRHH solo admite un servidor local")
    if analizada.username or analizada.password:
        raise ValueError("la URL no puede contener credenciales")
    if analizada.path not in {"", "/"} or analizada.query or analizada.fragment:
        raise ValueError("indique únicamente el origen local, sin ruta ni parámetros")
    return url


def _preparar_pantalla(pagina: Any, pantalla: PantallaRRHH, timeout_ms: int) -> str:
    pagina.wait_for_selector(".ct-expedientes", timeout=timeout_ms)
    if pantalla.tipo == "cuadro":
        pagina.wait_for_selector(".ct-exp-indicadores", timeout=timeout_ms)
        return ".ct-expedientes"
    if pantalla.tipo == "alta":
        pagina.locator('[data-ct-exp-vista="alta"]').first.click()
        pagina.wait_for_selector("#ct-alta-titulo", timeout=timeout_ms)
        return ".ct-alta"

    pagina.locator("[data-ct-exp-abrir]").first.click()
    pagina.wait_for_selector("#ct-exp-tarea-titulo", timeout=timeout_ms)
    pagina.locator(
        f'[data-ct-exp-tarea="{pantalla.referencia_tarea}"]',
    ).click()
    pagina.wait_for_function(
        """titulo => document.querySelector("#ct-exp-tarea-titulo")
          ?.textContent.trim() === titulo""",
        arg=pantalla.titulo_esperado,
        timeout=timeout_ms,
    )
    return ".ct-exp-cabecera-expediente"


def _asentar_captura(pagina: Any, selector_ancla: str) -> None:
    pagina.add_style_tag(content=CSS_CAPTURA_ESTABLE)
    pagina.evaluate(
        """selector => {
          const activo = document.activeElement;
          if (activo instanceof HTMLElement) activo.blur();
          const ancla = document.querySelector(selector);
          if (!ancla) throw new Error("ancla de captura ausente");
          const cabecera = document.querySelector(".cabecera-portal");
          const margen = cabecera?.getBoundingClientRect().height ?? 0;
          const destino = Math.max(
            0,
            window.scrollY + ancla.getBoundingClientRect().top - margen - 8,
          );
          window.scrollTo({ top: destino, left: 0, behavior: "instant" });
        }""",
        selector_ancla,
    )
    pagina.wait_for_timeout(80)


def _auditar_pantalla(pagina: Any, contexto: Any) -> dict[str, Any]:
    return pagina.evaluate(
        """() => {
          const raiz = document.documentElement;
          const salto = document.querySelector(".salto-contenido");
          const cajaSalto = salto?.getBoundingClientRect();
          return {
            ancho_cliente: raiz.clientWidth,
            ancho_contenido: raiz.scrollWidth,
            alto_cliente: raiz.clientHeight,
            alto_contenido: raiz.scrollHeight,
            desplazamiento_x: window.scrollX,
            desplazamiento_y: window.scrollY,
            foco: document.activeElement?.tagName ?? "",
            salto_visible: Boolean(cajaSalto && cajaSalto.right > 0 && cajaSalto.bottom > 0),
            cookies_visibles: document.cookie,
            almacenamiento_local: localStorage.length,
            almacenamiento_sesion: sessionStorage.length,
          };
        }""",
    ) | {"cookies_contexto": len(contexto.cookies())}


def _hallazgos_metricas(metricas: dict[str, Any]) -> list[str]:
    hallazgos: list[str] = []
    if metricas["ancho_contenido"] > metricas["ancho_cliente"]:
        hallazgos.append("desbordamiento horizontal")
    if metricas["desplazamiento_x"] != 0:
        hallazgos.append("captura desplazada horizontalmente")
    if metricas["salto_visible"]:
        hallazgos.append("enlace de salto visible por foco accidental")
    if metricas["cookies_visibles"] or metricas["cookies_contexto"]:
        hallazgos.append("se detectaron cookies")
    if metricas["almacenamiento_local"] or metricas["almacenamiento_sesion"]:
        hallazgos.append("se detectó almacenamiento web")
    return hallazgos


def capturar(
    url_base: str,
    salida: Path,
    ejecutable: str | None,
    timeout_ms: int,
) -> tuple[dict[str, Any], int]:
    """Ejecuta la matriz completa y devuelve informe y código de salida."""
    errores_matriz = validar_matriz()
    if errores_matriz:
        raise ValueError("; ".join(errores_matriz))
    if timeout_ms < 500 or timeout_ms > 120_000:
        raise ValueError("el timeout debe estar entre 500 y 120000 ms")

    try:
        from playwright.sync_api import sync_playwright
    except ImportError as error:
        raise RuntimeError("falta Python Playwright") from error

    salida.mkdir(parents=True, exist_ok=True)
    resultados: list[dict[str, Any]] = []
    url = f"{url_base}{RUTA_MODULO}"
    with sync_playwright() as playwright:
        opciones: dict[str, Any] = {"headless": True}
        if ejecutable:
            ruta = Path(ejecutable).expanduser().absolute()
            if not ruta.is_file():
                raise ValueError(f"no existe el navegador: {ruta}")
            opciones["executable_path"] = str(ruta)
        navegador = playwright.chromium.launch(**opciones)
        try:
            for tamano in TAMANOS:
                for pantalla in PANTALLAS:
                    contexto = navegador.new_context(
                        viewport={"width": tamano.ancho, "height": tamano.alto},
                        locale="es-ES",
                        timezone_id="Europe/Madrid",
                        reduced_motion="reduce",
                        service_workers="block",
                    )
                    pagina = contexto.new_page()
                    pagina.set_default_timeout(timeout_ms)
                    errores_consola: list[str] = []
                    errores_pagina: list[str] = []
                    pagina.on(
                        "console",
                        lambda mensaje: errores_consola.append(mensaje.text)
                        if mensaje.type == "error" else None,
                    )
                    pagina.on("pageerror", lambda error: errores_pagina.append(str(error)))
                    hallazgos: list[str] = []
                    metricas: dict[str, Any] = {}
                    destino = (
                        salida
                        / tamano.clave
                        / f"{pantalla.numero:02d}_{pantalla.clave}.png"
                    )
                    destino.parent.mkdir(parents=True, exist_ok=True)
                    try:
                        respuesta = pagina.goto(
                            url,
                            wait_until="domcontentloaded",
                            timeout=timeout_ms,
                        )
                        if respuesta is None or respuesta.status >= 400:
                            raise RuntimeError("la presentación local no respondió correctamente")
                        if respuesta.headers.get("x-vec-modo-presentacion") != "aislada-sintetica-v1":
                            raise RuntimeError("el servidor no acredita el modo de presentación")
                        ancla = _preparar_pantalla(pagina, pantalla, timeout_ms)
                        _asentar_captura(pagina, ancla)
                        metricas = _auditar_pantalla(pagina, contexto)
                        hallazgos.extend(_hallazgos_metricas(metricas))
                        hallazgos.extend(f"consola: {texto}" for texto in errores_consola)
                        hallazgos.extend(f"página: {texto}" for texto in errores_pagina)
                        pagina.screenshot(path=str(destino), full_page=False)
                    except Exception as error:
                        hallazgos.append(str(error))
                    finally:
                        contexto.close()
                    resultados.append({
                        "pantalla": asdict(pantalla),
                        "tamano": asdict(tamano),
                        "captura": str(destino.relative_to(salida)) if destino.is_file() else "",
                        "metricas": metricas,
                        "hallazgos": hallazgos,
                        "correcta": not hallazgos,
                    })
        finally:
            navegador.close()

    correctas = sum(resultado["correcta"] for resultado in resultados)
    informe = {
        "esquema": "vec.revision_web.contratacion_rrhh.v1",
        "generado_en": datetime.now(timezone.utc).isoformat(),
        "url_base": url_base,
        "pantallas": len(PANTALLAS),
        "tamanos": [asdict(tamano) for tamano in TAMANOS],
        "capturas_esperadas": len(PANTALLAS) * len(TAMANOS),
        "capturas_correctas": correctas,
        "correcto": correctas == len(resultados),
        "resultados": resultados,
    }
    return informe, 0 if informe["correcto"] else 1


def guardar_informe(informe: dict[str, Any], salida: Path) -> tuple[Path, Path]:
    ruta_json = salida / "informe.json"
    ruta_md = salida / "informe.md"
    ruta_json.write_text(
        json.dumps(informe, ensure_ascii=False, indent=2) + "\n",
        encoding="utf-8",
    )
    filas = [
        "# Revisión visual de contratación temporal para RRHH",
        "",
        f"- Pantallas: {informe['pantallas']}",
        f"- Capturas correctas: {informe['capturas_correctas']} de {informe['capturas_esperadas']}",
        f"- Resultado: {'VERDE' if informe['correcto'] else 'NO-GO'}",
        "",
        "| N.º | Pantalla | Tamaño | Resultado | Captura |",
        "|---:|---|---:|---|---|",
    ]
    for resultado in informe["resultados"]:
        pantalla = resultado["pantalla"]
        tamano = resultado["tamano"]
        estado = "OK" if resultado["correcta"] else "ERROR"
        captura = resultado["captura"] or "—"
        filas.append(
            f"| {pantalla['numero']} | {pantalla['nombre']} | "
            f"{tamano['ancho']}×{tamano['alto']} | {estado} | {captura} |",
        )
    ruta_md.write_text("\n".join(filas) + "\n", encoding="utf-8")
    return ruta_json, ruta_md


def crear_argumentos() -> argparse.ArgumentParser:
    parser = argparse.ArgumentParser(
        description="Captura las 17 pantallas de contratación indicadas por RRHH.",
    )
    parser.add_argument("--url-base", default="http://127.0.0.1:8081")
    parser.add_argument("--salida", type=Path, default=SALIDA_PREDETERMINADA)
    parser.add_argument("--ejecutable-navegador")
    parser.add_argument("--timeout-ms", type=int, default=15_000)
    return parser


def main(argv: Sequence[str] | None = None) -> int:
    argumentos = crear_argumentos().parse_args(argv)
    try:
        url_base = normalizar_url_base(argumentos.url_base)
        informe, codigo = capturar(
            url_base,
            argumentos.salida,
            argumentos.ejecutable_navegador,
            argumentos.timeout_ms,
        )
        ruta_json, ruta_md = guardar_informe(informe, argumentos.salida)
    except (ValueError, RuntimeError) as error:
        print(f"error: {error}", file=sys.stderr)
        return 2
    print(f"Informe JSON: {ruta_json}")
    print(f"Informe Markdown: {ruta_md}")
    return codigo


if __name__ == "__main__":
    raise SystemExit(main())
