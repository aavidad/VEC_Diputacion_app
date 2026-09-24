import assert from "node:assert/strict";
import test from "node:test";
import { readFileSync } from "node:fs";
import { MONTAJE_ORGANIZACION_HISTORICA, iniciarHistorico, iniciarImportacion,
  iniciarPestanasOrganizacion } from "./historico.js";

const html = readFileSync(new URL("./index.html", import.meta.url), "utf8");
const etiqueta = (id) => html.match(new RegExp(`<[a-z]+[^>]*\\bid="${id}"[^>]*>`))?.[0] ?? "";
const oculto = (id) => /\shidden(?=[\s>])/.test(etiqueta(id));

function documento() {
  const elementos = new Map();
  const ayuda = { history: [{ hidden: true }], import: [{ hidden: true }, { hidden: true }] };
  const elemento = () => {
    const atributos = new Map();
    return {
      hidden: false, tabIndex: 0,
      setAttribute(nombre, valor) { atributos.set(nombre, String(valor)); },
      getAttribute(nombre) { return atributos.get(nombre) ?? null; },
      focus() {},
    };
  };
  const obtener = (selector) => {
    if (!elementos.has(selector)) elementos.set(selector, elemento());
    return elementos.get(selector);
  };
  // Estado inicial publicado: solo el catálogo visible.
  for (const clave of ["history", "import"]) {
    obtener(`#tab-${clave}`).hidden = true;
    obtener(`#${clave}-panel`).hidden = true;
  }
  obtener(".org-tabs").hidden = true;
  obtener("#tab-catalog").setAttribute("aria-selected", "true");
  return {
    ayuda,
    querySelector: obtener,
    querySelectorAll: (selector) => ayuda[/data-org-requiere="(\w+)"/.exec(selector)?.[1]] ?? [],
    getElementById: (id) => obtener(`#${id}`),
  };
}

function conDocumento(fn) {
  const anterior = globalThis.document;
  globalThis.document = documento();
  try {
    return fn(globalThis.document);
  } finally {
    globalThis.document = anterior;
  }
}

test("la pantalla publicada abre en el catálogo y no ofrece histórico ni importación", () => {
  assert.match(etiqueta("tab-catalog"), /aria-selected="true"/);
  assert.equal(oculto("catalog-panel"), false);
  for (const id of ["tab-history", "tab-import", "history-panel", "import-panel"]) {
    assert.equal(oculto(id), true, id);
  }
  assert.match(html, /<nav class="org-tabs"[^>]*\shidden>/);
  for (const [, clave] of html.matchAll(/<p [^>]*data-org-requiere="(\w+)"[^>]*>/g)) {
    assert.ok(["history", "import"].includes(clave));
  }
  assert.ok([...html.matchAll(/<p [^>]*data-org-requiere="\w+"[^>]*>/g)].every(([p]) => /\shidden/.test(p)));
});

test("las rutas históricas no compuestas dejan el montaje apagado y sin pestañas", () => {
  assert.ok(Object.isFrozen(MONTAJE_ORGANIZACION_HISTORICA));
  assert.deepEqual({ ...MONTAJE_ORGANIZACION_HISTORICA }, { consulta: false, importacion: false });
  conDocumento((doc) => {
    const pestanas = iniciarPestanasOrganizacion();
    assert.deepEqual(pestanas.pestanas, ["catalog"]);
    assert.equal(doc.querySelector(".org-tabs").hidden, true);
    assert.equal(doc.querySelector("#tab-history").hidden, true);
    assert.equal(doc.querySelector("#tab-import").hidden, true);
    assert.ok([...doc.ayuda.history, ...doc.ayuda.import].every((p) => p.hidden));
    pestanas.seleccionar("history");
    assert.equal(doc.querySelector("#history-panel").hidden, true, "una pestaña no montada no se abre");
    assert.equal(doc.querySelector("#tab-catalog").getAttribute("aria-selected"), "true");
    // Sin montaje no se inician consulta ni importación: ninguna petición a rutas ausentes.
    let llamadas = 0;
    const cliente = { consultar: async () => { llamadas += 1; }, enviar: async () => { llamadas += 1; } };
    assert.equal(iniciarHistorico(cliente), null);
    assert.equal(iniciarImportacion(cliente), null);
    assert.equal(llamadas, 0);
  });
});

test("cuando el montaje declara las rutas, las pestañas aparecen tras el catálogo", () => {
  conDocumento((doc) => {
    const pestanas = iniciarPestanasOrganizacion({ consulta: true, importacion: false });
    assert.deepEqual(pestanas.pestanas, ["catalog", "history"]);
    assert.equal(doc.querySelector(".org-tabs").hidden, false);
    assert.equal(doc.querySelector("#tab-history").hidden, false);
    assert.equal(doc.querySelector("#tab-import").hidden, true);
    assert.equal(doc.ayuda.history[0].hidden, false);
    assert.ok(doc.ayuda.import.every((p) => p.hidden));
    pestanas.seleccionar("history");
    assert.equal(doc.querySelector("#history-panel").hidden, false);
    assert.equal(doc.querySelector("#catalog-panel").hidden, true);
    pestanas.seleccionar("import");
    assert.equal(doc.querySelector("#import-panel").hidden, true);
    pestanas.establecerBloqueo(() => true);
    pestanas.seleccionar("catalog");
    assert.equal(doc.querySelector("#history-panel").hidden, false, "una operación en curso bloquea el cambio");
  });
});
