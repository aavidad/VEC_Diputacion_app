import assert from "node:assert/strict";
import test from "node:test";
import { exigirParametrosConocidos, iniciarAreaPersonal } from "./aplicacion.js";
import { iniciarI18nAreaPersonal } from "./i18n.js";
import { lectorCatalogos } from "./textos-prueba.test-helper.mjs";

await iniciarI18nAreaPersonal({ querySelectorAll: () => [], documentElement: {} }, {
  leer: lectorCatalogos(), ubicacion: { href: "https://vec.example/area-personal/?lang=es" },
});

test("la cartera abre desde su URL filtrada sin pedir Mi Bolsa", async () => {
  const anterior = { document: globalThis.document, window: globalThis.window };
  const elementos = new Map();
  const documento = { title: "", activeElement: null, body: { dataset: {} }, documentElement: { lang: "es" },
    querySelectorAll: () => [], querySelector: () => null, addEventListener() {},
    getElementById(id) {
      if (id === "enlace-inicio-institucional") return null;
      if (!elementos.has(id)) elementos.set(id, { dataset: {}, hidden: false, innerHTML: "", textContent: "",
        setAttribute() {}, removeAttribute() {}, addEventListener() {}, replaceChildren() { this.innerHTML = ""; },
        querySelector: () => null, focus() {} });
      return elementos.get(id);
    } };
  const ventana = { location: { href: "https://vec.example/area-personal/?vista=meritos&filtro=retirados",
    search: "?vista=meritos&filtro=retirados", pathname: "/area-personal/", origin: "https://vec.example" },
  history: { pushState() {}, replaceState() {} }, addEventListener() {}, scrollTo() {} };
  globalThis.document = documento; globalThis.window = ventana;
  let montajes = 0;
  try {
    const estado = await iniciarAreaPersonal({
      cliente: { cargar() { assert.fail("Mi Bolsa es independiente de la cartera"); } },
      vistasDisponibles: new Set(["inicio", "llamamientos", "meritos"]),
      cargarVistaMeritos: async () => ({ montarMeritosReales({ contenedor, ubicacion, idioma }) {
        assert.equal(contenedor, documento.getElementById("meritos-reales-montaje"));
        assert.equal(ubicacion.search, "?vista=meritos&filtro=retirados");
        assert.equal(idioma, "es");
        ++montajes;
        return { pendiente: () => null, desmontar() {} };
      } }),
    });
    await Promise.resolve();
    assert.equal(montajes, 1);
    assert.equal(estado.soloMeritos, true);
    assert.equal(estado.datos.sesion.persona_ref, null);
    assert.equal(estado.datos.meta.origen, "Mis méritos");
  } finally { globalThis.document = anterior.document; globalThis.window = anterior.window; }
});

test("el filtro compartible solo pertenece a la vista de méritos", () => {
  assert.doesNotThrow(() => exigirParametrosConocidos(new URLSearchParams("vista=meritos&filtro=disponibles")));
  assert.throws(() => exigirParametrosConocidos(new URLSearchParams("vista=perfil&filtro=disponibles")));
});
