import assert from "node:assert/strict";
import { readFile } from "node:fs/promises";
import test from "node:test";
import {
  CATEGORIAS_MENU_BOLSA,
  VISTA_CANDIDATOS_BOLSA,
  VISTAS_INTERNAS_BOLSA,
  alternarGrupoBolsa,
  categoriaDeVistaBolsa,
  instalarMenuBolsa,
  resumenAccesosModulos,
  sincronizarMenuBolsa,
  vistaBolsaPendienteNoCompuesta,
} from "./portal-menu-bolsa.js";

test("P-WEB-14 no anuncia un total mientras Bolsa sigue comprobando", () => {
  const accesos = [{ disponible: true, estado: "disponible" }, { disponible: false, estado: "cargando" }];
  assert.equal(resumenAccesosModulos(accesos, false), "Comprobando módulos");
  assert.equal(resumenAccesosModulos([{ disponible: true, estado: "disponible" }], true), "Comprobando módulos");
  assert.equal(resumenAccesosModulos([{ disponible: true, estado: "disponible" }], false), "1 módulo disponible");
  assert.equal(resumenAccesosModulos([{ disponible: false, estado: "denegado" }], false), "Sin módulos disponibles");
  assert.doesNotMatch(resumenAccesosModulos([], false), /fase inicial/iu);
});

const directorio = new URL("./", import.meta.url);
const [html, estilos, codigoMenu, codigoPortal, codigoEventos] = await Promise.all([
  readFile(new URL("index.html", directorio), "utf8"),
  readFile(new URL("portal-menu-bolsa.css", directorio), "utf8"),
  readFile(new URL("portal-menu-bolsa.js", directorio), "utf8"),
  readFile(new URL("portal.js", directorio), "utf8"),
  readFile(new URL("portal-eventos.js", directorio), "utf8"),
]);

function crearControl({ categoria, grupo = false, submenu = "" }) {
  const atributos = new Map();
  if (grupo) {
    atributos.set("aria-controls", submenu);
    atributos.set("aria-expanded", "false");
  }
  return {
    dataset: {
      categoriaBolsa: categoria,
      ...(grupo ? { grupoBolsa: categoria } : {}),
    },
    getAttribute(nombre) { return atributos.get(nombre) ?? null; },
    setAttribute(nombre, valor) { atributos.set(nombre, String(valor)); },
  };
}

function crearRaizFalsa() {
  const controles = [
    crearControl({ categoria: "bolsas-candidatos", grupo: true, submenu: "submenu-bolsas-candidatos" }),
    crearControl({ categoria: "reglas", grupo: true, submenu: "submenu-reglas" }),
    crearControl({ categoria: "auditoria", grupo: true, submenu: "submenu-auditoria" }),
    crearControl({ categoria: "resumen" }),
  ];
  const submenus = new Map([
    ["submenu-bolsas-candidatos", { hidden: true }],
    ["submenu-reglas", { hidden: true }],
    ["submenu-auditoria", { hidden: true }],
  ]);
  const eventos = new Map();
  return {
    controles,
    submenus,
    eventos,
    contains(control) { return controles.includes(control); },
    querySelector(selector) {
      const resultado = selector.match(/^\[id="([a-z0-9-]+)"\]$/);
      return resultado ? submenus.get(resultado[1]) || null : null;
    },
    querySelectorAll(selector) {
      if (selector === "[data-categoria-bolsa]") return controles;
      if (selector === "[data-grupo-bolsa]") return controles.filter((control) => control.dataset.grupoBolsa);
      return [];
    },
    addEventListener(tipo, manejador) { eventos.set(tipo, manejador); },
    removeEventListener(tipo, manejador) {
      if (eventos.get(tipo) === manejador) eventos.delete(tipo);
    },
  };
}

