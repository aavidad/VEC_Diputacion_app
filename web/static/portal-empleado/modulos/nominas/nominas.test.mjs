import assert from "node:assert/strict";
import test from "node:test";
import { readFile } from "node:fs/promises";
import { cargarTextos } from "../../../comun/textos.js";
import { TEXTOS_NOMINAS, crearTraductorNominas } from "./i18n.js";
import { montarVistaNominas } from "./vista.js";

test("Nóminas usa catálogos completos en los idiomas del portal", async () => {
  const [es, en] = await Promise.all(["es", "en"].map((idioma) => cargarTextos("nominas", { idioma, avisar: assert.fail })));
  assert.deepEqual(Object.keys(es.seccion("general")), Object.keys(en.seccion("general")));
  assert.equal(es.faltantes.length, 0);
  assert.equal(en.faltantes.length, 0);
  assert.equal(crearTraductorNominas(es)("titulo"), "Nóminas y retribuciones");
  assert.equal(crearTraductorNominas(en)("titulo"), "Payslips and pay");
  assert.equal(crearTraductorNominas(en)("seleccionado", { periodo: "2026-09", version: 2 }), "Selected payslip: 2026-09, version 2.");
  assert.throws(() => crearTraductorNominas(en)("clave_ausente"), /clave de textos desconocida/);
  assert.match(es.traducir("general.explicacion_no_configurado"), /No se muestran importes, pagos ni recibos de ejemplo/);
  assert.match(es.traducir("general.explicacion_vacio"), /no acredita un importe cero ni un pago/);
  assert.match(es.traducir("general.certificados_pendientes"), /No hay una consulta de certificados fiscales conectada/);
});

test("si falta el catálogo elegido se usa el respaldo común", async () => {
  const respaldo = JSON.parse(await readFile(new URL("../../../textos/es/nominas.json", import.meta.url), "utf8"));
  const avisos = [];
  const textos = await cargarTextos("nominas", {
    idioma: "en", porDefecto: "es", avisar: (aviso) => avisos.push(aviso),
    leer: async (url) => {
      if (url.pathname.endsWith("/en/nominas.json")) throw new Error("catálogo no disponible");
      return respaldo;
    },
  });
  assert.equal(textos.idioma, "es");
  assert.equal(crearTraductorNominas(textos)("titulo"), "Nóminas y retribuciones");
  assert.equal(avisos.length, 1);
});

