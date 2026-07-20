"""Auditorías de DOM, navegación, menús y flujos de interacción."""

from __future__ import annotations

import json
import re
import time
from typing import Any, Iterable

from .modelo import Escenario, Flujo, Superficie, hallazgo as _hallazgo


AUDITORIA_DOM_JS = r"""
async () => {
  const visible = (elemento) => {
    if (!(elemento instanceof Element)) return false;
    const estilo = getComputedStyle(elemento);
    const caja = elemento.getBoundingClientRect();
    return !elemento.hidden && estilo.display !== "none" &&
      estilo.visibility !== "hidden" && estilo.visibility !== "collapse" &&
      caja.width > 0 && caja.height > 0;
  };
  const textoUtil = (valor) => String(valor || "").replace(/\s+/g, " ").trim();
  const textoDescendiente = (elemento) => {
    const clon = elemento.cloneNode(true);
    clon.querySelectorAll('[aria-hidden="true"], [hidden], script, style').forEach((nodo) => nodo.remove());
    const textosImagen = Array.from(clon.querySelectorAll("img[alt]"), (imagen) => imagen.alt).join(" ");
    return textoUtil(`${clon.textContent || ""} ${textosImagen}`);
  };
  const nombreAccesible = (elemento) => {
    const referencias = textoUtil(elemento.getAttribute("aria-labelledby"));
    if (referencias) {
      const nombre = referencias.split(/\s+/).map((id) => document.getElementById(id)?.textContent || "").join(" ");
      if (textoUtil(nombre)) return textoUtil(nombre);
    }
    const aria = textoUtil(elemento.getAttribute("aria-label"));
    if (aria) return aria;
    if (elemento.labels?.length) {
      const etiquetas = Array.from(elemento.labels, (etiqueta) => etiqueta.textContent || "").join(" ");
      if (textoUtil(etiquetas)) return textoUtil(etiquetas);
    }
    const tipo = textoUtil(elemento.getAttribute("type")).toLowerCase();
    if (elemento.tagName === "INPUT" && ["button", "submit", "reset"].includes(tipo)) {
      if (textoUtil(elemento.value)) return textoUtil(elemento.value);
    }
    if (elemento.tagName === "INPUT" && tipo === "image" && textoUtil(elemento.alt)) return textoUtil(elemento.alt);
    const descendiente = textoDescendiente(elemento);
    if (descendiente) return descendiente;
    return textoUtil(elemento.getAttribute("title"));
  };
  const selectorBreve = (elemento) => {
    if (elemento.id) return `#${CSS.escape(elemento.id)}`;
    const partes = [];
    let actual = elemento;
    while (actual && actual !== document.documentElement && partes.length < 4) {
      let parte = actual.tagName.toLowerCase();
      if (actual.classList.length) parte += `.${CSS.escape(actual.classList[0])}`;
      const hermanos = actual.parentElement ? Array.from(actual.parentElement.children).filter((n) => n.tagName === actual.tagName) : [];
      if (hermanos.length > 1) parte += `:nth-of-type(${hermanos.indexOf(actual) + 1})`;
      partes.unshift(parte);
      actual = actual.parentElement;
    }
    return partes.join(" > ");
  };

  const ids = new Map();
  document.querySelectorAll("[id]").forEach((elemento) => {
    if (elemento.id) ids.set(elemento.id, (ids.get(elemento.id) || 0) + 1);
  });
  const idsDuplicados = Array.from(ids, ([id, cantidad]) => ({ id, cantidad }))
    .filter((item) => item.cantidad > 1);

  const selectorControles = [
    "button", "input:not([type=hidden])", "select", "textarea", "a[href]", "summary",
    "[contenteditable=true]", "[role=button]", "[role=link]", "[role=checkbox]",
    "[role=radio]", "[role=switch]", "[role=tab]", "[role=menuitem]"
  ].join(",");
  const controlesSinNombre = Array.from(new Set(document.querySelectorAll(selectorControles)))
    .filter(visible).filter((elemento) => !nombreAccesible(elemento)).slice(0, 100)
    .map((elemento) => ({
      selector: selectorBreve(elemento), etiqueta: elemento.tagName.toLowerCase(),
      tipo: elemento.getAttribute("type") || elemento.getAttribute("role") || "",
    }));

  const anchoCliente = document.documentElement.clientWidth;
  const anchoDocumento = Math.max(document.documentElement.scrollWidth, document.body?.scrollWidth || 0);
  const hayDesbordamiento = anchoDocumento > anchoCliente + 2;
  const elementosDesbordados = hayDesbordamiento
    ? Array.from(document.body.querySelectorAll("*")).filter(visible).filter((elemento) => {
        const caja = elemento.getBoundingClientRect();
        return caja.right > anchoCliente + 2 || caja.left < -2;
      }).slice(0, 30).map((elemento) => ({
        selector: selectorBreve(elemento), izquierda: Math.round(elemento.getBoundingClientRect().left),
        derecha: Math.round(elemento.getBoundingClientRect().right),
      }))
    : [];

  // La puerta de esta entrega es escritorio/portátil. En móvil las tablas se
  // conservan desplazables, pero su rediseño visual está expresamente fuera de
  // alcance; no se rebaja la comprobación de 1024 px ni de anchuras superiores.
  const auditarColumnasOperativas = anchoCliente >= 1024;
  const columnasOperativas = (auditarColumnasOperativas
    ? Array.from(document.querySelectorAll("[data-tabla-prioritaria]")) : [])
    .filter(visible).map((contenedor) => {
      const prioridad = contenedor.getAttribute("data-tabla-prioritaria") || "";
      const requeridas = prioridad === "estado-acciones" ? ["estado", "acciones"] : ["estado"];
      const cajaContenedor = contenedor.getBoundingClientRect();
      const limiteIzquierdo = Math.max(0, cajaContenedor.left) - 2;
      const limiteDerecho = Math.min(anchoCliente, cajaContenedor.right) + 2;
      const celdas = Array.from(contenedor.querySelectorAll(
        '[data-columna="estado"], [data-columna="acciones"]',
      )).filter(visible);
      const recortadas = celdas.filter((celda) => {
        const caja = celda.getBoundingClientRect();
        return caja.left < limiteIzquierdo || caja.right > limiteDerecho;
      }).map((celda) => ({
        columna: celda.getAttribute("data-columna"),
        izquierda: Math.round(celda.getBoundingClientRect().left),
        derecha: Math.round(celda.getBoundingClientRect().right),
      }));
      const sinFijar = celdas.filter((celda) => getComputedStyle(celda).position !== "sticky")
        .map((celda) => celda.getAttribute("data-columna"));
      const controlesRecortados = Array.from(contenedor.querySelectorAll(
        '[data-columna="acciones"] button, [data-columna="acciones"] a[href]',
      )).filter(visible).filter((control) => {
        const caja = control.getBoundingClientRect();
        return caja.left < limiteIzquierdo || caja.right > limiteDerecho;
      }).map((control) => ({
        nombre: nombreAccesible(control),
        izquierda: Math.round(control.getBoundingClientRect().left),
        derecha: Math.round(control.getBoundingClientRect().right),
      }));
      const contenidosEstadoRecortados = Array.from(contenedor.querySelectorAll(
        '[data-columna="estado"] .estado-chip, [data-columna="estado"] .insignia',
      )).filter(visible).filter((contenido) => {
        const celda = contenido.closest('[data-columna="estado"]');
        if (!celda) return true;
        const caja = contenido.getBoundingClientRect();
        const cajaCelda = celda.getBoundingClientRect();
        const izquierda = Math.max(limiteIzquierdo, cajaCelda.left) - 2;
        const derecha = Math.min(limiteDerecho, cajaCelda.right) + 2;
        return caja.left < izquierda || caja.right > derecha ||
          contenido.scrollWidth > contenido.clientWidth + 2 ||
          contenido.scrollHeight > contenido.clientHeight + 2;
      }).map((contenido) => ({
        nombre: nombreAccesible(contenido) || (contenido.textContent || "").trim(),
        izquierda: Math.round(contenido.getBoundingClientRect().left),
        derecha: Math.round(contenido.getBoundingClientRect().right),
      }));
      const solapadas = Array.from(contenedor.querySelectorAll("tr")).filter(visible).flatMap((fila) => {
        const estado = fila.querySelector('[data-columna="estado"]');
        const acciones = fila.querySelector('[data-columna="acciones"]');
        if (!visible(estado) || !visible(acciones)) return [];
        const cajaEstado = estado.getBoundingClientRect();
        const cajaAcciones = acciones.getBoundingClientRect();
        return cajaEstado.right > cajaAcciones.left + 2
          ? [{ estado_derecha: Math.round(cajaEstado.right), acciones_izquierda: Math.round(cajaAcciones.left) }]
          : [];
      });
      const cabecera = contenedor.querySelector("thead tr");
      const faltantes = requeridas.filter((columna) =>
        !visible(cabecera?.querySelector(`[data-columna="${columna}"]`)),
      );
      const filasSinColumnas = Array.from(contenedor.querySelectorAll("tbody tr"))
        .filter(visible).flatMap((fila, indice) => {
          if (fila.querySelector("td[colspan]")) return [];
          const columnasFaltantes = requeridas.filter((columna) =>
            !visible(fila.querySelector(`[data-columna="${columna}"]`)),
          );
          return columnasFaltantes.length > 0
            ? [{ fila: indice + 1, columnas_faltantes: columnasFaltantes }]
            : [];
        });
      return {
        selector: selectorBreve(contenedor), prioridad, faltantes, recortadas,
        sin_fijar: Array.from(new Set(sinFijar)), controles_recortados: controlesRecortados,
        contenidos_estado_recortados: contenidosEstadoRecortados,
        filas_sin_columnas: filasSinColumnas, solapadas,
        correcta: faltantes.length === 0 && recortadas.length === 0 &&
          sinFijar.length === 0 && controlesRecortados.length === 0 &&
          contenidosEstadoRecortados.length === 0 && filasSinColumnas.length === 0 &&
          solapadas.length === 0,
      };
    });

  const leerAlmacen = (obtenerAlmacen) => {
    try {
      const almacen = obtenerAlmacen();
      return Array.from({ length: almacen.length }, (_, indice) => almacen.key(indice)).filter(Boolean);
    }
    catch (error) { return [`<no se pudo inspeccionar: ${error}>`]; }
  };
  const leerIndexedDB = async () => {
    try {
      if (!window.indexedDB || typeof window.indexedDB.databases !== "function") {
        return ["<API de enumeración IndexedDB no disponible>"];
      }
      return (await window.indexedDB.databases())
        .map((base) => textoUtil(base.name) || "<base sin nombre>");
    } catch (error) { return [`<no se pudo inspeccionar IndexedDB: ${error}>`]; }
  };
  const leerCaches = async () => {
    try {
      if (!window.caches || typeof window.caches.keys !== "function") {
        return ["<Cache Storage no disponible>"];
      }
      return await window.caches.keys();
    } catch (error) { return [`<no se pudo inspeccionar Cache Storage: ${error}>`]; }
  };
  let cookie = "";
  try { cookie = document.cookie || ""; } catch (error) { cookie = `<no se pudo inspeccionar: ${error}>`; }
  return {
    ids_duplicados: idsDuplicados,
    controles_sin_nombre: controlesSinNombre,
    desbordamiento_horizontal: {
      existe: hayDesbordamiento, ancho_cliente: anchoCliente,
      ancho_documento: anchoDocumento, elementos: elementosDesbordados,
    },
    columnas_operativas: columnasOperativas,
    almacenamiento: {
      local: leerAlmacen(() => window.localStorage), sesion: leerAlmacen(() => window.sessionStorage),
      indexeddb: await leerIndexedDB(), cache: await leerCaches(),
      cookie_documento: cookie,
    },
  };
}
"""

