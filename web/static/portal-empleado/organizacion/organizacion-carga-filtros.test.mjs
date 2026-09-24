import assert from "node:assert/strict";
import test from "node:test";
import {
  API_ORGANIZACION,
  ESQUEMA_ORGANIZACION,
  crearCliente,
  iniciarOrganizacion,
} from "./organizacion.js";

function documento() {
  const elementos = new Map();
  const obtener = (selector) => {
    if (!elementos.has(selector)) {
      elementos.set(selector, {
        value: "",
        hidden: false,
        disabled: false,
        textContent: "",
        innerHTML: "",
        querySelectorAll: () => [],
        setAttribute() {},
        focus() { this.enFoco = true; },
      });
    }
    return elementos.get(selector);
  };
  return {
    querySelector: obtener,
    querySelectorAll: () => [],
  };
}

const catalogo = () => ({
  data: {
    esquema: ESQUEMA_ORGANIZACION,
    catalogo_id: "rpt",
    catalogo_version: 1,
    catalogo_revision: 1,
    edicion_habilitada: false,
    catalogo_huella_sha256: "a".repeat(64),
    estado: "borrador",
    fuente_ref: "referencia pública",
    descripcion: "Catálogo de prueba",
    unidades: [
      { clave: "centro-1", etiqueta: "Centro Norte", tipo: "centro" },
      { clave: "centro-2", etiqueta: "Centro Sur", tipo: "centro" },
    ],
  },
});

test("GET pendiente conserva filtros y un evento adelantado no intenta renderizar", async () => {
  const anterior = globalThis.document;
  globalThis.document = documento();
  try {
    let resolver;
    const llamadas = [];
    const cliente = crearCliente((url, options) => {
      llamadas.push({ url, options });
      return new Promise((resolve) => { resolver = resolve; });
    });
    const carga = iniciarOrganizacion(cliente);
    const filtro = document.querySelector("#filter-text");
    const tipo = document.querySelector("#filter-type");
    assert.equal(filtro.disabled, true);
    assert.equal(tipo.disabled, true);
    assert.equal(document.querySelector("#table-wrap").hidden, true);
    filtro.value = "Norte";
    tipo.value = "centro";
    assert.doesNotThrow(() => filtro.oninput());
    assert.doesNotThrow(() => tipo.onchange());
    resolver({ ok: true, json: async () => catalogo() });
    await carga;
    assert.equal(filtro.value, "Norte");
    assert.equal(tipo.value, "centro");
    assert.equal(filtro.disabled, false);
    assert.equal(tipo.disabled, false);
    assert.equal(document.querySelector("#result-count").textContent, "1 de 2 unidades");
    assert.match(document.querySelector("#rows").innerHTML, /Centro Norte/);
    assert.doesNotMatch(document.querySelector("#rows").innerHTML, /Centro Sur/);
    assert.deepEqual(llamadas.map(({ url }) => url), [API_ORGANIZACION]);
    assert.equal(llamadas[0].options.cache, "no-store");
  } finally {
    globalThis.document = anterior;
  }
});

test("GET denegado no muestra datos; un reintento recupera filtros y foco sin POST", async () => {
  const anterior = globalThis.document;
  globalThis.document = documento();
  try {
    const llamadas = [];
    let intento = 0;
    const cliente = crearCliente(async (url, options) => {
      llamadas.push({ url, options });
      intento++;
      if (intento === 1) return { ok: false, status: 403, json: async () => catalogo() };
      return { ok: true, json: async () => catalogo() };
    });
    const filtro = document.querySelector("#filter-text");
    filtro.value = "Sur";
    await iniciarOrganizacion(cliente);
    assert.equal(filtro.disabled, true);
    assert.equal(document.querySelector("#filter-type").disabled, true);
    assert.equal(document.querySelector("#table-wrap").hidden, true);
    assert.equal(document.querySelector("#rows").innerHTML, "");
    assert.equal(document.querySelector("#result-count").textContent, "");
    assert.match(document.querySelector("#state").innerHTML, /Reintentar/);
    assert.equal(document.querySelector("#retry").enFoco, true);
    assert.doesNotThrow(() => filtro.oninput());
    await document.querySelector("#retry").onclick();
    assert.equal(filtro.value, "Sur");
    assert.equal(filtro.disabled, false);
    assert.equal(document.querySelector("#result-count").textContent, "1 de 2 unidades");
    assert.equal(filtro.enFoco, true);
    assert.deepEqual(llamadas.map(({ url }) => url), [API_ORGANIZACION, API_ORGANIZACION]);
    assert.ok(llamadas.every(({ options }) => !options.method || options.method === "GET"));
  } finally {
    globalThis.document = anterior;
  }
});
