import assert from "node:assert/strict";
import { readFile } from "node:fs/promises";
import test from "node:test";
import { renderizarErrorCargaAreaPersonal } from "./aplicacion.js";
import { iniciarI18nAreaPersonal, traducir } from "./i18n.js";
import { textoContactoPropio } from "./i18n-contacto-propio.js";
import { tabla } from "./vistas/comunes.js";
import { exigirMismaVersion, versionDe, versionesDe } from "../portal-empleado/versiones-cache.test-helper.mjs";
import { catalogoPlano, lectorCatalogos } from "./textos-prueba.test-helper.mjs";

test("Área personal comparte la versión calculada entre entrada, módulos y caché", async () => {
  const leer = (ruta) => readFile(new URL(ruta, import.meta.url), "utf8");
  const [html, cache, arranque, aplicacion, contrato, cliente, contacto, seguimiento, manifiesto] = await Promise.all([
    "index.html", "cache-publica-v1.json", "arranque.js", "aplicacion.js", "contrato.js",
    "cliente-http.js", "contacto-propio.js", "vistas/seguimiento-tramites.js",
    "../../produccion.manifest",
  ].map(leer));
  exigirMismaVersion([html, cache], "arranque.js");
  exigirMismaVersion([arranque, contacto], "cliente-http.js");
  exigirMismaVersion([contrato, aplicacion, seguimiento], "mi-bolsa-portal.js");
  exigirMismaVersion([aplicacion, seguimiento], "mi-bolsa-historial.js");
  versionDe(arranque, "./aplicacion.js");
  versionDe(cliente, "./contrato.js");
  versionDe(html, "/area-personal/area-personal.css");
  assert.equal(versionesDe(arranque, "./cliente-http.js").length, 2);
  assert.ok(arranque.includes('from "./i18n.js"'));
  assert.ok(aplicacion.includes('from "./i18n.js"'));
  assert.equal(JSON.parse(cache).version, "20261003-pwa-ci-v5");
  for (const ruta of ["mi-bolsa-portal.js", "mi-bolsa-historial.js", "contrato.js", "cliente-http.js",
    "contacto-propio.js", "vistas/seguimiento-tramites.js", "aplicacion.js", "arranque.js"])
    assert.equal(manifiesto.split(`static/area-personal/${ruta}`).length - 1, 1, ruta);
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
  await iniciarI18nAreaPersonal({ querySelectorAll: () => [] }, { leer: lectorCatalogos(entradas), ubicacion: { href: "https://vec.example/area-personal/?lang=es" } });
  assert.equal(textoContactoPropio("titulo"), traducir("areaPersonal.contacto.titulo"));
  assert.equal(textoContactoPropio("titulo"), entradas["areaPersonal.contacto.titulo"]);
  assert.match(renderizarErrorCargaAreaPersonal(new Error("causa privada")), /Error del catálogo compartido/);
  assert.match(tabla({ descripcion: "", columnas: [], filas: [] }), /Vacío del catálogo compartido/);
});