CSS_ESTABILIZAR = """
*, *::before, *::after {
  animation-duration: 0s !important;
  animation-delay: 0s !important;
  caret-color: transparent !important;
  scroll-behavior: auto !important;
  transition-duration: 0s !important;
  transition-delay: 0s !important;
}
"""


def milisegundos_restantes(fin: float) -> int:
    return max(1, int((fin - time.monotonic()) * 1000))


def esperar_vista(page: Any, escenario: Escenario, timeout_ms: int) -> tuple[bool, str]:
    fin = time.monotonic() + timeout_ms / 1000
    try:
        for selector in escenario.selectores_listos:
            page.wait_for_selector(selector, state="visible", timeout=milisegundos_restantes(fin))
        page.wait_for_function(
            r"""([selector, esperado]) => {
              const nodo = document.querySelector(selector);
              return nodo && nodo.textContent.replace(/\s+/g, " ").trim() === esperado;
            }""",
            arg=[escenario.selector_titulo, escenario.titulo_esperado],
            timeout=milisegundos_restantes(fin),
        )
        return True, ""
    except Exception as error:
        return False, str(error)


def nombre_elemento(locator: Any) -> str:
    return locator.evaluate(r"""(elemento) => {
      const normalizar = (texto) => String(texto || "").replace(/\s+/g, " ").trim();
      const referencias = normalizar(elemento.getAttribute("aria-labelledby"));
      if (referencias) {
        const texto = referencias.split(/\s+/).map((id) => document.getElementById(id)?.textContent || "").join(" ");
        if (normalizar(texto)) return normalizar(texto);
      }
      return normalizar(elemento.getAttribute("aria-label")) || normalizar(elemento.innerText) ||
        normalizar(elemento.getAttribute("title")) || normalizar(elemento.getAttribute("alt"));
    }""")


