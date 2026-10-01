import assert from "node:assert/strict";
import { readFile } from "node:fs/promises";
import test from "node:test";
import { cargarMensajesPortal } from "./portal-i18n.js?v=20261001-ct-a-i18n-v1";
import { renderizarPreferencias } from "../area-personal/preferencias.js";

const catalogo = async (idioma) => JSON.parse(await readFile(
  new URL(`../textos/${idioma}/preferencias.json`, import.meta.url), "utf8"));

function hojas(seccion, prefijo = "", resultado = {}) {
  for (const [clave, valor] of Object.entries(seccion)) {
    const ruta = prefijo ? `${prefijo}.${clave}` : clave;
    if (typeof valor === "string") resultado[ruta] = valor;
    else hojas(valor, ruta, resultado);
  }
  return resultado;
}

test("RRHH consume los textos ES y EN exactos de los JSON comunes", async () => {
  const [es, en] = await Promise.all([catalogo("es"), catalogo("en")]);
  const [portalES, portalEN] = await Promise.all([cargarMensajesPortal("es"), cargarMensajesPortal("en")]);
  for (const [catalogoPortal, esperado] of [[portalES, hojas(es.portal)], [portalEN, hojas(en.portal)]]) {
    for (const [clave, texto] of Object.entries(esperado)) assert.equal(catalogoPortal[clave], texto, clave);
  }
  const fuente = await readFile(new URL("./portal-i18n.js", import.meta.url), "utf8");
  assert.match(fuente, /"preferencias"/u);
  for (const mensaje of [...Object.values(hojas(es.portal)), ...Object.values(hojas(en.portal))]) {
    if (mensaje.length >= 20) assert.ok(!fuente.includes(mensaje), "mensaje visible incrustado en JavaScript");
  }
});

test("Área personal cambia los textos de preferencias con el idioma del documento", async () => {
  const [es, en] = await Promise.all([catalogo("es"), catalogo("en")]);
  const anterior = globalThis.document;
  try {
    for (const [idioma, mensajes] of [["es", es.areaPersonal], ["en", en.areaPersonal]]) {
      globalThis.document = { documentElement: { lang: idioma } };
      const html = renderizarPreferencias({ error: { codigo: "servicio" } });
      assert.ok(html.includes(mensajes.preferencias.sinDatos));
      assert.ok(html.includes(mensajes.preferencias.error.servicio));
    }
  } finally {
    globalThis.document = anterior;
  }
});

test("los dos paquetes web declaran ambos catálogos", async () => {
  for (const nombre of ["interno", "produccion"]) {
    const manifiesto = await readFile(new URL(`../../${nombre}.manifest`, import.meta.url), "utf8");
    for (const idioma of ["es", "en"]) {
      assert.ok(manifiesto.split("\n").includes(`static/textos/${idioma}/preferencias.json`));
    }
  }
});
