"""Orquestación y CLI de la captura web; Playwright se importa al ejecutar."""

from __future__ import annotations

import argparse
from dataclasses import asdict
from datetime import datetime, timezone
from pathlib import Path
from typing import Any, Sequence

from .informes import codigo_salida, guardar_informes, resumir_resultados
from .modelo import (
    MANIFIESTO,
    SALIDA_PREDETERMINADA,
    SUPERFICIES,
    TAMANOS_VISTA,
    normalizar_url_base,
    validar_manifiesto,
)
from .navegador import capturar_escenario, verificar_servidor_presentacion
from .pantallas_rrhh import (
    MANIFIESTO_PANTALLAS_RRHH,
    MATRIZ_PANTALLAS_RRHH,
    TAMANOS_PANTALLAS_RRHH,
    tamanos_para_pantalla,
    validar_matriz_pantallas_rrhh,
)


def crear_argumentos() -> argparse.ArgumentParser:
    parser = argparse.ArgumentParser(
        description="Captura y revisa todas las vistas y flujos DEMO de la presentación VEC.",
    )
    parser.add_argument(
        "--url-base", "--base-url", default="http://127.0.0.1:8081",
        help="URL base del servidor (predeterminada: %(default)s).",
    )
    parser.add_argument(
        "--salida", type=Path, default=SALIDA_PREDETERMINADA,
        help="Directorio de capturas e informes (predeterminado: var/revision-web).",
    )
    parser.add_argument(
        "--timeout-ms", type=int, default=12_000,
        help="Límite por navegación/estado de aplicación en milisegundos.",
    )
    parser.add_argument(
        "--superficie", action="append", choices=tuple(SUPERFICIES),
        help="Limita el recorrido a una superficie; puede repetirse. Sin esta opción recorre todas.",
    )
    parser.add_argument(
        "--solo-pantallas-rrhh", action="store_true",
        help="Recorre solo la matriz 1..17 en viewport 1536, 1440 y 1280.",
    )
    parser.add_argument(
        "--tolerante", "--modo-tolerante", action="store_true",
        help="Genera todos los hallazgos, pero devuelve código 0.",
    )
    parser.add_argument(
        "--con-interfaz", action="store_true",
        help="Muestra Chromium durante la revisión (por defecto se ejecuta sin interfaz).",
    )
    parser.add_argument(
        "--ejecutable-navegador", type=Path,
        help="Ruta opcional a Chromium; si se omite se usa el navegador administrado por Playwright.",
    )
    parser.add_argument(
        "--red-docker-interna", action="store_true",
        help="Autoriza una IP privada literal como destino al ejecutar el revisor dentro de Docker.",
    )
    return parser