def primer_locator_visible(page: Any, selector: str) -> Any | None:
    localizadores = page.locator(selector)
    for indice in range(localizadores.count()):
        candidato = localizadores.nth(indice)
        try:
            if candidato.is_visible():
                return candidato
        except Exception:
            continue
    return None


def revelar_opcion_menu(
    page: Any,
    selector: str,
    timeout_ms: int,
    controles_expandidos: list[Any],
) -> tuple[Any | None, dict[str, Any] | None]:
    """Devuelve una opción visible, abriendo sus acordeones accesibles si procede."""
    opcion = primer_locator_visible(page, selector)
    if opcion is not None:
        return opcion, None

    opciones = page.locator(selector)
    if opciones.count() == 0:
        return None, _hallazgo("opcion_menu_ausente", f"No existe la opción de menú {selector}.")

    for indice in range(opciones.count()):
        candidata = opciones.nth(indice)
        contenedores = candidata.evaluate(r"""(elemento) => {
          const identificadores = [];
          for (let padre = elemento.parentElement; padre; padre = padre.parentElement) {
            if (padre.hidden && padre.id) identificadores.push(padre.id);
          }
          return identificadores.reverse();
        }""")
        for identificador in contenedores:
            valor = str(identificador).replace("\\", "\\\\").replace('"', '\\"')
            controles = page.locator(f'[aria-controls="{valor}"]')
            control = next(
                (
                    controles.nth(numero)
                    for numero in range(controles.count())
                    if controles.nth(numero).is_visible()
                ),
                None,
            )
            if control is None:
                return None, _hallazgo(
                    "opcion_menu_inaccesible",
                    f"La opción {selector} está oculta y #{identificador} no tiene un control visible.",
                )
            if not control.is_enabled() or control.get_attribute("aria-disabled") == "true":
                return None, _hallazgo(
                    "opcion_menu_inaccesible",
                    f"El control que revela {selector} está deshabilitado.",
                )
            if not nombre_elemento(control):
                return None, _hallazgo(
                    "opcion_menu_inaccesible",
                    f"El control que revela {selector} no tiene nombre accesible.",
                )
            if control.get_attribute("aria-expanded") != "true":
                control.focus(timeout=min(timeout_ms, 2_000))
                if not control.evaluate("(elemento) => document.activeElement === elemento"):
                    return None, _hallazgo(
                        "opcion_menu_inaccesible",
                        f"El control que revela {selector} no puede recibir el foco.",
                    )
                page.keyboard.press("Enter")
                control.wait_for(state="visible", timeout=min(timeout_ms, 2_000))
                if control.get_attribute("aria-expanded") != "true":
                    return None, _hallazgo(
                        "opcion_menu_inaccesible",
                        f"El control de {selector} no refleja aria-expanded=true tras activarlo.",
                    )
                controles_expandidos.append(control)
        try:
            candidata.wait_for(state="visible", timeout=min(timeout_ms, 2_000))
            return candidata, None
        except Exception:
            continue

    return None, _hallazgo(
        "opcion_menu_inaccesible",
        f"La opción de menú {selector} existe, pero no puede revelarse mediante sus controles.",
    )


