import assert from "node:assert/strict";
import test from "node:test";
import { iniciarAreaPersonal } from "./aplicacion.js";
import { iniciarI18nAreaPersonal } from "./i18n.js";
import { lectorCatalogos } from "./textos-prueba.test-helper.mjs";

await iniciarI18nAreaPersonal({ querySelectorAll: () => [], documentElement: {} }, {
  leer: lectorCatalogos(), pantalla: "preferencias",
  ubicacion: { href: "https://vec.example/area-personal/?lang=es" },
});

function entorno(vista) {
  const elementos = new Map();
  const eventos = new Map();
  const documento = { title: "", activeElement: null, body: { dataset: {} },
    documentElement: { lang: "es" }, querySelectorAll: () => [], querySelector: () => null,
    addEventListener(tipo, fn) { eventos.set(tipo, fn); },
    getElementById(id) {
      if (id === "enlace-inicio-institucional") return null;
      if (!elementos.has(id)) elementos.set(id, { dataset: {}, hidden: false, innerHTML: "", textContent: "",
        setAttribute() {}, removeAttribute() {}, addEventListener() {},
        replaceChildren() { this.innerHTML = ""; }, querySelector: () => null, focus() {} });
      return elementos.get(id);
    } };
  const ventana = { location: { search: `?vista=${vista}`, pathname: "/area-personal/", origin: "https://vec.example" },
    history: { pushState(_estado, _titulo, url) { ventana.location.search = new URL(url, ventana.location.origin).search; },
      replaceState() {} }, addEventListener() {}, scrollTo() {} };
  return { documento, ventana, eventos };
}

test("la persona sin participación abre inscripción sin consultar Mi Bolsa ni inventar identidad", async () => {
  const anteriores = { document: globalThis.document, window: globalThis.window };
  const { documento, ventana } = entorno("inscripcion");
  globalThis.document = documento; globalThis.window = ventana;
  let montajes = 0;
  try {
    const estado = await iniciarAreaPersonal({
      cliente: { cargar() { assert.fail("inscripción no depende de Mi Bolsa"); } },
      vistasDisponibles: new Set(["inicio", "llamamientos", "inscripcion"]),
      cargarVistaInscripcion: async () => ({ montarInscripcionBolsa({ contenedor, idioma }) {
        assert.equal(contenedor, documento.getElementById("inscripcion-bolsa-montaje"));
        assert.equal(typeof idioma, "string"); ++montajes; return { destruir() {} };
      } }),
    });
    await Promise.resolve();
    assert.equal(montajes, 1);
    assert.equal(estado.soloInscripcion, true);
    assert.equal(estado.datos.sesion.persona_ref, null);
    assert.equal(estado.datos.sesion.nombre_visible, "");
    assert.deepEqual(estado.datos.capacidades, {});
  } finally { globalThis.document = anteriores.document; globalThis.window = anteriores.window; }
});

test("la respuesta antigua de Mi Bolsa no remonta inscripción tras cambiar de vista", async () => {
  const anteriores = { document: globalThis.document, window: globalThis.window };
  const { documento, ventana, eventos } = entorno("llamamientos");
  globalThis.document = documento; globalThis.window = ventana;
  let terminar;
  let montajes = 0;
  try {
    const inicio = iniciarAreaPersonal({
      cliente: { cargar: () => new Promise((resolve) => { terminar = resolve; }) },
      vistasDisponibles: new Set(["inicio", "llamamientos", "inscripcion"]),
      cargarVistaInscripcion: async () => ({ montarInscripcionBolsa() { ++montajes; return { destruir() {} }; } }),
    });
    await eventos.get("click")({ preventDefault() {}, target: { closest: (selector) =>
      selector === "[data-ruta]" ? { dataset: { ruta: "inscripcion" } } : null } });
    terminar({ consulta: { consultada_en: "2026-10-09T09:00:00Z", participaciones: [] } });
    const estado = await inicio;
    await Promise.resolve();
    assert.equal(estado.vista, "inscripcion");
    assert.equal(estado.soloInscripcion, true);
    assert.equal(montajes, 1);
  } finally { globalThis.document = anteriores.document; globalThis.window = anteriores.window; }
});

test("con la inscripción desactivada en vistas.json no se monta ni se consulta su API", async () => {
  const anteriores = { document: globalThis.document, window: globalThis.window };
  const { documento, ventana } = entorno("inscripcion");
  globalThis.document = documento; globalThis.window = ventana;
  let cargasMiBolsa = 0;
  try {
    const estado = await iniciarAreaPersonal({
      cliente: { async cargar() { ++cargasMiBolsa; return { consulta: { consultada_en: "2026-10-09T09:00:00Z", participaciones: [] } }; } },
      vistasDisponibles: new Set(["inicio", "llamamientos"]),
      cargarVistaInscripcion: async () => assert.fail("la vista apagada no se carga"),
    });
    await Promise.resolve();
    assert.notEqual(estado.vista, "inscripcion");
    assert.equal(estado.soloInscripcion, false);
    assert.equal(cargasMiBolsa, 1);
  } finally { globalThis.document = anteriores.document; globalThis.window = anteriores.window; }
});