const vista = await readFile(new URL("./vista.js", import.meta.url), "utf8");
const i18n = await readFile(new URL("./i18n.js", import.meta.url), "utf8");
const i18nVersionada = new URL("./i18n.js?v=20261007-u-nominas", import.meta.url);
assert.match(vista, /from "\.\/i18n\.js\?v=20261007-u-nominas"/);
assert.equal((await import(i18nVersionada.href)).crearTraductorNominas()("titulo"), crearTraductorNominas(TEXTOS_NOMINAS)("titulo"));
assert.match(i18n, /await cargarTextos\("nominas"\)/);
assert.doesNotMatch(i18n, /MENSAJES_NOMINAS_ES|"es-ES"|"en-GB"|Nóminas y retribuciones|Payslips and pay/);
assert.equal(typeof (await import("./vista.js")).montarVistaNominas, "function");
assert.doesNotMatch(vista, /datos-presentacion|datos-sinteticos|localStorage|sessionStorage|document\.cookie|fetch\(|"es-ES"|"en-GB"/);
assert.match(vista, /fuente\.consultar\(\{ signal: controlador\.signal \}\)/);
assert.match(vista, /descarga\.disabled = descargando \|\| !recibo\.descargable \|\| typeof fuente\?\.descargar !== "function"/);
assert.match(vista, /await validarDocumento\(await fuente\.descargar\(recibo\.referencia/);
assert.match(vista, /URL\.createObjectURL\(archivo\.contenido\)/);
assert.match(vista, /URL\.revokeObjectURL\(url\)/);

const css = await readFile(new URL("./nominas.css", import.meta.url), "utf8");
assert.match(css, /\.nominas-principal \{ display: grid; grid-template-columns:/);
assert.match(css, /\.nominas-tabla \{ max-width: 100%; max-height: min\(45vh, 420px\); min-width: 0; overflow: auto;/);
assert.match(css, /@media \(max-width: 900px\)/);
assert.match(css, /@media \(max-width: 620px\)/);
console.log("nominas presentation tests: ok");

function crearDOM() {
  class Nodo {
    constructor(doc, etiqueta) {
      this.ownerDocument = doc;
      this.tagName = etiqueta;
      this.children = [];
      this.dataset = {};
      this.atributos = new Map();
      this.listeners = new Map();
      this.textContent = "";
      this.hidden = false;
    }
    append(...nodos) { this.children.push(...nodos); nodos.forEach((nodo) => { nodo.parent = this; }); }
    replaceChildren(...nodos) { this.children = []; this.append(...nodos); }
    setAttribute(clave, valor) { this.atributos.set(clave, String(valor)); }
    addEventListener(clave, fn) { this.listeners.set(clave, fn); }
    focus() { this.ownerDocument.activeElement = this; }
    remove() { this.parent.children = this.parent.children.filter((nodo) => nodo !== this); }
    querySelectorAll(selector) {
      const coincide = (nodo) => selector === "button[aria-controls]"
        ? nodo.tagName === "button" && nodo.atributos.has("aria-controls") : nodo.tagName === selector;
      return this.children.flatMap((nodo) => [
        ...(coincide(nodo) ? [nodo] : []), ...nodo.querySelectorAll(selector),
      ]);
    }
    querySelector(selector) { return this.querySelectorAll(selector)[0] ?? null; }
  }
  const doc = { createElement: (etiqueta) => new Nodo(doc, etiqueta), activeElement: null };
  const raiz = new Nodo(doc, "root");
  const buscar = (clase) => {
    const visitar = (nodo) => nodo.className?.split(" ").includes(clase)
      ? nodo : nodo.children.map(visitar).find(Boolean);
    return visitar(raiz);
  };
  return { raiz, doc, buscar };
}

test("la ayuda extensa se abre solo con ? y Escape devuelve el foco", () => {
  const { raiz, doc, buscar } = crearDOM();
  const vistaMontada = montarVistaNominas({ raiz });
  const ayuda = buscar("nominas-ayuda");
  const boton = buscar("nominas-ayuda-boton");
  const texto = ayuda.querySelector("p");
  assert.equal(boton.textContent, "?");
  assert.equal(boton.atributos.get("aria-label"), crearTraductorNominas()("ayuda"));
  assert.equal(boton.atributos.get("aria-controls"), texto.id);
  assert.equal(texto.hidden, true);
  boton.listeners.get("click")();
  assert.equal(texto.hidden, false);
  assert.equal(boton.atributos.get("aria-expanded"), "true");
  boton.listeners.get("keydown")({ key: "Escape" });
  assert.equal(texto.hidden, true);
  assert.equal(doc.activeElement, boton);
  vistaMontada.desmontar();
});

test("filtrar período actualiza filas, detalle y foco sin conservar otro recibo", async () => {
  const { raiz, doc, buscar } = crearDOM();
  const recibos = [
    { referencia: "REC-SEP", periodo: "2026-09", tipo: "Nómina", version: 1, descargable: false },
    { referencia: "REC-AGO", periodo: "2026-08", tipo: "Nómina", version: 2, descargable: false },
  ];
  const vistaMontada = montarVistaNominas({ raiz, fuente: {
    consultar: async () => ({ estado: "disponible", origen: "Fuente de prueba", recibos }),
  } });
  await vistaMontada.consultar();
  const detalle = buscar("nominas-detalle");
  const texto = (nodo) => [nodo.textContent, ...nodo.children.map(texto)].join(" ");
  let selector = buscar("nominas-barra").querySelector("select");
  selector.value = "2026-08";
  selector.listeners.get("change")();
  selector = buscar("nominas-barra").querySelector("select");
  assert.equal(selector.value, "2026-08");
  assert.equal(doc.activeElement, selector);
  assert.match(texto(detalle), /REC-AGO/);
  assert.doesNotMatch(texto(detalle), /REC-SEP/);
  assert.equal(buscar("nominas-tabla").querySelectorAll("tr").length, 2);
  assert.equal(buscar("nominas-tabla").querySelector("button[aria-controls]").atributos.get("aria-pressed"), "true");
  vistaMontada.desmontar();
});

test("la vista recibe textos ingleses y fecha del formateador común", async () => {
  const base = await cargarTextos("nominas", { idioma: "en", avisar: assert.fail });
  assert.equal(base.fecha(new Date("2026-10-25T23:30:00Z"), {
    day: "2-digit", month: "2-digit", year: "numeric", timeZone: "Europe/Madrid",
  }), "26/10/2026");
  const fechas = [];
  const textos = {
    ...base,
    fecha: (valor, opciones) => {
      fechas.push({ valor, opciones });
      return "DATE_FROM_SHARED_FORMATTER";
    },
  };
  const { raiz, buscar } = crearDOM();
  const vistaMontada = montarVistaNominas({ raiz, textos, fuente: {
    consultar: async () => ({
      estado: "disponible", origen: "Authorised source", actualizado_en: "2026-10-25T23:30:00Z",
      recibos: [{ referencia: "REC-OCT", periodo: "2026-10", tipo: "Payslip", version: 1, fecha_emision: "2026-10-25T23:30:00Z", descargable: false }],
    }),
  } });
  await vistaMontada.consultar();
  const texto = (nodo) => [nodo.textContent, ...nodo.children.map(texto)].join(" ");
  assert.match(texto(raiz), /Payslips and pay[\s\S]*Payslip history/);
  assert.match(texto(raiz), /Authorised source/);
  assert.doesNotMatch(texto(raiz), /Nóminas y retribuciones|Historial de recibos/);
  assert.match(texto(buscar("nominas-metadatos")), /DATE_FROM_SHARED_FORMATTER/);
  assert.match(texto(buscar("nominas-tabla")), /DATE_FROM_SHARED_FORMATTER/);
  assert.match(texto(buscar("nominas-detalle")), /DATE_FROM_SHARED_FORMATTER/);
  assert.equal(fechas.length, 3);
  assert.ok(fechas.every(({ valor, opciones }) => valor instanceof Date && opciones.timeZone === "Europe/Madrid"));
  vistaMontada.desmontar();
});
