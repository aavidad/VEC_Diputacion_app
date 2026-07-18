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
    cabecera_presentacion_valida,
    construir_url,
    hallazgo as _hallazgo,
    slug_castellano,
)


def verificar_servidor_presentacion(browser: Any, url_base: str, timeout_ms: int) -> None:
    """Falla antes del recorrido si el proceso local no acredita ser la DEMO."""
    context = browser.new_context(service_workers="block")
    try:
        respuesta = context.request.get(
            construir_url(url_base, "/presentacion/"),
            timeout=timeout_ms,
            fail_on_status_code=False,
        )
        if respuesta.status != 200:
            raise RuntimeError(
                f"el servidor local de presentación respondió HTTP {respuesta.status} en la comprobación previa"
            )
        if not cabecera_presentacion_valida(respuesta.headers):
            raise RuntimeError(
                "el servidor local no emitió la cabecera técnica exacta del modo presentación; "
                "se bloquea el recorrido para impedir operaciones sobre otro servicio"
            )
    finally:
        context.close()


def esperar_red_estable(page: Any, pendientes: dict[int, dict[str, str]], timeout_ms: int) -> list[dict[str, Any]]:
    """Espera una ventana sin peticiones y denuncia las que continúan activas."""
    silencio_segundos = 0.25
    fin = time.monotonic() + min(timeout_ms, 3_000) / 1_000
    estable_desde: float | None = None
    while time.monotonic() < fin:
        ahora = time.monotonic()
        if pendientes:
            estable_desde = None
        else:
            estable_desde = estable_desde or ahora
            if ahora - estable_desde >= silencio_segundos:
                return []
        page.wait_for_timeout(min(50, max(1, int((fin - ahora) * 1_000))))
    if not pendientes:
        return []
    detalle = list(pendientes.values())[:30]
    return [_hallazgo(
        "red_no_estable",
        f"Persisten {len(pendientes)} peticiones de red al cerrar la observación acotada.",
        detalle,
    )]


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
    peticiones_pendientes: dict[int, dict[str, str]] = {}
    auditoria: dict[str, Any] = {}
    context = None
    captura_guardada = False
    cabecera_tecnica_confirmada = False

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

        def a_peticion(peticion: Any) -> None:
            peticiones_pendientes[id(peticion)] = {
                "url": peticion.url,
                "tipo": peticion.resource_type,
            }

        def a_peticion_finalizada(peticion: Any) -> None:
            peticiones_pendientes.pop(id(peticion), None)

        def a_recurso_fallido(peticion: Any) -> None:
            a_peticion_finalizada(peticion)
            fallo = peticion.failure
            recursos_fallidos.append({
                "url": peticion.url, "tipo": peticion.resource_type,
                "error": fallo if isinstance(fallo, str) else str(fallo or "fallo de red"),
            })

        page.on("console", al_mensaje_consola)
        page.on("pageerror", al_error_pagina)
        page.on("request", a_peticion)
        page.on("requestfinished", a_peticion_finalizada)
        page.on("response", a_respuesta)
        page.on("requestfailed", a_recurso_fallido)

        try:
            respuesta = page.goto(url, wait_until="domcontentloaded", timeout=timeout_ms)
            if respuesta is None:
                hallazgos.append(_hallazgo("respuesta_principal_ausente", "La navegación no devolvió una respuesta HTTP principal."))
            elif respuesta.status >= 400:
                hallazgos.append(_hallazgo("http_principal_roto", f"La vista respondió HTTP {respuesta.status}.", {"url": respuesta.url}))
            else:
                cabecera_tecnica_confirmada = cabecera_presentacion_valida(respuesta.headers)
                if not cabecera_tecnica_confirmada:
                    hallazgos.append(_hallazgo(
                        "servidor_presentacion_no_confirmado",
                        "La respuesta principal no acredita el modo de presentación aislada.",
                    ))
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
            hallazgos.extend(ejecutar_flujo(
                page, escenario, timeout_ms,
                demo_confirmada and cabecera_tecnica_confirmada,
            ))

        hallazgos.extend(esperar_red_estable(page, peticiones_pendientes, timeout_ms))
        try:
            page.screenshot(
                path=str(destino_captura), full_page=escenario.tipo == "vista", animations="disabled",
            )
            captura_guardada = True
        except Exception as error:
            hallazgos.append(_hallazgo("captura_fallida", "No se pudo guardar la captura.", str(error)))
        page.wait_for_timeout(50)

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
