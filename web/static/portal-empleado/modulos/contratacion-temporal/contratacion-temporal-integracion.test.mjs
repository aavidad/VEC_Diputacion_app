import assert from "node:assert/strict";
import { readFile, readdir } from "node:fs/promises";
import test from "node:test";

import { MENSAJES_CONTRATACION_TEMPORAL_ES } from "./i18n.js";
import {
  VISTAS_MODULOS_CONECTADOS,
  VISTAS_MODULOS_PERSONALES,
  moduloDeVistaPortal,
  rutaDeVistaPortal,
} from "../../portal-modulos-coordinador.js";

const directorio = new URL("./", import.meta.url);
const [
  contratoFuente,
  presentadorFuente,
  vistaFuente,
  estilos,
  coordinadorFuente,
  coordinadorPruebas,
  indicePortal,
  catalogoPresentacion,
] = await Promise.all([
  readFile(new URL("contrato.js", directorio), "utf8"),
  readFile(new URL("presentador.js", directorio), "utf8"),
  readFile(new URL("vista.js", directorio), "utf8"),
  readFile(new URL("contratacion-temporal.css", directorio), "utf8"),
  readFile(new URL("../../portal-modulos-coordinador.js", directorio), "utf8"),
  readFile(new URL("../../portal-modulos-coordinador.test.mjs", directorio), "utf8"),
  readFile(new URL("../../index.html", directorio), "utf8"),
  readFile(new URL("../../portal-catalogo-presentacion.js", directorio), "utf8"),
]);

test("i18n cubre los textos estáticos y CSS hereda tema, zoom y contraste", () => {
  const clavesEstaticas = [...vistaFuente.matchAll(/\bt\("([^"]+)"/g)]
    .map((coincidencia) => coincidencia[1]);
  assert.ok(clavesEstaticas.length > 50);
  for (const clave of clavesEstaticas) {
    assert.ok(
      Object.hasOwn(MENSAJES_CONTRATACION_TEMPORAL_ES, clave),
      `falta la traducción ${clave}`,
    );
  }
  assert.match(estilos, /var\(--portal-(?:tinta|superficie|borde|azul-700)\)/);
  assert.match(estilos, /@media \(max-width: 1180px\)/);
  assert.match(estilos, /@media \(max-width: 720px\)/);
  assert.match(estilos, /@media \(max-width: 420px\)/);
  assert.match(estilos, /prefers-reduced-motion: reduce/);
  assert.match(estilos, /forced-colors: active/);
  assert.match(estilos, /overflow-wrap: anywhere/);
  assert.match(estilos, /scroll-margin-block: 84px/);
  assert.match(estilos, /\.ct-estado-exito\s*\{[^}]+color: var\(--portal-tinta\)/s);
  assert.doesNotMatch(estilos, /font-family:|#[0-9a-f]{3,8}\b/i);
  assert.doesNotMatch(vistaFuente, /style="/);
});

test("el módulo no usa red, cookies, almacenamiento web ni registra claves", () => {
  const fuentes = `${contratoFuente}\n${presentadorFuente}\n${vistaFuente}`;
  assert.doesNotMatch(
    fuentes,
    /\b(?:fetch|XMLHttpRequest|WebSocket|EventSource)\s*\(|document\.cookie|localStorage|sessionStorage|indexedDB/i,
  );
  assert.doesNotMatch(fuentes, /\b(?:console|logger|registrarTraza)\s*\./i);
  assert.doesNotMatch(vistaFuente, /idempotencia|hmac|decisi[oó]n|atestaci[oó]n|token/i);
  assert.match(vistaFuente, /function escaparHTML/);
  assert.match(vistaFuente, /scrollIntoView\?\.\(\{ block: "nearest", inline: "nearest" \}\)/);
  assert.match(vistaFuente, /raiz\.innerHTML = renderizarAltaContratacionTemporal/);
  assert.match(vistaFuente, /if \(!montada\) return/);
  assert.doesNotMatch(vistaFuente, /preventScroll/);
});

test("el módulo completo se compone sin alterar las rutas de Bolsa, Cronos y Dietas", async () => {
  assert.deepEqual([...VISTAS_MODULOS_PERSONALES], ["cronos", "dietas"]);
  assert.ok(VISTAS_MODULOS_CONECTADOS.has("contratacion-temporal"));
  assert.equal(moduloDeVistaPortal("resumen"), "bolsa");
  assert.equal(moduloDeVistaPortal("contratacion-temporal"), "contratacion_temporal");
  assert.equal(moduloDeVistaPortal("cronos"), "cronos");
  assert.equal(moduloDeVistaPortal("dietas"), "dietas");
  assert.equal(rutaDeVistaPortal("resumen"), "#bolsa/resumen");
  assert.equal(rutaDeVistaPortal("cronos"), "#cronos");
  assert.equal(rutaDeVistaPortal("dietas"), "#dietas");
  assert.match(coordinadorFuente, /crearPresentadorCronos/);
  assert.match(coordinadorFuente, /montarModuloDietas/);
  assert.match(coordinadorPruebas, /Cronos y Dietas montan contenido administrativo/);
  assert.match(indicePortal, /modulos\/cronos\/cronos\.css/);
  assert.match(indicePortal, /modulos\/dietas\/dietas\.css/);
  assert.match(catalogoPresentacion, /clave: "contratacion_temporal"/);
  assert.match(coordinadorFuente, /import\("\.\/modulos\/contratacion-temporal\/adaptador-presentacion\.js"\)/);

  const archivos = (await readdir(directorio)).sort();
  for (const nombre of [
    "INTEGRACION.md", "adaptador-presentacion.js", "componentes-expedientes.js",
    "cliente-http-alta.js", "cliente-http.js", "contratacion-temporal-integracion.test.mjs",
    "contratacion-temporal.css",
    "contratacion-temporal.test.mjs", "contrato-expedientes.js", "contrato.js",
    "datos-presentacion.js", "expedientes-responsive.css", "expedientes.css",
    "i18n-expedientes.js", "i18n.js", "presentador-expedientes.js", "presentador.js",
    "vista-expedientes.js", "vista.js",
  ]) assert.ok(archivos.includes(nombre), `falta ${nombre}`);
});