test("las diez categorías reproducen la jerarquía funcional facilitada por RRHH", () => {
  assert.equal(Object.keys(CATEGORIAS_MENU_BOLSA).length, 10);
  assert.deepEqual(
    [...html.matchAll(/class="numero-menu [^"]+" aria-hidden="true">(\d+)<\/span>/g)].map((coincidencia) => Number(coincidencia[1])),
    [1, 2, 3, 4, 5, 6, 7, 8, 9, 10],
  );
  for (const texto of [
    "Bolsas y candidatos", "Llamamientos automáticos",
    "Contratos, ceses y reincorporaciones", "Reglas y baremación",
    "Consulta de candidatos", "Cuadro de mando",
    "Estadísticas", "Documentos y firma",
    "Correo y mensajería", "Auditoría y control",
  ]) assert.match(html, new RegExp(`<span>${texto}</span>`));
  // Rótulos de negocio, sin referencias internas ni dudas pendientes en el menú.
  assert.doesNotMatch(html, /\bB(5|6|7|12)\b|dudas? 1\d|relay/u);
});

test("las entradas con recorrido real describen su alcance concreto", () => {
  for (const texto of [
    "Se inician desde cada bolsa · orden de llamamiento pendiente de RRHH",
    "Fuente de contratos sin configurar · altas, ceses y reincorporaciones pendientes",
    "Borradores PDF en Contratación temporal · portafirmas pendiente",
    "Correo desde cada bolsa · buzón corporativo y SMS pendientes",
  ]) assert.ok(html.includes(texto), texto);
  assert.doesNotMatch(html, /Pendiente: sin servicio autorizado/u);
  for (const categoria of ["llamamientos", "contratos", "documentos", "comunicaciones"]) {
    const entrada = html.match(new RegExp(`<button[^>]*data-categoria-bolsa="${categoria}"[^>]*>`, "u"))?.[0] || "";
    assert.doesNotMatch(entrada, /disabled|aria-disabled/u);
  }
  assert.match(html, /data-categoria-bolsa="contratos" data-vista="contratos"/u);
  assert.match(html, /data-categoria-bolsa="documentos" data-vista="contratacion-temporal"/u);
});

test("el menú agrupa solo las rutas de gestión con superficie conservada", () => {
  const vistas = [
    "resumen", "elaboracion", "convocatorias", "solicitudes", "meritos", "baremacion",
    "alegaciones", "importacion", "llamamientos", "contratos", "documentos",
    "comunicaciones", "estadisticas", "auditoria", "configuracion", "reglas", "consulta",
  ];
  const vistasMapeadas = Object.values(CATEGORIAS_MENU_BOLSA).flat();
  const fragmentoMenu = html.match(/<nav class="navegacion-bolsa"[\s\S]+?<\/nav>/)?.[0] || "";
  const vistasEnPlantilla = [...fragmentoMenu.matchAll(/data-vista="([a-z-]+)"/g)].map((item) => item[1]);
  assert.equal(vistasMapeadas.length, 17);
  assert.equal(new Set(vistasMapeadas).size, 17, "cada vista debe pertenecer a una sola categoría");
  assert.deepEqual([...vistasMapeadas].sort(), [...vistas].sort());
  assert.equal(vistasEnPlantilla.length, 17);
  assert.deepEqual([...vistasEnPlantilla].filter((vista) => vistas.includes(vista)).sort(), [
    ...vistas.filter((vista) => !["documentos", "comunicaciones"].includes(vista)),
    "llamamientos",
  ].sort());
  assert.equal(vistasEnPlantilla.filter((vista) => vista === "contratacion-temporal").length, 1);
  assert.equal(vistasEnPlantilla.filter((vista) => vista === "llamamientos").length, 2);
  assert.equal(categoriaDeVistaBolsa("convocatorias"), "bolsas-candidatos");
  assert.equal(categoriaDeVistaBolsa("baremacion"), "reglas");
  assert.equal(categoriaDeVistaBolsa("configuracion"), "auditoria");
  assert.equal(categoriaDeVistaBolsa("cronos"), "");
  for (const vista of ["seleccion-inscripciones", "seleccion-pruebas", "seleccion-comunicaciones"]) {
    assert.equal(categoriaDeVistaBolsa(vista), "");
    assert.equal(vistaBolsaPendienteNoCompuesta(vista), false);
    assert.equal(VISTAS_INTERNAS_BOLSA.includes(vista), false);
    assert.equal(vistasEnPlantilla.includes(vista), false);
  }
});

test("B5 completa las dieciocho vistas internas sin duplicar su enlace", () => {
  const vistasMenu = Object.values(CATEGORIAS_MENU_BOLSA).flat();
  assert.equal(vistasMenu.length, 17);
  assert.equal(VISTAS_INTERNAS_BOLSA.length, 18);
  assert.equal(VISTAS_INTERNAS_BOLSA.filter((vista) => vista === VISTA_CANDIDATOS_BOLSA).length, 1);
  assert.equal(categoriaDeVistaBolsa(VISTA_CANDIDATOS_BOLSA), "bolsas-candidatos");
  assert.doesNotMatch(html.match(/<nav class="navegacion-bolsa"[\s\S]+?<\/nav>/)?.[0] || "", /data-vista="bolsa-candidatos"/);
});

test("la subvista activa se anuncia y B5 conserva abierto su grupo", () => {
  const atributos = new Map([["data-vista", "baremacion"]]);
  const enlace = {
    setAttribute(nombre, valor) { atributos.set(nombre, String(valor)); },
    removeAttribute(nombre) { atributos.delete(nombre); },
    getAttribute(nombre) { return atributos.get(nombre) || null; },
  };
  const raiz = { querySelectorAll(selector) { return selector === "[data-vista]" ? [enlace] : []; } };
  sincronizarMenuBolsa(raiz, "baremacion");
  assert.equal(enlace.getAttribute("aria-current"), "page");
  sincronizarMenuBolsa(raiz, VISTA_CANDIDATOS_BOLSA);
  assert.equal(enlace.getAttribute("aria-current"), null);
});

test("las entradas del menú llevan solo a recorridos disponibles y explican sus límites", () => {
  const entradas = ["llamamientos", "contratos", "documentos", "comunicaciones"];
  const fragmentoMenu = html.match(/<nav class="navegacion-bolsa"[\s\S]+?<\/nav>/)?.[0] || "";
  for (const categoria of entradas) {
    const boton = fragmentoMenu.match(new RegExp(`<button[^>]*data-categoria-bolsa="${categoria}"[^>]*>[\\s\\S]*?<\\/button>`))?.[0] || "";
    assert.doesNotMatch(boton, /categoria-menu-pendiente|\sdisabled(?:\s|=|>)|aria-disabled="true"/);
    assert.match(boton, /aria-describedby="estado-menu-[a-z]+"/);
  }
  assert.equal(vistaBolsaPendienteNoCompuesta("llamamientos"), true);
  assert.equal(vistaBolsaPendienteNoCompuesta("contratos"), true);
  assert.equal(vistaBolsaPendienteNoCompuesta("resumen"), false);
  assert.equal(vistaBolsaPendienteNoCompuesta("desconocida"), false);
  assert.match(codigoEventos, /if \(botonVista && !botonVista\.disabled\)/);
  assert.match(codigoEventos, /button:not\(:disabled\)/);
});

test("el grupo de la vista activa se abre y los demás quedan compactos", () => {
  const raiz = crearRaizFalsa();
  sincronizarMenuBolsa(raiz, "baremacion");
  const [bolsas, reglas, auditoria, resumen] = raiz.controles;
  assert.equal(reglas.dataset.categoriaActiva, "true");
  assert.equal(reglas.getAttribute("aria-expanded"), "true");
  assert.equal(raiz.submenus.get("submenu-reglas").hidden, false);
  assert.equal(bolsas.getAttribute("aria-expanded"), "false");
  assert.equal(auditoria.getAttribute("aria-expanded"), "false");
  assert.equal(resumen.dataset.categoriaActiva, undefined);

  assert.equal(alternarGrupoBolsa(raiz, reglas), true);
  assert.equal(reglas.getAttribute("aria-expanded"), "false");
  assert.equal(raiz.submenus.get("submenu-reglas").hidden, true);
});

test("el acordeón usa botones nativos, aria-expanded y delegación desmontable", () => {
  const raiz = crearRaizFalsa();
  const retirar = instalarMenuBolsa(raiz);
  const grupo = raiz.controles[0];
  let prevenido = false;
  raiz.eventos.get("click")({
    target: { closest: () => grupo },
    preventDefault() { prevenido = true; },
  });
  assert.equal(prevenido, true);
  assert.equal(grupo.getAttribute("aria-expanded"), "true");
  retirar();
  assert.equal(raiz.eventos.has("click"), false);
  assert.match(html, /aria-controls="submenu-bolsas-candidatos"/);
  assert.match(html, /id="submenu-bolsas-candidatos" role="group"/);
});

test("el menú conserva mínimo privilegio, adaptación y ausencia de estado ambiental", () => {
  assert.match(codigoPortal, /sincronizarMenuBolsa\(porId\("navegacion-bolsa"\), estado\.vista\)/);
  assert.match(codigoPortal, /instalarMenuBolsa\(porId\("navegacion-bolsa"\)\)/);
  assert.doesNotMatch(`${codigoMenu}\n${html}`, /app\.js|localStorage|sessionStorage|document\.cookie/);
  assert.doesNotMatch(codigoMenu, /textContent|innerText|dataset\.vista|navegar\(|fetch\(|location\.|history\./);
  assert.match(codigoMenu, /closest\?\.\("\[data-grupo-bolsa\]"\)/);
  assert.match(codigoPortal, /function vistaPermitida\(vista\)/);
  assert.match(codigoPortal, /if \(vista\.startsWith\("seleccion-"\)\) return false/);
  assert.match(estilos, /@media \(max-width: 1040px\)/);
  assert.match(estilos, /@media \(max-width: 780px\)/);
  assert.doesNotMatch(estilos, /#[0-9a-f]{3,8}\b|rgba?\(/i);
  assert.match(estilos, /@media \(forced-colors: active\)/);
  assert.match(estilos, /prefers-reduced-motion/);
  assert.match(estilos, /\.enlace-submenu:focus-visible/);
  assert.match(estilos, /overflow-wrap:\s*anywhere/);
});

test("accesoBolsaEfectivo abre el cuadro cuando hay bolsas reales aunque los borradores no estén disponibles", async () => {
  const { accesoBolsaEfectivo } = await import("./portal-menu-bolsa.js");
  const denegado = Object.freeze({ disponible: false, vista: "", estado: "error", etiqueta: "x" });
  const conBolsas = { carga: "listo", datos: { bolsas: [{ bolsa_ref: "bolsa:administrativo:2026-09-17" }] }, error: "" };
  assert.deepEqual(accesoBolsaEfectivo(denegado, conBolsas), { disponible: true, vista: "resumen", estado: "disponible", etiqueta: "Cuadro de bolsas" });
  // Un cuadro vacío responde con su contrato: el módulo está compuesto y autorizado.
  assert.equal(accesoBolsaEfectivo(denegado, { carga: "listo", datos: { bolsas: [] }, error: "" }).disponible, true);
  assert.equal(accesoBolsaEfectivo(denegado, { carga: "listo", datos: {}, error: "" }), denegado);
  assert.equal(accesoBolsaEfectivo(denegado, { carga: "cargando", datos: null, error: "" }), denegado);
  assert.equal(accesoBolsaEfectivo(denegado, { carga: "denegado", datos: null, error: "x" }), denegado);
  assert.equal(accesoBolsaEfectivo(denegado, { carga: "error", datos: null, error: "x" }), denegado);
  assert.equal(accesoBolsaEfectivo(denegado, undefined), denegado);
  const borradores = Object.freeze({ disponible: true, vista: "elaboracion", estado: "disponible", etiqueta: "Borradores disponibles" });
  assert.equal(accesoBolsaEfectivo(borradores, conBolsas), borradores);
});

// Regresión del 23/09/2026: las descripciones largas pintadas como etiqueta
// visible se montaban sobre el texto del menú lateral y lo partían palabra a
// palabra. La etiqueta visible es corta; la descripción completa queda para el
// lector de pantalla mediante aria-describedby.
test("las etiquetas visibles del menú son cortas y la descripción completa es accesible", () => {
  for (const categoria of ["llamamientos", "contratos", "documentos", "comunicaciones"]) {
    const visible = html.match(new RegExp(`<span class="etiqueta-menu" aria-hidden="true"[^>]*>([^<]*)</span><span class="solo-lectura" id="estado-menu-${categoria}"[^>]*>`, "u"));
    assert.ok(visible, `falta la etiqueta corta de ${categoria}`);
    assert.ok(visible[1].length <= 20, `etiqueta visible demasiado larga en ${categoria}: ${visible[1]}`);
  }
});

// Modelo mínimo del menú real de index.html: categorías, grupos y submenús.
function menuDesdeHTML() {
  const fragmento = html.match(/<nav class="navegacion-bolsa"[\s\S]+?<\/nav>/)?.[0] || "";
  const coincide = (elemento, selector) => {
    if (selector === ".submenu-bolsa [data-vista]") {
      return elemento.atributos["data-vista"] !== undefined && elemento.padre?.clase === "submenu-bolsa";
    }
    return selector.startsWith(".") && elemento.clase === selector.slice(1);
  };
  const nodo = (atributos, padre = null, clase = "") => {
    const propio = {
      hidden: false, dataset: {}, atributos, padre, hijos: [], clase, textContent: "",
      getAttribute: (nombre) => atributos[nombre] ?? null,
      closest: (selector) => {
        for (let actual = propio; actual; actual = actual.padre) if (actual.clase === selector.slice(1)) return actual;
        return null;
      },
      querySelector: (selector) => propio.querySelectorAll(selector)[0] || null,
      querySelectorAll: (selector) => {
        const salida = [];
        const visitar = (hijo) => { if (coincide(hijo, selector)) salida.push(hijo); hijo.hijos.forEach(visitar); };
        propio.hijos.forEach(visitar);
        return salida;
      },
    };
    padre?.hijos.push(propio);
    return propio;
  };
  const atributosDe = (etiqueta) => Object.fromEntries([...etiqueta.matchAll(/([a-z-]+)="([^"]*)"/gu)]
    .map(([, clave, valor]) => [clave, valor]));
  const raiz = nodo({});
  // Cada grupo es un bloque (su categoría y su submenú); cada categoría suelta, otro.
  const bloques = fragmento.match(/<div class="grupo-menu-bolsa">[\s\S]*?<\/div>\s*<\/div>|<button type="button" class="enlace-lateral categoria-menu-bolsa"[^>]*>/gu) || [];
  for (const bloque of bloques) {
    const categoria = bloque.match(/<button type="button" class="enlace-lateral categoria-menu-bolsa"[^>]*>/u)?.[0];
    if (!categoria) continue;
    const agrupado = bloque.startsWith('<div class="grupo-menu-bolsa">');
    const padre = agrupado ? nodo({}, raiz, "grupo-menu-bolsa") : raiz;
    const atributos = atributosDe(categoria);
    const control = nodo(atributos, padre, "categoria-menu-bolsa");
    control.dataset = {
      categoriaBolsa: atributos["data-categoria-bolsa"],
      ...(atributos["data-grupo-bolsa"] ? { grupoBolsa: atributos["data-grupo-bolsa"] } : {}),
    };
    nodo({}, control, "numero-menu");
    if (!agrupado) continue;
    const submenu = nodo({}, padre, "submenu-bolsa");
    for (const [etiqueta] of bloque.matchAll(/<button type="button" class="enlace-lateral enlace-submenu"[^>]*>/gu)) {
      nodo(atributosDe(etiqueta), submenu);
    }
  }
  return raiz;
}

function categoriasVisibles(raiz) {
  return raiz.querySelectorAll(".categoria-menu-bolsa")
    .filter((control) => !control.hidden && !(control.padre?.clase === "grupo-menu-bolsa" && control.padre.hidden))
    .map((control) => control.dataset.categoriaBolsa);
}

test("sin panel interno ni borradores, el menú de Bolsa solo ofrece lo que tiene servicio", async () => {
  const { aplicarDisponibilidadMenuBolsa } = await import("./portal-menu-bolsa.js");
  const raiz = menuDesdeHTML();
  const indicadores = aplicarDisponibilidadMenuBolsa(raiz, { panelInterno: false, borradores: null, contratacionTemporal: true });
  assert.deepEqual(categoriasVisibles(raiz), ["llamamientos", "resumen", "estadisticas", "documentos"]);
  assert.equal(indicadores.length, 4, "se renumeran solo las categorías visibles");
  // Grupos enteros sin servicio (convocatorias…, reglas, auditoría) quedan ocultos.
  assert.equal(raiz.querySelectorAll(".grupo-menu-bolsa").length, 3);
  assert.equal(raiz.querySelectorAll(".grupo-menu-bolsa").every((grupo) => grupo.hidden), true);
  // Sin contratación temporal, «Documentos y firma» tampoco se ofrece.
  aplicarDisponibilidadMenuBolsa(raiz, { panelInterno: false, borradores: false, contratacionTemporal: false });
  assert.deepEqual(categoriasVisibles(raiz), ["llamamientos", "resumen", "estadisticas"]);
});

test("cada capacidad real vuelve a ofrecer sus entradas del menú de Bolsa", async () => {
  const { aplicarDisponibilidadMenuBolsa } = await import("./portal-menu-bolsa.js");
  const raiz = menuDesdeHTML();
  aplicarDisponibilidadMenuBolsa(raiz, { panelInterno: false, borradores: true, contratacionTemporal: true });
  const grupoBolsas = raiz.querySelectorAll(".grupo-menu-bolsa")[0];
  assert.equal(grupoBolsas.hidden, false);
  assert.deepEqual(grupoBolsas.querySelectorAll(".submenu-bolsa [data-vista]").filter((control) => !control.hidden)
    .map((control) => control.getAttribute("data-vista")), ["elaboracion"]);
  const indicadores = aplicarDisponibilidadMenuBolsa(raiz, { panelInterno: true, borradores: true, contratacionTemporal: true });
  assert.equal(indicadores.length, 10);
  assert.deepEqual(categoriasVisibles(raiz), Object.keys(CATEGORIAS_MENU_BOLSA));
});

test("la navegación directa a una vista de Bolsa sin servicio no se permite", async () => {
  const { vistaBolsaNavegable, vistaBolsaOfrecida } = await import("./portal-menu-bolsa.js");
  const sinServicio = { panelInterno: false, borradores: false, contratacionTemporal: false };
  for (const vista of ["convocatorias", "solicitudes", "meritos", "alegaciones", "importacion", "contratos",
    "reglas", "baremacion", "consulta", "documentos", "comunicaciones", "auditoria", "configuracion", "elaboracion"]) {
    assert.equal(vistaBolsaNavegable(vista, sinServicio), false, vista);
  }
  for (const vista of ["resumen", "estadisticas", "llamamientos", VISTA_CANDIDATOS_BOLSA]) {
    assert.equal(vistaBolsaNavegable(vista, sinServicio), true, vista);
  }
  // Elaboración sin comprobar todavía: se deja abrir para que compruebe su API,
  // pero el menú no la ofrece hasta que conste disponible.
  assert.equal(vistaBolsaNavegable("elaboracion", { borradores: null }), true);
  assert.equal(vistaBolsaOfrecida("elaboracion", { borradores: null }), false);
  assert.equal(vistaBolsaOfrecida("desconocida", { panelInterno: true }), false);
});

test("el llamamiento con bolsa elegida no pinta el aviso del panel interno", () => {
  const inicio = codigoPortal.indexOf("function montarVistaBolsa(");
  const fin = codigoPortal.indexOf("function renderizarLlamamientoSinBolsa(", inicio);
  const montaje = codigoPortal.slice(inicio, fin);
  assert.ok(inicio > 0 && fin > inicio);
  assert.doesNotMatch(montaje, /renderizarFuenteNoDisponible\(\)\}\$\{superficieBorradorLlamamiento/u);
  assert.match(montaje, /encabezadoVista\("", tituloDeVista\(vista\)\[1\], ""\)\}\$\{superficieBorradorLlamamiento\.renderizar\(\)\}/u);
  assert.match(codigoPortal, /if \(moduloDeVistaPortal\(vista\) === "bolsa"\) return vistaBolsaNavegable\(vista, capacidadesBolsa\(\)\)/u);
});
