"""Captura aislada de un escenario mediante una instancia Playwright recibida."""

from __future__ import annotations

import time
from dataclasses import asdict
from pathlib import Path
from typing import Any

from .auditoria import (
    CSS_ESTABILIZAR,
    auditar_dom_y_estado,
    deduplicar_registros,
    ejecutar_flujo,
    esperar_vista,
    filtrar_abortos_media_exitosos,
    revisar_banner_demo,
    revisar_menu,
)
from .modelo import (
    SUPERFICIES,
    Escenario,
    Flujo,
    TamanoVista,
    construir_url,
    hallazgo as _hallazgo,
    slug_castellano,
)


def capturar_escenario(
    browser: Any,
    escenario: Escenario,
    tamano: TamanoVista,
    url_base: str,
    directorio_salida: Path,
    timeout_ms: int,
) -> dict[str, Any]:
    inicio = time.monotonic()
    superficie = SUPERFICIES[escenario.superficie]
    url = construir_url(url_base, escenario.ruta)
    ruta_captura = Path("capturas") / tamano.clave / escenario.tipo / superficie.clave / f"{slug_castellano(escenario.clave)}.png"
    destino_captura = directorio_salida / ruta_captura
    destino_captura.parent.mkdir(parents=True, exist_ok=True)

    hallazgos: list[dict[str, Any]] = []
    errores_consola: list[dict[str, Any]] = []
    errores_pagina: list[str] = []
    respuestas_http: list[dict[str, Any]] = []
    respuestas_correctas: list[dict[str, Any]] = []
    recursos_fallidos: list[dict[str, Any]] = []
    auditoria: dict[str, Any] = {}
    context = None
    captura_guardada = False

    try:
        context = browser.new_context(
            viewport={"width": tamano.ancho, "height": tamano.alto},
            locale="es-ES", timezone_id="Europe/Madrid", color_scheme="light",
            reduced_motion="reduce", service_workers="block", accept_downloads=False,
        )
        page = context.new_page()
        page.set_default_timeout(timeout_ms)
        page.set_default_navigation_timeout(timeout_ms)

        def al_mensaje_consola(mensaje: Any) -> None:
            if mensaje.type == "error":
                errores_consola.append({"texto": mensaje.text, "ubicacion": mensaje.location})

        def al_error_pagina(error: Any) -> None:
            errores_pagina.append(str(error))

        def a_respuesta(respuesta: Any) -> None:
            registro = {
                "estado": respuesta.status, "url": respuesta.url,
                "tipo": respuesta.request.resource_type,
            }
            (respuestas_http if respuesta.status >= 400 else respuestas_correctas).append(registro)

        def a_recurso_fallido(peticion: Any) -> None:
            fallo = peticion.failure
            recursos_fallidos.append({
                "url": peticion.url, "tipo": peticion.resource_type,
                "error": fallo if isinstance(fallo, str) else str(fallo or "fallo de red"),
            })

        page.on("console", al_mensaje_consola)
        page.on("pageerror", al_error_pagina)
        page.on("response", a_respuesta)
        page.on("requestfailed", a_recurso_fallido)

        try:
            respuesta = page.goto(url, wait_until="domcontentloaded", timeout=timeout_ms)
            if respuesta is None:
                hallazgos.append(_hallazgo("respuesta_principal_ausente", "La navegación no devolvió una respuesta HTTP principal."))
            elif respuesta.status >= 400:
                hallazgos.append(_hallazgo("http_principal_roto", f"La vista respondió HTTP {respuesta.status}.", {"url": respuesta.url}))
        except Exception as error:
            hallazgos.append(_hallazgo("navegacion_fallida", "La navegación no alcanzó DOMContentLoaded dentro del límite.", str(error)))

        try:
            page.add_style_tag(content=CSS_ESTABILIZAR)
        except Exception:
            pass

        lista, detalle = esperar_vista(page, escenario, timeout_ms)
        if not lista:
            hallazgos.append(_hallazgo(
                "vista_no_carga",
                f"La vista no alcanzó su estado listo ni el título {escenario.titulo_esperado!r}.", detalle,
            ))
        hallazgos.extend(revisar_menu(page, escenario, superficie, timeout_ms))
        demo_confirmada, hallazgos_banner = revisar_banner_demo(page, superficie)
        hallazgos.extend(hallazgos_banner)
        if isinstance(escenario, Flujo):
            hallazgos.extend(ejecutar_flujo(page, escenario, timeout_ms, demo_confirmada))

        page.wait_for_timeout(120)
        try:
            page.screenshot(
                path=str(destino_captura), full_page=escenario.tipo == "vista", animations="disabled",
            )
            captura_guardada = True
        except Exception as error:
            hallazgos.append(_hallazgo("captura_fallida", "No se pudo guardar la captura.", str(error)))
        page.wait_for_timeout(80)

        auditoria, hallazgos_dom = auditar_dom_y_estado(page, context)
        hallazgos.extend(hallazgos_dom)
        if respuestas_http:
            rotas = deduplicar_registros(respuestas_http)
            hallazgos.append(_hallazgo(
                "respuestas_http_rotas", f"Se observaron {len(rotas)} respuestas HTTP de error.", rotas,
            ))
        recursos_fallidos = filtrar_abortos_media_exitosos(recursos_fallidos, respuestas_correctas)
        if recursos_fallidos:
            rotos = deduplicar_registros(recursos_fallidos)
            hallazgos.append(_hallazgo("recursos_rotos", f"Fallaron {len(rotos)} recursos o peticiones.", rotos))
        if errores_consola:
            errores = deduplicar_registros(errores_consola)
            hallazgos.append(_hallazgo("errores_consola", f"La consola emitió {len(errores)} errores.", errores))
        if errores_pagina:
            errores_unicos = list(dict.fromkeys(errores_pagina))
            hallazgos.append(_hallazgo(
                "errores_javascript",
                f"La página lanzó {len(errores_unicos)} errores JavaScript no controlados.", errores_unicos,
            ))
    except Exception as error:
        hallazgos.append(_hallazgo("escenario_interrumpido", "El escenario se interrumpió de forma inesperada.", str(error)))
    finally:
        if context is not None:
            try:
                context.close()
            except Exception:
                pass

    hallazgos = deduplicar_registros(hallazgos)
    return {
        "tipo": escenario.tipo, "clave": escenario.clave, "nombre": escenario.nombre,
        "superficie": superficie.clave, "nombre_superficie": superficie.nombre,
        "ruta": escenario.ruta, "url": url, "tamano": asdict(tamano),
        "captura": ruta_captura.as_posix() if captura_guardada else None,
        "alcance_captura": "pagina-completa" if escenario.tipo == "vista" else "viewport-flujo",
        "correcto": not hallazgos,
        "duracion_ms": round((time.monotonic() - inicio) * 1000),
        "hallazgos": hallazgos, "metricas": auditoria,
    }
