import "./inicializar-i18n.test-helper.mjs";
import assert from "node:assert/strict";
import { readFile } from "node:fs/promises";
import { test } from "node:test";
import { iniciarAreaPersonal } from "./aplicacion.js";
import { iniciarI18nAreaPersonal } from "./i18n.js";
import { catalogoPlano, lectorCatalogos } from "./textos-prueba.test-helper.mjs";

const valores = { idioma: "navegador", tamano_texto: "normal", alto_contraste: false,
  tema: "sistema", inicio: "bolsas", filas: 20, aviso_correo_tareas: false, aviso_correo_plazos: false };
const preferencias = { catalogo: { version_ref: "usuarios-preferencias-v1", idiomas: [], tamanos_texto: [],
  temas: [], inicios: [], filas: [20], predeterminados: valores },
estado: { persona_ref: "persona:autorizada", version: 0, catalogo_version_ref: "usuarios-preferencias-v1", valores } };

function documentoFalso() {
  const eventos = new Map();
  const elementos = new Map();
  const documento = { title: "", activeElement: null, body: { dataset: { menuAbierto: "false" } },
    documentElement: { lang: "es" },
    addEventListener(tipo, fn) { eventos.set(tipo, fn); },
    querySelectorAll() { return []; },
    querySelector(selector) {
      if (selector === '[data-accion="ver-sesion"]') return this.getElementById("boton-sesion");
      return null;
    },
    getElementById(id) {
      if (id === "enlace-inicio-institucional") return null;
      if (!elementos.has(id)) elementos.set(id, { id, dataset: {}, hidden: id === "menu-identidad",
        textContent: "", innerHTML: "", className: "", setAttribute() {}, removeAttribute() {},
        addEventListener() {}, querySelector() { return { focus() {} }; }, replaceChildren() { this.innerHTML = ""; },
        focus() { documento.activeElement = this; } });
      return elementos.get(id);
    },
  };
  return { documento, eventos };
}

function pulsar(eventos, selector, valor) {
  const control = { dataset: selector === "[data-ruta]" ? { ruta: valor } : { accion: valor } };
  return eventos.get("click")({ preventDefault() {}, target: { closest(patron) {
    return patron === selector ? control : null;
  } } });
}

async function escenario(idioma, lectura, errorPreferencias = null) {
  const original = { document: globalThis.document, window: globalThis.window };
  const { documento, eventos } = documentoFalso();
  const ventana = { location: { href: `https://vec.example/area-personal/?lang=${idioma}`, search: `?lang=${idioma}`, pathname: "/area-personal/", origin: "https://vec.example" },
    history: { pushState() {} }, addEventListener() {}, scrollTo() {} };
  globalThis.document = documento;
  globalThis.window = ventana;
  try {
    await iniciarI18nAreaPersonal(documento, { leer: lectorCatalogos(), pantalla: "preferencias", preferidos: [idioma],
      ubicacion: { href: `https://vec.example/area-personal/?lang=${idioma}` } });
    let consultasBolsa = 0;
    const cliente = { async cargar() { consultasBolsa += 1; throw { codigo: "servicio_no_disponible" }; } };
    const vistasDisponibles = new Set(["inicio", "llamamientos", "preferencias"]);
    const estado = await iniciarAreaPersonal({ cliente, vistasDisponibles, preferencias: lectura, errorPreferencias });
    assert.equal(consultasBolsa, 1);
    assert.equal(estado.datos, null);
    pulsar(eventos, "[data-accion]", "ver-sesion");
    assert.equal(documento.getElementById("menu-identidad").hidden, false);
    await pulsar(eventos, "[data-accion]", "abrir-preferencias");
    assert.equal(documento.getElementById("menu-identidad").hidden, true);
    const vistaPreferencias = documento.getElementById("espacio-trabajo").innerHTML;
    assert.equal(estado.soloPreferencias, true);
    assert.equal(estado.datos.meta.origen, "GET /api/vec/usuarios/area-personal/mis-preferencias");
    const metodo = estado.datos.sesion.metodo;
    pulsar(eventos, "[data-ruta]", "llamamientos");
    await new Promise((resolve) => setImmediate(resolve));
    assert.equal(consultasBolsa, 2);
    const vueltaBolsa = documento.getElementById("espacio-trabajo").innerHTML;
    assert.equal(estado.datos, null);
    assert.doesNotMatch(vueltaBolsa, /formulario-preferencias/u);
    pulsar(eventos, "[data-accion]", "ver-sesion");
    await pulsar(eventos, "[data-accion]", "abrir-preferencias");
    const reabierta = documento.getElementById("espacio-trabajo").innerHTML;
    return { vistaPreferencias, vueltaBolsa, reabierta, metodo };
  } finally {
    globalThis.document = original.document;
    globalThis.window = original.window;
  }
}

test("GET preferencias 200 y Mi Bolsa 503 permite abrir el formulario y volver al error real, ES/EN", async () => {
  for (const [idioma, titulo] of [["es", "Mis preferencias"], ["en", "My preferences"]]) {
    const resultado = await escenario(idioma, preferencias);
    assert.match(resultado.vistaPreferencias, /id="formulario-preferencias"/u);
    assert.ok(resultado.vistaPreferencias.includes(titulo));
    assert.match(resultado.vueltaBolsa, /estado-error/u);
    assert.match(resultado.reabierta, /id="formulario-preferencias"/u);
  }
});

test("GET preferencias 403 conserva su denegación aunque Mi Bolsa falle, ES/EN", async () => {
  for (const [idioma, mensaje] of [["es", "No dispone de permiso"], ["en", "You do not have permission"]]) {
    const resultado = await escenario(idioma, null, { codigo: "denegado" });
    assert.doesNotMatch(resultado.vistaPreferencias, /id="formulario-preferencias"/u);
    assert.ok(resultado.vistaPreferencias.includes(mensaje));
    assert.doesNotMatch(resultado.reabierta, /id="formulario-preferencias"/u);
    assert.match(resultado.metodo, /no confirmada|not confirmed/u);
  }
});
