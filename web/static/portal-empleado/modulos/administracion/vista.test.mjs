import assert from "node:assert/strict";
import test from "node:test";
import { readFile } from "node:fs/promises";
import { CLAVES_I18N_ADMINISTRACION, crearTraductorAdministracion } from "./i18n.js";
import { montarVistaAdministracion, PESTANAS_ADMINISTRACION } from "./vista.js";

class Elemento {
  constructor(documento, etiqueta) {
    this.ownerDocument = documento;
    this.etiqueta = etiqueta;
    this.children = [];
    this.atributos = new Map();
    this.eventos = new Map();
    this.dataset = {};
    this.textContent = "";
  }
  append(...hijos) { this.children.push(...hijos); }
  replaceChildren(...hijos) { this.children = hijos; }
  setAttribute(clave, valor) { this.atributos.set(clave, String(valor)); }
  getAttribute(clave) { return this.atributos.get(clave) ?? null; }
  addEventListener(clave, fn) { this.eventos.set(clave, fn); }
  emitir(clave, datos = {}) { this.eventos.get(clave)?.({ preventDefault() {}, ...datos }); }
  focus() { this.ownerDocument.activeElement = this; }
  buscar(predicado) {
    if (predicado(this)) return this;
    for (const hijo of this.children) {
      const encontrado = hijo.buscar(predicado);
      if (encontrado) return encontrado;
    }
    return null;
  }
}

function documentoFalso() {
  const documento = { createElement(etiqueta) { return new Elemento(documento, etiqueta); } };
  documento.documentElement = documento.createElement("html");
  documento.body = documento.createElement("body");
  return documento;
}

test("ADMIN solo muestra no configurado y controles vacíos sin fuente", () => {
  const documento = documentoFalso();
  const raiz = documento.createElement("div");
  const vista = montarVistaAdministracion({ raiz });
  assert.equal(raiz.buscar((n) => n.dataset.estadoEntrega === "no_configurado")?.etiqueta, "section");
  assert.equal(raiz.buscar((n) => n.dataset.configuracionEstado === "no_configurado")?.etiqueta, "section");
  assert.deepEqual(PESTANAS_ADMINISTRACION, ["resumen", "roles", "catalogos", "calendarios", "reglas", "conectores", "modulos", "privacidad", "ia", "apariencia"]);

  raiz.buscar((n) => n.dataset.administracionPestana === "roles").emitir("click");
  const guardar = raiz.buscar((n) => n.etiqueta === "button" && n.textContent === "Guardar asignación");
  assert.equal(guardar.disabled, true);
  assert.equal(raiz.buscar((n) => n.dataset.configuracionEstado === "no_configurado")?.etiqueta, "section");

  raiz.buscar((n) => n.dataset.administracionPestana === "ia").emitir("click");
  const campos = raiz.buscar((n) => n.className === "administracion-ia-campos");
  assert.equal(campos.children.length, 3);
  for (const etiqueta of campos.children) {
    assert.equal(etiqueta.children[0].value, "");
    assert.equal(etiqueta.children[0].disabled, true);
  }
  vista.desmontar();
  assert.deepEqual(raiz.children, []);
});

test("ADMIN usa claves i18n cerradas y no carga ejemplos ni almacenamiento web", async () => {
  const t = crearTraductorAdministracion();
  assert.equal(t("no_configurado"), "No configurado");
  assert.throws(() => t("clave_inexistente"), /clave i18n/u);
  const vista = await readFile(new URL("./vista.js", import.meta.url), "utf8");
  const catalogo = await readFile(new URL("./i18n.js", import.meta.url), "utf8");
  const claves = [...vista.matchAll(/t\("([a-z_]+)"\)/gu)].map((coincidencia) => coincidencia[1]);
  claves.forEach((clave) => assert.ok(CLAVES_I18N_ADMINISTRACION.includes(clave), "clave ausente: " + clave));
  assert.doesNotMatch(vista, /datos-presentacion|fetch\(|localStorage|sessionStorage|document\.cookie|indexedDB|localhost/u);
  assert.doesNotMatch(catalogo, /demostrat|sintétic|fictici|ejemplo|localhost/iu);
});
