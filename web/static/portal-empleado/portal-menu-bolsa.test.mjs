import assert from "node:assert/strict";
import { readFile } from "node:fs/promises";
import test from "node:test";
import {
  CATEGORIAS_MENU_BOLSA,
  VISTA_CANDIDATOS_BOLSA,
  VISTAS_INTERNAS_BOLSA,
  alternarGrupoBolsa,
  aplicarDisponibilidadMenuBolsa,
  categoriaDeVistaBolsa,
  instalarMenuBolsa,
  resumenAccesosModulos,
  sincronizarMenuBolsa,
  vistaBolsaPendienteNoCompuesta,
  vistaBolsaOfrecida,
} from "./portal-menu-bolsa.js?v=VEC_WEB_VERSION";

test("Auditoría RRHH se abre sin partir de una ficha", () => {
  assert.equal(vistaBolsaOfrecida("auditoria", { panelInterno: true }), false);
  assert.equal(vistaBolsaOfrecida("auditoria", { auditoriaConsulta: false }), false);
  assert.equal(vistaBolsaOfrecida("auditoria", { auditoriaConsulta: true }), true);
});

test("Reglas de cese solo se ofrecen tras su consulta V3 positiva", () => {
  assert.equal(vistaBolsaOfrecida("reglas", { panelInterno: true }), false);
  assert.equal(vistaBolsaOfrecida("reglas", { politicaCese: true }), true);
});

test("reglas de situación se ofrecen con Bolsa consultable y conservan la vista de cese", () => {
  assert.equal(vistaBolsaOfrecida("reglas-situacion", { bolsasConsultables: false }), false);
  assert.equal(vistaBolsaOfrecida("reglas-situacion", { bolsasConsultables: true }), true);
  assert.ok(VISTAS_INTERNAS_BOLSA.includes("reglas"));
  assert.ok(VISTAS_INTERNAS_BOLSA.includes("reglas-situacion"));
  assert.equal(categoriaDeVistaBolsa("reglas-situacion"), "reglas");
});

test("P-WEB-14 no anuncia un total mientras Bolsa sigue comprobando", () => {
  const accesos = [{ disponible: true, estado: "disponible" }, { disponible: false, estado: "cargando" }];
  assert.equal(resumenAccesosModulos(accesos, false), "Comprobando áreas");
  assert.equal(resumenAccesosModulos([{ disponible: true, estado: "disponible" }], true), "Comprobando áreas");
  assert.equal(resumenAccesosModulos([{ disponible: true, estado: "disponible" }], false), "1 área disponible");
  assert.equal(resumenAccesosModulos([{ disponible: false, estado: "denegado" }], false), "Sin áreas disponibles");
  assert.doesNotMatch(resumenAccesosModulos([], false), /fase inicial/iu);
  const dos = [{ disponible: true, estado: "disponible" }, { disponible: true, estado: "disponible" }];
  assert.equal(resumenAccesosModulos(dos, false), "2 áreas disponibles");
  // Los textos salen del catálogo i18n, no de literales del módulo.
  const claves = [];
  resumenAccesosModulos(dos, false, (clave, variables) => { claves.push([clave, variables]); return clave; });
  resumenAccesosModulos([], false, (clave) => { claves.push([clave]); return clave; });
  resumenAccesosModulos([], true, (clave) => { claves.push([clave]); return clave; });
  assert.deepEqual(claves, [["resumen_modulos_varios", { cantidad: "2" }], ["resumen_modulos_ninguno"], ["resumen_modulos_comprobando"]]);
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
    "Estadísticas",
    "Correo y mensajería", "Auditoría y control",
  ]) assert.match(html, new RegExp(`<span data-i18n-portal="txt_[a-z0-9_]+">${texto}</span>`));
  assert.match(html, /data-i18n-portal="bolsa_documentos_titulo">Documentos de Bolsa<\/span>/u);
  // Rótulos de negocio, sin referencias internas ni dudas pendientes en el menú.
  assert.doesNotMatch(html, /\bB(5|6|7|12)\b|dudas? 1\d|relay/u);
});

