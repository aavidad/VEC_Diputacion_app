import assert from "node:assert/strict";
import { readFile } from "node:fs/promises";
import test from "node:test";
import { renderizarErrorCargaAreaPersonal } from "./aplicacion.js";
import { iniciarI18nAreaPersonal, traducir } from "./i18n.js";
import { textoContactoPropio } from "./i18n-contacto-propio.js";
import { tabla } from "./vistas/comunes.js";

test("el HTML y los módulos cambiados usan URLs nuevas bajo caché inmutable", async () => {
  const [html, arranque, aplicacion, vista] = await Promise.all([
    readFile(new URL("./index.html", import.meta.url), "utf8"),
    readFile(new URL("./arranque.js", import.meta.url), "utf8"),
    readFile(new URL("./aplicacion.js", import.meta.url), "utf8"),
    readFile(new URL("../comun/oportunidades/vista.js", import.meta.url), "utf8"),
  ]);
  const versionCSSArea = "20260924-f2-area-tema-v2";
  const versionCSSOportunidades = "20260924-f2-b15-area-v1";
  const versionPadre = "20260924-rescate-area-v7";
  assert.ok(html.includes(`/area-personal/area-personal.css?v=${versionCSSArea}`));
  assert.ok(html.includes(`/comun/oportunidades/oportunidades.css?v=${versionCSSOportunidades}`));
  assert.ok(!html.includes(`/area-personal/area-personal.css?v=${versionCSSOportunidades}`), "no reutilizar CSS anterior con caché inmutable");
  assert.ok(html.includes(`/area-personal/arranque.js?v=${versionPadre}`));
  assert.ok(arranque.includes(`./aplicacion.js?v=${versionPadre}`));
  assert.ok(arranque.includes('from "./i18n.js"'));
  assert.ok(aplicacion.includes('from "./i18n.js"'));
  assert.doesNotMatch(`${arranque}\n${aplicacion}`, /\.\/i18n\.js\?v=/);
  assert.ok(aplicacion.includes(`../comun/oportunidades/vista.js?v=${versionCSSOportunidades}`));
  assert.match(vista, /\.\/i18n\.js\?v=20260924-f2-web2/);
  assert.match(arranque, /\.\/cliente-http\.js\?v=20260924-f2-b11-v2/);
});

test("una caché antigua v1-v5 no puede sustituir los padres v6 del montaje", async () => {
  const [html, arranque, contacto] = await Promise.all([
    readFile(new URL("./index.html", import.meta.url), "utf8"),
    readFile(new URL("./arranque.js", import.meta.url), "utf8"),
    readFile(new URL("./contacto-propio.js", import.meta.url), "utf8"),
  ]);
  const nuevo = "20260924-rescate-area-v7";
  const padre = new URL(html.match(/src="(\/area-personal\/arranque\.js\?v=[^"]+)"/)?.[1] ?? "", "https://vec.example");
  const hijo = new URL(arranque.match(/from "(\.\/aplicacion\.js\?v=[^"]+)"/)?.[1] ?? "", padre);
  assert.equal(padre.searchParams.get("v"), nuevo);
  assert.equal(hijo.searchParams.get("v"), nuevo);
  for (const antiguo of ["20260924-f2-b15-area-v1", "20260924-f2-b15-area-v2", "20260924-f2-b15-area-v3", "20260924-f2-b15-area-v4", "20260924-rescate-area-v5", "20260924-rescate-area-v6"]) {
    assert.notEqual(padre.href, `https://vec.example/area-personal/arranque.js?v=${antiguo}`);
    assert.notEqual(hijo.href, `https://vec.example/area-personal/aplicacion.js?v=${antiguo}`);
  }
  assert.doesNotMatch(`${arranque}\n${contacto}`, /cliente-http\.js\?v=20260924-f2-b11-v1/);
  assert.match(arranque, /cliente-http\.js\?v=20260924-f2-b11-v2/);
  assert.match(contacto, /cliente-http\.js\?v=20260924-f2-b11-v2/);
});

test("un fallo de arranque usa i18n genérico y no expone su causa", async () => {
  const arranque = await readFile(new URL("./arranque.js", import.meta.url), "utf8");
  assert.match(arranque, /catch \{[\s\S]*traducir\("areaPersonal\.estado\.error\.titulo"\)/);
  assert.match(arranque, /traducir\("areaPersonal\.estado\.error\.detalle"\)/);
  assert.match(arranque, /traducir\("areaPersonal\.estado\.error\.carga\.garantia"\)/);
  assert.doesNotMatch(arranque, /error\.message|error instanceof Error/);
});

test("shell, Contacto y vistas comparten el catálogo cargado por la URL canónica", async () => {
  const entradas = {
    "areaPersonal.contacto.titulo": "Contacto del catálogo compartido",
    "areaPersonal.estado.error.titulo": "Error del catálogo compartido",
    "areaPersonal.tabla.sinResultados": "Vacío del catálogo compartido",
  };
  await iniciarI18nAreaPersonal({ querySelectorAll: () => [] }, async () => ({ ok: true, json: async () => entradas }));
  assert.equal(textoContactoPropio("titulo"), traducir("areaPersonal.contacto.titulo"));
  assert.equal(textoContactoPropio("titulo"), entradas["areaPersonal.contacto.titulo"]);
  assert.match(renderizarErrorCargaAreaPersonal(new Error("causa privada")), /Error del catálogo compartido/);
  assert.match(tabla({ descripcion: "", columnas: [], filas: [] }), /Vacío del catálogo compartido/);
});