def ejecutar_revision(args: argparse.Namespace) -> tuple[dict[str, Any], int]:
    url_base = normalizar_url_base(
        args.url_base, permitir_red_privada=bool(args.red_docker_interna),
    )
    if args.timeout_ms < 500 or args.timeout_ms > 120_000:
        raise ValueError("--timeout-ms debe estar entre 500 y 120000")
    manifiesto_completo = (*MANIFIESTO, *MANIFIESTO_PANTALLAS_RRHH)
    errores_manifiesto = validar_manifiesto(
        manifiesto_completo,
        tamanos=(*TAMANOS_VISTA, *TAMANOS_PANTALLAS_RRHH),
    )
    errores_manifiesto.extend(validar_matriz_pantallas_rrhh())
    if errores_manifiesto:
        raise ValueError("manifiesto inválido: " + "; ".join(errores_manifiesto))

    superficies_elegidas = set(args.superficie or SUPERFICIES)
    if args.solo_pantallas_rrhh and (
        args.superficie and superficies_elegidas != {"gestion-rrhh"}
    ):
        raise ValueError(
            "--solo-pantallas-rrhh solo admite --superficie gestion-rrhh",
        )
    fuente_escenarios = (
        MANIFIESTO_PANTALLAS_RRHH if args.solo_pantallas_rrhh
        else manifiesto_completo
    )
    escenarios = [
        escenario for escenario in fuente_escenarios
        if escenario.superficie in superficies_elegidas
    ]
    trabajos = [
        (escenario, tamano)
        for escenario in escenarios
        for tamano in (tamanos_para_pantalla(escenario.clave) or TAMANOS_VISTA)
    ]
    total = len(trabajos)
    resultados: list[dict[str, Any]] = []

    try:
        from playwright.sync_api import sync_playwright  # type: ignore[import-not-found]
    except ImportError as error:
        raise RuntimeError(
            "Falta Python Playwright. Instale `playwright` y ejecute `python -m playwright install chromium`."
        ) from error

    print(f"Revisión web: {total} escenarios en {len(TAMANOS_VISTA)} tamaños.", flush=True)
    try:
        with sync_playwright() as playwright:
            opciones_lanzamiento: dict[str, Any] = {"headless": not args.con_interfaz}
            if args.ejecutable_navegador:
                # Snap decide la aplicación desde argv[0], por eso no se resuelve el enlace simbólico.
                ejecutable = args.ejecutable_navegador.expanduser().absolute()
                if not ejecutable.is_file():
                    raise RuntimeError(f"no existe el ejecutable de navegador: {ejecutable}")
                opciones_lanzamiento["executable_path"] = str(ejecutable)
            browser = playwright.chromium.launch(**opciones_lanzamiento)
            try:
                verificar_servidor_presentacion(
                    browser, url_base, args.timeout_ms,
                    permitir_red_privada=bool(args.red_docker_interna),
                )
                completados = 0
                for escenario, tamano in trabajos:
                    resultado = capturar_escenario(
                        browser=browser, escenario=escenario, tamano=tamano,
                        url_base=url_base, directorio_salida=args.salida, timeout_ms=args.timeout_ms,
                        permitir_red_privada=bool(args.red_docker_interna),
                    )
                    resultados.append(resultado)
                    completados += 1
                    porcentaje = completados * 100 / total if total else 100
                    estado = "OK" if resultado["correcto"] else f"{len(resultado['hallazgos'])} hallazgo(s)"
                    print(
                        f"[{porcentaje:6.2f}%] {tamano.ancho}x{tamano.alto} · "
                        f"{escenario.tipo} · {escenario.nombre}: {estado}", flush=True,
                    )
            finally:
                browser.close()
    except RuntimeError:
        raise
    except Exception as error:
        raise RuntimeError(
            "No se pudo iniciar Chromium. Ejecute `python -m playwright install chromium` "
            "o indique `--ejecutable-navegador RUTA`. Detalle: " + str(error)
        ) from error

    resumen = resumir_resultados(resultados)
    salida = codigo_salida(resultados, args.tolerante)
    informe = {
        "version_esquema": 1,
        "generado_en": datetime.now(timezone.utc).isoformat(),
        "url_base": url_base,
        "modo": "tolerante" if args.tolerante else "estricto",
        "tolerante": bool(args.tolerante),
        "correcto": resumen["con_hallazgos"] == 0,
        "codigo_salida": salida,
        "tamanos": [
            asdict(tamano) for tamano in dict.fromkeys(
                tamano for _escenario, tamano in trabajos
            )
        ],
        "superficies": sorted(superficies_elegidas),
        "solo_pantallas_rrhh": bool(args.solo_pantallas_rrhh),
        "matriz_pantallas_rrhh": [
            {
                "numero": pantalla.numero,
                "clave": pantalla.clave,
                "nombre": pantalla.nombre,
                "ruta": pantalla.ruta,
                "perfil": pantalla.perfil,
                "expediente_ref": pantalla.expediente_ref,
                "tarea_ref": pantalla.tarea_ref,
                "pestana": pantalla.pestana,
                "selector_asentamiento": pantalla.selector_asentamiento,
                "nombre_captura": pantalla.nombre_captura,
                "criterios_visuales": list(pantalla.criterios_visuales),
                "paridad": pantalla.paridad,
                "brecha": pantalla.brecha,
                "bloqueo": pantalla.bloqueo,
            }
            for pantalla in MATRIZ_PANTALLAS_RRHH
        ],
        "resumen": resumen,
        "resultados": resultados,
    }
    return informe, salida


def main(argv: Sequence[str] | None = None) -> int:
    parser = crear_argumentos()
    args = parser.parse_args(argv)
    try:
        informe, salida = ejecutar_revision(args)
        ruta_json, ruta_markdown = guardar_informes(informe, args.salida)
    except (ValueError, RuntimeError) as error:
        parser.exit(2, f"error: {error}\n")
    print(f"Informe JSON: {ruta_json}", flush=True)
    print(f"Informe Markdown: {ruta_markdown}", flush=True)
    return salida