def revisar_menu(page: Any, escenario: Escenario, superficie: Superficie, timeout_ms: int) -> list[dict[str, Any]]:
    hallazgos: list[dict[str, Any]] = []
    menu_abierto = False
    controles_expandidos: list[Any] = []
    contenedor_menu = None
    desplazamiento_inicial = None
    try:
        if superficie.selector_contenedor_menu:
            contenedor = page.locator(superficie.selector_contenedor_menu).first
            contenedor_menu = contenedor
            if contenedor.count() == 0:
                return [_hallazgo("menu_ausente", f"No existe el menú esperado {superficie.selector_contenedor_menu}.")]
            if not contenedor.is_visible() and superficie.selector_abrir_menu:
                apertura = primer_locator_visible(page, superficie.selector_abrir_menu)
                if apertura is None or not apertura.is_enabled() or not nombre_elemento(apertura):
                    return [_hallazgo("menu_inaccesible", "El menú está oculto y su control de apertura no es accesible.")]
                apertura.click(timeout=timeout_ms)
                contenedor.wait_for(state="visible", timeout=min(timeout_ms, 2_000))
                menu_abierto = True
            if not contenedor.is_visible():
                return [_hallazgo("menu_inaccesible", "El contenedor del menú no es visible.")]
            if not nombre_elemento(contenedor):
                hallazgos.append(_hallazgo("menu_sin_nombre", "El contenedor del menú no tiene nombre accesible."))
            desplazamiento_inicial = contenedor.evaluate("elemento => elemento.scrollTop")

        for selector in escenario.selectores_menu or superficie.selectores_menu:
            opcion, hallazgo_opcion = revelar_opcion_menu(
                page, selector, timeout_ms, controles_expandidos,
            )
            if opcion is None:
                if hallazgo_opcion:
                    hallazgos.append(hallazgo_opcion)
                continue
            if not opcion.is_enabled() or opcion.get_attribute("aria-disabled") == "true":
                hallazgos.append(_hallazgo("opcion_menu_inaccesible", f"La opción de menú {selector} está deshabilitada."))
            if not nombre_elemento(opcion):
                hallazgos.append(_hallazgo("opcion_menu_sin_nombre", f"La opción de menú {selector} no tiene nombre accesible."))

        if escenario.selector_menu_actual:
            actuales = page.locator(escenario.selector_menu_actual)
            marcado = any(
                actuales.nth(indice).is_visible() and actuales.nth(indice).get_attribute("aria-current") == "page"
                for indice in range(actuales.count())
            )
            if not marcado:
                hallazgos.append(_hallazgo(
                    "menu_sin_estado_actual",
                    f"La vista no está marcada en el menú mediante aria-current: {escenario.selector_menu_actual}.",
                ))
    except Exception as error:
        hallazgos.append(_hallazgo("revision_menu_fallida", "No se pudo completar la revisión del menú.", str(error)))
    finally:
        for control in reversed(controles_expandidos):
            try:
                if control.is_visible() and control.get_attribute("aria-expanded") == "true":
                    control.focus(timeout=min(timeout_ms, 2_000))
                    page.keyboard.press("Enter")
                    if control.get_attribute("aria-expanded") != "false":
                        raise RuntimeError("el control no refleja aria-expanded=false")
            except Exception as error:
                hallazgos.append(_hallazgo(
                    "submenu_no_cierra",
                    "Un submenú no pudo restaurarse tras comprobar sus opciones.",
                    str(error),
                ))
        if menu_abierto:
            try:
                cierre = primer_locator_visible(page, superficie.selector_cerrar_menu) if superficie.selector_cerrar_menu else None
                if cierre is not None:
                    cierre.click(timeout=min(timeout_ms, 2_000))
                else:
                    page.locator(superficie.selector_abrir_menu).first.evaluate("(elemento) => elemento.click()")
            except Exception as error:
                hallazgos.append(_hallazgo("menu_no_cierra", "El menú móvil no se pudo cerrar tras revisarlo.", str(error)))
        if contenedor_menu is not None and desplazamiento_inicial is not None:
            try:
                contenedor_menu.evaluate(
                    "(elemento, posicion) => { elemento.scrollTop = posicion; }",
                    desplazamiento_inicial,
                )
            except Exception as error:
                hallazgos.append(_hallazgo(
                    "menu_no_restaura_posicion",
                    "El menú no pudo recuperar su posición tras la auditoría.",
                    str(error),
                ))
    return hallazgos