test("Documentos de Bolsa abre su consulta propia sin abrir Contratación", () => {
  const menu = html.match(/<nav class="navegacion-bolsa"[\s\S]+?<\/nav>/)?.[0] || "";
  assert.ok(menu.length > 0);
  assert.match(menu, /data-categoria-bolsa="documentos" data-vista="documentos"/u);
  assert.doesNotMatch(menu, /data-categoria-bolsa="documentos"[^>]*disabled/u);
  assert.doesNotMatch(menu, /id="bolsa-documentos-pendientes"/u);
  assert.doesNotMatch(menu, /data-categoria-bolsa="documentos" data-vista="contratacion-temporal"/u);
  for (const categoria of ["llamamientos", "contratos", "comunicaciones"]) {
    const entrada = html.match(new RegExp(`<button[^>]*data-categoria-bolsa="${categoria}"[^>]*>`, "u"))?.[0] || "";
    assert.doesNotMatch(entrada, /disabled|aria-disabled/u);
  }
  assert.match(html, /data-categoria-bolsa="contratos" data-vista="contratos"/u);
});

test("el menú agrupa solo las rutas de gestión con superficie conservada", () => {
  const vistas = [
    "resumen", "elaboracion", "convocatorias", "publicacion-bases", "solicitudes", "meritos", "baremacion",
    "alegaciones", "importacion", "llamamientos", "contratos", "documentos",
    "comunicaciones", "estadisticas", "auditoria", "configuracion", "reglas", "reglas-situacion", "consulta",
  ];
  const vistasMapeadas = Object.values(CATEGORIAS_MENU_BOLSA).flat();
  const fragmentoMenu = html.match(/<nav class="navegacion-bolsa"[\s\S]+?<\/nav>/)?.[0] || "";
  const vistasEnPlantilla = [...fragmentoMenu.matchAll(/data-vista="([a-z-]+)"/g)].map((item) => item[1]);
  assert.equal(vistasMapeadas.length, 19);
  assert.equal(new Set(vistasMapeadas).size, 19, "cada vista debe pertenecer a una sola categoría");
  assert.deepEqual([...vistasMapeadas].sort(), [...vistas].sort());
  assert.equal(vistasEnPlantilla.length, 19);
  assert.deepEqual([...vistasEnPlantilla].filter((vista) => vistas.includes(vista)).sort(), [
    ...vistas,
  ].sort());
  assert.equal(vistasEnPlantilla.filter((vista) => vista === "contratacion-temporal").length, 0);
  assert.equal(vistasEnPlantilla.filter((vista) => vista === "llamamientos").length, 1);
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

test("B5 completa las veinte vistas internas sin duplicar su enlace", () => {
  const vistasMenu = Object.values(CATEGORIAS_MENU_BOLSA).flat();
  assert.equal(vistasMenu.length, 19);
  assert.equal(VISTAS_INTERNAS_BOLSA.length, 20);
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

test("las entradas del menú llevan solo a recorridos disponibles", () => {
  const entradas = ["llamamientos", "contratos", "comunicaciones"];
  const fragmentoMenu = html.match(/<nav class="navegacion-bolsa"[\s\S]+?<\/nav>/)?.[0] || "";
  for (const categoria of entradas) {
    const boton = fragmentoMenu.match(new RegExp(`<button[^>]*data-categoria-bolsa="${categoria}"[^>]*>[\\s\\S]*?<\\/button>`))?.[0] || "";
    assert.doesNotMatch(boton, /categoria-menu-pendiente|\sdisabled(?:\s|=|>)|aria-disabled="true"/);
    assert.doesNotMatch(boton, /aria-describedby|pendiente/iu);
  }
  assert.doesNotMatch(fragmentoMenu, /data-categoria-bolsa="documentos"[^>]*disabled/u);
  assert.equal(vistaBolsaPendienteNoCompuesta("llamamientos"), false);
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
  assert.match(codigoPortal, /sincronizarMenuBolsa\(porId\("navegacion-bolsa"\), estado\.vista === "bolsa-candidatos"/);
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
// palabra. Las etiquetas visibles que quedan son cortas.
test("las etiquetas visibles del menú son cortas", () => {
  const menu = html.match(/<nav class="navegacion-bolsa"[\s\S]+?<\/nav>/)?.[0] || "";
  const etiquetas = [...menu.matchAll(/<span class="etiqueta-menu" aria-hidden="true"[^>]*>([^<]*)<\/span>/gu)].map((m) => m[1]);
  assert.deepEqual(etiquetas, ["Consulta", "Correo"]);
  for (const etiqueta of etiquetas) assert.ok(etiqueta.length <= 20, etiqueta);
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
    for (const [etiqueta] of bloque.matchAll(/<(?:button|a)[^>]*class="enlace-lateral enlace-submenu"[^>]*>/gu)) {
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
  const indicadores = aplicarDisponibilidadMenuBolsa(raiz, { bolsasConsultables: true, panelInterno: false, borradores: false, contratacionTemporal: true });
  assert.deepEqual(categoriasVisibles(raiz), ["bolsas-candidatos", "llamamientos", "reglas", "resumen", "estadisticas", "documentos", "comunicaciones", "auditoria"]);
  assert.equal(indicadores.length, 8, "se renumeran solo las categorías visibles");
  // Los grupos Reglas y Auditoría aparecen: ranking y configuración usan la lectura RRHH.
  assert.equal(raiz.querySelectorAll(".grupo-menu-bolsa").length, 3);
  assert.equal(raiz.querySelectorAll(".grupo-menu-bolsa").filter((grupo) => !grupo.hidden).length, 3);
  // El aviso de Documentos de Bolsa no depende de la capacidad de CT.
  aplicarDisponibilidadMenuBolsa(raiz, { bolsasConsultables: true, panelInterno: false, borradores: false, contratacionTemporal: false });
  assert.deepEqual(categoriasVisibles(raiz), ["bolsas-candidatos", "llamamientos", "reglas", "resumen", "estadisticas", "documentos", "comunicaciones", "auditoria"]);
});

test("el menú espera la lectura positiva de bolsas sin impedir el estado de una URL directa", async () => {
  const { aplicarDisponibilidadMenuBolsa, vistaBolsaNavegable } = await import("./portal-menu-bolsa.js");
  const raiz = menuDesdeHTML();
  aplicarDisponibilidadMenuBolsa(raiz, { bolsasConsultables: false, borradores: false, contratacionTemporal: false });
  assert.deepEqual(categoriasVisibles(raiz), []);
  for (const vista of ["resumen", "estadisticas", "llamamientos"]) {
    assert.equal(vistaBolsaNavegable(vista, { bolsasConsultables: false }), true, vista);
  }
});

// E10/P1: RRHH sin panel interno no perdía «Elaboración y borradores» aunque su
// API respondiera: mientras se comprueba y cuando responde, la entrada se ofrece.
// Recorrido en Chrome del 25/09/2026 (fallo 2): la sonda de la API de
// borradores daba 404 en cada carga contra un servidor que no la sirve. Ya no se
// sondea al cargar: Elaboración solo se ofrece cuando consta disponible.
test("Elaboración solo se ofrece a RRHH cuando su API consta disponible", async () => {
  const { aplicarDisponibilidadMenuBolsa } = await import("./portal-menu-bolsa.js");
  const raiz = menuDesdeHTML();
  const elaboracion = () => raiz.querySelectorAll(".submenu-bolsa [data-vista]")
    .find((control) => control.getAttribute("data-vista") === "elaboracion");
  aplicarDisponibilidadMenuBolsa(raiz, { bolsasConsultables: true, panelInterno: false, borradores: true, contratacionTemporal: true });
  assert.equal(elaboracion().hidden, false);
  assert.deepEqual(categoriasVisibles(raiz), ["bolsas-candidatos", "llamamientos", "reglas", "resumen", "estadisticas", "documentos", "comunicaciones", "auditoria"]);
  for (const borradores of [null, false]) {
    aplicarDisponibilidadMenuBolsa(raiz, { bolsasConsultables: true, panelInterno: false, borradores, contratacionTemporal: true });
    assert.equal(elaboracion().hidden, true, String(borradores));
    assert.deepEqual(categoriasVisibles(raiz), ["bolsas-candidatos", "llamamientos", "reglas", "resumen", "estadisticas", "documentos", "comunicaciones", "auditoria"]);
  }
});

test("el panel antiguo no anuncia recorridos sin API compuesta", async () => {
  const { aplicarDisponibilidadMenuBolsa } = await import("./portal-menu-bolsa.js");
  const raiz = menuDesdeHTML();
  aplicarDisponibilidadMenuBolsa(raiz, { bolsasConsultables: true, panelInterno: false, borradores: true, contratacionTemporal: true });
  const grupoBolsas = raiz.querySelectorAll(".grupo-menu-bolsa")[0];
  assert.equal(grupoBolsas.hidden, false);
  assert.deepEqual(grupoBolsas.querySelectorAll(".submenu-bolsa [data-vista]").filter((control) => !control.hidden)
    .map((control) => control.getAttribute("data-vista")), ["elaboracion", "alegaciones"]);
  const indicadores = aplicarDisponibilidadMenuBolsa(raiz, { bolsasConsultables: true, panelInterno: true, borradores: true, contratacionTemporal: true });
  assert.equal(indicadores.length, 8);
  assert.deepEqual(categoriasVisibles(raiz), ["bolsas-candidatos", "llamamientos", "reglas", "resumen", "estadisticas", "documentos", "comunicaciones", "auditoria"]);
});

test("la navegación directa a una vista de Bolsa sin servicio no se permite", async () => {
  const { vistaBolsaNavegable, vistaBolsaOfrecida } = await import("./portal-menu-bolsa.js");
  const sinServicio = { bolsasConsultables: true, panelInterno: false, borradores: false, contratacionTemporal: false };
  for (const vista of ["contratos",
    "reglas", "consulta", "auditoria", "elaboracion"]) {
    assert.equal(vistaBolsaNavegable(vista, sinServicio), false, vista);
  }
  for (const vista of ["resumen", "estadisticas", "llamamientos", "baremacion", "configuracion", "convocatorias", "documentos", "solicitudes", "meritos", "alegaciones", VISTA_CANDIDATOS_BOLSA]) {
    assert.equal(vistaBolsaNavegable(vista, sinServicio), true, vista);
  }
  // Inscripciones: con el servicio apagado (o sin comprobar) no aparece en el
  // menú; su enlace directo abre y explica que no está disponible.
  assert.equal(vistaBolsaOfrecida("solicitudes", { panelInterno: true, bolsasConsultables: true }), false);
  assert.equal(vistaBolsaOfrecida("solicitudes", { inscripciones: false }), false);
  assert.equal(vistaBolsaOfrecida("solicitudes", { inscripciones: true }), true);
  assert.equal(vistaBolsaOfrecida("meritos", { inscripciones: false, bolsasConsultables: true }), false);
  assert.equal(vistaBolsaOfrecida("meritos", { inscripciones: true }), true);
  assert.equal(vistaBolsaOfrecida("convocatorias", { inscripciones: false, bolsasConsultables: true }), false);
  assert.equal(vistaBolsaOfrecida("convocatorias", { inscripciones: true }), true);
  // Elaboración sin comprobar: no se ofrece en el menú, pero su enlace directo
  // se deja abrir (abrirla es lo que comprueba su API); si consta que falta, no.
  assert.equal(vistaBolsaNavegable("elaboracion", { borradores: null }), true);
  assert.equal(vistaBolsaOfrecida("elaboracion", { borradores: null }), false);
  assert.equal(vistaBolsaOfrecida("elaboracion", { borradores: true }), true);
  assert.equal(vistaBolsaOfrecida("elaboracion", { borradores: false }), false);
  assert.equal(vistaBolsaOfrecida("desconocida", { panelInterno: true }), false);
});

test("el llamamiento conserva B7 y monta gestión de ofertas solo tras elegir bolsa vigente", () => {
  const inicio = codigoPortal.indexOf("function montarVistaBolsa(");
  const fin = codigoPortal.indexOf("function actualizarVistaBolsa(", inicio);
  const montaje = codigoPortal.slice(inicio, fin);
  assert.ok(inicio > 0 && fin > inicio);
  assert.doesNotMatch(montaje, /renderizarFuenteNoDisponible\(\)\}\$\{superficieBorradorLlamamiento/u);
  assert.match(montaje, /renderizarPantallaLlamamientos\(\{ estado, encabezadoVista, escaparHTML, presentador: presentadorPanelInterno \}\)/u);
  assert.match(montaje, /estado\.llamamientoDesdeMenu \? estado\.bolsaSeleccionada/u);
  assert.match(montaje, /superficieRRHHPlazos\.instalar\(document\)/u);
  assert.match(montaje, /superficieRRHHPlazos\.activar\(bolsaRef\)/u);
  assert.match(montaje, /superficieOfertasBolsa\.activar\(bolsaRef\)/u);
  assert.match(montaje, /superficieRRHHPlazos\.renderizar\(\)/u);
  assert.match(montaje, /superficieOfertasBolsa\.renderizar\(\)/u);
  assert.match(montaje, /data-gestion-ofertas-accion="volver"/u);
  assert.match(montaje, /data-gestion-ofertas-accion="abrir"/u);
  assert.match(montaje, /bolsasVigentesParaLlamamiento\(estado\.datosBolsas\)/u);
  assert.match(montaje, /data-gestion-ofertas-aviso/u);
  assert.match(codigoPortal, /function salidaGestionOfertasBloqueada\(\)/u);
  assert.match(codigoPortal, /if \(salidaGestionOfertasBloqueada\(\)\) \{/u);
  assert.match(codigoPortal, /history\.pushState\(\{ \.\.\.estadoAnterior, vecOfertas: \{ id: marcaHistorialGestionOfertas, lado: "gestion" \} \}, "", rutaVisibleGestionOfertasBolsa\)/u);
  assert.match(codigoPortal, /rutaOfertasSolicitadaRef = vista === "llamamientos"/u);
  assert.match(codigoPortal, /history\.go\(haciaAtras \|\| !marcaHistorialGestionOfertas \? 1 : -1\)/u);
  assert.match(montaje, /rutaOfertasSolicitadaRef \|\| new URLSearchParams\(window\.location\.search\)/u);
  assert.match(montaje, /asegurarHistorialGestionOfertas\(\)/u);
  assert.match(codigoPortal, /history\.replaceState\(\{ \.\.\.anterior, vecOfertas: \{ id: marcaHistorialGestionOfertas, lado: "origen" \} \}, "", rutaGestionOfertasBolsa\(\)\)/u);
  assert.match(codigoPortal, /history\.pushState\(\{ \.\.\.anterior, vecOfertas: \{ id: marcaHistorialGestionOfertas, lado: "gestion" \} \}, "", rutaVisibleGestionOfertasBolsa\)/u);
  assert.doesNotMatch(montaje, /superficieBorradorLlamamiento\.renderizar/u);
  assert.match(codigoPortal, /if \(moduloDeVistaPortal\(vista\) === "bolsa"\) return vistaBolsaNavegable\(vista, capacidadesBolsa\(\)\)/u);
});

test("Atrás durante un envío incierto vuelve a gestión sin sustituir la entrada del llamamiento", () => {
  const inicio = codigoPortal.indexOf("function compensarHistorialGestionOfertas(");
  const fin = codigoPortal.indexOf("\nfunction actualizarSalidaGestionOfertasBolsa()", inicio);
  assert.ok(inicio > 0 && fin > inicio);
  const compensador = codigoPortal.slice(inicio, fin);
  const origen = "/portal-empleado/#bolsa/llamamientos";
  const gestion = "/portal-empleado/?ofertas_bolsa_ref=bolsa%3A1#bolsa/llamamientos";
  const entradas = [
    { ruta: origen, state: { vecOfertas: { id: "intento-1", lado: "origen" } } },
    { ruta: gestion, state: { vecOfertas: { id: "intento-1", lado: "gestion" } } },
  ];
  let indice = 1;
  const ventana = { location: {} };
  const situar = () => {
    const url = new URL(entradas[indice].ruta, "https://vec.example");
    ventana.location = { pathname: url.pathname, search: url.search, hash: url.hash };
  };
  situar();
  const historial = { get state() { return entradas[indice].state; }, go(desplazamiento) { indice += desplazamiento; situar(); } };
  const crear = new Function("window", "history", "anunciarSalidaGestionOfertas", `
    let rutaVisibleGestionOfertasBolsa = ${JSON.stringify(gestion)};
    let marcaHistorialGestionOfertas = "intento-1";
    let compensandoHistorialGestionOfertas = false;
    ${compensador}
    return compensarHistorialGestionOfertas;
  `);
  const compensar = crear(ventana, historial, () => {});
  historial.go(-1); // Atrás mientras el POST no tiene resultado confirmado.
  compensar();
  compensar(); // popstate de la compensación: no vuelve a desplazarse.
  assert.equal(indice, 1);
  assert.deepEqual(entradas.map((entrada) => entrada.ruta), [origen, gestion]);
  historial.go(-1); // Tras confirmar el reintento, Atrás sigue llevando al llamamiento.
  assert.equal(ventana.location.hash, "#bolsa/llamamientos");
  assert.equal(ventana.location.search, "");
  historial.go(1); // Adelante recupera la subvista de gestión.
  assert.equal(ventana.location.search, "?ofertas_bolsa_ref=bolsa%3A1");
});

test("enlace directo crea A/B y Forward incierto vuelve a B sin destruir C", () => {
  const inicioAsegurar = codigoPortal.indexOf("function asegurarHistorialGestionOfertas() {");
  const finAsegurar = codigoPortal.indexOf("\nfunction cerrarGestionOfertasBolsa()", inicioAsegurar);
  const inicioCompensar = codigoPortal.indexOf("function compensarHistorialGestionOfertas(");
  const finCompensar = codigoPortal.indexOf("\nfunction actualizarSalidaGestionOfertasBolsa()", inicioCompensar);
  assert.ok(inicioAsegurar > 0 && finAsegurar > inicioAsegurar && inicioCompensar > 0 && finCompensar > inicioCompensar);
  const origen = "/portal-empleado/#bolsa/llamamientos";
  const gestion = "/portal-empleado/?ofertas_bolsa_ref=bolsa%3A1#bolsa/llamamientos";
  const destino = "/portal-empleado/#portal";
  const entradas = [{ ruta: gestion, state: null }];
  let indice = 0;
  const ventana = { location: {} };
  const situar = () => {
    const url = new URL(entradas[indice].ruta, "https://vec.example");
    ventana.location = { pathname: url.pathname, search: url.search, hash: url.hash };
  };
  situar();
  const historial = {
    get state() { return entradas[indice].state; },
    replaceState(state, _titulo, ruta) { entradas[indice] = { state, ruta }; situar(); },
    pushState(state, _titulo, ruta) { entradas.splice(indice + 1); entradas.push({ state, ruta }); indice++; situar(); },
    go(desplazamiento) { indice += desplazamiento; situar(); },
  };
  const crear = new Function("window", "history", "globalThis", "anunciarSalidaGestionOfertas", `
    let rutaVisibleGestionOfertasBolsa = ${JSON.stringify(gestion)};
    let marcaHistorialGestionOfertas = "";
    let compensandoHistorialGestionOfertas = false;
    function rutaGestionOfertasBolsa() { return ${JSON.stringify(origen)}; }
    ${codigoPortal.slice(inicioAsegurar, finAsegurar)}
    ${codigoPortal.slice(inicioCompensar, finCompensar)}
    return { asegurarHistorialGestionOfertas, compensarHistorialGestionOfertas };
  `);
  const control = crear(ventana, historial, { crypto: { randomUUID: () => "enlace-1" } }, () => {});
  control.asegurarHistorialGestionOfertas();
  assert.deepEqual(entradas.map((entrada) => entrada.ruta), [origen, gestion]);
  historial.pushState(null, "", destino);
  historial.go(-1); // Volver a B desde C conserva la marca.
  control.asegurarHistorialGestionOfertas();
  assert.equal(entradas.length, 3);
  historial.go(1); // Forward incierto intenta salir hacia C.
  control.compensarHistorialGestionOfertas();
  control.compensarHistorialGestionOfertas();
  assert.equal(indice, 1);
  assert.deepEqual(entradas.map((entrada) => entrada.ruta), [origen, gestion, destino]);
  historial.go(-1); // Tras resolver, Atrás llega a A.
  assert.equal(entradas[indice].ruta, origen);
  historial.go(1); // Adelante recupera B.
  assert.equal(entradas[indice].ruta, gestion);
});


test("los controles de rutas sin consumidor se retiran del DOM y se conserva la auditoría de la ficha", () => {
  const retirados = [];
  const raiz = { querySelectorAll(selector) {
    if (selector !== "[data-vista]") return [];
    return ["documentos", "contratos", "importacion", "auditoria"].map((vista) => ({
      getAttribute: () => vista, dataset: {}, remove: () => retirados.push(vista),
    }));
  } };
  aplicarDisponibilidadMenuBolsa(raiz, { auditoriaConsulta: true });
  assert.deepEqual(retirados, ["contratos"]);
});


test("las rutas pendientes conservan su vista al navegar y al pintar tras F5 sin habilitar operaciones", () => {
  const inicio = codigoPortal.indexOf("function vistaPermitida(");
  const fin = codigoPortal.indexOf("\n}\n", inicio);
  const decidir = new Function("vistaBolsaPendienteNoCompuesta", `${codigoPortal.slice(inicio, fin + 3)}; return vistaPermitida;`)(vistaBolsaPendienteNoCompuesta);
  for (const vista of ["contratos"]) {
    assert.equal(decidir(vista), true);
    assert.equal(vistaBolsaOfrecida(vista, { bolsasConsultables: true, panelInterno: true }), false);
  }
  const montar = codigoPortal.slice(codigoPortal.indexOf("function montarVistaBolsa("), codigoPortal.indexOf("function actualizarVistaBolsa("));
  assert.match(montar, /if \(vistaBolsaPendienteNoCompuesta\(vista\)\) \{\s*contenedor.innerHTML = renderizarFuenteNoDisponible\(\);\s*return;/u);
});
