import assert from "node:assert/strict";
import test from "node:test";
import { readFileSync } from "node:fs";
import { descargarResumenServicios } from "./servicios-descarga.js";
import { MIME_EXPORTACION_SERVICIOS } from "./cliente-http-exportacion-servicios.js";

const bytes = new TextEncoder().encode('Inicio,Fin\n2020-01-01,2020-12-31\n');
const archivo = () => ({ bytes, mime: MIME_EXPORTACION_SERVICIOS, nombre: "servicios.csv", huella: "a".repeat(64) });
function documento(falla = false) {
  const registro = []; let blob;
  const enlace = { remove() { registro.push("remove"); }, click() { registro.push("click"); if (falla) throw new Error("descarga no disponible"); } };
  const d = { createElement: () => enlace, body: { append() {} }, defaultView: { Blob, URL: { createObjectURL(dato) { blob = dato; registro.push("create"); return "blob:local"; }, revokeObjectURL(url) { registro.push(url); } } } };
  return { d, registro, enlace, blob: () => blob };
}

test("descarga los mismos bytes que entregó el servidor sin serializar filas", async () => {
  const { d, registro, enlace, blob } = documento();
  descargarResumenServicios(d, archivo());
  assert.deepEqual(new Uint8Array(await blob().arrayBuffer()), bytes);
  assert.equal(blob().type, MIME_EXPORTACION_SERVICIOS); assert.equal(enlace.download, "servicios.csv");
  assert.deepEqual(registro, ["create", "click", "remove", "blob:local"]);
});

test("se retiran enlace y URL temporal aunque falle el inicio de descarga", () => {
  const { d, registro } = documento(true);
  assert.throws(() => descargarResumenServicios(d, archivo()));
  assert.deepEqual(registro, ["create", "click", "remove", "blob:local"]);
});

test("filas de la vista y nombres ajenos no se convierten en archivos", () => {
  const { d, registro } = documento();
  for (const invalido of [{ estado: "disponible", items: [] }, { ...archivo(), nombre: "../privado.csv" }, { ...archivo(), bytes: new Uint8Array() }, { ...archivo(), mime: "text/html" }, { ...archivo(), huella: "" }]) assert.throws(() => descargarResumenServicios(d, invalido));
  assert.deepEqual(registro, []);
});

test("catálogos de exportación ES/EN comparten cabeceras, formato y textos de recuperación", () => {
  const catalogo = (lang) => JSON.parse(readFileSync(new URL(`../../../textos/${lang}/personal-exportacion-servicios.json`, import.meta.url)));
  const es = catalogo("es"), en = catalogo("en");
  assert.deepEqual(es.formato, en.formato);
  assert.deepEqual(Object.keys(es.csv), ["fecha_inicio", "fecha_fin", "clase", "dias", "estado"]);
  assert.deepEqual(Object.keys(es.csv), Object.keys(en.csv));
  assert.deepEqual(Object.keys(es.general), Object.keys(en.general));
  for (const clave of Object.keys(es.general)) assert.ok(es.general[clave] && en.general[clave]);
  assert.doesNotMatch(es.general.preparada, /guardado|entregado/u);
  assert.match(es.general.denegado, /Actualice/u);
});