def revisar_banner_demo(page: Any, superficie: Superficie) -> tuple[bool, list[dict[str, Any]]]:
    if not superficie.privada:
        return True, []
    selector = superficie.selector_banner_demo
    if not selector:
        return False, [_hallazgo("banner_demo_no_definido", "La superficie privada no define su banner DEMO.")]
    banner = primer_locator_visible(page, selector)
    if banner is None:
        return False, [_hallazgo("banner_demo_ausente", f"El banner DEMO privado no está visible: {selector}.")]
    texto = re.sub(r"\s+", " ", banner.inner_text()).strip()
    marcador = re.search(r"\b(?:demo|demostraci[oó]n|presentaci[oó]n)\b", texto, re.IGNORECASE)
    sintetico = re.search(r"sint[eé]tic|sin validez|sin efectos", texto, re.IGNORECASE)
    if not marcador or not sintetico:
        return False, [_hallazgo(
            "banner_demo_invalido",
            "El banner privado no identifica claramente la demostración y sus datos/efectos sintéticos.",
            {"texto": texto},
        )]
    return True, []


def ejecutar_flujo(page: Any, flujo: Flujo, timeout_ms: int, demo_confirmada: bool) -> list[dict[str, Any]]:
    if flujo.requiere_demo and not demo_confirmada:
        return [_hallazgo("flujo_demo_bloqueado", "El flujo no se ejecutó porque no se pudo confirmar el aislamiento DEMO.")]
    fin = time.monotonic() + timeout_ms / 1000
    for numero, paso in enumerate(flujo.pasos, start=1):
        try:
            restante = milisegundos_restantes(fin)
            locator = page.locator(paso.selector).first
            if paso.accion == "esperar":
                locator.wait_for(state="visible", timeout=restante)
                if paso.texto_esperado:
                    texto = re.sub(r"\s+", " ", locator.inner_text(timeout=restante)).strip()
                    if paso.texto_esperado.casefold() not in texto.casefold():
                        raise RuntimeError(f"no aparece el texto esperado {paso.texto_esperado!r}")
            elif paso.accion in {"esperar-habilitado", "esperar-deshabilitado"}:
                locator.wait_for(state="attached", timeout=restante)
                deshabilitado = locator.is_disabled() or locator.get_attribute("aria-disabled") == "true"
                esperado = paso.accion == "esperar-deshabilitado"
                if deshabilitado != esperado:
                    estado = "deshabilitado" if esperado else "habilitado"
                    raise RuntimeError(f"el control no está {estado}")
            elif paso.accion == "enfocar":
                locator.scroll_into_view_if_needed(timeout=restante)
            elif paso.accion == "abrir-menu":
                locator.wait_for(state="attached", timeout=restante)
                if locator.is_visible() and locator.get_attribute("aria-expanded") != "true":
                    locator.click(timeout=restante)
            elif paso.accion == "clic-confirmando":
                selectores_confirmacion_demo = ("operacion-presentacion", "preparar-llamamiento-demo")
                if (not flujo.requiere_demo
                        or not any(marcador in paso.selector for marcador in selectores_confirmacion_demo)):
                    raise RuntimeError("se rechazó un intento de confirmar una operación fuera del adaptador DEMO")
                page.once("dialog", lambda dialogo: dialogo.accept())
                locator.click(timeout=restante)
            elif paso.accion == "clic":
                locator.click(timeout=restante)
            else:
                raise RuntimeError(f"acción de flujo no admitida: {paso.accion}")
        except Exception as error:
            return [_hallazgo(
                "flujo_interrumpido", f"Falló el paso {numero} ({paso.accion}) del flujo {flujo.nombre}.",
                {"selector": paso.selector, "error": str(error)},
            )]
    return []


