import assert from "node:assert/strict";
import { readFile } from "node:fs/promises";
import test from "node:test";
import {
  CATEGORIAS_MENU_BOLSA,
  alternarGrupoBolsa,
  categoriaDeVistaBolsa,
  instalarMenuBolsa,
  sincronizarMenuBolsa,
} from "./portal-menu-bolsa.js";

const directorio = new URL("./", import.meta.url);
const [html, estilos, codigoMenu, codigoPortal] = await Promise.all([
  readFile(new URL("index.html", directorio), "utf8"),
  readFile(new URL("portal-menu-bolsa.css", directorio), "utf8"),
  readFile(new URL("portal-menu-bolsa.js", directorio), "utf8"),
  readFile(new URL("portal.js", directorio), "utf8"),
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
    "Gestión de bolsas y candidatos", "Llamamientos automáticos según bases y Reglamento",
    "Contratos, ceses y reincorporaciones", "Motor de reglas configurable",
    "Portal de consulta para candidatos", "Cuadro de mando para dirección",
    "Estadísticas y explotación de datos", "Generación y firma de documentos",
    "Correo y mensajería", "Auditoría, trazabilidad y control",
  ]) assert.match(html, new RegExp(texto));
});

test("ninguna ruta de la gestión actual desaparece al agrupar el menú", () => {
  const vistas = [
    "resumen", "elaboracion", "convocatorias", "solicitudes", "meritos", "baremacion",
    "alegaciones", "importacion", "llamamientos", "contratos", "documentos",
    "comunicaciones", "estadisticas", "auditoria", "configuracion", "reglas", "consulta",
  ];
  const vistasMapeadas = Object.values(CATEGORIAS_MENU_BOLSA).flat();
  const fragmentoMenu = html.match(/<nav class="navegacion-bolsa"[\s\S]+?<\/nav>/)?.[0] || "";
  const vistasEnPlantilla = [...fragmentoMenu.matchAll(/data-vista="([a-z]+)"/g)].map((item) => item[1]);
  assert.equal(vistasMapeadas.length, 17);
  assert.equal(new Set(vistasMapeadas).size, 17, "cada vista debe pertenecer a una sola categoría");
  assert.deepEqual([...vistasMapeadas].sort(), [...vistas].sort());
  assert.equal(vistasEnPlantilla.length, 17);
  assert.equal(new Set(vistasEnPlantilla).size, 17, "la plantilla no debe duplicar rutas entre categorías");
  assert.deepEqual([...vistasEnPlantilla].sort(), [...vistas].sort());
  for (const vista of vistas) assert.match(html, new RegExp(`data-vista="${vista}"`));
  assert.equal(categoriaDeVistaBolsa("convocatorias"), "bolsas-candidatos");
  assert.equal(categoriaDeVistaBolsa("baremacion"), "reglas");
  assert.equal(categoriaDeVistaBolsa("configuracion"), "auditoria");
  assert.equal(categoriaDeVistaBolsa("cronos"), "");
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
  assert.match(codigoPortal, /control\.disabled = true/);
  assert.match(estilos, /@media \(max-width: 1040px\)/);
  assert.match(estilos, /@media \(max-width: 780px\)/);
  assert.match(estilos, /\.categoria-menu-bolsa \.numero-menu\s*\{[^}]*border-radius:\s*50%/);
  assert.match(estilos, /@media \(forced-colors: active\)/);
  assert.match(estilos, /prefers-reduced-motion/);
});