def auditar_dom_y_estado(page: Any, context: Any) -> tuple[dict[str, Any], list[dict[str, Any]]]:
    try:
        auditoria = page.evaluate(AUDITORIA_DOM_JS)
    except Exception as error:
        return {}, [_hallazgo("auditoria_dom_fallida", "No se pudo ejecutar la auditoría del DOM.", str(error))]
    hallazgos: list[dict[str, Any]] = []
    if auditoria["ids_duplicados"]:
        hallazgos.append(_hallazgo("ids_duplicados", f"Se encontraron {len(auditoria['ids_duplicados'])} identificadores HTML duplicados.", auditoria["ids_duplicados"]))
    if auditoria["controles_sin_nombre"]:
        hallazgos.append(_hallazgo("controles_sin_nombre_accesible", f"Hay {len(auditoria['controles_sin_nombre'])} controles visibles sin nombre accesible.", auditoria["controles_sin_nombre"]))
    if auditoria["desbordamiento_horizontal"]["existe"]:
        hallazgos.append(_hallazgo("desbordamiento_horizontal", "La página desborda horizontalmente el ancho disponible.", auditoria["desbordamiento_horizontal"]))
    columnas_defectuosas = [
        tabla for tabla in auditoria.get("columnas_operativas", [])
        if not tabla.get("correcta", False)
    ]
    if columnas_defectuosas:
        hallazgos.append(_hallazgo(
            "columnas_operativas_recortadas",
            "Estado o Acciones no permanecen completamente visibles y operables en una tabla prioritaria.",
            columnas_defectuosas,
        ))

    almacen = auditoria["almacenamiento"]
    try:
        cookies = context.cookies()
    except Exception as error:
        cookies = [{"error": str(error)}]
    auditoria["almacenamiento"]["cookies_contexto"] = [
        {"nombre": cookie.get("name", ""), "dominio": cookie.get("domain", ""), "ruta": cookie.get("path", "")}
        for cookie in cookies
    ]
    if (almacen["local"] or almacen["sesion"] or almacen["indexeddb"] or
            almacen["cache"] or almacen["cookie_documento"] or cookies):
        hallazgos.append(_hallazgo(
            "estado_navegador_detectado",
            "La superficie creó o expuso estado persistente del navegador en un contexto limpio.",
            auditoria["almacenamiento"],
        ))
    return auditoria, hallazgos


def deduplicar_registros(registros: Iterable[dict[str, Any]]) -> list[dict[str, Any]]:
    unicos: list[dict[str, Any]] = []
    vistos: set[str] = set()
    for registro in registros:
        clave = json.dumps(registro, ensure_ascii=False, sort_keys=True, default=str)
        if clave not in vistos:
            vistos.add(clave)
            unicos.append(registro)
    return unicos


def filtrar_abortos_media_exitosos(
    recursos_fallidos: Iterable[dict[str, Any]],
    respuestas_correctas: Iterable[dict[str, Any]],
) -> list[dict[str, Any]]:
    """Descarta un aborto normal de ``preload=metadata`` tras un HTTP válido."""
    media_correctos = {
        respuesta.get("url") for respuesta in respuestas_correctas
        if respuesta.get("tipo") == "media" and int(respuesta.get("estado", 999)) < 400
    }
    return [
        recurso for recurso in recursos_fallidos
        if not (recurso.get("tipo") == "media" and recurso.get("url") in media_correctos
                and "ERR_ABORTED" in str(recurso.get("error", "")))
    ]
